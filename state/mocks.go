package state

import (
	"github.com/shimmeringbee/da"
	"github.com/stretchr/testify/mock"
)

var _ GatewayMapper = (*MockGatewayMapper)(nil)

type MockGatewayMapper struct {
	mock.Mock
}

func (m *MockGatewayMapper) Gateways() map[string]da.Gateway {
	args := m.Called()
	return args.Get(0).(map[string]da.Gateway)
}

func (m *MockGatewayMapper) Capability(id string, cap da.Capability) any {
	args := m.Called(id, cap)
	return args.Get(0)
}

func (m *MockGatewayMapper) Device(id string) (da.Device, bool) {
	args := m.Called(id)
	return args.Get(0).(da.Device), args.Bool(1)
}

func (m *MockGatewayMapper) GatewayName(gw da.Gateway) (string, bool) {
	args := m.Called(gw)
	return args.String(0), args.Bool(1)
}
