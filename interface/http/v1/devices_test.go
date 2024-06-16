package v1

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/controller/interface/converters/exporter"
	"github.com/shimmeringbee/controller/interface/converters/invoker"
	"github.com/shimmeringbee/controller/layers"
	"github.com/shimmeringbee/controller/state"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
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
		mgm := state.MockMux{}
		defer mgm.AssertExpectations(t)

		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		mgwTwo := mocks.Gateway{}
		defer mgwTwo.AssertExpectations(t)

		mgm.On("Gateways").Return(map[string]da.Gateway{
			"one": &mgwOne,
			"two": &mgwTwo,
		})

		daDeviceOne := mocks.SimpleDevice{
			SGateway:      &mgwOne,
			SIdentifier:   SimpleIdentifier{id: "one-one"},
			SCapabilities: []da.Capability{},
		}

		expectedDeviceOne := exporter.ExportedDevice{
			Identifier:   "one-one",
			Capabilities: map[string]interface{}{"capOne": struct{}{}},
			Gateway:      "one",
		}

		mgwOne.On("Devices").Return([]da.Device{daDeviceOne})

		daDeviceTwo := mocks.SimpleDevice{
			SGateway:      &mgwTwo,
			SIdentifier:   SimpleIdentifier{id: "two-two"},
			SCapabilities: []da.Capability{},
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

		do := state.NewDeviceOrganiser(nil)

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
		mgm := state.MockMux{}
		defer mgm.AssertExpectations(t)

		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		daDeviceOne := mocks.SimpleDevice{
			SGateway:      &mgwOne,
			SIdentifier:   SimpleIdentifier{id: "one-one"},
			SCapabilities: []da.Capability{},
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
		mgm := state.MockMux{}
		defer mgm.AssertExpectations(t)

		mgm.On("Device", "one").Return(mocks.SimpleDevice{}, false)

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
		do := state.NewDeviceOrganiser(nil)
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

func Test_deviceController_useDeviceCapabilityAction(t *testing.T) {
	t.Run("returns a 404 if device is not present", func(t *testing.T) {
		mgm := state.MockMux{}
		defer mgm.AssertExpectations(t)

		mgm.On("Device", "one").Return(mocks.SimpleDevice{}, false)

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
		mgm := state.MockMux{}
		defer mgm.AssertExpectations(t)

		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		device := mocks.SimpleDevice{
			SCapabilities: []da.Capability{},
			SGateway:      &mgwOne,
		}

		mgm.On("Device", "one").Return(device, true)

		mda := invoker.MockDeviceInvoker{}
		defer mda.AssertExpectations(t)

		mda.On("InvokeDevice", mock.Anything, mock.Anything, mock.Anything, mock.Anything, device, "name", "action", []byte(nil)).Return(nil, invoker.CapabilityNotSupported)

		controller := deviceController{gatewayMapper: &mgm, deviceInvoker: mda.InvokeDevice, stack: layers.PassThruStack{}}

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
		mgm := state.MockMux{}
		defer mgm.AssertExpectations(t)

		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		capOne := da.Capability(1)

		device := mocks.SimpleDevice{
			SCapabilities: []da.Capability{capOne},
			SGateway:      &mgwOne,
		}
		mgm.On("Device", "one").Return(device, true)

		mda := invoker.MockDeviceInvoker{}
		defer mda.AssertExpectations(t)

		bodyText := "{}"

		mda.On("InvokeDevice", mock.Anything, mock.Anything, mock.Anything, mock.Anything, device, "name", "action", []byte(bodyText)).Return(nil, invoker.ActionNotSupported)

		controller := deviceController{gatewayMapper: &mgm, deviceInvoker: mda.InvokeDevice, stack: layers.PassThruStack{}}

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
		mgm := state.MockMux{}
		defer mgm.AssertExpectations(t)

		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		capOne := da.Capability(1)

		device := mocks.SimpleDevice{
			SCapabilities: []da.Capability{capOne},
			SGateway:      &mgwOne,
		}
		mgm.On("Device", "one").Return(device, true)

		mda := invoker.MockDeviceInvoker{}
		defer mda.AssertExpectations(t)

		bodyText := "{}"

		mda.On("InvokeDevice", mock.Anything, mock.Anything, mock.Anything, mock.Anything, device, "name", "action", []byte(bodyText)).Return([]byte{}, fmt.Errorf("unknown error"))

		controller := deviceController{gatewayMapper: &mgm, deviceInvoker: mda.InvokeDevice, stack: layers.PassThruStack{}}

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
		mgm := state.MockMux{}
		defer mgm.AssertExpectations(t)

		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		capOne := da.Capability(1)

		device := mocks.SimpleDevice{
			SCapabilities: []da.Capability{capOne},
			SGateway:      &mgwOne,
		}
		mgm.On("Device", "one").Return(device, true)

		mda := invoker.MockDeviceInvoker{}
		defer mda.AssertExpectations(t)

		bodyText := "{}"

		mda.On("InvokeDevice", mock.Anything, mock.Anything, mock.Anything, mock.Anything, device, "name", "action", []byte(bodyText)).Return([]byte{}, fmt.Errorf("%w: unknown error", invoker.ActionUserError))

		controller := deviceController{gatewayMapper: &mgm, deviceInvoker: mda.InvokeDevice, stack: layers.PassThruStack{}}

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
		mgm := state.MockMux{}
		defer mgm.AssertExpectations(t)

		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		capOne := da.Capability(1)

		device := mocks.SimpleDevice{
			SCapabilities: []da.Capability{capOne},
			SGateway:      &mgwOne,
		}
		mgm.On("Device", "one").Return(device, true)

		mda := invoker.MockDeviceInvoker{}
		defer mda.AssertExpectations(t)

		bodyText := "{}"

		mda.On("InvokeDevice", mock.Anything, mock.Anything, mock.Anything, mock.Anything, device, "name", "action", []byte(bodyText)).Return(struct{}{}, nil)

		controller := deviceController{gatewayMapper: &mgm, deviceInvoker: mda.InvokeDevice, stack: layers.PassThruStack{}}

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
		mgm := &state.MockMux{}
		defer mgm.AssertExpectations(t)

		mgwOne := &mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		capOne := da.Capability(1)

		device := mocks.SimpleDevice{
			SCapabilities: []da.Capability{capOne},
			SGateway:      mgwOne,
		}
		mgm.On("Device", "one").Return(device, true)

		mda := &invoker.MockDeviceInvoker{}
		defer mda.AssertExpectations(t)

		bodyText := "{}"

		mos := &layers.MockOutputStack{}
		defer mos.AssertExpectations(t)

		mda.On("InvokeDevice", mock.Anything, mock.Anything, "test", layers.Maintain, device, "name", "action", []byte(bodyText)).Return(struct{}{}, nil)

		controller := deviceController{gatewayMapper: mgm, deviceInvoker: mda.InvokeDevice, stack: mos}

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
