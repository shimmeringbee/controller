package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shimmeringbee/controller/state"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/da/capabilities/color"
	"time"
)

type ExportedDevice struct {
	Metadata     state.DeviceMetadata
	Identifier   string
	Capabilities map[string]interface{}
	Gateway      string
}

type ExportedGateway struct {
	Identifier   string
	Capabilities []string
	SelfDevice   string
}

const DefaultCapabilityTimeout = 1 * time.Second

type DeviceExporter struct {
	DeviceOrganiser *state.DeviceOrganiser
	GatewayMapper   state.GatewayMapper
}

func (de *DeviceExporter) ExportDevice(ctx context.Context, daDevice da.Device) ExportedDevice {
	capabilityList := map[string]interface{}{}

	for _, capFlag := range daDevice.Capabilities() {
		uncastCapability := daDevice.Gateway().Capability(capFlag)

		if basicCapability, ok := uncastCapability.(da.BasicCapability); ok {
			capabilityList[basicCapability.Name()] = de.ExportCapability(ctx, daDevice, uncastCapability)
		}
	}

	md, _ := de.DeviceOrganiser.Device(daDevice.Identifier().String())
	gwName, _ := de.GatewayMapper.GatewayName(daDevice.Gateway())

	return ExportedDevice{
		Identifier:   daDevice.Identifier().String(),
		Capabilities: capabilityList,
		Metadata:     md,
		Gateway:      gwName,
	}
}

func (de *DeviceExporter) ExportCapability(pctx context.Context, device da.Device, uncastCapability interface{}) interface{} {
	ctx, cancel := context.WithTimeout(pctx, DefaultCapabilityTimeout)
	defer cancel()

	var retVal interface{}

	switch capability := uncastCapability.(type) {
	case capabilities.HasProductInformation:
		retVal = de.convertHasProductInformation(ctx, device, capability)
	case capabilities.TemperatureSensor:
		retVal = de.convertTemperatureSensor(ctx, device, capability)
	case capabilities.RelativeHumiditySensor:
		retVal = de.convertRelativeHumiditySensor(ctx, device, capability)
	case capabilities.PressureSensor:
		retVal = de.convertPressureSensor(ctx, device, capability)
	case capabilities.DeviceDiscovery:
		retVal = de.convertDeviceDiscovery(ctx, device, capability)
	case capabilities.EnumerateDevice:
		retVal = de.convertEnumerateDevice(ctx, device, capability)
	case capabilities.AlarmSensor:
		retVal = de.convertAlarmSensor(ctx, device, capability)
	case capabilities.OnOff:
		retVal = de.convertOnOff(ctx, device, capability)
	case capabilities.PowerSupply:
		retVal = de.convertPowerSupply(ctx, device, capability)
	case capabilities.AlarmWarningDevice:
		retVal = de.convertAlarmWarningDevice(ctx, device, capability)
	case capabilities.Level:
		retVal = de.convertLevel(ctx, device, capability)
	case capabilities.Color:
		retVal = de.convertColor(ctx, device, capability)
	default:
		return struct{}{}
	}

	if capWithLUT, ok := uncastCapability.(capabilities.WithLastUpdateTime); ok {
		if retWithSUT, ok := retVal.(SettableUpdateTime); ok {
			if lut, err := capWithLUT.LastUpdateTime(ctx, device); err == nil {
				retWithSUT.SetUpdateTime(lut)
			}
		}
	}

	if capWithLCT, ok := uncastCapability.(capabilities.WithLastChangeTime); ok {
		if retWithSCT, ok := retVal.(SettableChangeTime); ok {
			if lut, err := capWithLCT.LastChangeTime(ctx, device); err == nil {
				retWithSCT.SetChangeTime(lut)
			}
		}
	}

	return retVal
}

type SettableUpdateTime interface {
	SetUpdateTime(time.Time)
}

type SettableChangeTime interface {
	SetChangeTime(time.Time)
}

type NullableTime time.Time

func (n NullableTime) MarshalJSON() ([]byte, error) {
	under := time.Time(n)

	if under.IsZero() {
		return []byte("null"), nil
	} else {
		return json.Marshal(under)
	}
}

type LastUpdate struct {
	LastUpdate *NullableTime `json:",omitempty"`
}

func (lut *LastUpdate) SetUpdateTime(t time.Time) {
	nullableTime := NullableTime(t)
	lut.LastUpdate = &nullableTime
}

type LastChange struct {
	LastChange *NullableTime `json:",omitempty"`
}

func (lct *LastChange) SetChangeTime(t time.Time) {
	nullableTime := NullableTime(t)
	lct.LastChange = &nullableTime
}

type HasProductInformation struct {
	Name         string `json:",omitempty"`
	Manufacturer string `json:",omitempty"`
	Serial       string `json:",omitempty"`
}

func (de *DeviceExporter) convertHasProductInformation(ctx context.Context, device da.Device, hpi capabilities.HasProductInformation) interface{} {
	pi, err := hpi.ProductInformation(ctx, device)
	if err != nil {
		return nil
	}

	return &HasProductInformation{
		Name:         pi.Name,
		Manufacturer: pi.Manufacturer,
		Serial:       pi.Serial,
	}
}

