package state

import (
	"sync"
)

type EventPublisher interface {
	Publish(interface{})
}

type EventSubscriber interface {
	Subscribe(chan interface{})
	Unsubscribe(chan interface{})
}

var _ EventPublisher = (*EventBus)(nil)
var _ EventSubscriber = (*EventBus)(nil)

type nullEventPublisher struct{}

func (_ nullEventPublisher) Publish(interface{}) {}

var NullEventPublisher = nullEventPublisher{}

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

func (b *EventBus) Unsubscribe(ch chan interface{}) {
	b.channelsLock.Lock()
	defer b.channelsLock.Unlock()

	for i, c := range b.channels {
		if c == ch {
			b.channels = append(b.channels[:i], b.channels[i+1:]...)
			return
		}
	}
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
