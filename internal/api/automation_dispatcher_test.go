package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func eventAutomationJob(key string, sessionID int64, kind string, run func(context.Context) error) AutomationJob {
	return AutomationJob{Key: key, SessionID: sessionID, Kind: kind, Mode: JobModeEvent, Run: run}
}

func snapshotAutomationJob(sessionID int64, kind string, run func(context.Context) error) AutomationJob {
	return AutomationJob{SessionID: sessionID, Kind: kind, Mode: JobModeSnapshot, Run: run}
}

func shutdownDispatcher(t *testing.T, d *Dispatcher) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, d.Shutdown(ctx))
}

func TestDispatcherLimitsConcurrencyAndRunsDifferentSessionsConcurrently(t *testing.T) {
	d := NewDispatcher(context.Background(), DispatcherOptions{Workers: 2, QueueSize: 8})
	t.Cleanup(func() { shutdownDispatcher(t, d) })

	release := make(chan struct{})
	started := make(chan int64, 3)
	var running atomic.Int32
	var maximum atomic.Int32
	for sessionID := int64(1); sessionID <= 3; sessionID++ {
		id := sessionID
		require.NoError(t, d.Submit(context.Background(), eventAutomationJob("event", id, "limit", func(context.Context) error {
			current := running.Add(1)
			defer running.Add(-1)
			for {
				old := maximum.Load()
				if current <= old || maximum.CompareAndSwap(old, current) {
					break
				}
			}
			started <- id
			<-release
			return nil
		})))
	}

	first := receiveWithin(t, started, time.Second, "first dispatcher worker")
	second := receiveWithin(t, started, time.Second, "second dispatcher worker")
	assert.NotEqual(t, first, second)
	select {
	case third := <-started:
		t.Fatalf("third job started above worker limit: session %d", third)
	case <-time.After(30 * time.Millisecond):
	}
	assert.Equal(t, int32(2), maximum.Load())
	close(release)
	receiveWithin(t, started, time.Second, "third dispatcher job")
}

func TestDispatcherRunsSameSessionInAcceptedOrder(t *testing.T) {
	d := NewDispatcher(context.Background(), DispatcherOptions{Workers: 3, QueueSize: 8})
	t.Cleanup(func() { shutdownDispatcher(t, d) })

	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	otherStarted := make(chan struct{})
	done := make(chan struct{}, 3)
	var mu sync.Mutex
	order := make([]int, 0, 2)
	require.NoError(t, d.Submit(context.Background(), eventAutomationJob("same-1", 7, "ordered", func(context.Context) error {
		close(firstStarted)
		<-releaseFirst
		mu.Lock()
		order = append(order, 1)
		mu.Unlock()
		done <- struct{}{}
		return nil
	})))
	receiveWithin(t, firstStarted, time.Second, "first same-session job")
	require.NoError(t, d.Submit(context.Background(), eventAutomationJob("same-2", 7, "ordered", func(context.Context) error {
		mu.Lock()
		order = append(order, 2)
		mu.Unlock()
		done <- struct{}{}
		return nil
	})))
	require.NoError(t, d.Submit(context.Background(), eventAutomationJob("other", 8, "ordered", func(context.Context) error {
		close(otherStarted)
		done <- struct{}{}
		return nil
	})))
	receiveWithin(t, otherStarted, time.Second, "different-session job")
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("different session did not run concurrently")
	}
	mu.Lock()
	assert.Empty(t, order, "second same-session job ran while first was blocked")
	mu.Unlock()
	close(releaseFirst)
	receiveWithin(t, done, time.Second, "first same-session completion")
	receiveWithin(t, done, time.Second, "second same-session completion")
	mu.Lock()
	assert.Equal(t, []int{1, 2}, order)
	mu.Unlock()
}

