package v1

import (
	"embed"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/controller/interface/converters/exporter"
	"github.com/shimmeringbee/controller/interface/converters/invoker"
	"github.com/shimmeringbee/controller/interface/http/auth"
	"github.com/shimmeringbee/controller/layers"
	"github.com/shimmeringbee/controller/state"
	"github.com/shimmeringbee/logwrap"
	"net/http"
)

//go:embed openapi.json
var openapi embed.FS

func ConstructRouter(mapper state.GatewayMapper, deviceOrganiser *state.DeviceOrganiser, stack layers.OutputStack, l logwrap.Logger, ap auth.AuthenticationProvider, eventbus state.EventSubscriber) http.Handler {
	protected := mux.NewRouter()

	deviceConverter := exporter.DeviceExporter{
		GatewayMapper:   mapper,
		DeviceOrganiser: deviceOrganiser,
	}

	dc := deviceController{
		gatewayMapper:   mapper,
		deviceExporter:  &deviceConverter,
		deviceInvoker:   invoker.InvokeDeviceAction,
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

	wc := websocketController{
		eventbus: eventbus,
		eventMapper: websocketEventMapper{
			gatewayMapper:   mapper,
			deviceOrganiser: deviceOrganiser,
			deviceExporter:  &deviceConverter,
		},
	}

	protected.HandleFunc("/devices", dc.listDevices).Methods("GET")
	protected.HandleFunc("/devices/{identifier}", dc.getDevice).Methods("GET")
	protected.HandleFunc("/devices/{identifier}", dc.updateDevice).Methods("PATCH")
	protected.HandleFunc("/devices/{identifier}/capabilities/{name}/{action}", dc.useDeviceCapabilityAction).Methods("POST")

	protected.HandleFunc("/gateways", gc.listGateways).Methods("GET")
	protected.HandleFunc("/gateways/{identifier}", gc.getGateway).Methods("GET")
	protected.HandleFunc("/gateways/{identifier}/devices", gc.listDevicesOnGateway).Methods("GET")

	protected.HandleFunc("/zones", zc.listZones).Methods("GET")
	protected.HandleFunc("/zones", zc.createZone).Methods("POST")
	protected.HandleFunc("/zones/{identifier}", zc.getZone).Methods("GET")
	protected.HandleFunc("/zones/{identifier}", zc.deleteZone).Methods("DELETE")
	protected.HandleFunc("/zones/{identifier}", zc.updateZone).Methods("PATCH")
	protected.HandleFunc("/zones/{identifier}/devices/{deviceIdentifier}", zc.addDeviceToZone).Methods("PUT")
	protected.HandleFunc("/zones/{identifier}/devices/{deviceIdentifier}", zc.removeDeviceToZone).Methods("DELETE")
	protected.HandleFunc("/zones/{identifier}/subzones/{subzoneIdentifier}", zc.addSubzoneToZone).Methods("PUT")
	protected.HandleFunc("/zones/{identifier}/subzones/{subzoneIdentifier}", zc.removeSubzoneToZone).Methods("DELETE")

	protected.HandleFunc("/websocket", wc.serveWebsocket).Methods("GET")

	apiRoot := mux.NewRouter()
	apiRoot.Handle("/openapi.json", http.FileServer(http.FS(openapi))).Methods("GET")
	apiRoot.Handle("/auth/type", authenticationType(ap)).Methods("GET")
	apiRoot.Handle("/auth/check", ap.AuthenticationMiddleware(http.HandlerFunc(authenticationCheck))).Methods("GET")
	apiRoot.PathPrefix("/auth").Handler(ap.AuthenticationRouter())
	apiRoot.PathPrefix("/").Handler(ap.AuthenticationMiddleware(protected))

	return apiRoot
}
