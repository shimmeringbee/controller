package main

import (
	"github.com/shimmeringbee/da"
	"github.com/stretchr/testify/assert"
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
	t.Run("Lookup returns a pass thru layer", func(t *testing.T) {
		ps := PassThruStack{}

		expectedLayer := PassThruLayer{}
		actualLayer := ps.Lookup("anything")

		assert.Equal(t, expectedLayer, actualLayer)
	})
}
