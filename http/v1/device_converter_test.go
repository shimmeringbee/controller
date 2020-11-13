package v1

import (
	"context"
	"github.com/shimmeringbee/controller/metadata"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

type mockDeviceConverter struct {
	mock.Mock
}

func (m *mockDeviceConverter) convertDevice(ctx context.Context, daDevice da.Device) device {
	args := m.Called(ctx, daDevice)
	return args.Get(0).(device)
}

type mockGatewayConverter struct {
	mock.Mock
}

func (m *mockGatewayConverter) convertDAGatewayToGateway(daGateway da.Gateway) gateway {
	args := m.Called(daGateway)
	return args.Get(0).(gateway)
}

func Test_convertDADeviceToDevice(t *testing.T) {
	t.Run("converts a da device with basic information and capability list", func(t *testing.T) {
		mgwOne := mockGateway{}
		defer mgwOne.AssertExpectations(t)

		capOne := da.Capability(1)

		mockCapOne := mockBasicCapability{}
		defer mockCapOne.AssertExpectations(t)
		mockCapOne.On("Name").Return("capOne")
		mgwOne.On("Capability", capOne).Return(&mockCapOne)

		do := metadata.NewDeviceOrganiser()
		do.NewZone("one")
		do.AddDevice("one-one")
		do.NameDevice("one-one", "fancyname")
		do.AddDeviceToZone("one-one", 1)

		input := da.BaseDevice{
			DeviceGateway:      &mgwOne,
			DeviceIdentifier:   SimpleIdentifier{id: "one-one"},
			DeviceCapabilities: []da.Capability{capOne},
		}

		expected := device{
			Identifier:   "one-one",
			Capabilities: map[string]interface{}{"capOne": struct{}{}},
			Metadata: metadata.DeviceMetadata{
				Name:  "fancyname",
				Zones: []int{1},
			},
			Gateway: "gw",
		}

		mgm := mockGatewayMapper{}
		defer mgm.AssertExpectations(t)

		mgm.On("GatewayName", mock.Anything).Return("gw", true)

		dc := DeviceConverter{deviceOrganiser: &do, gatewayMapper: &mgm}
		actual := dc.convertDevice(context.Background(), input)

		assert.Equal(t, expected, actual)
	})
}

type mockHasProductInformation struct {
	mock.Mock
}

func (m *mockHasProductInformation) ProductInformation(c context.Context, d da.Device) (capabilities.ProductInformation, error) {
	args := m.Called(c, d)
	return args.Get(0).(capabilities.ProductInformation), args.Error(1)
}

func Test_convertHasProductInformation(t *testing.T) {
	t.Run("retrieves and returns all data from HasProductInformation", func(t *testing.T) {
		d := da.BaseDevice{}

		mhpi := mockHasProductInformation{}
		defer mhpi.AssertExpectations(t)

		mhpi.On("ProductInformation", mock.Anything, d).Return(capabilities.ProductInformation{
			Present:      capabilities.Manufacturer | capabilities.Name | capabilities.Serial,
			Manufacturer: "manufacturer",
			Name:         "name",
			Serial:       "serial",
		}, nil)

		expected := HasProductInformation{
			Name:         "name",
			Manufacturer: "manufacturer",
			Serial:       "serial",
		}

		dc := DeviceConverter{}
		actual := dc.convertHasProductInformation(context.Background(), d, &mhpi)

		assert.Equal(t, expected, actual)
	})
}

type mockTemperatureSensor struct {
	mock.Mock
}

func (m *mockTemperatureSensor) Reading(c context.Context, d da.Device) ([]capabilities.TemperatureReading, error) {
	args := m.Called(c, d)
	return args.Get(0).([]capabilities.TemperatureReading), args.Error(1)
}

func Test_convertTemperatureSensor(t *testing.T) {
	t.Run("retrieves and returns all data from TemperatureSensor", func(t *testing.T) {
		d := da.BaseDevice{}

		mts := mockTemperatureSensor{}
		defer mts.AssertExpectations(t)

		mts.On("Reading", mock.Anything, d).Return([]capabilities.TemperatureReading{
			{
				Value: 100,
			},
		}, nil)

		expected := TemperatureSensor{
			Readings: []capabilities.TemperatureReading{
				{
					Value: 100,
				},
			},
		}

		dc := DeviceConverter{}
		actual := dc.convertTemperatureSensor(context.Background(), d, &mts)

		assert.Equal(t, expected, actual)
	})
}

type mockRelativeHumiditySensor struct {
	mock.Mock
}

func (m *mockRelativeHumiditySensor) Reading(c context.Context, d da.Device) ([]capabilities.RelativeHumidityReading, error) {
	args := m.Called(c, d)
	return args.Get(0).([]capabilities.RelativeHumidityReading), args.Error(1)
}

func Test_convertRelativeHumiditySensor(t *testing.T) {
	t.Run("retrieves and returns all data from RelativeHumiditySensor", func(t *testing.T) {
		d := da.BaseDevice{}

		mts := mockRelativeHumiditySensor{}
		defer mts.AssertExpectations(t)

		mts.On("Reading", mock.Anything, d).Return([]capabilities.RelativeHumidityReading{
			{
				Value: 100,
			},
		}, nil)

		expected := RelativeHumiditySensor{
			Readings: []capabilities.RelativeHumidityReading{
				{
					Value: 100,
				},
			},
		}

		dc := DeviceConverter{}
		actual := dc.convertRelativeHumiditySensor(context.Background(), d, &mts)

		assert.Equal(t, expected, actual)
	})
}

type mockPressureSensor struct {
	mock.Mock
}

func (m *mockPressureSensor) Reading(c context.Context, d da.Device) ([]capabilities.PressureReading, error) {
	args := m.Called(c, d)
	return args.Get(0).([]capabilities.PressureReading), args.Error(1)
}

func Test_convertPressureSensor(t *testing.T) {
	t.Run("retrieves and returns all data from PressureSensor", func(t *testing.T) {
		d := da.BaseDevice{}

		mts := mockPressureSensor{}
		defer mts.AssertExpectations(t)

		mts.On("Reading", mock.Anything, d).Return([]capabilities.PressureReading{
			{
				Value: 100,
			},
		}, nil)

		expected := PressureSensor{
			Readings: []capabilities.PressureReading{
				{
					Value: 100,
				},
			},
		}

		dc := DeviceConverter{}
		actual := dc.convertPressureSensor(context.Background(), d, &mts)

		assert.Equal(t, expected, actual)
	})
}

type mockDeviceDiscovery struct {
	mock.Mock
}

func (m *mockDeviceDiscovery) Status(c context.Context, d da.Device) (capabilities.DeviceDiscoveryStatus, error) {
	args := m.Called(c, d)
	return args.Get(0).(capabilities.DeviceDiscoveryStatus), args.Error(1)
}

func (m *mockDeviceDiscovery) Enable(c context.Context, d da.Device, du time.Duration) error {
	args := m.Called(c, d, du)
	return args.Error(0)
}

func (m *mockDeviceDiscovery) Disable(c context.Context, d da.Device) error {
	args := m.Called(c, d)
	return args.Error(0)
}

func Test_convertDeviceDiscovery(t *testing.T) {
	t.Run("retrieves and returns all data from DeviceDiscovery", func(t *testing.T) {
		d := da.BaseDevice{}

		mdd := mockDeviceDiscovery{}
		defer mdd.AssertExpectations(t)

		mdd.On("Status", mock.Anything, d).Return(capabilities.DeviceDiscoveryStatus{
			Discovering:       true,
			RemainingDuration: 12 * time.Second,
		}, nil)

		expected := DeviceDiscovery{
			Discovering: true,
			Duration:    12000,
		}

		dc := DeviceConverter{}
		actual := dc.convertDeviceDiscovery(context.Background(), d, &mdd)

		assert.Equal(t, expected, actual)
	})
}

type mockEnumerateDevice struct {
	mock.Mock
}

func (m *mockEnumerateDevice) Status(c context.Context, d da.Device) (capabilities.EnumerationStatus, error) {
	args := m.Called(c, d)
	return args.Get(0).(capabilities.EnumerationStatus), args.Error(1)
}

func (m *mockEnumerateDevice) Enumerate(c context.Context, d da.Device) error {
	args := m.Called(c, d)
	return args.Error(0)
}

func Test_convertEnumerateDevice(t *testing.T) {
	t.Run("retrieves and returns all data from EnumerateDevice", func(t *testing.T) {
		d := da.BaseDevice{}

		med := mockEnumerateDevice{}
		defer med.AssertExpectations(t)

		med.On("Status", mock.Anything, d).Return(capabilities.EnumerationStatus{
			Enumerating: true,
		}, nil)

		expected := EnumerateDevice{
			Enumerating: true,
		}

		dc := DeviceConverter{}
		actual := dc.convertEnumerateDevice(context.Background(), d, &med)

		assert.Equal(t, expected, actual)
	})
}

type mockAlarmSensor struct {
	mock.Mock
}

func (m *mockAlarmSensor) Status(c context.Context, d da.Device) (map[capabilities.SensorType]bool, error) {
	args := m.Called(c, d)
	return args.Get(0).(map[capabilities.SensorType]bool), args.Error(1)
}

func Test_convertAlarmSensor(t *testing.T) {
	t.Run("retrieves and returns all data from AlarmSensor", func(t *testing.T) {
		d := da.BaseDevice{}

		mas := mockAlarmSensor{}
		defer mas.AssertExpectations(t)

		mas.On("Status", mock.Anything, d).Return(map[capabilities.SensorType]bool{
			capabilities.FireBreakGlass:       true,
			capabilities.DeviceBatteryFailure: false,
		}, nil)

		expected := AlarmSensor{
			Alarms: map[string]bool{
				"FireBreakGlass":       true,
				"DeviceBatteryFailure": false,
			},
		}

		dc := DeviceConverter{}
		actual := dc.convertAlarmSensor(context.Background(), d, &mas)

		assert.Equal(t, expected, actual)
	})
}

type mockOnOff struct {
	mock.Mock
}

func (m *mockOnOff) Status(c context.Context, d da.Device) (bool, error) {
	args := m.Called(c, d)
	return args.Bool(0), args.Error(1)
}

func (m *mockOnOff) On(c context.Context, d da.Device) error {
	args := m.Called(c, d)
	return args.Error(0)
}

func (m *mockOnOff) Off(c context.Context, d da.Device) error {
	args := m.Called(c, d)
	return args.Error(0)
}

func Test_convertOnOff(t *testing.T) {
	t.Run("retrieves and returns all data from OnOff", func(t *testing.T) {
		d := da.BaseDevice{}

		moo := mockOnOff{}
		defer moo.AssertExpectations(t)

		moo.Mock.On("Status", mock.Anything, d).Return(true, nil)

		expected := OnOff{
			State: true,
		}

		dc := DeviceConverter{}
		actual := dc.convertOnOff(context.Background(), d, &moo)

		assert.Equal(t, expected, actual)
	})
}

type mockPowerSupply struct {
	mock.Mock
}

func (m *mockPowerSupply) Status(c context.Context, d da.Device) (capabilities.PowerStatus, error) {
	args := m.Called(c, d)
	return args.Get(0).(capabilities.PowerStatus), args.Error(1)
}

func Test_convertPowerStatus(t *testing.T) {
	t.Run("retrieves and returns all data from PowerSupply", func(t *testing.T) {
		d := da.BaseDevice{}

		mps := mockPowerSupply{}
		defer mps.AssertExpectations(t)

		mps.Mock.On("Status", mock.Anything, d).Return(capabilities.PowerStatus{
			Mains: []capabilities.PowerMainsStatus{
				{
					Voltage:   250,
					Frequency: 50.1,
					Available: true,
					Present:   capabilities.Voltage | capabilities.Frequency | capabilities.Available,
				},
			},
			Battery: []capabilities.PowerBatteryStatus{
				{
					Voltage:        3.2,
					MaximumVoltage: 3.7,
					MinimumVoltage: 3.1,
					Remaining:      0.21,
					Available:      true,
					Present:        capabilities.Voltage | capabilities.MaximumVoltage | capabilities.MinimumVoltage | capabilities.Remaining | capabilities.Available,
				},
			},
		}, nil)

		mainsVoltage := 250.0
		mainsFrequency := 50.1
		mainsAvailable := true

		batteryVoltage := 3.2
		batteryMaximumVoltage := 3.7
		batteryMinimumVoltage := 3.1
		batteryRemaining := 0.21
		batteryAvailable := true

		expected := PowerStatus{
			Mains: []PowerMainsStatus{
				{
					Voltage:   &mainsVoltage,
					Frequency: &mainsFrequency,
					Available: &mainsAvailable,
				},
			},
			Battery: []PowerBatteryStatus{
				{
					Voltage:        &batteryVoltage,
					MaximumVoltage: &batteryMaximumVoltage,
					MinimumVoltage: &batteryMinimumVoltage,
					Remaining:      &batteryRemaining,
					Available:      &batteryAvailable,
				},
			},
		}

		dc := DeviceConverter{}
		actual := dc.convertPowerSupply(context.Background(), d, &mps)

		assert.Equal(t, expected, actual)
	})
}
