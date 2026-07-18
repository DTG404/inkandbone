package api

import (
	"context"
	"errors"
	"sort"
	"sync"
	"sync/atomic"
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
	Key          string     `json:"key"`
	Status       string     `json:"status"`
	FailureCount int        `json:"failure_count"`
	CoolingDown  bool       `json:"cooling_down"`
	OpenedAt     *time.Time `json:"opened_at,omitempty"`
	LastSuccess  *time.Time `json:"last_success,omitempty"`
	LastError    string     `json:"last_error,omitempty"`
}

type breakerEntry struct {
	state         string
	epoch         uint64
	failureCount  int
	openedAt      time.Time
	probeInFlight bool
	probePermitID uint64
	lastSuccess   time.Time
	lastError     string
}

// BreakerRegistry owns independent closed/open/half-open breakers by provider
// and automation key.
type BreakerRegistry struct {
	mu           sync.Mutex
	now          func() time.Time
	threshold    int
	cooldown     time.Duration
	nextPermitID uint64
	entries      map[string]*breakerEntry
}

// BreakerPermit identifies one admitted operation in one breaker epoch.
// A permit resolves at most once, and only against the operation that acquired it.
type BreakerPermit struct {
	registry *BreakerRegistry
	key      string
	epoch    uint64
	id       uint64
	probe    bool
	done     atomic.Bool
}

type permitResult uint8

const (
	permitSucceeded permitResult = iota
	permitFailed
	permitAborted
)

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

// Acquire admits an operation and returns its correlated permit. After
// cooldown, only one caller can acquire the half-open probe permit.
func (r *BreakerRegistry) Acquire(key string) (*BreakerPermit, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry := r.entry(key)
	r.refresh(entry)
	probe := false
	switch entry.state {
	case BreakerClosed:
	case BreakerHalfOpen:
		if entry.probeInFlight {
			return nil, false
		}
		probe = true
	default:
		return nil, false
	}

	r.nextPermitID++
	permit := &BreakerPermit{
		registry: r,
		key:      key,
		epoch:    entry.epoch,
		id:       r.nextPermitID,
		probe:    probe,
	}
	if probe {
		entry.probeInFlight = true
		entry.probePermitID = permit.id
	}
	return permit, true
}

// Complete maps nil to success, caller/root cancellation to abort, and all
// other errors (including deadline expiry) to breaker failure.
func (p *BreakerPermit) Complete(err error) {
	switch {
	case err == nil:
		p.Success()
	case errors.Is(err, context.Canceled):
		p.Abort()
	default:
		p.Failure(err)
	}
}

// Success resolves this permit successfully.
func (p *BreakerPermit) Success() {
	p.resolve(permitSucceeded, nil)
}

// Failure resolves this permit as a genuine operation failure.
func (p *BreakerPermit) Failure(err error) {
	p.resolve(permitFailed, err)
}

// Abort resolves this permit without changing failure health. Aborting a
// half-open probe immediately releases the probe slot for another caller.
func (p *BreakerPermit) Abort() {
	p.resolve(permitAborted, nil)
}

func (p *BreakerPermit) resolve(result permitResult, err error) {
	if p == nil || p.registry == nil || !p.done.CompareAndSwap(false, true) {
		return
	}
	p.registry.resolve(p, result, err)
}

func (r *BreakerRegistry) resolve(permit *BreakerPermit, result permitResult, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry := r.entry(permit.key)
	r.refresh(entry)
	if permit.epoch != entry.epoch {
		return
	}

	if permit.probe {
		if entry.state != BreakerHalfOpen || !entry.probeInFlight || entry.probePermitID != permit.id {
			return
		}
		switch result {
		case permitAborted:
			entry.probeInFlight = false
			entry.probePermitID = 0
		case permitSucceeded:
			entry.state = BreakerClosed
			entry.epoch++
			entry.failureCount = 0
			entry.openedAt = time.Time{}
			entry.probeInFlight = false
			entry.probePermitID = 0
			entry.lastSuccess = r.now()
			entry.lastError = ""
		case permitFailed:
			entry.state = BreakerOpen
			entry.epoch++
			entry.failureCount = r.threshold
			entry.openedAt = r.now()
			entry.probeInFlight = false
			entry.probePermitID = 0
			entry.lastError = sanitizeAutomationError(err)
		}
		return
	}

	if entry.state != BreakerClosed {
		return
	}
	switch result {
	case permitAborted:
		return
	case permitSucceeded:
		entry.failureCount = 0
		entry.lastSuccess = r.now()
		entry.lastError = ""
	case permitFailed:
		entry.failureCount++
		entry.lastError = sanitizeAutomationError(err)
		if entry.failureCount >= r.threshold {
			entry.state = BreakerOpen
			entry.epoch++
			entry.openedAt = r.now()
		}
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
			OpenedAt:     copyTime(entry.openedAt),
			LastSuccess:  copyTime(entry.lastSuccess),
			LastError:    entry.lastError,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result
}

func (r *BreakerRegistry) entry(key string) *breakerEntry {
	entry := r.entries[key]
	if entry == nil {
		entry = &breakerEntry{state: BreakerClosed, epoch: 1}
		r.entries[key] = entry
	}
	return entry
}

func (r *BreakerRegistry) refresh(entry *breakerEntry) {
	if entry.state == BreakerOpen && !r.now().Before(entry.openedAt.Add(r.cooldown)) {
		entry.state = BreakerHalfOpen
		entry.probeInFlight = false
		entry.probePermitID = 0
	}
}

func copyTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	copy := value
	return &copy
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

func (s *Server) acquireAutomation(settingKey string) (*BreakerPermit, bool) {
	return s.breakers.Acquire(s.automationBreakerKey(settingKey))
}
