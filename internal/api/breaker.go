package api

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/digitalghost404/inkandbone/internal/ai"
)

const (
	BreakerClosed   = "closed"
	BreakerOpen     = "open"
	BreakerHalfOpen = "half-open"
)

const (
	defaultAutomationFailureThreshold = 3
	defaultAutomationCooldown         = time.Minute
)

// AutomationHealth is an immutable value snapshot of one automation breaker.
type AutomationHealth struct {
	Key          string    `json:"key"`
	Status       string    `json:"status"`
	FailureCount int       `json:"failure_count"`
	CoolingDown  bool      `json:"cooling_down"`
	OpenedAt     time.Time `json:"opened_at,omitempty"`
	LastSuccess  time.Time `json:"last_success,omitempty"`
	LastError    string    `json:"last_error,omitempty"`
}

type breakerEntry struct {
	state                    string
	failureCount             int
	openedAt                 time.Time
	probeInFlight            bool
	resolvedProbeBeforeAllow bool
	lastSuccess              time.Time
	lastError                string
}

// BreakerRegistry owns independent closed/open/half-open breakers by provider
// and automation key.
type BreakerRegistry struct {
	mu        sync.Mutex
	now       func() time.Time
	threshold int
	cooldown  time.Duration
	entries   map[string]*breakerEntry
}

// NewBreakerRegistry constructs a registry. The clock is injectable so state
// transitions remain deterministic in tests.
func NewBreakerRegistry(now func() time.Time, threshold int, cooldown time.Duration) *BreakerRegistry {
	if now == nil {
		now = time.Now
	}
	if threshold < 1 {
		threshold = 1
	}
	if cooldown < 0 {
		cooldown = 0
	}
	return &BreakerRegistry{
		now:       now,
		threshold: threshold,
		cooldown:  cooldown,
		entries:   make(map[string]*breakerEntry),
	}
}

// Allow reports whether work may start for key. After cooldown only one
// caller receives permission to perform the half-open probe.
func (r *BreakerRegistry) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry := r.entry(key)
	r.refresh(entry)
	switch entry.state {
	case BreakerClosed:
		entry.resolvedProbeBeforeAllow = false
		return true
	case BreakerHalfOpen:
		if entry.probeInFlight {
			return false
		}
		entry.probeInFlight = true
		return true
	default:
		return false
	}
}

// Success records a successful allowed operation. Results arriving while a
// breaker is still open are stale and cannot close it early.
func (r *BreakerRegistry) Success(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry := r.entry(key)
	r.refresh(entry)
	now := r.now()
	switch entry.state {
	case BreakerClosed:
		entry.failureCount = 0
		entry.lastSuccess = now
		entry.lastError = ""
	case BreakerHalfOpen:
		if !entry.probeInFlight {
			return
		}
		entry.state = BreakerClosed
		entry.failureCount = 0
		entry.openedAt = time.Time{}
		entry.probeInFlight = false
		entry.resolvedProbeBeforeAllow = true
		entry.lastSuccess = now
		entry.lastError = ""
	}
}

// Failure records an allowed operation failure. A failed half-open probe
// starts a fresh cooldown; stale failures while open or after a resolved probe
// do not extend or re-open the breaker.
func (r *BreakerRegistry) Failure(key string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry := r.entry(key)
	r.refresh(entry)
	switch entry.state {
	case BreakerClosed:
		if entry.resolvedProbeBeforeAllow {
			return
		}
		entry.failureCount++
		entry.lastError = sanitizeAutomationError(err)
		if entry.failureCount >= r.threshold {
			entry.state = BreakerOpen
			entry.openedAt = r.now()
		}
	case BreakerHalfOpen:
		if !entry.probeInFlight {
			return
		}
		entry.state = BreakerOpen
		entry.failureCount = r.threshold
		entry.openedAt = r.now()
		entry.probeInFlight = false
		entry.lastError = sanitizeAutomationError(err)
	}
}

// Snapshot returns independently allocated, key-sorted values suitable for
// serialization. It also exposes cooldown-complete breakers as half-open/ready.
func (r *BreakerRegistry) Snapshot() []AutomationHealth {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]AutomationHealth, 0, len(r.entries))
	for key, entry := range r.entries {
		r.refresh(entry)
		result = append(result, AutomationHealth{
			Key:          key,
			Status:       entry.state,
			FailureCount: entry.failureCount,
			CoolingDown:  entry.state == BreakerOpen,
			OpenedAt:     entry.openedAt,
			LastSuccess:  entry.lastSuccess,
			LastError:    entry.lastError,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result
}

func (r *BreakerRegistry) entry(key string) *breakerEntry {
	entry := r.entries[key]
	if entry == nil {
		entry = &breakerEntry{state: BreakerClosed}
		r.entries[key] = entry
	}
	return entry
}

func (r *BreakerRegistry) refresh(entry *breakerEntry) {
	if entry.state == BreakerOpen && !r.now().Before(entry.openedAt.Add(r.cooldown)) {
		entry.state = BreakerHalfOpen
		entry.probeInFlight = false
	}
}

func sanitizeAutomationError(err error) string {
	switch {
	case err == nil:
		return "automation request failed"
	case errors.Is(err, context.DeadlineExceeded):
		return "request timed out"
	case errors.Is(err, context.Canceled):
		return "request canceled"
	default:
		// Provider errors can contain request URLs, response bodies, credentials,
		// or prompt fragments. Health responses expose only a safe category.
		return "automation provider request failed"
	}
}

func (s *Server) automationBreakerKey(settingKey string) string {
	provider := "unknown"
	if named, ok := s.aiClient.(ai.ProviderNamer); ok && named.ProviderName() != "" {
		provider = named.ProviderName()
	}
	return provider + ":" + settingKey
}

func (s *Server) canRunAutomation(settingKey string) bool {
	return s.breakers.Allow(s.automationBreakerKey(settingKey))
}

func (s *Server) recordAutoSuccess(settingKey string) {
	s.breakers.Success(s.automationBreakerKey(settingKey))
}

func (s *Server) recordAutoFailure(settingKey string, err error) {
	s.breakers.Failure(s.automationBreakerKey(settingKey), err)
}
