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
	Publisher Publisher
	stop      chan bool

	DeviceOrganiser   *metadata.DeviceOrganiser
	GatewayMux        gateway.Mapper
	GatewaySubscriber gateway.Subscriber
	OutputStack       layers.OutputStack

	deviceExporter exporter.DeviceExporter
	Logger         logwrap.Logger

	PublishStateOnConnect  bool
	PublishAggregatedState bool
	PublishIndividualState bool
}

func (i *Interface) IncomingMessage(ctx context.Context, topic string, payload []byte) error {
	return nil
}

func EmptyPublisher(ctx context.Context, topic string, payload []byte) error {
	return nil
}

func (i *Interface) Connected(ctx context.Context, publisher Publisher) error {
	i.Publisher = publisher

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

	if i.PublishAggregatedState {
		if err := i.publishDeviceCapabilityAggregated(ctx, topic, result); err != nil {
			i.Logger.LogError(ctx, "Failed to public Aggregated state of capability.", logwrap.Datum("capability", capName), logwrap.Err(err))
		}
	}

	if i.PublishIndividualState {
		if err := i.publishDeviceCapabilityIndividual(ctx, topic, result); err != nil {
			i.Logger.LogError(ctx, "Failed to public Individual state of capability.", logwrap.Datum("capability", capName), logwrap.Err(err))
		}
	}
}

func (i *Interface) publishDeviceCapabilityAggregated(ctx context.Context, topic string, result interface{}) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err = i.Publisher(ctx, topic, payload); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	return nil
}

