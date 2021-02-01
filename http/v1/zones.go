package v1

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/controller/metadata"
	"io/ioutil"
	"net/http"
	"strconv"
)

type zoneController struct {
	deviceOrganiser *metadata.DeviceOrganiser
	gatewayMapper   GatewayMapper
	deviceConverter deviceConverter
}

func includesString(haystack []string, needle string) bool {
	for _, straw := range haystack {
		if needle == straw {
			return true
		}
	}

	return false
}

func (z *zoneController) listZones(w http.ResponseWriter, r *http.Request) {
	includes, _ := r.URL.Query()["include"]
	devices := includesString(includes, "devices")
	subzones := includesString(includes, "subzones")

	returnZones := []zone{}

	for _, nZ := range z.deviceOrganiser.RootZones() {
		returnZones = append(returnZones, z.enumerateZone(nZ, devices, subzones))
	}

	data, err := json.Marshal(returnZones)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Add("content-type", "application/json")
	w.Write(data)
}

func (z *zoneController) enumerateZone(nZ metadata.Zone, includeDevices bool, includeSubzones bool) zone {
	var subZones []zone
	var devices []device

	if includeSubzones {
		subZones = z.enumerateZones(nZ.SubZones, includeDevices)
	}

	if includeDevices {
		for _, id := range nZ.Devices {
			if daDevice, found := z.gatewayMapper.Device(id); found {
				dev := z.deviceConverter.convertDevice(context.Background(), daDevice)
				devices = append(devices, dev)
			}
		}
	}

	return zone{
		Identifier: nZ.Identifier,
		Name:       nZ.Name,
		SubZones:   subZones,
		Devices:    devices,
	}
}

func (z *zoneController) enumerateZones(zoneIds []int, devices bool) []zone {
	var zones []zone

	for _, zoneId := range zoneIds {
		if nZ, found := z.deviceOrganiser.Zone(zoneId); found {
			zones = append(zones, z.enumerateZone(nZ, devices, true))
		}
	}

	return zones
}

func (z *zoneController) getZone(w http.ResponseWriter, r *http.Request) {
	includes, _ := r.URL.Query()["include"]
	devices := includesString(includes, "devices")
	subzones := includesString(includes, "subzones")

	params := mux.Vars(r)

	stringId, ok := params["identifier"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	id, err := strconv.Atoi(stringId)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	nZ, found := z.deviceOrganiser.Zone(id)
	if !found {
		http.NotFound(w, r)
		return
	}

	convertedZone := z.enumerateZone(nZ, devices, subzones)

	data, err := json.Marshal(convertedZone)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Add("content-type", "application/json")
	w.Write(data)
}

type createZoneRequest struct {
	Name string
}

func (z *zoneController) createZone(w http.ResponseWriter, r *http.Request) {
	request := createZoneRequest{}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(data, &request)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	nZ := z.deviceOrganiser.NewZone(request.Name)
	convertedZone := z.enumerateZone(nZ, false, false)

	data, err = json.Marshal(convertedZone)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Add("content-type", "application/json")
	w.Write(data)
}

func (z *zoneController) deleteZone(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	stringId, ok := params["identifier"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	id, err := strconv.Atoi(stringId)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	err = z.deviceOrganiser.DeleteZone(id)
	switch {
	case errors.Is(err, metadata.ErrNotFound):
		http.NotFound(w, r)
	case errors.Is(err, metadata.ErrHasDevices):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, metadata.ErrOrphanZone):
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	http.Error(w, http.StatusText(http.StatusNoContent), http.StatusNoContent)
}

type updateZoneRequest struct {
	Name          *string
	ReorderBefore *int
	ReorderAfter  *int
}

func (z *zoneController) updateZone(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	stringId, ok := params["identifier"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	id, err := strconv.Atoi(stringId)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	request := updateZoneRequest{}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(data, &request)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	nZ, found := z.deviceOrganiser.Zone(id)
	if !found {
		http.NotFound(w, r)
		return
	}

	if request.Name != nil {
		if err := z.deviceOrganiser.NameZone(nZ.Identifier, *request.Name); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	if request.ReorderAfter != nil {
		if err := z.deviceOrganiser.ReorderZoneAfter(nZ.Identifier, *request.ReorderAfter); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}

	if request.ReorderBefore != nil {
		if err := z.deviceOrganiser.ReorderZoneBefore(nZ.Identifier, *request.ReorderBefore); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}

	nZ, _ = z.deviceOrganiser.Zone(id)

	convertedZone := z.enumerateZone(nZ, false, false)

	data, err = json.Marshal(convertedZone)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Add("content-type", "application/json")
	w.Write(data)
}

func (z *zoneController) addDeviceToZone(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	stringZoneId, ok := params["identifier"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	deviceId, ok := params["deviceIdentifier"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	zoneId, err := strconv.Atoi(stringZoneId)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := z.deviceOrganiser.AddDeviceToZone(deviceId, zoneId); err != nil {
		if errors.Is(err, metadata.ErrNotFound) {
			http.NotFound(w, r)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		return
	}

	http.Error(w, http.StatusText(http.StatusNoContent), http.StatusNoContent)
}

func (z *zoneController) removeDeviceToZone(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	stringZoneId, ok := params["identifier"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	deviceId, ok := params["deviceIdentifier"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	zoneId, err := strconv.Atoi(stringZoneId)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := z.deviceOrganiser.RemoveDeviceFromZone(deviceId, zoneId); err != nil {
		if errors.Is(err, metadata.ErrNotFound) {
			http.NotFound(w, r)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		return
	}

	http.Error(w, http.StatusText(http.StatusNoContent), http.StatusNoContent)
}

func (z *zoneController) addSubzoneToZone(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	stringZoneId, ok := params["identifier"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	stringSubzoneId, ok := params["subzoneIdentifier"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	zoneId, err := strconv.Atoi(stringZoneId)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	subzoneId, err := strconv.Atoi(stringSubzoneId)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := z.deviceOrganiser.MoveZone(subzoneId, zoneId); err != nil {
		if errors.Is(err, metadata.ErrNotFound) {
			http.NotFound(w, r)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		return
	}

	http.Error(w, http.StatusText(http.StatusNoContent), http.StatusNoContent)
}

func (z *zoneController) removeSubzoneToZone(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	stringZoneId, ok := params["identifier"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	stringSubzoneId, ok := params["subzoneIdentifier"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	zoneId, err := strconv.Atoi(stringZoneId)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	subzoneId, err := strconv.Atoi(stringSubzoneId)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	zo, found := z.deviceOrganiser.Zone(subzoneId)
	if !found || zo.ParentZone != zoneId {
		http.NotFound(w, r)
	}

	if err := z.deviceOrganiser.MoveZone(subzoneId, metadata.RootZoneId); err != nil {
		if errors.Is(err, metadata.ErrNotFound) {
			http.NotFound(w, r)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		return
	}

	http.Error(w, http.StatusText(http.StatusNoContent), http.StatusNoContent)
}
