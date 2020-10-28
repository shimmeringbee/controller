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

type SimpleIdentifier struct {
	id string
}

func (s SimpleIdentifier) String() string {
	return s.id
}

func Test_deviceController_listDevices(t *testing.T) {
	t.Run("returns a list of devices across multiple gateways", func(t *testing.T) {
		mgm := mockGatewayMapper{}
		defer mgm.AssertExpectations(t)

		mgwOne := mockGateway{}
		defer mgwOne.AssertExpectations(t)

		mgwTwo := mockGateway{}
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

		expectedDeviceOne := device{
			Identifier:   "one-one",
			Capabilities: []string{"capOne"},
		}

		mgwOne.On("Devices").Return([]da.Device{daDeviceOne})

		daDeviceTwo := da.BaseDevice{
			DeviceGateway:      &mgwTwo,
			DeviceIdentifier:   SimpleIdentifier{id: "two-two"},
			DeviceCapabilities: []da.Capability{},
		}

		expectedDeviceTwo := device{
			Identifier:   "two-two",
			Capabilities: []string{"capTwo"},
		}

		mgwTwo.On("Devices").Return([]da.Device{daDeviceTwo})

		mdc := mockDeviceConverter{}
		defer mdc.AssertExpectations(t)
		mdc.On("convertDADeviceToDevice", daDeviceOne).Return(expectedDeviceOne)
		mdc.On("convertDADeviceToDevice", daDeviceTwo).Return(expectedDeviceTwo)

		controller := deviceController{gatewayMapper: &mgm, deviceConverter: mdc.convertDADeviceToDevice}

		expectedDevices := map[string]device{
			"one-one": {
				Identifier:   "one-one",
				Capabilities: []string{"capOne"},
				Gateway:      "one",
			},
			"two-two": {
				Identifier:   "two-two",
				Capabilities: []string{"capTwo"},
				Gateway:      "two",
			},
		}

		req, err := http.NewRequest("GET", "/api/v1/devices", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/api/v1/devices", controller.listDevices)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		actualData := []byte(rr.Body.String())
		actualDevices := map[string]device{}

		err = json.Unmarshal(actualData, &actualDevices)
		assert.NoError(t, err)

		assert.Equal(t, expectedDevices, actualDevices)
	})
}
