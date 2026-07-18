package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	authCookieName             = "ttrpg_session"
	sessionIdleLifetime        = 30 * time.Minute
	sessionAbsoluteLifetime    = 12 * time.Hour
	loginAttemptWindow         = time.Minute
	maxLoginFailures           = 5
	maxTrackedFailureAddresses = 4096
)

// SessionInfo describes the browser's current authentication state.
type SessionInfo struct {
	Authenticated bool   `json:"authenticated"`
	CSRFToken     string `json:"csrf_token,omitempty"`
}

type sessionRecord struct {
	csrf      string
	createdAt time.Time
	lastSeen  time.Time
	clients   map[*hubClient]struct{}
}

type loginFailures struct {
	windowStarted time.Time
	count         int
}

type sessionManager struct {
	mu               sync.Mutex
	secret           string
	sessions         map[string]sessionRecord
	failures         map[string]loginFailures
	now              func() time.Time
	compareSecret    func(got, want string) bool
	lastFailurePrune time.Time
}

func newSessionManager(secret string) *sessionManager {
	return &sessionManager{
		secret:        secret,
		sessions:      make(map[string]sessionRecord),
		failures:      make(map[string]loginFailures),
		now:           time.Now,
		compareSecret: secretMatches,
	}
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func secretMatches(got, want string) bool {
	if len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func (m *sessionManager) create() (string, sessionRecord, error) {
	sessionToken, err := randomToken()
	if err != nil {
		return "", sessionRecord{}, err
	}
	csrfToken, err := randomToken()
	if err != nil {
		return "", sessionRecord{}, err
	}
	now := m.now()
	record := sessionRecord{
		csrf:      csrfToken,
		createdAt: now,
		lastSeen:  now,
		clients:   make(map[*hubClient]struct{}),
	}
	m.mu.Lock()
	m.sessions[sessionToken] = record
	m.mu.Unlock()
	return sessionToken, record, nil
}

func (m *sessionManager) get(token string, refreshIdle bool) (sessionRecord, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.getLocked(token, refreshIdle)
}

func (m *sessionManager) getLocked(token string, refreshIdle bool) (sessionRecord, bool) {
	record, ok := m.sessions[token]
	if !ok {
		return sessionRecord{}, false
	}
	now := m.now()
	if now.Sub(record.lastSeen) >= sessionIdleLifetime || now.Sub(record.createdAt) >= sessionAbsoluteLifetime {
		m.invalidateLocked(token)
		return sessionRecord{}, false
	}
	if refreshIdle {
		record.lastSeen = now
		m.sessions[token] = record
	}
	return record, true
}

func (m *sessionManager) delete(token string) {
	m.mu.Lock()
	m.invalidateLocked(token)
	m.mu.Unlock()
}

// Lock order is sessionManager.mu then hubClient.mu. Hub.mu is never held
// while either lock is acquired. Keeping invalidation and enqueue under this
// order gives cookie events a total order with logout and expiry.
func (m *sessionManager) invalidateLocked(token string) {
	record, ok := m.sessions[token]
	if !ok {
		return
	}
	delete(m.sessions, token)
	for client := range record.clients {
		client.revoke()
	}
}

func (m *sessionManager) registerClient(token string, client *hubClient) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	record, ok := m.getLocked(token, false)
	if !ok {
		return false
	}
	record.clients[client] = struct{}{}
	m.sessions[token] = record
	return true
}

func (m *sessionManager) unregisterClient(token string, client *hubClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	record, ok := m.sessions[token]
	if !ok {
		return
	}
	delete(record.clients, client)
}

func (m *sessionManager) deliverEvent(token string, client *hubClient, event Event) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	record, ok := m.getLocked(token, false)
	if !ok {
		return false
	}
	if _, registered := record.clients[client]; !registered {
		return false
	}
	return client.enqueue(event)
}

