package exporter

import (
	"context"
	"encoding/json"
	"github.com/shimmeringbee/controller/state"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	capabilitymocks "github.com/shimmeringbee/da/capabilities/mocks"
	"github.com/shimmeringbee/da/mocks"
	"github.com/shimmeringbee/persistence/impl/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
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

		do := state.NewDeviceOrganiser(memory.New())
		do.NewZone("one")
		do.AddDevice("one-one")
		do.NameDevice("one-one", "fancyname")
		do.AddDeviceToZone("one-one", 1)

		mdev := &mocks.MockDevice{}
		defer mdev.AssertExpectations(t)

		mdev.On("Gateway").Return(&mgwOne)
		mdev.On("Identifier").Return(SimpleIdentifier{id: "one-one"})
		mdev.On("Capabilities").Return([]da.Capability{capOne})
		mdev.On("Capability", capOne).Return(&mockCapOne)

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
		actual := dc.ExportDevice(context.Background(), mdev)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertHasProductInformation(t *testing.T) {
	t.Run("retrieves and returns all data from HasProductInformation", func(t *testing.T) {
		mhpi := capabilitymocks.ProductInformation{}
		defer mhpi.AssertExpectations(t)

		mhpi.On("Get", mock.Anything).Return(capabilities.ProductInfo{
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
		actual := dc.convertHasProductInformation(context.Background(), &mhpi)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertTemperatureSensor(t *testing.T) {
	t.Run("retrieves and returns all data from TemperatureSensor", func(t *testing.T) {
		mts := capabilitymocks.TemperatureSensor{}
		defer mts.AssertExpectations(t)

		mts.On("Reading", mock.Anything).Return([]capabilities.TemperatureReading{
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
		actual := dc.convertTemperatureSensor(context.Background(), &mts)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertRelativeHumiditySensor(t *testing.T) {
	t.Run("retrieves and returns all data from RelativeHumiditySensor", func(t *testing.T) {
		mts := capabilitymocks.RelativeHumiditySensor{}
		defer mts.AssertExpectations(t)

		mts.On("Reading", mock.Anything).Return([]capabilities.RelativeHumidityReading{
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
		actual := dc.convertRelativeHumiditySensor(context.Background(), &mts)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertPressureSensor(t *testing.T) {
	t.Run("retrieves and returns all data from PressureSensor", func(t *testing.T) {
		mts := capabilitymocks.PressureSensor{}
		defer mts.AssertExpectations(t)

		mts.On("Reading", mock.Anything).Return([]capabilities.PressureReading{
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
		actual := dc.convertPressureSensor(context.Background(), &mts)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertDeviceDiscovery(t *testing.T) {
	t.Run("retrieves and returns all data from DeviceDiscovery", func(t *testing.T) {
		mdd := capabilitymocks.DeviceDiscovery{}
		defer mdd.AssertExpectations(t)

		mdd.On("Status", mock.Anything).Return(capabilities.DeviceDiscoveryStatus{
			Discovering:       true,
			RemainingDuration: 12 * time.Second,
		}, nil)

		expected := &DeviceDiscovery{
			Discovering: true,
			Duration:    12000,
		}

		dc := DeviceExporter{}
		actual := dc.convertDeviceDiscovery(context.Background(), &mdd)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertEnumerateDevice(t *testing.T) {
	t.Run("retrieves and returns all data from EnumerateDevice", func(t *testing.T) {
		med := capabilitymocks.EnumerateDevice{}
		defer med.AssertExpectations(t)

		med.On("Status", mock.Anything).Return(capabilities.EnumerationStatus{
			Enumerating: true,
			CapabilityStatus: map[da.Capability]capabilities.EnumerationCapability{
				capabilities.OnOffFlag: {Attached: true, Errors: []error{io.EOF}},
			},
		}, nil)

		expected := &EnumerateDevice{
			Enumerating: true,
			Status: map[string]EnumerateDeviceCapability{
				"OnOff": {
					Attached: true,
					Errors:   []string{io.EOF.Error()},
				},
			},
		}

		dc := DeviceExporter{}
		actual := dc.convertEnumerateDevice(context.Background(), &med)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertAlarmSensor(t *testing.T) {
	t.Run("retrieves and returns all data from AlarmSensor", func(t *testing.T) {
		mas := capabilitymocks.AlarmSensor{}
		defer mas.AssertExpectations(t)

		mas.On("Status", mock.Anything).Return(map[capabilities.SensorType]bool{
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
		actual := dc.convertAlarmSensor(context.Background(), &mas)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertOnOff(t *testing.T) {
	t.Run("retrieves and returns all data from OnOff", func(t *testing.T) {
		moo := capabilitymocks.OnOff{}
		defer moo.AssertExpectations(t)

		moo.Mock.On("Status", mock.Anything).Return(true, nil)

		expected := &OnOff{
			State: true,
		}

		dc := DeviceExporter{}
		actual := dc.convertOnOff(context.Background(), &moo)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertPowerStatus(t *testing.T) {
	t.Run("retrieves and returns all data from PowerSupply", func(t *testing.T) {
		mps := capabilitymocks.PowerSupply{}
		defer mps.AssertExpectations(t)

		mps.Mock.On("Status", mock.Anything).Return(capabilities.PowerState{
			Mains: []capabilities.PowerMainsState{
				{
					Voltage:   250,
					Frequency: 50.1,
					Available: true,
					Present:   capabilities.Voltage | capabilities.Frequency | capabilities.Available,
				},
			},
			Battery: []capabilities.PowerBatteryState{
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
		actual := dc.convertPowerSupply(context.Background(), &mps)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertAlarmWarningDevice(t *testing.T) {
	t.Run("retrieves and returns all data from AlarmWarningDevice", func(t *testing.T) {
		mawd := capabilitymocks.AlarmWarningDevice{}
		defer mawd.AssertExpectations(t)

		retVal := capabilities.WarningDeviceState{
			Warning:           true,
			AlarmType:         capabilities.PanicAlarm,
			Volume:            0.8,
			Visual:            true,
			DurationRemaining: 60 * time.Second,
		}

		mawd.Mock.On("Status", mock.Anything).Return(retVal, nil)

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
		actual := dc.convertAlarmWarningDevice(context.Background(), &mawd)

		assert.Equal(t, expected, actual)
	})
}

type mockTemperatureSensorWithUpdateTime struct {
	mock.Mock
}

func (m *mockTemperatureSensorWithUpdateTime) Reading(c context.Context) ([]capabilities.TemperatureReading, error) {
	args := m.Called(c)
	return args.Get(0).([]capabilities.TemperatureReading), args.Error(1)
}

func (m *mockTemperatureSensorWithUpdateTime) LastUpdateTime(c context.Context) (time.Time, error) {
	args := m.Called(c)
	return args.Get(0).(time.Time), args.Error(1)
}

type mockTemperatureSensorWithChangeTime struct {
	mock.Mock
}

func (m *mockTemperatureSensorWithChangeTime) Reading(c context.Context) ([]capabilities.TemperatureReading, error) {
	args := m.Called(c)
	return args.Get(0).([]capabilities.TemperatureReading), args.Error(1)
}

func (m *mockTemperatureSensorWithChangeTime) LastChangeTime(c context.Context) (time.Time, error) {
	args := m.Called(c)
	return args.Get(0).(time.Time), args.Error(1)
}

func Test_convertCapabilityWithLastUpdateTime(t *testing.T) {
	t.Run("retrieves and returns all data from TemperatureSensor", func(t *testing.T) {
		mts := mockTemperatureSensorWithUpdateTime{}
		defer mts.AssertExpectations(t)

		expectedTime := NullableTime(time.Now())

		mts.On("Reading", mock.Anything).Return([]capabilities.TemperatureReading{
			{
				Value: 100,
			},
		}, nil)

		mts.On("LastUpdateTime", mock.Anything).Return(time.Time(expectedTime), nil)

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
		actual := dc.ExportCapability(context.Background(), &mts)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertCapabilityWithLastChangeTime(t *testing.T) {
	t.Run("retrieves and returns all data from TemperatureSensor", func(t *testing.T) {
		mts := mockTemperatureSensorWithChangeTime{}
		defer mts.AssertExpectations(t)

		expectedTime := NullableTime(time.Now())

		mts.On("Reading", mock.Anything).Return([]capabilities.TemperatureReading{
			{
				Value: 100,
			},
		}, nil)

		mts.On("LastChangeTime", mock.Anything).Return(time.Time(expectedTime), nil)

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
		actual := dc.ExportCapability(context.Background(), &mts)

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
