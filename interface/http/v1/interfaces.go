package v1

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/stretchr/testify/mock"
)

type mockGatewayMapper struct {
	mock.Mock
}

func (m *mockGatewayMapper) Gateways() map[string]da.Gateway {
	args := m.Called()
	return args.Get(0).(map[string]da.Gateway)
}

func (m *mockGatewayMapper) Capability(id string, cap da.Capability) interface{} {
	args := m.Called(id, cap)
	return args.Get(0)
}

func (m *mockGatewayMapper) Device(id string) (da.Device, bool) {
	args := m.Called(id)
	return args.Get(0).(da.Device), args.Bool(1)
}

func (m *mockGatewayMapper) GatewayName(gw da.Gateway) (string, bool) {
	args := m.Called(gw)
	return args.String(0), args.Bool(1)
}

type mockGateway struct {
	mock.Mock
}

func (m *mockGateway) ReadEvent(ctx context.Context) (interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0), args.Error(1)
}

func (m *mockGateway) Capability(c da.Capability) interface{} {
	args := m.Called(c)
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

type mockBasicCapability struct {
	mock.Mock
}

func (m *mockBasicCapability) Capability() da.Capability {
	args := m.Called()
	return args.Get(0).(da.Capability)
}

func (m *mockBasicCapability) Name() string {
	args := m.Called()
	return args.String(0)
}
