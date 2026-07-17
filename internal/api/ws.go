package api

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub manages WebSocket connections and broadcasts events to all clients.
// Each client gets a dedicated send channel and write goroutine so that
// broadcast (called from Hub.Run) never shares a *websocket.Conn with the
// per-connection read goroutine in ServeWS.
type Hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]*hubClient
	bus     *Bus
	origins map[string]struct{}
}

type hubClient struct {
	mu           sync.Mutex
	conn         *websocket.Conn
	send         chan Event
	done         chan struct{}
	revoked      bool
	deliverEvent func(Event) bool
	onClose      func()
	writeJSON    func(any) error
	closeConn    func() error
}

func NewHub(bus *Bus) *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]*hubClient),
		bus:     bus,
		origins: make(map[string]struct{}),
	}
}

// SetAllowedOrigins replaces the explicit WebSocket origin allowlist.
func (h *Hub) SetAllowedOrigins(origins []string) {
	normalized := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		if parsed, ok := normalizeOrigin(origin); ok {
			normalized[parsed.String()] = struct{}{}
		}
	}

	h.mu.Lock()
	h.origins = normalized
	h.mu.Unlock()
}

// Run subscribes to the event bus and broadcasts all events to connected clients.
// Call in a goroutine.
func (h *Hub) Run() {
	ch := h.bus.Subscribe()
	for event := range ch {
		h.broadcast(event)
	}
}

func (h *Hub) broadcast(event Event) {
	h.mu.Lock()
	clients := make([]*hubClient, 0, len(h.clients))
	for _, client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.Unlock()

	for _, client := range clients {
		if !client.deliver(event) {
			client.revoke()
			h.removeClient(client)
		}
	}
}

// ClientCount returns the number of currently connected WebSocket clients.
func (h *Hub) ClientCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

// ServeWS upgrades an HTTP connection to WebSocket and registers it with the hub.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	h.ServeWSAuthorized(w, r, nil)
}

// ServeWSAuthorized upgrades and registers a connection whose authorization
// is configured before registration. A nil configurator preserves unsecured
// and bearer-authenticated connection behavior.
func (h *Hub) ServeWSAuthorized(w http.ResponseWriter, r *http.Request, configure func(*hubClient) bool) {
	upgrader := websocket.Upgrader{
		CheckOrigin: h.originAllowed,
		Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
			if status >= http.StatusInternalServerError {
				serverError(w, r, reason)
				return
			}
			w.Header().Set("Sec-WebSocket-Version", "13")
			http.Error(w, http.StatusText(status), status)
		},
	}
	responseHeader := make(http.Header)
	if id := requestID(r); id != "" {
		responseHeader.Set("X-Request-ID", id)
	}
	conn, err := upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}

	client := newHubClient(conn)
	if configure != nil && !configure(client) {
		client.revoke()
		return
	}
	client.mu.Lock()
	if client.revoked {
		client.mu.Unlock()
		if client.onClose != nil {
			client.onClose()
		}
		return
	}
	h.mu.Lock()
	h.clients[conn] = client
	h.mu.Unlock()
	client.mu.Unlock()

	// Write goroutine — the only goroutine that calls WriteJSON on this conn.
	go client.runWriter()

	defer func() {
		client.revoke()
		h.removeClient(client)
		if client.onClose != nil {
			client.onClose()
		}
	}()

	// Read loop — keeps connection alive; client messages are ignored for now.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

func newHubClient(conn *websocket.Conn) *hubClient {
	client := newHubClientWithIO(conn.WriteJSON, conn.Close)
	client.conn = conn
	return client
}

func newHubClientWithIO(writeJSON func(any) error, closeConn func() error) *hubClient {
	return &hubClient{
		send:      make(chan Event, 64),
		done:      make(chan struct{}),
		writeJSON: writeJSON,
		closeConn: closeConn,
	}
}

func (c *hubClient) deliver(event Event) bool {
	if c.deliverEvent != nil {
		return c.deliverEvent(event)
	}
	return c.enqueue(event)
}

func (c *hubClient) enqueue(event Event) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.revoked {
		return false
	}
	select {
	case c.send <- event:
	default:
		// Slow client; drop rather than block the broadcast goroutine.
	}
	return true
}

func (c *hubClient) revoke() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.revoked {
		return
	}
	c.revoked = true
	close(c.done)
	_ = c.closeConn()
}

func (c *hubClient) runWriter() {
	defer c.revoke()
	for {
		select {
		case <-c.done:
			return
		case event := <-c.send:
			select {
			case <-c.done:
				return
			default:
			}
			if err := c.writeJSON(event); err != nil {
				return
			}
		}
	}
}

func (h *Hub) removeClient(client *hubClient) {
	h.mu.Lock()
	if current, ok := h.clients[client.conn]; ok && current == client {
		delete(h.clients, client.conn)
	}
	h.mu.Unlock()
}

type normalizedOrigin struct {
	scheme   string
	hostPort string
}

func (o normalizedOrigin) String() string {
	return o.scheme + "://" + o.hostPort
}

func (h *Hub) originAllowed(r *http.Request) bool {
	origins := r.Header.Values("Origin")
	if len(origins) != 1 {
		return false
	}

	origin, ok := normalizeOrigin(origins[0])
	if !ok {
		return false
	}

	requestScheme := "http"
	if r.TLS != nil {
		requestScheme = "https"
	}
	requestHost, ok := normalizeHostPort(r.Host, requestScheme)
	if ok && origin.scheme == requestScheme && origin.hostPort == requestHost {
		return true
	}

	h.mu.Lock()
	_, ok = h.origins[origin.String()]
	h.mu.Unlock()
	return ok
}

func normalizeOrigin(raw string) (normalizedOrigin, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return normalizedOrigin{}, false
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return normalizedOrigin{}, false
	}
	hostPort, ok := normalizeURLHostPort(u, scheme)
	if !ok {
		return normalizedOrigin{}, false
	}
	return normalizedOrigin{scheme: scheme, hostPort: hostPort}, true
}

func normalizeURLHostPort(u *url.URL, scheme string) (string, bool) {
	host := u.Hostname()
	if host == "" {
		return "", false
	}
	port := u.Port()
	if port == "" {
		port = defaultPort(scheme)
	}
	return net.JoinHostPort(strings.ToLower(host), port), true
}

func normalizeHostPort(raw, scheme string) (string, bool) {
	host, port, err := net.SplitHostPort(raw)
	if err != nil {
		if strings.Contains(raw, ":") {
			if net.ParseIP(raw) == nil {
				return "", false
			}
			host = raw
		} else {
			host = raw
		}
	}
	if host == "" || strings.ContainsAny(host, "/?#@") {
		return "", false
	}
	if port == "" {
		port = defaultPort(scheme)
	}
	return net.JoinHostPort(strings.ToLower(host), port), true
}

func defaultPort(scheme string) string {
	if scheme == "https" {
		return "443"
	}
	return "80"
}
