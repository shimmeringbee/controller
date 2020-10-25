package main

import (
	"context"
	"github.com/shimmeringbee/da"
	"sync"
	"time"
)

type GatewayMapper interface {
	Gateways() map[string]da.Gateway
	Capability(string, da.Capability) interface{}
}

type GatewaySubscriber interface {
	Listen(chan interface{})
}

var _ GatewayMapper = (*GatewayMux)(nil)
var _ GatewaySubscriber = (*GatewayMux)(nil)

type GatewayMux struct {
	lock sync.RWMutex

	gatewayByIdentifier map[string]da.Gateway
	gatewayByName       map[string]da.Gateway
	shutdownCh          []chan struct{}

	listeners []chan interface{}
}

func (m *GatewayMux) Add(n string, g da.Gateway) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.gatewayByName[n] = g

	ch := make(chan struct{}, 1)
	m.shutdownCh = append(m.shutdownCh, ch)

	go m.monitorGateway(g, ch)
}

func (m *GatewayMux) monitorGateway(g da.Gateway, shutCh chan struct{}) {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)

		if event, err := g.ReadEvent(ctx); err != nil && err != context.DeadlineExceeded {
			cancel()
			return
		} else if event != nil {
			switch e := event.(type) {
			case da.DeviceAdded:
				m.lock.Lock()
				m.gatewayByIdentifier[e.Identifier().String()] = e.Gateway()
				m.lock.Unlock()
			case da.DeviceRemoved:
				m.lock.Lock()
				delete(m.gatewayByIdentifier, e.Identifier().String())
				m.lock.Unlock()
			}

			m.sendToListeners(event)
		}

		cancel()

		select {
		case _ = <-shutCh:
			return
		default:
		}
	}
}

func (m *GatewayMux) Gateways() map[string]da.Gateway {
	m.lock.RLock()
	defer m.lock.RUnlock()

	result := make(map[string]da.Gateway, len(m.gatewayByName))
	for k, v := range m.gatewayByName {
		result[k] = v
	}
	return result
}

func (m *GatewayMux) Capability(d string, c da.Capability) interface{} {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if gw, found := m.gatewayByIdentifier[d]; found {
		return gw.Capability(c)
	}

	return nil
}

func (m *GatewayMux) sendToListeners(e interface{}) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for _, ch := range m.listeners {
		select {
		case ch <- e:
		default:
		}
	}
}

func (m *GatewayMux) Listen(ch chan interface{}) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.listeners = append(m.listeners, ch)
}

func (m *GatewayMux) Stop() {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for _, ch := range m.shutdownCh {
		ch <- struct{}{}
	}
}
