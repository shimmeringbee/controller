package v1

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	gw "github.com/shimmeringbee/controller/gateway"
	"github.com/shimmeringbee/controller/interface/exporter"
	"github.com/shimmeringbee/controller/layers"
	"github.com/shimmeringbee/controller/metadata"
	"github.com/shimmeringbee/da"
	"io/ioutil"
	"net/http"
)

const DefaultHttpOutputLayer string = "http"

type deviceExporter interface {
	ExportDevice(context.Context, da.Device) exporter.ExportedDevice
}

type deviceController struct {
	gatewayMapper   gw.Mapper
	deviceExporter  deviceExporter
	deviceInvoker   exporter.Invoker
	deviceOrganiser *metadata.DeviceOrganiser
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

	outputLayer := d.stack.Lookup(layer)
	if outputLayer == nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
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

	if data, err := d.deviceInvoker(r.Context(), outputLayer, retention, daDevice, capabilityName, capabilityAction, body); err != nil {
		if errors.Is(err, exporter.ActionNotSupported) {
			http.NotFound(w, r)
		} else if errors.Is(err, exporter.ActionUserError) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		} else if errors.Is(err, exporter.CapabilityNotSupported) {
			http.NotFound(w, r)
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