type TemperatureSensor struct {
	Readings []capabilities.TemperatureReading
	LastUpdate
	LastChange
}

func (de *DeviceExporter) convertTemperatureSensor(ctx context.Context, device da.Device, ts capabilities.TemperatureSensor) interface{} {
	tsReadings, err := ts.Reading(ctx, device)
	if err != nil {
		return nil
	}

	return &TemperatureSensor{
		Readings: tsReadings,
	}
}

type RelativeHumiditySensor struct {
	Readings []capabilities.RelativeHumidityReading
	LastUpdate
	LastChange
}

func (de *DeviceExporter) convertRelativeHumiditySensor(ctx context.Context, device da.Device, rhs capabilities.RelativeHumiditySensor) interface{} {
	rhReadings, err := rhs.Reading(ctx, device)
	if err != nil {
		return nil
	}

	return &RelativeHumiditySensor{
		Readings: rhReadings,
	}
}

type PressureSensor struct {
	Readings []capabilities.PressureReading
	LastUpdate
	LastChange
}

func (de *DeviceExporter) convertPressureSensor(ctx context.Context, device da.Device, ps capabilities.PressureSensor) interface{} {
	psReadings, err := ps.Reading(ctx, device)
	if err != nil {
		return nil
	}

	return &PressureSensor{
		Readings: psReadings,
	}
}

type DeviceDiscovery struct {
	Discovering bool
	Duration    int `json:",omitempty"`
	LastUpdate
	LastChange
}

func (de *DeviceExporter) convertDeviceDiscovery(ctx context.Context, device da.Device, dd capabilities.DeviceDiscovery) interface{} {
	discoveryState, err := dd.Status(ctx, device)
	if err != nil {
		return nil
	}

	remainingMilliseconds := int(discoveryState.RemainingDuration / time.Millisecond)

	return &DeviceDiscovery{
		Discovering: discoveryState.Discovering,
		Duration:    remainingMilliseconds,
	}
}

type EnumerateDevice struct {
	Enumerating bool
	LastUpdate
	LastChange
}

func (de *DeviceExporter) convertEnumerateDevice(ctx context.Context, device da.Device, ed capabilities.EnumerateDevice) interface{} {
	enumerateDeviceState, err := ed.Status(ctx, device)
	if err != nil {
		return nil
	}

	return &EnumerateDevice{
		Enumerating: enumerateDeviceState.Enumerating,
	}
}

type AlarmSensor struct {
	Alarms map[string]bool
	LastUpdate
	LastChange
}

func (de *DeviceExporter) convertAlarmSensor(ctx context.Context, device da.Device, as capabilities.AlarmSensor) interface{} {
	alarmSensorState, err := as.Status(ctx, device)
	if err != nil {
		return nil
	}

	alarms := map[string]bool{}

	for k, v := range alarmSensorState {
		alarms[k.String()] = v
	}

	return &AlarmSensor{
		Alarms: alarms,
	}
}

type OnOff struct {
	State bool
	LastUpdate
	LastChange
}

func (de *DeviceExporter) convertOnOff(ctx context.Context, device da.Device, oo capabilities.OnOff) interface{} {
	state, err := oo.Status(ctx, device)
	if err != nil {
		return nil
	}

	return &OnOff{
		State: state,
	}
}

type PowerStatus struct {
	Mains   []PowerMainsStatus   `json:",omitempty"`
	Battery []PowerBatteryStatus `json:",omitempty"`
	LastUpdate
	LastChange
}

type PowerMainsStatus struct {
	Voltage   *float64 `json:",omitempty"`
	Frequency *float64 `json:",omitempty"`
	Available *bool    `json:",omitempty"`
}

type PowerBatteryStatus struct {
	Voltage        *float64 `json:",omitempty"`
	MaximumVoltage *float64 `json:",omitempty"`
	MinimumVoltage *float64 `json:",omitempty"`
	Remaining      *float64 `json:",omitempty"`
	Available      *bool    `json:",omitempty"`
}

func (de *DeviceExporter) convertPowerSupply(ctx context.Context, d da.Device, capability capabilities.PowerSupply) interface{} {
	state, err := capability.Status(ctx, d)
	if err != nil {
		return nil
	}

	var mains []PowerMainsStatus
	var battery []PowerBatteryStatus

	for _, m := range state.Mains {
		newMains := PowerMainsStatus{}

		if m.Present&capabilities.Voltage == capabilities.Voltage {
			newMains.Voltage = &m.Voltage
		}

		if m.Present&capabilities.Frequency == capabilities.Frequency {
			newMains.Frequency = &m.Frequency
		}

		if m.Present&capabilities.Available == capabilities.Available {
			newMains.Available = &m.Available
		}

		mains = append(mains, newMains)
	}

	for _, b := range state.Battery {
		newBattery := PowerBatteryStatus{}

		if b.Present&capabilities.Voltage == capabilities.Voltage {
			newBattery.Voltage = &b.Voltage
		}

		if b.Present&capabilities.MaximumVoltage == capabilities.MaximumVoltage {
			newBattery.MaximumVoltage = &b.MaximumVoltage
		}

		if b.Present&capabilities.MinimumVoltage == capabilities.MinimumVoltage {
			newBattery.MinimumVoltage = &b.MinimumVoltage
		}

		if b.Present&capabilities.Remaining == capabilities.Remaining {
			newBattery.Remaining = &b.Remaining
		}

		if b.Present&capabilities.Available == capabilities.Available {
			newBattery.Available = &b.Available
		}

		battery = append(battery, newBattery)
	}

	return &PowerStatus{
		Mains:   mains,
		Battery: battery,
	}
}

