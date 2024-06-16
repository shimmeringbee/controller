package layers

import (
	"github.com/shimmeringbee/da/mocks"
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

	t.Run("Device returns own device", func(t *testing.T) {
		daDevice := mocks.SimpleDevice{}

		expectedResult := daDevice

		pt := PassThruLayer{}

		actualResult := pt.Device(Maintain, daDevice)

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
