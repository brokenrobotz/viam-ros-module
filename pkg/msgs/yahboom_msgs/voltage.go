package yahboom_msgs

import (
	"github.com/bluenviron/goroslib/v2/pkg/msg"
)

type Battery struct {
	msg.Package `ros:"yahboom_msgs"`
	data        float32
}
