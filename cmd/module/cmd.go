package main

import (
	"context"
	"github.com/brokenrobotz/viam-ros-module/base"
	"github.com/brokenrobotz/viam-ros-module/camera"
	"github.com/brokenrobotz/viam-ros-module/sensors"
	"github.com/brokenrobotz/viam-ros-module/sensors/battery"
	"github.com/brokenrobotz/viam-ros-module/viamrosnode"

	viambase "go.viam.com/rdk/components/base"
	viamcamera "go.viam.com/rdk/components/camera"
	viamsensor "go.viam.com/rdk/components/sensor"

	"github.com/brokenrobotz/viam-ros-module/imu"
	viammovementsensor "go.viam.com/rdk/components/movementsensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/module"
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

	myMod, err := module.NewModuleFromArgs(ctx, logger)
	if err != nil {
		return err
	}

	err = myMod.AddModelFromRegistry(ctx, viammovementsensor.API, imu.Model)
	err = myMod.AddModelFromRegistry(ctx, viamsensor.API, battery.BatteryModel)
	err = myMod.AddModelFromRegistry(ctx, viamsensor.API, battery.VoltageModel)
	err = myMod.AddModelFromRegistry(ctx, viamsensor.API, sensors.EditionModel)
	err = myMod.AddModelFromRegistry(ctx, viamsensor.API, sensors.DiagnosticsModel)
	err = myMod.AddModelFromRegistry(ctx, viambase.API, base.RosBaseModel)
	err = myMod.AddModelFromRegistry(ctx, viamcamera.API, camera.ROSLidarModel)
	err = myMod.AddModelFromRegistry(ctx, viamcamera.API, camera.RosCameraModel)

	err = myMod.Start(ctx)
	defer myMod.Close(ctx)
	defer viamrosnode.ShutdownNodes()

	if err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}
