package exporter

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/stretchr/testify/mock"
)

type MockDeviceExporter struct {
	mock.Mock
}

func (m *MockDeviceExporter) ExportDevice(ctx context.Context, daDevice da.Device) ExportedDevice {
	args := m.Called(ctx, daDevice)
	return args.Get(0).(ExportedDevice)
}

func (m *MockDeviceExporter) ExportSimpleDevice(ctx context.Context, daDevice da.Device) ExportedSimpleDevice {
	args := m.Called(ctx, daDevice)
	return args.Get(0).(ExportedSimpleDevice)
}

func (m *MockDeviceExporter) ExportCapability(ctx context.Context, e interface{}) interface{} {
	args := m.Called(ctx, e)
	return args.Get(0)
}

type MockGatewayExporter struct {
	mock.Mock
}

func (m *MockGatewayExporter) ConvertDAGatewayToGateway(daGateway da.Gateway) ExportedGateway {
	args := m.Called(daGateway)
	return args.Get(0).(ExportedGateway)
}
