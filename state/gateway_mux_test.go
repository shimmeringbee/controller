package state

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	mocks2 "github.com/shimmeringbee/da/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestGatewayMux_Add(t *testing.T) {
	t.Run("added gateway is available via Gateways() list", func(t *testing.T) {
		mg := &mocks2.Gateway{}
		mg.On("ReadEvent", mock.Anything).Return(nil, context.DeadlineExceeded).Maybe()

		d := mocks2.SimpleDevice{SIdentifier: zigbee.GenerateLocalAdministeredIEEEAddress()}
		mg.On("Self").Return(d)

		defer mg.AssertExpectations(t)

		name := "mock"

		m := GatewayMux{gatewayByName: map[string]da.Gateway{}, deviceByIdentifier: map[string]da.Device{}}
		m.Add(name, mg)
		defer m.Stop()

		expectedGws := map[string]da.Gateway{name: mg}
		actualGws := m.Gateways()

		assert.Equal(t, expectedGws, actualGws)
		assert.Contains(t, m.deviceByIdentifier, d.Identifier().String())

		gatewayName, found := m.GatewayName(mg)
		assert.True(t, found)
		assert.Equal(t, name, gatewayName)
	})

	t.Run("announced devices are added to the gateway mux cache for routing", func(t *testing.T) {
		mg := &mocks2.Gateway{}

		d := mocks2.SimpleDevice{
			SGateway:    mg,
			SIdentifier: zigbee.GenerateLocalAdministeredIEEEAddress(),
		}

		mg.On("ReadEvent", mock.Anything).Return(da.DeviceAdded{Device: d}, nil).Once()
		mg.On("ReadEvent", mock.Anything).Return(nil, context.DeadlineExceeded).Maybe()
		defer mg.AssertExpectations(t)

		mep := mockEventPublisher{}
		mep.On("Publish", mock.Anything)
		defer mep.AssertExpectations(t)

		selfD := mocks2.SimpleDevice{SIdentifier: zigbee.GenerateLocalAdministeredIEEEAddress()}
		mg.On("Self").Return(selfD)

		name := "mock"

		m := GatewayMux{gatewayByName: map[string]da.Gateway{}, deviceByIdentifier: map[string]da.Device{}, eventPublisher: &mep}
		m.Add(name, mg)
		time.Sleep(50 * time.Millisecond)
		m.Stop()

		actualGw := m.deviceByIdentifier[d.Identifier().String()].Gateway()
		assert.Equal(t, mg, actualGw)

		foundDevice, found := m.Device(d.Identifier().String())
		assert.True(t, found)
		assert.Equal(t, d, foundDevice)
	})

	t.Run("DeviceLoaded update the gateway mux cache for routing", func(t *testing.T) {
		mg := &mocks2.Gateway{}

		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		d1 := mocks2.SimpleDevice{
			SGateway:      mg,
			SIdentifier:   addr,
			SCapabilities: []da.Capability{},
		}

		capOne := da.Capability(0x01)

		d2 := mocks2.SimpleDevice{
			SGateway:      mg,
			SIdentifier:   addr,
			SCapabilities: []da.Capability{capOne},
		}

		mg.On("ReadEvent", mock.Anything).Return(da.DeviceAdded{Device: d1}, nil).Once()
		mg.On("ReadEvent", mock.Anything).Return(da.DeviceAdded{Device: d2}, nil).Once()
		mg.On("ReadEvent", mock.Anything).Return(nil, context.DeadlineExceeded).Maybe()
		defer mg.AssertExpectations(t)

		mep := mockEventPublisher{}
		mep.On("Publish", mock.Anything)
		defer mep.AssertExpectations(t)

		selfD := mocks2.SimpleDevice{SIdentifier: zigbee.GenerateLocalAdministeredIEEEAddress()}
		mg.On("Self").Return(selfD)

		name := "mock"

		m := GatewayMux{gatewayByName: map[string]da.Gateway{}, deviceByIdentifier: map[string]da.Device{}, eventPublisher: &mep}
		m.Add(name, mg)
		time.Sleep(50 * time.Millisecond)
		m.Stop()

		actualGw := m.deviceByIdentifier[d1.Identifier().String()].Gateway()
		assert.Equal(t, mg, actualGw)

		assert.Equal(t, []da.Capability{capOne}, m.deviceByIdentifier[d1.Identifier().String()].Capabilities())
	})

	t.Run("EnumerateDeviceStopped update the gateway mux cache for routing", func(t *testing.T) {
		mg := &mocks2.Gateway{}

		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		d1 := mocks2.SimpleDevice{
			SGateway:      mg,
			SIdentifier:   addr,
			SCapabilities: []da.Capability{},
		}

		capOne := da.Capability(0x01)

		d2 := mocks2.SimpleDevice{
			SGateway:      mg,
			SIdentifier:   addr,
			SCapabilities: []da.Capability{capOne},
		}

		mg.On("ReadEvent", mock.Anything).Return(da.DeviceAdded{Device: d1}, nil).Once()
		mg.On("ReadEvent", mock.Anything).Return(capabilities.EnumerateDeviceStopped{Device: d2}, nil).Once()
		mg.On("ReadEvent", mock.Anything).Return(nil, context.DeadlineExceeded).Maybe()
		defer mg.AssertExpectations(t)

		mep := mockEventPublisher{}
		mep.On("Publish", mock.Anything)
		defer mep.AssertExpectations(t)

		selfD := mocks2.SimpleDevice{SIdentifier: zigbee.GenerateLocalAdministeredIEEEAddress()}
		mg.On("Self").Return(selfD)

		name := "mock"

		m := GatewayMux{gatewayByName: map[string]da.Gateway{}, deviceByIdentifier: map[string]da.Device{}, eventPublisher: &mep}
		m.Add(name, mg)
		time.Sleep(50 * time.Millisecond)
		m.Stop()

		actualGw := m.deviceByIdentifier[d1.Identifier().String()].Gateway()
		assert.Equal(t, mg, actualGw)

		assert.Equal(t, []da.Capability{capOne}, m.deviceByIdentifier[d1.Identifier().String()].Capabilities())
	})

	t.Run("announced devices are added and then removed from the gateway mux cache for routing", func(t *testing.T) {
		mg := &mocks2.Gateway{}

		d := mocks2.SimpleDevice{
			SGateway:      mg,
			SIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SCapabilities: []da.Capability{},
		}

		mg.On("ReadEvent", mock.Anything).Return(da.DeviceAdded{Device: d}, nil).Once()
		mg.On("ReadEvent", mock.Anything).Return(da.DeviceRemoved{Device: d}, nil).Once()
		mg.On("ReadEvent", mock.Anything).Return(nil, context.DeadlineExceeded).Maybe()
		defer mg.AssertExpectations(t)

		mep := mockEventPublisher{}
		mep.On("Publish", mock.Anything)
		defer mep.AssertExpectations(t)

		selfD := mocks2.SimpleDevice{SIdentifier: zigbee.GenerateLocalAdministeredIEEEAddress()}
		mg.On("Self").Return(selfD)

		name := "mock"

		m := GatewayMux{gatewayByName: map[string]da.Gateway{}, deviceByIdentifier: map[string]da.Device{}, eventPublisher: &mep}
		m.Add(name, mg)
		time.Sleep(50 * time.Millisecond)
		m.Stop()

		_, found := m.deviceByIdentifier[d.Identifier().String()]
		assert.False(t, found)
	})
}

type mockEventPublisher struct {
	mock.Mock
}

func (m *mockEventPublisher) Publish(e interface{}) {
	m.Called(e)
}
