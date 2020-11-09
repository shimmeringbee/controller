package main

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestDeviceOrganiser_Zones(t *testing.T) {
	t.Run("NewZone generates a new zone creates it at the root with an incrementing id", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}}

		zoneOne := do.NewZone("one")
		zoneTwo := do.NewZone("two")

		assert.Equal(t, "one", zoneOne.Name)
		assert.Equal(t, 1, zoneOne.Identifier)

		assert.Equal(t, "two", zoneTwo.Name)
		assert.Equal(t, 2, zoneTwo.Identifier)
	})

	t.Run("RootZones returns the constructed root", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}}

		zoneOne := do.NewZone("one")

		roots := do.RootZones()
		assert.Len(t, roots, 1)
		assert.Contains(t, roots, zoneOne)
	})

	t.Run("GetZone returns a zone by id", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}}

		zoneOne := do.NewZone("one")

		foundZone, found := do.Zone(zoneOne.Identifier)
		assert.True(t, found)
		assert.Equal(t, zoneOne, foundZone)
	})

	t.Run("GetZone returns false if it can't find the zone by id", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}}

		_, found := do.Zone(1)
		assert.False(t, found)
	})

	t.Run("NameZone returns an error if the zone does not exist", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}}

		err := do.NameZone(1, "New Name")
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("NameZone updates a zones name", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}}

		zoneOne := do.NewZone("one")

		newName := "New Name"
		err := do.NameZone(zoneOne.Identifier, newName)
		assert.NoError(t, err)

		changedZone, _ := do.Zone(zoneOne.Identifier)
		assert.Equal(t, newName, changedZone.Name)
	})

	t.Run("MoveZone errors if the zone being moved does not exist", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}}

		err := do.MoveZone(1, -1)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("MoveZone errors if the parent zone does not exist", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}}

		zoneOne := do.NewZone("one")

		err := do.MoveZone(zoneOne.Identifier, -1)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("MoveZone errors if the moved zone and parent are equal", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}}

		zoneOne := do.NewZone("one")

		err := do.MoveZone(zoneOne.Identifier, zoneOne.Identifier)
		assert.True(t, errors.Is(err, ErrMoveSameZone))
	})

	t.Run("MoveZone succeeds in moving one root entry under another, removing the old root entry", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}}

		zoneOne := do.NewZone("one")
		zoneTwo := do.NewZone("two")

		err := do.MoveZone(zoneTwo.Identifier, zoneOne.Identifier)
		assert.NoError(t, err)

		roots := do.rootZones
		assert.Len(t, roots, 1)
		assert.Equal(t, roots[0], zoneOne.Identifier)

		afterOne, _ := do.Zone(zoneOne.Identifier)
		afterTwo, _ := do.Zone(zoneTwo.Identifier)

		assert.Contains(t, afterOne.SubZones, zoneTwo.Identifier)
		assert.Equal(t, afterOne.Identifier, afterTwo.ParentZone)
	})

	t.Run("MoveZone succeeds in moving a sub zone under another sub zone", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}}

		zoneOne := do.NewZone("one")
		zoneTwo := do.NewZone("two")
		zoneThree := do.NewZone("three")

		err := do.MoveZone(zoneTwo.Identifier, zoneOne.Identifier)
		assert.NoError(t, err)
		err = do.MoveZone(zoneThree.Identifier, zoneOne.Identifier)
		assert.NoError(t, err)

		err = do.MoveZone(zoneThree.Identifier, zoneTwo.Identifier)
		assert.NoError(t, err)

		checkOne, _ := do.Zone(zoneOne.Identifier)
		checkTwo, _ := do.Zone(zoneTwo.Identifier)
		checkThree, _ := do.Zone(zoneThree.Identifier)

		assert.Len(t, checkOne.SubZones, 1)
		assert.Len(t, checkTwo.SubZones, 1)
		assert.Contains(t, checkTwo.SubZones, checkThree.Identifier)
		assert.Equal(t, checkTwo.Identifier, checkThree.ParentZone)
	})

	t.Run("MoveZone errors if moving a zone to be under one of its sub zones", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}}

		zoneOne := do.NewZone("one")
		zoneTwo := do.NewZone("two")
		zoneThree := do.NewZone("three")

		err := do.MoveZone(zoneTwo.Identifier, zoneOne.Identifier)
		assert.NoError(t, err)
		err = do.MoveZone(zoneThree.Identifier, zoneTwo.Identifier)
		assert.NoError(t, err)

		err = do.MoveZone(zoneOne.Identifier, zoneThree.Identifier)
		assert.True(t, errors.Is(err, ErrCircularReference))
	})

	t.Run("MoveZone succeeds in moving a sub zone back to root", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}}

		zoneOne := do.NewZone("one")
		zoneTwo := do.NewZone("two")

		err := do.MoveZone(zoneTwo.Identifier, zoneOne.Identifier)
		assert.NoError(t, err)

		err = do.MoveZone(zoneTwo.Identifier, RootZoneId)
		assert.NoError(t, err)

		checkOne, _ := do.Zone(zoneOne.Identifier)
		assert.Len(t, checkOne.SubZones, 0)

		roots := do.rootZones
		assert.Len(t, roots, 2)
		assert.Contains(t, roots, zoneTwo.Identifier)

		assert.Equal(t, RootZoneId, zoneTwo.ParentZone)
	})

	t.Run("DeleteZone errors if zone can not be found", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}}

		err := do.DeleteZone(1)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("DeleteZone errors if zone has subzone found", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}}

		zoneOne := do.NewZone("one")
		zoneTwo := do.NewZone("two")

		err := do.MoveZone(zoneTwo.Identifier, zoneOne.Identifier)
		assert.NoError(t, err)

		err = do.DeleteZone(zoneOne.Identifier)
		assert.True(t, errors.Is(err, ErrOrphanZone))
	})

	t.Run("DeleteZone errors if zone has device", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}}

		zoneOne := do.NewZone("one")
		do.zones[zoneOne.Identifier].Devices = []string{"device"}

		err := do.DeleteZone(zoneOne.Identifier)
		assert.True(t, errors.Is(err, ErrHasDevices))
	})

	t.Run("DeleteZone succeeds deleting a subzone", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}}

		zoneOne := do.NewZone("one")
		zoneTwo := do.NewZone("two")

		err := do.MoveZone(zoneTwo.Identifier, zoneOne.Identifier)
		assert.NoError(t, err)

		err = do.DeleteZone(zoneTwo.Identifier)
		assert.NoError(t, err)

		checkOne, _ := do.Zone(zoneOne.Identifier)

		assert.NotContains(t, do.zones, zoneTwo.Identifier)
		assert.NotContains(t, checkOne.SubZones, zoneTwo.Identifier)
	})

	t.Run("DeleteZone succeeds deleting a root zone", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}}

		zoneOne := do.NewZone("one")

		err := do.DeleteZone(zoneOne.Identifier)
		assert.NoError(t, err)
		assert.NotContains(t, do.zones, zoneOne.Identifier)
		assert.NotContains(t, do.rootZones, zoneOne.Identifier)
	})
}

