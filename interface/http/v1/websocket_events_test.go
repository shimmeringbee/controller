package v1

import (
	"context"
	"github.com/shimmeringbee/controller/interface/converters/exporter"
	"github.com/shimmeringbee/controller/state"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	damocks "github.com/shimmeringbee/da/capabilities/mocks"
	"github.com/shimmeringbee/da/mocks"
	"github.com/shimmeringbee/persistence/impl/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestWebsocketEventMapper_MapEvent(t *testing.T) {
	t.Run("maps an event from a capability of a device", func(t *testing.T) {
		do := state.NewDeviceOrganiser(memory.New(), state.NullEventPublisher)

		gm := &state.MockGatewayMapper{}
		defer gm.AssertExpectations(t)

		wem := websocketEventMapper{
			deviceOrganiser: &do,
			gatewayMapper:   gm,
			deviceExporter: &exporter.DeviceExporter{
				DeviceOrganiser: &do,
				GatewayMapper:   gm,
			},
		}

		mgw := &mocks.Gateway{}
		defer mgw.AssertExpectations(t)

		moo := &damocks.OnOff{}
		defer moo.AssertExpectations(t)

		mdev := &mocks.MockDevice{}
		defer mdev.AssertExpectations(t)

		mdev.On("Identifier").Return(SimpleIdentifier{id: "device"})
		mdev.On("Capability", capabilities.OnOffFlag).Return(moo)

		moo.Mock.On("Name").Return("OnOff")
		moo.Mock.On("Status", mock.Anything).Return(true, nil)

		expectedInitial := [][]byte{[]byte(`{"Type":"DeviceUpdateCapability","Identifier":"device","Capability":"OnOff","Payload":{"State":true}}`)}

		actualInitial, err := wem.MapEvent(context.TODO(), capabilities.OnOffUpdate{
			Device: mdev,
		})

		assert.NoError(t, err)
		assert.Equal(t, expectedInitial, actualInitial)
	})

	t.Run("maps addition of device", func(t *testing.T) {
		do := state.NewDeviceOrganiser(memory.New(), state.NullEventPublisher)
		gm := &state.MockGatewayMapper{}
		defer gm.AssertExpectations(t)

		mgw := &mocks.Gateway{}
		defer mgw.AssertExpectations(t)

		mdev := &mocks.MockDevice{}
		defer mdev.AssertExpectations(t)

		mdev.On("Identifier").Return(SimpleIdentifier{id: "device"})
		mdev.On("Capabilities").Return([]da.Capability{capabilities.ProductInformationFlag})
		mdev.On("Gateway").Return(mgw)

		mhpi := &damocks.ProductInformation{}
		defer mhpi.AssertExpectations(t)

		mhpi.On("Name").Return("ProductInformation")
		mhpi.On("Get", mock.Anything).Return(capabilities.ProductInfo{
			Manufacturer: "Manufacturer",
			Name:         "Name",
		}, nil)

		mdev.On("Capability", capabilities.ProductInformationFlag).Return(mhpi)

		gm.On("GatewayName", mgw).Return("gwname", true)

		wem := websocketEventMapper{
			deviceOrganiser: &do,
			gatewayMapper:   gm,
			deviceExporter: &exporter.DeviceExporter{
				DeviceOrganiser: &do,
				GatewayMapper:   gm,
			},
		}

		actualData, err := wem.MapEvent(context.TODO(), da.DeviceAdded{Device: mdev})

		expectedData := [][]byte{
			[]byte(`{"Type":"DeviceUpdate","Metadata":{},"Identifier":"device","Capabilities":["ProductInformation"],"Gateway":"gwname"}`),
			[]byte(`{"Type":"DeviceUpdateCapability","Identifier":"device","Capability":"ProductInformation","Payload":{"Name":"Name","Manufacturer":"Manufacturer"}}`),
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedData, actualData)
	})

	t.Run("maps stopped enumeration of device", func(t *testing.T) {
		do := state.NewDeviceOrganiser(memory.New(), state.NullEventPublisher)
		gm := &state.MockGatewayMapper{}
		defer gm.AssertExpectations(t)

		mgw := &mocks.Gateway{}
		defer mgw.AssertExpectations(t)

		mdev := &mocks.MockDevice{}
		defer mdev.AssertExpectations(t)

		mdev.On("Identifier").Return(SimpleIdentifier{id: "device"})
		mdev.On("Capabilities").Return([]da.Capability{capabilities.ProductInformationFlag})
		mdev.On("Gateway").Return(mgw)

		mhpi := &damocks.ProductInformation{}
		defer mhpi.AssertExpectations(t)

		mhpi.On("Name").Return("ProductInformation")
		mhpi.On("Get", mock.Anything).Return(capabilities.ProductInfo{
			Manufacturer: "Manufacturer",
			Name:         "Name",
		}, nil)

		mdev.On("Capability", capabilities.ProductInformationFlag).Return(mhpi)

		gm.On("GatewayName", mgw).Return("gwname", true)

		wem := websocketEventMapper{
			deviceOrganiser: &do,
			gatewayMapper:   gm,
			deviceExporter: &exporter.DeviceExporter{
				DeviceOrganiser: &do,
				GatewayMapper:   gm,
			},
		}

		actualData, err := wem.MapEvent(context.TODO(), capabilities.EnumerateDeviceStopped{Device: mdev})

		expectedData := [][]byte{
			[]byte(`{"Type":"DeviceUpdate","Metadata":{},"Identifier":"device","Capabilities":["ProductInformation"],"Gateway":"gwname"}`),
			[]byte(`{"Type":"DeviceUpdateCapability","Identifier":"device","Capability":"ProductInformation","Payload":{"Name":"Name","Manufacturer":"Manufacturer"}}`),
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedData, actualData)
	})

	t.Run("maps device metadata update", func(t *testing.T) {
		do := state.NewDeviceOrganiser(memory.New(), state.NullEventPublisher)
		gm := &state.MockGatewayMapper{}
		defer gm.AssertExpectations(t)

		mgw := &mocks.Gateway{}
		defer mgw.AssertExpectations(t)

		mdev := &mocks.MockDevice{}
		defer mdev.AssertExpectations(t)

		mdev.On("Identifier").Return(SimpleIdentifier{id: "device"})
		mdev.On("Capabilities").Return([]da.Capability{capabilities.ProductInformationFlag})
		mdev.On("Gateway").Return(mgw)

		gm.On("Device", "device").Return(mdev, true)

		mhpi := &damocks.ProductInformation{}
		defer mhpi.AssertExpectations(t)

		mhpi.On("Name").Return("ProductInformation")

		mdev.On("Capability", capabilities.ProductInformationFlag).Return(mhpi)

		gm.On("GatewayName", mgw).Return("gwname", true)

		wem := websocketEventMapper{
			deviceOrganiser: &do,
			gatewayMapper:   gm,
			deviceExporter: &exporter.DeviceExporter{
				DeviceOrganiser: &do,
				GatewayMapper:   gm,
			},
		}

		actualData, err := wem.MapEvent(context.TODO(), state.DeviceMetadataUpdate{Identifier: mdev.Identifier().String()})

		expectedData := [][]byte{
			[]byte(`{"Type":"DeviceUpdate","Metadata":{},"Identifier":"device","Capabilities":["ProductInformation"],"Gateway":"gwname"}`),
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedData, actualData)
	})

	t.Run("maps device added to zone event", func(t *testing.T) {
		do := state.NewDeviceOrganiser(memory.New(), state.NullEventPublisher)
		gm := &state.MockGatewayMapper{}
		defer gm.AssertExpectations(t)

		mgw := &mocks.Gateway{}
		defer mgw.AssertExpectations(t)

		mdev := &mocks.MockDevice{}
		defer mdev.AssertExpectations(t)

		mdev.On("Identifier").Return(SimpleIdentifier{id: "device"})
		mdev.On("Capabilities").Return([]da.Capability{capabilities.ProductInformationFlag})
		mdev.On("Gateway").Return(mgw)

		gm.On("Device", "device").Return(mdev, true)

		mhpi := &damocks.ProductInformation{}
		defer mhpi.AssertExpectations(t)

		mhpi.On("Name").Return("ProductInformation")

		mdev.On("Capability", capabilities.ProductInformationFlag).Return(mhpi)

		gm.On("GatewayName", mgw).Return("gwname", true)

		wem := websocketEventMapper{
			deviceOrganiser: &do,
			gatewayMapper:   gm,
			deviceExporter: &exporter.DeviceExporter{
				DeviceOrganiser: &do,
				GatewayMapper:   gm,
			},
		}

		actualData, err := wem.MapEvent(context.TODO(), state.DeviceAddedToZone{DeviceIdentifier: mdev.Identifier().String()})

		expectedData := [][]byte{
			[]byte(`{"Type":"DeviceUpdate","Metadata":{},"Identifier":"device","Capabilities":["ProductInformation"],"Gateway":"gwname"}`),
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedData, actualData)
	})

	t.Run("maps device removed from zone event", func(t *testing.T) {
		do := state.NewDeviceOrganiser(memory.New(), state.NullEventPublisher)
		gm := &state.MockGatewayMapper{}
		defer gm.AssertExpectations(t)

		mgw := &mocks.Gateway{}
		defer mgw.AssertExpectations(t)

		mdev := &mocks.MockDevice{}
		defer mdev.AssertExpectations(t)

		mdev.On("Identifier").Return(SimpleIdentifier{id: "device"})
		mdev.On("Capabilities").Return([]da.Capability{capabilities.ProductInformationFlag})
		mdev.On("Gateway").Return(mgw)

		gm.On("Device", "device").Return(mdev, true)

		mhpi := &damocks.ProductInformation{}
		defer mhpi.AssertExpectations(t)

		mhpi.On("Name").Return("ProductInformation")

		mdev.On("Capability", capabilities.ProductInformationFlag).Return(mhpi)

		gm.On("GatewayName", mgw).Return("gwname", true)

		wem := websocketEventMapper{
			deviceOrganiser: &do,
			gatewayMapper:   gm,
			deviceExporter: &exporter.DeviceExporter{
				DeviceOrganiser: &do,
				GatewayMapper:   gm,
			},
		}

		actualData, err := wem.MapEvent(context.TODO(), state.DeviceRemovedFromZone{DeviceIdentifier: mdev.Identifier().String()})

		expectedData := [][]byte{
			[]byte(`{"Type":"DeviceUpdate","Metadata":{},"Identifier":"device","Capabilities":["ProductInformation"],"Gateway":"gwname"}`),
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedData, actualData)
	})

	t.Run("maps creation of zone", func(t *testing.T) {
		wem := websocketEventMapper{}

		actualData, err := wem.MapEvent(context.TODO(), state.ZoneCreate{
			Identifier: 1,
			Name:       "one",
			AfterZone:  2,
		})

		expectedData := [][]byte{[]byte(`{"Type":"ZoneUpdate","Identifier":1,"Name":"one","Parent":0,"After":2}`)}

		assert.NoError(t, err)
		assert.Equal(t, expectedData, actualData)
	})

	t.Run("maps update of zone", func(t *testing.T) {
		wem := websocketEventMapper{}

		actualData, err := wem.MapEvent(context.TODO(), state.ZoneUpdate{
			Identifier: 1,
			Name:       "one",
			ParentZone: 10,
			AfterZone:  2,
		})

		expectedData := [][]byte{[]byte(`{"Type":"ZoneUpdate","Identifier":1,"Name":"one","Parent":10,"After":2}`)}

		assert.NoError(t, err)
		assert.Equal(t, expectedData, actualData)
	})

	t.Run("maps remove of zone", func(t *testing.T) {
		wem := websocketEventMapper{}

		actualData, err := wem.MapEvent(context.TODO(), state.ZoneRemove{
			Identifier: 1,
		})

		expectedData := [][]byte{[]byte(`{"Type":"ZoneRemove","Identifier":1}`)}

		assert.NoError(t, err)
		assert.Equal(t, expectedData, actualData)
	})

	t.Run("maps remove of device", func(t *testing.T) {
		wem := websocketEventMapper{}

		actualData, err := wem.MapEvent(context.TODO(), da.DeviceRemoved{
			Device: mocks.SimpleDevice{
				SIdentifier: SimpleIdentifier{id: "one"},
			},
		})

		expectedData := [][]byte{[]byte(`{"Type":"DeviceRemove","Identifier":"one"}`)}

		assert.NoError(t, err)
		assert.Equal(t, expectedData, actualData)
	})
}

