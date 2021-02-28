package v1

import (
	"context"
	"encoding/json"
	"github.com/shimmeringbee/controller/metadata"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/da/capabilities/color"
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

		expected := &HasProductInformation{
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

		expected := &TemperatureSensor{
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

		expected := &RelativeHumiditySensor{
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

		expected := &PressureSensor{
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

		expected := &DeviceDiscovery{
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

		expected := &EnumerateDevice{
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

		expected := &AlarmSensor{
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

		expected := &OnOff{
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

		expected := &PowerStatus{
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

type mockAlarmWarningDevice struct {
	mock.Mock
}

func (m *mockAlarmWarningDevice) Alarm(c context.Context, d da.Device, a capabilities.AlarmType, vol float64, vis bool, dur time.Duration) error {
	args := m.Called(c, d, a, vol, vis, dur)
	return args.Error(0)
}

func (m *mockAlarmWarningDevice) Clear(c context.Context, d da.Device) error {
	args := m.Called(c, d)
	return args.Error(0)
}

func (m *mockAlarmWarningDevice) Alert(c context.Context, d da.Device, alarm capabilities.AlarmType, alert capabilities.AlertType, vol float64, vis bool) error {
	args := m.Called(c, d, alarm, alert, vol, vis)
	return args.Error(0)
}

func (m *mockAlarmWarningDevice) Status(c context.Context, d da.Device) (capabilities.WarningDeviceState, error) {
	args := m.Called(c, d)
	return args.Get(0).(capabilities.WarningDeviceState), args.Error(1)
}

func Test_convertAlarmWarningDevice(t *testing.T) {
	t.Run("retrieves and returns all data from AlarmWarningDevice", func(t *testing.T) {
		d := da.BaseDevice{}

		mawd := mockAlarmWarningDevice{}
		defer mawd.AssertExpectations(t)

		retVal := capabilities.WarningDeviceState{
			Warning:           true,
			AlarmType:         capabilities.PanicAlarm,
			Volume:            0.8,
			Visual:            true,
			DurationRemaining: 60 * time.Second,
		}

		mawd.Mock.On("Status", mock.Anything, d).Return(retVal, nil)

		alarmTypeText := "Panic"

		remainingDuration := int(retVal.DurationRemaining / time.Millisecond)

		expected := &AlarmWarningDeviceStatus{
			Warning:   retVal.Warning,
			AlarmType: &alarmTypeText,
			Volume:    &retVal.Volume,
			Visual:    &retVal.Visual,
			Duration:  &remainingDuration,
		}

		dc := DeviceConverter{}
		actual := dc.convertAlarmWarningDevice(context.Background(), d, &mawd)

		assert.Equal(t, expected, actual)
	})
}

type mockLevel struct {
	mock.Mock
}

func (m *mockLevel) Status(c context.Context, d da.Device) (capabilities.LevelStatus, error) {
	args := m.Called(c, d)
	return args.Get(0).(capabilities.LevelStatus), args.Error(1)
}

func (m *mockLevel) Change(c context.Context, d da.Device, l float64, t time.Duration) error {
	args := m.Called(c, d, l, t)
	return args.Error(0)
}

func Test_convertLevel(t *testing.T) {
	t.Run("retrieves and returns all data from OnOff", func(t *testing.T) {
		d := da.BaseDevice{}

		ml := mockLevel{}
		defer ml.AssertExpectations(t)

		ml.Mock.On("Status", mock.Anything, d).Return(capabilities.LevelStatus{
			CurrentLevel:      0.5,
			TargetLevel:       0.7,
			DurationRemaining: 100 * time.Millisecond,
		}, nil)

		expected := &Level{
			Current:           0.5,
			Target:            0.7,
			DurationRemaining: 100,
		}

		dc := DeviceConverter{}
		actual := dc.convertLevel(context.Background(), d, &ml)

		assert.Equal(t, expected, actual)
	})
}

type mockColor struct {
	mock.Mock
}

func (m *mockColor) ChangeColor(ctx context.Context, d da.Device, color color.ConvertibleColor, duration time.Duration) error {
	args := m.Called(ctx, d, color, duration)
	return args.Error(0)
}

func (m *mockColor) ChangeTemperature(ctx context.Context, d da.Device, f float64, duration time.Duration) error {
	args := m.Called(ctx, d, f, duration)
	return args.Error(0)
}

func (m *mockColor) SupportsColor(ctx context.Context, device da.Device) (bool, error) {
	args := m.Called(ctx, device)
	return args.Bool(0), args.Error(1)
}

func (m *mockColor) SupportsTemperature(ctx context.Context, device da.Device) (bool, error) {
	args := m.Called(ctx, device)
	return args.Bool(0), args.Error(1)
}

func (m *mockColor) Status(ctx context.Context, d da.Device) (capabilities.ColorStatus, error) {
	args := m.Called(ctx, d)
	return args.Get(0).(capabilities.ColorStatus), args.Error(1)
}

func Test_convertColor(t *testing.T) {
	t.Run("retrieves and returns all data from Color, color output", func(t *testing.T) {
		d := da.BaseDevice{}

		mc := mockColor{}
		defer mc.AssertExpectations(t)

		mc.Mock.On("Status", mock.Anything, d).Return(capabilities.ColorStatus{
			Mode: capabilities.ColorMode,
			Color: capabilities.ColorSettings{
				Current: color.SRGBColor{R: 255, G: 192, B: 128},
				Target:  color.SRGBColor{R: 254, G: 191, B: 127},
			},
			DurationRemaining: 100 * time.Millisecond,
		}, nil)

		mc.Mock.On("SupportsColor", mock.Anything, d).Return(true, nil)
		mc.Mock.On("SupportsTemperature", mock.Anything, d).Return(true, nil)

		expected := &Color{
			DurationRemaining: 100,
			Current: &ColorState{
				Color: &ColorOutput{
					XYY: ColorOutputXYY{
						X:  0.4175453065956716,
						Y:  0.3949406589978435,
						Y2: 0.6051987031791847,
					},
					HSV: ColorOutputHSV{
						Hue:        30.236220472440944,
						Saturation: 0.4980392156862745,
						Value:      1,
					},
					RGB: ColorOutputRGB{
						R: 255,
						G: 192,
						B: 128,
					},
					Hex: "ffc080",
				},
			},
			Target: &ColorState{
				Color: &ColorOutput{
					XYY: ColorOutputXYY{
						X:  0.4175453065956716,
						Y:  0.3949406589978435,
						Y2: 0.6051987031791847,
					},
					HSV: ColorOutputHSV{
						Hue:        30.236220472440944,
						Saturation: 0.4980392156862745,
						Value:      1,
					},
					RGB: ColorOutputRGB{
						R: 255,
						G: 192,
						B: 128,
					},
					Hex: "ffc080",
				},
			},
			Supports: ColorSupports{
				Color:       true,
				Temperature: true,
			},
		}

		dc := DeviceConverter{}
		actual := dc.convertColor(context.Background(), d, &mc)

		assert.Equal(t, expected, actual)
	})

	t.Run("retrieves and returns all data from Color, temperature output", func(t *testing.T) {
		d := da.BaseDevice{}

		mc := mockColor{}
		defer mc.AssertExpectations(t)

		mc.Mock.On("Status", mock.Anything, d).Return(capabilities.ColorStatus{
			Mode: capabilities.TemperatureMode,
			Temperature: capabilities.TemperatureSettings{
				Current: 2400,
				Target:  2500,
			},
			DurationRemaining: 100 * time.Millisecond,
		}, nil)

		mc.Mock.On("SupportsColor", mock.Anything, d).Return(true, nil)
		mc.Mock.On("SupportsTemperature", mock.Anything, d).Return(true, nil)

		expected := &Color{
			DurationRemaining: 100,
			Current: &ColorState{
				Temperature: 2400,
			},
			Target: &ColorState{
				Temperature: 2500,
			},
			Supports: ColorSupports{
				Color:       true,
				Temperature: true,
			},
		}

		dc := DeviceConverter{}
		actual := dc.convertColor(context.Background(), d, &mc)

		assert.Equal(t, expected, actual)
	})
}

type mockDeviceRemoval struct {
	mock.Mock
}

func (m *mockDeviceRemoval) Remove(c context.Context, d da.Device) error {
	args := m.Called(c, d)
	return args.Error(0)
}

type mockTemperatureSensorWithUpdateTime struct {
	mock.Mock
}

func (m *mockTemperatureSensorWithUpdateTime) Reading(c context.Context, d da.Device) ([]capabilities.TemperatureReading, error) {
	args := m.Called(c, d)
	return args.Get(0).([]capabilities.TemperatureReading), args.Error(1)
}

func (m *mockTemperatureSensorWithUpdateTime) LastUpdateTime(c context.Context, d da.Device) (time.Time, error) {
	args := m.Called(c, d)
	return args.Get(0).(time.Time), args.Error(1)
}

type mockTemperatureSensorWithChangeTime struct {
	mock.Mock
}

func (m *mockTemperatureSensorWithChangeTime) Reading(c context.Context, d da.Device) ([]capabilities.TemperatureReading, error) {
	args := m.Called(c, d)
	return args.Get(0).([]capabilities.TemperatureReading), args.Error(1)
}

func (m *mockTemperatureSensorWithChangeTime) LastChangeTime(c context.Context, d da.Device) (time.Time, error) {
	args := m.Called(c, d)
	return args.Get(0).(time.Time), args.Error(1)
}

func Test_convertCapabilityWithLastUpdateTime(t *testing.T) {
	t.Run("retrieves and returns all data from TemperatureSensor", func(t *testing.T) {
		d := da.BaseDevice{}

		mts := mockTemperatureSensorWithUpdateTime{}
		defer mts.AssertExpectations(t)

		expectedTime := NullableTime(time.Now())

		mts.On("Reading", mock.Anything, d).Return([]capabilities.TemperatureReading{
			{
				Value: 100,
			},
		}, nil)

		mts.On("LastUpdateTime", mock.Anything, d).Return(time.Time(expectedTime), nil)

		expected := &TemperatureSensor{
			Readings: []capabilities.TemperatureReading{
				{
					Value: 100,
				},
			},
			LastUpdate: LastUpdate{
				LastUpdate: &expectedTime,
			},
		}

		dc := DeviceConverter{}
		actual := dc.convertDADeviceCapability(context.Background(), d, &mts)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertCapabilityWithLastChangeTime(t *testing.T) {
	t.Run("retrieves and returns all data from TemperatureSensor", func(t *testing.T) {
		d := da.BaseDevice{}

		mts := mockTemperatureSensorWithChangeTime{}
		defer mts.AssertExpectations(t)

		expectedTime := NullableTime(time.Now())

		mts.On("Reading", mock.Anything, d).Return([]capabilities.TemperatureReading{
			{
				Value: 100,
			},
		}, nil)

		mts.On("LastChangeTime", mock.Anything, d).Return(time.Time(expectedTime), nil)

		expected := &TemperatureSensor{
			Readings: []capabilities.TemperatureReading{
				{
					Value: 100,
				},
			},
			LastChange: LastChange{
				LastChange: &expectedTime,
			},
		}

		dc := DeviceConverter{}
		actual := dc.convertDADeviceCapability(context.Background(), d, &mts)

		assert.Equal(t, expected, actual)
	})
}

func TestNullableTime_MarshalJSON(t *testing.T) {
	t.Run("empty time marshals as null", func(t *testing.T) {
		n := NullableTime(time.Time{})

		data, err := json.Marshal(n)

		assert.NoError(t, err)
		assert.Equal(t, []byte("null"), data)
	})

	t.Run("full time marshals as normal", func(t *testing.T) {
		tn := time.Now()
		expectedData, _ := json.Marshal(tn)

		n := NullableTime(tn)

		data, err := json.Marshal(n)

		assert.NoError(t, err)
		assert.Equal(t, expectedData, data)
	})
}
