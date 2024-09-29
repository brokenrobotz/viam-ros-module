package camera

import (
	"context"
	"errors"
	"fmt"
	"github.com/bluenviron/goroslib/v2"
	"github.com/bluenviron/goroslib/v2/pkg/msgs/sensor_msgs"
	"github.com/brokenrobotz/viam-ros-module/viamrosnode"
	"github.com/golang/geo/r3"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/gostream"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/pointcloud"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/rimage/transform"
	"go.viam.com/rdk/ros"
	"math"
	"strings"
	"sync"
	"time"
)

var ROSLidarModel = resource.NewModel("brokenrobotz", "ros", "lidar")
var ROSDummyLidarModel = resource.NewModel("brokenrobotz", "ros", "lidar-dummy")

type ROSLidar struct {
	resource.Named

	mu         sync.Mutex
	primaryUri string
	topic      string
	timeRate   time.Duration // ms to publish
	node       *goroslib.Node
	subscriber *goroslib.Subscriber
	msg        *sensor_msgs.LaserScan
	pcMsg      pointcloud.PointCloud
	logger     logging.Logger
}

func init() {
	resource.RegisterComponent(
		camera.API,
		ROSLidarModel,
		resource.Registration[camera.Camera, *ROSLidarConfig]{
			Constructor: NewROSLidar,
		},
	)

	resource.RegisterComponent(
		camera.API,
		ROSDummyLidarModel,
		resource.Registration[camera.Camera, resource.NoNativeConfig]{
			Constructor: NewROSLidarDummy,
		},
	)

}

func NewROSLidar(
	ctx context.Context,
	deps resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
) (camera.Camera, error) {
	l := &ROSLidar{
		Named:  conf.ResourceName().AsNamed(),
		logger: logger,
	}

	if err := l.Reconfigure(ctx, deps, conf); err != nil {
		return nil, err
	}

	return l, nil
}

func NewROSLidarDummy(
	ctx context.Context,
	deps resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
) (camera.Camera, error) {

	l := &ROSLidar{
		Named:  conf.ResourceName().AsNamed(),
		logger: logger,
	}

	msgs, err := loadMessages(conf.Attributes.String("bag"))
	if err != nil {
		return nil, err
	}

	l.msg = &msgs[0]

	return l, nil
}

// Reconfigure clean this up
func (l *ROSLidar) Reconfigure(
	_ context.Context,
	_ resource.Dependencies,
	conf resource.Config,
) error {
	var err error
	l.mu.Lock()
	defer l.mu.Unlock()
	l.primaryUri = conf.Attributes.String("primary_uri")
	l.topic = conf.Attributes.String("topic")

	if len(strings.TrimSpace(l.primaryUri)) == 0 {
		return errors.New("ROS primary uri must be set to hostname:port")
	}

	if len(strings.TrimSpace(l.topic)) == 0 {
		return errors.New("ROS topic must be set to valid imu topic")
	}

	if l.subscriber != nil {
		l.subscriber.Close()
	}

	l.node, err = viamrosnode.GetInstance(l.primaryUri)
	if err != nil {
		return err
	}

	l.subscriber, err = goroslib.NewSubscriber(goroslib.SubscriberConf{
		Node:     l.node,
		Topic:    l.topic,
		Callback: l.processMessage,
	})
	if err != nil {
		return err
	}
	return nil
}

func (l *ROSLidar) processMessage(msg *sensor_msgs.LaserScan) {
	l.logger.Debug("received LaserScan message")
	l.msg = msg
}

func (l *ROSLidar) Projector(_ context.Context) (transform.Projector, error) {
	return nil, fmt.Errorf("not implemented")
}

func (l *ROSLidar) Images(_ context.Context) ([]camera.NamedImage, resource.ResponseMetadata, error) {
	return nil, resource.ResponseMetadata{}, fmt.Errorf("not implemented")
}

func (l *ROSLidar) Stream(ctx context.Context, errHandlers ...gostream.ErrorHandler) (gostream.VideoStream, error) {
	return nil, fmt.Errorf("not implemented")
}

func (l *ROSLidar) NextPointCloud(_ context.Context) (pointcloud.PointCloud, error) {

	l.mu.Lock()
	msg := l.msg
	l.mu.Unlock()

	if msg == nil {
		return nil, errors.New("lidar is not ready")
	}
	l.logger.Debugf("Scan with: %d points", len(msg.Ranges))
	return convertMsg(msg)
}

func convertMsg(msg *sensor_msgs.LaserScan) (pointcloud.PointCloud, error) {

	pc := pointcloud.New()

	for i, r := range msg.Ranges {
		if r < msg.RangeMin || r > msg.RangeMax {
			// TODO(erh): is this right? needed?
			continue
		}

		p := r3.Vector{}
		ang := msg.AngleMin + (float32(i) * msg.AngleIncrement)
		p.Y = 1000 * math.Sin(float64(ang)) * float64(r)
		p.X = 1000 * math.Cos(float64(ang)) * float64(r)

		d := pointcloud.NewBasicData()
		d.SetIntensity(uint16(msg.Intensities[i]))

		err := pc.Set(p, d)
		if err != nil {
			return nil, err
		}
	}

	return pc, nil
}

func (l *ROSLidar) Properties(_ context.Context) (camera.Properties, error) {
	return camera.Properties{
		SupportsPCD: true,
	}, nil
}

func (l *ROSLidar) Close(_ context.Context) error {
	if l.subscriber != nil {
		l.subscriber.Close()
	}

	if l.node != nil {
		l.node.Close()
	}
	return nil
}

func loadMessages(fn string) ([]sensor_msgs.LaserScan, error) {

	bag, err := ros.ReadBag(fn)
	if err != nil {
		return nil, err
	}

	err = ros.WriteTopicsJSON(bag, 0, 0, nil)
	if err != nil {
		return nil, err
	}

	all, err := ros.AllMessagesForTopic(bag, "scan")
	if err != nil {
		return nil, err
	}

	fixed := []sensor_msgs.LaserScan{}

	for _, m := range all {
		mm := sensor_msgs.LaserScan{}
		// TODO(erh): there must be a better way to do this

		data := m["data"].(map[string]interface{})

		mm.AngleMin = float32(data["angle_min"].(float64))
		mm.AngleMax = float32(data["angle_max"].(float64))
		mm.AngleIncrement = float32(data["angle_increment"].(float64))
		mm.TimeIncrement = float32(data["time_increment"].(float64))
		mm.ScanTime = float32(data["scan_time"].(float64))
		mm.RangeMin = float32(data["range_min"].(float64))
		mm.RangeMax = float32(data["range_max"].(float64))

		mm.Intensities = fixArray(data["intensities"])
		mm.Ranges = fixArray(data["ranges"])

		fixed = append(fixed, mm)
	}

	return fixed, nil
}

func fixArray(a interface{}) []float32 {
	b := a.([]interface{})
	c := []float32{}
	for _, n := range b {
		c = append(c, float32(n.(float64)))
	}
	return c
}
