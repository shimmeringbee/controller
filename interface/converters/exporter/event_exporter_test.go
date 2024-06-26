package exporter

import (
	"context"
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

func TestEventExporter_MapEvent(t *testing.T) {
	t.Run("maps an event from a capability of a device", func(t *testing.T) {
		do := state.NewDeviceOrganiser(memory.New(), state.NullEventPublisher)

		gm := &state.MockGatewayMapper{}
		defer gm.AssertExpectations(t)

		wem := eventExporter{
			deviceOrganiser: &do,
			gatewayMapper:   gm,
			deviceExporter: &deviceExporter{
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

		expectedInitial := []any{
			DeviceUpdateCapabilityMessage{
				DeviceMessage: DeviceMessage{
					Message: Message{
						Type: "DeviceUpdateCapability",
					},
				},
				Identifier: "device",
				Capability: "OnOff",
				Payload: &OnOff{
					State: true,
				},
			},
		}

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

		wem := eventExporter{
			deviceOrganiser: &do,
			gatewayMapper:   gm,
			deviceExporter: &deviceExporter{
				DeviceOrganiser: &do,
				GatewayMapper:   gm,
			},
		}

		actualData, err := wem.MapEvent(context.TODO(), da.DeviceAdded{Device: mdev})

		expectedData := []any{
			DeviceUpdateMessage{
				DeviceMessage: DeviceMessage{
					Message{
						Type: DeviceUpdateMessageName,
					},
				},
				ExportedSimpleDevice: ExportedSimpleDevice{
					Metadata:     state.DeviceMetadata{},
					Identifier:   "device",
					Capabilities: []string{"ProductInformation"},
					Gateway:      "gwname",
				},
			},
			DeviceUpdateCapabilityMessage{
				DeviceMessage: DeviceMessage{
					Message: Message{
						Type: DeviceUpdateCapabilityMessageName,
					},
				},
				Identifier: "device",
				Capability: "ProductInformation",
				Payload: &ProductInformation{
					Name:         "Name",
					Manufacturer: "Manufacturer",
				},
			}}

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

		wem := eventExporter{
			deviceOrganiser: &do,
			gatewayMapper:   gm,
			deviceExporter: &deviceExporter{
				DeviceOrganiser: &do,
				GatewayMapper:   gm,
			},
		}

		actualData, err := wem.MapEvent(context.TODO(), capabilities.EnumerateDeviceStopped{Device: mdev})

		expectedData := []any{
			DeviceUpdateMessage{
				DeviceMessage: DeviceMessage{
					Message{
						Type: DeviceUpdateMessageName,
					},
				},
				ExportedSimpleDevice: ExportedSimpleDevice{
					Metadata:     state.DeviceMetadata{},
					Identifier:   "device",
					Capabilities: []string{"ProductInformation"},
					Gateway:      "gwname",
				},
			},
			DeviceUpdateCapabilityMessage{
				DeviceMessage: DeviceMessage{
					Message: Message{
						Type: DeviceUpdateCapabilityMessageName,
					},
				},
				Identifier: "device",
				Capability: "ProductInformation",
				Payload: &ProductInformation{
					Name:         "Name",
					Manufacturer: "Manufacturer",
				},
			}}

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

		wem := eventExporter{
			deviceOrganiser: &do,
			gatewayMapper:   gm,
			deviceExporter: &deviceExporter{
				DeviceOrganiser: &do,
				GatewayMapper:   gm,
			},
		}

		actualData, err := wem.MapEvent(context.TODO(), state.DeviceMetadataUpdate{Identifier: mdev.Identifier().String()})

		expectedData := []any{
			DeviceUpdateMessage{
				DeviceMessage: DeviceMessage{
					Message{
						Type: DeviceUpdateMessageName,
					},
				},
				ExportedSimpleDevice: ExportedSimpleDevice{
					Metadata:     state.DeviceMetadata{},
					Identifier:   "device",
					Capabilities: []string{"ProductInformation"},
					Gateway:      "gwname",
				},
			},
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

		wem := eventExporter{
			deviceOrganiser: &do,
			gatewayMapper:   gm,
			deviceExporter: &deviceExporter{
				DeviceOrganiser: &do,
				GatewayMapper:   gm,
			},
		}

		actualData, err := wem.MapEvent(context.TODO(), state.DeviceAddedToZone{DeviceIdentifier: mdev.Identifier().String()})

		expectedData := []any{
			DeviceUpdateMessage{
				DeviceMessage: DeviceMessage{
					Message{
						Type: DeviceUpdateMessageName,
					},
				},
				ExportedSimpleDevice: ExportedSimpleDevice{
					Metadata:     state.DeviceMetadata{},
					Identifier:   "device",
					Capabilities: []string{"ProductInformation"},
					Gateway:      "gwname",
				},
			},
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

		wem := eventExporter{
			deviceOrganiser: &do,
			gatewayMapper:   gm,
			deviceExporter: &deviceExporter{
				DeviceOrganiser: &do,
				GatewayMapper:   gm,
			},
		}

		actualData, err := wem.MapEvent(context.TODO(), state.DeviceRemovedFromZone{DeviceIdentifier: mdev.Identifier().String()})

		expectedData := []any{
			DeviceUpdateMessage{
				DeviceMessage: DeviceMessage{
					Message{
						Type: DeviceUpdateMessageName,
					},
				},
				ExportedSimpleDevice: ExportedSimpleDevice{
					Metadata:     state.DeviceMetadata{},
					Identifier:   "device",
					Capabilities: []string{"ProductInformation"},
					Gateway:      "gwname",
				},
			},
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedData, actualData)
	})

	t.Run("maps creation of zone", func(t *testing.T) {
		wem := eventExporter{}

		actualData, err := wem.MapEvent(context.TODO(), state.ZoneCreate{
			Identifier: 1,
			Name:       "one",
			AfterZone:  2,
		})

		expectedData := []any{
			ZoneUpdateMessage{
				ZoneMessage: ZoneMessage{
					Message: Message{
						Type: ZoneUpdateMessageName,
					},
					Identifier: 1,
				},
				Name:   "one",
				Parent: 0,
				After:  2,
			},
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedData, actualData)
	})

	t.Run("maps update of zone", func(t *testing.T) {
		wem := eventExporter{}

		actualData, err := wem.MapEvent(context.TODO(), state.ZoneUpdate{
			Identifier: 1,
			Name:       "one",
			ParentZone: 10,
			AfterZone:  2,
		})

		expectedData := []any{
			ZoneUpdateMessage{
				ZoneMessage: ZoneMessage{
					Message: Message{
						Type: ZoneUpdateMessageName,
					},
					Identifier: 1,
				},
				Name:   "one",
				Parent: 10,
				After:  2,
			},
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedData, actualData)
	})

	t.Run("maps remove of zone", func(t *testing.T) {
		wem := eventExporter{}

		actualData, err := wem.MapEvent(context.TODO(), state.ZoneRemove{
			Identifier: 1,
		})

		expectedData := []any{
			ZoneRemoveMessage{
				ZoneMessage: ZoneMessage{
					Message: Message{
						Type: ZoneRemoveMessageName,
					},
					Identifier: 1,
				},
			},
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedData, actualData)
	})

	t.Run("maps remove of device", func(t *testing.T) {
		wem := eventExporter{}

		actualData, err := wem.MapEvent(context.TODO(), da.DeviceRemoved{
			Device: mocks.SimpleDevice{
				SIdentifier: SimpleIdentifier{id: "one"},
			},
		})

		expectedData := []any{
			DeviceRemoveMessage{
				DeviceMessage: DeviceMessage{
					Message: Message{
						Type: DeviceRemoveMessageName,
					},
				},
				Identifier: "one",
			},
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedData, actualData)
	})
}

func TestEventExporter_InitialEvents(t *testing.T) {
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

		wem := eventExporter{
			deviceOrganiser: &do,
			gatewayMapper:   gm,
		}

		expectedInitial := []any{
			ZoneUpdateMessage{
				ZoneMessage: ZoneMessage{
					Message: Message{
						Type: ZoneUpdateMessageName,
					},
					Identifier: 1,
				},
				Name:   "root",
				Parent: 0,
				After:  0,
			},
			ZoneUpdateMessage{
				ZoneMessage: ZoneMessage{
					Message: Message{
						Type: ZoneUpdateMessageName,
					},
					Identifier: 2,
				},
				Name:   "child",
				Parent: 1,
				After:  0,
			},
			ZoneUpdateMessage{
				ZoneMessage: ZoneMessage{
					Message: Message{
						Type: ZoneUpdateMessageName,
					},
					Identifier: 3,
				},
				Name:   "child2",
				Parent: 1,
				After:  2,
			},
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

		wem := eventExporter{
			deviceOrganiser: &do,
			gatewayMapper:   gm,
		}

		expectedInitial := []any{
			ZoneUpdateMessage{
				ZoneMessage: ZoneMessage{
					Message: Message{
						Type: ZoneUpdateMessageName,
					},
					Identifier: 1,
				},
				Name:   "a",
				Parent: 0,
				After:  0,
			},
			ZoneUpdateMessage{
				ZoneMessage: ZoneMessage{
					Message: Message{
						Type: ZoneUpdateMessageName,
					},
					Identifier: 2,
				},
				Name:   "b",
				Parent: 0,
				After:  1,
			},
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

		wem := eventExporter{
			deviceOrganiser: &do,
			gatewayMapper:   gm,
			deviceExporter: &deviceExporter{
				DeviceOrganiser: &do,
				GatewayMapper:   gm,
			},
		}

		expectedInitial := []any{
			ZoneUpdateMessage{
				ZoneMessage: ZoneMessage{
					Message: Message{
						Type: ZoneUpdateMessageName,
					},
					Identifier: 1,
				},
				Name:   "root",
				Parent: 0,
				After:  0,
			},
			GatewayUpdateMessage{
				GatewayMessage: GatewayMessage{
					Message: Message{
						Type: GatewayUpdateMessageName,
					},
				},
				ExportedGateway: ExportedGateway{
					Identifier:   "gwname",
					Capabilities: []string{"ProductInformation"},
					SelfDevice:   "selfdevice",
				},
			},
			DeviceUpdateMessage{
				DeviceMessage: DeviceMessage{
					Message{
						Type: DeviceUpdateMessageName,
					},
				},
				ExportedSimpleDevice: ExportedSimpleDevice{
					Metadata: state.DeviceMetadata{
						Name:  "device name",
						Zones: []int{1},
					},
					Identifier:   "device",
					Capabilities: []string{"ProductInformation"},
					Gateway:      "gwname",
				},
			},
			DeviceUpdateCapabilityMessage{
				DeviceMessage: DeviceMessage{
					Message: Message{
						Type: DeviceUpdateCapabilityMessageName,
					},
				},
				Identifier: "device",
				Capability: "ProductInformation",
				Payload: &ProductInformation{
					Name:         "Name",
					Manufacturer: "Manufacturer",
				},
			},
		}

		actualInitial, err := wem.InitialEvents(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, expectedInitial, actualInitial)
	})
}
