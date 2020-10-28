package v1

import (
	"github.com/shimmeringbee/da"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

type mockDeviceConverter struct {
	mock.Mock
}

func (m *mockDeviceConverter) convertDADeviceToDevice(daDevice da.Device) device {
	args := m.Called(daDevice)
	return args.Get(0).(device)
}

type mockGatewayConverter struct {
	mock.Mock
}

func (m *mockGatewayConverter) convertDAGatewayToGateway(daGateway da.Gateway) gateway {
	args := m.Called(daGateway)
	return args.Get(0).(gateway)
}

func Test_convertDADeviceToDevice(t *testing.T) {
	t.Run("converts a da device with basic information and capability list", func(t *testing.T) {
		mgwOne := mockGateway{}
		defer mgwOne.AssertExpectations(t)

		capOne := da.Capability(1)

		mockCapOne := mockBasicCapability{}
		defer mockCapOne.AssertExpectations(t)
		mockCapOne.On("Name").Return("capOne")
		mgwOne.On("Capability", capOne).Return(&mockCapOne)

		input := da.BaseDevice{
			DeviceGateway:      &mgwOne,
			DeviceIdentifier:   SimpleIdentifier{id: "one-one"},
			DeviceCapabilities: []da.Capability{capOne},
		}

		expected := device{
			Identifier:   "one-one",
			Capabilities: []string{"capOne"},
		}

		actual := convertDADeviceToDevice(input)

		assert.Equal(t, expected, actual)
	})
}

func Test_convertDAGatewayToGateway(t *testing.T) {
	t.Run("converts a da gateway with basic information and capability list", func(t *testing.T) {
		mgwOne := mockGateway{}
		defer mgwOne.AssertExpectations(t)

		mgwOne.On("Self").Return(da.BaseDevice{
			DeviceIdentifier: SimpleIdentifier{id: "self"},
		})

		capOne := da.Capability(1)
		mockCapOne := mockBasicCapability{}
		defer mockCapOne.AssertExpectations(t)
		mockCapOne.On("Name").Return("capOne")

		capTwo := da.Capability(2)
		mockCapTwo := mockBasicCapability{}
		defer mockCapTwo.AssertExpectations(t)
		mockCapTwo.On("Name").Return("capTwo")

		mgwOne.On("Capabilities").Return([]da.Capability{capOne, capTwo})
		mgwOne.On("Capability", capOne).Return(&mockCapOne)
		mgwOne.On("Capability", capTwo).Return(&mockCapTwo)

		expected := gateway{
			Capabilities: []string{"capOne", "capTwo"},
			SelfDevice:   "self",
		}

		actual := convertDAGatewayToGateway(&mgwOne)

		assert.Equal(t, expected, actual)
	})
}
