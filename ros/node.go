package ros

import (
	"context"
	"go.viam.com/rdk/logging"
	"net"
	"net/http"
	"time"
)

type NodeConf struct {
	MasterAddress string
	Namespace     string
	Name          string
	Host          string
	ApiSlavePort  int
	TcpRosPort    int
	UdpRosPort    int
	LogLevel      logging.Level
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	args          []string
}

type Node struct {
	conf          NodeConf
	ctx           context.Context
	ctxCancel     func()
	httpClient    *http.Client
	masterAddress *net.TCPAddr
	nodeAddress   *net.TCPAddr
}
