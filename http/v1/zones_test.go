package v1

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/controller/metadata"
	"github.com/shimmeringbee/da"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_zoneController_listZones(t *testing.T) {
	t.Run("returns a list of root zones", func(t *testing.T) {
		do := metadata.NewDeviceOrganiser()
		zoneOne := do.NewZone("one")
		zoneTwo := do.NewZone("two")
		_ = do.NewZone("three")

		err := do.MoveZone(zoneTwo.Identifier, zoneOne.Identifier)
		assert.NoError(t, err)

		controller := zoneController{deviceOrganiser: &do}

		req, err := http.NewRequest("GET", "/zones", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/zones", controller.listZones)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		expectedZones := []zone{
			{
				Identifier: 1,
				Name:       "one",
			},
			{
				Identifier: 3,
				Name:       "three",
			},
		}

		actualData := []byte(rr.Body.String())
		actualZones := []zone{}

		err = json.Unmarshal(actualData, &actualZones)
		assert.NoError(t, err)

		assert.Equal(t, expectedZones, actualZones)
	})

	t.Run("returns a list of root zones, with devices", func(t *testing.T) {
		do := metadata.NewDeviceOrganiser()
		do.AddDevice("devOne")
		do.AddDevice("devThree")
		zoneOne := do.NewZone("one")
		zoneTwo := do.NewZone("two")
		_ = do.NewZone("three")
		do.AddDeviceToZone("devOne", 1)
		do.AddDeviceToZone("devThree", 3)

		err := do.MoveZone(zoneTwo.Identifier, zoneOne.Identifier)
		assert.NoError(t, err)

		mgm := mockGatewayMapper{}
		defer mgm.AssertExpectations(t)

		daDevOne := da.BaseDevice{DeviceIdentifier: SimpleIdentifier{id: "devOne"}}
		daDevThree := da.BaseDevice{DeviceIdentifier: SimpleIdentifier{id: "devThree"}}

		mgm.On("Device", "devOne").Return(daDevOne, true)
		mgm.On("Device", "devThree").Return(daDevThree, true)

		mdc := mockDeviceConverter{}
		defer mdc.AssertExpectations(t)

		convDevOne := device{
			Identifier: "devOne",
		}

		convDevThree := device{
			Identifier: "devThree",
		}

		mdc.On("convertDevice", mock.Anything, daDevOne, mock.Anything).Return(convDevOne)
		mdc.On("convertDevice", mock.Anything, daDevThree, mock.Anything).Return(convDevThree)

		controller := zoneController{deviceOrganiser: &do, gatewayMapper: &mgm, deviceConverter: &mdc}

		req, err := http.NewRequest("GET", "/zones?include=devices", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/zones", controller.listZones)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		expectedZones := []zone{
			{
				Identifier: 1,
				Name:       "one",
				Devices: []device{
					{
						Identifier: "devOne",
					},
				},
			},
			{
				Identifier: 3,
				Name:       "three",
				Devices: []device{
					{
						Identifier: "devThree",
					},
				},
			},
		}

		actualData := []byte(rr.Body.String())
		actualZones := []zone{}

		err = json.Unmarshal(actualData, &actualZones)
		assert.NoError(t, err)

		assert.Equal(t, expectedZones, actualZones)
	})

	t.Run("returns a list of root zones, with sub zones", func(t *testing.T) {
		do := metadata.NewDeviceOrganiser()
		zoneOne := do.NewZone("one")
		zoneTwo := do.NewZone("two")
		_ = do.NewZone("three")

		err := do.MoveZone(zoneTwo.Identifier, zoneOne.Identifier)
		assert.NoError(t, err)

		controller := zoneController{deviceOrganiser: &do}

		req, err := http.NewRequest("GET", "/zones?include=subzones", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/zones", controller.listZones)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		expectedZones := []zone{
			{
				Identifier: 1,
				Name:       "one",
				SubZones: []zone{
					{
						Identifier: 2,
						Name:       "two",
					},
				},
			},
			{
				Identifier: 3,
				Name:       "three",
			},
		}

		actualData := []byte(rr.Body.String())
		actualZones := []zone{}

		err = json.Unmarshal(actualData, &actualZones)
		assert.NoError(t, err)

		assert.Equal(t, expectedZones, actualZones)
	})
}

