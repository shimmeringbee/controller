package v1

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	gw "github.com/shimmeringbee/controller/gateway"
	"github.com/shimmeringbee/controller/layers"
	"github.com/shimmeringbee/controller/metadata"
	"github.com/shimmeringbee/da"
	"io/ioutil"
	"net/http"
)

type deviceConverter interface {
	ConvertDevice(context.Context, da.Device) ExportedDevice
}

type deviceAction func(context.Context, da.Device, interface{}, string, []byte) (interface{}, error)

type deviceController struct {
	gatewayMapper   gw.Mapper
	deviceConverter deviceConverter
	deviceAction    deviceAction
	deviceOrganiser *metadata.DeviceOrganiser
	stack           layers.OutputStack
}

func (d *deviceController) listDevices(w http.ResponseWriter, r *http.Request) {
	apiDevices := make(map[string]ExportedDevice)

	for _, gateway := range d.gatewayMapper.Gateways() {
		for _, daDevice := range gateway.Devices() {
			d := d.deviceConverter.ConvertDevice(r.Context(), daDevice)
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

	apiDevice := d.deviceConverter.ConvertDevice(r.Context(), daDevice)
	data, err := json.Marshal(apiDevice)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Add("content-type", "application/json")
	w.Write(data)
}

type updateDeviceRequest struct {
	Name *string
}

func (d *deviceController) updateDevice(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	id, ok := params["identifier"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	request := updateDeviceRequest{}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(data, &request)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if request.Name != nil {
		if err := d.deviceOrganiser.NameDevice(id, *request.Name); err != nil {
			if errors.Is(err, metadata.ErrNotFound) {
				http.NotFound(w, r)
			} else {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}

			return
		}
	}

	http.Error(w, http.StatusText(http.StatusNoContent), http.StatusNoContent)
}
