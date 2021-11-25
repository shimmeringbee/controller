package gateway

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestGatewayMux_Add(t *testing.T) {
	t.Run("added gateway is available via Gateways() list", func(t *testing.T) {
		mg := mockGateway{}
		mg.On("ReadEvent", mock.Anything).Return(nil, context.DeadlineExceeded).Maybe()

		d := da.BaseDevice{DeviceIdentifier: zigbee.GenerateLocalAdministeredIEEEAddress()}
		mg.On("Self").Return(d)

		defer mg.AssertExpectations(t)

		name := "mock"

		m := Mux{gatewayByName: map[string]da.Gateway{}, deviceByIdentifier: map[string]da.Device{}}
		m.Add(name, &mg)
		defer m.Stop()

		expectedGws := map[string]da.Gateway{name: &mg}
		actualGws := m.Gateways()

		assert.Equal(t, expectedGws, actualGws)
		assert.Contains(t, m.deviceByIdentifier, d.DeviceIdentifier.String())

		gatewayName, found := m.GatewayName(&mg)
		assert.True(t, found)
		assert.Equal(t, name, gatewayName)
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

		mep := mockEventPublisher{}
		mep.On("Publish", mock.Anything)
		defer mep.AssertExpectations(t)

		selfD := da.BaseDevice{DeviceIdentifier: zigbee.GenerateLocalAdministeredIEEEAddress()}
		mg.On("Self").Return(selfD)

		name := "mock"

		m := Mux{gatewayByName: map[string]da.Gateway{}, deviceByIdentifier: map[string]da.Device{}, eventPublisher: &mep}
		m.Add(name, &mg)
		time.Sleep(50 * time.Millisecond)
		m.Stop()

		actualGw := m.deviceByIdentifier[d.Identifier().String()].Gateway()
		assert.Equal(t, &mg, actualGw)

		foundDevice, found := m.Device(d.Identifier().String())
		assert.True(t, found)
		assert.Equal(t, d, foundDevice)
	})

	t.Run("DeviceLoaded update the gateway mux cache for routing", func(t *testing.T) {
		mg := mockGateway{}

		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		d1 := da.BaseDevice{
			DeviceGateway:      &mg,
			DeviceIdentifier:   addr,
			DeviceCapabilities: []da.Capability{},
		}

		capOne := da.Capability(0x01)

		d2 := da.BaseDevice{
			DeviceGateway:      &mg,
			DeviceIdentifier:   addr,
			DeviceCapabilities: []da.Capability{capOne},
		}

		mg.On("ReadEvent", mock.Anything).Return(da.DeviceAdded{Device: d1}, nil).Once()
		mg.On("ReadEvent", mock.Anything).Return(da.DeviceLoaded{Device: d2}, nil).Once()
		mg.On("ReadEvent", mock.Anything).Return(nil, context.DeadlineExceeded).Maybe()
		defer mg.AssertExpectations(t)

		mep := mockEventPublisher{}
		mep.On("Publish", mock.Anything)
		defer mep.AssertExpectations(t)

		selfD := da.BaseDevice{DeviceIdentifier: zigbee.GenerateLocalAdministeredIEEEAddress()}
		mg.On("Self").Return(selfD)

		name := "mock"

		m := Mux{gatewayByName: map[string]da.Gateway{}, deviceByIdentifier: map[string]da.Device{}, eventPublisher: &mep}
		m.Add(name, &mg)
		time.Sleep(50 * time.Millisecond)
		m.Stop()

		actualGw := m.deviceByIdentifier[d1.Identifier().String()].Gateway()
		assert.Equal(t, &mg, actualGw)

		assert.Equal(t, []da.Capability{capOne}, m.deviceByIdentifier[d1.Identifier().String()].Capabilities())
	})

	t.Run("EnumerateDeviceSuccess update the gateway mux cache for routing", func(t *testing.T) {
		mg := mockGateway{}

		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		d1 := da.BaseDevice{
			DeviceGateway:      &mg,
			DeviceIdentifier:   addr,
			DeviceCapabilities: []da.Capability{},
		}

		capOne := da.Capability(0x01)

		d2 := da.BaseDevice{
			DeviceGateway:      &mg,
			DeviceIdentifier:   addr,
			DeviceCapabilities: []da.Capability{capOne},
		}

		mg.On("ReadEvent", mock.Anything).Return(da.DeviceAdded{Device: d1}, nil).Once()
		mg.On("ReadEvent", mock.Anything).Return(capabilities.EnumerateDeviceSuccess{Device: d2}, nil).Once()
		mg.On("ReadEvent", mock.Anything).Return(nil, context.DeadlineExceeded).Maybe()
		defer mg.AssertExpectations(t)

		mep := mockEventPublisher{}
		mep.On("Publish", mock.Anything)
		defer mep.AssertExpectations(t)

		selfD := da.BaseDevice{DeviceIdentifier: zigbee.GenerateLocalAdministeredIEEEAddress()}
		mg.On("Self").Return(selfD)

		name := "mock"

		m := Mux{gatewayByName: map[string]da.Gateway{}, deviceByIdentifier: map[string]da.Device{}, eventPublisher: &mep}
		m.Add(name, &mg)
		time.Sleep(50 * time.Millisecond)
		m.Stop()

		actualGw := m.deviceByIdentifier[d1.Identifier().String()].Gateway()
		assert.Equal(t, &mg, actualGw)

		assert.Equal(t, []da.Capability{capOne}, m.deviceByIdentifier[d1.Identifier().String()].Capabilities())
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

		mep := mockEventPublisher{}
		mep.On("Publish", mock.Anything)
		defer mep.AssertExpectations(t)

		selfD := da.BaseDevice{DeviceIdentifier: zigbee.GenerateLocalAdministeredIEEEAddress()}
		mg.On("Self").Return(selfD)

		name := "mock"

		m := Mux{gatewayByName: map[string]da.Gateway{}, deviceByIdentifier: map[string]da.Device{}, eventPublisher: &mep}
		m.Add(name, &mg)
		time.Sleep(50 * time.Millisecond)
		m.Stop()

		_, found := m.deviceByIdentifier[d.Identifier().String()]
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

		mep := mockEventPublisher{}
		mep.On("Publish", mock.Anything)
		defer mep.AssertExpectations(t)

		selfD := da.BaseDevice{DeviceIdentifier: zigbee.GenerateLocalAdministeredIEEEAddress()}
		mg.On("Self").Return(selfD)

		name := "mock"

		m := Mux{gatewayByName: map[string]da.Gateway{}, deviceByIdentifier: map[string]da.Device{}, eventPublisher: &mep}
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

		time.Sleep(1 * time.Millisecond)
		selfD := da.BaseDevice{DeviceGateway: &mg, DeviceIdentifier: zigbee.GenerateLocalAdministeredIEEEAddress()}
		mg.On("Self").Return(selfD)

		name := "mock"

		m := Mux{gatewayByName: map[string]da.Gateway{}, deviceByIdentifier: map[string]da.Device{}}
		m.Add(name, &mg)
		time.Sleep(50 * time.Millisecond)
		m.Stop()

		actualCapability := m.Capability(d.Identifier().String(), capability)
		assert.Nil(t, actualCapability)
	})
}

type mockEventPublisher struct {
	mock.Mock
}

func (m *mockEventPublisher) Publish(e interface{}) {
	m.Called(e)
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

func (m *mockGateway) Capabilities() []da.Capability {
	args := m.Called()
	return args.Get(0).([]da.Capability)
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
