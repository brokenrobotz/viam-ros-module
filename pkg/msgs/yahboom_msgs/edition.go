package yahboom_msgs

import (
	"github.com/bluenviron/goroslib/v2/pkg/msg"
)

type Edition struct {
	msg.Package `ros:"transbot_msgs"`
	Edition     float32 `rosname:"edition"`
}
