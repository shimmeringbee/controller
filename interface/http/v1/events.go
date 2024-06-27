package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/shimmeringbee/controller/interface/converters/exporter"
	"github.com/shimmeringbee/controller/state"
	"github.com/shimmeringbee/logwrap"
	"net/http"
	"time"
)

type eventsController struct {
	eventbus    state.EventSubscriber
	eventMapper exporter.EventExporter
	logger      logwrap.Logger
}

const ConnectionEventBufferSize = 16

func (z *eventsController) serveServerSideEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	doneCh := r.Context().Done()
	eventsCh := make(chan any, ConnectionEventBufferSize)

	z.eventbus.Subscribe(eventsCh)
	defer z.eventbus.Unsubscribe(eventsCh)

	flusher := w.(http.Flusher)

	z.sendLoop(func(b []byte) error {
		data := append(b, '\n', '\n')
		if n, err := w.Write(data); err != nil {
			return err
		} else if len(data) != n {
			return fmt.Errorf("failed to send full event: %d != %d", len(data), n)
		}

		flusher.Flush()
		return nil
	}, eventsCh, doneCh)
}

var wsUpgrader = websocket.Upgrader{}

func (z *eventsController) serveWebsocket(w http.ResponseWriter, r *http.Request) {
	c, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer c.Close()

	err = z.serverWebsocketConnection(c)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (z *eventsController) serverWebsocketConnection(c *websocket.Conn) error {
	eventsCh := make(chan any, ConnectionEventBufferSize)
	shutdownCh := make(chan struct{})

	z.eventbus.Subscribe(eventsCh)

	defer func() {
		z.eventbus.Unsubscribe(eventsCh)
		close(eventsCh)

		shutdownCh <- struct{}{}
		close(shutdownCh)
	}()

	go z.sendLoop(func(b []byte) error {
		return c.WriteMessage(websocket.TextMessage, b)
	}, eventsCh, shutdownCh)
	return z.serviceIncoming(c)
}

func (z *eventsController) sendLoop(publish func([]byte) error, ch chan any, shutCh <-chan struct{}) {
	initCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	events, err := z.eventMapper.InitialEvents(initCtx)
	cancel()
	if err != nil {
		return
	}

	for _, e := range events {
		if d, err := json.Marshal(e); err != nil {
			z.logger.LogError(context.Background(), "Failed to marshal message to websocket.", logwrap.Err(err))
			return
		} else {
			if err := publish(d); err != nil {
				z.logger.LogError(context.Background(), "Failed to send initial message to websocket.", logwrap.Err(err))
				return
			}
		}
	}

	for {
		select {
		case event := <-ch:
			if event == nil {
				return
			}

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
					if err := publish(d); err != nil {
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

func (z *eventsController) serviceIncoming(c *websocket.Conn) error {
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
