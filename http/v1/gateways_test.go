package v1

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/da"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_gatewayController_listGateways(t *testing.T) {
	t.Run("returns a list of gateways", func(t *testing.T) {
		mgm := mockGatewayMapper{}
		defer mgm.AssertExpectations(t)

		mgwOne := mockGateway{}
		defer mgwOne.AssertExpectations(t)

		mgwTwo := mockGateway{}
		mgwTwo.On("EnsureGatewaysAreNotEqual").Maybe()
		defer mgwTwo.AssertExpectations(t)

		mgm.On("Gateways").Return(map[string]da.Gateway{
			"one": &mgwOne,
			"two": &mgwTwo,
		})

		mdc := mockGatewayConverter{}
		defer mdc.AssertExpectations(t)
		mdc.On("convertDAGatewayToGateway", &mgwOne).Return(gateway{
			Capabilities: []string{"capOne"},
			SelfDevice:   "one",
		})
		mdc.On("convertDAGatewayToGateway", &mgwTwo).Return(gateway{
			Capabilities: []string{"capTwo"},
			SelfDevice:   "two",
		})

		controller := gatewayController{gatewayMapper: &mgm, gatewayConverter: mdc.convertDAGatewayToGateway}

		expectedGateways := map[string]gateway{
			"one": {
				Identifier:   "one",
				Capabilities: []string{"capOne"},
				SelfDevice:   "one",
			},
			"two": {
				Identifier:   "two",
				Capabilities: []string{"capTwo"},
				SelfDevice:   "two",
			},
		}

		req, err := http.NewRequest("GET", "/api/v1/gateways", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/api/v1/gateways", controller.listGateways)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		actualData := []byte(rr.Body.String())
		actualGateways := map[string]gateway{}

		err = json.Unmarshal(actualData, &actualGateways)
		assert.NoError(t, err)

		assert.Equal(t, expectedGateways, actualGateways)
	})
}

func Test_gatewayController_getGateway(t *testing.T) {
	t.Run("returns a gateway if present", func(t *testing.T) {
		mgm := mockGatewayMapper{}
		defer mgm.AssertExpectations(t)

		mgwOne := mockGateway{}
		defer mgwOne.AssertExpectations(t)

		mgwTwo := mockGateway{}
		mgwTwo.On("EnsureGatewaysAreNotEqual").Maybe()
		defer mgwTwo.AssertExpectations(t)

		mgm.On("Gateways").Return(map[string]da.Gateway{
			"one": &mgwOne,
		})

		mdc := mockGatewayConverter{}
		defer mdc.AssertExpectations(t)
		mdc.On("convertDAGatewayToGateway", &mgwOne).Return(gateway{
			Capabilities: []string{"capOne"},
			SelfDevice:   "one",
		})

		controller := gatewayController{gatewayMapper: &mgm, gatewayConverter: mdc.convertDAGatewayToGateway}

		expectedGateways := gateway{
			Identifier:   "one",
			Capabilities: []string{"capOne"},
			SelfDevice:   "one",
		}

		req, err := http.NewRequest("GET", "/api/v1/gateways/one", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/api/v1/gateways/{identifier}", controller.getGateway)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		actualData := []byte(rr.Body.String())
		actualGateways := gateway{}

		err = json.Unmarshal(actualData, &actualGateways)
		assert.NoError(t, err)

		assert.Equal(t, expectedGateways, actualGateways)
	})
}

func Test_gatewayController_listDevicesOnGateway(t *testing.T) {
	t.Run("returns 404, not found when gateway does not exist", func(t *testing.T) {
		mgm := mockGatewayMapper{}
		defer mgm.AssertExpectations(t)
		mgm.On("Gateways").Return(map[string]da.Gateway{})

		controller := gatewayController{gatewayMapper: &mgm}

		req, err := http.NewRequest("GET", "/api/v1/gateways/non-existent/devices", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/api/v1/gateways/{identifier}/devices", controller.listDevicesOnGateway)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns list of devices found on gateway", func(t *testing.T) {
		mgm := mockGatewayMapper{}
		defer mgm.AssertExpectations(t)

		mgwOne := mockGateway{}
		defer mgwOne.AssertExpectations(t)

		mgm.On("Gateways").Return(map[string]da.Gateway{
			"one": &mgwOne,
		})

		daDeviceOne := da.BaseDevice{
			DeviceGateway:      &mgwOne,
			DeviceIdentifier:   SimpleIdentifier{id: "one-one"},
			DeviceCapabilities: []da.Capability{},
		}

		expectedDeviceOne := device{
			Identifier:   "one-one",
			Capabilities: map[string]interface{}{"capOne": struct{}{}},
		}

		mgwOne.On("Devices").Return([]da.Device{daDeviceOne})

		mdc := mockDeviceConverter{}
		defer mdc.AssertExpectations(t)
		mdc.On("convertDADeviceToDevice", daDeviceOne).Return(expectedDeviceOne)

		controller := gatewayController{gatewayMapper: &mgm, deviceConverter: mdc.convertDADeviceToDevice}

		expectedDevices := map[string]device{
			"one-one": {
				Identifier:   "one-one",
				Capabilities: map[string]interface{}{"capOne": map[string]interface{}{}},
				Gateway:      "one",
			},
		}

		req, err := http.NewRequest("GET", "/api/v1/gateways/one/devices", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/api/v1/gateways/{identifier}/devices", controller.listDevicesOnGateway)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		actualData := []byte(rr.Body.String())
		actualDevices := map[string]device{}

		err = json.Unmarshal(actualData, &actualDevices)
		assert.NoError(t, err)

		assert.Equal(t, expectedDevices, actualDevices)
	})
}
