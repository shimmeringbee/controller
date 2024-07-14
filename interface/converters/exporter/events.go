package exporter

import (
	"encoding/json"
	"github.com/shimmeringbee/da/capabilities"
	"time"
)

const (
	HeartBeatMessageName = "HeartBeat"

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

func (m Message) MessageType() string {
	return m.Type
}

type Typer interface {
	MessageType() string
}

type HeartBeatMessage struct {
	Message
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
	ExportedGateway
}

type DeviceMessage struct {
	Message
}

type DeviceUpdateMessage struct {
	DeviceMessage
	ExportedSimpleDevice
}

type DeviceUpdateCapabilityMessage struct {
	DeviceMessage
	Identifier string
	Capability string
	Payload    any
}

type DeviceRemoveMessage struct {
	DeviceMessage
	Identifier string
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

type TemperatureSensor struct {
	Readings []capabilities.TemperatureReading
	LastUpdate
	LastChange
}

type RelativeHumiditySensor struct {
	Readings []capabilities.RelativeHumidityReading
	LastUpdate
	LastChange
}

type PressureSensor struct {
	Readings []capabilities.PressureReading
	LastUpdate
	LastChange
}

type DeviceDiscovery struct {
	Discovering bool
	Duration    int `json:",omitempty"`
	LastUpdate
	LastChange
}

type AlarmSensor struct {
	Alarms map[string]bool
	LastUpdate
	LastChange
}

type OnOff struct {
	State bool
	LastUpdate
	LastChange
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

type AlarmWarningDeviceStatus struct {
	Warning   bool
	AlarmType *string  `json:",omitempty"`
	Volume    *float64 `json:",omitempty"`
	Visual    *bool    `json:",omitempty"`
	Duration  *int     `json:",omitempty"`
	LastUpdate
	LastChange
}

type IdentifyStatus struct {
	Identifying bool
	Duration    *int `json:",omitempty"`
	LastUpdate
	LastChange
}

type DeviceWorkaroundsStatus struct {
	Enabled []string
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

type ProductInformation struct {
	Name         string `json:",omitempty"`
	Manufacturer string `json:",omitempty"`
	Serial       string `json:",omitempty"`
	Version      string `json:",omitempty"`
}
