package v1

import (
	"github.com/gorilla/mux"
	"net/http"
)

func ConstructRouter(mapper GatewayMapper) http.Handler {
	r := mux.NewRouter()

	dc := deviceController{
		gatewayMapper:   mapper,
		deviceConverter: convertDADeviceToDevice,
	}

	gc := gatewayController{
		gatewayMapper:    mapper,
		gatewayConverter: convertDAGatewayToGateway,
		deviceConverter:  convertDADeviceToDevice,
	}

	r.HandleFunc("/devices", dc.listDevices).Methods("GET")
	r.HandleFunc("/devices/{identifier}", dc.getDevice).Methods("GET")

	r.HandleFunc("/gateways", gc.listGateways).Methods("GET")
	r.HandleFunc("/gateways/{identifier}", gc.getGateway).Methods("GET")
	r.HandleFunc("/gateways/{identifier}/devices", gc.listDevicesOnGateway).Methods("GET")

	return r
}