func (i *Interface) Disconnected() {
	i.Publisher = EmptyPublisher
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
	case da.DeviceAdded:
		i.publishDevice(ctx, event.Device)
	case da.DeviceLoaded:
		i.publishDevice(ctx, event.Device)
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
		i.publishDevice(ctx, event.Device)
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

func (i *Interface) publishDeviceCapabilityIndividual(ctx context.Context, topic string, result interface{}) error {
	switch c := result.(type) {
	case *exporter.AlarmSensor:
		return i.publishDeviceCapabilityIndividualAlarmSensor(ctx, topic, c)
	case *exporter.AlarmWarningDeviceStatus:
		return i.publishDeviceCapabilityIndividualAlarmWarningDevice(ctx, topic, c)
	case *exporter.Color:
		return i.publishDeviceCapabilityIndividualColor(ctx, topic, c)
	case *exporter.DeviceDiscovery:
		return i.publishDeviceCapabilityIndividualDeviceDiscovery(ctx, topic, c)
	case *exporter.EnumerateDevice:
		return i.publishDeviceCapabilityIndividualEnumerateDevice(ctx, topic, c)
	case *exporter.Level:
		return i.publishDeviceCapabilityIndividualLevel(ctx, topic, c)
	case *exporter.OnOff:
		return i.publishDeviceCapabilityIndividualOnOff(ctx, topic, c)
	case *exporter.PowerStatus:
		return i.publishDeviceCapabilityIndividualPower(ctx, topic, c)
	case *exporter.PressureSensor:
		return i.publishDeviceCapabilityIndividualPressureSensor(ctx, topic, c)
	case *exporter.RelativeHumiditySensor:
		return i.publishDeviceCapabilityIndividualRelativeHumiditySensor(ctx, topic, c)
	case *exporter.TemperatureSensor:
		return i.publishDeviceCapabilityIndividualTemperatureSensor(ctx, topic, c)
	case *exporter.HasProductInformation:
		return i.publishDeviceCapabilityIndividualHasProductInformation(ctx, topic, c)
	}

	return nil
}

func (i *Interface) publishDeviceCapabilityIndividualAlarmSensor(ctx context.Context, topic string, c *exporter.AlarmSensor) error {
	for alarm, state := range c.Alarms {
		if err := i.Publisher(ctx, fmt.Sprintf("%s/Alarms/%s", topic, alarm), []byte(fmt.Sprintf("%v", state))); err != nil {
			return fmt.Errorf("failed to publish data to mqtt: %w", err)
		}
	}

	return nil
}

func (i *Interface) publishDeviceCapabilityIndividualAlarmWarningDevice(ctx context.Context, topic string, c *exporter.AlarmWarningDeviceStatus) error {
	if err := i.Publisher(ctx, fmt.Sprintf("%s/Warning", topic), []byte(fmt.Sprintf("%v", c.Warning))); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	if err := i.Publisher(ctx, fmt.Sprintf("%s/AlarmType", topic), fmtPtrString(c.AlarmType)); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	if err := i.Publisher(ctx, fmt.Sprintf("%s/Volume", topic), fmtPtrFloat64(c.Volume)); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	if err := i.Publisher(ctx, fmt.Sprintf("%s/Visual", topic), fmtPtrBool(c.Visual)); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	if err := i.Publisher(ctx, fmt.Sprintf("%s/Duration", topic), fmtPtrInt(c.Duration)); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	return nil
}

func (i *Interface) publishDeviceCapabilityIndividualColor(ctx context.Context, topic string, c *exporter.Color) error {
	if err := i.Publisher(ctx, fmt.Sprintf("%s/Current", topic), fmtPtrToJSON(c.Current)); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	if err := i.Publisher(ctx, fmt.Sprintf("%s/Target", topic), fmtPtrToJSON(c.Target)); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	if err := i.Publisher(ctx, fmt.Sprintf("%s/Duration", topic), []byte(fmt.Sprintf("%v", c.DurationRemaining))); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	if err := i.Publisher(ctx, fmt.Sprintf("%s/Supports/Color", topic), []byte(fmt.Sprintf("%v", c.Supports.Color))); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	if err := i.Publisher(ctx, fmt.Sprintf("%s/Supports/Temperature", topic), []byte(fmt.Sprintf("%v", c.Supports.Temperature))); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	return nil
}

func (i *Interface) publishDeviceCapabilityIndividualDeviceDiscovery(ctx context.Context, topic string, c *exporter.DeviceDiscovery) error {
	if err := i.Publisher(ctx, fmt.Sprintf("%s/Discovering", topic), []byte(fmt.Sprintf("%v", c.Discovering))); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	if err := i.Publisher(ctx, fmt.Sprintf("%s/Duration", topic), []byte(fmt.Sprintf("%d", c.Duration))); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	return nil
}

func (i *Interface) publishDeviceCapabilityIndividualEnumerateDevice(ctx context.Context, topic string, c *exporter.EnumerateDevice) error {
	if err := i.Publisher(ctx, fmt.Sprintf("%s/Enumerating", topic), []byte(fmt.Sprintf("%v", c.Enumerating))); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	return nil
}

func (i *Interface) publishDeviceCapabilityIndividualLevel(ctx context.Context, topic string, c *exporter.Level) error {
	if err := i.Publisher(ctx, fmt.Sprintf("%s/Current", topic), []byte(fmt.Sprintf("%f", c.Current))); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	if c.Target > 0 {
		if err := i.Publisher(ctx, fmt.Sprintf("%s/Target", topic), []byte(fmt.Sprintf("%f", c.Target))); err != nil {
			return fmt.Errorf("failed to publish data to mqtt: %w", err)
		}
	} else {
		if err := i.Publisher(ctx, fmt.Sprintf("%s/Target", topic), []byte(`null`)); err != nil {
			return fmt.Errorf("failed to publish data to mqtt: %w", err)
		}
	}

	return nil
}

func (i *Interface) publishDeviceCapabilityIndividualOnOff(ctx context.Context, topic string, c *exporter.OnOff) error {
	if err := i.Publisher(ctx, fmt.Sprintf("%s/Current", topic), []byte(fmt.Sprintf("%v", c.State))); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	return nil
}

func (i *Interface) publishDeviceCapabilityIndividualPower(ctx context.Context, topic string, c *exporter.PowerStatus) error {
	for j, mains := range c.Mains {
		if err := i.Publisher(ctx, fmt.Sprintf("%s/Mains/%d/Voltage", topic, j), fmtPtrFloat64(mains.Voltage)); err != nil {
			return fmt.Errorf("failed to publish data to mqtt: %w", err)
		}

		if err := i.Publisher(ctx, fmt.Sprintf("%s/Mains/%d/Frequency", topic, j), fmtPtrFloat64(mains.Frequency)); err != nil {
			return fmt.Errorf("failed to publish data to mqtt: %w", err)
		}

		if err := i.Publisher(ctx, fmt.Sprintf("%s/Mains/%d/Available", topic, j), fmtPtrBool(mains.Available)); err != nil {
			return fmt.Errorf("failed to publish data to mqtt: %w", err)
		}
	}

	for j, battery := range c.Battery {
		if err := i.Publisher(ctx, fmt.Sprintf("%s/Battery/%d/Voltage", topic, j), fmtPtrFloat64(battery.Voltage)); err != nil {
			return fmt.Errorf("failed to publish data to mqtt: %w", err)
		}

		if err := i.Publisher(ctx, fmt.Sprintf("%s/Battery/%d/MinimumVoltage", topic, j), fmtPtrFloat64(battery.MinimumVoltage)); err != nil {
			return fmt.Errorf("failed to publish data to mqtt: %w", err)
		}

		if err := i.Publisher(ctx, fmt.Sprintf("%s/Battery/%d/MaximumVoltage", topic, j), fmtPtrFloat64(battery.MaximumVoltage)); err != nil {
			return fmt.Errorf("failed to publish data to mqtt: %w", err)
		}

		if err := i.Publisher(ctx, fmt.Sprintf("%s/Battery/%d/Remaining", topic, j), fmtPtrFloat64(battery.Remaining)); err != nil {
			return fmt.Errorf("failed to publish data to mqtt: %w", err)
		}

		if err := i.Publisher(ctx, fmt.Sprintf("%s/Battery/%d/Available", topic, j), fmtPtrBool(battery.Available)); err != nil {
			return fmt.Errorf("failed to publish data to mqtt: %w", err)
		}
	}

	return nil
}

func (i *Interface) publishDeviceCapabilityIndividualPressureSensor(ctx context.Context, topic string, c *exporter.PressureSensor) error {
	for j, reading := range c.Readings {
		if err := i.Publisher(ctx, fmt.Sprintf("%s/Reading/%d/Value", topic, j), []byte(fmt.Sprintf("%f", reading.Value))); err != nil {
			return fmt.Errorf("failed to publish data to mqtt: %w", err)
		}
	}

	return nil
}

func (i *Interface) publishDeviceCapabilityIndividualRelativeHumiditySensor(ctx context.Context, topic string, c *exporter.RelativeHumiditySensor) error {
	for j, reading := range c.Readings {
		if err := i.Publisher(ctx, fmt.Sprintf("%s/Reading/%d/Value", topic, j), []byte(fmt.Sprintf("%f", reading.Value))); err != nil {
			return fmt.Errorf("failed to publish data to mqtt: %w", err)
		}
	}

	return nil
}

func (i *Interface) publishDeviceCapabilityIndividualTemperatureSensor(ctx context.Context, topic string, c *exporter.TemperatureSensor) error {
	for j, reading := range c.Readings {
		if err := i.Publisher(ctx, fmt.Sprintf("%s/Reading/%d/Value", topic, j), []byte(fmt.Sprintf("%f", reading.Value))); err != nil {
			return fmt.Errorf("failed to publish data to mqtt: %w", err)
		}
	}

	return nil
}

func (i *Interface) publishDeviceCapabilityIndividualHasProductInformation(ctx context.Context, topic string, c *exporter.HasProductInformation) error {
	if err := i.Publisher(ctx, fmt.Sprintf("%s/Product", topic), fmtString(c.Name)); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	if err := i.Publisher(ctx, fmt.Sprintf("%s/Manufacturer", topic), fmtString(c.Manufacturer)); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	if err := i.Publisher(ctx, fmt.Sprintf("%s/Serial", topic), fmtString(c.Serial)); err != nil {
		return fmt.Errorf("failed to publish data to mqtt: %w", err)
	}

	return nil
}

func fmtPtrToJSON(v interface{}) []byte {
	if v == nil {
		return []byte("null")
	}

	data, err := json.Marshal(v)
	if err != nil {
		return []byte("null")
	}

	return data
}

func fmtPtrString(s *string) []byte {
	if s == nil {
		return []byte("null")
	}

	return []byte(*s)
}

func fmtString(s string) []byte {
	if len(s) == 0 {
		return []byte("null")
	}

	return []byte(s)
}

func fmtPtrFloat64(s *float64) []byte {
	if s == nil {
		return []byte("null")
	}

	return []byte(fmt.Sprintf("%f", *s))
}

func fmtPtrInt(s *int) []byte {
	if s == nil {
		return []byte("null")
	}

	return []byte(fmt.Sprintf("%d", *s))
}

func fmtPtrBool(s *bool) []byte {
	if s == nil {
		return []byte("null")
	}

	return []byte(fmt.Sprintf("%v", *s))
}