func TestWebsocketEventMapper_InitialEvents(t *testing.T) {
	t.Run("returns slice of slice of bytes for messages describing a set of nested zones", func(t *testing.T) {
		do := state.NewDeviceOrganiser(memory.New(), state.NullEventPublisher)

		gm := &state.MockGatewayMapper{}
		defer gm.AssertExpectations(t)
		gm.On("Gateways").Return(map[string]da.Gateway{})

		r := do.NewZone("root")
		c := do.NewZone("child")
		c2 := do.NewZone("child2")
		do.MoveZone(c.Identifier, r.Identifier)
		do.MoveZone(c2.Identifier, r.Identifier)

		wem := websocketEventMapper{
			deviceOrganiser: &do,
			gatewayMapper:   gm,
		}

		expectedInitial := [][]byte{
			[]byte(`{"Type":"ZoneUpdate","Identifier":1,"Name":"root","Parent":0,"After":0}`),
			[]byte(`{"Type":"ZoneUpdate","Identifier":2,"Name":"child","Parent":1,"After":0}`),
			[]byte(`{"Type":"ZoneUpdate","Identifier":3,"Name":"child2","Parent":1,"After":2}`),
		}

		actualInitial, err := wem.InitialEvents(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, expectedInitial, actualInitial)
	})

	t.Run("returns slice of slice of bytes for messages describing a set of root zones", func(t *testing.T) {
		do := state.NewDeviceOrganiser(memory.New(), state.NullEventPublisher)

		gm := &state.MockGatewayMapper{}
		defer gm.AssertExpectations(t)
		gm.On("Gateways").Return(map[string]da.Gateway{})

		_ = do.NewZone("a")
		_ = do.NewZone("b")

		wem := websocketEventMapper{
			deviceOrganiser: &do,
			gatewayMapper:   gm,
		}

		expectedInitial := [][]byte{
			[]byte(`{"Type":"ZoneUpdate","Identifier":1,"Name":"a","Parent":0,"After":0}`),
			[]byte(`{"Type":"ZoneUpdate","Identifier":2,"Name":"b","Parent":0,"After":1}`),
		}

		actualInitial, err := wem.InitialEvents(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, expectedInitial, actualInitial)
	})

	t.Run("returns a gateway, with one device with a capability inside a zone", func(t *testing.T) {
		do := state.NewDeviceOrganiser(memory.New(), state.NullEventPublisher)
		gm := &state.MockGatewayMapper{}
		defer gm.AssertExpectations(t)

		do.NewZone("root")
		do.AddDevice("device")
		do.NameDevice("device", "device name")
		do.AddDeviceToZone("device", 1)

		mgw := &mocks.Gateway{}
		defer mgw.AssertExpectations(t)

		mdev := &mocks.MockDevice{}
		defer mdev.AssertExpectations(t)

		mdev.On("Identifier").Return(SimpleIdentifier{id: "device"})
		mdev.On("Capabilities").Return([]da.Capability{capabilities.ProductInformationFlag})
		mdev.On("Gateway").Return(mgw)

		mgw.On("Devices").Return([]da.Device{mdev})
		mgw.On("Capabilities").Return([]da.Capability{capabilities.ProductInformationFlag})
		mgw.On("Self").Return(mocks.SimpleDevice{SIdentifier: SimpleIdentifier{"selfdevice"}})

		mhpi := &damocks.ProductInformation{}
		defer mhpi.AssertExpectations(t)

		mhpi.On("Name").Return("ProductInformation")
		mhpi.On("Get", mock.Anything).Return(capabilities.ProductInfo{
			Manufacturer: "Manufacturer",
			Name:         "Name",
		}, nil)

		mdev.On("Capability", capabilities.ProductInformationFlag).Return(mhpi)

		gm.On("GatewayName", mgw).Return("gwname", true)
		gm.On("Gateways").Return(map[string]da.Gateway{"gwname": mgw})

		wem := websocketEventMapper{
			deviceOrganiser: &do,
			gatewayMapper:   gm,
			deviceExporter: &exporter.DeviceExporter{
				DeviceOrganiser: &do,
				GatewayMapper:   gm,
			},
		}

		expectedInitial := [][]byte{
			[]byte(`{"Type":"ZoneUpdate","Identifier":1,"Name":"root","Parent":0,"After":0}`),
			[]byte(`{"Type":"GatewayUpdate","Identifier":"gwname","Capabilities":["ProductInformation"],"SelfDevice":"selfdevice"}`),
			[]byte(`{"Type":"DeviceUpdate","Metadata":{"Name":"device name","Zones":[1]},"Identifier":"device","Capabilities":["ProductInformation"],"Gateway":"gwname"}`),
			[]byte(`{"Type":"DeviceUpdateCapability","Identifier":"device","Capability":"ProductInformation","Payload":{"Name":"Name","Manufacturer":"Manufacturer"}}`),
		}

		actualInitial, err := wem.InitialEvents(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, expectedInitial, actualInitial)
	})
}