func Test_zoneController_getZone(t *testing.T) {
	t.Run("returns an individual zone", func(t *testing.T) {
		do := metadata.NewDeviceOrganiser()
		zoneOne := do.NewZone("one")
		zoneTwo := do.NewZone("two")

		err := do.MoveZone(zoneTwo.Identifier, zoneOne.Identifier)
		assert.NoError(t, err)

		controller := zoneController{deviceOrganiser: &do}

		req, err := http.NewRequest("GET", "/zones/2", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/zones/{identifier}", controller.getZone)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		expectedZone := zone{
			Identifier: 2,
			Name:       "two",
			SubZones:   nil,
		}

		actualData := []byte(rr.Body.String())
		actualZone := zone{}

		err = json.Unmarshal(actualData, &actualZone)
		assert.NoError(t, err)

		assert.Equal(t, expectedZone, actualZone)
	})

	t.Run("returns an individual zone, with devices", func(t *testing.T) {
		do := metadata.NewDeviceOrganiser()
		zoneOne := do.NewZone("one")
		zoneTwo := do.NewZone("two")
		do.AddDevice("devTwo")
		do.AddDeviceToZone("devTwo", 2)

		err := do.MoveZone(zoneTwo.Identifier, zoneOne.Identifier)
		assert.NoError(t, err)

		mgm := mockGatewayMapper{}
		defer mgm.AssertExpectations(t)

		daDevTwo := da.BaseDevice{DeviceIdentifier: SimpleIdentifier{id: "devTwo"}}

		mgm.On("Device", "devTwo").Return(daDevTwo, true)

		mdc := mockDeviceConverter{}
		defer mdc.AssertExpectations(t)

		convDevTwo := device{
			Identifier: "devTwo",
		}

		mdc.On("convertDevice", mock.Anything, daDevTwo).Return(convDevTwo)

		controller := zoneController{deviceOrganiser: &do, gatewayMapper: &mgm, deviceConverter: &mdc}

		req, err := http.NewRequest("GET", "/zones/2?include=devices", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/zones/{identifier}", controller.getZone)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		expectedZone := zone{
			Identifier: 2,
			Name:       "two",
			SubZones:   nil,
			Devices: []device{
				{
					Identifier: "devTwo",
				},
			},
		}

		actualData := []byte(rr.Body.String())
		actualZone := zone{}

		err = json.Unmarshal(actualData, &actualZone)
		assert.NoError(t, err)

		assert.Equal(t, expectedZone, actualZone)
	})

	t.Run("returns an individual zone, with sub zones", func(t *testing.T) {
		do := metadata.NewDeviceOrganiser()
		zoneOne := do.NewZone("one")
		zoneTwo := do.NewZone("two")

		err := do.MoveZone(zoneTwo.Identifier, zoneOne.Identifier)
		assert.NoError(t, err)

		controller := zoneController{deviceOrganiser: &do}

		req, err := http.NewRequest("GET", "/zones/1?include=subzones", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/zones/{identifier}", controller.getZone)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		expectedZone := zone{
			Identifier: 1,
			Name:       "one",
			SubZones: []zone{
				{
					Identifier: 2,
					Name:       "two",
				},
			},
		}

		actualData := []byte(rr.Body.String())
		actualZone := zone{}

		err = json.Unmarshal(actualData, &actualZone)
		assert.NoError(t, err)

		assert.Equal(t, expectedZone, actualZone)
	})
}

func Test_zoneController_createZone(t *testing.T) {
	t.Run("creates an individual zone", func(t *testing.T) {
		do := metadata.NewDeviceOrganiser()

		controller := zoneController{deviceOrganiser: &do}

		req, err := http.NewRequest("POST", "/zones", strings.NewReader(`{"Name":"zone"}`))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/zones", controller.createZone)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		z, found := do.Zone(1)
		assert.True(t, found)
		assert.Equal(t, "zone", z.Name)

		expectedZone := zone{
			Identifier: 1,
			Name:       "zone",
			SubZones:   nil,
		}

		actualData := []byte(rr.Body.String())
		actualZone := zone{}

		err = json.Unmarshal(actualData, &actualZone)
		assert.NoError(t, err)

		assert.Equal(t, expectedZone, actualZone)
	})
}

