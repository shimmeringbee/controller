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

type MockGatewayExporter struct {
	mock.Mock
}

func (m *MockGatewayExporter) ConvertDAGatewayToGateway(daGateway da.Gateway) ExportedGateway {
	args := m.Called(daGateway)
	return args.Get(0).(ExportedGateway)
}
