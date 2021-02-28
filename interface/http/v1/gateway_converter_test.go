package v1

import (
	"github.com/shimmeringbee/da"
	"github.com/stretchr/testify/assert"
	"testing"
)

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
