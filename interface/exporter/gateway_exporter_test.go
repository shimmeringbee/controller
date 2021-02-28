package exporter

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/mocks"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExportGateway(t *testing.T) {
	t.Run("converts a da ExportedGateway with basic information and capability list", func(t *testing.T) {
		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		mgwOne.On("Self").Return(da.BaseDevice{
			DeviceIdentifier: SimpleIdentifier{id: "self"},
		})

		capOne := da.Capability(1)
		mockCapOne := mocks.BasicCapability{}
		defer mockCapOne.AssertExpectations(t)
		mockCapOne.On("Name").Return("capOne")

		capTwo := da.Capability(2)
		mockCapTwo := mocks.BasicCapability{}
		defer mockCapTwo.AssertExpectations(t)
		mockCapTwo.On("Name").Return("capTwo")

		mgwOne.On("Capabilities").Return([]da.Capability{capOne, capTwo})
		mgwOne.On("Capability", capOne).Return(&mockCapOne)
		mgwOne.On("Capability", capTwo).Return(&mockCapTwo)

		expected := ExportedGateway{
			Capabilities: []string{"capOne", "capTwo"},
			SelfDevice:   "self",
		}

		actual := ExportGateway(&mgwOne)

		assert.Equal(t, expected, actual)
	})
}
