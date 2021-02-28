package mqtt

import (
	"github.com/shimmeringbee/controller/gateway"
	"github.com/shimmeringbee/controller/layers"
	"github.com/shimmeringbee/controller/metadata"
)

type Publisher func(prefix string, payload []byte)

type Interface struct {
	publisher Publisher
	stop      chan bool

	DeviceOrganiser   *metadata.DeviceOrganiser
	GatewayMux        gateway.GatewayMapper
	GatewaySubscriber gateway.GatewaySubscriber
	OutputStack       layers.OutputStack
}

func (i *Interface) IncomingMessage(topic string, payload []byte) {

}

func (i *Interface) Connected(publisher Publisher, publishAll bool) {
	i.publisher = publisher

	if publishAll {

	}
}

func (i *Interface) Disconnected() {
	i.publisher = nil
}

func (i *Interface) Start() {
	i.stop = make(chan bool, 1)

	ch := make(chan interface{}, 100)
	i.GatewaySubscriber.Listen(ch)

	go i.handleEvents(ch)
}

func (i *Interface) Stop() {
	if i.stop != nil {
		i.stop <- true
	}
}

func (i *Interface) handleEvents(ch chan interface{}) {
	for {
		select {
		case event := <-ch:
			_ = event
		case <-i.stop:
			return
		}
	}
}
