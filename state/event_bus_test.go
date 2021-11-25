package state

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEventBus(t *testing.T) {
	t.Run("subscribing to the bus results in published events being received", func(t *testing.T) {
		listenCh := make(chan interface{}, 1)
		expectedEvent := struct{}{}

		eb := NewEventBus()
		eb.Subscribe(listenCh)
		eb.Publish(expectedEvent)

		select {
		case actualEvent := <-listenCh:
			assert.Equal(t, expectedEvent, actualEvent)
		default:
			assert.Fail(t, "no event received")
		}
	})
}