func TestDispatcherCoalescesOnlyPendingSnapshots(t *testing.T) {
	d := NewDispatcher(context.Background(), DispatcherOptions{Workers: 1, QueueSize: 4})
	t.Cleanup(func() { shutdownDispatcher(t, d) })

	runningStarted := make(chan struct{})
	releaseRunning := make(chan struct{})
	require.NoError(t, d.Submit(context.Background(), snapshotAutomationJob(4, settingAutoUpdateRecap, func(context.Context) error {
		close(runningStarted)
		<-releaseRunning
		return nil
	})))
	receiveWithin(t, runningStarted, time.Second, "running snapshot")

	var oldPending atomic.Int32
	var replacement atomic.Int32
	var secondSnapshot atomic.Int32
	require.NoError(t, d.Submit(context.Background(), snapshotAutomationJob(5, settingAutoUpdateRecap, func(context.Context) error {
		oldPending.Add(1)
		return nil
	})))
	require.NoError(t, d.Submit(context.Background(), snapshotAutomationJob(5, settingAutoUpdateRecap, func(context.Context) error {
		replacement.Add(1)
		return nil
	})))
	// The already-running identity cannot be replaced; this queues a second run.
	require.NoError(t, d.Submit(context.Background(), snapshotAutomationJob(4, settingAutoUpdateRecap, func(context.Context) error {
		secondSnapshot.Add(1)
		return nil
	})))
	close(releaseRunning)
	require.Eventually(t, func() bool {
		return replacement.Load() == 1 && secondSnapshot.Load() == 1
	}, time.Second, time.Millisecond)
	assert.Zero(t, oldPending.Load())
}

func TestDispatcherNeverCoalescesEventJobs(t *testing.T) {
	d := NewDispatcher(context.Background(), DispatcherOptions{Workers: 1, QueueSize: 8})
	t.Cleanup(func() { shutdownDispatcher(t, d) })
	var ran atomic.Int32
	block := make(chan struct{})
	started := make(chan struct{})
	require.NoError(t, d.Submit(context.Background(), eventAutomationJob("currency-1", 1, settingAutoUpdateCurrency, func(context.Context) error {
		close(started)
		<-block
		ran.Add(1)
		return nil
	})))
	receiveWithin(t, started, time.Second, "blocking event")
	for i := 0; i < 3; i++ {
		require.NoError(t, d.Submit(context.Background(), eventAutomationJob("currency", 1, settingAutoUpdateCurrency, func(context.Context) error {
			ran.Add(1)
			return nil
		})))
	}
	close(block)
	require.Eventually(t, func() bool { return ran.Load() == 4 }, time.Second, time.Millisecond)
}

func TestDispatcherSnapshotSaturationIsImmediateAndObservable(t *testing.T) {
	d := NewDispatcher(context.Background(), DispatcherOptions{Workers: 1, QueueSize: 1})
	t.Cleanup(func() { shutdownDispatcher(t, d) })
	started := make(chan struct{})
	release := make(chan struct{})
	require.NoError(t, d.Submit(context.Background(), eventAutomationJob("running", 1, "running", func(context.Context) error {
		close(started)
		<-release
		return nil
	})))
	receiveWithin(t, started, time.Second, "running job")
	require.NoError(t, d.Submit(context.Background(), snapshotAutomationJob(2, settingAutoUpdateRecap, func(context.Context) error { return nil })))
	begin := time.Now()
	err := d.Submit(context.Background(), snapshotAutomationJob(3, settingAutoUpdateRecap, func(context.Context) error { return nil }))
	assert.ErrorIs(t, err, ErrAutomationQueueFull)
	assert.Less(t, time.Since(begin), 100*time.Millisecond)
	health := dispatcherHealthByKind(d.Snapshot())[settingAutoUpdateRecap]
	assert.Equal(t, 1, health.Queued)
	assert.Equal(t, "automation queue full", health.LastError)
	close(release)
}

