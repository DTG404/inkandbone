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
	clients map[*websocket.Conn]hubClient
	bus     *Bus
	origins map[string]struct{}
}

type hubClient struct {
	send       chan Event
	authorized func() bool
}

func NewHub(bus *Bus) *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]hubClient),
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
	defer h.mu.Unlock()
	for conn, client := range h.clients {
		if client.authorized != nil && !client.authorized() {
			delete(h.clients, conn)
			close(client.send)
			continue
		}
		select {
		case client.send <- event:
		default:
			// slow client; drop rather than block the broadcast goroutine
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
// is revalidated before each protected event is broadcast. A nil validator
// preserves unsecured and bearer-authenticated connection behavior.
func (h *Hub) ServeWSAuthorized(w http.ResponseWriter, r *http.Request, authorized func() bool) {
	upgrader := websocket.Upgrader{CheckOrigin: h.originAllowed}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}

	send := make(chan Event, 64)

	h.mu.Lock()
	h.clients[conn] = hubClient{send: send, authorized: authorized}
	h.mu.Unlock()

	// Write goroutine — the only goroutine that calls WriteJSON on this conn.
	go func() {
		for event := range send {
			if err := conn.WriteJSON(event); err != nil {
				log.Printf("ws write error: %v", err)
				break
			}
		}
		conn.Close()
	}()

	defer func() {
		h.mu.Lock()
		if client, ok := h.clients[conn]; ok {
			delete(h.clients, conn)
			close(client.send)
		}
		h.mu.Unlock()
	}()

	// Read loop — keeps connection alive; client messages are ignored for now.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
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