type AlarmWarningDeviceStatus struct {
	Warning   bool
	AlarmType *string  `json:",omitempty"`
	Volume    *float64 `json:",omitempty"`
	Visual    *bool    `json:",omitempty"`
	Duration  *int     `json:",omitempty"`
	LastUpdate
	LastChange
}

func (de *DeviceExporter) convertAlarmWarningDevice(ctx context.Context, d da.Device, capability capabilities.AlarmWarningDevice) interface{} {
	state, err := capability.Status(ctx, d)
	if err != nil {
		return nil
	}

	status := &AlarmWarningDeviceStatus{
		Warning: state.Warning,
	}

	if state.Warning {
		alarmType := state.AlarmType.String()
		duration := int(state.DurationRemaining / time.Millisecond)

		status.Volume = &state.Volume
		status.Visual = &state.Visual
		status.AlarmType = &alarmType
		status.Duration = &duration
	}

	return status
}

type Level struct {
	Current           float64
	Target            float64 `json:",omitempty"`
	DurationRemaining int     `json:",omitempty"`
	LastUpdate
	LastChange
}

func (de *DeviceExporter) convertLevel(ctx context.Context, device da.Device, l capabilities.Level) interface{} {
	state, err := l.Status(ctx, device)
	if err != nil {
		return nil
	}

	durationInMilliseconds := int(state.DurationRemaining / time.Millisecond)

	return &Level{
		Current:           state.CurrentLevel,
		Target:            state.TargetLevel,
		DurationRemaining: durationInMilliseconds,
	}
}

type ColorOutputXYY struct {
	X  float64
	Y  float64
	Y2 float64
}

type ColorOutputHSV struct {
	Hue        float64
	Saturation float64
	Value      float64
}

type ColorOutputRGB struct {
	R uint8
	G uint8
	B uint8
}

type ColorOutput struct {
	XYY ColorOutputXYY
	HSV ColorOutputHSV
	RGB ColorOutputRGB
	Hex string
}

type ColorState struct {
	Temperature float64      `json:",omitempty"`
	Color       *ColorOutput `json:",omitempty"`
}

type ColorSupports struct {
	Color       bool
	Temperature bool
}

type Color struct {
	Current           *ColorState
	Target            *ColorState `json:",omitempty"`
	DurationRemaining int         `json:",omitempty"`
	Supports          ColorSupports
	LastUpdate
	LastChange
}

func (de *DeviceExporter) convertColor(ctx context.Context, device da.Device, c capabilities.Color) interface{} {
	state, err := c.Status(ctx, device)
	if err != nil {
		return nil
	}

	supportsColor, err := c.SupportsColor(ctx, device)
	if err != nil {
		return nil
	}

	supportsTemperature, err := c.SupportsTemperature(ctx, device)
	if err != nil {
		return nil
	}

	durationInMilliseconds := int(state.DurationRemaining / time.Millisecond)

	currentColor := ColorState{}
	var targetColor *ColorState

	if state.Mode == capabilities.TemperatureMode {
		currentColor.Temperature = state.Temperature.Current

		if state.Temperature.Target > 0 {
			targetColor = &ColorState{}
			targetColor.Temperature = state.Temperature.Target
		}
	} else {
		currentColor.Color = convertConvertibleColorToColorOutput(state.Color.Current)

		if state.Color.Target != nil {
			targetColor = &ColorState{}
			targetColor.Color = convertConvertibleColorToColorOutput(state.Color.Current)
		}
	}

	return &Color{
		Current:           &currentColor,
		Target:            targetColor,
		DurationRemaining: durationInMilliseconds,
		Supports: ColorSupports{
			Color:       supportsColor,
			Temperature: supportsTemperature,
		},
	}
}

func convertConvertibleColorToColorOutput(current color.ConvertibleColor) *ColorOutput {
	x, y, y2 := current.XYY()
	h, s, v := current.HSV()
	r, g, b := current.RGB()
	hex := fmt.Sprintf("%02x%02x%02x", r, g, b)

	return &ColorOutput{
		XYY: ColorOutputXYY{
			X:  x,
			Y:  y,
			Y2: y2,
		},
		HSV: ColorOutputHSV{
			Hue:        h,
			Saturation: s,
			Value:      v,
		},
		RGB: ColorOutputRGB{
			R: r,
			G: g,
			B: b,
		},
		Hex: hex,
	}
}