func TestDispatcherEventSubmissionBackpressureCancellationAndShutdown(t *testing.T) {
	d := NewDispatcher(context.Background(), DispatcherOptions{Workers: 1, QueueSize: 1})
	started := make(chan struct{})
	release := make(chan struct{})
	require.NoError(t, d.Submit(context.Background(), eventAutomationJob("running", 1, "event", func(context.Context) error {
		close(started)
		<-release
		return nil
	})))
	receiveWithin(t, started, time.Second, "running event")
	require.NoError(t, d.Submit(context.Background(), eventAutomationJob("queued", 2, "event", func(context.Context) error { return nil })))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err := d.Submit(ctx, eventAutomationJob("blocked", 3, "event", func(context.Context) error { return nil }))
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	shutdownDone := make(chan error, 1)
	go func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
		defer shutdownCancel()
		shutdownDone <- d.Shutdown(shutdownCtx)
	}()
	require.Eventually(t, func() bool {
		return errors.Is(d.Submit(context.Background(), eventAutomationJob("late", 4, "event", func(context.Context) error { return nil })), ErrAutomationDispatcherClosed)
	}, time.Second, time.Millisecond)
	close(release)
	require.NoError(t, receiveWithin(t, shutdownDone, time.Second, "dispatcher shutdown"))
}

func TestDispatcherShutdownDrainsAndDeadlineForceCancels(t *testing.T) {
	t.Run("drain", func(t *testing.T) {
		d := NewDispatcher(context.Background(), DispatcherOptions{Workers: 1, QueueSize: 4})
		release := make(chan struct{})
		started := make(chan struct{})
		var ran atomic.Int32
		require.NoError(t, d.Submit(context.Background(), eventAutomationJob("first", 1, "drain", func(context.Context) error {
			close(started)
			<-release
			ran.Add(1)
			return nil
		})))
		receiveWithin(t, started, time.Second, "draining job")
		require.NoError(t, d.Submit(context.Background(), eventAutomationJob("second", 2, "drain", func(context.Context) error {
			ran.Add(1)
			return nil
		})))
		done := make(chan error, 1)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			done <- d.Shutdown(ctx)
		}()
		select {
		case err := <-done:
			t.Fatalf("shutdown returned without draining: %v", err)
		case <-time.After(30 * time.Millisecond):
		}
		close(release)
		require.NoError(t, receiveWithin(t, done, time.Second, "drained shutdown"))
		assert.Equal(t, int32(2), ran.Load())
	})

	t.Run("force cancellation", func(t *testing.T) {
		d := NewDispatcher(context.Background(), DispatcherOptions{Workers: 1, QueueSize: 4})
		started := make(chan struct{})
		canceled := make(chan struct{})
		require.NoError(t, d.Submit(context.Background(), eventAutomationJob("running", 1, "force", func(ctx context.Context) error {
			close(started)
			<-ctx.Done()
			close(canceled)
			return ctx.Err()
		})))
		receiveWithin(t, started, time.Second, "force-canceled job")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		defer cancel()
		assert.ErrorIs(t, d.Shutdown(ctx), context.DeadlineExceeded)
		receiveWithin(t, canceled, time.Second, "job cancellation")
		retryCtx, retryCancel := context.WithTimeout(context.Background(), time.Second)
		defer retryCancel()
		require.NoError(t, d.Shutdown(retryCtx))
	})
}

func TestDispatcherParentCancellationAndConcurrentShutdownAreSafe(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	d := NewDispatcher(parent, DispatcherOptions{Workers: 2, QueueSize: 8})
	started := make(chan struct{})
	canceled := make(chan struct{})
	require.NoError(t, d.Submit(context.Background(), eventAutomationJob("root", 1, "root", func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		close(canceled)
		return ctx.Err()
	})))
	receiveWithin(t, started, time.Second, "root-owned job")
	cancelParent()
	receiveWithin(t, canceled, time.Second, "root cancellation")

	results := make(chan error, 8)
	var callers sync.WaitGroup
	for i := 0; i < 8; i++ {
		callers.Add(1)
		go func() {
			defer callers.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			results <- d.Shutdown(ctx)
		}()
	}
	callers.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	assert.ErrorIs(t, d.Submit(context.Background(), eventAutomationJob("late", 2, "root", func(context.Context) error { return nil })), ErrAutomationDispatcherClosed)
}

