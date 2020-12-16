package layers

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestPassThruLayer(t *testing.T) {
	t.Run("Name returns PassThru", func(t *testing.T) {
		pt := PassThruLayer{}

		expectedName := "PassThru"
		actualName := pt.Name()

		assert.Equal(t, expectedName, actualName)
	})

	t.Run("MaintainedStatus returns PassThru", func(t *testing.T) {
		pt := PassThruLayer{}

		expectedState := struct{}{}
		actualState := pt.MaintainedStatus(da.Capability(0x0000), nil)

		assert.Equal(t, expectedState, actualState)
	})

	t.Run("Capability calls underlying device for capability implementation", func(t *testing.T) {
		mg := mockGateway{}
		defer mg.AssertExpectations(t)

		daDevice := da.BaseDevice{DeviceGateway: &mg}
		capability := da.Capability(0x0001)

		expectedResult := struct{}{}

		mg.On("Capability", capability).Return(expectedResult)

		pt := PassThruLayer{}

		actualResult := pt.Capability(Maintain, capability, daDevice)

		assert.Equal(t, expectedResult, actualResult)
	})
}

func TestPassThruStack(t *testing.T) {
	t.Run("Layers returns a a list of layers, with just one", func(t *testing.T) {
		ps := PassThruStack{}

		expectedLayer := []string{"PassThru"}
		actualLayer := ps.Layers()

		assert.Equal(t, expectedLayer, actualLayer)
	})

	t.Run("Lookup returns a pass thru layer", func(t *testing.T) {
		ps := PassThruStack{}

		expectedLayer := PassThruLayer{}
		actualLayer := ps.Lookup("anything")

		assert.Equal(t, expectedLayer, actualLayer)
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
