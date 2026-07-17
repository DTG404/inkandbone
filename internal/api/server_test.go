package api

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/digitalghost404/inkandbone/internal/ai"
	"github.com/digitalghost404/inkandbone/internal/db"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	d, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })
	return NewServer(d, t.TempDir(), nil)
}

func newTestServerWithOptions(t *testing.T, options ServerOptions) *Server {
	t.Helper()
	d, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })
	return NewServerWithOptions(d, t.TempDir(), nil, options)
}

func newTestServerWithDir(t *testing.T, dir string) *Server {
	t.Helper()
	d, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })
	return NewServer(d, dir, nil)
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
	return NewServer(d, t.TempDir(), c)
}
