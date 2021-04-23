package invoker

import (
	"context"
	"github.com/shimmeringbee/controller/layers"
	"github.com/shimmeringbee/da"
	"github.com/stretchr/testify/mock"
)

type MockDeviceInvoker struct {
	mock.Mock
}

func (m *MockDeviceInvoker) InvokeDevice(ctx context.Context, o layers.OutputLayer, r layers.RetentionLevel, dad da.Device, capabilityName string, actionName string, payload []byte) (interface{}, error) {
	args := m.Called(ctx, o, r, dad, capabilityName, actionName, payload)
	return args.Get(0), args.Error(1)
}
