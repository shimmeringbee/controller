package exporter

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/controller/state"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
)

type EventExporter interface {
	MapEvent(ctx context.Context, e any) ([]any, error)
	InitialEvents(ctx context.Context) ([]any, error)
}

type eventExporter struct {
	gatewayMapper   state.GatewayMapper
	deviceExporter  DeviceExporter
	deviceOrganiser *state.DeviceOrganiser
}

// EventToCapability maps a device abstraction capabilities event message back to the capability flag.
func eventToCapability(v any) (da.Device, da.Capability, bool) {
	switch e := v.(type) {
	case capabilities.AlarmSensorUpdate:
		return e.Device, capabilities.AlarmSensorFlag, true
	case capabilities.AlarmWarningDeviceUpdate:
		return e.Device, capabilities.AlarmWarningDeviceFlag, true
	case capabilities.DeviceDiscoveryEnabled:
		return e.Gateway.Self(), capabilities.DeviceDiscoveryFlag, true
	case capabilities.DeviceDiscoveryDisabled:
		return e.Gateway.Self(), capabilities.DeviceDiscoveryFlag, true
	case capabilities.EnumerateDeviceStart:
		return e.Device, capabilities.EnumerateDeviceFlag, true
	case capabilities.EnumerateDeviceStopped:
		return e.Device, capabilities.EnumerateDeviceFlag, true
	case capabilities.IdentifyUpdate:
		return e.Device, capabilities.IdentifyFlag, true
	case capabilities.IlluminationSensorUpdate:
		return e.Device, capabilities.IlluminationSensorFlag, true
	case capabilities.LocalDebugStart:
		return e.Device, capabilities.LocalDebugFlag, true
	case capabilities.LocalDebugSuccess:
		return e.Device, capabilities.LocalDebugFlag, true
	case capabilities.LocalDebugFailure:
		return e.Device, capabilities.LocalDebugFlag, true
	case capabilities.MessageCaptureStart:
		return e.Device, capabilities.MessageCaptureDebugFlag, true
	case capabilities.MessageCapture:
		return e.Device, capabilities.MessageCaptureDebugFlag, true
	case capabilities.MessageCaptureStop:
		return e.Device, capabilities.MessageCaptureDebugFlag, true
	case capabilities.OccupancySensorUpdate:
		return e.Device, capabilities.OccupancySensorFlag, true
	case capabilities.OnOffUpdate:
		return e.Device, capabilities.OnOffFlag, true
	case capabilities.PowerStatusUpdate:
		return e.Device, capabilities.PowerSupplyFlag, true
	case capabilities.PressureSensorUpdate:
		return e.Device, capabilities.PressureSensorFlag, true
	case capabilities.RelativeHumiditySensorUpdate:
		return e.Device, capabilities.RelativeHumiditySensorFlag, true
	case capabilities.RemoteDebugStart:
		return e.Device, capabilities.RemoteDebugFlag, true
	case capabilities.RemoteDebugSuccess:
		return e.Device, capabilities.RemoteDebugFlag, true
	case capabilities.RemoteDebugFailure:
		return e.Device, capabilities.RemoteDebugFlag, true
	case capabilities.TemperatureSensorUpdate:
		return e.Device, capabilities.TemperatureSensorFlag, true
	default:
		return nil, 0, false
	}
}

func NewEventExporter(gm state.GatewayMapper, de DeviceExporter, do *state.DeviceOrganiser) EventExporter {
	return &eventExporter{
		gatewayMapper:   gm,
		deviceExporter:  de,
		deviceOrganiser: do,
	}
}

func (w eventExporter) MapEvent(ctx context.Context, v any) ([]any, error) {
	switch e := v.(type) {
	case da.DeviceAdded:
		return w.generateDeviceMessages(ctx, e.Device), nil
	case capabilities.EnumerateDeviceStopped:
		return w.generateDeviceMessages(ctx, e.Device), nil

	case da.DeviceRemoved:
		return w.generateDeviceRemove(e.Device.Identifier())

	case state.DeviceMetadataUpdate:
		if dad, ok := w.gatewayMapper.Device(e.Identifier); ok {
			return w.generateDeviceUpdateMessage(ctx, dad)
		} else {
			return nil, nil
		}
	case state.DeviceAddedToZone:
		if dad, ok := w.gatewayMapper.Device(e.DeviceIdentifier); ok {
			return w.generateDeviceUpdateMessage(ctx, dad)
		} else {
			return nil, nil
		}
	case state.DeviceRemovedFromZone:
		if dad, ok := w.gatewayMapper.Device(e.DeviceIdentifier); ok {
			return w.generateDeviceUpdateMessage(ctx, dad)
		} else {
			return nil, nil
		}

	case state.ZoneCreate:
		return w.generateZoneCreate(e)
	case state.ZoneUpdate:
		return w.generateZoneUpdate(e)
	case state.ZoneRemove:
		return w.generateZoneRemove(e)

	case da.CapabilityAdded:
	case da.CapabilityRemoved:

	default:
		if d, c, found := eventToCapability(e); found {
			return w.generateDeviceUpdateCapabilityMessage(ctx, d, c)
		}
	}

	return nil, fmt.Errorf("unimplemented map event")
}

