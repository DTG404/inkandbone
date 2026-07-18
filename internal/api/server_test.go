package api

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/digitalghost404/inkandbone/internal/ai"
	"github.com/digitalghost404/inkandbone/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	d, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })
	s := NewServer(d, t.TempDir(), nil)
	cleanupTestServer(t, s)
	return s
}

func newTestServerWithOptions(t *testing.T, options ServerOptions) *Server {
	t.Helper()
	d, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })
	s := NewServerWithOptions(d, t.TempDir(), nil, options)
	cleanupTestServer(t, s)
	return s
}

func TestServerOptionsConfigureAutomationBreakerCooldownWithoutChangingDefault(t *testing.T) {
	defaultServer := newTestServer(t)
	assert.Equal(t, time.Minute, defaultServer.breakers.cooldown)

	configured := newTestServerWithOptions(t, ServerOptions{AutomationBreakerCooldown: 100 * time.Millisecond})
	assert.Equal(t, 100*time.Millisecond, configured.breakers.cooldown)
}

func newTestServerWithDir(t *testing.T, dir string) *Server {
	t.Helper()
	d, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })
	s := NewServer(d, dir, nil)
	cleanupTestServer(t, s)
	return s
}

type stubCompleter struct {
	response string
	mu       sync.Mutex
	prompts  []string
}

func (s *stubCompleter) Generate(_ context.Context, prompt string, _ int) (string, error) {
	s.mu.Lock()
	s.prompts = append(s.prompts, prompt)
	s.mu.Unlock()
	return s.response, nil
}

func (s *stubCompleter) capturedPrompts() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.Join(s.prompts, "\n")
}

func (s *stubCompleter) promptCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.prompts)
}

func newTestServerWithAI(t *testing.T, c ai.Completer) *Server {
	t.Helper()
	d, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })
	s := NewServer(d, t.TempDir(), c)
	cleanupTestServer(t, s)
	return s
}

func cleanupTestServer(t *testing.T, s *Server) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		require.NoError(t, s.Shutdown(ctx))
	})
}
