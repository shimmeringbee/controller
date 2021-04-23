package invoker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shimmeringbee/controller/layers"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	color2 "github.com/shimmeringbee/da/capabilities/color"
	"time"
)

type Invoker func(ctx context.Context, s layers.OutputStack, l string, r layers.RetentionLevel, dad da.Device, capabilityName string, actionName string, payload []byte) (interface{}, error)

type ActionError string

func (e ActionError) Error() string {
	return string(e)
}

const CapabilityNotSupported = ActionError("capability not available on device")
const ActionNotSupported = ActionError("action not available on capability")
const ActionUserError = ActionError("user provided bad data")

func InvokeDeviceAction(ctx context.Context, s layers.OutputStack, l string, r layers.RetentionLevel, dad da.Device, capabilityName string, actionName string, payload []byte) (interface{}, error) {
	l, r, err := resolveOutputLayerAndRetention(l, r, payload)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal payload: %w", err)
	}

	o := s.Lookup(l)

	for _, capFlag := range dad.Capabilities() {
		uncastCap := o.Capability(r, capFlag, dad)

		if uncastCap != nil {
			if castCap, ok := uncastCap.(da.BasicCapability); ok {
				if castCap.Name() == capabilityName {
					return doDeviceCapabilityAction(ctx, dad, uncastCap, actionName, payload)
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
	case capabilities.Level:
		return doLevel(ctx, d, cast, a, b)
	case capabilities.Color:
		return doColor(ctx, d, cast, a, b)
	case capabilities.DeviceRemoval:
		return doDeviceRemoval(ctx, d, cast, a, b)
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

type LevelChange struct {
	Level    float64
	Duration int
}

func doLevel(ctx context.Context, d da.Device, c capabilities.Level, a string, b []byte) (interface{}, error) {
	switch a {
	case "Change":
		input := LevelChange{}
		if err := json.Unmarshal(b, &input); err != nil {
			return nil, fmt.Errorf("%w: unable to parse user data: %s", ActionUserError, err.Error())
		}

		duration := time.Duration(input.Duration) * time.Millisecond
		return struct{}{}, c.Change(ctx, d, input.Level, duration)
	}

	return nil, ActionNotSupported
}

type ColorChangeTemperature struct {
	Temperature float64
	Duration    int
}

type ColorChangeColorXYY struct {
	X  float64
	Y  float64
	Y2 float64
}

type ColorChangeColorHSV struct {
	Hue        float64
	Saturation float64
	Value      float64
}

type ColorChangeColorRGB struct {
	R uint8
	G uint8
	B uint8
}

type ColorChangeColor struct {
	XYY      *ColorChangeColorXYY
	HSV      *ColorChangeColorHSV
	RGB      *ColorChangeColorRGB
	Duration int
}

func doColor(ctx context.Context, d da.Device, c capabilities.Color, a string, b []byte) (interface{}, error) {
	switch a {
	case "ChangeColor":
		input := ColorChangeColor{}
		if err := json.Unmarshal(b, &input); err != nil {
			return nil, fmt.Errorf("%w: unable to parse user data: %s", ActionUserError, err.Error())
		}

		var color color2.ConvertibleColor

		if input.XYY != nil {
			color = color2.XYColor{
				X:  input.XYY.X,
				Y:  input.XYY.Y,
				Y2: input.XYY.Y2,
			}
		} else if input.HSV != nil {
			color = color2.HSVColor{
				Hue:   input.HSV.Hue,
				Sat:   input.HSV.Saturation,
				Value: input.HSV.Value,
			}
		} else if input.RGB != nil {
			color = color2.SRGBColor{
				R: input.RGB.R,
				G: input.RGB.G,
				B: input.RGB.B,
			}
		} else {
			return nil, fmt.Errorf("%w: unable to parse user data: %s", ActionUserError, fmt.Errorf("no recognised color"))
		}

		duration := time.Duration(input.Duration) * time.Millisecond
		return struct{}{}, c.ChangeColor(ctx, d, color, duration)
	case "ChangeTemperature":
		input := ColorChangeTemperature{}
		if err := json.Unmarshal(b, &input); err != nil {
			return nil, fmt.Errorf("%w: unable to parse user data: %s", ActionUserError, err.Error())
		}

		duration := time.Duration(input.Duration) * time.Millisecond
		return struct{}{}, c.ChangeTemperature(ctx, d, input.Temperature, duration)
	}

	return nil, ActionNotSupported
}

func doDeviceRemoval(ctx context.Context, d da.Device, c capabilities.DeviceRemoval, a string, b []byte) (interface{}, error) {
	switch a {
	case "Remove":
		return struct{}{}, c.Remove(ctx, d)
	}

	return nil, ActionNotSupported
}
