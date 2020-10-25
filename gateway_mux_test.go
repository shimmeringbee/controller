package main

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestGatewayMux_ListenAndSend(t *testing.T) {
	t.Run("listener channels that are registered are sent events", func(t *testing.T) {
		listenCh := make(chan interface{}, 1)
		expectedEvent := struct{}{}

		m := GatewayMux{}
		m.Listen(listenCh)
		m.sendToListeners(expectedEvent)

		select {
		case actualEvent := <-listenCh:
			assert.Equal(t, expectedEvent, actualEvent)
		default:
			assert.Fail(t, "no event received")
		}
	})
}

func TestGatewayMux_Add(t *testing.T) {
	t.Run("added gateway is available via Gateways() list", func(t *testing.T) {
		mg := mockGateway{}
		mg.On("ReadEvent", mock.Anything).Return(nil, context.DeadlineExceeded).Maybe()

		defer mg.AssertExpectations(t)

		name := "mock"

		m := GatewayMux{gatewayByName: map[string]da.Gateway{}}
		m.Add(name, &mg)
		defer m.Stop()

		expectedGws := map[string]da.Gateway{name: &mg}
		actualGws := m.Gateways()

		assert.Equal(t, expectedGws, actualGws)
	})

	t.Run("announced devices are added to the gateway mux cache for routing", func(t *testing.T) {
		mg := mockGateway{}

		d := da.BaseDevice{
			DeviceGateway:      &mg,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{},
		}

		mg.On("ReadEvent", mock.Anything).Return(da.DeviceAdded{Device: d}, nil).Once()
		mg.On("ReadEvent", mock.Anything).Return(nil, context.DeadlineExceeded).Maybe()
		defer mg.AssertExpectations(t)

		name := "mock"

		m := GatewayMux{gatewayByName: map[string]da.Gateway{}, gatewayByIdentifier: map[string]da.Gateway{}}
		m.Add(name, &mg)
		time.Sleep(50 * time.Millisecond)
		m.Stop()

		actualGw := m.gatewayByIdentifier[d.Identifier().String()]
		assert.Equal(t, &mg, actualGw)
	})

	t.Run("announced devices are added and then removed from the gateway mux cache for routing", func(t *testing.T) {
		mg := mockGateway{}

		d := da.BaseDevice{
			DeviceGateway:      &mg,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{},
		}

		mg.On("ReadEvent", mock.Anything).Return(da.DeviceAdded{Device: d}, nil).Once()
		mg.On("ReadEvent", mock.Anything).Return(da.DeviceRemoved{Device: d}, nil).Once()
		mg.On("ReadEvent", mock.Anything).Return(nil, context.DeadlineExceeded).Maybe()
		defer mg.AssertExpectations(t)

		name := "mock"

		m := GatewayMux{gatewayByName: map[string]da.Gateway{}, gatewayByIdentifier: map[string]da.Gateway{}}
		m.Add(name, &mg)
		time.Sleep(50 * time.Millisecond)
		m.Stop()

		_, found := m.gatewayByIdentifier[d.Identifier().String()]
		assert.False(t, found)
	})
}

func TestGatewayMux_Capability(t *testing.T) {
	t.Run("capabilities are retrieved from the devices owning gateway", func(t *testing.T) {
		mg := mockGateway{}

		d := da.BaseDevice{
			DeviceGateway:      &mg,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{},
		}

		capability := da.Capability(1)
		expectedCapability := struct{}{}

		mg.On("ReadEvent", mock.Anything).Return(da.DeviceAdded{Device: d}, nil).Once()
		mg.On("ReadEvent", mock.Anything).Return(nil, context.DeadlineExceeded).Maybe()
		defer mg.AssertExpectations(t)
		mg.On("Capability", capability).Return(expectedCapability)

		name := "mock"

		m := GatewayMux{gatewayByName: map[string]da.Gateway{}, gatewayByIdentifier: map[string]da.Gateway{}}
		m.Add(name, &mg)
		time.Sleep(50 * time.Millisecond)
		m.Stop()

		actualCapability := m.Capability(d.Identifier().String(), capability)
		assert.Equal(t, expectedCapability, actualCapability)
	})

	t.Run("capabilities are returns nil if gateway is unknown", func(t *testing.T) {
		mg := mockGateway{}

		d := da.BaseDevice{
			DeviceGateway:      &mg,
			DeviceIdentifier:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			DeviceCapabilities: []da.Capability{},
		}

		capability := da.Capability(1)

		mg.On("ReadEvent", mock.Anything).Return(nil, context.DeadlineExceeded).Maybe()
		defer mg.AssertExpectations(t)

		name := "mock"

		m := GatewayMux{gatewayByName: map[string]da.Gateway{}, gatewayByIdentifier: map[string]da.Gateway{}}
		m.Add(name, &mg)
		time.Sleep(50 * time.Millisecond)
		m.Stop()

		actualCapability := m.Capability(d.Identifier().String(), capability)
		assert.Nil(t, actualCapability)
	})
}

type mockGateway struct {
	mock.Mock
}

func (m *mockGateway) ReadEvent(ctx context.Context) (interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0), args.Error(1)
}

func (m *mockGateway) Capability(capability da.Capability) interface{} {
	args := m.Called(capability)
	return args.Get(0)
}

func (m *mockGateway) Self() da.Device {
	args := m.Called()
	return args.Get(0).(da.Device)
}

func (m *mockGateway) Devices() []da.Device {
	args := m.Called()
	return args.Get(0).([]da.Device)
}

func (m *mockGateway) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockGateway) Stop() error {
	args := m.Called()
	return args.Error(0)
}