func Test_zoneController_deleteZone(t *testing.T) {
	t.Run("deletes a zone", func(t *testing.T) {
		do := metadata.NewDeviceOrganiser()
		zoneOne := do.NewZone("one")

		controller := zoneController{deviceOrganiser: &do}

		req, err := http.NewRequest("DELETE", "/zones/1", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/zones/{identifier}", controller.deleteZone)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		_, found := do.Zone(zoneOne.Identifier)
		assert.False(t, found)
	})
}

func Test_zoneController_updateZone(t *testing.T) {
	t.Run("updates an individual zone", func(t *testing.T) {
		do := metadata.NewDeviceOrganiser()
		do.NewZone("old")

		controller := zoneController{deviceOrganiser: &do}

		req, err := http.NewRequest("PATCH", "/zones/1", strings.NewReader(`{"Name":"zone"}`))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/zones/{identifier}", controller.updateZone)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		z, found := do.Zone(1)
		assert.True(t, found)
		assert.Equal(t, "zone", z.Name)

		expectedZone := zone{
			Identifier: 1,
			Name:       "zone",
			SubZones:   nil,
		}

		actualData := []byte(rr.Body.String())
		actualZone := zone{}

		err = json.Unmarshal(actualData, &actualZone)
		assert.NoError(t, err)

		assert.Equal(t, expectedZone, actualZone)
	})
}

func Test_zoneController_addDeviceToZone(t *testing.T) {
	t.Run("add a device to a zone", func(t *testing.T) {
		do := metadata.NewDeviceOrganiser()
		do.NewZone("zone")
		do.AddDevice("id")

		controller := zoneController{deviceOrganiser: &do}

		req, err := http.NewRequest("PUT", "/zones/1/devices/id", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/zones/{identifier}/devices/{deviceIdentifier}", controller.addDeviceToZone)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)

		z, _ := do.Zone(1)
		assert.Contains(t, z.Devices, "id")
	})
}

func Test_zoneController_removeDeviceToZone(t *testing.T) {
	t.Run("remove a device from a zone", func(t *testing.T) {
		do := metadata.NewDeviceOrganiser()
		do.NewZone("zone")
		do.AddDevice("id")
		err := do.AddDeviceToZone("id", 1)
		assert.NoError(t, err)

		controller := zoneController{deviceOrganiser: &do}

		req, err := http.NewRequest("DELETE", "/zones/1/devices/id", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/zones/{identifier}/devices/{deviceIdentifier}", controller.removeDeviceToZone)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)

		z, _ := do.Zone(1)
		assert.NotContains(t, z.Devices, "id")
	})
}

func Test_zoneController_addSubzoneToZone(t *testing.T) {
	t.Run("add a device to a zone", func(t *testing.T) {
		do := metadata.NewDeviceOrganiser()
		do.NewZone("zone1")
		zTwo := do.NewZone("zone2")

		controller := zoneController{deviceOrganiser: &do}

		req, err := http.NewRequest("PUT", "/zones/1/subzones/2", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/zones/{identifier}/subzones/{subzoneIdentifier}", controller.addSubzoneToZone)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)

		z, _ := do.Zone(1)
		assert.Contains(t, z.SubZones, zTwo.Identifier)
	})
}

func Test_zoneController_removeSubzoneToZone(t *testing.T) {
	t.Run("remove a device from a zone", func(t *testing.T) {
		do := metadata.NewDeviceOrganiser()
		zOne := do.NewZone("zone1")
		zTwo := do.NewZone("zone2")

		err := do.MoveZone(zTwo.Identifier, zOne.Identifier)
		assert.NoError(t, err)

		controller := zoneController{deviceOrganiser: &do}

		req, err := http.NewRequest("DELETE", "/zones/1/subzones/2", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/zones/{identifier}/subzones/{subzoneIdentifier}", controller.removeSubzoneToZone)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)

		z, _ := do.Zone(1)
		assert.NotContains(t, z.SubZones, zTwo.Identifier)
	})
}
