package battery

import (
	"context"
	"errors"
	"github.com/bluenviron/goroslib/v2"
	"github.com/edaniels/golog"
	"github.com/shawnbmccarthy/viam-ros-module/pkg/msgs/yahboom_msgs"
	"github.com/shawnbmccarthy/viam-ros-module/viamrosnode"
	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/resource"
	"strings"
	"sync"
)

var BatteryModel = resource.NewModel("brokenrobotz", "ros", "battery")

type BatterySensor struct {
	resource.Named

	mu         sync.Mutex
	primaryUri string
	topic      string
	node       *goroslib.Node
	subscriber *goroslib.Subscriber
	msg        *yahboom_msgs.Battery
	logger     golog.Logger
}

func init() {
	resource.RegisterComponent(
		sensor.API,
		BatteryModel,
		resource.Registration[sensor.Sensor, *BatterySensorConfig]{
			Constructor: NewBatterySensor,
		},
	)
}

func NewBatterySensor(
	ctx context.Context,
	deps resource.Dependencies,
	conf resource.Config,
	logger golog.Logger,
) (sensor.Sensor, error) {
	b := &BatterySensor{
		Named:  conf.ResourceName().AsNamed(),
		logger: logger,
	}

	if err := b.Reconfigure(ctx, deps, conf); err != nil {
		return nil, err
	}

	return b, nil
}

func (b *BatterySensor) Reconfigure(
	_ context.Context,
	_ resource.Dependencies,
	conf resource.Config,
) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.primaryUri = conf.Attributes.String("primary_uri")
	b.topic = conf.Attributes.String("topic")

	if len(strings.TrimSpace(b.primaryUri)) == 0 {
		return errors.New("ROS primary uri must be set to hostname:port")
	}

	if len(strings.TrimSpace(b.topic)) == 0 {
		return errors.New("ROS topic must be set to valid sensor topic")
	}

	if b.subscriber != nil {
		b.subscriber.Close()
	}

	var err error
	b.node, err = viamrosnode.GetInstance(b.primaryUri)
	if err != nil {
		return err
	}

	b.subscriber, err = goroslib.NewSubscriber(goroslib.SubscriberConf{
		Node:     b.node,
		Topic:    b.topic,
		Callback: b.processMessage,
	})

	if err != nil {
		return err
	}

	return nil
}

func (b *BatterySensor) processMessage(msg *yahboom_msgs.Battery) {
	b.msg = msg
}

func (b *BatterySensor) Readings(
	_ context.Context,
	_ map[string]interface{},
) (map[string]interface{}, error) {
	if b.msg == nil {
		return nil, errors.New("battery message not prepared")
	}
	return map[string]interface{}{"voltage": b.msg.Voltage}, nil
}

func (b *BatterySensor) Close(_ context.Context) error {
	if b.subscriber != nil {
		b.subscriber.Close()
	}
	return nil
}
