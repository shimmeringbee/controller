package v1

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"time"
)

const DefaultCapabilityTimeout = 1 * time.Second

func convertDADeviceToDevice(ctx context.Context, daDevice da.Device) device {
	capabilityList := map[string]interface{}{}

	for _, capFlag := range daDevice.Capabilities() {
		uncastCapability := daDevice.Gateway().Capability(capFlag)

		if basicCapability, ok := uncastCapability.(da.BasicCapability); ok {
			capabilityList[basicCapability.Name()] = convertDADeviceCapability(ctx, daDevice, uncastCapability)
		}
	}

	return device{
		Identifier:   daDevice.Identifier().String(),
		Capabilities: capabilityList,
	}
}

func convertDADeviceCapability(pctx context.Context, device da.Device, uncastCapability interface{}) interface{} {
	ctx, cancel := context.WithTimeout(pctx, DefaultCapabilityTimeout)
	defer cancel()

	switch capability := uncastCapability.(type) {
	case capabilities.HasProductInformation:
		return convertHasProductInformation(ctx, device, capability)
	case capabilities.TemperatureSensor:
		return convertTemperatureSensor(ctx, device, capability)
	case capabilities.RelativeHumiditySensor:
		return convertRelativeHumiditySensor(ctx, device, capability)
	case capabilities.PressureSensor:
		return convertPressureSensor(ctx, device, capability)
	case capabilities.DeviceDiscovery:
		return convertDeviceDiscovery(ctx, device, capability)
	case capabilities.EnumerateDevice:
		return convertEnumerateDevice(ctx, device, capability)
	case capabilities.AlarmSensor:
		return convertAlarmSensor(ctx, device, capability)
	case capabilities.OnOff:
		return convertOnOff(ctx, device, capability)
	default:
		return struct{}{}
	}
}

type HasProductInformation struct {
	Name         string `json:",omitempty"`
	Manufacturer string `json:",omitempty"`
	Serial       string `json:",omitempty"`
}

func convertHasProductInformation(ctx context.Context, device da.Device, hpi capabilities.HasProductInformation) interface{} {
	pi, err := hpi.ProductInformation(ctx, device)
	if err != nil {
		return nil
	}

	return HasProductInformation{
		Name:         pi.Name,
		Manufacturer: pi.Manufacturer,
		Serial:       pi.Serial,
	}
}

type TemperatureSensor struct {
	Readings []capabilities.TemperatureReading
}

func convertTemperatureSensor(ctx context.Context, device da.Device, ts capabilities.TemperatureSensor) interface{} {
	tsReadings, err := ts.Reading(ctx, device)
	if err != nil {
		return nil
	}

	return TemperatureSensor{
		Readings: tsReadings,
	}
}

type RelativeHumiditySensor struct {
	Readings []capabilities.RelativeHumidityReading
}

func convertRelativeHumiditySensor(ctx context.Context, device da.Device, rhs capabilities.RelativeHumiditySensor) interface{} {
	rhReadings, err := rhs.Reading(ctx, device)
	if err != nil {
		return nil
	}

	return RelativeHumiditySensor{
		Readings: rhReadings,
	}
}

type PressureSensor struct {
	Readings []capabilities.PressureReading
}

func convertPressureSensor(ctx context.Context, device da.Device, ps capabilities.PressureSensor) interface{} {
	psReadings, err := ps.Reading(ctx, device)
	if err != nil {
		return nil
	}

	return PressureSensor{
		Readings: psReadings,
	}
}

type DeviceDiscovery struct {
	Discovering bool
	Duration    int `json:",omitempty"`
}

func convertDeviceDiscovery(ctx context.Context, device da.Device, dd capabilities.DeviceDiscovery) interface{} {
	discoveryState, err := dd.Status(ctx, device)
	if err != nil {
		return nil
	}

	remainingMilliseconds := int(discoveryState.RemainingDuration / time.Millisecond)

	return DeviceDiscovery{
		Discovering: discoveryState.Discovering,
		Duration:    remainingMilliseconds,
	}
}

type EnumerateDevice struct {
	Enumerating bool
}

func convertEnumerateDevice(ctx context.Context, device da.Device, ed capabilities.EnumerateDevice) interface{} {
	enumerateDeviceState, err := ed.Status(ctx, device)
	if err != nil {
		return nil
	}

	return EnumerateDevice{
		Enumerating: enumerateDeviceState.Enumerating,
	}
}

type AlarmSensor struct {
	Alarms map[string]bool
}

func convertAlarmSensor(ctx context.Context, device da.Device, as capabilities.AlarmSensor) interface{} {
	alarmSensorState, err := as.Status(ctx, device)
	if err != nil {
		return nil
	}

	alarms := map[string]bool{}

	for _, v := range alarmSensorState {
		alarms[v.SensorType.String()] = v.InAlarm
	}

	return AlarmSensor{
		Alarms: alarms,
	}
}

type OnOff struct {
	State bool
}

func convertOnOff(ctx context.Context, device da.Device, oo capabilities.OnOff) interface{} {
	state, err := oo.State(ctx, device)
	if err != nil {
		return nil
	}

	return OnOff{
		State: state,
	}
}
