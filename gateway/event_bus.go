package gateway

import (
	"sync"
)

type EventPublisher interface {
	Publish(interface{})
}

type EventSubscriber interface {
	Subscribe(chan interface{})
}

var _ EventPublisher = (*EventBus)(nil)
var _ EventSubscriber = (*EventBus)(nil)

type EventBus struct {
	channels     []chan interface{}
	channelsLock *sync.RWMutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		channelsLock: &sync.RWMutex{},
	}
}

func (b *EventBus) Subscribe(ch chan interface{}) {
	b.channelsLock.Lock()
	defer b.channelsLock.Unlock()

	b.channels = append(b.channels, ch)
}

func (b *EventBus) Publish(e interface{}) {
	b.channelsLock.RLock()
	defer b.channelsLock.RUnlock()

	for _, ch := range b.channels {
		select {
		case ch <- e:
		default:
		}
	}
}
