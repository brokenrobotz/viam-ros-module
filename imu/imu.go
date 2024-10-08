package imu

/*
 * RosImu
 * The mapping from the ROS IMU message only supports:
 * Orientation
 * Angular Velocity
 * Linear Velocity
 *
 * If we have other topics in ROS to produce
 * GPS, Compass Headings, etc:
 * we will need to add more subscribers and more
 * message type support
 */
import (
	"context"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/ros"
	"strings"
	"sync"

	"github.com/golang/geo/r3"
	geo "github.com/kellydunn/golang-geo"
	"github.com/pkg/errors"

	"github.com/bluenviron/goroslib/v2"
	"github.com/bluenviron/goroslib/v2/pkg/msgs/geometry_msgs"
	"github.com/bluenviron/goroslib/v2/pkg/msgs/sensor_msgs"
	"github.com/brokenrobotz/viam-ros-module/viamrosnode"
	"go.viam.com/rdk/components/movementsensor"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/spatialmath"
)

var Model = resource.NewModel("brokenrobotz", "ros", "imu")
var DummyImuModel = resource.NewModel("brokenrobotz", "ros", "imu-dummy")

type RosImu struct {
	resource.Named

	mu         sync.Mutex
	primaryUri string
	topic      string
	node       *goroslib.Node
	subscriber *goroslib.Subscriber
	msg        *sensor_msgs.Imu
	logger     logging.Logger
}

func init() {
	resource.RegisterComponent(
		movementsensor.API,
		Model,
		resource.Registration[movementsensor.MovementSensor, *RosImuConfig]{
			Constructor: NewRosImu,
		},
	)

	resource.RegisterComponent(
		movementsensor.API,
		DummyImuModel,
		resource.Registration[movementsensor.MovementSensor, resource.NoNativeConfig]{
			Constructor: NewRosImuDummy,
		},
	)
}

func NewRosImu(
	ctx context.Context,
	deps resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
) (movementsensor.MovementSensor, error) {
	r := &RosImu{
		Named:  conf.ResourceName().AsNamed(),
		logger: logger,
	}

	if err := r.Reconfigure(ctx, deps, conf); err != nil {
		return nil, err
	}

	return r, nil
}

func NewRosImuDummy(
	_ context.Context,
	_ resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
) (movementsensor.MovementSensor, error) {
	r := &RosImu{
		Named:  conf.ResourceName().AsNamed(),
		logger: logger,
	}
	msgs, err := loadMessages(conf.Attributes.String("bag"))
	if err != nil {
		return nil, err
	}

	r.msg = &msgs[0]

	return r, nil
}

