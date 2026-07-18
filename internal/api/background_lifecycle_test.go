package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/digitalghost404/inkandbone/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedPendingEmbedding(t *testing.T, database *db.DB, content string) int64 {
	t.Helper()
	ruleset, err := database.GetRulesetByName("dnd5e")
	require.NoError(t, err)
	require.NotNil(t, ruleset)
	require.NoError(t, database.CreateRulebookChunks(ruleset.ID, []db.RulebookChunk{{
		Source: "Lifecycle Test", Heading: "Test", Content: content,
	}}))
	return ruleset.ID
}

func receiveWithin[T any](t *testing.T, ch <-chan T, timeout time.Duration, label string) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for %s", label)
		var zero T
		return zero
	}
}

func TestLifecycleShutdownWaitsForStartupEmbeddingJob(t *testing.T) {
	database, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	seedPendingEmbedding(t, database, "startup chunk")

	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	s := NewServerWithOptions(database, t.TempDir(), nil, ServerOptions{
		embedText: func(context.Context, string) ([]float32, error) {
			close(entered)
			<-release
			return []float32{1}, nil
		},
	})
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	})
	receiveWithin(t, entered, time.Second, "startup embedding job")

	shutdownDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		shutdownDone <- s.Shutdown(ctx)
	}()
	select {
	case err := <-shutdownDone:
		t.Fatalf("Shutdown returned before DB-owning job completed: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	releaseOnce.Do(func() { close(release) })
	require.NoError(t, receiveWithin(t, shutdownDone, time.Second, "background-aware shutdown"))
}

func TestLifecycleCanceledEmbeddingLoopStopsBeforeNextChunk(t *testing.T) {
	database, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	rulesetID := seedPendingEmbedding(t, database, "first chunk")
	require.NoError(t, database.CreateRulebookChunks(rulesetID, []db.RulebookChunk{{
		Source: "Lifecycle Test", Heading: "Second", Content: "second chunk",
	}}))

	entered := make(chan struct{})
	var calls atomic.Int32
	s := NewServerWithOptions(database, t.TempDir(), nil, ServerOptions{
		embedText: func(ctx context.Context, _ string) ([]float32, error) {
			if calls.Add(1) == 1 {
				close(entered)
			}
			<-ctx.Done()
			return nil, ctx.Err()
		},
	})
	cleanupTestServer(t, s)
	receiveWithin(t, entered, time.Second, "first embedding call")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, s.Shutdown(ctx))
	assert.Equal(t, int32(1), calls.Load(), "canceled loop must not start the next chunk")
}

func TestLifecycleRejectsNewDatabaseJobsAfterShutdownStarts(t *testing.T) {
	s := newTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	require.NoError(t, s.Shutdown(ctx))
	cancel()

	ran := make(chan struct{})
	accepted := s.startLifecycleJob(func(context.Context) { close(ran) })
	assert.False(t, accepted)
	select {
	case <-ran:
		t.Fatal("job ran after shutdown began")
	case <-time.After(30 * time.Millisecond):
	}
}

func TestLifecycleShutdownWaitsForRulebookEmbeddingJob(t *testing.T) {
	database, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	ruleset, err := database.GetRulesetByName("dnd5e")
	require.NoError(t, err)

	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	s := NewServerWithOptions(database, t.TempDir(), nil, ServerOptions{
		embedText: func(context.Context, string) ([]float32, error) {
			close(entered)
			<-release
			return []float32{1}, nil
		},
	})
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	})
	req := httptest.NewRequest(http.MethodPost,
		"/api/rulesets/"+stringID(ruleset.ID)+"/rulebook?source=Tracked",
		strings.NewReader("# Heading\nTracked content."))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	receiveWithin(t, entered, time.Second, "rulebook embedding job")

	shutdownDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		shutdownDone <- s.Shutdown(ctx)
	}()
	select {
	case err := <-shutdownDone:
		t.Fatalf("Shutdown returned before rulebook embedding completed: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	releaseOnce.Do(func() { close(release) })
	require.NoError(t, receiveWithin(t, shutdownDone, time.Second, "rulebook-aware shutdown"))
}

type cancelObservingCompleter struct {
	started chan struct{}
}

func (c *cancelObservingCompleter) Generate(ctx context.Context, _ string, _ int) (string, error) {
	close(c.started)
	<-ctx.Done()
	return "", ctx.Err()
}

func TestTalentDescriptionPropagatesRequestCancellation(t *testing.T) {
	completer := &cancelObservingCompleter{started: make(chan struct{})}
	s := newTestServerWithAI(t, completer)
	requestCtx, cancelRequest := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet,
		"/api/talent-description?name=Celerity&system=vtm", nil).WithContext(requestCtx)
	w := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		s.ServeHTTP(w, req)
		close(done)
	}()
	receiveWithin(t, completer.started, time.Second, "talent AI request")
	cancelRequest()
	receiveWithin(t, done, time.Second, "canceled talent handler")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.True(t, errors.Is(requestCtx.Err(), context.Canceled))
}

func stringID(id int64) string {
	return strconv.FormatInt(id, 10)
}
