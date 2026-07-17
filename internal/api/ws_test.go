package api

import (
	"crypto/tls"
	"encoding/json"
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

func TestAuthenticatedWebSocketRevokedAfterLogout(t *testing.T) {
	secret := strings.Repeat("s", 32)
	s := newSecureTestServer(t, secret, "https://table.example")
	cookie, csrf := loginTestSession(t, s, secret, "192.0.2.4:1234")
	srv, conn := openServerWebSocket(t, s, cookie, "")

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/logout", nil)
	require.NoError(t, err)
	req.AddCookie(cookie)
	req.Header.Set("X-CSRF-Token", csrf)
	response, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	response.Body.Close()
	require.Equal(t, http.StatusNoContent, response.StatusCode)

	s.bus.Publish(Event{Type: EventDiceRolled, Payload: map[string]any{"result": 18}})
	assertWebSocketRevokedWithoutEvent(t, s.hub, conn)
}

func TestAuthenticatedWebSocketIdleRevalidationDoesNotRefreshSession(t *testing.T) {
	secret := strings.Repeat("s", 32)
	s := newSecureTestServer(t, secret, "https://table.example")
	now := time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC)
	s.sessions.now = func() time.Time { return now }
	cookie, _ := loginTestSession(t, s, secret, "192.0.2.4:1234")
	_, conn := openServerWebSocket(t, s, cookie, "")
	lastSeenAfterUpgrade := s.sessions.sessions[cookie.Value].lastSeen

	now = now.Add(29 * time.Minute)
	s.bus.Publish(Event{Type: EventDiceRolled, Payload: map[string]any{"result": 18}})
	conn.SetReadDeadline(time.Now().Add(time.Second))
	var received Event
	require.NoError(t, conn.ReadJSON(&received))
	assert.Equal(t, EventDiceRolled, received.Type)
	assert.Equal(t, lastSeenAfterUpgrade, s.sessions.sessions[cookie.Value].lastSeen)

	now = now.Add(2 * time.Minute)
	s.bus.Publish(Event{Type: EventDiceRolled, Payload: map[string]any{"result": 19}})
	assertWebSocketRevokedWithoutEvent(t, s.hub, conn)
}

func TestAuthenticatedWebSocketRevokedAfterAbsoluteExpiry(t *testing.T) {
	secret := strings.Repeat("s", 32)
	s := newSecureTestServer(t, secret, "https://table.example")
	now := time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC)
	s.sessions.now = func() time.Time { return now }
	cookie, _ := loginTestSession(t, s, secret, "192.0.2.4:1234")
	srv, conn := openServerWebSocket(t, s, cookie, "")

	for range 24 {
		now = now.Add(29 * time.Minute)
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/auth/session", nil)
		require.NoError(t, err)
		req.AddCookie(cookie)
		response, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		var info SessionInfo
		require.NoError(t, json.NewDecoder(response.Body).Decode(&info))
		response.Body.Close()
		require.True(t, info.Authenticated)
	}
	now = now.Add(25 * time.Minute)

	s.bus.Publish(Event{Type: EventDiceRolled, Payload: map[string]any{"result": 18}})
	assertWebSocketRevokedWithoutEvent(t, s.hub, conn)
}

func TestBearerAuthenticatedWebSocketRemainsAuthorized(t *testing.T) {
	secret := strings.Repeat("s", 32)
	s := newSecureTestServer(t, secret, "https://table.example")
	_, conn := openServerWebSocket(t, s, nil, secret)

	s.bus.Publish(Event{Type: EventDiceRolled, Payload: map[string]any{"result": 18}})
	conn.SetReadDeadline(time.Now().Add(time.Second))
	var received Event
	require.NoError(t, conn.ReadJSON(&received))
	assert.Equal(t, EventDiceRolled, received.Type)
}

func openServerWebSocket(t *testing.T, s *Server, cookie *http.Cookie, bearer string) (*httptest.Server, *websocket.Conn) {
	t.Helper()
	srv := httptest.NewServer(s)
	t.Cleanup(srv.Close)
	header := http.Header{"Origin": []string{srv.URL}}
	if cookie != nil {
		header.Set("Cookie", cookie.String())
	}
	if bearer != "" {
		header.Set("Authorization", "Bearer "+bearer)
	}
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, response, err := websocket.DefaultDialer.Dial(wsURL, header)
	if response != nil {
		response.Body.Close()
	}
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	waitForHubClientCount(t, s.hub, 1)
	waitForBusSubscriber(t, s.bus)
	return srv, conn
}

func waitForHubClientCount(t *testing.T, hub *Hub, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for hub.ClientCount() != want {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d WebSocket clients; got %d", want, hub.ClientCount())
		}
		time.Sleep(time.Millisecond)
	}
}

func waitForBusSubscriber(t *testing.T, bus *Bus) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		bus.mu.Lock()
		count := len(bus.subscribers)
		bus.mu.Unlock()
		if count > 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for WebSocket hub bus subscription")
		}
		time.Sleep(time.Millisecond)
	}
}

func assertWebSocketRevokedWithoutEvent(t *testing.T, hub *Hub, conn *websocket.Conn) {
	t.Helper()
	waitForHubClientCount(t, hub, 0)
	conn.SetReadDeadline(time.Now().Add(time.Second))
	var received Event
	err := conn.ReadJSON(&received)
	require.Error(t, err, "revoked WebSocket received protected event: %#v", received)
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
