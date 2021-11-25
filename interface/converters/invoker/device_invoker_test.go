package invoker

import (
	"context"
	"encoding/json"
	"github.com/shimmeringbee/controller/layers"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/da/capabilities/color"
	"github.com/shimmeringbee/da/capabilities/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestInvokeDeviceAction(t *testing.T) {
	t.Run("a payload with no output layer details uses provided values", func(t *testing.T) {
		d := da.BaseDevice{DeviceCapabilities: []da.Capability{capabilities.DeviceDiscoveryFlag}}
		expectedDuration := 10 * time.Minute

		inputBytes, _ := json.Marshal(DeviceDiscoveryEnable{Duration: 600000})
		capability := "DeviceDiscovery"
		action := "Enable"

		mockCapability := &mocks.DeviceDiscovery{}
		defer mockCapability.AssertExpectations(t)
		mockCapability.On("Enable", mock.Anything, d, expectedDuration).Return(nil)
		mockCapability.On("Name").Return(capability)

		mos := layers.MockOutputStack{}
		defer mos.AssertExpectations(t)

		mol := layers.MockOutputLayer{}
		defer mol.AssertExpectations(t)

		mol.On("Capability", layers.Maintain, capabilities.DeviceDiscoveryFlag, d).Return(mockCapability)

		expectedLayer := "layer"

		mos.On("Lookup", expectedLayer).Return(&mol)

		_, err := InvokeDeviceAction(context.Background(), &mos, expectedLayer, layers.Maintain, d, capability, action, inputBytes)
		assert.NoError(t, err)
	})

	t.Run("a payload with overridden output layer details uses new values", func(t *testing.T) {
		d := da.BaseDevice{DeviceCapabilities: []da.Capability{capabilities.DeviceDiscoveryFlag}}
		expectedDuration := 0 * time.Minute

		inputBytes := []byte(`{
  "control": {
    "output": {
      "layer": "layer",
      "retention": "maintain"
    }
  }
}`)

		capability := "DeviceDiscovery"
		action := "Enable"

		mockCapability := &mocks.DeviceDiscovery{}
		defer mockCapability.AssertExpectations(t)
		mockCapability.On("Enable", mock.Anything, d, expectedDuration).Return(nil)
		mockCapability.On("Name").Return(capability)

		mos := layers.MockOutputStack{}
		defer mos.AssertExpectations(t)

		mol := layers.MockOutputLayer{}
		defer mol.AssertExpectations(t)

		mol.On("Capability", layers.Maintain, capabilities.DeviceDiscoveryFlag, d).Return(mockCapability)

		expectedLayer := "layer"

		mos.On("Lookup", expectedLayer).Return(&mol)

		_, err := InvokeDeviceAction(context.Background(), &mos, "unusedLayer", layers.OneShot, d, capability, action, inputBytes)
		assert.NoError(t, err)
	})
}

func Test_doDeviceCapabilityAction_DeviceDiscovery(t *testing.T) {
	t.Run("Enable invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.DeviceDiscovery{}
		defer mockCapability.AssertExpectations(t)

		device := da.BaseDevice{}
		expectedDuration := 10 * time.Minute
		mockCapability.On("Enable", mock.Anything, device, expectedDuration).Return(nil)

		inputBytes, _ := json.Marshal(DeviceDiscoveryEnable{Duration: 600000})
		action := "Enable"

		expectedResult := struct{}{}

		actualResult, err := doDeviceCapabilityAction(context.Background(), device, mockCapability, action, inputBytes)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})

	t.Run("Disable invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.DeviceDiscovery{}
		defer mockCapability.AssertExpectations(t)

		device := da.BaseDevice{}
		mockCapability.On("Disable", mock.Anything, device).Return(nil)

		action := "Disable"

		expectedResult := struct{}{}

		actualResult, err := doDeviceCapabilityAction(context.Background(), device, mockCapability, action, nil)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})
}

