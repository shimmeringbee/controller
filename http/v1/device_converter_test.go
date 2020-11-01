package v1

import (
	"context"
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

func (m *mockDeviceConverter) convertDADeviceToDevice(ctx context.Context, daDevice da.Device) device {
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

		input := da.BaseDevice{
			DeviceGateway:      &mgwOne,
			DeviceIdentifier:   SimpleIdentifier{id: "one-one"},
			DeviceCapabilities: []da.Capability{capOne},
		}

		expected := device{
			Identifier:   "one-one",
			Capabilities: map[string]interface{}{"capOne": struct{}{}},
		}

		actual := convertDADeviceToDevice(context.Background(), input)

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

		actual := convertHasProductInformation(context.Background(), d, &mhpi)

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

		actual := convertTemperatureSensor(context.Background(), d, &mts)

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

		actual := convertRelativeHumiditySensor(context.Background(), d, &mts)

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

		actual := convertPressureSensor(context.Background(), d, &mts)

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

		actual := convertDeviceDiscovery(context.Background(), d, &mdd)

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

		actual := convertEnumerateDevice(context.Background(), d, &med)

		assert.Equal(t, expected, actual)
	})
}

type mockAlarmSensor struct {
	mock.Mock
}

func (m *mockAlarmSensor) Status(c context.Context, d da.Device) ([]capabilities.AlarmSensorState, error) {
	args := m.Called(c, d)
	return args.Get(0).([]capabilities.AlarmSensorState), args.Error(1)
}

func Test_convertAlarmSensor(t *testing.T) {
	t.Run("retrieves and returns all data from AlarmSensor", func(t *testing.T) {
		d := da.BaseDevice{}

		mas := mockAlarmSensor{}
		defer mas.AssertExpectations(t)

		mas.On("Status", mock.Anything, d).Return([]capabilities.AlarmSensorState{
			{
				SensorType: capabilities.FireBreakGlass,
				InAlarm:    true,
			},
			{
				SensorType: capabilities.DeviceBatteryFailure,
				InAlarm:    false,
			},
		}, nil)

		expected := AlarmSensor{
			Alarms: map[string]bool{
				"FireBreakGlass":       true,
				"DeviceBatteryFailure": false,
			},
		}

		actual := convertAlarmSensor(context.Background(), d, &mas)

		assert.Equal(t, expected, actual)
	})
}
