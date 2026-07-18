package ai

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteSSEPreservesNewlines(t *testing.T) {
	var b bytes.Buffer
	require.NoError(t, WriteSSE(&b, SSEEvent{Type: "delta", Delta: "first\nsecond ☃"}))
	assert.Equal(t, "data: {\"type\":\"delta\",\"delta\":\"first\\nsecond ☃\"}\n\n", b.String())
}

type failingSSEWriter struct{ err error }

func (w failingSSEWriter) Write([]byte) (int, error) { return 0, w.err }

func TestWriteSSEReturnsWriteFailure(t *testing.T) {
	want := errors.New("closed")
	assert.ErrorIs(t, WriteSSE(failingSSEWriter{err: want}, SSEEvent{Type: "done"}), want)
}
