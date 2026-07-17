package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeoutProviderClientsUseBoundedStreamingTransport(t *testing.T) {
	clients := map[string]*http.Client{
		"anthropic":  NewClient("test").http,
		"deepseek":   NewDeepSeekClient("test").http,
		"openrouter": NewOpenRouterClient("test").http,
		"ollama":     NewOllamaClient("test").http,
	}

	for name, client := range clients {
		t.Run(name, func(t *testing.T) {
			require.NotNil(t, client.Transport)
			transport, ok := client.Transport.(*http.Transport)
			require.True(t, ok, "provider must use the shared HTTP transport")
			assert.Zero(t, client.Timeout, "streaming clients must not use http.Client.Timeout")
			assert.Positive(t, transport.ResponseHeaderTimeout)
			assert.Positive(t, transport.TLSHandshakeTimeout)
			assert.Positive(t, transport.IdleConnTimeout)
		})
	}
}

func TestTimeoutAutomationRequestHasAbsoluteDeadline(t *testing.T) {
	old := automationRequestTimeout
	automationRequestTimeout = 60 * time.Millisecond
	t.Cleanup(func() { automationRequestTimeout = old })

	requestCanceled := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		<-r.Context().Done()
		close(requestCanceled)
	}))
	t.Cleanup(upstream.Close)

	started := time.Now()
	_, err := NewClientWithURL("test", upstream.URL).Generate(context.Background(), "prompt", 16)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(started), 500*time.Millisecond)
	select {
	case <-requestCanceled:
	case <-time.After(time.Second):
		t.Fatal("provider request was not canceled upstream")
	}
}

func TestTimeoutProviderStopsWaitingForResponseHeaders(t *testing.T) {
	old := responseHeaderTimeout
	responseHeaderTimeout = 50 * time.Millisecond
	t.Cleanup(func() { responseHeaderTimeout = old })

	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		<-release
	}))
	t.Cleanup(func() {
		close(release)
		upstream.Close()
	})

	started := time.Now()
	_, err := NewClientWithURL("test", upstream.URL).Generate(context.Background(), "prompt", 16)
	require.Error(t, err)
	var timeoutError interface{ Timeout() bool }
	require.ErrorAs(t, err, &timeoutError)
	assert.True(t, timeoutError.Timeout())
	assert.Less(t, time.Since(started), 500*time.Millisecond)
}

func TestTimeoutGMStreamSurvivesPeriodicBytesBeyondHeaderTimeout(t *testing.T) {
	oldHeader, oldGM := responseHeaderTimeout, gmRequestTimeout
	responseHeaderTimeout = 25 * time.Millisecond
	gmRequestTimeout = 500 * time.Millisecond
	t.Cleanup(func() {
		responseHeaderTimeout = oldHeader
		gmRequestTimeout = oldGM
	})

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		for i := 0; i < 5; i++ {
			fmt.Fprintf(w, "data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"x\"}}\n\n")
			flusher.Flush()
			time.Sleep(20 * time.Millisecond)
		}
	}))
	t.Cleanup(upstream.Close)

	text, err := NewClientWithURL("test", upstream.URL).StreamRespond(
		context.Background(), "system", []ChatMessage{{Role: "user", Content: "hello"}}, 16, httptest.NewRecorder(),
	)
	require.NoError(t, err)
	assert.Equal(t, "xxxxx", text)
}

func TestTimeoutGMStreamHasAbsoluteDeadline(t *testing.T) {
	oldHeader, oldGM := responseHeaderTimeout, gmRequestTimeout
	responseHeaderTimeout = 100 * time.Millisecond
	gmRequestTimeout = 75 * time.Millisecond
	t.Cleanup(func() {
		responseHeaderTimeout = oldHeader
		gmRequestTimeout = oldGM
	})

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		for {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(10 * time.Millisecond):
				fmt.Fprintf(w, "data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"x\"}}\n\n")
				flusher.Flush()
			}
		}
	}))
	t.Cleanup(upstream.Close)

	_, err := NewClientWithURL("test", upstream.URL).StreamRespond(
		context.Background(), "system", []ChatMessage{{Role: "user", Content: "hello"}}, 16, httptest.NewRecorder(),
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded), "expected deadline error, got %v", err)
}

func TestTimeoutCallerCancellationWinsOverProviderDeadline(t *testing.T) {
	old := automationRequestTimeout
	automationRequestTimeout = time.Second
	t.Cleanup(func() { automationRequestTimeout = old })

	upstream := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(upstream.Close)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewClientWithURL("test", upstream.URL).Generate(ctx, "prompt", 16)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
