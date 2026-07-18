package api

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
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
		permit, ok := r.Acquire("deepseek:auto_update_recap")
		require.True(t, ok)
		permit.Failure(errors.New("down"))
	}

	_, allowed := r.Acquire("deepseek:auto_update_recap")
	assert.False(t, allowed)
	_, allowed = r.Acquire("deepseek:auto_extract_npcs")
	assert.True(t, allowed, "unrelated automation remains available")
	now = now.Add(time.Minute)
	probe, allowed := r.Acquire("deepseek:auto_update_recap")
	require.True(t, allowed)
	_, allowed = r.Acquire("deepseek:auto_update_recap")
	assert.False(t, allowed, "only one half-open probe may run")
	probe.Success()
	_, allowed = r.Acquire("deepseek:auto_update_recap")
	assert.True(t, allowed)
}

func TestBreakerOpensExactlyAtThreshold(t *testing.T) {
	now := time.Unix(200, 0)
	r := NewBreakerRegistry(func() time.Time { return now }, 3, time.Minute)
	key := "anthropic:auto_extract_items"

	for i := 0; i < 2; i++ {
		permit, ok := r.Acquire(key)
		require.True(t, ok)
		permit.Failure(errors.New("provider failed"))
		assert.Equal(t, i+1, healthByKey(r.Snapshot())[key].FailureCount)
	}
	permit, ok := r.Acquire(key)
	require.True(t, ok)
	permit.Failure(errors.New("provider failed"))
	_, allowed := r.Acquire(key)
	assert.False(t, allowed)
}

func TestBreakerHalfOpenFailureRestartsCooldown(t *testing.T) {
	now := time.Unix(300, 0)
	r := NewBreakerRegistry(func() time.Time { return now }, 1, time.Minute)
	key := "ollama:auto_generate_map"

	permit, ok := r.Acquire(key)
	require.True(t, ok)
	permit.Failure(errors.New("first"))
	now = now.Add(time.Minute)
	probe, ok := r.Acquire(key)
	require.True(t, ok)
	probe.Failure(errors.New("probe"))
	_, allowed := r.Acquire(key)
	assert.False(t, allowed)
	now = now.Add(time.Minute - time.Nanosecond)
	_, allowed = r.Acquire(key)
	assert.False(t, allowed)
	now = now.Add(time.Nanosecond)
	_, allowed = r.Acquire(key)
	assert.True(t, allowed)
}

func TestBreakerSnapshotIsSanitizedSortedAndIndependent(t *testing.T) {
	now := time.Unix(500, 0)
	r := NewBreakerRegistry(func() time.Time { return now }, 1, time.Minute)

	for _, key := range []string{"ollama:auto_update_stats", "anthropic:auto_update_recap"} {
		permit, ok := r.Acquire(key)
		require.True(t, ok)
		permit.Failure(errors.New("request failed for https://provider.invalid?api_key=super-secret\nheader: bearer token"))
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
		permit, ok := r.Acquire(key)
		require.True(t, ok)
		permit.Failure(errors.New("once"))
	}
	permit, ok := r.Acquire("deepseek:auto_update_recap")
	require.True(t, ok)
	permit.Success()

	byKey := healthByKey(r.Snapshot())
	assert.Zero(t, byKey["deepseek:auto_update_recap"].FailureCount)
	assert.Equal(t, 1, byKey["deepseek:auto_extract_npcs"].FailureCount)
}

