package mqtt

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/controller/gateway"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	capmocks "github.com/shimmeringbee/da/capabilities/mocks"
	"github.com/shimmeringbee/da/mocks"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestInterface_Connected(t *testing.T) {
	t.Run("publisher is set correctly", func(t *testing.T) {
		i := Interface{}

		m := &MockPublisher{}
		defer m.AssertExpectations(t)

		err := i.Connected(context.Background(), m.Publish)
		assert.NoError(t, err)

		assert.NotNil(t, i.publisher)
	})

	t.Run("publishes capabilities if set to publish on connect", func(t *testing.T) {
		mapper := &gateway.MockMapper{}
		defer mapper.AssertExpectations(t)

		gw := &mocks.Gateway{}
		defer gw.AssertExpectations(t)

		mapper.On("Gateways").Return(map[string]da.Gateway{"one": gw})

		capFlagOne := capabilities.HasProductInformationFlag
		capFlagTwo := capabilities.OnOffFlag

		d := da.BaseDevice{
			DeviceGateway:      gw,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{capFlagOne, capFlagTwo},
		}

		gw.On("Devices").Return([]da.Device{d})

		hpi := &capmocks.HasProductInformation{}
		hpi.On("Name").Return("HasProductInformation")
		hpi.On("ProductInformation", mock.Anything, d).Return(capabilities.ProductInformation{
			Present: capabilities.Name,
			Name:    "Mock",
		}, nil)
		defer hpi.AssertExpectations(t)

		oo := &capmocks.OnOff{}
		oo.Mock.On("Name").Return("OnOff")
		oo.Mock.On("Status", mock.Anything, d).Return(true, nil)
		defer oo.AssertExpectations(t)

		gw.On("Capability", capFlagOne).Return(hpi)
		gw.On("Capability", capFlagTwo).Return(oo)

		i := Interface{GatewayMux: mapper, Logger: logwrap.New(discard.Discard()), PublishStateOnConnect: true, PublishSummaryState: true}

		m := &MockPublisher{}
		defer m.AssertExpectations(t)

		m.On("Publish", mock.Anything, fmt.Sprintf("devices/%s/capabilities/HasProductInformation", d.DeviceIdentifier.String()), []byte{0x7b, 0x22, 0x4e, 0x61, 0x6d, 0x65, 0x22, 0x3a, 0x22, 0x4d, 0x6f, 0x63, 0x6b, 0x22, 0x7d}).Return(nil)
		m.On("Publish", mock.Anything, fmt.Sprintf("devices/%s/capabilities/OnOff", d.DeviceIdentifier.String()), []byte{0x7b, 0x22, 0x53, 0x74, 0x61, 0x74, 0x65, 0x22, 0x3a, 0x74, 0x72, 0x75, 0x65, 0x7d}).Return(nil)

		err := i.Connected(context.Background(), m.Publish)
		assert.NoError(t, err)

		time.Sleep(50 * time.Millisecond)
	})
}

