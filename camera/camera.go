package camera

import (
	"context"
	"errors"
	"fmt"
	"github.com/bluenviron/goroslib/v2"
	"github.com/bluenviron/goroslib/v2/pkg/msgs/sensor_msgs"
	"github.com/edaniels/golog"
	"github.com/shawnbmccarthy/viam-ros-module/viamrosnode"
	viamcamera "go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/resource"
	"image"
	"strings"
	"sync"
)

var RosCameraModel = resource.NewModel("brokenrobotz", "ros", "camera")

func NewRosMediaSource(
	ctx context.Context,
	deps resource.Dependencies,
	conf resource.Config,
	logger golog.Logger,
) (*RosMediaSource, error) {
	rms := &RosMediaSource{
		Named:  conf.ResourceName().AsNamed(),
		ctx:    ctx,
		logger: logger,
	}

	if err := rms.Reconfigure(ctx, deps, conf); err != nil {
		logger.Error("problem with reconfigure")
		return nil, err
	}
	return rms, nil
}

type RosMediaSource struct {
	resource.Named

	ctx        context.Context
	logger     golog.Logger
	mu         sync.Mutex
	img        image.Image
	primaryUri string
	topic      string
	node       *goroslib.Node
	subscriber *goroslib.Subscriber
}

func (rs *RosMediaSource) Reconfigure(
	_ context.Context,
	_ resource.Dependencies,
	conf resource.Config,
) error {
	var err error
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.primaryUri = conf.Attributes.String("primary_uri")
	rs.topic = conf.Attributes.String("topic")

	if len(strings.TrimSpace(rs.primaryUri)) == 0 {
		return errors.New("ROS primary uri must be set to hostname:port")
	}

	if len(strings.TrimSpace(rs.topic)) == 0 {
		return errors.New("ROS topic must be set to valid camera topic")
	}

	if rs.subscriber != nil {
		rs.subscriber.Close()
	}

	rs.node, err = viamrosnode.GetInstance(rs.primaryUri)
	if err != nil {
		return err
	}

	rs.subscriber, err = goroslib.NewSubscriber(goroslib.SubscriberConf{
		Node:     rs.node,
		Topic:    rs.topic,
		Callback: rs.updateImageFromRosMsg,
	})
	if err != nil {
		return err
	}

	return nil
}

func (rs *RosMediaSource) Read(_ context.Context) (image.Image, func(), error) {
	if rs.img != nil {
		return rs.img, func() {}, nil
	} else {
		return nil, nil, fmt.Errorf("image is not ready")
	}
}

func (rs *RosMediaSource) Close(_ context.Context) error {
	rs.subscriber.Close()
	return nil
}

func (rs *RosMediaSource) updateImageFromRosMsg(msg *sensor_msgs.Image) {

	if msg == nil || len(msg.Data) == 0 {
		rs.logger.Warn("ROS image data not ready")
		return
	}
	rs.mu.Lock()
	newImage := convertImage(msg)
	rs.mu.Unlock()

	rs.img = newImage
}

func convertImage(msg *sensor_msgs.Image) image.Image {
	ri := RosImage{height: int(msg.Height), width: int(msg.Width), step: int(msg.Step), data: msg.Data}
	rgbaImage := image.NewRGBA(ri.Bounds())
	for x := 0; x < int(msg.Height); x++ {
		for y := 0; y < int(msg.Width); y++ {
			rgbaImage.Set(x, y, ri.At(x, y))
		}
	}
	return rgbaImage
}

func init() {
	resource.RegisterComponent(
		viamcamera.API,
		RosCameraModel,
		resource.Registration[viamcamera.Camera, *RosMediaSourceConfig]{
			Constructor: NewRosCamera,
		},
	)
}

func NewRosCamera(
	ctx context.Context,
	deps resource.Dependencies,
	conf resource.Config,
	logger golog.Logger,
) (viamcamera.Camera, error) {
	rosVideoSrc, err := NewRosMediaSource(ctx, deps, conf, logger)
	if err != nil {
		logger.Error("problem creating ROS media source")
		return nil, err
	}

	camModel := viamcamera.NewPinholeModelWithBrownConradyDistortion(nil, nil)
	videoSrc, err := viamcamera.NewVideoSourceFromReader(
		ctx,
		rosVideoSrc,
		&camModel,
		viamcamera.ColorStream,
	)
	if err != nil {
		logger.Error("problem created new video source")
		return nil, err
	}
	return viamcamera.FromVideoSource(conf.ResourceName(), videoSrc), nil
}
