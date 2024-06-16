package v1

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/controller/interface/converters/exporter"
	"github.com/shimmeringbee/controller/state"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_gatewayController_listGateways(t *testing.T) {
	t.Run("returns a list of gateways", func(t *testing.T) {
		mgm := state.MockMux{}
		defer mgm.AssertExpectations(t)

		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		mgwTwo := mocks.Gateway{}
		mgwTwo.On("EnsureGatewaysAreNotEqual").Maybe()
		defer mgwTwo.AssertExpectations(t)

		mgm.On("Gateways").Return(map[string]da.Gateway{
			"one": &mgwOne,
			"two": &mgwTwo,
		})

		mdc := exporter.MockGatewayExporter{}
		defer mdc.AssertExpectations(t)
		mdc.On("ConvertDAGatewayToGateway", &mgwOne).Return(exporter.ExportedGateway{
			Capabilities: []string{"capOne"},
			SelfDevice:   "one",
		})
		mdc.On("ConvertDAGatewayToGateway", &mgwTwo).Return(exporter.ExportedGateway{
			Capabilities: []string{"capTwo"},
			SelfDevice:   "two",
		})

		controller := gatewayController{gatewayMapper: &mgm, gatewayConverter: mdc.ConvertDAGatewayToGateway}

		expectedGateways := map[string]exporter.ExportedGateway{
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

		req, err := http.NewRequest("GET", "/gateways", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/gateways", controller.listGateways)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		actualData := []byte(rr.Body.String())
		actualGateways := map[string]exporter.ExportedGateway{}

		err = json.Unmarshal(actualData, &actualGateways)
		assert.NoError(t, err)

		assert.Equal(t, expectedGateways, actualGateways)
	})
}

func Test_gatewayController_getGateway(t *testing.T) {
	t.Run("returns a 404 if ExportedGateway is not present", func(t *testing.T) {
		mgm := state.MockMux{}
		defer mgm.AssertExpectations(t)

		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		mgm.On("Gateways").Return(map[string]da.Gateway{})

		controller := gatewayController{gatewayMapper: &mgm}

		req, err := http.NewRequest("GET", "/gateways/one", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/gateways/{identifier}", controller.getGateway)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns a ExportedGateway if present", func(t *testing.T) {
		mgm := state.MockMux{}
		defer mgm.AssertExpectations(t)

		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		mgm.On("Gateways").Return(map[string]da.Gateway{
			"one": &mgwOne,
		})

		mdc := exporter.MockGatewayExporter{}
		defer mdc.AssertExpectations(t)
		mdc.On("ConvertDAGatewayToGateway", &mgwOne).Return(exporter.ExportedGateway{
			Capabilities: []string{"capOne"},
			SelfDevice:   "one",
		})

		controller := gatewayController{gatewayMapper: &mgm, gatewayConverter: mdc.ConvertDAGatewayToGateway}

		expectedGateways := exporter.ExportedGateway{
			Identifier:   "one",
			Capabilities: []string{"capOne"},
			SelfDevice:   "one",
		}

		req, err := http.NewRequest("GET", "/gateways/one", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/gateways/{identifier}", controller.getGateway)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		actualData := []byte(rr.Body.String())
		actualGateways := exporter.ExportedGateway{}

		err = json.Unmarshal(actualData, &actualGateways)
		assert.NoError(t, err)

		assert.Equal(t, expectedGateways, actualGateways)
	})
}

func Test_gatewayController_listDevicesOnGateway(t *testing.T) {
	t.Run("returns 404, not found when ExportedGateway does not exist", func(t *testing.T) {
		mgm := state.MockMux{}
		defer mgm.AssertExpectations(t)
		mgm.On("Gateways").Return(map[string]da.Gateway{})

		controller := gatewayController{gatewayMapper: &mgm}

		req, err := http.NewRequest("GET", "/gateways/non-existent/devices", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/gateways/{identifier}/devices", controller.listDevicesOnGateway)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns list of devices found on ExportedGateway", func(t *testing.T) {
		mgm := state.MockMux{}
		defer mgm.AssertExpectations(t)

		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		mgm.On("Gateways").Return(map[string]da.Gateway{
			"one": &mgwOne,
		})

		mdev := &mocks.MockDevice{}
		defer mdev.AssertExpectations(t)

		expectedDeviceOne := exporter.ExportedDevice{
			Identifier:   "one-one",
			Capabilities: map[string]interface{}{"capOne": struct{}{}},
			Gateway:      "one",
		}

		mgwOne.On("Devices").Return([]da.Device{mdev})

		mdc := exporter.MockDeviceExporter{}
		defer mdc.AssertExpectations(t)
		mdc.On("ExportDevice", mock.Anything, mdev).Return(expectedDeviceOne)

		controller := gatewayController{gatewayMapper: &mgm, deviceConverter: &mdc}

		expectedDevices := map[string]exporter.ExportedDevice{
			"one-one": {
				Identifier:   "one-one",
				Capabilities: map[string]interface{}{"capOne": map[string]interface{}{}},
				Gateway:      "one",
			},
		}

		req, err := http.NewRequest("GET", "/gateways/one/devices", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/gateways/{identifier}/devices", controller.listDevicesOnGateway)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		actualData := []byte(rr.Body.String())
		actualDevices := map[string]exporter.ExportedDevice{}

		err = json.Unmarshal(actualData, &actualDevices)
		assert.NoError(t, err)

		assert.Equal(t, expectedDevices, actualDevices)
	})
}
