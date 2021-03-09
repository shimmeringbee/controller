package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/controller/gateway"
	"github.com/shimmeringbee/controller/layers"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/da/capabilities/color"
	capabilitymocks "github.com/shimmeringbee/da/capabilities/mocks"
	"github.com/shimmeringbee/da/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type MockDeviceAction struct {
	mock.Mock
}

func (m *MockDeviceAction) doAction(ctx context.Context, d da.Device, c interface{}, a string, b []byte) (interface{}, error) {
	args := m.Called(ctx, d, c, a, b)
	return args.Get(0), args.Error(1)
}

func Test_deviceController_useDeviceCapabilityAction(t *testing.T) {
	t.Run("returns a 404 if device is not present", func(t *testing.T) {
		mgm := gateway.MockMapper{}
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
		mgm := gateway.MockMapper{}
		defer mgm.AssertExpectations(t)

		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		mgm.On("Device", "one").Return(da.BaseDevice{
			DeviceCapabilities: []da.Capability{},
			DeviceGateway:      &mgwOne,
		}, true)

		controller := deviceController{gatewayMapper: &mgm, stack: layers.PassThruStack{}}

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
		mgm := gateway.MockMapper{}
		defer mgm.AssertExpectations(t)

		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		mbc := mocks.BasicCapability{}
		defer mbc.AssertExpectations(t)
		mbc.On("Name").Return("name")

		capOne := da.Capability(1)

		mgwOne.On("Capability", capOne).Return(&mbc)

		device := da.BaseDevice{
			DeviceCapabilities: []da.Capability{capOne},
			DeviceGateway:      &mgwOne,
		}
		mgm.On("Device", "one").Return(device, true)

		mda := MockDeviceAction{}
		defer mda.AssertExpectations(t)

		bodyText := "{}"

		mda.On("doAction", mock.Anything, device, &mbc, "action", []byte(bodyText)).Return(nil, ActionNotSupported)

		controller := deviceController{gatewayMapper: &mgm, deviceAction: mda.doAction, stack: layers.PassThruStack{}}

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
		mgm := gateway.MockMapper{}
		defer mgm.AssertExpectations(t)

		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		mbc := mocks.BasicCapability{}
		defer mbc.AssertExpectations(t)
		mbc.On("Name").Return("name")

		capOne := da.Capability(1)

		mgwOne.On("Capability", capOne).Return(&mbc)

		device := da.BaseDevice{
			DeviceCapabilities: []da.Capability{capOne},
			DeviceGateway:      &mgwOne,
		}
		mgm.On("Device", "one").Return(device, true)

		mda := MockDeviceAction{}
		defer mda.AssertExpectations(t)

		bodyText := "{}"

		mda.On("doAction", mock.Anything, device, &mbc, "action", []byte(bodyText)).Return([]byte{}, fmt.Errorf("unknown error"))

		controller := deviceController{gatewayMapper: &mgm, deviceAction: mda.doAction, stack: layers.PassThruStack{}}

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
		mgm := gateway.MockMapper{}
		defer mgm.AssertExpectations(t)

		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		mbc := mocks.BasicCapability{}
		defer mbc.AssertExpectations(t)
		mbc.On("Name").Return("name")

		capOne := da.Capability(1)

		mgwOne.On("Capability", capOne).Return(&mbc)

		device := da.BaseDevice{
			DeviceCapabilities: []da.Capability{capOne},
			DeviceGateway:      &mgwOne,
		}
		mgm.On("Device", "one").Return(device, true)

		mda := MockDeviceAction{}
		defer mda.AssertExpectations(t)

		bodyText := "{}"

		mda.On("doAction", mock.Anything, device, &mbc, "action", []byte(bodyText)).Return([]byte{}, fmt.Errorf("%w: unknown error", ActionUserError))

		controller := deviceController{gatewayMapper: &mgm, deviceAction: mda.doAction, stack: layers.PassThruStack{}}

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

	t.Run("returns a 400 if the layer does not exist", func(t *testing.T) {
		mgm := gateway.MockMapper{}
		defer mgm.AssertExpectations(t)

		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		mbc := mocks.BasicCapability{}
		defer mbc.AssertExpectations(t)

		capOne := da.Capability(1)

		device := da.BaseDevice{
			DeviceCapabilities: []da.Capability{capOne},
			DeviceGateway:      &mgwOne,
		}
		mgm.On("Device", "one").Return(device, true)

		mda := MockDeviceAction{}
		defer mda.AssertExpectations(t)

		bodyText := "{}"

		controller := deviceController{gatewayMapper: &mgm, deviceAction: mda.doAction, stack: layers.NoLayersStack{}}

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
		mgm := gateway.MockMapper{}
		defer mgm.AssertExpectations(t)

		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		mbc := mocks.BasicCapability{}
		defer mbc.AssertExpectations(t)
		mbc.On("Name").Return("name")

		capOne := da.Capability(1)

		mgwOne.On("Capability", capOne).Return(&mbc)

		device := da.BaseDevice{
			DeviceCapabilities: []da.Capability{capOne},
			DeviceGateway:      &mgwOne,
		}
		mgm.On("Device", "one").Return(device, true)

		mda := MockDeviceAction{}
		defer mda.AssertExpectations(t)

		bodyText := "{}"

		mda.On("doAction", mock.Anything, device, &mbc, "action", []byte(bodyText)).Return(struct{}{}, nil)

		controller := deviceController{gatewayMapper: &mgm, deviceAction: mda.doAction, stack: layers.PassThruStack{}}

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

	t.Run("returns a 200 with the body of the action, with custom layer and retention set", func(t *testing.T) {
		mgm := &gateway.MockMapper{}
		defer mgm.AssertExpectations(t)

		mgwOne := &mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		mbc := &mocks.BasicCapability{}
		defer mbc.AssertExpectations(t)
		mbc.On("Name").Return("name")

		capOne := da.Capability(1)

		device := da.BaseDevice{
			DeviceCapabilities: []da.Capability{capOne},
			DeviceGateway:      mgwOne,
		}
		mgm.On("Device", "one").Return(device, true)

		mda := &MockDeviceAction{}
		defer mda.AssertExpectations(t)

		bodyText := "{}"

		mda.On("doAction", mock.Anything, device, mbc, "action", []byte(bodyText)).Return(struct{}{}, nil)

		mol := &layers.MockOutputLayer{}
		defer mol.AssertExpectations(t)

		mos := &layers.MockOutputStack{}
		defer mos.AssertExpectations(t)

		mos.On("Lookup", "test").Return(mol)
		mol.On("Capability", layers.Maintain, capOne, device).Return(mbc)

		controller := deviceController{gatewayMapper: mgm, deviceAction: mda.doAction, stack: mos}

		body := strings.NewReader(bodyText)

		req, err := http.NewRequest("POST", "/devices/one/capabilities/name/action?layer=test&retention=maintain", body)
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
		mockCapability := &capabilitymocks.DeviceDiscovery{}
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
		mockCapability := &capabilitymocks.DeviceDiscovery{}
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
		mockCapability := &capabilitymocks.EnumerateDevice{}
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
		mockCapability := &capabilitymocks.OnOff{}
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
		mockCapability := &capabilitymocks.OnOff{}
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
		mockCapability := &capabilitymocks.AlarmWarningDevice{}
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
		mockCapability := &capabilitymocks.AlarmWarningDevice{}
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
		mockCapability := &capabilitymocks.AlarmWarningDevice{}
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
		mockCapability := &capabilitymocks.Level{}
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
		mockCapability := &capabilitymocks.Color{}
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
		mockCapability := &capabilitymocks.Color{}
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
		mockCapability := &capabilitymocks.Color{}
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
		mockCapability := &capabilitymocks.Color{}
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
		mockCapability := &capabilitymocks.DeviceRemoval{}
		defer mockCapability.AssertExpectations(t)

		device := da.BaseDevice{}
		mockCapability.On("Remove", mock.Anything, device).Return(nil)

		action := "Remove"

		expectedResult := struct{}{}

		actualResult, err := doDeviceCapabilityAction(context.Background(), device, mockCapability, action, nil)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, actualResult)
	})
}