package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/digitalghost404/inkandbone/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAuthCookieName = "ttrpg_session"

func newSecureTestServer(t *testing.T, secret string, origin string) *Server {
	t.Helper()
	database, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	s := NewServerWithOptions(database, t.TempDir(), nil, ServerOptions{
		Security: ListenSecurityConfig{
			AuthSecret:     secret,
			TLSCertFile:    "configured-cert.pem",
			TLSKeyFile:     "configured-key.pem",
			AllowedOrigins: []string{origin},
		},
	})
	s.mux.HandleFunc("POST /api/test-mutation", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	s.mux.HandleFunc("GET /login-probe", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	return s
}

func loginTestSession(t *testing.T, s *Server, secret, remoteAddr string) (*http.Cookie, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"secret":"`+secret+`"}`))
	req.RemoteAddr = remoteAddr
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
	cookie := requireAuthCookie(t, w.Result().Cookies())

	req = httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var info SessionInfo
	require.NoError(t, json.NewDecoder(w.Body).Decode(&info))
	require.True(t, info.Authenticated)
	require.NotEmpty(t, info.CSRFToken)
	return cookie, info.CSRFToken
}

func requireAuthCookie(t *testing.T, cookies []*http.Cookie) *http.Cookie {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == testAuthCookieName {
			return cookie
		}
	}
	require.FailNow(t, "authentication cookie not found")
	return nil
}

func TestAuthLoginCreatesOpaqueSession(t *testing.T) {
	secret := strings.Repeat("s", 32)
	s := newSecureTestServer(t, secret, "https://table.example")

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"secret":"`+secret+`"}`))
	req.RemoteAddr = "192.0.2.4:1234"
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	cookie := requireAuthCookie(t, w.Result().Cookies())
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)
	assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
	assert.Equal(t, "/", cookie.Path)
	assert.NotEqual(t, secret, cookie.Value)
	assert.NotEmpty(t, cookie.Value)
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}

func TestAuthSessionResponseIsNotCacheable(t *testing.T) {
	secret := strings.Repeat("s", 32)
	s := newSecureTestServer(t, secret, "https://table.example")
	cookie, _ := loginTestSession(t, s, secret, "192.0.2.4:1234")
	req := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}

func TestAuthLoginRejectsInvalidSecret(t *testing.T) {
	s := newSecureTestServer(t, strings.Repeat("s", 32), "https://table.example")
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"secret":"wrong"}`))
	req.RemoteAddr = "192.0.2.4:1234"
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, w.Result().Cookies())
}

func TestAuthLoginRateLimitsFailedAttemptsPerAddress(t *testing.T) {
	s := newSecureTestServer(t, strings.Repeat("s", 32), "https://table.example")
	for attempt := 1; attempt <= 6; attempt++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"secret":"wrong"}`))
		req.RemoteAddr = "192.0.2.4:1234"
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
		if attempt <= 5 {
			assert.Equal(t, http.StatusUnauthorized, w.Code, "attempt %d", attempt)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, w.Code, "attempt %d", attempt)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"secret":"wrong"}`))
	req.RemoteAddr = "198.51.100.8:1234"
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthLoginConcurrentFailuresAreAtomicallyLimited(t *testing.T) {
	s := newSecureTestServer(t, strings.Repeat("s", 32), "https://table.example")
	const attempts = 20
	var comparisons atomic.Int32
	enteredComparison := make(chan struct{}, attempts)
	releaseComparisons := make(chan struct{})
	s.sessions.compareSecret = func(_, _ string) bool {
		comparisons.Add(1)
		enteredComparison <- struct{}{}
		<-releaseComparisons
		return false
	}

	start := make(chan struct{})
	statuses := make(chan int, attempts)
	for range attempts {
		go func() {
			<-start
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"secret":"wrong"}`))
			req.RemoteAddr = "192.0.2.4:1234"
			w := httptest.NewRecorder()
			s.ServeHTTP(w, req)
			statuses <- w.Code
		}()
	}
	close(start)

	for range maxLoginFailures {
		select {
		case <-enteredComparison:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for reserved credential comparisons")
		}
	}
	for range attempts - maxLoginFailures {
		select {
		case status := <-statuses:
			assert.Equal(t, http.StatusTooManyRequests, status)
		case <-time.After(time.Second):
			t.Fatal("rate-limited attempts did not complete while comparisons were blocked")
		}
	}
	select {
	case <-enteredComparison:
		t.Fatal("more than five concurrent attempts reached credential comparison")
	default:
	}

	close(releaseComparisons)
	for range maxLoginFailures {
		assert.Equal(t, http.StatusUnauthorized, <-statuses)
	}
	assert.EqualValues(t, maxLoginFailures, comparisons.Load())
}

func TestAuthLoginPrunesExpiredFailureAddresses(t *testing.T) {
	s := newSecureTestServer(t, strings.Repeat("s", 32), "https://table.example")
	now := time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC)
	s.sessions.now = func() time.Time { return now }

	failedLogin(t, s, "192.0.2.1:1234")
	failedLogin(t, s, "192.0.2.2:1234")
	require.Len(t, s.sessions.failures, 2)

	now = now.Add(loginAttemptWindow + time.Second)
	failedLogin(t, s, "192.0.2.3:1234")

	require.Len(t, s.sessions.failures, 1)
	_, retained := s.sessions.failures["192.0.2.3"]
	assert.True(t, retained)
}