func TestDispatcherSubmitCoalesceShutdownStress(t *testing.T) {
	for iteration := 0; iteration < 20; iteration++ {
		d := NewDispatcher(context.Background(), DispatcherOptions{Workers: 4, QueueSize: 8})
		start := make(chan struct{})
		errorsSeen := make(chan error, 48)
		var acceptedEvents atomic.Int32
		var ranEvents atomic.Int32
		var submitters sync.WaitGroup
		for i := 0; i < 32; i++ {
			submitterID := i
			submitters.Add(1)
			go func() {
				defer submitters.Done()
				<-start
				err := d.Submit(context.Background(), eventAutomationJob("stress", int64(submitterID%5+1), "stress-event", func(context.Context) error {
					ranEvents.Add(1)
					return nil
				}))
				if err == nil {
					acceptedEvents.Add(1)
				} else if !errors.Is(err, ErrAutomationDispatcherClosed) {
					errorsSeen <- err
				}
			}()
		}
		for i := 0; i < 16; i++ {
			submitters.Add(1)
			go func() {
				defer submitters.Done()
				<-start
				err := d.Submit(context.Background(), snapshotAutomationJob(99, settingAutoUpdateRecap, func(context.Context) error { return nil }))
				if err != nil && !errors.Is(err, ErrAutomationQueueFull) && !errors.Is(err, ErrAutomationDispatcherClosed) {
					errorsSeen <- err
				}
			}()
		}
		close(start)
		shutdownDone := make(chan error, 1)
		go func() { shutdownDone <- d.Shutdown(context.Background()) }()
		submitters.Wait()
		require.NoError(t, receiveWithin(t, shutdownDone, time.Second, "stress shutdown"))
		close(errorsSeen)
		for err := range errorsSeen {
			require.NoError(t, err)
		}
		assert.Equal(t, acceptedEvents.Load(), ranEvents.Load(), "every accepted stress event must run")
	}
}

func TestDispatcherRecoversJobPanicAndContinuesSameSession(t *testing.T) {
	d := NewDispatcher(context.Background(), DispatcherOptions{Workers: 1, QueueSize: 4})
	continued := make(chan struct{})
	require.NoError(t, d.Submit(context.Background(), eventAutomationJob("panic", 77, "panic-kind", func(context.Context) error {
		panic("SECRET panic payload")
	})))
	require.NoError(t, d.Submit(context.Background(), eventAutomationJob("after-panic", 77, "continuation-kind", func(context.Context) error {
		close(continued)
		return nil
	})))
	receiveWithin(t, continued, time.Second, "same-session job after panic")
	shutdownDispatcher(t, d)
	health := dispatcherHealthByKind(d.Snapshot())["panic-kind"]
	assert.Zero(t, health.Running)
	assert.Equal(t, "automation job panicked", health.LastError)
	assert.NotContains(t, health.LastError, "SECRET")
}

func TestDispatcherRejectsPreCanceledCallerBeforeOwnership(t *testing.T) {
	d := NewDispatcher(context.Background(), DispatcherOptions{Workers: 1, QueueSize: 4})
	t.Cleanup(func() { shutdownDispatcher(t, d) })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var ran atomic.Int32
	for _, job := range []AutomationJob{
		eventAutomationJob("canceled-event", 1, "canceled-event", func(context.Context) error { ran.Add(1); return nil }),
		snapshotAutomationJob(2, "canceled-snapshot", func(context.Context) error { ran.Add(1); return nil }),
	} {
		assert.ErrorIs(t, d.Submit(ctx, job), context.Canceled)
	}
	time.Sleep(20 * time.Millisecond)
	assert.Zero(t, ran.Load(), "pre-canceled jobs must never transfer ownership")
}

