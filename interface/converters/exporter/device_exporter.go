package exporter

import (
	"context"
	"encoding/json"
	"github.com/shimmeringbee/controller/state"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
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
		uncastCapability := daDevice.Capability(capFlag)

		if basicCapability, ok := uncastCapability.(da.BasicCapability); ok {
			capabilityList[basicCapability.Name()] = de.ExportCapability(ctx, uncastCapability)
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

func (de *DeviceExporter) ExportCapability(pctx context.Context, uncastCapability interface{}) interface{} {
	ctx, cancel := context.WithTimeout(pctx, DefaultCapabilityTimeout)
	defer cancel()

	var retVal interface{}

	switch capability := uncastCapability.(type) {
	case capabilities.ProductInformation:
		retVal = de.convertHasProductInformation(ctx, capability)
	case capabilities.TemperatureSensor:
		retVal = de.convertTemperatureSensor(ctx, capability)
	case capabilities.RelativeHumiditySensor:
		retVal = de.convertRelativeHumiditySensor(ctx, capability)
	case capabilities.PressureSensor:
		retVal = de.convertPressureSensor(ctx, capability)
	case capabilities.DeviceDiscovery:
		retVal = de.convertDeviceDiscovery(ctx, capability)
	case capabilities.EnumerateDevice:
		retVal = de.convertEnumerateDevice(ctx, capability)
	case capabilities.Identify:
		retVal = de.convertIdentify(ctx, capability)
	case capabilities.AlarmSensor:
		retVal = de.convertAlarmSensor(ctx, capability)
	case capabilities.OnOff:
		retVal = de.convertOnOff(ctx, capability)
	case capabilities.PowerSupply:
		retVal = de.convertPowerSupply(ctx, capability)
	case capabilities.AlarmWarningDevice:
		retVal = de.convertAlarmWarningDevice(ctx, capability)
	case capabilities.DeviceWorkarounds:
		retVal = de.convertDeviceWorkarounds(ctx, capability)
	default:
		return struct{}{}
	}

	if capWithLUT, ok := uncastCapability.(capabilities.WithLastUpdateTime); ok {
		if retWithSUT, ok := retVal.(SettableUpdateTime); ok {
			if lut, err := capWithLUT.LastUpdateTime(ctx); err == nil {
				retWithSUT.SetUpdateTime(lut)
			}
		}
	}

	if capWithLCT, ok := uncastCapability.(capabilities.WithLastChangeTime); ok {
		if retWithSCT, ok := retVal.(SettableChangeTime); ok {
			if lut, err := capWithLCT.LastChangeTime(ctx); err == nil {
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
	Version      string `json:",omitempty"`
}

func (de *DeviceExporter) convertHasProductInformation(ctx context.Context, hpi capabilities.ProductInformation) interface{} {
	pi, err := hpi.Get(ctx)
	if err != nil {
		return nil
	}

	return &HasProductInformation{
		Name:         pi.Name,
		Manufacturer: pi.Manufacturer,
		Serial:       pi.Serial,
		Version:      pi.Version,
	}
}

type TemperatureSensor struct {
	Readings []capabilities.TemperatureReading
	LastUpdate
	LastChange
}

func (de *DeviceExporter) convertTemperatureSensor(ctx context.Context, ts capabilities.TemperatureSensor) interface{} {
	tsReadings, err := ts.Reading(ctx)
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

func (de *DeviceExporter) convertRelativeHumiditySensor(ctx context.Context, rhs capabilities.RelativeHumiditySensor) interface{} {
	rhReadings, err := rhs.Reading(ctx)
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

func (de *DeviceExporter) convertPressureSensor(ctx context.Context, ps capabilities.PressureSensor) interface{} {
	psReadings, err := ps.Reading(ctx)
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

func (de *DeviceExporter) convertDeviceDiscovery(ctx context.Context, dd capabilities.DeviceDiscovery) interface{} {
	discoveryState, err := dd.Status(ctx)
	if err != nil {
		return nil
	}

	remainingMilliseconds := int(discoveryState.RemainingDuration / time.Millisecond)

	return &DeviceDiscovery{
		Discovering: discoveryState.Discovering,
		Duration:    remainingMilliseconds,
	}
}

type EnumerateDeviceCapability struct {
	Attached bool
	Errors   []string
}

type EnumerateDevice struct {
	Enumerating bool
	Status      map[string]EnumerateDeviceCapability
	LastUpdate
	LastChange
}

func (de *DeviceExporter) convertEnumerateDevice(ctx context.Context, ed capabilities.EnumerateDevice) interface{} {
	enumerateDeviceState, err := ed.Status(ctx)
	if err != nil {
		return nil
	}

	results := map[string]EnumerateDeviceCapability{}

	for c, ec := range enumerateDeviceState.CapabilityStatus {
		var errorText []string

		for _, e := range ec.Errors {
			errorText = append(errorText, e.Error())
		}

		results[capabilities.StandardNames[c]] = EnumerateDeviceCapability{
			Attached: ec.Attached,
			Errors:   errorText,
		}
	}

	return &EnumerateDevice{
		Enumerating: enumerateDeviceState.Enumerating,
		Status:      results,
	}
}

type AlarmSensor struct {
	Alarms map[string]bool
	LastUpdate
	LastChange
}

func (de *DeviceExporter) convertAlarmSensor(ctx context.Context, as capabilities.AlarmSensor) interface{} {
	alarmSensorState, err := as.Status(ctx)
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

func (de *DeviceExporter) convertOnOff(ctx context.Context, oo capabilities.OnOff) interface{} {
	state, err := oo.Status(ctx)
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

func (de *DeviceExporter) convertPowerSupply(ctx context.Context, capability capabilities.PowerSupply) interface{} {
	state, err := capability.Status(ctx)
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

func (de *DeviceExporter) convertAlarmWarningDevice(ctx context.Context, capability capabilities.AlarmWarningDevice) interface{} {
	state, err := capability.Status(ctx)
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

type IdentifyStatus struct {
	Identifying bool
	Duration    *int `json:",omitempty"`

	LastUpdate
	LastChange
}

func (de *DeviceExporter) convertIdentify(ctx context.Context, capability capabilities.Identify) interface{} {
	state, err := capability.Status(ctx)
	if err != nil {
		return nil
	}

	status := &IdentifyStatus{
		Identifying: state.Identifying,
	}

	if state.Identifying {
		duration := int(state.Remaining / time.Millisecond)
		status.Duration = &duration
	}

	return status
}

type DeviceWorkaroundsStatus struct {
	Enabled []string
}

func (de *DeviceExporter) convertDeviceWorkarounds(ctx context.Context, capability capabilities.DeviceWorkarounds) interface{} {
	state, err := capability.Enabled(ctx)
	if err != nil {
		return nil
	}

	status := &DeviceWorkaroundsStatus{
		Enabled: state,
	}

	return status
}