func (r *RosImu) Reconfigure(
	_ context.Context,
	_ resource.Dependencies,
	conf resource.Config,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.primaryUri = conf.Attributes.String("primary_uri")
	r.topic = conf.Attributes.String("topic")

	if len(strings.TrimSpace(r.primaryUri)) == 0 {
		return errors.New("ROS primary uri must be set to hostname:port")
	}

	if len(strings.TrimSpace(r.topic)) == 0 {
		return errors.New("ROS topic must be set to valid imu topic")
	}

	if r.subscriber != nil {
		r.subscriber.Close()
	}

	var err error
	r.node, err = viamrosnode.GetInstance(r.primaryUri)
	if err != nil {
		return err
	}

	r.subscriber, err = goroslib.NewSubscriber(goroslib.SubscriberConf{
		Node:     r.node,
		Topic:    r.topic,
		Callback: r.processMessage,
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *RosImu) processMessage(msg interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	//var msgT *sensor_msgs.Imu

	if msgT, ok := msg.(*sensor_msgs.Imu); ok {
		r.msg = msgT
	}
}

func (r *RosImu) Position(
	_ context.Context,
	_ map[string]interface{},
) (*geo.Point, float64, error) {
	return geo.NewPoint(0, 0), 0, movementsensor.ErrMethodUnimplementedPosition
}

func (r *RosImu) LinearVelocity(
	_ context.Context,
	_ map[string]interface{},
) (r3.Vector, error) {
	return r3.Vector{}, movementsensor.ErrMethodUnimplementedLinearVelocity
}

func (r *RosImu) AngularVelocity(
	_ context.Context,
	_ map[string]interface{},
) (spatialmath.AngularVelocity, error) {
	if r.msg == nil {
		return spatialmath.AngularVelocity{}, errors.New("message unavailable")
	}
	av := spatialmath.AngularVelocity{
		X: r.msg.AngularVelocity.X,
		Y: r.msg.AngularVelocity.Y,
		Z: r.msg.AngularVelocity.Z,
	}
	return av, nil
}

func (r *RosImu) LinearAcceleration(
	_ context.Context,
	_ map[string]interface{},
) (r3.Vector, error) {
	if r.msg == nil {
		return r3.Vector{}, errors.New("message unavailable")
	}
	la := r3.Vector{
		X: r.msg.LinearAcceleration.X,
		Y: r.msg.LinearAcceleration.Y,
		Z: r.msg.LinearAcceleration.Z,
	}
	return la, nil
}

func (r *RosImu) CompassHeading(
	_ context.Context,
	_ map[string]interface{},
) (float64, error) {
	return 0, movementsensor.ErrMethodUnimplementedCompassHeading
}

func (r *RosImu) Orientation(
	_ context.Context,
	_ map[string]interface{},
) (spatialmath.Orientation, error) {
	q := r.msg.Orientation
	return &spatialmath.Quaternion{Real: q.W, Imag: q.X, Jmag: q.Y, Kmag: q.Z}, nil
}

func (r *RosImu) Properties(
	_ context.Context,
	_ map[string]interface{},
) (*movementsensor.Properties, error) {
	return &movementsensor.Properties{
		PositionSupported:           false,
		AngularVelocitySupported:    true,
		CompassHeadingSupported:     false,
		OrientationSupported:        true,
		LinearVelocitySupported:     false,
		LinearAccelerationSupported: true,
	}, nil
}

func (r *RosImu) Readings(
	ctx context.Context,
	extra map[string]interface{},
) (map[string]interface{}, error) {
	if r.msg == nil {
		return nil, errors.New("message unavailable")
	}

	return nil, nil
}

func (r *RosImu) Accuracy(
	_ context.Context,
	_ map[string]interface{},
) (*movementsensor.Accuracy, error) {
	return nil, movementsensor.ErrMethodUnimplementedAccuracy
}

func (r *RosImu) Close(_ context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.subscriber != nil {
		r.subscriber.Close()
	}
	return nil
}

/*
 * used for basic testing
 */
func loadMessages(fn string) ([]sensor_msgs.Imu, error) {
	bag, err := ros.ReadBag(fn)
	if err != nil {
		return nil, err
	}

	err = ros.WriteTopicsJSON(bag, 0, 0, nil)
	if err != nil {
		return nil, err
	}

	all, err := ros.AllMessagesForTopic(bag, "imu_data")
	if err != nil {
		return nil, err
	}

	var fixed []sensor_msgs.Imu

	for _, m := range all {
		mm := sensor_msgs.Imu{}
		data := m["data"].(map[string]interface{})

		orientation := data["orientation"].(map[string]interface{})
		orientationCovariance := data["orientation_covariance"].([]interface{})
		angularVel := data["angular_velocity"].(map[string]interface{})
		angularVelCovariance := data["angular_velocity_covariance"].([]interface{})
		linearAccel := data["linear_acceleration"].(map[string]interface{})
		linearAccelCovariance := data["linear_acceleration_covariance"].([]interface{})

		mm.Orientation = geometry_msgs.Quaternion{
			X: orientation["x"].(float64),
			Y: orientation["y"].(float64),
			Z: orientation["z"].(float64),
			W: orientation["w"].(float64),
		}
		for idx, oc := range orientationCovariance {
			mm.OrientationCovariance[idx] = oc.(float64)
		}

		mm.AngularVelocity = geometry_msgs.Vector3{
			X: angularVel["x"].(float64),
			Y: angularVel["y"].(float64),
			Z: angularVel["z"].(float64),
		}
		for idx, avc := range angularVelCovariance {
			mm.AngularVelocityCovariance[idx] = avc.(float64)
		}

		mm.LinearAcceleration = geometry_msgs.Vector3{
			X: linearAccel["x"].(float64),
			Y: linearAccel["y"].(float64),
			Z: linearAccel["z"].(float64),
		}
		for idx, lac := range linearAccelCovariance {
			mm.LinearAccelerationCovariance[idx] = lac.(float64)
		}

		fixed = append(fixed, mm)
	}
	return fixed, nil
}