func TestAutomationManualEndpointsRejectPreCanceledCaller(t *testing.T) {
	t.Run("session reanalysis", func(t *testing.T) {
		s := newTestServerWithAI(t, &stubCompleter{response: `[]`})
		_, sessionID := seedCampaign(t, s.db)
		_, err := s.db.CreateMessage(sessionID, "assistant", "The relic remains in Ashen Tower.", false, nil)
		require.NoError(t, err)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+strconv.FormatInt(sessionID, 10)+"/reanalyze", nil).WithContext(ctx)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
		assert.Equal(t, http.StatusRequestTimeout, w.Code)
		assert.NotEqual(t, http.StatusAccepted, w.Code)
	})

	t.Run("manual xp suggestion", func(t *testing.T) {
		s := newTestServer(t)
		ruleset, err := s.db.GetRulesetByName("vtm")
		require.NoError(t, err)
		campaignID, err := s.db.CreateCampaign(ruleset.ID, "Canceled Caller", "")
		require.NoError(t, err)
		characterID, err := s.db.CreateCharacter(campaignID, "Avery")
		require.NoError(t, err)
		require.NoError(t, s.db.UpdateCharacterData(characterID, `{"xp":20}`))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		req := httptest.NewRequest(http.MethodPost, "/api/characters/"+strconv.FormatInt(characterID, 10)+"/suggest-advances", strings.NewReader(`{}`)).WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
		assert.Equal(t, http.StatusRequestTimeout, w.Code)
		assert.NotEqual(t, http.StatusAccepted, w.Code)
	})
}

func TestAutomationSettingsRetainSanitizedDispatcherAndBreakerErrors(t *testing.T) {
	s := newTestServer(t)
	shutdownDispatcher(t, s.automations)
	s.automations = NewDispatcher(context.Background(), DispatcherOptions{Workers: 1, QueueSize: 1})
	release := make(chan struct{})
	var releaseOnce sync.Once
	started := make(chan struct{})
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	require.NoError(t, s.automations.Submit(context.Background(), eventAutomationJob("block", 1, "block", func(context.Context) error {
		close(started)
		<-release
		return nil
	})))
	receiveWithin(t, started, time.Second, "health blocker")
	require.NoError(t, s.automations.Submit(context.Background(), snapshotAutomationJob(2, settingAutoUpdateRecap, func(context.Context) error { return nil })))
	require.ErrorIs(t, s.automations.Submit(context.Background(), snapshotAutomationJob(3, settingAutoUpdateRecap, func(context.Context) error { return nil })), ErrAutomationQueueFull)
	permit, ok := s.breakers.Acquire(s.automationBreakerKey(settingAutoUpdateRecap))
	require.True(t, ok)
	permit.Failure(errors.New("SECRET https://provider.invalid/raw-prompt"))

	readRecap := func() map[string]any {
		req := httptest.NewRequest(http.MethodGet, "/api/settings/automations", nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		var settings []map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &settings))
		return settingByKey(settings)[settingAutoUpdateRecap]
	}
	lastError, _ := readRecap()["last_error"].(string)
	assert.Contains(t, lastError, "automation provider request failed")
	assert.Contains(t, lastError, ErrAutomationQueueFull.Error())
	assert.NotContains(t, lastError, "SECRET")
	assert.NotContains(t, lastError, "provider.invalid")

	releaseOnce.Do(func() { close(release) })
	require.Eventually(t, func() bool {
		health := dispatcherHealthByKind(s.automations.Snapshot())[settingAutoUpdateRecap]
		return health.Queued == 0 && health.Running == 0
	}, time.Second, time.Millisecond)
	lastError, _ = readRecap()["last_error"].(string)
	assert.Contains(t, lastError, "automation provider request failed")
	assert.Contains(t, lastError, ErrAutomationQueueFull.Error(), "job completion must not erase admission failure health")
}

type blockingReanalysisCompleter struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (c *blockingReanalysisCompleter) Generate(context.Context, string, int) (string, error) {
	c.once.Do(func() { close(c.started) })
	<-c.release
	return `[]`, nil
}

