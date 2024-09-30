package sensors

import (
	"context"
	"errors"
	"github.com/bluenviron/goroslib/v2"
	"github.com/bluenviron/goroslib/v2/pkg/msgs/diagnostic_msgs"
	"github.com/bluenviron/goroslib/v2/pkg/msgs/std_msgs"
	"github.com/brokenrobotz/viam-ros-module/viamrosnode"
	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"strings"
	"sync"
)

var DiagnosticsModel = resource.NewModel("brokenrobotz", "ros", "diagnostics")

type DiagnosticsSensor struct {
	resource.Named

	mu         sync.Mutex
	primaryUri string
	topic      string
	node       *goroslib.Node
	subscriber *goroslib.Subscriber
	msg        *diagnostic_msgs.DiagnosticArray
	logger     logging.Logger
}

func init() {
	resource.RegisterComponent(
		sensor.API,
		DiagnosticsModel,
		resource.Registration[sensor.Sensor, *DiagnosticsSensorConfig]{
			Constructor: NewDiagnosticsSensor,
		},
	)
}

func NewDiagnosticsSensor(
	ctx context.Context,
	deps resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
) (sensor.Sensor, error) {
	d := &DiagnosticsSensor{
		Named:  conf.ResourceName().AsNamed(),
		logger: logger,
	}

	if err := d.Reconfigure(ctx, deps, conf); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *DiagnosticsSensor) Reconfigure(
	_ context.Context,
	_ resource.Dependencies,
	conf resource.Config,
) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.primaryUri = conf.Attributes.String("primary_uri")
	d.topic = conf.Attributes.String("topic")

	if len(strings.TrimSpace(d.primaryUri)) == 0 {
		return errors.New("ROS primary uri must be set to hostname:port")
	}

	if len(strings.TrimSpace(d.topic)) == 0 {
		return errors.New("ROS topic must be set to valid sensor topic")
	}

	if d.subscriber != nil {
		d.subscriber.Close()
	}

	var err error
	d.node, err = viamrosnode.GetInstance(d.primaryUri)
	if err != nil {
		return err
	}

	d.subscriber, err = goroslib.NewSubscriber(goroslib.SubscriberConf{
		Node:     d.node,
		Topic:    d.topic,
		Callback: d.processMessage,
	})
	if err != nil {
		return err
	}

	return nil
}

func (d *DiagnosticsSensor) processMessage(msg *diagnostic_msgs.DiagnosticArray) {
	d.msg = msg
}

func (d *DiagnosticsSensor) Readings(
	_ context.Context,
	_ map[string]interface{},
) (map[string]interface{}, error) {
	if d.msg == nil {
		return nil, errors.New("diagnostics message not prepared")
	}
	return map[string]interface{}{
		"status": convertStatusToMap(d.msg.Status),
	}, nil
}

func (d *DiagnosticsSensor) Close(_ context.Context) error {
	if d.subscriber != nil {
		d.subscriber.Close()
	}
	return nil
}

func convertHeaderToMap(header std_msgs.Header) map[string]interface{} {
	return map[string]interface{}{
		"seq":      header.Seq,
		"stamp":    header.Stamp,
		"frame_id": header.FrameId,
	}
}

func convertStatusToMap(statuses []diagnostic_msgs.DiagnosticStatus) []map[string]interface{} {
	var result []map[string]interface{}
	for _, status := range statuses {
		statusMap := map[string]interface{}{
			"level":       status.Level,
			"name":        status.Name,
			"message":     status.Message,
			"hardware_id": status.HardwareId,
			"values":      convertKeyValuesToMap(status.Values),
		}
		result = append(result, statusMap)
	}
	return result
}

func convertKeyValuesToMap(values []diagnostic_msgs.KeyValue) []map[string]interface{} {
	result := make([]map[string]interface{}, len(values))
	for i, kv := range values {
		kvMap := map[string]interface{}{
			"k": kv.Key,
			"v": kv.Value,
		}
		result[i] = kvMap
	}
	return result
}
