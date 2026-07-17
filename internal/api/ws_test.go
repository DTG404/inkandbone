package api

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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

	assertWebSocketRevokedWithoutEvent(t, s.hub, conn)
}

func TestAuthenticatedWebSocketRevokedWhenHTTPDiscoversIdleExpiry(t *testing.T) {
	secret := strings.Repeat("s", 32)
	s := newSecureTestServer(t, secret, "https://table.example")
	now := time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC)
	s.sessions.now = func() time.Time { return now }
	cookie, _ := loginTestSession(t, s, secret, "192.0.2.4:1234")
	srv, conn := openServerWebSocket(t, s, cookie, "")
	now = now.Add(sessionIdleLifetime + time.Second)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/auth/session", nil)
	require.NoError(t, err)
	req.AddCookie(cookie)
	response, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	var info SessionInfo
	require.NoError(t, json.NewDecoder(response.Body).Decode(&info))
	response.Body.Close()
	require.False(t, info.Authenticated)

	assertWebSocketRevokedWithoutEvent(t, s.hub, conn)
}

func TestAuthenticatedWebSocketLogoutOrderedWithEventEnqueue(t *testing.T) {
	secret := strings.Repeat("s", 32)
	s := newSecureTestServer(t, secret, "https://table.example")
	cookie, csrf := loginTestSession(t, s, secret, "192.0.2.4:1234")
	srv, _ := openServerWebSocket(t, s, cookie, "")
	client := soleHubClient(t, s.hub)

	client.mu.Lock()
	clientLocked := true
	defer func() {
		if clientLocked {
			client.mu.Unlock()
		}
	}()
	s.bus.Publish(Event{Type: EventDiceRolled, Payload: map[string]any{"result": 18}})
	waitForSessionLockHeld(t, s.sessions)

	logoutDone := make(chan int, 1)
	go func() {
		logoutDone <- logoutStatus(srv.URL, cookie, csrf)
	}()
	select {
	case status := <-logoutDone:
		t.Fatalf("logout returned %d while an authorized enqueue still held the session order", status)
	case <-time.After(100 * time.Millisecond):
	}

	client.mu.Unlock()
	clientLocked = false
	select {
	case status := <-logoutDone:
		require.Equal(t, http.StatusNoContent, status)
	case <-time.After(time.Second):
		t.Fatal("logout did not complete after the ordered enqueue was released")
	}
	waitForHubClientCount(t, s.hub, 0)
}

func TestHubClientRevocationInterruptsWriterWithoutDrainingBacklog(t *testing.T) {
	connectionClosed := make(chan struct{})
	writeStarted := make(chan struct{})
	var writes atomic.Int32
	var closes atomic.Int32
	client := newHubClientWithIO(
		func(any) error {
			writes.Add(1)
			close(writeStarted)
			<-connectionClosed
			return errors.New("connection closed")
		},
		func() error {
			closes.Add(1)
			close(connectionClosed)
			return nil
		},
	)
	writerDone := make(chan struct{})
	go func() {
		client.runWriter()
		close(writerDone)
	}()

	require.True(t, client.enqueue(Event{Type: EventDiceRolled}))
	select {
	case <-writeStarted:
	case <-time.After(time.Second):
		t.Fatal("writer did not block on the first event")
	}
	for range cap(client.send) {
		require.True(t, client.enqueue(Event{Type: EventDiceRolled}))
	}
	require.Len(t, client.send, cap(client.send))

	client.revoke()
	client.revoke()
	select {
	case <-writerDone:
	case <-time.After(time.Second):
		t.Fatal("revocation did not interrupt the blocked writer")
	}
	assert.EqualValues(t, 1, writes.Load())
	assert.EqualValues(t, 1, closes.Load())
	assert.Len(t, client.send, cap(client.send), "revocation must not close and drain buffered events")
	assert.False(t, client.enqueue(Event{Type: EventDiceRolled}))
}

func soleHubClient(t *testing.T, hub *Hub) *hubClient {
	t.Helper()
	hub.mu.Lock()
	defer hub.mu.Unlock()
	require.Len(t, hub.clients, 1)
	for _, client := range hub.clients {
		return client
	}
	return nil
}

func waitForSessionLockHeld(t *testing.T, sessions *sessionManager) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		if !sessions.mu.TryLock() {
			return
		}
		sessions.mu.Unlock()
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for event delivery to hold the session lock")
		}
		time.Sleep(time.Millisecond)
	}
}

func logoutStatus(serverURL string, cookie *http.Cookie, csrf string) int {
	req, err := http.NewRequest(http.MethodPost, serverURL+"/api/auth/logout", nil)
	if err != nil {
		return 0
	}
	req.AddCookie(cookie)
	req.Header.Set("X-CSRF-Token", csrf)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0
	}
	response.Body.Close()
	return response.StatusCode
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
