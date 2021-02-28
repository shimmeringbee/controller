package gateway

import (
	"github.com/shimmeringbee/da"
	"github.com/stretchr/testify/mock"
)

var _ Mapper = (*MockMapper)(nil)

type MockMapper struct {
	mock.Mock
}

func (m *MockMapper) Gateways() map[string]da.Gateway {
	args := m.Called()
	return args.Get(0).(map[string]da.Gateway)
}

func (m *MockMapper) Capability(id string, cap da.Capability) interface{} {
	args := m.Called(id, cap)
	return args.Get(0)
}

func (m *MockMapper) Device(id string) (da.Device, bool) {
	args := m.Called(id)
	return args.Get(0).(da.Device), args.Bool(1)
}

func (m *MockMapper) GatewayName(gw da.Gateway) (string, bool) {
	args := m.Called(gw)
	return args.String(0), args.Bool(1)
}