func Test_doDeviceCapabilityAction_EnumerateDevice(t *testing.T) {
	t.Run("Enumerate invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.EnumerateDevice{}
		defer mockCapability.AssertExpectations(t)

		device := da.BaseDevice{}
		mockCapability.On("Enumerate", mock.Anything, device).Return(nil)

		action := "Enumerate"

		expectedResult := struct{}{}

		actualResult, err := doDeviceCapabilityAction(context.Background(), device, mockCapability, action, nil)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})
}

func Test_doDeviceCapabilityAction_OnOff(t *testing.T) {
	t.Run("On invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.OnOff{}
		defer mockCapability.AssertExpectations(t)

		device := da.BaseDevice{}
		mockCapability.Mock.On("On", mock.Anything, device).Return(nil)

		action := "On"

		expectedResult := struct{}{}

		actualResult, err := doDeviceCapabilityAction(context.Background(), device, mockCapability, action, nil)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})

	t.Run("Off invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.OnOff{}
		defer mockCapability.AssertExpectations(t)

		device := da.BaseDevice{}
		mockCapability.Mock.On("Off", mock.Anything, device).Return(nil)

		action := "Off"

		expectedResult := struct{}{}

		actualResult, err := doDeviceCapabilityAction(context.Background(), device, mockCapability, action, nil)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})
}

func Test_doAlarmWarningDevice_Test_doAlarmWarningDevice(t *testing.T) {
	t.Run("Alarm invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.AlarmWarningDevice{}
		defer mockCapability.AssertExpectations(t)

		device := da.BaseDevice{}
		mockCapability.On("Alarm", mock.Anything, device, capabilities.PanicAlarm, 0.5, true, 60*time.Second).Return(nil)

		action := "Alarm"

		expectedResult := struct{}{}

		inputBytes, _ := json.Marshal(AlarmWarningDeviceAlarm{
			AlarmType: "Panic",
			Volume:    0.5,
			Visual:    true,
			Duration:  60000,
		})
		actualResult, err := doDeviceCapabilityAction(context.Background(), device, mockCapability, action, inputBytes)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})

	t.Run("Alert invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.AlarmWarningDevice{}
		defer mockCapability.AssertExpectations(t)

		device := da.BaseDevice{}
		mockCapability.On("Alert", mock.Anything, device, capabilities.PanicAlarm, capabilities.PreAlarmAlert, 0.5, true).Return(nil)

		action := "Alert"

		expectedResult := struct{}{}

		inputBytes, _ := json.Marshal(AlarmWarningDeviceAlert{
			AlarmType: "Panic",
			AlertType: "PreAlarm",
			Volume:    0.5,
			Visual:    true,
		})
		actualResult, err := doDeviceCapabilityAction(context.Background(), device, mockCapability, action, inputBytes)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})

	t.Run("Clear invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.AlarmWarningDevice{}
		defer mockCapability.AssertExpectations(t)

		device := da.BaseDevice{}
		mockCapability.On("Clear", mock.Anything, device).Return(nil)

		action := "Clear"

		expectedResult := struct{}{}

		actualResult, err := doDeviceCapabilityAction(context.Background(), device, mockCapability, action, nil)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})
}

func Test_doDeviceCapabilityAction_Level(t *testing.T) {
	t.Run("Change invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.Level{}
		defer mockCapability.AssertExpectations(t)

		device := da.BaseDevice{}
		level := 0.5
		expectedDuration := 1 * time.Second
		mockCapability.On("Change", mock.Anything, device, level, expectedDuration).Return(nil)

		inputBytes, _ := json.Marshal(LevelChange{Level: level, Duration: 1000})
		action := "Change"

		expectedResult := struct{}{}

		actualResult, err := doLevel(context.Background(), device, mockCapability, action, inputBytes)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})
}

