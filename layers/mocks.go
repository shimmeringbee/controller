package layers

import (
	"github.com/shimmeringbee/da"
	"github.com/stretchr/testify/mock"
)

type NoLayersStack struct {
}

func (p NoLayersStack) Layers() []string {
	return []string{}
}

func (p NoLayersStack) Lookup(name string) OutputLayer {
	return nil
}

type MockOutputStack struct {
	mock.Mock
}

func (m *MockOutputStack) Layers() []string {
	called := m.Called()
	return called.Get(0).([]string)
}

func (m *MockOutputStack) Lookup(name string) OutputLayer {
	called := m.Called(name)
	return called.Get(0).(OutputLayer)
}

type MockOutputLayer struct {
	mock.Mock
}

func (m *MockOutputLayer) Name() string {
	called := m.Called()
	return called.String(0)
}

func (m *MockOutputLayer) Device(rl RetentionLevel, d da.Device) da.Device {
	called := m.Called(rl, d)
	return called.Get(0).(da.Device)
}

func (m *MockOutputLayer) MaintainedStatus(c da.Capability, d da.Device) any {
	called := m.Called(c, d)
	return called.Get(0)
}
