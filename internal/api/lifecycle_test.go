package api

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var lifecycleHTTPClient = &http.Client{Timeout: time.Second}

func startLifecycleServer(t *testing.T, s *Server) (string, <-chan error) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	serveErr := make(chan error, 1)
	go func() { serveErr <- s.startOnListener(listener, "", "") }()
	select {
	case <-s.httpServerReady:
	case err := <-serveErr:
		t.Fatalf("server stopped before installing owned http.Server: %v", err)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for owned http.Server installation")
	}
	return "http://" + listener.Addr().String(), serveErr
}

func TestLifecycleInvalidTLSClosesOwnedListener(t *testing.T) {
	s := newTestServer(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()

	err = s.startOnListener(listener, t.TempDir()+"/missing-cert.pem", t.TempDir()+"/missing-key.pem")
	require.Error(t, err)

	rebound, err := net.Listen("tcp", address)
	require.NoError(t, err, "Start must close its listener when TLS setup fails")
	require.NoError(t, rebound.Close())
}

func TestLifecycleShutdownWaitsForInflightHandler(t *testing.T) {
	s := newTestServer(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	s.mux.HandleFunc("GET /lifecycle-block", func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusNoContent)
	})

	baseURL, serveErr := startLifecycleServer(t, s)

	requestDone := make(chan error, 1)
	go func() {
		response, err := lifecycleHTTPClient.Get(baseURL + "/lifecycle-block") //nolint:gosec
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
	require.NoError(t, receiveWithin(t, shutdownDone, time.Second, "shutdown completion"))
	require.NoError(t, receiveWithin(t, requestDone, time.Second, "request completion"))
	assert.ErrorIs(t, receiveWithin(t, serveErr, time.Second, "server stop"), http.ErrServerClosed)
	retryCtx, cancelRetry := context.WithTimeout(context.Background(), time.Second)
	defer cancelRetry()
	require.NoError(t, s.Shutdown(retryCtx), "Shutdown must be idempotent")
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

	baseURL, serveErr := startLifecycleServer(t, s)
	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		response, _ := lifecycleHTTPClient.Get(baseURL + "/lifecycle-context") //nolint:gosec
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
	assert.ErrorIs(t, receiveWithin(t, serveErr, time.Second, "server stop"), http.ErrServerClosed)
	select {
	case <-requestDone:
	case <-time.After(time.Second):
		t.Fatal("canceled request did not finish")
	}
}

func TestLifecycleStartOwnsHardenedHTTPServerAndRejectsDoubleStart(t *testing.T) {
	s := newTestServer(t)
	_, serveErr := startLifecycleServer(t, s)

	s.httpServerMu.Lock()
	owned := s.httpServer
	s.httpServerMu.Unlock()
	require.NotNil(t, owned)
	assert.Equal(t, 10*time.Second, owned.ReadHeaderTimeout)
	assert.Equal(t, 120*time.Second, owned.IdleTimeout)
	assert.Equal(t, 1<<20, owned.MaxHeaderBytes)

	secondListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	doubleStart := make(chan error, 1)
	go func() { doubleStart <- s.startOnListener(secondListener, "", "") }()
	select {
	case err := <-doubleStart:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("second Start call blocked")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, s.Shutdown(ctx))
	assert.ErrorIs(t, receiveWithin(t, serveErr, time.Second, "server stop"), http.ErrServerClosed)
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

	baseURL, serveErr := startLifecycleServer(t, s)
	requestDone := make(chan error, 1)
	go func() {
		response, err := lifecycleHTTPClient.Get(baseURL + "/lifecycle-retry") //nolint:gosec
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
	require.NoError(t, receiveWithin(t, requestDone, time.Second, "retried request completion"))
	assert.ErrorIs(t, receiveWithin(t, serveErr, time.Second, "server stop"), http.ErrServerClosed)
}
