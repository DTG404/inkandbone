package api

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func reserveAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	return address
}

func waitForServer(t *testing.T, address string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 20*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("server did not listen on %s", address)
}

func TestLifecycleShutdownWaitsForInflightHandler(t *testing.T) {
	s := newTestServer(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	s.mux.HandleFunc("GET /lifecycle-block", func(w http.ResponseWriter, _ *http.Request) {
		once.Do(func() { close(entered) })
		<-release
		w.WriteHeader(http.StatusNoContent)
	})

	address := reserveAddress(t)
	serveErr := make(chan error, 1)
	go func() { serveErr <- s.Start(address, "", "") }()
	waitForServer(t, address)

	requestDone := make(chan error, 1)
	go func() {
		response, err := http.Get("http://" + address + "/lifecycle-block") //nolint:gosec
		if response != nil {
			_ = response.Body.Close()
		}
		requestDone <- err
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("handler did not start")
	}

	shutdownDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		shutdownDone <- s.Shutdown(ctx)
	}()
	select {
	case err := <-shutdownDone:
		t.Fatalf("Shutdown returned before handler completed: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	require.NoError(t, <-shutdownDone)
	require.NoError(t, <-requestDone)
	assert.True(t, errors.Is(<-serveErr, http.ErrServerClosed))
	require.NoError(t, s.Shutdown(context.Background()), "Shutdown must be idempotent")
}

func TestLifecycleRootCancellationReachesHandlers(t *testing.T) {
	root, cancelRoot := context.WithCancel(context.Background())
	s := newTestServerWithOptions(t, ServerOptions{RootContext: root})
	entered := make(chan struct{})
	canceled := make(chan struct{})
	s.mux.HandleFunc("GET /lifecycle-context", func(_ http.ResponseWriter, r *http.Request) {
		close(entered)
		<-r.Context().Done()
		close(canceled)
	})

	address := reserveAddress(t)
	serveErr := make(chan error, 1)
	go func() { serveErr <- s.Start(address, "", "") }()
	waitForServer(t, address)
	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		response, _ := http.Get("http://" + address + "/lifecycle-context") //nolint:gosec
		if response != nil {
			_ = response.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("handler did not start")
	}
	cancelRoot()
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("root cancellation did not reach handler")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, s.Shutdown(ctx))
	assert.True(t, errors.Is(<-serveErr, http.ErrServerClosed))
	select {
	case <-requestDone:
	case <-time.After(time.Second):
		t.Fatal("canceled request did not finish")
	}
}

func TestLifecycleStartOwnsHardenedHTTPServerAndRejectsDoubleStart(t *testing.T) {
	s := newTestServer(t)
	address := reserveAddress(t)
	serveErr := make(chan error, 1)
	go func() { serveErr <- s.Start(address, "", "") }()
	waitForServer(t, address)

	s.httpServerMu.Lock()
	owned := s.httpServer
	s.httpServerMu.Unlock()
	require.NotNil(t, owned)
	assert.Equal(t, 10*time.Second, owned.ReadHeaderTimeout)
	assert.Equal(t, 120*time.Second, owned.IdleTimeout)
	assert.Equal(t, 1<<20, owned.MaxHeaderBytes)

	secondAddress := reserveAddress(t)
	doubleStart := make(chan error, 1)
	go func() { doubleStart <- s.Start(secondAddress, "", "") }()
	select {
	case err := <-doubleStart:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("second Start call blocked")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, s.Shutdown(ctx))
	assert.True(t, errors.Is(<-serveErr, http.ErrServerClosed))
}

func TestLifecycleShutdownDeadlineCanBeRetried(t *testing.T) {
	s := newTestServer(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	s.mux.HandleFunc("GET /lifecycle-retry", func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusNoContent)
	})

	address := reserveAddress(t)
	serveErr := make(chan error, 1)
	go func() { serveErr <- s.Start(address, "", "") }()
	waitForServer(t, address)
	requestDone := make(chan error, 1)
	go func() {
		response, err := http.Get("http://" + address + "/lifecycle-retry") //nolint:gosec
		if response != nil {
			_ = response.Body.Close()
		}
		requestDone <- err
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("handler did not start")
	}

	shortCtx, cancelShort := context.WithTimeout(context.Background(), 25*time.Millisecond)
	err := s.Shutdown(shortCtx)
	cancelShort()
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	close(release)
	retryCtx, cancelRetry := context.WithTimeout(context.Background(), time.Second)
	defer cancelRetry()
	require.NoError(t, s.Shutdown(retryCtx))
	require.NoError(t, <-requestDone)
	assert.True(t, errors.Is(<-serveErr, http.ErrServerClosed))
}
