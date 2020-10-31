package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"io/ioutil"
	"net/http"
	"time"
)

type ActionError string

func (e ActionError) Error() string {
	return string(e)
}

const ActionNotSupported = ActionError("action not available on capability")
const ActionUserError = ActionError("user provided bad data")

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

	for _, capFlag := range daDevice.Capabilities() {
		uncastCap := daDevice.Gateway().Capability(capFlag)

		if uncastCap != nil {
			if castCap, ok := uncastCap.(da.BasicCapability); ok {
				if castCap.Name() == capabilityName {
					body, err := ioutil.ReadAll(r.Body)
					if err != nil {
						http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
						return
					}

					if data, err := d.deviceAction(r.Context(), daDevice, uncastCap, capabilityAction, body); err != nil {
						if err == ActionNotSupported {
							http.NotFound(w, r)
							return
						} else if errors.Is(err, ActionUserError) {
							http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
							return
						}
					} else {
						if jsonData, err := json.Marshal(data); err != nil {
							http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
							return
						} else {
							w.WriteHeader(http.StatusOK)
							w.Write(jsonData)
							return
						}
					}
				}
			}
		}
	}

	http.NotFound(w, r)
}

func doDeviceCapabilityAction(ctx context.Context, d da.Device, c interface{}, a string, b []byte) (interface{}, error) {
	switch cast := c.(type) {
	case capabilities.DeviceDiscovery:
		return doDeviceDiscovery(ctx, d, cast, a, b)
	case capabilities.EnumerateDevice:
		return doEnumerateDevice(ctx, d, cast, a, b)
	}

	return nil, ActionNotSupported
}

type DeviceDiscoveryEnable struct {
	Duration int
}

func doDeviceDiscovery(ctx context.Context, d da.Device, c capabilities.DeviceDiscovery, a string, b []byte) (interface{}, error) {
	switch a {
	case "Enable":
		input := DeviceDiscoveryEnable{}
		if err := json.Unmarshal(b, &input); err != nil {
			return nil, fmt.Errorf("%w: unable to parse user data: %s", ActionUserError, err.Error())
		}

		duration := time.Duration(input.Duration) * time.Millisecond
		return struct{}{}, c.Enable(ctx, d, duration)
	case "Disable":
		return struct{}{}, c.Disable(ctx, d)
	}

	return nil, ActionNotSupported
}

func doEnumerateDevice(ctx context.Context, d da.Device, c capabilities.EnumerateDevice, a string, b []byte) (interface{}, error) {
	switch a {
	case "Enumerate":
		return struct{}{}, c.Enumerate(ctx, d)
	}

	return nil, ActionNotSupported
}
