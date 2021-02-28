package application

import (
	"github.com/shimmeringbee/da"
	"reflect"
	"sync"
)

type DeviceFilter struct {
	m       *sync.RWMutex
	Devices map[da.Identifier]bool
	AutoAdd func(da.Device) bool
}

func (d *DeviceFilter) FilterDevices(devices []da.Device) []da.Device {
	var filteredDevices []da.Device

	for _, device := range devices {
		if d.HasDevice(device.Identifier()) {
			filteredDevices = append(filteredDevices, device)
		}
	}

	return filteredDevices
}

func (d *DeviceFilter) FilterEvents(ch chan interface{}) chan interface{} {
	downCh := make(chan interface{}, 1)

	go func() {
		for msg := range ch {
			device := getDeviceFromEvent(msg)
			if device == nil || d.HasDevice(device.Identifier()) {
				downCh <- msg
			}
		}
	}()

	return downCh
}

func (d *DeviceFilter) AddDevice(id da.Identifier) {
	d.m.Lock()
	defer d.m.Unlock()

	d.Devices[id] = true
}

func (d *DeviceFilter) RemoveDevice(id da.Identifier) {
	d.m.Lock()
	defer d.m.Unlock()

	delete(d.Devices, id)
}

func (d *DeviceFilter) HasDevice(id da.Identifier) bool {
	d.m.RLock()
	defer d.m.RUnlock()

	_, found := d.Devices[id]
	return found
}

func (d *DeviceFilter) ListDevices() []da.Identifier {
	d.m.RLock()
	defer d.m.RUnlock()

	var devices []da.Identifier

	for id, _ := range d.Devices {
		devices = append(devices, id)
	}

	return devices
}

func getDeviceFromEvent(v interface{}) da.Device {
	structVal := reflect.ValueOf(v)
	structType := structVal.Type()

	for i := 0; i < structType.NumField(); i++ {
		fieldType := structType.Field(i).Type

		if fieldType.String() == "da.Device" {
			fieldValue := structVal.Field(i)

			d := fieldValue.Elem().Interface().(da.Device)
			return d
		}
	}

	return nil
}
