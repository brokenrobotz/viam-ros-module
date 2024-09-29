package sensors

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

var EditionModel = resource.NewModel("brokenrobotz", "ros", "edition")

type EditionSensor struct {
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
		EditionModel,
		resource.Registration[sensor.Sensor, *EditionSensorConfig]{
			Constructor: NewEditionSensor,
		},
	)
}

func NewEditionSensor(
	ctx context.Context,
	deps resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
) (sensor.Sensor, error) {
	e := &EditionSensor{
		Named:  conf.ResourceName().AsNamed(),
		logger: logger,
	}

	if err := e.Reconfigure(ctx, deps, conf); err != nil {
		return nil, err
	}

	return e, nil
}

func (e *EditionSensor) Reconfigure(
	_ context.Context,
	_ resource.Dependencies,
	conf resource.Config,
) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.primaryUri = conf.Attributes.String("primary_uri")
	e.topic = conf.Attributes.String("topic")

	if len(strings.TrimSpace(e.primaryUri)) == 0 {
		return errors.New("ROS primary uri must be set to hostname:port")
	}

	if len(strings.TrimSpace(e.topic)) == 0 {
		return errors.New("ROS topic must be set to valid sensor topic")
	}

	if e.subscriber != nil {
		e.subscriber.Close()
	}

	var err error
	e.node, err = viamrosnode.GetInstance(e.primaryUri)
	if err != nil {
		return err
	}

	e.subscriber, err = goroslib.NewSubscriber(goroslib.SubscriberConf{
		Node:     e.node,
		Topic:    e.topic,
		Callback: e.processMessage,
	})
	if err != nil {
		return err
	}

	return nil
}

func (e *EditionSensor) processMessage(msg *std_msgs.Float32) {
	e.msg = msg
}

func (e *EditionSensor) Readings(
	_ context.Context,
	_ map[string]interface{},
) (map[string]interface{}, error) {
	if e.msg == nil {
		return nil, errors.New("edition message not prepared")
	}
	return map[string]interface{}{"edition": e.msg.Data}, nil
}

func (e *EditionSensor) Close(_ context.Context) error {
	if e.subscriber != nil {
		e.subscriber.Close()
	}
	return nil
}
