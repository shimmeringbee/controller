package v1

import (
	"embed"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/controller/gateway"
	"github.com/shimmeringbee/controller/interface/exporter"
	"github.com/shimmeringbee/controller/layers"
	"github.com/shimmeringbee/controller/metadata"
	"net/http"
)

//go:embed openapi.json
var openapi embed.FS

func ConstructRouter(mapper gateway.Mapper, deviceOrganiser *metadata.DeviceOrganiser, stack layers.OutputStack) http.Handler {
	r := mux.NewRouter()

	deviceConverter := exporter.DeviceExporter{
		GatewayMapper:   mapper,
		DeviceOrganiser: deviceOrganiser,
	}

	dc := deviceController{
		gatewayMapper:   mapper,
		deviceExporter:  &deviceConverter,
		deviceAction:    doDeviceCapabilityAction,
		deviceOrganiser: deviceOrganiser,
		stack:           stack,
	}

	gc := gatewayController{
		gatewayMapper:    mapper,
		gatewayConverter: exporter.ExportGateway,
		deviceConverter:  &deviceConverter,
	}

	zc := zoneController{
		gatewayMapper:   mapper,
		deviceConverter: &deviceConverter,
		deviceOrganiser: deviceOrganiser,
	}

	r.HandleFunc("/devices", dc.listDevices).Methods("GET")
	r.HandleFunc("/devices/{identifier}", dc.getDevice).Methods("GET")
	r.HandleFunc("/devices/{identifier}", dc.updateDevice).Methods("PATCH")
	r.HandleFunc("/devices/{identifier}/capabilities/{name}/{action}", dc.useDeviceCapabilityAction).Methods("POST")

	r.HandleFunc("/gateways", gc.listGateways).Methods("GET")
	r.HandleFunc("/gateways/{identifier}", gc.getGateway).Methods("GET")
	r.HandleFunc("/gateways/{identifier}/devices", gc.listDevicesOnGateway).Methods("GET")

	r.HandleFunc("/zones", zc.listZones).Methods("GET")
	r.HandleFunc("/zones", zc.createZone).Methods("POST")
	r.HandleFunc("/zones/{identifier}", zc.getZone).Methods("GET")
	r.HandleFunc("/zones/{identifier}", zc.deleteZone).Methods("DELETE")
	r.HandleFunc("/zones/{identifier}", zc.updateZone).Methods("PATCH")
	r.HandleFunc("/zones/{identifier}/devices/{deviceIdentifier}", zc.addDeviceToZone).Methods("PUT")
	r.HandleFunc("/zones/{identifier}/devices/{deviceIdentifier}", zc.removeDeviceToZone).Methods("DELETE")
	r.HandleFunc("/zones/{identifier}/subzones/{subzoneIdentifier}", zc.addSubzoneToZone).Methods("PUT")
	r.HandleFunc("/zones/{identifier}/subzones/{subzoneIdentifier}", zc.removeSubzoneToZone).Methods("DELETE")

	r.Handle("/openapi.json", http.FileServer(http.FS(openapi))).Methods("GET")

	return r
}