func TestBreakerAllowsOnlyOneConcurrentHalfOpenProbe(t *testing.T) {
	now := time.Unix(700, 0)
	r := NewBreakerRegistry(func() time.Time { return now }, 1, time.Minute)
	key := "deepseek:auto_update_recap"
	permit, ok := r.Acquire(key)
	require.True(t, ok)
	permit.Failure(errors.New("open"))
	now = now.Add(time.Minute)

	start := make(chan struct{})
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for range 64 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, ok := r.Acquire(key); ok {
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
	permit, ok := r.Acquire("unknown:auto_update_recap")
	require.True(t, ok)
	permit.Failure(context.DeadlineExceeded)
	_, allowed := r.Acquire("unknown:auto_update_recap")
	assert.True(t, allowed, "zero cooldown permits an immediate probe")
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

func TestBreakerPermitsRejectStaleCompletionsAcrossEpochs(t *testing.T) {
	now := time.Unix(800, 0)
	r := NewBreakerRegistry(func() time.Time { return now }, 2, time.Minute)
	key := "deepseek:auto_update_recap"

	a, ok := r.Acquire(key)
	require.True(t, ok)
	b, ok := r.Acquire(key)
	require.True(t, ok)
	staleSuccess, ok := r.Acquire(key)
	require.True(t, ok)
	staleFailure, ok := r.Acquire(key)
	require.True(t, ok)

	a.Failure(errors.New("first"))
	b.Failure(errors.New("opens"))
	now = now.Add(time.Minute)
	probe, ok := r.Acquire(key)
	require.True(t, ok)

	staleSuccess.Success()
	staleFailure.Failure(errors.New("late closed-era failure"))
	_, allowed := r.Acquire(key)
	assert.False(t, allowed, "stale completions cannot resolve the active probe")

	probe.Success()
	postRecovery, ok := r.Acquire(key)
	require.True(t, ok)
	postRecovery.Failure(errors.New("new era failure"))

	staleSuccess.Failure(errors.New("duplicate stale failure"))
	staleFailure.Success()
	health := healthByKey(r.Snapshot())[key]
	assert.Equal(t, BreakerClosed, health.Status)
	assert.Equal(t, 1, health.FailureCount, "stale completions cannot corrupt post-recovery state")
}

func TestBreakerPermitAbortDoesNotCountClosedCancellation(t *testing.T) {
	r := NewBreakerRegistry(time.Now, 1, time.Minute)
	key := "ollama:auto_extract_items"

	permit, ok := r.Acquire(key)
	require.True(t, ok)
	permit.Complete(context.Canceled)

	health := healthByKey(r.Snapshot())[key]
	assert.Equal(t, BreakerClosed, health.Status)
	assert.Zero(t, health.FailureCount)
	assert.Empty(t, health.LastError)
	_, allowed := r.Acquire(key)
	assert.True(t, allowed)
}

func TestBreakerPermitAbortReleasesHalfOpenProbe(t *testing.T) {
	now := time.Unix(900, 0)
	r := NewBreakerRegistry(func() time.Time { return now }, 1, time.Minute)
	key := "anthropic:auto_detect_objectives"

	initial, ok := r.Acquire(key)
	require.True(t, ok)
	initial.Failure(errors.New("open"))
	now = now.Add(time.Minute)
	probe, ok := r.Acquire(key)
	require.True(t, ok)
	probe.Abort()

	nextProbe, ok := r.Acquire(key)
	require.True(t, ok, "aborting a canceled probe must release the half-open slot")
	nextProbe.Success()
	assert.Equal(t, BreakerClosed, healthByKey(r.Snapshot())[key].Status)
}

func TestBreakerDeadlineStillCountsAsFailure(t *testing.T) {
	r := NewBreakerRegistry(time.Now, 1, time.Minute)
	permit, ok := r.Acquire("deepseek:auto_update_stats")
	require.True(t, ok)
	permit.Complete(context.DeadlineExceeded)

	health := r.Snapshot()[0]
	assert.Equal(t, BreakerOpen, health.Status)
	assert.Equal(t, "request timed out", health.LastError)
}

func TestBreakerRetryCancellationAbortsHalfOpenProbe(t *testing.T) {
	now := time.Unix(950, 0)
	r := NewBreakerRegistry(func() time.Time { return now }, 1, time.Minute)
	key := "anthropic:auto_update_recap"
	initial, ok := r.Acquire(key)
	require.True(t, ok)
	initial.Failure(errors.New("open"))
	now = now.Add(time.Minute)
	probe, ok := r.Acquire(key)
	require.True(t, ok)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := retryWithBackoff(ctx, 2, func(context.Context) error {
		return errors.New("provider failed before canceled backoff")
	})
	require.ErrorIs(t, err, context.Canceled)
	probe.Complete(err)

	_, allowed := r.Acquire(key)
	assert.True(t, allowed, "retry-layer cancellation must release the half-open probe")
}

func TestBreakerSnapshotJSONOmitsUnsetTimestampsAndDeepCopiesSetTimestamps(t *testing.T) {
	now := time.Unix(1000, 0).UTC()
	r := NewBreakerRegistry(func() time.Time { return now }, 2, time.Minute)
	key := "openrouter:auto_suggest_xp"
	permit, ok := r.Acquire(key)
	require.True(t, ok)

	encoded, err := json.Marshal(r.Snapshot())
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "opened_at")
	assert.NotContains(t, string(encoded), "last_success")

	permit.Success()
	first := r.Snapshot()
	require.NotNil(t, first[0].LastSuccess)
	*first[0].LastSuccess = time.Unix(1, 0)
	second := r.Snapshot()
	require.NotNil(t, second[0].LastSuccess)
	assert.Equal(t, now, *second[0].LastSuccess)
}

func TestAutoGenerateMapCountsSVGFailureOncePerInvocation(t *testing.T) {
	completer := &detectThenFailMapCompleter{}
	s := newTestServerWithAI(t, completer)
	_, sessionID := seedCampaign(t, s.db)

	for range 3 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		s.autoGenerateMap(ctx, sessionID, "The party enters the named Ashen Tower.")
		cancel()
	}

	key := "unknown:" + settingAutoGenerateMap
	health := healthByKey(s.breakers.Snapshot())[key]
	assert.Equal(t, BreakerOpen, health.Status)
	assert.Equal(t, 3, health.FailureCount)
	assert.EqualValues(t, 6, completer.calls.Load(), "each invocation uses one detection call and one failing SVG call")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	s.autoGenerateMap(ctx, sessionID, "The party enters the named Ashen Tower.")
	assert.EqualValues(t, 6, completer.calls.Load(), "the open breaker blocks the entire invocation")
}