func TestInterface_serviceUpdateOnEvent(t *testing.T) {
	t.Run("AlarmSensorUpdate publishes a summary update if enabled", func(t *testing.T) {
		mapper := &gateway.MockMapper{}
		defer mapper.AssertExpectations(t)

		m := &MockPublisher{}
		defer m.AssertExpectations(t)

		gw := &mocks.Gateway{}
		defer gw.AssertExpectations(t)

		d := da.BaseDevice{
			DeviceGateway:      gw,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{capabilities.AlarmSensorFlag},
		}

		name := "AlarmSensor"
		mc := &capmocks.AlarmSensor{}
		mc.On("Name").Return(name)
		mc.On("Status", mock.Anything, d).Return(map[capabilities.SensorType]bool{capabilities.General: true}, nil)
		defer mc.AssertExpectations(t)

		gw.On("Capability", capabilities.AlarmSensorFlag).Return(mc)

		i := Interface{GatewayMux: mapper, Logger: logwrap.New(discard.Discard()), PublishSummaryState: true, publisher: m.Publish}

		expectedPayload := `{"Alarms":{"General":true}}`
		m.On("Publish", mock.Anything, fmt.Sprintf("devices/%s/capabilities/%s", d.DeviceIdentifier.String(), name), []byte(expectedPayload)).Return(nil)

		i.serviceUpdateOnEvent(capabilities.AlarmSensorUpdate{
			Device: d,
		})

		time.Sleep(50 * time.Millisecond)
	})

	t.Run("AlarmWarningDevice publishes a summary update if enabled", func(t *testing.T) {
		mapper := &gateway.MockMapper{}
		defer mapper.AssertExpectations(t)

		m := &MockPublisher{}
		defer m.AssertExpectations(t)

		gw := &mocks.Gateway{}
		defer gw.AssertExpectations(t)

		d := da.BaseDevice{
			DeviceGateway:      gw,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{capabilities.AlarmWarningDeviceFlag},
		}

		name := "AlarmWarningDevice"
		mc := &capmocks.AlarmWarningDevice{}
		mc.On("Name").Return(name)
		mc.On("Status", mock.Anything, d).Return(capabilities.WarningDeviceState{
			Warning:           true,
			AlarmType:         capabilities.FireAlarm,
			Volume:            .5,
			Visual:            true,
			DurationRemaining: time.Minute,
		}, nil)
		defer mc.AssertExpectations(t)

		gw.On("Capability", capabilities.AlarmWarningDeviceFlag).Return(mc)

		i := Interface{GatewayMux: mapper, Logger: logwrap.New(discard.Discard()), PublishSummaryState: true, publisher: m.Publish}

		expectedPayload := `{"Warning":true,"AlarmType":"Fire","Volume":0.5,"Visual":true,"Duration":60000}`
		m.On("Publish", mock.Anything, fmt.Sprintf("devices/%s/capabilities/%s", d.DeviceIdentifier.String(), name), []byte(expectedPayload)).Return(nil)

		i.serviceUpdateOnEvent(capabilities.AlarmWarningDeviceUpdate{
			Device: d,
		})

		time.Sleep(50 * time.Millisecond)
	})

	t.Run("Color publishes a summary update if enabled", func(t *testing.T) {
		mapper := &gateway.MockMapper{}
		defer mapper.AssertExpectations(t)

		m := &MockPublisher{}
		defer m.AssertExpectations(t)

		gw := &mocks.Gateway{}
		defer gw.AssertExpectations(t)

		d := da.BaseDevice{
			DeviceGateway:      gw,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{capabilities.ColorFlag},
		}

		name := "Color"
		mc := &capmocks.Color{}
		mc.On("Name").Return(name)
		mc.On("Status", mock.Anything, d).Return(capabilities.ColorStatus{
			Mode: capabilities.TemperatureMode,
			Temperature: capabilities.TemperatureSettings{
				Current: 6500,
			},
			DurationRemaining: time.Minute,
		}, nil)
		mc.On("SupportsColor", mock.Anything, d).Return(false, nil)
		mc.On("SupportsTemperature", mock.Anything, d).Return(true, nil)
		defer mc.AssertExpectations(t)

		gw.On("Capability", capabilities.ColorFlag).Return(mc)

		i := Interface{GatewayMux: mapper, Logger: logwrap.New(discard.Discard()), PublishSummaryState: true, publisher: m.Publish}

		expectedPayload := `{"Current":{"Temperature":6500},"DurationRemaining":60000,"Supports":{"Color":false,"Temperature":true}}`
		m.On("Publish", mock.Anything, fmt.Sprintf("devices/%s/capabilities/%s", d.DeviceIdentifier.String(), name), []byte(expectedPayload)).Return(nil)

		i.serviceUpdateOnEvent(capabilities.ColorStatusUpdate{
			Device: d,
		})

		time.Sleep(50 * time.Millisecond)
	})

	t.Run("DeviceDiscovery publishes a summary on Enable if enabled", func(t *testing.T) {
		mapper := &gateway.MockMapper{}
		defer mapper.AssertExpectations(t)

		m := &MockPublisher{}
		defer m.AssertExpectations(t)

		gw := &mocks.Gateway{}
		defer gw.AssertExpectations(t)

		d := da.BaseDevice{
			DeviceGateway:      gw,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{capabilities.DeviceDiscoveryFlag},
		}

		gw.On("Self").Return(d)

		name := "DeviceDiscovery"
		mc := &capmocks.DeviceDiscovery{}
		mc.On("Name").Return(name)
		mc.On("Status", mock.Anything, d).Return(capabilities.DeviceDiscoveryStatus{
			Discovering:       true,
			RemainingDuration: time.Minute,
		}, nil)
		defer mc.AssertExpectations(t)

		gw.On("Capability", capabilities.DeviceDiscoveryFlag).Return(mc)

		i := Interface{GatewayMux: mapper, Logger: logwrap.New(discard.Discard()), PublishSummaryState: true, publisher: m.Publish}

		expectedPayload := `{"Discovering":true,"Duration":60000}`
		m.On("Publish", mock.Anything, fmt.Sprintf("devices/%s/capabilities/%s", d.DeviceIdentifier.String(), name), []byte(expectedPayload)).Return(nil)

		i.serviceUpdateOnEvent(capabilities.DeviceDiscoveryEnabled{
			Gateway: gw,
		})

		time.Sleep(50 * time.Millisecond)
	})

	t.Run("DeviceDiscovery publishes a summary on Disable if enabled", func(t *testing.T) {
		mapper := &gateway.MockMapper{}
		defer mapper.AssertExpectations(t)

		m := &MockPublisher{}
		defer m.AssertExpectations(t)

		gw := &mocks.Gateway{}
		defer gw.AssertExpectations(t)

		d := da.BaseDevice{
			DeviceGateway:      gw,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{capabilities.DeviceDiscoveryFlag},
		}

		gw.On("Self").Return(d)

		name := "DeviceDiscovery"
		mc := &capmocks.DeviceDiscovery{}
		mc.On("Name").Return(name)
		mc.On("Status", mock.Anything, d).Return(capabilities.DeviceDiscoveryStatus{
			Discovering: false,
		}, nil)
		defer mc.AssertExpectations(t)

		gw.On("Capability", capabilities.DeviceDiscoveryFlag).Return(mc)

		i := Interface{GatewayMux: mapper, Logger: logwrap.New(discard.Discard()), PublishSummaryState: true, publisher: m.Publish}

		expectedPayload := `{"Discovering":false}`
		m.On("Publish", mock.Anything, fmt.Sprintf("devices/%s/capabilities/%s", d.DeviceIdentifier.String(), name), []byte(expectedPayload)).Return(nil)

		i.serviceUpdateOnEvent(capabilities.DeviceDiscoveryDisabled{
			Gateway: gw,
		})

		time.Sleep(50 * time.Millisecond)
	})

	t.Run("EnumerateDevice publishes a summary on Start if enabled", func(t *testing.T) {
		mapper := &gateway.MockMapper{}
		defer mapper.AssertExpectations(t)

		m := &MockPublisher{}
		defer m.AssertExpectations(t)

		gw := &mocks.Gateway{}
		defer gw.AssertExpectations(t)

		d := da.BaseDevice{
			DeviceGateway:      gw,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{capabilities.EnumerateDeviceFlag},
		}

		name := "EnumerateDevice"
		mc := &capmocks.EnumerateDevice{}
		mc.On("Name").Return(name)
		mc.On("Status", mock.Anything, d).Return(capabilities.EnumerationStatus{
			Enumerating: true,
		}, nil)
		defer mc.AssertExpectations(t)

		gw.On("Capability", capabilities.EnumerateDeviceFlag).Return(mc)

		i := Interface{GatewayMux: mapper, Logger: logwrap.New(discard.Discard()), PublishSummaryState: true, publisher: m.Publish}

		expectedPayload := `{"Enumerating":true}`
		m.On("Publish", mock.Anything, fmt.Sprintf("devices/%s/capabilities/%s", d.DeviceIdentifier.String(), name), []byte(expectedPayload)).Return(nil)

		i.serviceUpdateOnEvent(capabilities.EnumerateDeviceStart{
			Device: d,
		})

		time.Sleep(50 * time.Millisecond)
	})

	t.Run("EnumerateDevice publishes a summary on Success if enabled", func(t *testing.T) {
		mapper := &gateway.MockMapper{}
		defer mapper.AssertExpectations(t)

		m := &MockPublisher{}
		defer m.AssertExpectations(t)

		gw := &mocks.Gateway{}
		defer gw.AssertExpectations(t)

		d := da.BaseDevice{
			DeviceGateway:      gw,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{capabilities.EnumerateDeviceFlag},
		}

		name := "EnumerateDevice"
		mc := &capmocks.EnumerateDevice{}
		mc.On("Name").Return(name)
		mc.On("Status", mock.Anything, d).Return(capabilities.EnumerationStatus{
			Enumerating: false,
		}, nil)
		defer mc.AssertExpectations(t)

		gw.On("Capability", capabilities.EnumerateDeviceFlag).Return(mc)

		i := Interface{GatewayMux: mapper, Logger: logwrap.New(discard.Discard()), PublishSummaryState: true, publisher: m.Publish}

		expectedPayload := `{"Enumerating":false}`
		m.On("Publish", mock.Anything, fmt.Sprintf("devices/%s/capabilities/%s", d.DeviceIdentifier.String(), name), []byte(expectedPayload)).Return(nil)

		i.serviceUpdateOnEvent(capabilities.EnumerateDeviceSuccess{
			Device: d,
		})

		time.Sleep(50 * time.Millisecond)
	})

	t.Run("EnumerateDevice publishes a summary on Failure if enabled", func(t *testing.T) {
		mapper := &gateway.MockMapper{}
		defer mapper.AssertExpectations(t)

		m := &MockPublisher{}
		defer m.AssertExpectations(t)

		gw := &mocks.Gateway{}
		defer gw.AssertExpectations(t)

		d := da.BaseDevice{
			DeviceGateway:      gw,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{capabilities.EnumerateDeviceFlag},
		}

		name := "EnumerateDevice"
		mc := &capmocks.EnumerateDevice{}
		mc.On("Name").Return(name)
		mc.On("Status", mock.Anything, d).Return(capabilities.EnumerationStatus{
			Enumerating: false,
		}, nil)
		defer mc.AssertExpectations(t)

		gw.On("Capability", capabilities.EnumerateDeviceFlag).Return(mc)

		i := Interface{GatewayMux: mapper, Logger: logwrap.New(discard.Discard()), PublishSummaryState: true, publisher: m.Publish}

		expectedPayload := `{"Enumerating":false}`
		m.On("Publish", mock.Anything, fmt.Sprintf("devices/%s/capabilities/%s", d.DeviceIdentifier.String(), name), []byte(expectedPayload)).Return(nil)

		i.serviceUpdateOnEvent(capabilities.EnumerateDeviceFailure{
			Device: d,
		})

		time.Sleep(50 * time.Millisecond)
	})

	t.Run("Level publishes a summary on Update if enabled", func(t *testing.T) {
		mapper := &gateway.MockMapper{}
		defer mapper.AssertExpectations(t)

		m := &MockPublisher{}
		defer m.AssertExpectations(t)

		gw := &mocks.Gateway{}
		defer gw.AssertExpectations(t)

		d := da.BaseDevice{
			DeviceGateway:      gw,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{capabilities.LevelFlag},
		}

		name := "Level"
		mc := &capmocks.Level{}
		mc.On("Name").Return(name)
		mc.On("Status", mock.Anything, d).Return(capabilities.LevelStatus{
			CurrentLevel: 0.5,
		}, nil)
		defer mc.AssertExpectations(t)

		gw.On("Capability", capabilities.LevelFlag).Return(mc)

		i := Interface{GatewayMux: mapper, Logger: logwrap.New(discard.Discard()), PublishSummaryState: true, publisher: m.Publish}

		expectedPayload := `{"Current":0.5}`
		m.On("Publish", mock.Anything, fmt.Sprintf("devices/%s/capabilities/%s", d.DeviceIdentifier.String(), name), []byte(expectedPayload)).Return(nil)

		i.serviceUpdateOnEvent(capabilities.LevelStatusUpdate{
			Device: d,
		})

		time.Sleep(50 * time.Millisecond)
	})

	t.Run("OnOff publishes a summary on Update if enabled", func(t *testing.T) {
		mapper := &gateway.MockMapper{}
		defer mapper.AssertExpectations(t)

		m := &MockPublisher{}
		defer m.AssertExpectations(t)

		gw := &mocks.Gateway{}
		defer gw.AssertExpectations(t)

		d := da.BaseDevice{
			DeviceGateway:      gw,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{capabilities.OnOffFlag},
		}

		name := "OnOff"
		mc := &capmocks.OnOff{}
		mc.Mock.On("Name").Return(name)
		mc.Mock.On("Status", mock.Anything, d).Return(true, nil)
		defer mc.AssertExpectations(t)

		gw.On("Capability", capabilities.OnOffFlag).Return(mc)

		i := Interface{GatewayMux: mapper, Logger: logwrap.New(discard.Discard()), PublishSummaryState: true, publisher: m.Publish}

		expectedPayload := `{"State":true}`
		m.On("Publish", mock.Anything, fmt.Sprintf("devices/%s/capabilities/%s", d.DeviceIdentifier.String(), name), []byte(expectedPayload)).Return(nil)

		i.serviceUpdateOnEvent(capabilities.OnOffState{
			Device: d,
		})

		time.Sleep(50 * time.Millisecond)
	})

	t.Run("PowerSupply publishes a summary on Update if enabled", func(t *testing.T) {
		mapper := &gateway.MockMapper{}
		defer mapper.AssertExpectations(t)

		m := &MockPublisher{}
		defer m.AssertExpectations(t)

		gw := &mocks.Gateway{}
		defer gw.AssertExpectations(t)

		d := da.BaseDevice{
			DeviceGateway:      gw,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{capabilities.PowerSupplyFlag},
		}

		name := "PowerSupply"
		mc := &capmocks.PowerSupply{}
		mc.On("Name").Return(name)
		mc.On("Status", mock.Anything, d).Return(capabilities.PowerStatus{
			Mains: []capabilities.PowerMainsStatus{
				{
					Voltage:   220,
					Frequency: 50,
					Available: true,
					Present:   capabilities.Voltage | capabilities.Frequency | capabilities.Available,
				},
			},
			Battery: []capabilities.PowerBatteryStatus{
				{
					Voltage:        3.8,
					MaximumVoltage: 4.2,
					MinimumVoltage: 3.7,
					Remaining:      0.8,
					Available:      true,
					Present:        capabilities.Voltage | capabilities.MinimumVoltage | capabilities.MaximumVoltage | capabilities.Remaining | capabilities.Available,
				},
			},
		}, nil)
		defer mc.AssertExpectations(t)

		gw.On("Capability", capabilities.PowerSupplyFlag).Return(mc)

		i := Interface{GatewayMux: mapper, Logger: logwrap.New(discard.Discard()), PublishSummaryState: true, publisher: m.Publish}

		expectedPayload := `{"Mains":[{"Voltage":220,"Frequency":50,"Available":true}],"Battery":[{"Voltage":3.8,"MaximumVoltage":4.2,"MinimumVoltage":3.7,"Remaining":0.8,"Available":true}]}`
		m.On("Publish", mock.Anything, fmt.Sprintf("devices/%s/capabilities/%s", d.DeviceIdentifier.String(), name), []byte(expectedPayload)).Return(nil)

		i.serviceUpdateOnEvent(capabilities.PowerStatusUpdate{
			Device: d,
		})

		time.Sleep(50 * time.Millisecond)
	})

	t.Run("PressureSensor publishes a summary on Update if enabled", func(t *testing.T) {
		mapper := &gateway.MockMapper{}
		defer mapper.AssertExpectations(t)

		m := &MockPublisher{}
		defer m.AssertExpectations(t)

		gw := &mocks.Gateway{}
		defer gw.AssertExpectations(t)

		d := da.BaseDevice{
			DeviceGateway:      gw,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{capabilities.PressureSensorFlag},
		}

		name := "PressureSensor"
		mc := &capmocks.PressureSensor{}
		mc.On("Name").Return(name)
		mc.On("Reading", mock.Anything, d).Return([]capabilities.PressureReading{
			{
				Value: 1024000,
			},
		}, nil)
		defer mc.AssertExpectations(t)

		gw.On("Capability", capabilities.PressureSensorFlag).Return(mc)

		i := Interface{GatewayMux: mapper, Logger: logwrap.New(discard.Discard()), PublishSummaryState: true, publisher: m.Publish}

		expectedPayload := `{"Readings":[{"Value":1024000}]}`
		m.On("Publish", mock.Anything, fmt.Sprintf("devices/%s/capabilities/%s", d.DeviceIdentifier.String(), name), []byte(expectedPayload)).Return(nil)

		i.serviceUpdateOnEvent(capabilities.PressureSensorState{
			Device: d,
		})

		time.Sleep(50 * time.Millisecond)
	})

	t.Run("RelativeHumidity publishes a summary on Update if enabled", func(t *testing.T) {
		mapper := &gateway.MockMapper{}
		defer mapper.AssertExpectations(t)

		m := &MockPublisher{}
		defer m.AssertExpectations(t)

		gw := &mocks.Gateway{}
		defer gw.AssertExpectations(t)

		d := da.BaseDevice{
			DeviceGateway:      gw,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{capabilities.RelativeHumiditySensorFlag},
		}

		name := "RelativeHumidity"
		mc := &capmocks.RelativeHumiditySensor{}
		mc.On("Name").Return(name)
		mc.On("Reading", mock.Anything, d).Return([]capabilities.RelativeHumidityReading{
			{
				Value: 0.8,
			},
		}, nil)
		defer mc.AssertExpectations(t)

		gw.On("Capability", capabilities.RelativeHumiditySensorFlag).Return(mc)

		i := Interface{GatewayMux: mapper, Logger: logwrap.New(discard.Discard()), PublishSummaryState: true, publisher: m.Publish}

		expectedPayload := `{"Readings":[{"Value":0.8}]}`
		m.On("Publish", mock.Anything, fmt.Sprintf("devices/%s/capabilities/%s", d.DeviceIdentifier.String(), name), []byte(expectedPayload)).Return(nil)

		i.serviceUpdateOnEvent(capabilities.RelativeHumiditySensorState{
			Device: d,
		})

		time.Sleep(50 * time.Millisecond)
	})

	t.Run("TemperatureSensor publishes a summary on Update if enabled", func(t *testing.T) {
		mapper := &gateway.MockMapper{}
		defer mapper.AssertExpectations(t)

		m := &MockPublisher{}
		defer m.AssertExpectations(t)

		gw := &mocks.Gateway{}
		defer gw.AssertExpectations(t)

		d := da.BaseDevice{
			DeviceGateway:      gw,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{capabilities.TemperatureSensorFlag},
		}

		name := "TemperatureSensor"
		mc := &capmocks.TemperatureSensor{}
		mc.On("Name").Return(name)
		mc.On("Reading", mock.Anything, d).Return([]capabilities.TemperatureReading{
			{
				Value: 290,
			},
		}, nil)
		defer mc.AssertExpectations(t)

		gw.On("Capability", capabilities.TemperatureSensorFlag).Return(mc)

		i := Interface{GatewayMux: mapper, Logger: logwrap.New(discard.Discard()), PublishSummaryState: true, publisher: m.Publish}

		expectedPayload := `{"Readings":[{"Value":290}]}`
		m.On("Publish", mock.Anything, fmt.Sprintf("devices/%s/capabilities/%s", d.DeviceIdentifier.String(), name), []byte(expectedPayload)).Return(nil)

		i.serviceUpdateOnEvent(capabilities.TemperatureSensorState{
			Device: d,
		})

		time.Sleep(50 * time.Millisecond)
	})
}

type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) Publish(ctx context.Context, prefix string, payload []byte) error {
	args := m.Called(ctx, prefix, payload)
	return args.Error(0)
}
