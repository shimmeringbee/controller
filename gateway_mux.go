package main

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"sync"
	"time"
)

type GatewayMapper interface {
	Gateways() map[string]da.Gateway
	Capability(string, da.Capability) interface{}
	Device(string) (da.Device, bool)
}

type GatewaySubscriber interface {
	Listen(chan interface{})
}

var _ GatewayMapper = (*GatewayMux)(nil)
var _ GatewaySubscriber = (*GatewayMux)(nil)

type GatewayMux struct {
	lock sync.RWMutex

	deviceByIdentifier map[string]da.Device
	gatewayByName      map[string]da.Gateway
	shutdownCh         []chan struct{}

	listeners []chan interface{}
}

func (m *GatewayMux) Add(n string, g da.Gateway) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.gatewayByName[n] = g

	ch := make(chan struct{}, 1)
	m.shutdownCh = append(m.shutdownCh, ch)

	selfDevice := g.Self()
	m.deviceByIdentifier[selfDevice.Identifier().String()] = selfDevice

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
				m.deviceByIdentifier[e.Identifier().String()] = e.Device
				m.lock.Unlock()
			case da.DeviceRemoved:
				m.lock.Lock()
				delete(m.deviceByIdentifier, e.Identifier().String())
				m.lock.Unlock()
			case da.DeviceLoaded:
				m.lock.Lock()
				m.deviceByIdentifier[e.Identifier().String()] = e.Device
				m.lock.Unlock()
			case capabilities.EnumerateDeviceSuccess:
				m.lock.Lock()
				m.deviceByIdentifier[e.Identifier().String()] = e.Device
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

	if daDevice, found := m.deviceByIdentifier[d]; found {
		return daDevice.Gateway().Capability(c)
	}

	return nil
}
func (m *GatewayMux) Device(id string) (da.Device, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	d, found := m.deviceByIdentifier[id]
	return d, found
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
