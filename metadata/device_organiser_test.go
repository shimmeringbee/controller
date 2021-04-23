package metadata

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestDeviceOrganiser_Zones(t *testing.T) {
	t.Run("NewZone generates a new zone creates it at the root with an incrementing id", func(t *testing.T) {
		do := NewDeviceOrganiser()

		zoneOne := do.NewZone("one")
		zoneTwo := do.NewZone("two")

		assert.Equal(t, "one", zoneOne.Name)
		assert.Equal(t, 1, zoneOne.Identifier)

		assert.Equal(t, "two", zoneTwo.Name)
		assert.Equal(t, 2, zoneTwo.Identifier)
	})

	t.Run("RootZones returns the constructed root", func(t *testing.T) {
		do := NewDeviceOrganiser()

		zoneOne := do.NewZone("one")

		roots := do.RootZones()
		assert.Len(t, roots, 1)
		assert.Contains(t, roots, zoneOne)
	})

	t.Run("GetZone returns a zone by id", func(t *testing.T) {
		do := NewDeviceOrganiser()

		zoneOne := do.NewZone("one")

		foundZone, found := do.Zone(zoneOne.Identifier)
		assert.True(t, found)
		assert.Equal(t, zoneOne, foundZone)
	})

	t.Run("GetZone returns false if it can't find the zone by id", func(t *testing.T) {
		do := NewDeviceOrganiser()

		_, found := do.Zone(1)
		assert.False(t, found)
	})

	t.Run("NameZone returns an error if the zone does not exist", func(t *testing.T) {
		do := NewDeviceOrganiser()

		err := do.NameZone(1, "NewDeviceOrganiser Name")
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("NameZone updates a zones name", func(t *testing.T) {
		do := NewDeviceOrganiser()

		zoneOne := do.NewZone("one")

		newName := "NewDeviceOrganiser Name"
		err := do.NameZone(zoneOne.Identifier, newName)
		assert.NoError(t, err)

		changedZone, _ := do.Zone(zoneOne.Identifier)
		assert.Equal(t, newName, changedZone.Name)
	})

	t.Run("MoveZone errors if the zone being moved does not exist", func(t *testing.T) {
		do := NewDeviceOrganiser()

		err := do.MoveZone(1, -1)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("MoveZone errors if the parent zone does not exist", func(t *testing.T) {
		do := NewDeviceOrganiser()

		zoneOne := do.NewZone("one")

		err := do.MoveZone(zoneOne.Identifier, -1)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("MoveZone errors if the moved zone and parent are equal", func(t *testing.T) {
		do := NewDeviceOrganiser()

		zoneOne := do.NewZone("one")

		err := do.MoveZone(zoneOne.Identifier, zoneOne.Identifier)
		assert.True(t, errors.Is(err, ErrSameZone))
	})

	t.Run("MoveZone succeeds in moving one root entry under another, removing the old root entry", func(t *testing.T) {
		do := NewDeviceOrganiser()

		zoneOne := do.NewZone("one")
		zoneTwo := do.NewZone("two")

		err := do.MoveZone(zoneTwo.Identifier, zoneOne.Identifier)
		assert.NoError(t, err)

		roots := do.hiddenRoot.SubZones
		assert.Len(t, roots, 1)
		assert.Equal(t, roots[0], zoneOne.Identifier)

		afterOne, _ := do.Zone(zoneOne.Identifier)
		afterTwo, _ := do.Zone(zoneTwo.Identifier)

		assert.Contains(t, afterOne.SubZones, zoneTwo.Identifier)
		assert.Equal(t, afterOne.Identifier, afterTwo.ParentZone)
	})

	t.Run("MoveZone succeeds in moving a sub zone under another sub zone", func(t *testing.T) {
		do := NewDeviceOrganiser()

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
		do := NewDeviceOrganiser()

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
		do := NewDeviceOrganiser()

		zoneOne := do.NewZone("one")
		zoneTwo := do.NewZone("two")

		err := do.MoveZone(zoneTwo.Identifier, zoneOne.Identifier)
		assert.NoError(t, err)

		err = do.MoveZone(zoneTwo.Identifier, RootZoneId)
		assert.NoError(t, err)

		checkOne, _ := do.Zone(zoneOne.Identifier)
		assert.Len(t, checkOne.SubZones, 0)

		roots := do.hiddenRoot.SubZones
		assert.Len(t, roots, 2)
		assert.Contains(t, roots, zoneTwo.Identifier)

		assert.Equal(t, RootZoneId, zoneTwo.ParentZone)
	})

	t.Run("ReorderZoneBefore errors if zone being reordered does not exist", func(t *testing.T) {
		do := NewDeviceOrganiser()
		beforeZone := do.NewZone("before")

		err := do.ReorderZoneBefore(999, beforeZone.Identifier)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("ReorderZoneBefore errors if before zone does not exist", func(t *testing.T) {
		do := NewDeviceOrganiser()
		beforeZone := do.NewZone("before")

		err := do.ReorderZoneBefore(beforeZone.Identifier, 999)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("ReorderZoneBefore errors if zones do not have same parent", func(t *testing.T) {
		do := NewDeviceOrganiser()
		a := do.NewZone("a")
		b := do.NewZone("b")

		c := do.NewZone("c")
		d := do.NewZone("d")

		do.MoveZone(c.Identifier, a.Identifier)
		do.MoveZone(d.Identifier, b.Identifier)

		err := do.ReorderZoneBefore(c.Identifier, d.Identifier)
		assert.True(t, errors.Is(err, ErrMustHaveSameParent))
	})

	t.Run("ReorderZoneBefore errors if zones are the same zone", func(t *testing.T) {
		do := NewDeviceOrganiser()
		moveZone := do.NewZone("before")

		err := do.ReorderZoneBefore(moveZone.Identifier, moveZone.Identifier)
		assert.True(t, errors.Is(err, ErrSameZone))
	})

	t.Run("ReorderZoneBefore succeeds reordering a zone, mid list", func(t *testing.T) {
		do := NewDeviceOrganiser()

		_ = do.NewZone("a")
		b := do.NewZone("b")
		c := do.NewZone("c")

		err := do.ReorderZoneBefore(c.Identifier, b.Identifier)
		assert.NoError(t, err)
		afterOrder := do.hiddenRoot.SubZones

		assert.Equal(t, []int{1, 3, 2}, afterOrder)
	})

	t.Run("ReorderZoneBefore succeeds reordering a zone, to list head", func(t *testing.T) {
		do := NewDeviceOrganiser()

		a := do.NewZone("a")
		_ = do.NewZone("b")
		c := do.NewZone("c")

		err := do.ReorderZoneBefore(c.Identifier, a.Identifier)
		assert.NoError(t, err)
		afterOrder := do.hiddenRoot.SubZones

		assert.Equal(t, []int{3, 1, 2}, afterOrder)
	})

	t.Run("ReorderZoneAfter errors if zone being reordered does not exist", func(t *testing.T) {
		do := NewDeviceOrganiser()
		afterZone := do.NewZone("After")

		err := do.ReorderZoneAfter(999, afterZone.Identifier)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("ReorderZoneAfter errors if After zone does not exist", func(t *testing.T) {
		do := NewDeviceOrganiser()
		afterZone := do.NewZone("After")

		err := do.ReorderZoneAfter(afterZone.Identifier, 999)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("ReorderZoneAfter errors if zones do not have same parent", func(t *testing.T) {
		do := NewDeviceOrganiser()
		a := do.NewZone("a")
		b := do.NewZone("b")

		c := do.NewZone("c")
		d := do.NewZone("d")

		do.MoveZone(c.Identifier, a.Identifier)
		do.MoveZone(d.Identifier, b.Identifier)

		err := do.ReorderZoneAfter(c.Identifier, d.Identifier)
		assert.True(t, errors.Is(err, ErrMustHaveSameParent))
	})

	t.Run("ReorderZoneAfter errors if zones are the same zone", func(t *testing.T) {
		do := NewDeviceOrganiser()
		moveZone := do.NewZone("After")

		err := do.ReorderZoneAfter(moveZone.Identifier, moveZone.Identifier)
		assert.True(t, errors.Is(err, ErrSameZone))
	})

	t.Run("ReorderZoneAfter succeeds reordering a zone, mid list", func(t *testing.T) {
		do := NewDeviceOrganiser()

		_ = do.NewZone("a")
		b := do.NewZone("b")
		c := do.NewZone("c")

		err := do.ReorderZoneAfter(b.Identifier, c.Identifier)
		assert.NoError(t, err)
		afterOrder := do.hiddenRoot.SubZones

		assert.Equal(t, []int{1, 3, 2}, afterOrder)
	})

	t.Run("ReorderZoneAfter succeeds reordering a zone, to list tail", func(t *testing.T) {
		do := NewDeviceOrganiser()

		a := do.NewZone("a")
		_ = do.NewZone("b")
		c := do.NewZone("c")

		err := do.ReorderZoneAfter(a.Identifier, c.Identifier)
		assert.NoError(t, err)
		afterOrder := do.hiddenRoot.SubZones

		assert.Equal(t, []int{2, 3, 1}, afterOrder)
	})

	t.Run("DeleteZone errors if zone can not be found", func(t *testing.T) {
		do := NewDeviceOrganiser()

		err := do.DeleteZone(1)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("DeleteZone errors if zone has subzone found", func(t *testing.T) {
		do := NewDeviceOrganiser()

		zoneOne := do.NewZone("one")
		zoneTwo := do.NewZone("two")

		err := do.MoveZone(zoneTwo.Identifier, zoneOne.Identifier)
		assert.NoError(t, err)

		err = do.DeleteZone(zoneOne.Identifier)
		assert.True(t, errors.Is(err, ErrOrphanZone))
	})

	t.Run("DeleteZone errors if zone has device", func(t *testing.T) {
		do := NewDeviceOrganiser()

		zoneOne := do.NewZone("one")
		do.zones[zoneOne.Identifier].Devices = []string{"device"}

		err := do.DeleteZone(zoneOne.Identifier)
		assert.True(t, errors.Is(err, ErrHasDevices))
	})

	t.Run("DeleteZone succeeds deleting a subzone", func(t *testing.T) {
		do := NewDeviceOrganiser()

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
		do := NewDeviceOrganiser()

		zoneOne := do.NewZone("one")

		err := do.DeleteZone(zoneOne.Identifier)
		assert.NoError(t, err)
		assert.NotContains(t, do.zones, zoneOne.Identifier)
		assert.NotContains(t, do.hiddenRoot.SubZones, zoneOne.Identifier)
	})
}

func TestDeviceOrganiser_Devices(t *testing.T) {
	t.Run("AddDevice adds a device", func(t *testing.T) {
		do := NewDeviceOrganiser()
		do.AddDevice("id")

		_, found := do.Device("id")
		assert.True(t, found)
	})

	t.Run("Device returns false if device is not present", func(t *testing.T) {
		do := NewDeviceOrganiser()
		_, found := do.Device("id")
		assert.False(t, found)
	})

	t.Run("Device returns true if device is present", func(t *testing.T) {
		do := NewDeviceOrganiser()
		do.AddDevice("id")
		_, found := do.Device("id")
		assert.True(t, found)
	})

	t.Run("NameDevice errors if device doesn't exist", func(t *testing.T) {
		do := NewDeviceOrganiser()

		err := do.NameDevice("id", "name")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("NameDevice sets a name on a device", func(t *testing.T) {
		do := NewDeviceOrganiser()
		do.AddDevice("id")

		err := do.NameDevice("id", "name")
		assert.NoError(t, err)

		dm, found := do.Device("id")
		assert.True(t, found)
		assert.Equal(t, "name", dm.Name)
	})

	t.Run("NameDevice errors if the device does not exist", func(t *testing.T) {
		do := NewDeviceOrganiser()

		err := do.NameDevice("id", "name")
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("AddDevice does not overwrite an existing device", func(t *testing.T) {
		do := NewDeviceOrganiser()
		do.AddDevice("id")

		err := do.NameDevice("id", "name")
		assert.NoError(t, err)

		do.AddDevice("id")

		dm, found := do.Device("id")
		assert.True(t, found)
		assert.Equal(t, "name", dm.Name)
	})

	t.Run("RemoveDevice removes an added device", func(t *testing.T) {
		do := NewDeviceOrganiser()
		do.AddDevice("id")

		do.RemoveDevice("id")
		_, found := do.Device("id")
		assert.False(t, found)
	})

	t.Run("AddDeviceToZone errors if the device can not be found", func(t *testing.T) {
		do := NewDeviceOrganiser()

		err := do.AddDeviceToZone("id", 1)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("AddDeviceToZone errors if the zone can not be found", func(t *testing.T) {
		do := NewDeviceOrganiser()

		do.AddDevice("id")

		err := do.AddDeviceToZone("id", 1)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("AddDeviceToZone adds the zone to the device and device to zone", func(t *testing.T) {
		do := NewDeviceOrganiser()

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
		do := NewDeviceOrganiser()

		err := do.RemoveDeviceFromZone("id", 1)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("RemoveDeviceFromZone errors if the zone can not be found", func(t *testing.T) {
		do := NewDeviceOrganiser()

		do.AddDevice("id")

		err := do.RemoveDeviceFromZone("id", 1)
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("RemoveDeviceFromZone removes the devices from the zone and zone from device", func(t *testing.T) {
		do := NewDeviceOrganiser()

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
		do := NewDeviceOrganiser()

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

func TestDeviceOrganiser_persistZones(t *testing.T) {
	t.Run("saves and reloads zones successfully", func(t *testing.T) {
		file, err := ioutil.TempFile(os.TempDir(), "shimmeringbeecontrollertest")
		file.Close()

		assert.NoError(t, err)
		defer os.Remove(file.Name())

		do := NewDeviceOrganiser()

		one := do.NewZone("one")
		two := do.NewZone("two")
		three := do.NewZone("three")
		four := do.NewZone("four")

		err = do.ReorderZoneBefore(four.Identifier, one.Identifier)
		assert.NoError(t, err)

		err = do.MoveZone(two.Identifier, one.Identifier)
		assert.NoError(t, err)
		err = do.MoveZone(three.Identifier, two.Identifier)
		assert.NoError(t, err)

		err = SaveZones(file.Name(), &do)
		assert.NoError(t, err)

		newDo := NewDeviceOrganiser()
		err = LoadZones(file.Name(), &newDo)
		assert.Equal(t, do.nextZoneId, newDo.nextZoneId)
		assert.Equal(t, do.zones, newDo.zones)
	})
}

func TestDeviceOrganiser_persistDevices(t *testing.T) {
	t.Run("saves and reloads devices successfully", func(t *testing.T) {
		file, err := ioutil.TempFile(os.TempDir(), "shimmeringbeecontrollertest")
		file.Close()

		assert.NoError(t, err)
		defer os.Remove(file.Name())

		do := NewDeviceOrganiser()
		zone := do.NewZone("one")

		do.AddDevice("id")
		do.NameDevice("id", "name")
		do.AddDeviceToZone("id", zone.Identifier)

		err = SaveDevices(file.Name(), &do)
		assert.NoError(t, err)

		newDo := NewDeviceOrganiser()
		newDo.NewZone("one")

		err = LoadDevices(file.Name(), &newDo)
		assert.NoError(t, err)

		device, found := newDo.Device("id")
		assert.True(t, found)
		assert.Equal(t, "name", device.Name)
		assert.Contains(t, device.Zones, zone.Identifier)

		zone, _ = newDo.Zone(zone.Identifier)
		assert.Contains(t, zone.Devices, "id")
	})
}