func TestAutoGenerateMapResolvesLegitimateNoopAsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		response string
		seedMap  bool
	}{
		{"no location", `{"new_location":false}`, false},
		{"duplicate location", `{"new_location":true,"name":"Ashen Tower","context":"Already mapped"}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completer := &stubCompleter{response: tt.response}
			s := newTestServerWithAI(t, completer)
			campaignID, sessionID := seedCampaign(t, s.db)
			if tt.seedMap {
				_, err := s.db.CreateMap(campaignID, "Ashen Tower", "maps/existing.svg")
				require.NoError(t, err)
			}

			now := time.Unix(1100, 0)
			s.breakers = NewBreakerRegistry(func() time.Time { return now }, 1, time.Minute)
			key := "unknown:" + settingAutoGenerateMap
			initial, ok := s.breakers.Acquire(key)
			require.True(t, ok)
			initial.Failure(errors.New("open"))
			now = now.Add(time.Minute)

			s.autoGenerateMap(context.Background(), sessionID, "The party approaches Ashen Tower.")
			health := healthByKey(s.breakers.Snapshot())[key]
			assert.Equal(t, BreakerClosed, health.Status)
			assert.Zero(t, health.FailureCount)
			require.NotNil(t, health.LastSuccess)
			assert.Equal(t, now, *health.LastSuccess)
			assert.Equal(t, 1, completer.promptCount())
		})
	}
}

func TestBreakerHighCountStaleCompletionsCannotResolveProbe(t *testing.T) {
	now := time.Unix(1200, 0)
	r := NewBreakerRegistry(func() time.Time { return now }, 1, time.Minute)
	key := "deepseek:auto_update_recap"
	const staleCount = 1024

	stale := make([]*BreakerPermit, staleCount)
	for i := range stale {
		permit, ok := r.Acquire(key)
		require.True(t, ok)
		stale[i] = permit
	}
	stale[0].Failure(errors.New("open"))
	now = now.Add(time.Minute)
	probe, ok := r.Acquire(key)
	require.True(t, ok)

	var wg sync.WaitGroup
	for i, permit := range stale[1:] {
		wg.Add(1)
		go func(index int, permit *BreakerPermit) {
			defer wg.Done()
			if index%2 == 0 {
				permit.Success()
				return
			}
			permit.Failure(errors.New("stale"))
		}(i, permit)
	}
	wg.Wait()
	_, allowed := r.Acquire(key)
	assert.False(t, allowed)
	probe.Success()

	postRecovery, ok := r.Acquire(key)
	require.True(t, ok)
	postRecovery.Failure(errors.New("current"))
	health := healthByKey(r.Snapshot())[key]
	assert.Equal(t, BreakerOpen, health.Status)
	assert.Equal(t, 1, health.FailureCount)
}

type detectThenFailMapCompleter struct {
	calls atomic.Int32
}

func (c *detectThenFailMapCompleter) Generate(ctx context.Context, prompt string, _ int) (string, error) {
	c.calls.Add(1)
	if strings.Contains(prompt, "map assistant") {
		return `{"new_location":true,"name":"Ashen Tower","context":"A named ruined tower"}`, nil
	}
	<-ctx.Done()
	return "", ctx.Err()
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
