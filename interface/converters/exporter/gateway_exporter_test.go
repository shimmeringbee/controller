package exporter

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/mocks"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExportGateway(t *testing.T) {
	t.Run("converts a da gateway with basic information and capability list", func(t *testing.T) {
		mgwOne := mocks.Gateway{}
		defer mgwOne.AssertExpectations(t)

		mdev := &mocks.MockDevice{}
		defer mdev.AssertExpectations(t)

		mdev.On("Identifier").Return(SimpleIdentifier{id: "self"})

		mgwOne.On("Self").Return(mdev)

		capOne := da.Capability(1)
		mockCapOne := mocks.BasicCapability{}
		defer mockCapOne.AssertExpectations(t)

		capTwo := da.Capability(2)
		mockCapTwo := mocks.BasicCapability{}
		defer mockCapTwo.AssertExpectations(t)

		mgwOne.On("Capabilities").Return([]da.Capability{capOne, capTwo})

		expected := ExportedGateway{
			Capabilities: []string{"EnumerateDevice", "ProductInformation"},
			SelfDevice:   "self",
		}

		actual := ExportGateway(&mgwOne)

		assert.Equal(t, expected, actual)
	})
}
