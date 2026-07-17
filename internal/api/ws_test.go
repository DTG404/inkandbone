package api

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHubBroadcastsEvents(t *testing.T) {
	bus := NewBus()
	hub := NewHub(bus)
	go hub.Run()

	srv := httptest.NewServer(http.HandlerFunc(hub.ServeWS))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	header := http.Header{"Origin": []string{srv.URL}}
	conn, _, err := websocket.DefaultDialer.Dial(url, header)
	require.NoError(t, err)
	defer conn.Close()

	// Wait for the hub to register the client
	deadline := time.Now().Add(500 * time.Millisecond)
	for hub.ClientCount() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for hub to register client")
		}
		time.Sleep(1 * time.Millisecond)
	}

	bus.Publish(Event{Type: EventDiceRolled, Payload: map[string]any{"result": 18}})

	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var received Event
	err = conn.ReadJSON(&received)
	require.NoError(t, err)
	assert.Equal(t, EventDiceRolled, received.Type)
}

func TestWebSocketRejectsDisallowedOrigin(t *testing.T) {
	hub := NewHub(NewBus())
	srv := httptest.NewServer(http.HandlerFunc(hub.ServeWS))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	header := http.Header{"Origin": []string{"https://attacker.example"}}
	conn, response, err := websocket.DefaultDialer.Dial(wsURL, header)
	if conn != nil {
		conn.Close()
	}
	require.Error(t, err)
	require.NotNil(t, response)
	assert.Equal(t, http.StatusForbidden, response.StatusCode)
}

func TestWebSocketOrigin(t *testing.T) {
	tests := []struct {
		name           string
		host           string
		origin         string
		allowedOrigins []string
		tls            bool
		want           bool
	}{
		{
			name:   "same HTTP origin",
			host:   "table.example:7432",
			origin: "http://table.example:7432",
			want:   true,
		},
		{
			name:   "same origin is case insensitive",
			host:   "table.example:7432",
			origin: "http://TABLE.EXAMPLE:7432",
			want:   true,
		},
		{
			name:   "same HTTPS origin",
			host:   "table.example:7432",
			origin: "https://table.example:7432",
			tls:    true,
			want:   true,
		},
		{
			name:   "same host with wrong scheme",
			host:   "table.example:7432",
			origin: "https://table.example:7432",
			want:   false,
		},
		{
			name:   "same origin normalizes default HTTP port",
			host:   "table.example:80",
			origin: "http://table.example",
			want:   true,
		},
		{
			name:           "explicit origin",
			host:           "table.internal:7432",
			origin:         "https://players.example",
			allowedOrigins: []string{"https://players.example"},
			want:           true,
		},
		{
			name:           "explicit origin normalizes case and default port",
			host:           "table.internal:7432",
			origin:         "https://players.example",
			allowedOrigins: []string{"HTTPS://PLAYERS.EXAMPLE:443"},
			want:           true,
		},
		{
			name:           "explicit origin preserves non-default port",
			host:           "table.internal:7432",
			origin:         "https://players.example:8443",
			allowedOrigins: []string{"https://players.example:8443"},
			want:           true,
		},
		{
			name:           "explicit origin rejects different port",
			host:           "table.internal:7432",
			origin:         "https://players.example:9443",
			allowedOrigins: []string{"https://players.example:8443"},
			want:           false,
		},
		{
			name:   "cross origin",
			host:   "table.example:7432",
			origin: "http://attacker.example:7432",
			want:   false,
		},
		{
			name: "missing origin",
			host: "table.example:7432",
			want: false,
		},
		{
			name:   "malformed origin",
			host:   "table.example:7432",
			origin: "https://[::1",
			want:   false,
		},
		{
			name:   "origin with path",
			host:   "table.example:7432",
			origin: "http://table.example:7432/not-an-origin",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := NewHub(NewBus())
			hub.SetAllowedOrigins(tt.allowedOrigins)
			r := httptest.NewRequest(http.MethodGet, "http://"+tt.host+"/ws", nil)
			r.Host = tt.host
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			if tt.tls {
				r.TLS = &tls.ConnectionState{}
			}

			assert.Equal(t, tt.want, hub.originAllowed(r))
		})
	}
}
