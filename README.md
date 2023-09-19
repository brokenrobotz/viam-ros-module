# Viam & ROS Melodic

In the office we have an old [Yahboom ROS Robot](https://category.yahboom.net/collections/ros-robotics/products/transbot-jetson_nano) 
which was parked in the lab closet. We thought it would be fun to see if we could use Viam and integrate it with ROS to 
1. collect data from the robot in real time
2. store the data in the cloud
3. integrated with both the Viam ML and SLAM services
4. Build an internet enabled web app and mobile app

## Installation

First, we need to [install viam](https://docs.viam.com/installation/) on the system where ROS is running.

If you have a transbot (or similar yahboom robot) running ROS Melodic and would like to see how viam works with ROS, you can
install this module from the [viam registry](https://docs.viam.com/extend/modular-resources/configure/). If you have a 
Robot running ROS that is not a yahboom, we might need to add some custom messages as well as new topics.

see the [BUILDING.md](./BUILDING.md) document for more instructions.

## Design Decisions

After getting the jetson flashed and built according to the Yahboom tutorial, we made a decision to use the goroslib open
source library to integrate with the Viam GO SDK. ROS Melodic makes use of Python 3.6.9, the viam SDK requires Python 3.9
or greater.

To prove out our idea we wanted to implement the basic set of options:
1. twist messages
2. capture image data
3. capture lidar data
4. capture imu data
5. capture sensor data from custom messages

The [cmd/module/cmd.go](./cmd/module/cmd.go) file is the entry point into the program, this will start a rosnode using the
goroslib module, we create one node per roscore primary address. Typically only one node will be created.

This node will support subscriptions and publishers to the various nodes.

We wanted to stay away from adding any ros packages (i.e. ros-bridge etc.) to ensure we don't cause any collision with 
software that was existing.

## Contributing

We welcome pull requests and issues, if there are any issues you can email:

* [shawn@viam.com](mailto:shawn@viam.com)
* [solution-eng@viam.com](mailto:solution-eng@viam.com)

### Supported components
1. The [base](./base/base.go) converts Viam grpc calls to twist messages which are published to the `/cmd_vel` topic
2. The [camera](./camera/camera.go) converts ROS Image messages to jpg format to be processed by Viam.
3. The [lidar](./camera/lidar.go) converts ROS LaserScan messages to Viam pointcloud data.
4. The [imu](./imu/imu.go) converts ROS IMU Message to Viam movementsensor data.
5. The [battery sensor](./sensors/batterysensor.go) converts the Transbot Battery message to Viam sensor data
6. The [edition sensor](./sensors/editionsensor.go) converts the Transbot Edition message to Viam sensor data


## References
1. [viam documentation](https://docs.viam.com/)
2. [goroslib](https://github.com/bluenviron/goroslib)
3. [ROS Melodic](http://wiki.ros.org/melodic)
4. [Yahboom Tutorial](http://www.yahboom.net/study/Transbot-jetson_nano)
5. [Viam Go Documentation](https://pkg.go.dev/go.viam.com/rdk)

