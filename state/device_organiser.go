package state

import (
	"encoding/json"
	"fmt"
	"github.com/shimmeringbee/controller/config"
	"github.com/shimmeringbee/persistence"
	"io/ioutil"
	"os"
	"sync"
	"sync/atomic"
)

type Zone struct {
	Identifier int
	Name       string
	ParentZone int

	SubZones []int    `json:"-"`
	Devices  []string `json:"-"`
}

type DeviceMetadata struct {
	Name  string `json:",omitempty"`
	Zones []int  `json:",omitempty"`
}

type DeviceOrganiser struct {
	nextZoneId *int64

	zoneLock   *sync.Mutex
	zones      map[int]*Zone
	hiddenRoot *Zone

	deviceLock *sync.Mutex
	devices    map[string]*DeviceMetadata
}

type ZoneError string

func (z ZoneError) Error() string {
	return string(z)
}

const (
	ErrCircularReference  = ZoneError("operation would result in circular reference in zone")
	ErrNotFound           = ZoneError("not found")
	ErrSameZone           = ZoneError("zone can not be moved/reordered to itself")
	ErrOrphanZone         = ZoneError("operation would result in orphaned zone")
	ErrHasDevices         = ZoneError("zone has devices")
	ErrMustHaveSameParent = ZoneError("zones being reordered must have same parent")
)

const RootZoneId int = 0
const DefaultFilePermissions = 0600

func NewDeviceOrganiser(persistence.Section) DeviceOrganiser {
	initialZoneId := int64(0)
	hiddenZone := &Zone{Identifier: RootZoneId, Name: "Hidden Root"}

	return DeviceOrganiser{
		nextZoneId: &initialZoneId,
		zoneLock:   &sync.Mutex{},
		zones:      map[int]*Zone{RootZoneId: hiddenZone},
		hiddenRoot: hiddenZone,
		deviceLock: &sync.Mutex{},
		devices:    map[string]*DeviceMetadata{},
	}
}

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

	for _, zoneId := range d.hiddenRoot.SubZones {
		rootZones = append(rootZones, *d.zones[zoneId])
	}

	return rootZones
}