func (w eventExporter) generateZoneCreate(zc state.ZoneCreate) ([]any, error) {
	return []any{ZoneUpdateMessage{
		ZoneMessage: ZoneMessage{
			Identifier: zc.Identifier,
			Message: Message{
				Type: ZoneUpdateMessageName,
			},
		},
		Name:   zc.Name,
		Parent: 0,
		After:  zc.AfterZone,
	}}, nil
}

func (w eventExporter) generateZoneUpdate(zu state.ZoneUpdate) ([]any, error) {
	return []any{ZoneUpdateMessage{
		ZoneMessage: ZoneMessage{
			Identifier: zu.Identifier,
			Message: Message{
				Type: ZoneUpdateMessageName,
			},
		},
		Name:   zu.Name,
		Parent: zu.ParentZone,
		After:  zu.AfterZone,
	}}, nil
}

func (w eventExporter) generateZoneRemove(zu state.ZoneRemove) ([]any, error) {
	return []any{ZoneRemoveMessage{
		ZoneMessage: ZoneMessage{
			Identifier: zu.Identifier,
			Message: Message{
				Type: ZoneRemoveMessageName,
			},
		},
	}}, nil
}

func (w eventExporter) generateDeviceRemove(identifier da.Identifier) ([]any, error) {
	return []any{DeviceRemoveMessage{
		DeviceMessage: DeviceMessage{
			Message: Message{
				Type: DeviceRemoveMessageName,
			},
		},
		Identifier: identifier.String(),
	}}, nil
}

func (w eventExporter) generateGatewayUpdateMessage(gwName string, gateway da.Gateway) ([]any, error) {
	exportedGw := ExportGateway(gateway)
	exportedGw.Identifier = gwName

	return []any{GatewayUpdateMessage{
		GatewayMessage: GatewayMessage{
			Message: Message{
				Type: GatewayUpdateMessageName,
			},
		},
		ExportedGateway: exportedGw,
	}}, nil
}

func (w eventExporter) generateDeviceUpdateMessage(ctx context.Context, d da.Device) ([]any, error) {
	exportedDevice := w.deviceExporter.ExportSimpleDevice(ctx, d)

	return []any{DeviceUpdateMessage{
		DeviceMessage: DeviceMessage{
			Message: Message{
				Type: DeviceUpdateMessageName,
			},
		},
		ExportedSimpleDevice: exportedDevice,
	}}, nil
}

func (w eventExporter) generateDeviceUpdateCapabilityMessage(ctx context.Context, daDevice da.Device, capFlag da.Capability) ([]any, error) {
	uncastCapability := daDevice.Capability(capFlag)

	basic, ok := uncastCapability.(da.BasicCapability)
	if !ok {
		return nil, nil
	}

	out := w.deviceExporter.ExportCapability(ctx, uncastCapability)

	return []any{
		DeviceUpdateCapabilityMessage{
			DeviceMessage: DeviceMessage{
				Message: Message{
					Type: DeviceUpdateCapabilityMessageName,
				},
			},
			Identifier: daDevice.Identifier().String(),
			Capability: basic.Name(),
			Payload:    out,
		},
	}, nil
}

func (w eventExporter) generateZoneUpdateMessage(zone state.Zone, after int) (any, error) {
	return ZoneUpdateMessage{
		ZoneMessage: ZoneMessage{
			Message: Message{
				Type: ZoneUpdateMessageName,
			},
			Identifier: zone.Identifier,
		},
		Name:   zone.Name,
		Parent: zone.ParentZone,
		After:  after,
	}, nil
}

func (w eventExporter) InitialEvents(ctx context.Context) ([]any, error) {
	var events []any

	after := 0

	for _, zone := range w.deviceOrganiser.RootZones() {
		events = append(events, w.initialEventsZone(zone, after)...)
		after = zone.Identifier
	}

	for gwName, gateway := range w.gatewayMapper.Gateways() {
		if data, err := w.generateGatewayUpdateMessage(gwName, gateway); err == nil {
			events = append(events, data...)
		}

		for _, device := range gateway.Devices() {
			events = append(events, w.generateDeviceMessages(ctx, device)...)
		}
	}

	return events, nil
}

func (w eventExporter) generateDeviceMessages(ctx context.Context, device da.Device) []any {
	var events []any

	if data, err := w.generateDeviceUpdateMessage(ctx, device); err == nil {
		events = append(events, data...)
	}

	for _, capFlag := range device.Capabilities() {
		if data, err := w.generateDeviceUpdateCapabilityMessage(ctx, device, capFlag); err == nil && data != nil {
			events = append(events, data...)
		}
	}

	return events
}

func (w eventExporter) initialEventsZone(zone state.Zone, after int) []any {
	var events []any

	if data, err := w.generateZoneUpdateMessage(zone, after); err == nil {
		events = append(events, data)
	}

	after = 0
	for _, zoneId := range zone.SubZones {
		if z, found := w.deviceOrganiser.Zone(zoneId); found {
			events = append(events, w.initialEventsZone(z, after)...)
			after = zoneId
		}
	}

	return events
}