func Test_doDeviceCapabilityAction_Color(t *testing.T) {
	t.Run("ChangeColor invokes the capability, XYY", func(t *testing.T) {
		mockCapability := &mocks.Color{}
		defer mockCapability.AssertExpectations(t)

		device := da.BaseDevice{}
		color := color.XYColor{
			X:  0.25,
			Y:  0.50,
			Y2: 0.75,
		}

		expectedDuration := 1 * time.Second
		mockCapability.On("ChangeColor", mock.Anything, device, color, expectedDuration).Return(nil)

		inputBytes, _ := json.Marshal(ColorChangeColor{
			XYY: &ColorChangeColorXYY{
				X:  color.X,
				Y:  color.Y,
				Y2: color.Y2,
			},
			HSV:      nil,
			RGB:      nil,
			Duration: 1000,
		})
		action := "ChangeColor"

		expectedResult := struct{}{}

		actualResult, err := doColor(context.Background(), device, mockCapability, action, inputBytes)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})

	t.Run("ChangeColor invokes the capability, HSV", func(t *testing.T) {
		mockCapability := &mocks.Color{}
		defer mockCapability.AssertExpectations(t)

		device := da.BaseDevice{}
		color := color.HSVColor{
			Hue:   180.0,
			Sat:   0.5,
			Value: 1.0,
		}

		expectedDuration := 1 * time.Second
		mockCapability.On("ChangeColor", mock.Anything, device, color, expectedDuration).Return(nil)

		inputBytes, _ := json.Marshal(ColorChangeColor{
			XYY: nil,
			HSV: &ColorChangeColorHSV{
				Hue:        color.Hue,
				Saturation: color.Sat,
				Value:      color.Value,
			},
			RGB:      nil,
			Duration: 1000,
		})
		action := "ChangeColor"

		expectedResult := struct{}{}

		actualResult, err := doColor(context.Background(), device, mockCapability, action, inputBytes)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})

	t.Run("ChangeColor invokes the capability, RGB", func(t *testing.T) {
		mockCapability := &mocks.Color{}
		defer mockCapability.AssertExpectations(t)

		device := da.BaseDevice{}
		color := color.SRGBColor{
			R: 192,
			G: 128,
			B: 64,
		}

		expectedDuration := 1 * time.Second
		mockCapability.On("ChangeColor", mock.Anything, device, color, expectedDuration).Return(nil)

		inputBytes, _ := json.Marshal(ColorChangeColor{
			XYY: nil,
			HSV: nil,
			RGB: &ColorChangeColorRGB{
				R: color.R,
				G: color.G,
				B: color.B,
			},
			Duration: 1000,
		})
		action := "ChangeColor"

		expectedResult := struct{}{}

		actualResult, err := doColor(context.Background(), device, mockCapability, action, inputBytes)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})

	t.Run("ChangeTemperature invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.Color{}
		defer mockCapability.AssertExpectations(t)

		device := da.BaseDevice{}
		temperature := 2500.5
		expectedDuration := 1 * time.Second
		mockCapability.On("ChangeTemperature", mock.Anything, device, temperature, expectedDuration).Return(nil)

		inputBytes, _ := json.Marshal(ColorChangeTemperature{Temperature: temperature, Duration: 1000})
		action := "ChangeTemperature"

		expectedResult := struct{}{}

		actualResult, err := doColor(context.Background(), device, mockCapability, action, inputBytes)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})
}

func Test_doDeviceCapabilityAction_DeviceRemoval(t *testing.T) {
	t.Run("Remove invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.DeviceRemoval{}
		defer mockCapability.AssertExpectations(t)

		device := da.BaseDevice{}
		mockCapability.On("Remove", mock.Anything, device, capabilities.Force).Return(nil)

		inputBytes, _ := json.Marshal(RemoveDevice{Force: true})
		action := "Remove"

		expectedResult := struct{}{}

		actualResult, err := doDeviceCapabilityAction(context.Background(), device, mockCapability, action, inputBytes)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})
}
