package mqtt

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInterface_Connected(t *testing.T) {
	t.Run("publisher is set correctly", func(t *testing.T) {
		i := Interface{}

		publisher := func(prefix string, payload []byte) {
			t.Fatal("incorrectly called publisher")
		}

		i.Connected(publisher, false)

		assert.NotNil(t, i.publisher)
	})
}
