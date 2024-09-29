package battery

import (
	"context"
	"errors"
	"github.com/bluenviron/goroslib/v2"
	"github.com/bluenviron/goroslib/v2/pkg/msgs/std_msgs"
	"github.com/brokenrobotz/viam-ros-module/viamrosnode"
	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"strings"
	"sync"
)

var VoltageModel = resource.NewModel("brokenrobotz", "ros", "voltage")

type VoltageSensor struct {
	resource.Named

	mu         sync.Mutex
	primaryUri string
	topic      string
	node       *goroslib.Node
	subscriber *goroslib.Subscriber
	msg        *std_msgs.Float32
	logger     logging.Logger
}

func init() {
	resource.RegisterComponent(
		sensor.API,
		VoltageModel,
		resource.Registration[sensor.Sensor, *VoltageSensorConfig]{
			Constructor: NewVoltageSensor,
		},
	)
}

func NewVoltageSensor(
	ctx context.Context,
	deps resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
) (sensor.Sensor, error) {
	v := &VoltageSensor{
		Named:  conf.ResourceName().AsNamed(),
		logger: logger,
	}

	if err := v.Reconfigure(ctx, deps, conf); err != nil {
		return nil, err
	}

	return v, nil
}

func (v *VoltageSensor) Reconfigure(
	_ context.Context,
	_ resource.Dependencies,
	conf resource.Config,
) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.primaryUri = conf.Attributes.String("primary_uri")
	v.topic = conf.Attributes.String("topic")

	if len(strings.TrimSpace(v.primaryUri)) == 0 {
		return errors.New("ROS primary uri must be set to hostname:port")
	}
	if len(strings.TrimSpace(v.topic)) == 0 {
		return errors.New("ROS topic must be set to valid sensor topic")
	}

	if v.subscriber != nil {
		v.subscriber.Close()
	}

	var err error
	v.node, err = viamrosnode.GetInstance(v.primaryUri)
	if err != nil {
		return err
	}

	v.subscriber, err = goroslib.NewSubscriber(goroslib.SubscriberConf{
		Node:     v.node,
		Topic:    v.topic,
		Callback: v.processMessage,
	})

	if err != nil {
		return err
	}

	return nil
}

func (v *VoltageSensor) processMessage(msg *std_msgs.Float32) {
	v.msg = msg
}

func (v *VoltageSensor) Readings(
	_ context.Context,
	_ map[string]interface{},
) (map[string]interface{}, error) {
	if v.msg == nil {
		return nil, errors.New("battery message not prepared")
	}
	return map[string]interface{}{"voltage": v.msg.Data}, nil
}

func (v *VoltageSensor) Close(_ context.Context) error {
	if v.subscriber != nil {
		v.subscriber.Close()
	}
	return nil
}