func (m *sessionManager) reserveLoginAttempt(address string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	if m.lastFailurePrune.IsZero() || now.Sub(m.lastFailurePrune) >= loginAttemptWindow {
		for failedAddress, failure := range m.failures {
			if now.Sub(failure.windowStarted) >= loginAttemptWindow {
				delete(m.failures, failedAddress)
			}
		}
		m.lastFailurePrune = now
	}
	failure, tracked := m.failures[address]
	if !tracked && len(m.failures) >= maxTrackedFailureAddresses {
		return false
	}
	if failure.windowStarted.IsZero() || now.Sub(failure.windowStarted) >= loginAttemptWindow {
		failure = loginFailures{windowStarted: now}
	}
	if failure.count >= maxLoginFailures {
		return false
	}
	failure.count++
	m.failures[address] = failure
	return true
}

func (m *sessionManager) clearFailures(address string) {
	m.mu.Lock()
	delete(m.failures, address)
	m.mu.Unlock()
}

func remoteAddress(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	if r.RemoteAddr == "" {
		return "unknown"
	}
	return r.RemoteAddr
}

func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if s.sessions == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	address := remoteAddress(r)
	if !s.sessions.reserveLoginAttempt(address) {
		http.Error(w, "too many login attempts", http.StatusTooManyRequests)
		return
	}

	var input struct {
		Secret string `json:"secret"`
	}
	if err := decodeJSON(w, r, &input, shortJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if !s.sessions.compareSecret(input.Secret, s.sessions.secret) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	s.sessions.clearFailures(address)
	token, _, err := s.sessions.create()
	if err != nil {
		serverError(w, r, err)
		return
	}
	http.SetCookie(w, s.authCookie(r, token, 0))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAuthSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	info := SessionInfo{Authenticated: s.sessions == nil}
	if s.sessions != nil {
		if bearerAuthenticated(r, s.sessions.secret) {
			info.Authenticated = true
		} else if cookie, err := r.Cookie(authCookieName); err == nil {
			if record, ok := s.sessions.get(cookie.Value, true); ok {
				info.Authenticated = true
				info.CSRFToken = record.csrf
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(info)
}

func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if s.sessions != nil {
		if cookie, err := r.Cookie(authCookieName); err == nil {
			s.sessions.delete(cookie.Value)
		}
	}
	http.SetCookie(w, s.authCookie(r, "", -1))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) authCookie(r *http.Request, value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     authCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   s.secureCookies || r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	}
}

func bearerAuthenticated(r *http.Request, secret string) bool {
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization == "" {
		return false
	}
	parts := strings.Fields(authorization)
	return len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && secretMatches(parts[1], secret)
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func (s *Server) requireAuthentication(w http.ResponseWriter, r *http.Request) bool {
	if authorization := strings.TrimSpace(r.Header.Get("Authorization")); authorization != "" {
		if bearerAuthenticated(r, s.sessions.secret) {
			return true
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}

	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	safeMethod := isSafeMethod(r.Method)
	record, ok := s.sessions.get(cookie.Value, safeMethod)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	if !safeMethod && !secretMatches(r.Header.Get("X-CSRF-Token"), record.csrf) {
		http.Error(w, "invalid CSRF token", http.StatusForbidden)
		return false
	}
	if !safeMethod {
		if _, ok := s.sessions.get(cookie.Value, true); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return false
		}
	}
	return true
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if s.sessions == nil || bearerAuthenticated(r, s.sessions.secret) {
		s.hub.ServeWSAuthorized(w, r, nil)
		return
	}
	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sessionToken := cookie.Value
	s.hub.ServeWSAuthorized(w, r, func(client *hubClient) bool {
		client.deliverEvent = func(event Event) bool {
			return s.sessions.deliverEvent(sessionToken, client, event)
		}
		client.onClose = func() {
			s.sessions.unregisterClient(sessionToken, client)
		}
		return s.sessions.registerClient(sessionToken, client)
	})
}
