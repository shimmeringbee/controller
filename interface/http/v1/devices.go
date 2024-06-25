package v1

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/controller/interface/converters/exporter"
	"github.com/shimmeringbee/controller/interface/converters/invoker"
	"github.com/shimmeringbee/controller/layers"
	"github.com/shimmeringbee/controller/state"
	"github.com/shimmeringbee/da"
	"io/ioutil"
	"net/http"
)

const DefaultHttpOutputLayer string = "http"

type deviceExporter interface {
	ExportDevice(context.Context, da.Device) exporter.ExportedDevice
	ExportSimpleDevice(context.Context, da.Device) exporter.ExportedSimpleDevice
	ExportCapability(context.Context, interface{}) interface{}
}

type deviceController struct {
	gatewayMapper   state.GatewayMapper
	deviceExporter  deviceExporter
	deviceInvoker   invoker.Invoker
	deviceOrganiser *state.DeviceOrganiser
	stack           layers.OutputStack
}

func (d *deviceController) listDevices(w http.ResponseWriter, r *http.Request) {
	apiDevices := make(map[string]exporter.ExportedDevice)

	for _, gateway := range d.gatewayMapper.Gateways() {
		for _, daDevice := range gateway.Devices() {
			d := d.deviceExporter.ExportDevice(r.Context(), daDevice)
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

	apiDevice := d.deviceExporter.ExportDevice(r.Context(), daDevice)
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
			if errors.Is(err, state.ErrNotFound) {
				http.NotFound(w, r)
			} else {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}

			return
		}
	}

	http.Error(w, http.StatusText(http.StatusNoContent), http.StatusNoContent)
}

func (d *deviceController) useDeviceCapabilityAction(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	id, ok := params["identifier"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	capabilityName, ok := params["name"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	capabilityAction, ok := params["action"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	daDevice, found := d.gatewayMapper.Device(id)
	if !found {
		http.NotFound(w, r)
		return
	}

	layer := r.URL.Query().Get("layer")
	if layer == "" {
		layer = DefaultHttpOutputLayer
	}

	retention := layers.OneShot
	if r.URL.Query().Get("retention") == "maintain" {
		retention = layers.Maintain
	}

	var body []byte
	var err error

	if r.Body != nil {
		body, err = ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if r.Body.Close() != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	if data, err := d.deviceInvoker(r.Context(), d.stack, layer, retention, daDevice, capabilityName, capabilityAction, body); err != nil {
		if errors.Is(err, invoker.ActionNotSupported) {
			http.NotFound(w, r)
		} else if errors.Is(err, invoker.ActionUserError) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		} else if errors.Is(err, invoker.CapabilityNotSupported) {
			http.NotFound(w, r)
		} else if errors.Is(err, context.DeadlineExceeded) {
			http.Error(w, "Device action exceeded permitted time.", http.StatusInternalServerError)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	} else {
		if jsonData, err := json.Marshal(data); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(jsonData)
		}
	}
}