func TestAutomationReanalysisHealthAttributesCompositeEvent(t *testing.T) {
	completer := &blockingReanalysisCompleter{started: make(chan struct{}), release: make(chan struct{})}
	s := newTestServerWithAI(t, completer)
	_, sessionID := seedCampaign(t, s.db)
	_, err := s.db.CreateMessage(sessionID, "assistant", "A new mission objective names Captain Voss.", false, nil)
	require.NoError(t, err)
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(completer.release) }) })

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+strconv.FormatInt(sessionID, 10)+"/reanalyze", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.Equal(t, http.StatusAccepted, w.Code)
	receiveWithin(t, completer.started, time.Second, "composite reanalysis")

	settingsReq := httptest.NewRequest(http.MethodGet, "/api/settings/automations", nil)
	settingsW := httptest.NewRecorder()
	s.ServeHTTP(settingsW, settingsReq)
	require.Equal(t, http.StatusOK, settingsW.Code)
	var settings []map[string]any
	require.NoError(t, json.Unmarshal(settingsW.Body.Bytes(), &settings))
	byKey := settingByKey(settings)
	assert.Equal(t, float64(1), byKey[settingAutoDetectObj]["running"])
	assert.Equal(t, float64(1), byKey[settingAutoExtractNPCs]["running"])

	releaseOnce.Do(func() { close(completer.release) })
	require.Eventually(t, func() bool {
		health := dispatcherHealthByKind(s.automations.Snapshot())
		return health[settingAutoDetectObj].Running == 0 && health[settingAutoExtractNPCs].Running == 0
	}, time.Second, time.Millisecond)
}

func TestServerShutdownDrainsDispatcherAndCloseForcesCancellation(t *testing.T) {
	t.Run("graceful drain", func(t *testing.T) {
		s := newTestServer(t)
		started := make(chan struct{})
		release := make(chan struct{})
		require.NoError(t, s.automations.Submit(context.Background(), eventAutomationJob("server-drain", 1, "server", func(context.Context) error {
			close(started)
			<-release
			return nil
		})))
		receiveWithin(t, started, time.Second, "server dispatcher job")
		done := make(chan error, 1)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			done <- s.Shutdown(ctx)
		}()
		select {
		case err := <-done:
			t.Fatalf("server shutdown returned before automation drain: %v", err)
		case <-time.After(30 * time.Millisecond):
		}
		close(release)
		require.NoError(t, receiveWithin(t, done, time.Second, "server dispatcher drain"))
	})

	t.Run("force close", func(t *testing.T) {
		s := newTestServer(t)
		started := make(chan struct{})
		canceled := make(chan struct{})
		require.NoError(t, s.automations.Submit(context.Background(), eventAutomationJob("server-close", 1, "server", func(ctx context.Context) error {
			close(started)
			<-ctx.Done()
			close(canceled)
			return ctx.Err()
		})))
		receiveWithin(t, started, time.Second, "server force-close job")
		require.NoError(t, s.Close())
		receiveWithin(t, canceled, time.Second, "server close cancellation")
	})
}

func TestAutomationSettingsExposeCompatibleCombinedSanitizedHealth(t *testing.T) {
	s := newTestServer(t)
	permit, ok := s.breakers.Acquire(s.automationBreakerKey(settingAutoUpdateRecap))
	require.True(t, ok)
	permit.Failure(errors.New("https://provider.invalid?token=SECRET prompt body"))
	permit, ok = s.breakers.Acquire(s.automationBreakerKey(settingAutoUpdateRecap))
	require.True(t, ok)
	permit.Failure(errors.New("SECRET"))
	permit, ok = s.breakers.Acquire(s.automationBreakerKey(settingAutoUpdateRecap))
	require.True(t, ok)
	permit.Failure(errors.New("SECRET"))

	req := httptest.NewRequest(http.MethodGet, "/api/settings/automations", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var settings []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &settings))
	require.Len(t, settings, len(AllAutomationSettings()))
	for _, setting := range settings {
		assert.NotEmpty(t, setting["key"])
		assert.NotEmpty(t, setting["label"])
		_, hasEnabled := setting["enabled"]
		assert.True(t, hasEnabled)
		assert.Contains(t, setting, "status")
		assert.Contains(t, setting, "queued")
		assert.Contains(t, setting, "running")
		assert.Contains(t, setting, "last_success")
		assert.Contains(t, setting, "last_error")
		assert.Contains(t, setting, "failure_count")
		assert.Contains(t, setting, "cooling_down")
	}
	recap := settingByKey(settings)[settingAutoUpdateRecap]
	assert.Equal(t, BreakerOpen, recap["status"])
	assert.Equal(t, float64(3), recap["failure_count"])
	assert.Equal(t, true, recap["cooling_down"])
	assert.NotContains(t, recap["last_error"], "SECRET")
	assert.NotContains(t, recap["last_error"], "provider.invalid")
}

