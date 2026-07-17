package api

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/digitalghost404/inkandbone/internal/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBreakerRecoversThroughHalfOpenProbe(t *testing.T) {
	now := time.Unix(100, 0)
	r := NewBreakerRegistry(func() time.Time { return now }, 3, time.Minute)
	for range 3 {
		require.True(t, r.Allow("deepseek:auto_update_recap"))
		r.Failure("deepseek:auto_update_recap", errors.New("down"))
	}

	assert.False(t, r.Allow("deepseek:auto_update_recap"))
	assert.True(t, r.Allow("deepseek:auto_extract_npcs"), "unrelated automation remains available")
	now = now.Add(time.Minute)
	assert.True(t, r.Allow("deepseek:auto_update_recap"))
	assert.False(t, r.Allow("deepseek:auto_update_recap"), "only one half-open probe may run")
	r.Success("deepseek:auto_update_recap")
	assert.True(t, r.Allow("deepseek:auto_update_recap"))
}

func TestBreakerOpensExactlyAtThreshold(t *testing.T) {
	now := time.Unix(200, 0)
	r := NewBreakerRegistry(func() time.Time { return now }, 3, time.Minute)
	key := "anthropic:auto_extract_items"

	for i := 0; i < 2; i++ {
		require.True(t, r.Allow(key))
		r.Failure(key, errors.New("provider failed"))
		assert.True(t, r.Allow(key), "failure %d must remain below threshold", i+1)
	}
	r.Failure(key, errors.New("provider failed"))
	assert.False(t, r.Allow(key))
}

func TestBreakerHalfOpenFailureRestartsCooldown(t *testing.T) {
	now := time.Unix(300, 0)
	r := NewBreakerRegistry(func() time.Time { return now }, 1, time.Minute)
	key := "ollama:auto_generate_map"

	require.True(t, r.Allow(key))
	r.Failure(key, errors.New("first"))
	now = now.Add(time.Minute)
	require.True(t, r.Allow(key))
	r.Failure(key, errors.New("probe"))
	assert.False(t, r.Allow(key))
	now = now.Add(time.Minute - time.Nanosecond)
	assert.False(t, r.Allow(key))
	now = now.Add(time.Nanosecond)
	assert.True(t, r.Allow(key))
}

func TestBreakerIgnoresStaleResultsOutsideAnActiveProbe(t *testing.T) {
	now := time.Unix(400, 0)
	r := NewBreakerRegistry(func() time.Time { return now }, 1, time.Minute)
	key := "openrouter:auto_detect_objectives"

	require.True(t, r.Allow(key))
	r.Failure(key, errors.New("opens"))
	now = now.Add(30 * time.Second)
	r.Failure(key, errors.New("stale failure"))
	r.Success(key)
	now = now.Add(30 * time.Second)
	require.True(t, r.Allow(key), "stale results must neither extend cooldown nor close an open breaker")
	r.Success(key)
	r.Failure(key, errors.New("late duplicate probe result"))
	assert.True(t, r.Allow(key), "a stale result from a resolved probe must not re-open the breaker")
}

func TestBreakerSnapshotIsSanitizedSortedAndIndependent(t *testing.T) {
	now := time.Unix(500, 0)
	r := NewBreakerRegistry(func() time.Time { return now }, 1, time.Minute)

	for _, key := range []string{"ollama:auto_update_stats", "anthropic:auto_update_recap"} {
		require.True(t, r.Allow(key))
		r.Failure(key, errors.New("request failed for https://provider.invalid?api_key=super-secret\nheader: bearer token"))
	}

	first := r.Snapshot()
	require.Len(t, first, 2)
	assert.Equal(t, "anthropic:auto_update_recap", first[0].Key)
	assert.Equal(t, BreakerOpen, first[0].Status)
	assert.True(t, first[0].CoolingDown)
	assert.NotContains(t, first[0].LastError, "super-secret")
	assert.NotContains(t, first[0].LastError, "provider.invalid")
	first[0].Key = "mutated"
	first[0].LastError = "mutated"

	second := r.Snapshot()
	assert.Equal(t, "anthropic:auto_update_recap", second[0].Key)
	assert.NotEqual(t, "mutated", second[0].LastError)

	now = now.Add(time.Minute)
	ready := r.Snapshot()
	assert.Equal(t, BreakerHalfOpen, ready[0].Status)
	assert.False(t, ready[0].CoolingDown)
}

func TestBreakerSuccessResetsOnlyItsKey(t *testing.T) {
	now := time.Unix(600, 0)
	r := NewBreakerRegistry(func() time.Time { return now }, 2, time.Minute)
	for _, key := range []string{"deepseek:auto_update_recap", "deepseek:auto_extract_npcs"} {
		require.True(t, r.Allow(key))
		r.Failure(key, errors.New("once"))
	}
	r.Success("deepseek:auto_update_recap")

	byKey := healthByKey(r.Snapshot())
	assert.Zero(t, byKey["deepseek:auto_update_recap"].FailureCount)
	assert.Equal(t, 1, byKey["deepseek:auto_extract_npcs"].FailureCount)
}

func TestBreakerAllowsOnlyOneConcurrentHalfOpenProbe(t *testing.T) {
	now := time.Unix(700, 0)
	r := NewBreakerRegistry(func() time.Time { return now }, 1, time.Minute)
	key := "deepseek:auto_update_recap"
	require.True(t, r.Allow(key))
	r.Failure(key, errors.New("open"))
	now = now.Add(time.Minute)

	start := make(chan struct{})
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for range 64 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if r.Allow(key) {
				allowed.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	assert.EqualValues(t, 1, allowed.Load())
}

func TestBreakerNormalizesInvalidConfiguration(t *testing.T) {
	r := NewBreakerRegistry(nil, 0, -time.Second)
	require.True(t, r.Allow("unknown:auto_update_recap"))
	r.Failure("unknown:auto_update_recap", context.DeadlineExceeded)
	assert.True(t, r.Allow("unknown:auto_update_recap"), "zero cooldown permits an immediate probe")
	assert.Equal(t, "request timed out", r.Snapshot()[0].LastError)
}

func TestBreakerKeyUsesProviderNameAndUnknownOnlyForUnnamedDouble(t *testing.T) {
	tests := []struct {
		name   string
		client ai.Completer
		want   string
	}{
		{"real provider", ai.NewDeepSeekClient("key"), "deepseek:auto_update_recap"},
		{"unnamed test double", breakerUnnamedCompleter{}, "unknown:auto_update_recap"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{aiClient: tt.client}
			assert.Equal(t, tt.want, s.automationBreakerKey(settingAutoUpdateRecap))
		})
	}
}

type breakerUnnamedCompleter struct{}

func (breakerUnnamedCompleter) Generate(context.Context, string, int) (string, error) {
	return "", nil
}

func healthByKey(health []AutomationHealth) map[string]AutomationHealth {
	result := make(map[string]AutomationHealth, len(health))
	for _, entry := range health {
		result[entry.Key] = entry
	}
	return result
}
