module github.com/shimmeringbee/controller

go 1.22.0

toolchain go1.22.2

require (
	github.com/eclipse/paho.mqtt.golang v1.3.4
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/peterbourgon/ff/v3 v3.0.0
	github.com/shimmeringbee/da v0.0.0-20240510193548-96e721e05984
	github.com/shimmeringbee/logwrap v0.1.3
	github.com/shimmeringbee/zda v0.0.0-20210428211833-a313157d62bc
	github.com/shimmeringbee/zigbee v0.0.0-20240614090423-d67fd427d102
	github.com/shimmeringbee/zstack v0.0.0-20210807171913-f73efc814fd2
	github.com/stretchr/testify v1.9.0
	github.com/tidwall/gjson v1.11.0
	go.bug.st/serial.v1 v0.0.0-20191202182710-24a6610f0541
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

require (
	github.com/creack/goselect v0.1.2 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5 // indirect
	golang.org/x/sys v0.0.0-20210525143221-35b2ab0089ea // indirect
)

replace github.com/shimmeringbee/zda => ../zda

replace github.com/shimmeringbee/zcl => ../zcl

replace github.com/shimmeringbee/zstack => ../zstack