func TestAutomationSettingsReflectDispatcherRunningAndLastSuccess(t *testing.T) {
	s := newTestServer(t)
	started := make(chan struct{})
	release := make(chan struct{})
	require.NoError(t, s.automations.Submit(context.Background(), snapshotAutomationJob(12, settingAutoUpdateRecap, func(context.Context) error {
		close(started)
		<-release
		return nil
	})))
	receiveWithin(t, started, time.Second, "running settings automation")

	readSettings := func() map[string]map[string]any {
		req := httptest.NewRequest(http.MethodGet, "/api/settings/automations", nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		var settings []map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &settings))
		return settingByKey(settings)
	}
	running := readSettings()[settingAutoUpdateRecap]
	assert.Equal(t, float64(1), running["running"])
	assert.Equal(t, float64(0), running["queued"])

	close(release)
	require.Eventually(t, func() bool {
		completed := readSettings()[settingAutoUpdateRecap]
		return completed["running"] == float64(0) && completed["last_success"] != nil
	}, time.Second, time.Millisecond)
}

func TestAutomationManualEndpointsRejectClosedDispatcher(t *testing.T) {
	t.Run("session reanalysis", func(t *testing.T) {
		s := newTestServerWithAI(t, &stubCompleter{response: `[]`})
		_, sessionID := seedCampaign(t, s.db)
		_, err := s.db.CreateMessage(sessionID, "assistant", "The relic is hidden in Ashen Tower.", false, nil)
		require.NoError(t, err)
		shutdownDispatcher(t, s.automations)

		req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+strconv.FormatInt(sessionID, 10)+"/reanalyze", nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.NotEqual(t, http.StatusAccepted, w.Code)
		health := dispatcherHealthByKind(s.automations.Snapshot())[settingAutoDetectObj]
		assert.Equal(t, ErrAutomationDispatcherClosed.Error(), health.LastError)
	})

	t.Run("manual xp suggestion", func(t *testing.T) {
		s := newTestServer(t)
		ruleset, err := s.db.GetRulesetByName("vtm")
		require.NoError(t, err)
		campaignID, err := s.db.CreateCampaign(ruleset.ID, "Closed Dispatcher", "")
		require.NoError(t, err)
		characterID, err := s.db.CreateCharacter(campaignID, "Avery")
		require.NoError(t, err)
		require.NoError(t, s.db.UpdateCharacterData(characterID, `{"xp":20}`))
		shutdownDispatcher(t, s.automations)

		req := httptest.NewRequest(http.MethodPost, "/api/characters/"+strconv.FormatInt(characterID, 10)+"/suggest-advances", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.NotEqual(t, http.StatusAccepted, w.Code)
	})
}

func saturateServerAutomationDispatcher(t *testing.T, s *Server) {
	t.Helper()
	shutdownDispatcher(t, s.automations)
	s.automations = NewDispatcher(context.Background(), DispatcherOptions{Workers: 1, QueueSize: 1})
	started := make(chan struct{})
	release := make(chan struct{})
	require.NoError(t, s.automations.Submit(context.Background(), eventAutomationJob("running", 101, "saturation", func(context.Context) error {
		close(started)
		<-release
		return nil
	})))
	receiveWithin(t, started, time.Second, "saturation worker")
	require.NoError(t, s.automations.Submit(context.Background(), eventAutomationJob("queued", 102, "saturation", func(context.Context) error { return nil })))
	t.Cleanup(func() { close(release) })
}

func TestAutomationManualEndpointsDoNotAcceptCanceledBackpressure(t *testing.T) {
	t.Run("session reanalysis", func(t *testing.T) {
		s := newTestServerWithAI(t, &stubCompleter{response: `[]`})
		_, sessionID := seedCampaign(t, s.db)
		_, err := s.db.CreateMessage(sessionID, "assistant", "The relic remains in Ashen Tower.", false, nil)
		require.NoError(t, err)
		saturateServerAutomationDispatcher(t, s)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		defer cancel()
		req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+strconv.FormatInt(sessionID, 10)+"/reanalyze", nil).WithContext(ctx)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
		assert.Equal(t, http.StatusRequestTimeout, w.Code)
		assert.NotEqual(t, http.StatusAccepted, w.Code)
	})

	t.Run("manual xp suggestion", func(t *testing.T) {
		s := newTestServer(t)
		ruleset, err := s.db.GetRulesetByName("vtm")
		require.NoError(t, err)
		campaignID, err := s.db.CreateCampaign(ruleset.ID, "Saturated Dispatcher", "")
		require.NoError(t, err)
		characterID, err := s.db.CreateCharacter(campaignID, "Avery")
		require.NoError(t, err)
		require.NoError(t, s.db.UpdateCharacterData(characterID, `{"xp":20}`))
		saturateServerAutomationDispatcher(t, s)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		defer cancel()
		req := httptest.NewRequest(http.MethodPost, "/api/characters/"+strconv.FormatInt(characterID, 10)+"/suggest-advances", strings.NewReader(`{}`)).WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
		assert.Equal(t, http.StatusRequestTimeout, w.Code)
		assert.NotEqual(t, http.StatusAccepted, w.Code)
	})
}

