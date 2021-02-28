package v1

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/controller/gateway"
	"github.com/shimmeringbee/controller/interface/exporter"
	"github.com/shimmeringbee/controller/metadata"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type SimpleIdentifier struct {
	id string
}

func (s SimpleIdentifier) String() string {
	return s.id
}

func Test_deviceController_listDevices(t *testing.T) {
	t.Run("returns a list of devices across multiple gateways", func(t *testing.T) {
		mgm := gateway.MockMapper{}
		defer mgm.AssertExpectations(t)

		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		mgwTwo := mocks.Gateway{}
		defer mgwTwo.AssertExpectations(t)

		mgm.On("Gateways").Return(map[string]da.Gateway{
			"one": &mgwOne,
			"two": &mgwTwo,
		})

		daDeviceOne := da.BaseDevice{
			DeviceGateway:      &mgwOne,
			DeviceIdentifier:   SimpleIdentifier{id: "one-one"},
			DeviceCapabilities: []da.Capability{},
		}

		expectedDeviceOne := exporter.ExportedDevice{
			Identifier:   "one-one",
			Capabilities: map[string]interface{}{"capOne": struct{}{}},
			Gateway:      "one",
		}

		mgwOne.On("Devices").Return([]da.Device{daDeviceOne})

		daDeviceTwo := da.BaseDevice{
			DeviceGateway:      &mgwTwo,
			DeviceIdentifier:   SimpleIdentifier{id: "two-two"},
			DeviceCapabilities: []da.Capability{},
		}

		expectedDeviceTwo := exporter.ExportedDevice{
			Identifier:   "two-two",
			Capabilities: map[string]interface{}{"capTwo": struct{}{}},
			Gateway:      "two",
		}

		mgwTwo.On("Devices").Return([]da.Device{daDeviceTwo})

		mdc := exporter.MockDeviceExporter{}
		defer mdc.AssertExpectations(t)
		mdc.On("ExportDevice", mock.Anything, daDeviceOne).Return(expectedDeviceOne)
		mdc.On("ExportDevice", mock.Anything, daDeviceTwo).Return(expectedDeviceTwo)

		do := metadata.NewDeviceOrganiser()

		controller := deviceController{gatewayMapper: &mgm, deviceExporter: &mdc, deviceOrganiser: &do}

		expectedDevices := map[string]exporter.ExportedDevice{
			"one-one": {
				Identifier:   "one-one",
				Capabilities: map[string]interface{}{"capOne": map[string]interface{}{}},
				Gateway:      "one",
			},
			"two-two": {
				Identifier:   "two-two",
				Capabilities: map[string]interface{}{"capTwo": map[string]interface{}{}},
				Gateway:      "two",
			},
		}

		req, err := http.NewRequest("GET", "/devices", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/devices", controller.listDevices)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		actualData := []byte(rr.Body.String())
		actualDevices := map[string]exporter.ExportedDevice{}

		err = json.Unmarshal(actualData, &actualDevices)
		assert.NoError(t, err)

		assert.Equal(t, expectedDevices, actualDevices)
	})
}

func Test_deviceController_getDevice(t *testing.T) {
	t.Run("returns a device if present", func(t *testing.T) {
		mgm := gateway.MockMapper{}
		defer mgm.AssertExpectations(t)

		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		daDeviceOne := da.BaseDevice{
			DeviceGateway:      &mgwOne,
			DeviceIdentifier:   SimpleIdentifier{id: "one-one"},
			DeviceCapabilities: []da.Capability{},
		}

		mgm.On("Device", "one").Return(daDeviceOne, true)

		expectedDeviceOne := exporter.ExportedDevice{
			Identifier:   "one-one",
			Capabilities: map[string]interface{}{"capOne": struct{}{}},
			Gateway:      "one",
		}

		mdc := exporter.MockDeviceExporter{}
		defer mdc.AssertExpectations(t)
		mdc.On("ExportDevice", mock.Anything, daDeviceOne, mock.Anything).Return(expectedDeviceOne)

		controller := deviceController{gatewayMapper: &mgm, deviceExporter: &mdc}

		expectedDevice := exporter.ExportedDevice{
			Identifier:   "one-one",
			Capabilities: map[string]interface{}{"capOne": map[string]interface{}{}},
			Gateway:      "one",
		}

		req, err := http.NewRequest("GET", "/devices/one", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/devices/{identifier}", controller.getDevice)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		actualData := []byte(rr.Body.String())
		actualDevice := exporter.ExportedDevice{}

		err = json.Unmarshal(actualData, &actualDevice)
		assert.NoError(t, err)

		assert.Equal(t, expectedDevice, actualDevice)
	})

	t.Run("returns a 404 if device is not present", func(t *testing.T) {
		mgm := gateway.MockMapper{}
		defer mgm.AssertExpectations(t)

		mgm.On("Device", "one").Return(da.BaseDevice{}, false)

		controller := deviceController{gatewayMapper: &mgm}

		req, err := http.NewRequest("GET", "/devices/one", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/devices/{identifier}", controller.getDevice)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

func Test_deviceController_updateDevice(t *testing.T) {
	t.Run("updates an individual ExportedDevice with name", func(t *testing.T) {
		do := metadata.NewDeviceOrganiser()
		do.AddDevice("one")

		controller := deviceController{deviceOrganiser: &do}

		req, err := http.NewRequest("PATCH", "/devices/one", strings.NewReader(`{"Name":"ExportedDevice"}`))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/devices/{identifier}", controller.updateDevice)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)

		d, found := do.Device("one")
		assert.True(t, found)
		assert.Equal(t, "ExportedDevice", d.Name)
	})
}
