package v1

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/da"
	"net/http"
)

type gatewayConverter func(da.Gateway) gateway

type gatewayController struct {
	gatewayMapper    GatewayMapper
	gatewayConverter gatewayConverter
	deviceConverter  deviceConverter
}

func (g *gatewayController) listGateways(w http.ResponseWriter, r *http.Request) {
	apiGateways := make(map[string]gateway)

	for name, gw := range g.gatewayMapper.Gateways() {
		tg := g.gatewayConverter(gw)
		tg.Identifier = name

		apiGateways[name] = tg
	}

	data, err := json.Marshal(apiGateways)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Add("content-type", "application/json")
	w.Write(data)
}

func (g *gatewayController) getGateway(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	id, ok := params["identifier"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	gw, ok := g.gatewayMapper.Gateways()[id]
	if !ok {
		http.NotFound(w, r)
		return
	}

	outputGw := g.gatewayConverter(gw)
	outputGw.Identifier = id

	data, err := json.Marshal(outputGw)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Add("content-type", "application/json")
	w.Write(data)
}

func (g *gatewayController) listDevicesOnGateway(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	id, ok := params["identifier"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	gw, ok := g.gatewayMapper.Gateways()[id]
	if !ok {
		http.NotFound(w, r)
		return
	}

	apiDevices := make(map[string]device)

	for _, daDevice := range gw.Devices() {
		d := g.deviceConverter(r.Context(), daDevice)
		d.Gateway = id

		apiDevices[d.Identifier] = d
	}

	data, err := json.Marshal(apiDevices)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Add("content-type", "application/json")
	w.Write(data)
}
