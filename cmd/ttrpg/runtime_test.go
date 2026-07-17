package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type eventLog struct {
	mu     sync.Mutex
	events []string
}

func (l *eventLog) add(event string) {
	l.mu.Lock()
	l.events = append(l.events, event)
	l.mu.Unlock()
}

func (l *eventLog) index(event string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, candidate := range l.events {
		if candidate == event {
			return i
		}
	}
	return -1
}

type fakeHTTPService struct {
	log           *eventLog
	startErr      error
	shutdownErr   error
	shutdownCalls int
	started       chan struct{}
	stopped       chan struct{}
	stopOnce      sync.Once
	stopErr       error
}

func newFakeHTTPService(log *eventLog) *fakeHTTPService {
	return &fakeHTTPService{log: log, started: make(chan struct{}), stopped: make(chan struct{})}
}

func (s *fakeHTTPService) Start(string, string, string) error {
	s.log.add("http_start")
	close(s.started)
	if s.startErr != nil {
		return s.startErr
	}
	<-s.stopped
	if s.stopErr != nil {
		return s.stopErr
	}
	return http.ErrServerClosed
}

func (s *fakeHTTPService) Shutdown(context.Context) error {
	s.log.add("http_shutdown")
	s.stopOnce.Do(func() { close(s.stopped) })
	if s.shutdownCalls == 0 {
		s.shutdownCalls++
		return s.shutdownErr
	}
	s.shutdownCalls++
	return nil
}

func (s *fakeHTTPService) Close() error {
	s.log.add("http_force_close")
	s.stopOnce.Do(func() { close(s.stopped) })
	return nil
}

type fakeMCPService struct {
	log      *eventLog
	startErr error
	started  chan struct{}
}

func (s *fakeMCPService) Start(ctx context.Context, _ io.ReadCloser, _ io.Writer) error {
	s.log.add("mcp_start")
	close(s.started)
	if s.startErr != nil {
		return s.startErr
	}
	<-ctx.Done()
	s.log.add("mcp_stop")
	return nil
}

type nopReadCloser struct{ io.Reader }

func (nopReadCloser) Close() error { return nil }

func TestRunServicesReturnsStartFailureAfterJoiningMCPAndClosingDB(t *testing.T) {
	log := &eventLog{}
	startErr := errors.New("bind failed")
	web := newFakeHTTPService(log)
	web.startErr = startErr
	mcp := &fakeMCPService{log: log, started: make(chan struct{})}
	root, cancel := context.WithCancel(context.Background())

	err := runServices(root, cancel, web, mcp, true,
		nopReadCloser{Reader: bytes.NewReader(nil)}, &bytes.Buffer{},
		func() error { log.add("db_close"); return nil },
		runtimeConfig{shutdownTimeout: time.Second})
	require.ErrorIs(t, err, startErr)
	assert.Greater(t, log.index("mcp_stop"), log.index("mcp_start"))
	assert.Greater(t, log.index("db_close"), log.index("mcp_stop"))
}

func TestRunServicesForceClosesHTTPBeforeDBOnShutdownFailure(t *testing.T) {
	log := &eventLog{}
	shutdownErr := errors.New("graceful deadline")
	web := newFakeHTTPService(log)
	web.shutdownErr = shutdownErr
	mcp := &fakeMCPService{log: log, started: make(chan struct{})}
	root, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runServices(root, cancel, web, mcp, true,
			nopReadCloser{Reader: bytes.NewReader(nil)}, &bytes.Buffer{},
			func() error { log.add("db_close"); return nil },
			runtimeConfig{shutdownTimeout: time.Second})
	}()

	select {
	case <-web.started:
	case <-time.After(time.Second):
		t.Fatal("HTTP service did not start")
	}
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, shutdownErr)
	case <-time.After(time.Second):
		t.Fatal("runtime did not stop")
	}
	assert.Greater(t, log.index("http_force_close"), log.index("http_shutdown"))
	assert.Greater(t, log.index("db_close"), log.index("http_force_close"))
	assert.Greater(t, log.index("db_close"), log.index("mcp_stop"))
}

func TestRunServicesReturnsMCPFailureAfterStoppingHTTPAndClosingDB(t *testing.T) {
	log := &eventLog{}
	web := newFakeHTTPService(log)
	mcpErr := errors.New("stdio failed")
	mcp := &fakeMCPService{log: log, startErr: mcpErr, started: make(chan struct{})}
	root, cancel := context.WithCancel(context.Background())

	err := runServices(root, cancel, web, mcp, true,
		nopReadCloser{Reader: bytes.NewReader(nil)}, &bytes.Buffer{},
		func() error { log.add("db_close"); return nil },
		runtimeConfig{shutdownTimeout: time.Second})
	require.ErrorIs(t, err, mcpErr)
	require.NotEqual(t, -1, log.index("http_start"))
	require.NotEqual(t, -1, log.index("http_shutdown"))
	assert.Greater(t, log.index("db_close"), log.index("http_start"))
	assert.Greater(t, log.index("db_close"), log.index("http_shutdown"))
}

func TestRunServicesTreatsWrappedCancellationAsNormalJoinedStop(t *testing.T) {
	log := &eventLog{}
	web := newFakeHTTPService(log)
	web.stopErr = fmt.Errorf("listener stopped: %w", context.Canceled)
	root, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runServices(root, cancel, web, nil, false,
			nopReadCloser{Reader: bytes.NewReader(nil)}, &bytes.Buffer{},
			func() error { log.add("db_close"); return nil },
			runtimeConfig{shutdownTimeout: time.Second})
	}()

	select {
	case <-web.started:
	case <-time.After(time.Second):
		t.Fatal("HTTP service did not start")
	}
	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("runtime did not join wrapped cancellation")
	}
	assert.Greater(t, log.index("db_close"), log.index("http_shutdown"))
}

func TestRunHelpWritesUsageAndReturnsSuccess(t *testing.T) {
	var output bytes.Buffer
	err := run([]string{"-h"}, nil, &output)
	require.NoError(t, err)
	assert.Contains(t, output.String(), "Usage of ttrpg:")
	assert.Contains(t, output.String(), "-listen")
}

func TestSetupFailureJoinsDatabaseCloseError(t *testing.T) {
	setupErr := errors.New("embedded filesystem failed")
	closeErr := errors.New("database close failed")
	err := setupFailure(setupErr, func() error { return closeErr })
	require.ErrorIs(t, err, setupErr)
	require.ErrorIs(t, err, closeErr)
}
