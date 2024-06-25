package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shimmeringbee/controller/interface/converters/exporter"
	"github.com/shimmeringbee/controller/interface/converters/invoker"
	"github.com/shimmeringbee/controller/layers"
	"github.com/shimmeringbee/controller/state"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"strings"
	"time"
)

type Publisher func(ctx context.Context, topic string, payload []byte) error

type mqttError string

func (m mqttError) Error() string {
	return string(m)
}

const DefaultMqttOutputLayer string = "mqtt"

const UnknownTopic = mqttError("unknown topic")
const UnknownDevice = mqttError("unknown device")
const UnknownOutputLayer = mqttError("output layer requested could not be found")

type Interface struct {
	Publisher Publisher
	stop      chan bool

	DeviceOrganiser *state.DeviceOrganiser
	GatewayMux      state.GatewayMapper
	EventSubscriber state.EventSubscriber
	OutputStack     layers.OutputStack
	DeviceInvoker   invoker.Invoker

	deviceExporter exporter.DeviceExporter
	Logger         logwrap.Logger

	PublishStateOnConnect  bool
	PublishAggregatedState bool
	PublishIndividualState bool
}

func (i *Interface) IncomingMessage(ctx context.Context, topic string, payload []byte) error {
	topicParts := strings.Split(topic, "/")

	if len(topicParts) > 0 {
		switch topicParts[0] {
		case "devices":
			return i.IncomingMessageDevices(ctx, topicParts[1:], payload)
		}
	}

	return fmt.Errorf("%w: %s", UnknownTopic, topic)
}

func (i *Interface) IncomingMessageDevices(ctx context.Context, topic []string, payload []byte) error {
	if len(topic) > 0 {
		d, ok := i.GatewayMux.Device(topic[0])

		if ok {
			return i.IncomingMessageDevicesWith(ctx, topic[1:], payload, d)
		}
	}

	return fmt.Errorf("%w: %s", UnknownDevice, topic)
}

func (i *Interface) IncomingMessageDevicesWith(ctx context.Context, topic []string, payload []byte, d da.Device) error {
	if len(topic) > 0 {
		switch topic[0] {
		case "capabilities":
			return i.IncomingMessageDevicesWithCapabilities(ctx, topic[1:], payload, d)
		}
	}

	return fmt.Errorf("%w: %s", UnknownTopic, topic)
}

func (i *Interface) IncomingMessageDevicesWithCapabilities(ctx context.Context, topic []string, payload []byte, d da.Device) error {
	if len(topic) >= 3 && topic[2] == "invoke" {
		if _, err := i.DeviceInvoker(ctx, i.OutputStack, DefaultMqttOutputLayer, layers.OneShot, d, topic[0], topic[1], payload); err != nil {
			return fmt.Errorf("unable to invoke action on device: %w", err)
		}

		return nil
	}

	return fmt.Errorf("%w: %s", UnknownTopic, topic)
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
	capability := daDevice.Capability(capFlag)

	basicCapability, ok := capability.(da.BasicCapability)
	if !ok {
		return
	}

	capName := basicCapability.Name()
	result := i.deviceExporter.ExportCapability(ctx, capability)

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

func (i *Interface) publishDeviceCapabilityAggregated(ctx context.Context, topic string, result any) error {
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

	ch := make(chan any, 100)
	i.EventSubscriber.Subscribe(ch)

	go i.handleEvents(ch)
}

func (i *Interface) Stop() {
	if i.stop != nil {
		i.stop <- true
	}
}

func (i *Interface) handleEvents(ch chan any) {
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

func (i *Interface) serviceUpdateOnEvent(e any) {
	ctx, cancel := context.WithTimeout(context.Background(), MaximumServiceUpdateTime)
	defer cancel()

	switch event := e.(type) {
	case da.DeviceAdded:
		i.publishDevice(ctx, event.Device)
	case capabilities.AlarmSensorUpdate:
		i.publishDeviceCapability(ctx, event.Device, capabilities.AlarmSensorFlag)
	case capabilities.AlarmWarningDeviceUpdate:
		i.publishDeviceCapability(ctx, event.Device, capabilities.AlarmWarningDeviceFlag)
	case capabilities.DeviceDiscoveryEnabled:
		i.publishDeviceCapability(ctx, event.Gateway.Self(), capabilities.DeviceDiscoveryFlag)
	case capabilities.DeviceDiscoveryDisabled:
		i.publishDeviceCapability(ctx, event.Gateway.Self(), capabilities.DeviceDiscoveryFlag)
	case capabilities.EnumerateDeviceStart:
		i.publishDeviceCapability(ctx, event.Device, capabilities.EnumerateDeviceFlag)
	case capabilities.EnumerateDeviceStopped:
		i.publishDevice(ctx, event.Device)
	case capabilities.OnOffUpdate:
		i.publishDeviceCapability(ctx, event.Device, capabilities.OnOffFlag)
	case capabilities.PowerStatusUpdate:
		i.publishDeviceCapability(ctx, event.Device, capabilities.PowerSupplyFlag)
	case capabilities.PressureSensorUpdate:
		i.publishDeviceCapability(ctx, event.Device, capabilities.PressureSensorFlag)
	case capabilities.RelativeHumiditySensorUpdate:
		i.publishDeviceCapability(ctx, event.Device, capabilities.RelativeHumiditySensorFlag)
	case capabilities.TemperatureSensorUpdate:
		i.publishDeviceCapability(ctx, event.Device, capabilities.TemperatureSensorFlag)
	}
}

func (i *Interface) publishDeviceCapabilityIndividual(ctx context.Context, topic string, result any) error {
	switch c := result.(type) {
	case *exporter.AlarmSensor:
		return i.publishDeviceCapabilityIndividualAlarmSensor(ctx, topic, c)
	case *exporter.AlarmWarningDeviceStatus:
		return i.publishDeviceCapabilityIndividualAlarmWarningDevice(ctx, topic, c)
	case *exporter.DeviceDiscovery:
		return i.publishDeviceCapabilityIndividualDeviceDiscovery(ctx, topic, c)
	case *exporter.EnumerateDevice:
		return i.publishDeviceCapabilityIndividualEnumerateDevice(ctx, topic, c)
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
	case *exporter.ProductInformation:
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

	for capName, status := range c.Status {
		if err := i.Publisher(ctx, fmt.Sprintf("%s/Status/%s/Attached", topic, capName), []byte(fmt.Sprintf("%v", status.Attached))); err != nil {
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

func (i *Interface) publishDeviceCapabilityIndividualHasProductInformation(ctx context.Context, topic string, c *exporter.ProductInformation) error {
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

func fmtPtrToJSON(v any) []byte {
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
