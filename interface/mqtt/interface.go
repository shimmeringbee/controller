package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shimmeringbee/controller/gateway"
	"github.com/shimmeringbee/controller/interface/exporter"
	"github.com/shimmeringbee/controller/layers"
	"github.com/shimmeringbee/controller/metadata"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"time"
)

type Publisher func(ctx context.Context, topic string, payload []byte) error

type Interface struct {
	publisher Publisher
	stop      chan bool

	DeviceOrganiser   *metadata.DeviceOrganiser
	GatewayMux        gateway.Mapper
	GatewaySubscriber gateway.Subscriber
	OutputStack       layers.OutputStack

	deviceExporter exporter.DeviceExporter
	Logger         logwrap.Logger

	PublishStateOnConnect bool
	PublishSummaryState   bool
}

func (i *Interface) IncomingMessage(ctx context.Context, topic string, payload []byte) error {
	return nil
}

func (i *Interface) Connected(ctx context.Context, publisher Publisher) error {
	i.publisher = publisher

	if i.PublishStateOnConnect {
		i.Logger.LogInfo(ctx, "MQTT connected, publishing current state of all devices and capabilities.")
		go i.publishAll()
	}

	return nil
}

func (i *Interface) publishAll() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, gw := range i.GatewayMux.Gateways() {
		for _, d := range gw.Devices() {
			i.publishDevice(ctx, d)
		}
	}
}

func (i *Interface) publishDevice(ctx context.Context, device da.Device) {
	deviceCtx := i.Logger.AddOptionsToContext(ctx, logwrap.Datum("device", device.Identifier().String()))

	for _, capability := range device.Capabilities() {
		i.publishDeviceCapability(deviceCtx, device, capability)
	}
}

func (i *Interface) publishDeviceCapability(ctx context.Context, daDevice da.Device, capFlag da.Capability) {
	capability := daDevice.Gateway().Capability(capFlag)

	basicCapability, ok := capability.(da.BasicCapability)
	if !ok {
		return
	}

	capName := basicCapability.Name()
	result := i.deviceExporter.ExportCapability(ctx, daDevice, capability)

	topic := fmt.Sprintf("devices/%s/capabilities/%s", daDevice.Identifier().String(), capName)

	if i.PublishSummaryState {
		if err := i.publishDeviceCapabilitySummary(ctx, topic, result); err != nil {
			i.Logger.LogError(ctx, "Failed to public summary state of capability.", logwrap.Datum("capability", capName), logwrap.Err(err))
		}
	}
}

func (i *Interface) publishDeviceCapabilitySummary(ctx context.Context, topic string, result interface{}) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err = i.publisher(ctx, topic, payload); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	return nil
}

func (i *Interface) Disconnected() {
	i.publisher = nil
}

func (i *Interface) Start() {
	i.stop = make(chan bool, 1)

	ch := make(chan interface{}, 100)
	i.GatewaySubscriber.Listen(ch)

	go i.handleEvents(ch)
}

func (i *Interface) Stop() {
	if i.stop != nil {
		i.stop <- true
	}
}

func (i *Interface) handleEvents(ch chan interface{}) {
	for {
		select {
		case event := <-ch:
			i.serviceUpdateOnEvent(event)
		case <-i.stop:
			return
		}
	}
}

const MaximumServiceUpdateTime = 1 * time.Second

func (i *Interface) serviceUpdateOnEvent(e interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), MaximumServiceUpdateTime)
	defer cancel()

	switch event := e.(type) {
	case capabilities.AlarmSensorUpdate:
		i.publishDeviceCapability(ctx, event.Device, capabilities.AlarmSensorFlag)
	case capabilities.AlarmWarningDeviceUpdate:
		i.publishDeviceCapability(ctx, event.Device, capabilities.AlarmWarningDeviceFlag)
	case capabilities.ColorStatusUpdate:
		i.publishDeviceCapability(ctx, event.Device, capabilities.ColorFlag)
	case capabilities.DeviceDiscoveryEnabled:
		i.publishDeviceCapability(ctx, event.Gateway.Self(), capabilities.DeviceDiscoveryFlag)
	case capabilities.DeviceDiscoveryDisabled:
		i.publishDeviceCapability(ctx, event.Gateway.Self(), capabilities.DeviceDiscoveryFlag)
	case capabilities.EnumerateDeviceStart:
		i.publishDeviceCapability(ctx, event.Device, capabilities.EnumerateDeviceFlag)
	case capabilities.EnumerateDeviceFailure:
		i.publishDeviceCapability(ctx, event.Device, capabilities.EnumerateDeviceFlag)
	case capabilities.EnumerateDeviceSuccess:
		i.publishDeviceCapability(ctx, event.Device, capabilities.EnumerateDeviceFlag)
	case capabilities.LevelStatusUpdate:
		i.publishDeviceCapability(ctx, event.Device, capabilities.LevelFlag)
	case capabilities.OnOffState:
		i.publishDeviceCapability(ctx, event.Device, capabilities.OnOffFlag)
	case capabilities.PowerStatusUpdate:
		i.publishDeviceCapability(ctx, event.Device, capabilities.PowerSupplyFlag)
	case capabilities.PressureSensorState:
		i.publishDeviceCapability(ctx, event.Device, capabilities.PressureSensorFlag)
	case capabilities.RelativeHumiditySensorState:
		i.publishDeviceCapability(ctx, event.Device, capabilities.RelativeHumiditySensorFlag)
	case capabilities.TemperatureSensorState:
		i.publishDeviceCapability(ctx, event.Device, capabilities.TemperatureSensorFlag)
	}
}
