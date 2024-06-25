package state

import (
	"sync"
)

type EventPublisher interface {
	Publish(any)
}

type EventSubscriber interface {
	Subscribe(chan any)
	Unsubscribe(chan any)
}

var _ EventPublisher = (*EventBus)(nil)
var _ EventSubscriber = (*EventBus)(nil)

type nullEventPublisher struct{}

func (_ nullEventPublisher) Publish(any) {}

var NullEventPublisher = nullEventPublisher{}

type EventBus struct {
	channels     []chan any
	channelsLock *sync.RWMutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		channelsLock: &sync.RWMutex{},
	}
}

func (b *EventBus) Subscribe(ch chan any) {
	b.channelsLock.Lock()
	defer b.channelsLock.Unlock()

	b.channels = append(b.channels, ch)
}

func (b *EventBus) Unsubscribe(ch chan any) {
	b.channelsLock.Lock()
	defer b.channelsLock.Unlock()

	for i, c := range b.channels {
		if c == ch {
			b.channels = append(b.channels[:i], b.channels[i+1:]...)
			return
		}
	}
}

func (b *EventBus) Publish(e any) {
	b.channelsLock.RLock()
	defer b.channelsLock.RUnlock()

	for _, ch := range b.channels {
		select {
		case ch <- e:
		default:
		}
	}
}