func (d *DeviceOrganiser) NewZone(name string) Zone {
	newId := int(atomic.AddInt64(d.nextZoneId, 1))

	newZone := &Zone{
		Identifier: newId,
		Name:       name,
		SubZones:   nil,
		Devices:    nil,
	}

	d.zoneLock.Lock()
	defer d.zoneLock.Unlock()

	d.hiddenRoot.SubZones = append(d.hiddenRoot.SubZones, newId)
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

func filterString(haystack []string, needle string) []string {
	var result []string

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

	if len(zone.SubZones) > 0 {
		return ErrOrphanZone
	}

	if len(zone.Devices) > 0 {
		return ErrHasDevices
	}

	delete(d.zones, id)

	parent, found := d.zones[zone.ParentZone]
	if found {
		parent.SubZones = filterInt(parent.SubZones, id)
	}

	return nil
}

func (d *DeviceOrganiser) MoveZone(id int, newParentId int) error {
	if id == newParentId {
		return ErrSameZone
	}

	d.zoneLock.Lock()
	defer d.zoneLock.Unlock()

	zone, found := d.zones[id]
	if !found {
		return fmt.Errorf("zone not found: %w", ErrNotFound)
	}

	var newParent *Zone

	newParent, found = d.zones[newParentId]
	if !found {
		return fmt.Errorf("new parent not found: %w", ErrNotFound)
	}

	for _, subId := range d.enumerateZoneDescendents(id) {
		if newParentId == subId {
			return ErrCircularReference
		}
	}

	if oldParent, found := d.zones[zone.ParentZone]; !found {
		return fmt.Errorf("old parent not found: %w", ErrNotFound)
	} else {
		oldParent.SubZones = filterInt(oldParent.SubZones, id)
	}

	zone.ParentZone = newParentId

	newParent.SubZones = append(newParent.SubZones, id)

	return nil
}

func (d *DeviceOrganiser) ReorderZoneBefore(id int, beforeId int) error {
	if id == beforeId {
		return ErrSameZone
	}

	d.zoneLock.Lock()
	defer d.zoneLock.Unlock()

	zone, found := d.zones[id]
	if !found {
		return fmt.Errorf("zone not found: %w", ErrNotFound)
	}

	beforeZone, found := d.zones[beforeId]
	if !found {
		return fmt.Errorf("before zone not found: %w", ErrNotFound)
	}

	if zone.ParentZone != beforeZone.ParentZone {
		return fmt.Errorf("zones do not share parent: %w", ErrMustHaveSameParent)
	}

	parentZone, found := d.zones[zone.ParentZone]
	if !found {
		return fmt.Errorf("could not find parent zone, corrupt state: %w", ErrNotFound)
	}

	var newSubZoneOrder []int

	for _, subZoneId := range parentZone.SubZones {
		if subZoneId == beforeId {
			newSubZoneOrder = append(newSubZoneOrder, id)
		}

		if subZoneId != id {
			newSubZoneOrder = append(newSubZoneOrder, subZoneId)
		}
	}

	parentZone.SubZones = newSubZoneOrder

	return nil
}

func (d *DeviceOrganiser) ReorderZoneAfter(id int, afterId int) error {
	if id == afterId {
		return ErrSameZone
	}

	d.zoneLock.Lock()
	defer d.zoneLock.Unlock()

	zone, found := d.zones[id]
	if !found {
		return fmt.Errorf("zone not found: %w", ErrNotFound)
	}

	beforeZone, found := d.zones[afterId]
	if !found {
		return fmt.Errorf("before zone not found: %w", ErrNotFound)
	}

	if zone.ParentZone != beforeZone.ParentZone {
		return fmt.Errorf("zones do not share parent: %w", ErrMustHaveSameParent)
	}

	parentZone, found := d.zones[zone.ParentZone]
	if !found {
		return fmt.Errorf("could not find parent zone, corrupt state: %w", ErrNotFound)
	}

	var newSubZoneOrder []int

	for _, subZoneId := range parentZone.SubZones {
		if subZoneId != id {
			newSubZoneOrder = append(newSubZoneOrder, subZoneId)
		}

		if subZoneId == afterId {
			newSubZoneOrder = append(newSubZoneOrder, id)
		}
	}

	parentZone.SubZones = newSubZoneOrder

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

func (d *DeviceOrganiser) AddDevice(id string) {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	if _, found := d.devices[id]; found {
		return
	}

	d.devices[id] = &DeviceMetadata{}
}

func (d *DeviceOrganiser) Device(id string) (DeviceMetadata, bool) {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	if dm, found := d.devices[id]; found {
		return *dm, true
	} else {
		return DeviceMetadata{}, false
	}
}

func (d *DeviceOrganiser) NameDevice(id string, name string) error {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	if dm, found := d.devices[id]; found {
		dm.Name = name
		return nil
	} else {
		return ErrNotFound
	}
}

func (d *DeviceOrganiser) RemoveDevice(id string) {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	device, found := d.devices[id]
	if !found {
		return
	}

	if len(device.Zones) > 0 {
		d.zoneLock.Lock()
		defer d.zoneLock.Unlock()

		for _, zoneId := range device.Zones {
			zone, zoneFound := d.zones[zoneId]
			if zoneFound {
				zone.Devices = filterString(zone.Devices, id)
			}
		}
	}

	delete(d.devices, id)
}

func (d *DeviceOrganiser) AddDeviceToZone(deviceId string, zoneId int) error {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	device, found := d.devices[deviceId]
	if !found {
		return ErrNotFound
	}

	d.zoneLock.Lock()
	defer d.zoneLock.Unlock()

	zone, found := d.zones[zoneId]
	if !found {
		return ErrNotFound
	}

	device.Zones = append(device.Zones, zoneId)
	zone.Devices = append(zone.Devices, deviceId)

	return nil
}

func (d *DeviceOrganiser) RemoveDeviceFromZone(deviceId string, zoneId int) error {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	device, found := d.devices[deviceId]
	if !found {
		return ErrNotFound
	}

	d.zoneLock.Lock()
	defer d.zoneLock.Unlock()

	zone, found := d.zones[zoneId]
	if !found {
		return ErrNotFound
	}

	device.Zones = filterInt(device.Zones, zoneId)
	zone.Devices = filterString(zone.Devices, deviceId)

	return nil
}

func (d *DeviceOrganiser) enumerateZoneDescendents(id int) []int {
	zone := d.zones[id]

	var subZones []int

	subZones = append(subZones, zone.SubZones...)

	for _, subId := range zone.SubZones {
		descendentZones := d.enumerateZoneDescendents(subId)
		subZones = append(subZones, descendentZones...)
	}

	return subZones
}

type SavedZones struct {
	NextZoneId int64
	Zones      []Zone
}

func SaveZones(fileLocation string, do *DeviceOrganiser) error {
	do.zoneLock.Lock()
	defer do.zoneLock.Unlock()

	var saved SavedZones

	recurseSaveZones(do, RootZoneId, &saved)

	saved.NextZoneId = *do.nextZoneId

	data, err := json.Marshal(saved)
	if err != nil {
		return err
	}

	return config.SafeWriteFile(fileLocation, data, DefaultFilePermissions)
}

func recurseSaveZones(do *DeviceOrganiser, id int, saved *SavedZones) {
	z := do.zones[id]

	if id != RootZoneId {
		saved.Zones = append(saved.Zones, *z)
	}

	for _, sid := range z.SubZones {
		recurseSaveZones(do, sid, saved)
	}
}

func LoadZones(fileLocation string, do *DeviceOrganiser) error {
	if _, err := os.Stat(fileLocation); err != nil {
		return nil
	}

	data, err := ioutil.ReadFile(fileLocation)
	if err != nil {
		return err
	}

	var loaded SavedZones

	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	do.zoneLock.Lock()
	do.nextZoneId = &loaded.NextZoneId

	for _, zone := range loaded.Zones {
		if zone.Identifier != RootZoneId {
			copyZone := zone
			copyZone.ParentZone = 0
			do.zones[zone.Identifier] = &copyZone
			do.hiddenRoot.SubZones = append(do.hiddenRoot.SubZones, zone.Identifier)
		}
	}

	do.zoneLock.Unlock()

	for _, zone := range loaded.Zones {
		if zone.ParentZone != RootZoneId {
			if err := do.MoveZone(zone.Identifier, zone.ParentZone); err != nil {
				return err
			}
		}
	}

	return nil
}

func LoadDevices(fileLocation string, do *DeviceOrganiser) error {
	if _, err := os.Stat(fileLocation); err != nil {
		return nil
	}

	data, err := ioutil.ReadFile(fileLocation)
	if err != nil {
		return err
	}

	loaded := map[string]DeviceMetadata{}
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	for id, dm := range loaded {
		do.AddDevice(id)
		if err := do.NameDevice(id, dm.Name); err != nil {
			return err
		}

		for _, zone := range dm.Zones {
			if err := do.AddDeviceToZone(id, zone); err != nil {
				return err
			}
		}
	}

	return nil
}

func SaveDevices(fileLocation string, do *DeviceOrganiser) error {
	do.deviceLock.Lock()
	defer do.deviceLock.Unlock()

	saved := map[string]DeviceMetadata{}

	for id, device := range do.devices {
		saved[id] = *device
	}

	data, err := json.Marshal(saved)
	if err != nil {
		return err
	}

	return config.SafeWriteFile(fileLocation, data, DefaultFilePermissions)
}
