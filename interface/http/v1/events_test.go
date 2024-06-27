package v1

import (
	"context"
	"github.com/gorilla/websocket"
	"github.com/shimmeringbee/controller/state"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func serverAndConnect(f http.HandlerFunc) (*websocket.Conn, func(), error) {
	server := httptest.NewServer(f)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/"

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, nil, err
	}

	return ws, func() {
		ws.Close()
		server.Close()
	}, nil
}

func Test_eventsController_serversideevents(t *testing.T) {
	t.Run("sends a marshalled and formatted version of the eventbus message to the server side events endpoint", func(t *testing.T) {
		eb := state.NewEventBus()

		mem := &mockEventMapper{}
		defer mem.AssertExpectations(t)

		inputEvent := "event"
		mem.On("InitialEvents", mock.Anything).Return([]any{}, nil)
		mem.On("MapEvent", mock.Anything, inputEvent).Return([]any{"data"}, nil)

		wc := eventsController{
			eventbus:    eb,
			eventMapper: mem,
			logger:      logwrap.New(discard.Discard()),
		}

		ctx, done := context.WithCancel(context.Background())

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		go wc.serveServerSideEvent(w, req)

		time.Sleep(50 * time.Millisecond)

		eb.Publish(inputEvent)

		time.Sleep(50 * time.Millisecond)
		done()

		result := w.Result()

		d, err := io.ReadAll(result.Body)
		assert.NoError(t, err)

		assert.Equal(t, "\"data\"\n\n", string(d))

		assert.Equal(t, "*", result.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Type", result.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "text/event-stream", result.Header.Get("Content-Type"))
		assert.Equal(t, "no-cache", result.Header.Get("Cache-Control"))
		assert.Equal(t, "keep-alive", result.Header.Get("Connection"))
	})

	t.Run("sends initial synchronisation events to the server side events handler", func(t *testing.T) {
		eb := state.NewEventBus()

		mem := &mockEventMapper{}
		defer mem.AssertExpectations(t)

		mem.On("InitialEvents", mock.Anything).Return([]any{"data"}, nil)

		wc := eventsController{
			eventbus:    eb,
			eventMapper: mem,
			logger:      logwrap.New(discard.Discard()),
		}

		ctx, done := context.WithCancel(context.Background())

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		go wc.serveServerSideEvent(w, req)

		time.Sleep(50 * time.Millisecond)
		done()

		result := w.Result()

		d, err := io.ReadAll(result.Body)
		assert.NoError(t, err)

		assert.Equal(t, "\"data\"\n\n", string(d))

		assert.Equal(t, "*", result.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Type", result.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "text/event-stream", result.Header.Get("Content-Type"))
		assert.Equal(t, "no-cache", result.Header.Get("Cache-Control"))
		assert.Equal(t, "keep-alive", result.Header.Get("Connection"))
	})
}

func Test_eventsController_websocket(t *testing.T) {
	t.Run("sends a marshalled and formatted version of the eventbus message to the websocket connection", func(t *testing.T) {
		eb := state.NewEventBus()

		mem := &mockEventMapper{}
		defer mem.AssertExpectations(t)

		inputEvent := "event"
		mem.On("InitialEvents", mock.Anything).Return([]any{}, nil)
		mem.On("MapEvent", mock.Anything, inputEvent).Return([]any{"data"}, nil)

		wc := eventsController{
			eventbus:    eb,
			eventMapper: mem,
			logger:      logwrap.New(discard.Discard()),
		}

		c, teardown, err := serverAndConnect(wc.serveWebsocket)
		require.NoError(t, err)
		defer teardown()

		eb.Publish(inputEvent)

		c.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
		mt, actualData, err := c.ReadMessage()

		assert.NoError(t, err)
		assert.Equal(t, websocket.TextMessage, mt)
		assert.Equal(t, "\"data\"", string(actualData))
	})

	t.Run("sends initial synchronisation events to the websocket connection", func(t *testing.T) {
		eb := state.NewEventBus()

		mem := &mockEventMapper{}
		defer mem.AssertExpectations(t)

		mem.On("InitialEvents", mock.Anything).Return([]any{"data"}, nil)

		wc := eventsController{
			eventbus:    eb,
			eventMapper: mem,
			logger:      logwrap.New(discard.Discard()),
		}

		c, teardown, err := serverAndConnect(wc.serveWebsocket)
		require.NoError(t, err)
		defer teardown()

		c.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
		mt, actualData, err := c.ReadMessage()

		assert.NoError(t, err)
		assert.Equal(t, websocket.TextMessage, mt)
		assert.Equal(t, "\"data\"", string(actualData))
	})
}

type mockEventMapper struct {
	mock.Mock
}

func (m *mockEventMapper) MapEvent(ctx context.Context, e any) ([]any, error) {
	args := m.Called(ctx, e)
	return args.Get(0).([]any), args.Error(1)
}

func (m *mockEventMapper) InitialEvents(ctx context.Context) ([]any, error) {
	args := m.Called(ctx)
	return args.Get(0).([]any), args.Error(1)
}
