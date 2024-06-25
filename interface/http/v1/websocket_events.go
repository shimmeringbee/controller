package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shimmeringbee/controller/interface/converters/exporter"
	"github.com/shimmeringbee/controller/state"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
)

type eventMapper interface {
	MapEvent(ctx context.Context, e interface{}) ([][]byte, error)
	InitialEvents(ctx context.Context) ([][]byte, error)
}

var _ eventMapper = (*websocketEventMapper)(nil)

type websocketEventMapper struct {
	gatewayMapper   state.GatewayMapper
	deviceExporter  deviceExporter
	deviceOrganiser *state.DeviceOrganiser
}

// EventToCapability maps a device abstraction capabilities event message back to the capability flag.
func eventToCapability(v interface{}) (da.Device, da.Capability, bool) {
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

func (w websocketEventMapper) MapEvent(ctx context.Context, v interface{}) ([][]byte, error) {
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

	default:
		if d, c, found := eventToCapability(e); found {
			return w.generateDeviceUpdateCapabilityMessage(ctx, d, c)
		}
	}

	return nil, fmt.Errorf("unimplemented map event")
}

func (w websocketEventMapper) generateZoneCreate(zc state.ZoneCreate) ([][]byte, error) {
	data, err := json.Marshal(ZoneUpdateMessage{
		ZoneMessage: ZoneMessage{
			Identifier: zc.Identifier,
			Message: Message{
				Type: ZoneUpdateMessageName,
			},
		},
		Name:   zc.Name,
		Parent: 0,
		After:  zc.AfterZone,
	})

	return [][]byte{data}, err
}

func (w websocketEventMapper) generateZoneUpdate(zu state.ZoneUpdate) ([][]byte, error) {
	data, err := json.Marshal(ZoneUpdateMessage{
		ZoneMessage: ZoneMessage{
			Identifier: zu.Identifier,
			Message: Message{
				Type: ZoneUpdateMessageName,
			},
		},
		Name:   zu.Name,
		Parent: zu.ParentZone,
		After:  zu.AfterZone,
	})

	return [][]byte{data}, err
}

func (w websocketEventMapper) generateZoneRemove(zu state.ZoneRemove) ([][]byte, error) {
	data, err := json.Marshal(ZoneRemoveMessage{
		ZoneMessage: ZoneMessage{
			Identifier: zu.Identifier,
			Message: Message{
				Type: ZoneRemoveMessageName,
			},
		},
	})

	return [][]byte{data}, err
}

func (w websocketEventMapper) generateDeviceRemove(identifier da.Identifier) ([][]byte, error) {
	data, err := json.Marshal(DeviceRemoveMessage{
		DeviceMessage: DeviceMessage{
			Message: Message{
				Type: DeviceRemoveMessageName,
			},
		},
		Identifier: identifier.String(),
	})

	return [][]byte{data}, err
}

func (w websocketEventMapper) generateGatewayUpdateMessage(gwName string, gateway da.Gateway) ([][]byte, error) {
	exportedGw := exporter.ExportGateway(gateway)
	exportedGw.Identifier = gwName

	data, err := json.Marshal(GatewayUpdateMessage{
		GatewayMessage: GatewayMessage{
			Message: Message{
				Type: GatewayUpdateMessageName,
			},
		},
		ExportedGateway: exportedGw,
	})

	return [][]byte{data}, err
}

func (w websocketEventMapper) generateDeviceUpdateMessage(ctx context.Context, d da.Device) ([][]byte, error) {
	exportedDevice := w.deviceExporter.ExportSimpleDevice(ctx, d)

	data, err := json.Marshal(DeviceUpdateMessage{
		DeviceMessage: DeviceMessage{
			Message: Message{
				Type: DeviceUpdateMessageName,
			},
		},
		ExportedSimpleDevice: exportedDevice,
	})

	return [][]byte{data}, err
}

func (w websocketEventMapper) generateDeviceUpdateCapabilityMessage(ctx context.Context, daDevice da.Device, capFlag da.Capability) ([][]byte, error) {
	uncastCapability := daDevice.Capability(capFlag)

	basic, ok := uncastCapability.(da.BasicCapability)
	if !ok {
		return nil, nil
	}

	out := w.deviceExporter.ExportCapability(ctx, uncastCapability)

	data, err := json.Marshal(DeviceUpdateCapabilityMessage{
		DeviceMessage: DeviceMessage{
			Message: Message{
				Type: DeviceUpdateCapabilityMessageName,
			},
		},
		Identifier: daDevice.Identifier().String(),
		Capability: basic.Name(),
		Payload:    out,
	})

	return [][]byte{data}, err
}

func (w websocketEventMapper) generateZoneUpdateMessage(zone state.Zone, after int) ([]byte, error) {
	return json.Marshal(ZoneUpdateMessage{
		ZoneMessage: ZoneMessage{
			Message: Message{
				Type: ZoneUpdateMessageName,
			},
			Identifier: zone.Identifier,
		},
		Name:   zone.Name,
		Parent: zone.ParentZone,
		After:  after,
	})
}

func (w websocketEventMapper) InitialEvents(ctx context.Context) ([][]byte, error) {
	var events [][]byte

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

func (w websocketEventMapper) generateDeviceMessages(ctx context.Context, device da.Device) [][]byte {
	var events [][]byte

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

func (w websocketEventMapper) initialEventsZone(zone state.Zone, after int) [][]byte {
	var events [][]byte

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

const (
	ZoneUpdateMessageName = "ZoneUpdate"
	ZoneRemoveMessageName = "ZoneRemove"

	GatewayUpdateMessageName = "GatewayUpdate"

	DeviceUpdateMessageName           = "DeviceUpdate"
	DeviceUpdateCapabilityMessageName = "DeviceUpdateCapability"
	DeviceRemoveMessageName           = "DeviceRemove"
)

type Message struct {
	Type string
}

type ZoneMessage struct {
	Message
	Identifier int
}

type ZoneUpdateMessage struct {
	ZoneMessage
	Name   string
	Parent int
	After  int
}

type ZoneRemoveMessage struct {
	ZoneMessage
}

type GatewayMessage struct {
	Message
}

type GatewayUpdateMessage struct {
	GatewayMessage
	exporter.ExportedGateway
}

type DeviceMessage struct {
	Message
}

type DeviceUpdateMessage struct {
	DeviceMessage
	exporter.ExportedSimpleDevice
}

type DeviceUpdateCapabilityMessage struct {
	DeviceMessage
	Identifier string
	Capability string
	Payload    interface{}
}

type DeviceRemoveMessage struct {
	DeviceMessage
	Identifier string
}
