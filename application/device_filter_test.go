package application

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestDeviceFilter_DeviceManagement(t *testing.T) {
	t.Run("adding and removing are represented in the list and has device", func(t *testing.T) {
		id := zigbee.GenerateLocalAdministeredIEEEAddress()

		df := DeviceFilter{
			m:       &sync.RWMutex{},
			Devices: make(map[da.Identifier]bool),
		}

		assert.False(t, df.HasDevice(id))
		devices := df.ListDevices()
		assert.Empty(t, devices)

		df.AddDevice(id)

		assert.True(t, df.HasDevice(id))
		devices = df.ListDevices()
		assert.Contains(t, devices, id)

		df.RemoveDevice(id)

		assert.False(t, df.HasDevice(id))
		devices = df.ListDevices()
		assert.Empty(t, devices)
	})
}

func TestDeviceFilter_FilterDevices(t *testing.T) {
	t.Run("filters devices", func(t *testing.T) {
		present := zigbee.IEEEAddress(1)
		absent := zigbee.IEEEAddress(2)

		df := DeviceFilter{
			m:       &sync.RWMutex{},
			Devices: make(map[da.Identifier]bool),
		}

		df.AddDevice(present)

		presentDevice := da.BaseDevice{DeviceIdentifier: present}
		absentDevice := da.BaseDevice{DeviceIdentifier: absent}

		list := []da.Device{presentDevice, absentDevice}
		filtered := df.FilterDevices(list)

		assert.Contains(t, filtered, presentDevice)
		assert.NotContains(t, filtered, absentDevice)
	})
}

func Test_getDeviceFromEvent(t *testing.T) {
	t.Run("returns device found in event", func(t *testing.T) {
		d := da.BaseDevice{DeviceIdentifier: zigbee.IEEEAddress(1)}
		castD := da.Device(d)

		e := da.DeviceAdded{Device: d}

		extractedDevice := getDeviceFromEvent(e)
		assert.Equal(t, castD, extractedDevice)
	})

	t.Run("returns nil if no device on event", func(t *testing.T) {
		e := struct{}{}

		extractedDevice := getDeviceFromEvent(e)
		assert.Nil(t, extractedDevice)
	})
}

func TestDeviceFilter_FilterEvents(t *testing.T) {
	t.Run("allows event with accepted device through", func(t *testing.T) {
		present := zigbee.IEEEAddress(1)

		df := DeviceFilter{
			m:       &sync.RWMutex{},
			Devices: make(map[da.Identifier]bool),
		}

		df.AddDevice(present)

		presentDevice := da.BaseDevice{DeviceIdentifier: present}
		e := da.DeviceAdded{Device: presentDevice}

		in := make(chan interface{})
		defer close(in)
		out := df.FilterEvents(in)

		in <- e

		var outMsg interface{}

		select {
		case outMsg = <-out:
		case <-time.After(10 * time.Millisecond):
		}

		assert.Equal(t, e, outMsg)
	})

	t.Run("doesn't allow event with non accepted device through", func(t *testing.T) {
		present := zigbee.IEEEAddress(1)

		df := DeviceFilter{
			m:       &sync.RWMutex{},
			Devices: make(map[da.Identifier]bool),
		}

		presentDevice := da.BaseDevice{DeviceIdentifier: present}
		e := da.DeviceAdded{Device: presentDevice}

		in := make(chan interface{})
		defer close(in)
		out := df.FilterEvents(in)

		in <- e

		var outMsg interface{}

		select {
		case outMsg = <-out:
		case <-time.After(10 * time.Millisecond):
		}

		assert.Nil(t, outMsg)
	})

	t.Run("allows events that do not have device", func(t *testing.T) {
		df := DeviceFilter{
			m:       &sync.RWMutex{},
			Devices: make(map[da.Identifier]bool),
		}

		e := struct{}{}

		in := make(chan interface{})
		defer close(in)
		out := df.FilterEvents(in)

		in <- e

		var outMsg interface{}

		select {
		case outMsg = <-out:
		case <-time.After(10 * time.Millisecond):
		}

		assert.Equal(t, e, outMsg)
	})
}
