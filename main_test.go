package main

import (
	"github.com/shimmeringbee/controller/state"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_updateDeviceOrganiserFromMux(t *testing.T) {
	t.Run("adds a device when a DeviceAdded event is received", func(t *testing.T) {
		do := state.NewDeviceOrganiser(nil)

		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		ch := updateDeviceOrganiserFromMux(&do)
		defer func() {
			ch <- nil
		}()

		ch <- da.DeviceAdded{
			Device: mocks.SimpleDevice{
				SIdentifier: addr,
			},
		}

		time.Sleep(10 * time.Millisecond)

		_, found := do.Device(addr.String())
		assert.True(t, found)
	})

	t.Run("removes a device when a DeviceRemoved event is received", func(t *testing.T) {
		do := state.NewDeviceOrganiser(nil)
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		do.AddDevice(addr.String())

		ch := updateDeviceOrganiserFromMux(&do)
		defer func() {
			ch <- nil
		}()

		ch <- da.DeviceRemoved{
			Device: mocks.SimpleDevice{
				SIdentifier: addr,
			},
		}

		time.Sleep(10 * time.Millisecond)

		_, found := do.Device(addr.String())
		assert.False(t, found)
	})
}
