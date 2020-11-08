package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type Zone struct {
	Identifier int
	Name       string

	ParentZone int
	ChildZones []int

	Devices []string
}

type DeviceOrganiser struct {
	nextZoneId *int64

	zoneLock  *sync.Mutex
	rootZones []int
	zones     map[int]*Zone
}

type ZoneError string

func (z ZoneError) Error() string {
	return string(z)
}

const (
	ErrCircularReference = ZoneError("operation would result in circular reference in zone")
	ErrNotFound          = ZoneError("zone not found")
	ErrMoveSameZone      = ZoneError("zone can not be moved to itself")
	ErrOrphanZone        = ZoneError("operation would result in orphaned zone")
	ErrHasDevices        = ZoneError("zone has devices")
)

const RootZoneId int = 0

func (d *DeviceOrganiser) Zone(id int) (Zone, bool) {
	d.zoneLock.Lock()
	defer d.zoneLock.Unlock()

	if zone, found := d.zones[id]; found {
		return *zone, found
	} else {
		return Zone{}, found
	}
}

func (d *DeviceOrganiser) RootZones() []Zone {
	d.zoneLock.Lock()
	defer d.zoneLock.Unlock()

	var rootZones []Zone

	for _, zoneId := range d.rootZones {
		rootZones = append(rootZones, *d.zones[zoneId])
	}

	return rootZones
}

func (d *DeviceOrganiser) NewZone(name string) Zone {
	newId := int(atomic.AddInt64(d.nextZoneId, 1))

	newZone := &Zone{
		Identifier: newId,
		Name:       name,
		ChildZones: nil,
		Devices:    nil,
	}

	d.zoneLock.Lock()
	defer d.zoneLock.Unlock()

	d.rootZones = append(d.rootZones, newId)
	d.zones[newId] = newZone

	return *newZone
}

func filterInt(haystack []int, needle int) []int {
	var result []int

	for _, check := range haystack {
		if check != needle {
			result = append(result, check)
		}
	}

	return result
}

func (d *DeviceOrganiser) DeleteZone(id int) error {
	d.zoneLock.Lock()
	defer d.zoneLock.Unlock()

	zone, found := d.zones[id]
	if !found {
		return fmt.Errorf("zone not found: %w", ErrNotFound)
	}

	if len(zone.ChildZones) > 0 {
		return ErrOrphanZone
	}

	if len(zone.Devices) > 0 {
		return ErrHasDevices
	}

	delete(d.zones, id)

	if zone.ParentZone == RootZoneId {
		d.rootZones = filterInt(d.rootZones, id)
	} else {
		parent, found := d.zones[zone.ParentZone]
		if found {
			parent.ChildZones = filterInt(parent.ChildZones, id)
		}
	}

	return nil
}

func (d *DeviceOrganiser) MoveZone(id int, newParentId int) error {
	if id == newParentId {
		return ErrMoveSameZone
	}

	d.zoneLock.Lock()
	defer d.zoneLock.Unlock()

	zone, found := d.zones[id]
	if !found {
		return fmt.Errorf("zone not found: %w", ErrNotFound)
	}

	var newParent *Zone

	if newParentId != RootZoneId {
		newParent, found = d.zones[newParentId]
		if !found {
			return fmt.Errorf("new parent not found: %w", ErrNotFound)
		}
	}

	for _, childId := range d.enumerateZoneDescendents(id) {
		if newParentId == childId {
			return ErrCircularReference
		}
	}

	if zone.ParentZone == RootZoneId {
		d.rootZones = filterInt(d.rootZones, id)
	} else {
		if oldParent, found := d.zones[zone.ParentZone]; !found {
			return fmt.Errorf("old parent not found: %w", ErrNotFound)
		} else {
			oldParent.ChildZones = filterInt(oldParent.ChildZones, id)
		}
	}

	zone.ParentZone = newParentId

	if newParent == nil {
		d.rootZones = append(d.rootZones, id)
	} else {
		newParent.ChildZones = append(newParent.ChildZones, id)
	}

	return nil
}

func (d *DeviceOrganiser) NameZone(id int, name string) error {
	d.zoneLock.Lock()
	defer d.zoneLock.Unlock()

	if zone, found := d.zones[id]; found {
		zone.Name = name
		return nil
	} else {
		return ErrNotFound
	}
}

func (d *DeviceOrganiser) enumerateZoneDescendents(id int) []int {
	zone := d.zones[id]

	var children []int

	children = append(children, zone.ChildZones...)

	for _, childId := range zone.ChildZones {
		grandChildren := d.enumerateZoneDescendents(childId)
		children = append(children, grandChildren...)
	}

	return children
}
