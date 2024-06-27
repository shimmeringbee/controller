package exporter

import (
	"context"
	"github.com/shimmeringbee/controller/state"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"time"
)

type ExportedDevice struct {
	Metadata     state.DeviceMetadata
	Identifier   string
	Capabilities map[string]any
	Gateway      string
}

type ExportedSimpleDevice struct {
	Metadata     state.DeviceMetadata
	Identifier   string
	Capabilities []string
	Gateway      string
}

type ExportedGateway struct {
	Identifier   string
	Capabilities []string
	SelfDevice   string
}

const DefaultCapabilityTimeout = 1 * time.Second

type deviceExporter struct {
	DeviceOrganiser *state.DeviceOrganiser
	GatewayMapper   state.GatewayMapper
}

func NewDeviceExporter(do *state.DeviceOrganiser, gm state.GatewayMapper) DeviceExporter {
	return &deviceExporter{
		DeviceOrganiser: do,
		GatewayMapper:   gm,
	}
}

func (de *deviceExporter) ExportDevice(ctx context.Context, daDevice da.Device) ExportedDevice {
	capabilityList := map[string]any{}

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

func (de *deviceExporter) ExportSimpleDevice(ctx context.Context, daDevice da.Device) ExportedSimpleDevice {
	capabilityList := []string{}

	for _, capFlag := range daDevice.Capabilities() {
		uncastCapability := daDevice.Capability(capFlag)

		if basicCapability, ok := uncastCapability.(da.BasicCapability); ok {
			capabilityList = append(capabilityList, basicCapability.Name())
		}
	}

	md, _ := de.DeviceOrganiser.Device(daDevice.Identifier().String())
	gwName, _ := de.GatewayMapper.GatewayName(daDevice.Gateway())

	return ExportedSimpleDevice{
		Identifier:   daDevice.Identifier().String(),
		Capabilities: capabilityList,
		Metadata:     md,
		Gateway:      gwName,
	}
}

func (de *deviceExporter) ExportCapability(pctx context.Context, uncastCapability any) any {
	ctx, cancel := context.WithTimeout(pctx, DefaultCapabilityTimeout)
	defer cancel()

	var retVal any

	switch capability := uncastCapability.(type) {
	case capabilities.ProductInformation:
		retVal = de.convertProductInformation(ctx, capability)
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

func (de *deviceExporter) convertProductInformation(ctx context.Context, hpi capabilities.ProductInformation) any {
	pi, err := hpi.Get(ctx)
	if err != nil {
		return nil
	}

	return &ProductInformation{
		Name:         pi.Name,
		Manufacturer: pi.Manufacturer,
		Serial:       pi.Serial,
		Version:      pi.Version,
	}
}

func (de *deviceExporter) convertTemperatureSensor(ctx context.Context, ts capabilities.TemperatureSensor) any {
	tsReadings, err := ts.Reading(ctx)
	if err != nil {
		return nil
	}

	return &TemperatureSensor{
		Readings: tsReadings,
	}
}

func (de *deviceExporter) convertRelativeHumiditySensor(ctx context.Context, rhs capabilities.RelativeHumiditySensor) any {
	rhReadings, err := rhs.Reading(ctx)
	if err != nil {
		return nil
	}

	return &RelativeHumiditySensor{
		Readings: rhReadings,
	}
}

func (de *deviceExporter) convertPressureSensor(ctx context.Context, ps capabilities.PressureSensor) any {
	psReadings, err := ps.Reading(ctx)
	if err != nil {
		return nil
	}

	return &PressureSensor{
		Readings: psReadings,
	}
}

func (de *deviceExporter) convertDeviceDiscovery(ctx context.Context, dd capabilities.DeviceDiscovery) any {
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

func (de *deviceExporter) convertEnumerateDevice(ctx context.Context, ed capabilities.EnumerateDevice) any {
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

func (de *deviceExporter) convertAlarmSensor(ctx context.Context, as capabilities.AlarmSensor) any {
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

func (de *deviceExporter) convertOnOff(ctx context.Context, oo capabilities.OnOff) any {
	state, err := oo.Status(ctx)
	if err != nil {
		return nil
	}

	return &OnOff{
		State: state,
	}
}

func (de *deviceExporter) convertPowerSupply(ctx context.Context, capability capabilities.PowerSupply) any {
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

func (de *deviceExporter) convertAlarmWarningDevice(ctx context.Context, capability capabilities.AlarmWarningDevice) any {
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

func (de *deviceExporter) convertIdentify(ctx context.Context, capability capabilities.Identify) any {
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

func (de *deviceExporter) convertDeviceWorkarounds(ctx context.Context, capability capabilities.DeviceWorkarounds) any {
	state, err := capability.Enabled(ctx)
	if err != nil {
		return nil
	}

	status := &DeviceWorkaroundsStatus{
		Enabled: state,
	}

	return status
}

type DeviceExporter interface {
	ExportDevice(context.Context, da.Device) ExportedDevice
	ExportSimpleDevice(context.Context, da.Device) ExportedSimpleDevice
	ExportCapability(context.Context, any) any
}