func TestDeviceOrganiser_Devices(t *testing.T) {
	t.Run("AddDevice adds a device", func(t *testing.T) {
		do := DeviceOrganiser{deviceLock: &sync.Mutex{}, devices: map[string]*DeviceMetadata{}}
		do.AddDevice("id")

		_, found := do.Device("id")
		assert.True(t, found)
	})

	t.Run("Device returns false if device is not present", func(t *testing.T) {
		do := DeviceOrganiser{deviceLock: &sync.Mutex{}, devices: map[string]*DeviceMetadata{}}
		_, found := do.Device("id")
		assert.False(t, found)
	})

	t.Run("Device returns true if device is present", func(t *testing.T) {
		do := DeviceOrganiser{deviceLock: &sync.Mutex{}, devices: map[string]*DeviceMetadata{}}
		do.AddDevice("id")
		_, found := do.Device("id")
		assert.True(t, found)
	})

	t.Run("NameDevice errors if device doesn't exist", func(t *testing.T) {
		do := DeviceOrganiser{deviceLock: &sync.Mutex{}, devices: map[string]*DeviceMetadata{}}

		err := do.NameDevice("id", "name")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("NameDevice sets a name on a device", func(t *testing.T) {
		do := DeviceOrganiser{deviceLock: &sync.Mutex{}, devices: map[string]*DeviceMetadata{}}
		do.AddDevice("id")

		err := do.NameDevice("id", "name")
		assert.NoError(t, err)

		dm, found := do.Device("id")
		assert.True(t, found)
		assert.Equal(t, "name", dm.Name)
	})

	t.Run("NameDevice errors if the device does not exist", func(t *testing.T) {
		do := DeviceOrganiser{deviceLock: &sync.Mutex{}, devices: map[string]*DeviceMetadata{}}

		err := do.NameDevice("id", "name")
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("AddDevice does not overwrite an existing device", func(t *testing.T) {
		do := DeviceOrganiser{deviceLock: &sync.Mutex{}, devices: map[string]*DeviceMetadata{}}
		do.AddDevice("id")

		err := do.NameDevice("id", "name")
		assert.NoError(t, err)

		do.AddDevice("id")

		dm, found := do.Device("id")
		assert.True(t, found)
		assert.Equal(t, "name", dm.Name)
	})

	t.Run("RemoveDevice removes an added device", func(t *testing.T) {
		do := DeviceOrganiser{deviceLock: &sync.Mutex{}, devices: map[string]*DeviceMetadata{}}
		do.AddDevice("id")

		do.RemoveDevice("id")
		_, found := do.Device("id")
		assert.False(t, found)
	})

	t.Run("AddDeviceToZone errors if the device can not be found", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}, deviceLock: &sync.Mutex{}, devices: map[string]*DeviceMetadata{}}

		err := do.AddDeviceToZone("id", 1)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("AddDeviceToZone errors if the zone can not be found", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}, deviceLock: &sync.Mutex{}, devices: map[string]*DeviceMetadata{}}

		do.AddDevice("id")

		err := do.AddDeviceToZone("id", 1)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("AddDeviceToZone adds the zone to the device and device to zone", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}, deviceLock: &sync.Mutex{}, devices: map[string]*DeviceMetadata{}}

		do.AddDevice("id")
		zone := do.NewZone("name")

		err := do.AddDeviceToZone("id", zone.Identifier)
		assert.NoError(t, err)

		checkDevice, _ := do.Device("id")
		checkZone, _ := do.Zone(zone.Identifier)

		assert.Contains(t, checkDevice.Zones, zone.Identifier)
		assert.Contains(t, checkZone.Devices, "id")
	})

	t.Run("RemoveDeviceFromZone errors if the device can not be found", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}, deviceLock: &sync.Mutex{}, devices: map[string]*DeviceMetadata{}}

		err := do.RemoveDeviceFromZone("id", 1)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("RemoveDeviceFromZone errors if the zone can not be found", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}, deviceLock: &sync.Mutex{}, devices: map[string]*DeviceMetadata{}}

		do.AddDevice("id")

		err := do.RemoveDeviceFromZone("id", 1)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("RemoveDeviceFromZone removes the devices from the zone and zone from device", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}, deviceLock: &sync.Mutex{}, devices: map[string]*DeviceMetadata{}}

		do.AddDevice("id")
		zone := do.NewZone("name")

		err := do.AddDeviceToZone("id", zone.Identifier)
		assert.NoError(t, err)

		err = do.RemoveDeviceFromZone("id", zone.Identifier)
		assert.NoError(t, err)

		checkDevice, _ := do.Device("id")
		checkZone, _ := do.Zone(zone.Identifier)

		assert.NotContains(t, checkDevice.Zones, zone.Identifier)
		assert.NotContains(t, checkZone.Devices, "id")
	})

	t.Run("RemoveDevice removes the device from any zones that its in", func(t *testing.T) {
		startId := int64(0)
		do := DeviceOrganiser{nextZoneId: &startId, zoneLock: &sync.Mutex{}, zones: map[int]*Zone{}, deviceLock: &sync.Mutex{}, devices: map[string]*DeviceMetadata{}}

		do.AddDevice("id")
		zone := do.NewZone("name")

		err := do.AddDeviceToZone("id", zone.Identifier)
		assert.NoError(t, err)

		do.RemoveDevice("id")

		checkZone, found := do.Zone(zone.Identifier)
		assert.True(t, found)

		assert.NotContains(t, checkZone.Devices, "id")
	})
}
