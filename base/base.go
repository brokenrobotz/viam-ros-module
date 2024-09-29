package base

import (
	"context"
	"errors"
	"github.com/bluenviron/goroslib/v2"
	"github.com/bluenviron/goroslib/v2/pkg/msgs/geometry_msgs"
	"github.com/edaniels/golog"
	"github.com/golang/geo/r3"
	"github.com/shawnbmccarthy/viam-ros-module/viamrosnode"
	viambase "go.viam.com/rdk/components/base"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/spatialmath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var RosBaseModel = resource.NewModel("brokenrobotz", "ros", "base")

type RosBase struct {
	resource.Named

	mu         sync.Mutex
	nodeName   string
	primaryUri string
	topic      string
	timeRate   time.Duration // ms to publish
	node       *goroslib.Node
	publisher  *goroslib.Publisher
	twistMsg   *geometry_msgs.Twist
	logger     golog.Logger
	msgRate    *goroslib.NodeRate
	closed     int32
	moving     bool
}

func init() {
	resource.RegisterComponent(
		viambase.API,
		RosBaseModel,
		resource.Registration[viambase.Base, *RosBaseConfig]{
			Constructor: NewTrackedBase,
		},
	)
}

func NewTrackedBase(
	ctx context.Context,
	deps resource.Dependencies,
	conf resource.Config,
	logger golog.Logger,
) (viambase.Base, error) {
	r := &RosBase{
		Named:  conf.ResourceName().AsNamed(),
		logger: logger,
	}

	if err := r.Reconfigure(ctx, deps, conf); err != nil {
		return nil, err
	}

	// set twist message thread - need to stop sending after stop
	go func() {
		for atomic.LoadInt32(&r.closed) == 0 {
			select {
			case <-r.msgRate.SleepChan():
				r.mu.Lock()
				r.publisher.Write(r.twistMsg)
				r.mu.Unlock()
			}
		}
	}()

	return r, nil
}

// Reconfigure clean this up
func (r *RosBase) Reconfigure(
	_ context.Context,
	_ resource.Dependencies,
	conf resource.Config,
) error {
	var err error
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nodeName = conf.Attributes.String("node_name")
	r.primaryUri = conf.Attributes.String("primary_uri")
	r.topic = conf.Attributes.String("topic")

	timeMs := conf.Attributes.Int("time_rate_ms", 250)

	r.timeRate = time.Duration(timeMs) * time.Millisecond
	r.twistMsg = &geometry_msgs.Twist{
		Linear:  geometry_msgs.Vector3{X: 0.0, Y: 0.0, Z: 0.0},
		Angular: geometry_msgs.Vector3{X: 0.0, Y: 0.0, Z: 0.0},
	}

	if len(strings.TrimSpace(r.primaryUri)) == 0 {
		return errors.New("ROS primary uri must be set to hostname:port")
	}

	if len(strings.TrimSpace(r.topic)) == 0 {
		return errors.New("ROS topic must be set to valid imu topic")
	}

	if r.publisher != nil {
		r.publisher.Close()
	}

	r.node, err = viamrosnode.GetInstance(r.primaryUri)
	if err != nil {
		return err
	}

	r.msgRate = r.node.TimeRate(r.timeRate)
	if err != nil {
		return err
	}

	// publisher for twist messages
	r.publisher, err = goroslib.NewPublisher(goroslib.PublisherConf{
		Node:  r.node,
		Topic: r.topic,
		Msg:   &geometry_msgs.Twist{},
	})
	if err != nil {
		return err
	}
	atomic.StoreInt32(&r.closed, 0)
	return nil
}

func (r *RosBase) MoveStraight(
	ctx context.Context,
	distanceMm int,
	mmPerSec float64,
	extra map[string]interface{},
) error {
	r.logger.Warn("MoveStraight Not Implemented")
	return nil
}

func (r *RosBase) Spin(
	ctx context.Context,
	angleDeg,
	degsPerSec float64,
	extra map[string]interface{},
) error {
	r.logger.Warn("Spin Not Implemented")
	return nil
}

func (r *RosBase) SetPower(
	_ context.Context,
	linear r3.Vector,
	angular r3.Vector,
	_ map[string]interface{},
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.twistMsg = &geometry_msgs.Twist{
		Linear:  geometry_msgs.Vector3{X: linear.Y, Y: 0.0, Z: 0.0},
		Angular: geometry_msgs.Vector3{X: 0.0, Y: 0.0, Z: angular.Z},
	}
	r.moving = true
	return nil
}

func (r *RosBase) SetVelocity(
	_ context.Context,
	linear r3.Vector,
	angular r3.Vector,
	_ map[string]interface{},
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.twistMsg = &geometry_msgs.Twist{
		Linear:  geometry_msgs.Vector3{X: linear.Y, Y: 0.0, Z: 0.0},
		Angular: geometry_msgs.Vector3{X: 0.0, Y: 0.0, Z: angular.Z},
	}
	r.moving = true
	return nil
}

func (r *RosBase) Stop(_ context.Context, _ map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.twistMsg = &geometry_msgs.Twist{
		Linear:  geometry_msgs.Vector3{X: 0.0, Y: 0.0, Z: 0.0},
		Angular: geometry_msgs.Vector3{X: 0.0, Y: 0.0, Z: 0.0},
	}
	r.moving = false
	return nil
}

func (r *RosBase) IsMoving(_ context.Context) (bool, error) {
	return r.moving, nil
}

func (r *RosBase) Close(_ context.Context) error {
	atomic.StoreInt32(&r.closed, 1)
	r.publisher.Close()
	return nil
}

func (r *RosBase) Properties(
	_ context.Context,
	_ map[string]interface{},
) (viambase.Properties, error) {
	return viambase.Properties{
		TurningRadiusMeters: 0.0,
		WidthMeters:         0.0,
	}, nil
}

func (r *RosBase) Geometries(_ context.Context, _ map[string]interface{}) ([]spatialmath.Geometry, error) {
	r.logger.Warn("Geometries Not Implemented")
	return nil, nil
}
