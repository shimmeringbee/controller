package v1

import (
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/shimmeringbee/controller/interface/converters/exporter"
	"github.com/shimmeringbee/controller/state"
	"github.com/shimmeringbee/logwrap"
	"net/http"
	"time"
)

var wsUpgrader = websocket.Upgrader{}

type websocketController struct {
	eventbus    state.EventSubscriber
	eventMapper exporter.EventExporter
	logger      logwrap.Logger
}

func (z *websocketController) serveWebsocket(w http.ResponseWriter, r *http.Request) {
	c, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer c.Close()

	err = z.handleConnection(c)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

const WebsocketConnectionEventBufferSize = 16

func (z *websocketController) handleConnection(c *websocket.Conn) error {
	eventsCh := make(chan any, WebsocketConnectionEventBufferSize)
	shutdownCh := make(chan struct{})
	defer close(eventsCh)
	defer func() {
		shutdownCh <- struct{}{}
		close(shutdownCh)
	}()

	z.eventbus.Subscribe(eventsCh)
	defer z.eventbus.Unsubscribe(eventsCh)

	initCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	initialEvents, err := z.eventMapper.InitialEvents(initCtx)
	cancel()
	if err != nil {
		return err
	}

	go z.serviceOutgoing(c, initialEvents, eventsCh, shutdownCh)
	return z.serviceIncoming(c)
}

func (z *websocketController) serviceOutgoing(c *websocket.Conn, events []any, ch chan any, shutCh chan struct{}) {
	for _, e := range events {
		if d, err := json.Marshal(e); err != nil {
			z.logger.LogError(context.Background(), "Failed to marshal message to websocket.", logwrap.Err(err))
			return
		} else {
			if err := c.WriteMessage(websocket.TextMessage, d); err != nil {
				z.logger.LogError(context.Background(), "Failed to send initial message to websocket.", logwrap.Err(err))
				return
			}
		}
	}

	for {
		select {
		case event := <-ch:
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			es, err := z.eventMapper.MapEvent(ctx, event)
			cancel()

			if err != nil {
				z.logger.LogError(ctx, "Failed to map event to websocket message.", logwrap.Err(err), logwrap.Datum("event", event))
				continue
			}

			for _, e := range es {
				if d, err := json.Marshal(e); err != nil {
					z.logger.LogError(context.Background(), "Failed to marshal message to websocket.", logwrap.Err(err))
					return
				} else {
					if err := c.WriteMessage(websocket.TextMessage, d); err != nil {
						z.logger.LogError(ctx, "Failed to send messages to websocket.", logwrap.Err(err))
						return
					}
				}
			}
		case <-shutCh:
			return
		}
	}
}

func (z *websocketController) serviceIncoming(c *websocket.Conn) error {
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			if _, ok := err.(*websocket.CloseError); ok {
				z.logger.LogDebug(context.Background(), "Websocket closed.", logwrap.Err(err))
				return nil
			}
			z.logger.LogError(context.Background(), "Failed to read message from websocket.", logwrap.Err(err))
			return err
		}
	}
}
