package exporter

import (
	"context"
	"encoding/json"
	"github.com/shimmeringbee/controller/state"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/da/capabilities/color"
	capabilitymocks "github.com/shimmeringbee/da/capabilities/mocks"
	"github.com/shimmeringbee/da/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

type SimpleIdentifier struct {
	id string
}

func (s SimpleIdentifier) String() string {
	return s.id
}

func TestDeviceExporter_ExportDevice(t *testing.T) {
	t.Run("converts a da device with basic information and capability list", func(t *testing.T) {
		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		capOne := da.Capability(1)

		mockCapOne := mocks.BasicCapability{}
		defer mockCapOne.AssertExpectations(t)
		mockCapOne.On("Name").Return("capOne")
		mgwOne.On("Capability", capOne).Return(&mockCapOne)

		do := state.NewDeviceOrganiser()
		do.NewZone("one")
		do.AddDevice("one-one")
		do.NameDevice("one-one", "fancyname")
		do.AddDeviceToZone("one-one", 1)

		input := da.BaseDevice{
			DeviceGateway:      &mgwOne,
			DeviceIdentifier:   SimpleIdentifier{id: "one-one"},
			DeviceCapabilities: []da.Capability{capOne},
		}

		expected := ExportedDevice{
			Identifier:   "one-one",
			Capabilities: map[string]interface{}{"capOne": struct{}{}},
			Metadata: state.DeviceMetadata{
				Name:  "fancyname",
				Zones: []int{1},
			},
			Gateway: "gw",
		}

		mgm := state.MockMux{}
		defer mgm.AssertExpectations(t)

		mgm.On("GatewayName", mock.Anything).Return("gw", true)

		dc := DeviceExporter{DeviceOrganiser: &do, GatewayMapper: &mgm}
		actual := dc.ExportDevice(context.Background(), input)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertHasProductInformation(t *testing.T) {
	t.Run("retrieves and returns all data from HasProductInformation", func(t *testing.T) {
		d := da.BaseDevice{}

		mhpi := capabilitymocks.HasProductInformation{}
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

		dc := DeviceExporter{}
		actual := dc.convertHasProductInformation(context.Background(), d, &mhpi)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertTemperatureSensor(t *testing.T) {
	t.Run("retrieves and returns all data from TemperatureSensor", func(t *testing.T) {
		d := da.BaseDevice{}

		mts := capabilitymocks.TemperatureSensor{}
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

		dc := DeviceExporter{}
		actual := dc.convertTemperatureSensor(context.Background(), d, &mts)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertRelativeHumiditySensor(t *testing.T) {
	t.Run("retrieves and returns all data from RelativeHumiditySensor", func(t *testing.T) {
		d := da.BaseDevice{}

		mts := capabilitymocks.RelativeHumiditySensor{}
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

		dc := DeviceExporter{}
		actual := dc.convertRelativeHumiditySensor(context.Background(), d, &mts)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertPressureSensor(t *testing.T) {
	t.Run("retrieves and returns all data from PressureSensor", func(t *testing.T) {
		d := da.BaseDevice{}

		mts := capabilitymocks.PressureSensor{}
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

		dc := DeviceExporter{}
		actual := dc.convertPressureSensor(context.Background(), d, &mts)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertDeviceDiscovery(t *testing.T) {
	t.Run("retrieves and returns all data from DeviceDiscovery", func(t *testing.T) {
		d := da.BaseDevice{}

		mdd := capabilitymocks.DeviceDiscovery{}
		defer mdd.AssertExpectations(t)

		mdd.On("Status", mock.Anything, d).Return(capabilities.DeviceDiscoveryStatus{
			Discovering:       true,
			RemainingDuration: 12 * time.Second,
		}, nil)

		expected := &DeviceDiscovery{
			Discovering: true,
			Duration:    12000,
		}

		dc := DeviceExporter{}
		actual := dc.convertDeviceDiscovery(context.Background(), d, &mdd)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertEnumerateDevice(t *testing.T) {
	t.Run("retrieves and returns all data from EnumerateDevice", func(t *testing.T) {
		d := da.BaseDevice{}

		med := capabilitymocks.EnumerateDevice{}
		defer med.AssertExpectations(t)

		med.On("Status", mock.Anything, d).Return(capabilities.EnumerationStatus{
			Enumerating: true,
		}, nil)

		expected := &EnumerateDevice{
			Enumerating: true,
		}

		dc := DeviceExporter{}
		actual := dc.convertEnumerateDevice(context.Background(), d, &med)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertAlarmSensor(t *testing.T) {
	t.Run("retrieves and returns all data from AlarmSensor", func(t *testing.T) {
		d := da.BaseDevice{}

		mas := capabilitymocks.AlarmSensor{}
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

		dc := DeviceExporter{}
		actual := dc.convertAlarmSensor(context.Background(), d, &mas)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertOnOff(t *testing.T) {
	t.Run("retrieves and returns all data from OnOff", func(t *testing.T) {
		d := da.BaseDevice{}

		moo := capabilitymocks.OnOff{}
		defer moo.AssertExpectations(t)

		moo.Mock.On("Status", mock.Anything, d).Return(true, nil)

		expected := &OnOff{
			State: true,
		}

		dc := DeviceExporter{}
		actual := dc.convertOnOff(context.Background(), d, &moo)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertPowerStatus(t *testing.T) {
	t.Run("retrieves and returns all data from PowerSupply", func(t *testing.T) {
		d := da.BaseDevice{}

		mps := capabilitymocks.PowerSupply{}
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

		dc := DeviceExporter{}
		actual := dc.convertPowerSupply(context.Background(), d, &mps)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertAlarmWarningDevice(t *testing.T) {
	t.Run("retrieves and returns all data from AlarmWarningDevice", func(t *testing.T) {
		d := da.BaseDevice{}

		mawd := capabilitymocks.AlarmWarningDevice{}
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

		dc := DeviceExporter{}
		actual := dc.convertAlarmWarningDevice(context.Background(), d, &mawd)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertLevel(t *testing.T) {
	t.Run("retrieves and returns all data from OnOff", func(t *testing.T) {
		d := da.BaseDevice{}

		ml := capabilitymocks.Level{}
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

		dc := DeviceExporter{}
		actual := dc.convertLevel(context.Background(), d, &ml)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertColor(t *testing.T) {
	t.Run("retrieves and returns all data from Color, color output", func(t *testing.T) {
		d := da.BaseDevice{}

		mc := capabilitymocks.Color{}
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

		dc := DeviceExporter{}
		actual := dc.convertColor(context.Background(), d, &mc)

		assert.Equal(t, expected, actual)
	})

	t.Run("retrieves and returns all data from Color, temperature output", func(t *testing.T) {
		d := da.BaseDevice{}

		mc := capabilitymocks.Color{}
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

		dc := DeviceExporter{}
		actual := dc.convertColor(context.Background(), d, &mc)

		assert.Equal(t, expected, actual)
	})
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

		dc := DeviceExporter{}
		actual := dc.ExportCapability(context.Background(), d, &mts)

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

		dc := DeviceExporter{}
		actual := dc.ExportCapability(context.Background(), d, &mts)

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