type drainingXPCompleter struct {
	started  chan struct{}
	release  chan struct{}
	canceled chan struct{}
}

func (c *drainingXPCompleter) Generate(ctx context.Context, _ string, _ int) (string, error) {
	close(c.started)
	select {
	case <-c.release:
		return `[]`, nil
	case <-ctx.Done():
		close(c.canceled)
		return "", ctx.Err()
	}
}

func TestAutomationManualXPUsesDispatcherDrainContext(t *testing.T) {
	completer := &drainingXPCompleter{
		started:  make(chan struct{}),
		release:  make(chan struct{}),
		canceled: make(chan struct{}),
	}
	s := newTestServerWithAI(t, completer)
	ruleset, err := s.db.GetRulesetByName("vtm")
	require.NoError(t, err)
	campaignID, err := s.db.CreateCampaign(ruleset.ID, "Drain XP", "")
	require.NoError(t, err)
	characterID, err := s.db.CreateCharacter(campaignID, "Avery")
	require.NoError(t, err)
	require.NoError(t, s.db.UpdateCharacterData(characterID, `{"clan":"Brujah","xp":20}`))

	req := httptest.NewRequest(http.MethodPost, "/api/characters/"+strconv.FormatInt(characterID, 10)+"/suggest-advances", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.Equal(t, http.StatusAccepted, w.Code)
	receiveWithin(t, completer.started, time.Second, "manual XP automation")

	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		done <- s.Shutdown(ctx)
	}()
	select {
	case <-completer.canceled:
		t.Fatal("graceful server shutdown canceled accepted XP work")
	case err := <-done:
		t.Fatalf("graceful server shutdown returned before XP work drained: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	close(completer.release)
	require.NoError(t, receiveWithin(t, done, time.Second, "manual XP drain"))
}

func TestAutomationLaunchSitesUseDispatcherAndOnlyRecapIsSnapshot(t *testing.T) {
	routeFiles := []string{"routes.go", "routes_messages.go", "routes_phase_c.go", "routes_advance.go"}
	var allRoutes strings.Builder
	for _, name := range routeFiles {
		contents, err := os.ReadFile(name)
		require.NoError(t, err)
		assert.NotContains(t, string(contents), "go s.auto", "%s still launches an unbounded automation goroutine", name)
		allRoutes.Write(contents)
	}
	assert.Equal(t, 1, strings.Count(allRoutes.String(), "JobModeSnapshot"), "only recap regeneration may be submitted as a snapshot")
	assert.Contains(t, allRoutes.String(), "settingAutoUpdateRecap")
}

func dispatcherHealthByKind(items []AutomationDispatchHealth) map[string]AutomationDispatchHealth {
	result := make(map[string]AutomationDispatchHealth, len(items))
	for _, item := range items {
		result[item.Kind] = item
	}
	return result
}

func settingByKey(items []map[string]any) map[string]map[string]any {
	result := make(map[string]map[string]any, len(items))
	for _, item := range items {
		key, _ := item["key"].(string)
		result[key] = item
	}
	return result
}