func TestAuthLoginFailurePruningIsThrottled(t *testing.T) {
	s := newSecureTestServer(t, strings.Repeat("s", 32), "https://table.example")
	now := time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC)
	s.sessions.now = func() time.Time { return now }

	failedLogin(t, s, "192.0.2.1:1234")
	firstPrune := s.sessions.lastFailurePrune
	for address := 2; address <= 50; address++ {
		failedLogin(t, s, fmt.Sprintf("192.0.2.%d:1234", address))
	}

	assert.Equal(t, now, firstPrune)
	assert.Equal(t, firstPrune, s.sessions.lastFailurePrune)
}

func failedLogin(t *testing.T, s *Server, remoteAddr string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"secret":"wrong"}`))
	req.RemoteAddr = remoteAddr
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthCookieRequiresCSRFForUnsafeRequests(t *testing.T) {
	secret := strings.Repeat("s", 32)
	s := newSecureTestServer(t, secret, "https://table.example")
	cookie, csrf := loginTestSession(t, s, secret, "192.0.2.4:1234")

	for _, tt := range []struct {
		name   string
		token  string
		status int
	}{
		{name: "missing", status: http.StatusForbidden},
		{name: "wrong", token: "wrong", status: http.StatusForbidden},
		{name: "valid", token: csrf, status: http.StatusNoContent},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/test-mutation", nil)
			req.AddCookie(cookie)
			if tt.token != "" {
				req.Header.Set("X-CSRF-Token", tt.token)
			}
			w := httptest.NewRecorder()
			s.ServeHTTP(w, req)
			assert.Equal(t, tt.status, w.Code)
		})
	}
}

func TestAuthFailedCSRFDoesNotRefreshIdleLifetime(t *testing.T) {
	secret := strings.Repeat("s", 32)
	s := newSecureTestServer(t, secret, "https://table.example")
	now := time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC)
	s.sessions.now = func() time.Time { return now }
	cookie, _ := loginTestSession(t, s, secret, "192.0.2.4:1234")
	initialLastSeen := s.sessions.sessions[cookie.Value].lastSeen

	now = now.Add(29 * time.Minute)
	req := httptest.NewRequest(http.MethodPost, "/api/test-mutation", nil)
	req.AddCookie(cookie)
	req.Header.Set("X-CSRF-Token", "wrong")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, initialLastSeen, s.sessions.sessions[cookie.Value].lastSeen)

	now = now.Add(2 * time.Minute)
	req = httptest.NewRequest(http.MethodGet, "/api/context", nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthBearerMasterSecretDoesNotRequireCSRF(t *testing.T) {
	secret := strings.Repeat("s", 32)
	s := newSecureTestServer(t, secret, "https://table.example")
	req := httptest.NewRequest(http.MethodPost, "/api/test-mutation", nil)
	req.Header.Set("Authorization", "Bearer "+secret)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestAuthLogoutRevokesSession(t *testing.T) {
	secret := strings.Repeat("s", 32)
	s := newSecureTestServer(t, secret, "https://table.example")
	cookie, csrf := loginTestSession(t, s, secret, "192.0.2.4:1234")
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(cookie)
	req.Header.Set("X-CSRF-Token", csrf)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	cleared := requireAuthCookie(t, w.Result().Cookies())
	assert.Less(t, cleared.MaxAge, 0)

	req = httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	var info SessionInfo
	require.NoError(t, json.NewDecoder(w.Body).Decode(&info))
	assert.False(t, info.Authenticated)
}

func TestAuthSessionExpiresAfterIdleAndAbsoluteLifetimes(t *testing.T) {
	secret := strings.Repeat("s", 32)
	start := time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC)

	t.Run("idle", func(t *testing.T) {
		s := newSecureTestServer(t, secret, "https://table.example")
		now := start
		s.sessions.now = func() time.Time { return now }
		cookie, _ := loginTestSession(t, s, secret, "192.0.2.4:1234")
		now = now.Add(30*time.Minute + time.Second)
		assertSessionAuthenticated(t, s, cookie, false)
	})

	t.Run("absolute", func(t *testing.T) {
		s := newSecureTestServer(t, secret, "https://table.example")
		now := start
		s.sessions.now = func() time.Time { return now }
		cookie, _ := loginTestSession(t, s, secret, "192.0.2.4:1234")
		for range 24 {
			now = now.Add(29 * time.Minute)
			assertSessionAuthenticated(t, s, cookie, true)
		}
		now = now.Add(25 * time.Minute)
		assertSessionAuthenticated(t, s, cookie, false)
	})
}

func assertSessionAuthenticated(t *testing.T, s *Server, cookie *http.Cookie, want bool) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var info SessionInfo
	require.NoError(t, json.NewDecoder(w.Body).Decode(&info))
	assert.Equal(t, want, info.Authenticated)
}

func TestAuthOnlyProtectsAPIAndWebSocketWhenEnabled(t *testing.T) {
	s := newSecureTestServer(t, strings.Repeat("s", 32), "https://table.example")

	for _, tt := range []struct {
		path   string
		status int
	}{
		{path: "/api/health", status: http.StatusOK},
		{path: "/api/context", status: http.StatusUnauthorized},
		{path: "/ws", status: http.StatusUnauthorized},
		{path: "/login-probe", status: http.StatusNoContent},
	} {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
		assert.Equal(t, tt.status, w.Code, tt.path)
	}

	loopback := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/context", nil)
	w := httptest.NewRecorder()
	loopback.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusUnauthorized, w.Code)
}

func TestAuthSecretMatchesExactly(t *testing.T) {
	want := strings.Repeat("s", 32)
	assert.True(t, secretMatches(want, want))
	assert.False(t, secretMatches(strings.Repeat("s", 31), want))
	assert.False(t, secretMatches(strings.Repeat("x", 32), want))
}
