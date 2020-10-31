package v1

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/da"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
