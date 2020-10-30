package v1

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/da"
	"net/http"
)

type deviceConverter func(da.Device) device

type deviceController struct {
	gatewayMapper   GatewayMapper
	deviceConverter deviceConverter
}

func (d *deviceController) listDevices(w http.ResponseWriter, r *http.Request) {
	apiDevices := make(map[string]device)

	for name, gateway := range d.gatewayMapper.Gateways() {
		for _, daDevice := range gateway.Devices() {
			d := d.deviceConverter(daDevice)
			d.Gateway = name

			apiDevices[d.Identifier] = d
		}
	}

	data, err := json.Marshal(apiDevices)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Add("content-type", "application/json")
	w.Write(data)
}

func (d *deviceController) getDevice(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	id, ok := params["identifier"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	daDevice, found := d.gatewayMapper.Device(id)
	if !found {
		http.NotFound(w, r)
		return
	}

	apiDevice := d.deviceConverter(daDevice)

	for gwId, gw := range d.gatewayMapper.Gateways() {
		if gw == daDevice.Gateway() {
			apiDevice.Gateway = gwId
			break
		}
	}

	data, err := json.Marshal(apiDevice)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Add("content-type", "application/json")
	w.Write(data)
}
