package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type mockDeviceAction struct {
	mock.Mock
}

func (m *mockDeviceAction) doAction(ctx context.Context, d da.Device, c interface{}, a string, b []byte) (interface{}, error) {
	args := m.Called(ctx, d, c, a, b)
	return args.Get(0), args.Error(1)
}

func Test_deviceController_useDeviceCapabilityAction(t *testing.T) {
	t.Run("returns a 404 if device is not present", func(t *testing.T) {
		mgm := mockGatewayMapper{}
		defer mgm.AssertExpectations(t)

		mgm.On("Device", "one").Return(da.BaseDevice{}, false)

		controller := deviceController{gatewayMapper: &mgm}

		req, err := http.NewRequest("POST", "/devices/one/capabilities/name/action", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/devices/{identifier}/capabilities/{name}/{action}", controller.useDeviceCapabilityAction).Methods("POST")
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns a 404 if device does not support capability", func(t *testing.T) {
		mgm := mockGatewayMapper{}
		defer mgm.AssertExpectations(t)

		mgwOne := mockGateway{}
		defer mgwOne.AssertExpectations(t)

		mgm.On("Device", "one").Return(da.BaseDevice{
			DeviceCapabilities: []da.Capability{},
			DeviceGateway:      &mgwOne,
		}, true)

		controller := deviceController{gatewayMapper: &mgm}

		req, err := http.NewRequest("POST", "/devices/one/capabilities/name/action", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/devices/{identifier}/capabilities/{name}/{action}", controller.useDeviceCapabilityAction).Methods("POST")
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns a 404 if action is not recognised on capability", func(t *testing.T) {
		mgm := mockGatewayMapper{}
		defer mgm.AssertExpectations(t)

		mgwOne := mockGateway{}
		defer mgwOne.AssertExpectations(t)

		mbc := mockBasicCapability{}
		defer mbc.AssertExpectations(t)
		mbc.On("Name").Return("name")

		capOne := da.Capability(1)

		mgwOne.On("Capability", capOne).Return(&mbc)

		device := da.BaseDevice{
			DeviceCapabilities: []da.Capability{capOne},
			DeviceGateway:      &mgwOne,
		}
		mgm.On("Device", "one").Return(device, true)

		mda := mockDeviceAction{}
		defer mda.AssertExpectations(t)

		bodyText := "{}"

		mda.On("doAction", mock.Anything, device, &mbc, "action", []byte(bodyText)).Return(nil, ActionNotSupported)

		controller := deviceController{gatewayMapper: &mgm, deviceAction: mda.doAction}

		body := strings.NewReader(bodyText)

		req, err := http.NewRequest("POST", "/devices/one/capabilities/name/action", body)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/devices/{identifier}/capabilities/{name}/{action}", controller.useDeviceCapabilityAction).Methods("POST")
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns a 500 if action causes an error", func(t *testing.T) {
		mgm := mockGatewayMapper{}
		defer mgm.AssertExpectations(t)

		mgwOne := mockGateway{}
		defer mgwOne.AssertExpectations(t)

		mbc := mockBasicCapability{}
		defer mbc.AssertExpectations(t)
		mbc.On("Name").Return("name")

		capOne := da.Capability(1)

		mgwOne.On("Capability", capOne).Return(&mbc)

		device := da.BaseDevice{
			DeviceCapabilities: []da.Capability{capOne},
			DeviceGateway:      &mgwOne,
		}
		mgm.On("Device", "one").Return(device, true)

		mda := mockDeviceAction{}
		defer mda.AssertExpectations(t)

		bodyText := "{}"

		mda.On("doAction", mock.Anything, device, &mbc, "action", []byte(bodyText)).Return([]byte{}, fmt.Errorf("unknown error"))

		controller := deviceController{gatewayMapper: &mgm, deviceAction: mda.doAction}

		body := strings.NewReader(bodyText)

		req, err := http.NewRequest("POST", "/devices/one/capabilities/name/action", body)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/devices/{identifier}/capabilities/{name}/{action}", controller.useDeviceCapabilityAction).Methods("POST")
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("returns a 400 if user provides invalid data", func(t *testing.T) {
		mgm := mockGatewayMapper{}
		defer mgm.AssertExpectations(t)

		mgwOne := mockGateway{}
		defer mgwOne.AssertExpectations(t)

		mbc := mockBasicCapability{}
		defer mbc.AssertExpectations(t)
		mbc.On("Name").Return("name")

		capOne := da.Capability(1)

		mgwOne.On("Capability", capOne).Return(&mbc)

		device := da.BaseDevice{
			DeviceCapabilities: []da.Capability{capOne},
			DeviceGateway:      &mgwOne,
		}
		mgm.On("Device", "one").Return(device, true)

		mda := mockDeviceAction{}
		defer mda.AssertExpectations(t)

		bodyText := "{}"

		mda.On("doAction", mock.Anything, device, &mbc, "action", []byte(bodyText)).Return([]byte{}, fmt.Errorf("%w: unknown error", ActionUserError))

		controller := deviceController{gatewayMapper: &mgm, deviceAction: mda.doAction}

		body := strings.NewReader(bodyText)

		req, err := http.NewRequest("POST", "/devices/one/capabilities/name/action", body)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/devices/{identifier}/capabilities/{name}/{action}", controller.useDeviceCapabilityAction).Methods("POST")
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("returns a 200 with the body of the action", func(t *testing.T) {
		mgm := mockGatewayMapper{}
		defer mgm.AssertExpectations(t)

		mgwOne := mockGateway{}
		defer mgwOne.AssertExpectations(t)

		mbc := mockBasicCapability{}
		defer mbc.AssertExpectations(t)
		mbc.On("Name").Return("name")

		capOne := da.Capability(1)

		mgwOne.On("Capability", capOne).Return(&mbc)

		device := da.BaseDevice{
			DeviceCapabilities: []da.Capability{capOne},
			DeviceGateway:      &mgwOne,
		}
		mgm.On("Device", "one").Return(device, true)

		mda := mockDeviceAction{}
		defer mda.AssertExpectations(t)

		bodyText := "{}"

		mda.On("doAction", mock.Anything, device, &mbc, "action", []byte(bodyText)).Return(struct{}{}, nil)

		controller := deviceController{gatewayMapper: &mgm, deviceAction: mda.doAction}

		body := strings.NewReader(bodyText)

		req, err := http.NewRequest("POST", "/devices/one/capabilities/name/action", body)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/devices/{identifier}/capabilities/{name}/{action}", controller.useDeviceCapabilityAction).Methods("POST")
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		bodyContent, _ := ioutil.ReadAll(rr.Body)
		assert.Equal(t, "{}", string(bodyContent))
	})
}

func Test_doDeviceCapabilityAction_DeviceDiscovery(t *testing.T) {
	t.Run("Enable invokes the capability", func(t *testing.T) {
		mockCapability := &mockDeviceDiscovery{}
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
		mockCapability := &mockDeviceDiscovery{}
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
		mockCapability := &mockEnumerateDevice{}
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
		mockCapability := &mockOnOff{}
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
		mockCapability := &mockOnOff{}
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
		mockCapability := &mockAlarmWarningDevice{}
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
		mockCapability := &mockAlarmWarningDevice{}
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
		mockCapability := &mockAlarmWarningDevice{}
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
