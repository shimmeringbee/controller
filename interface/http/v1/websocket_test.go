package v1

import (
	"context"
	"github.com/gorilla/websocket"
	"github.com/shimmeringbee/controller/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func Test_websocketController(t *testing.T) {
	t.Run("sends a marshalled and formatted version of the eventbus message to the websocket connection", func(t *testing.T) {
		eb := state.NewEventBus()

		mem := mockEventMapper{}
		defer mem.AssertExpectations(t)

		inputEvent := "event"
		expectedData := []byte("data")
		mem.On("InitialEvents", mock.Anything).Return([][]byte{}, nil)
		mem.On("MapEvent", mock.Anything, inputEvent).Return(expectedData, nil)

		wc := websocketController{
			eventbus:    eb,
			eventMapper: &mem,
		}

		c, teardown, err := serverAndConnect(wc.serveWebsocket)
		require.NoError(t, err)
		defer teardown()

		eb.Publish(inputEvent)

		c.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
		mt, actualData, err := c.ReadMessage()

		assert.NoError(t, err)
		assert.Equal(t, websocket.TextMessage, mt)
		assert.Equal(t, expectedData, actualData)
	})

	t.Run("sends initial synchronisation events to the websocket connection", func(t *testing.T) {
		eb := state.NewEventBus()

		mem := mockEventMapper{}
		defer mem.AssertExpectations(t)

		expectedData := []byte("data")
		mem.On("InitialEvents", mock.Anything).Return([][]byte{expectedData}, nil)

		wc := websocketController{
			eventbus:    eb,
			eventMapper: &mem,
		}

		c, teardown, err := serverAndConnect(wc.serveWebsocket)
		require.NoError(t, err)
		defer teardown()

		c.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
		mt, actualData, err := c.ReadMessage()

		assert.NoError(t, err)
		assert.Equal(t, websocket.TextMessage, mt)
		assert.Equal(t, expectedData, actualData)
	})
}

type mockEventMapper struct {
	mock.Mock
}

func (m *mockEventMapper) MapEvent(ctx context.Context, e interface{}) ([][]byte, error) {
	args := m.Called(ctx, e)
	return args.Get(0).([][]byte), args.Error(1)
}

func (m *mockEventMapper) InitialEvents(ctx context.Context) ([][]byte, error) {
	args := m.Called(ctx)
	return args.Get(0).([][]byte), args.Error(1)
}
