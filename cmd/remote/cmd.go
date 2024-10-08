package main

import (
	"context"
	"go.viam.com/rdk/logging"
	"os"

	"go.viam.com/rdk/config"
	robotimpl "go.viam.com/rdk/robot/impl"
	"go.viam.com/rdk/robot/web"

	_ "github.com/brokenrobotz/viam-ros-module/camera"
	_ "github.com/brokenrobotz/viam-ros-module/imu"
)

func main() {
	err := realMain()
	if err != nil {
		panic(err)
	}
}
func realMain() error {

	ctx := context.Background()
	logger := logging.NewDebugLogger("client")

	conf, err := config.ReadLocalConfig(ctx, os.Args[1], logger)
	if err != nil {
		return err
	}

	conf.Network.BindAddress = "0.0.0.0:8082"
	if err := conf.Network.Validate(""); err != nil {
		return err
	}

	myRobot, err := robotimpl.New(ctx, conf, logger)
	if err != nil {
		return err
	}

	return web.RunWebWithConfig(ctx, myRobot, conf, logger)
}
