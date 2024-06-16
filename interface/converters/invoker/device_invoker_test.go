package invoker

import (
	"context"
	"encoding/json"
	"github.com/shimmeringbee/controller/layers"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/da/capabilities/mocks"
	mocks2 "github.com/shimmeringbee/da/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestInvokeDeviceAction(t *testing.T) {
	t.Run("a payload with no output layer details uses provided values", func(t *testing.T) {
		mdev := &mocks2.MockDevice{}
		defer mdev.AssertExpectations(t)

		mdev.On("Capabilities").Return([]da.Capability{capabilities.DeviceDiscoveryFlag})

		expectedDuration := 10 * time.Minute

		inputBytes, _ := json.Marshal(DeviceDiscoveryEnable{Duration: 600000})
		capability := "DeviceDiscovery"
		action := "Enable"

		mockCapability := &mocks.DeviceDiscovery{}
		defer mockCapability.AssertExpectations(t)
		mockCapability.On("Enable", mock.Anything, expectedDuration).Return(nil)
		mockCapability.On("Name").Return(capability)

		mdev.On("Capability", capabilities.DeviceDiscoveryFlag).Return(mockCapability)

		mos := layers.MockOutputStack{}
		defer mos.AssertExpectations(t)

		mol := layers.MockOutputLayer{}
		defer mol.AssertExpectations(t)

		mol.On("Device", layers.Maintain, mdev).Return(mdev)

		expectedLayer := "layer"

		mos.On("Lookup", expectedLayer).Return(&mol)

		_, err := InvokeDeviceAction(context.Background(), &mos, expectedLayer, layers.Maintain, mdev, capability, action, inputBytes)
		assert.NoError(t, err)
	})

	t.Run("a payload with overridden output layer details uses new values", func(t *testing.T) {
		mdev := &mocks2.MockDevice{}
		defer mdev.AssertExpectations(t)

		mdev.On("Capabilities").Return([]da.Capability{capabilities.DeviceDiscoveryFlag})

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
		mockCapability.On("Enable", mock.Anything, expectedDuration).Return(nil)
		mockCapability.On("Name").Return(capability)

		mdev.On("Capability", capabilities.DeviceDiscoveryFlag).Return(mockCapability)

		mos := layers.MockOutputStack{}
		defer mos.AssertExpectations(t)

		mol := layers.MockOutputLayer{}
		defer mol.AssertExpectations(t)

		mol.On("Device", layers.Maintain, mdev).Return(mdev)

		expectedLayer := "layer"

		mos.On("Lookup", expectedLayer).Return(&mol)

		_, err := InvokeDeviceAction(context.Background(), &mos, "unusedLayer", layers.OneShot, mdev, capability, action, inputBytes)
		assert.NoError(t, err)
	})
}

func Test_doDeviceCapabilityAction_DeviceDiscovery(t *testing.T) {
	t.Run("Enable invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.DeviceDiscovery{}
		defer mockCapability.AssertExpectations(t)

		expectedDuration := 10 * time.Minute
		mockCapability.On("Enable", mock.Anything, expectedDuration).Return(nil)

		inputBytes, _ := json.Marshal(DeviceDiscoveryEnable{Duration: 600000})
		action := "Enable"

		expectedResult := struct{}{}

		actualResult, err := doDeviceCapabilityAction(context.Background(), mockCapability, action, inputBytes)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})

	t.Run("Disable invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.DeviceDiscovery{}
		defer mockCapability.AssertExpectations(t)

		mockCapability.On("Disable", mock.Anything).Return(nil)

		action := "Disable"

		expectedResult := struct{}{}

		actualResult, err := doDeviceCapabilityAction(context.Background(), mockCapability, action, nil)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})
}

func Test_doDeviceCapabilityAction_EnumerateDevice(t *testing.T) {
	t.Run("Enumerate invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.EnumerateDevice{}
		defer mockCapability.AssertExpectations(t)

		mockCapability.On("Enumerate", mock.Anything).Return(nil)

		action := "Enumerate"

		expectedResult := struct{}{}

		actualResult, err := doDeviceCapabilityAction(context.Background(), mockCapability, action, nil)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})
}

func Test_doDeviceCapabilityAction_OnOff(t *testing.T) {
	t.Run("On invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.OnOff{}
		defer mockCapability.AssertExpectations(t)

		mockCapability.Mock.On("On", mock.Anything).Return(nil)

		action := "On"

		expectedResult := struct{}{}

		actualResult, err := doDeviceCapabilityAction(context.Background(), mockCapability, action, nil)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})

	t.Run("Off invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.OnOff{}
		defer mockCapability.AssertExpectations(t)

		mockCapability.Mock.On("Off", mock.Anything).Return(nil)

		action := "Off"

		expectedResult := struct{}{}

		actualResult, err := doDeviceCapabilityAction(context.Background(), mockCapability, action, nil)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})
}

func Test_doAlarmWarningDevice_Test_doAlarmWarningDevice(t *testing.T) {
	t.Run("Alarm invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.AlarmWarningDevice{}
		defer mockCapability.AssertExpectations(t)

		mockCapability.On("Alarm", mock.Anything, capabilities.PanicAlarm, 0.5, true, 60*time.Second).Return(nil)

		action := "Alarm"

		expectedResult := struct{}{}

		inputBytes, _ := json.Marshal(AlarmWarningDeviceAlarm{
			AlarmType: "Panic",
			Volume:    0.5,
			Visual:    true,
			Duration:  60000,
		})
		actualResult, err := doDeviceCapabilityAction(context.Background(), mockCapability, action, inputBytes)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})

	t.Run("Alert invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.AlarmWarningDevice{}
		defer mockCapability.AssertExpectations(t)

		mockCapability.On("Alert", mock.Anything, capabilities.PanicAlarm, capabilities.PreAlarmAlert, 0.5, true).Return(nil)

		action := "Alert"

		expectedResult := struct{}{}

		inputBytes, _ := json.Marshal(AlarmWarningDeviceAlert{
			AlarmType: "Panic",
			AlertType: "PreAlarm",
			Volume:    0.5,
			Visual:    true,
		})
		actualResult, err := doDeviceCapabilityAction(context.Background(), mockCapability, action, inputBytes)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})

	t.Run("Clear invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.AlarmWarningDevice{}
		defer mockCapability.AssertExpectations(t)

		mockCapability.On("Clear", mock.Anything).Return(nil)

		action := "Clear"

		expectedResult := struct{}{}

		actualResult, err := doDeviceCapabilityAction(context.Background(), mockCapability, action, nil)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})
}

func Test_doDeviceCapabilityAction_DeviceRemoval(t *testing.T) {
	t.Run("Remove invokes the capability", func(t *testing.T) {
		mockCapability := &mocks.DeviceRemoval{}
		defer mockCapability.AssertExpectations(t)

		mockCapability.On("Remove", mock.Anything, capabilities.Force).Return(nil)

		inputBytes, _ := json.Marshal(RemoveDevice{Force: true})
		action := "Remove"

		expectedResult := struct{}{}

		actualResult, err := doDeviceCapabilityAction(context.Background(), mockCapability, action, inputBytes)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})
}
