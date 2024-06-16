module github.com/shimmeringbee/controller

go 1.22.0

toolchain go1.22.2

require (
	github.com/eclipse/paho.mqtt.golang v1.3.4
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/peterbourgon/ff/v3 v3.0.0
	github.com/shimmeringbee/da v0.0.0-20240615210808-95a0a3f8f08a
	github.com/shimmeringbee/logwrap v0.1.3
	github.com/shimmeringbee/persistence v0.0.0-20240615183316-1a60e6781413
	github.com/shimmeringbee/zda v0.0.0-20240615173732-388df40f2291
	github.com/shimmeringbee/zigbee v0.0.0-20240614104723-f4c0c0231568
	github.com/shimmeringbee/zstack v0.0.0-20240615174824-2ff56018304e
	github.com/stretchr/testify v1.9.0
	github.com/tidwall/gjson v1.11.0
	go.bug.st/serial.v1 v0.0.0-20191202182710-24a6610f0541
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

require (
	github.com/creack/goselect v0.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/expr-lang/expr v1.16.9 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/shimmeringbee/bytecodec v0.0.0-20240614104652-9d31c74dcd13 // indirect
	github.com/shimmeringbee/callbacks v0.0.0-20240614104656-b56cd6b4b604 // indirect
	github.com/shimmeringbee/retry v0.0.0-20240614104711-064c2726a8b4 // indirect
	github.com/shimmeringbee/unpi v0.0.0-20240614104715-5284f961bafc // indirect
	github.com/shimmeringbee/zcl v0.0.0-20240614104719-4eee02c0ffd1 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.0.0-20210525143221-35b2ab0089ea // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/shimmeringbee/zda => ../zda

replace github.com/shimmeringbee/zcl => ../zcl

replace github.com/shimmeringbee/zstack => ../zstack
