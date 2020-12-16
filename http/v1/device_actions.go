package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/controller/layers"
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

const DefaultHttpOutputLayer string = "http"

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
		uncastCap := d.stack.Lookup(DefaultHttpOutputLayer).Capability(layers.OneShot, capFlag, daDevice)

		if uncastCap != nil {
			if castCap, ok := uncastCap.(da.BasicCapability); ok {
				if castCap.Name() == capabilityName {
					body, err := ioutil.ReadAll(r.Body)
					if err != nil {
						http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
						return
					}

					if data, err := d.deviceAction(r.Context(), daDevice, uncastCap, capabilityAction, body); err != nil {
						if errors.Is(err, ActionNotSupported) {
							http.NotFound(w, r)
							return
						} else if errors.Is(err, ActionUserError) {
							http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
							return
						} else {
							http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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
	case capabilities.OnOff:
		return doOnOff(ctx, d, cast, a, b)
	case capabilities.AlarmWarningDevice:
		return doAlarmWarningDevice(ctx, d, cast, a, b)
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

func doOnOff(ctx context.Context, d da.Device, c capabilities.OnOff, a string, b []byte) (interface{}, error) {
	switch a {
	case "On":
		return struct{}{}, c.On(ctx, d)
	case "Off":
		return struct{}{}, c.Off(ctx, d)
	}

	return nil, ActionNotSupported
}

type AlarmWarningDeviceAlarm struct {
	AlarmType string
	Volume    float64
	Visual    bool
	Duration  int
}

type AlarmWarningDeviceAlert struct {
	AlarmType string
	AlertType string
	Volume    float64
	Visual    bool
}

func stringToAlarmType(alarmType string) (capabilities.AlarmType, bool) {
	for foundAT, foundName := range capabilities.AlarmTypeNameMapping {
		if foundName == alarmType {
			return foundAT, true
		}
	}

	return 0, false
}

func stringToAlertType(alertType string) (capabilities.AlertType, bool) {
	for foundAT, foundName := range capabilities.AlertTypeNameMapping {
		if foundName == alertType {
			return foundAT, true
		}
	}

	return 0, false
}

func doAlarmWarningDevice(ctx context.Context, d da.Device, c capabilities.AlarmWarningDevice, a string, b []byte) (interface{}, error) {
	switch a {
	case "Alarm":
		input := AlarmWarningDeviceAlarm{}
		if err := json.Unmarshal(b, &input); err != nil {
			return nil, fmt.Errorf("%w: unable to parse user data: %s", ActionUserError, err.Error())
		}

		duration := time.Duration(input.Duration) * time.Millisecond

		alarmType, found := stringToAlarmType(input.AlarmType)
		if !found {
			return nil, fmt.Errorf("%w: unable to parse user data: invalid alarm type", ActionUserError)
		}

		return struct{}{}, c.Alarm(ctx, d, alarmType, input.Volume, input.Visual, duration)
	case "Clear":
		return struct{}{}, c.Clear(ctx, d)
	case "Alert":
		input := AlarmWarningDeviceAlert{}
		if err := json.Unmarshal(b, &input); err != nil {
			return nil, fmt.Errorf("%w: unable to parse user data: %s", ActionUserError, err.Error())
		}

		alarmType, found := stringToAlarmType(input.AlarmType)
		if !found {
			return nil, fmt.Errorf("%w: unable to parse user data: invalid alarm type", ActionUserError)
		}

		alertType, found := stringToAlertType(input.AlertType)
		if !found {
			return nil, fmt.Errorf("%w: unable to parse user data: invalid alert type", ActionUserError)
		}

		return struct{}{}, c.Alert(ctx, d, alarmType, alertType, input.Volume, input.Visual)
	}

	return nil, ActionNotSupported
}
