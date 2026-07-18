package mcp

import (
	"bytes"
	"context"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type trackedReadCloser struct {
	io.ReadCloser
	closed atomic.Bool
}

func (r *trackedReadCloser) Close() error {
	r.closed.Store(true)
	return r.ReadCloser.Close()
}

func TestStartCancellationClosesInputAndJoins(t *testing.T) {
	s := newTestMCP(t)
	reader, writer := io.Pipe()
	input := &trackedReadCloser{ReadCloser: reader}
	t.Cleanup(func() { _ = writer.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Start(ctx, input, &bytes.Buffer{}) }()
	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("MCP Start did not join after cancellation")
	}
	require.True(t, input.closed.Load(), "cancellation must close the blocking input")
}

func TestStartReturnsOnEOFWithoutWaitingForCancellation(t *testing.T) {
	s := newTestMCP(t)
	reader, writer := io.Pipe()
	input := &trackedReadCloser{ReadCloser: reader}
	require.NoError(t, writer.Close())

	done := make(chan error, 1)
	go func() { done <- s.Start(context.Background(), input, &bytes.Buffer{}) }()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("MCP Start did not return on EOF")
	}
}
