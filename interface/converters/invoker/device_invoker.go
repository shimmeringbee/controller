package invoker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shimmeringbee/controller/layers"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"time"
)

type Invoker func(ctx context.Context, s layers.OutputStack, l string, r layers.RetentionLevel, dad da.Device, capabilityName string, actionName string, payload []byte) (any, error)

type ActionError string

func (e ActionError) Error() string {
	return string(e)
}

const CapabilityNotSupported = ActionError("capability not available on device")
const ActionNotSupported = ActionError("action not available on capability")
const ActionUserError = ActionError("user provided bad data")

func InvokeDeviceAction(ctx context.Context, s layers.OutputStack, l string, r layers.RetentionLevel, dad da.Device, capabilityName string, actionName string, payload []byte) (any, error) {
	invokeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	l, r, err := resolveOutputLayerAndRetention(l, r, payload)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal payload: %w", err)
	}

	o := s.Lookup(l)

	d := o.Device(r, dad)

	for _, capFlag := range d.Capabilities() {
		uncastCap := d.Capability(capFlag)

		if uncastCap != nil {
			if castCap, ok := uncastCap.(da.BasicCapability); ok {
				if castCap.Name() == capabilityName {
					return doDeviceCapabilityAction(invokeCtx, uncastCap, actionName, payload)
				}
			}
		}
	}

	return nil, CapabilityNotSupported
}

type OutputLayerMetadata struct {
	Layer     string `json:"layer"`
	Retention string `json:"retention"`
}

type ControlMetadata struct {
	OutputLayer OutputLayerMetadata `json:"output"`
}

type MetadataPayload struct {
	Control ControlMetadata `json:"control"`
}

func resolveOutputLayerAndRetention(l string, r layers.RetentionLevel, payload []byte) (string, layers.RetentionLevel, error) {
	if payload == nil || len(payload) == 0 {
		return l, r, nil
	}

	var metadata MetadataPayload
	if err := json.Unmarshal(payload, &metadata); err != nil {
		return l, r, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	if metadata.Control.OutputLayer.Layer != "" {
		l = metadata.Control.OutputLayer.Layer
	}

	switch metadata.Control.OutputLayer.Retention {
	case "oneshot":
		r = layers.OneShot
	case "maintain":
		r = layers.Maintain
	}

	return l, r, nil
}

func doDeviceCapabilityAction(ctx context.Context, c any, a string, b []byte) (any, error) {
	switch cast := c.(type) {
	case capabilities.DeviceDiscovery:
		return doDeviceDiscovery(ctx, cast, a, b)
	case capabilities.EnumerateDevice:
		return doEnumerateDevice(ctx, cast, a)
	case capabilities.OnOff:
		return doOnOff(ctx, cast, a)
	case capabilities.AlarmWarningDevice:
		return doAlarmWarningDevice(ctx, cast, a, b)
	case capabilities.DeviceRemoval:
		return doDeviceRemoval(ctx, cast, a, b)
	}

	return nil, ActionNotSupported
}

type DeviceDiscoveryEnable struct {
	Duration int
}

func doDeviceDiscovery(ctx context.Context, c capabilities.DeviceDiscovery, a string, b []byte) (any, error) {
	switch a {
	case "Enable":
		input := DeviceDiscoveryEnable{}
		if err := json.Unmarshal(b, &input); err != nil {
			return nil, fmt.Errorf("%w: unable to parse user data: %s", ActionUserError, err.Error())
		}

		duration := time.Duration(input.Duration) * time.Millisecond
		return struct{}{}, c.Enable(ctx, duration)
	case "Disable":
		return struct{}{}, c.Disable(ctx)
	}

	return nil, ActionNotSupported
}

func doEnumerateDevice(ctx context.Context, c capabilities.EnumerateDevice, a string) (any, error) {
	switch a {
	case "Enumerate":
		return struct{}{}, c.Enumerate(ctx)
	}

	return nil, ActionNotSupported
}

func doOnOff(ctx context.Context, c capabilities.OnOff, a string) (any, error) {
	switch a {
	case "On":
		return struct{}{}, c.On(ctx)
	case "Off":
		return struct{}{}, c.Off(ctx)
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

func doAlarmWarningDevice(ctx context.Context, c capabilities.AlarmWarningDevice, a string, b []byte) (any, error) {
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

		return struct{}{}, c.Alarm(ctx, alarmType, input.Volume, input.Visual, duration)
	case "Clear":
		return struct{}{}, c.Clear(ctx)
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

		return struct{}{}, c.Alert(ctx, alarmType, alertType, input.Volume, input.Visual)
	}

	return nil, ActionNotSupported
}

type RemoveDevice struct {
	Force bool
}

func doDeviceRemoval(ctx context.Context, c capabilities.DeviceRemoval, a string, b []byte) (any, error) {
	switch a {
	case "Remove":
		input := RemoveDevice{}
		if err := json.Unmarshal(b, &input); err != nil {
			return nil, fmt.Errorf("%w: unable to parse user data: %s", ActionUserError, err.Error())
		}

		rt := capabilities.Request
		if input.Force {
			rt = capabilities.Force
		}

		return struct{}{}, c.Remove(ctx, rt)
	}

	return nil, ActionNotSupported
}
