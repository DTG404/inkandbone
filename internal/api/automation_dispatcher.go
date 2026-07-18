package api

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

const (
	defaultAutomationWorkers   = 4
	defaultAutomationQueueSize = 64
)

var (
	ErrAutomationQueueFull        = errors.New("automation queue full")
	ErrAutomationDispatcherClosed = errors.New("automation dispatcher closed")
	errAutomationJobPanicked      = errors.New("automation job panicked")
)

type JobMode uint8

const (
	JobModeEvent JobMode = iota
	JobModeSnapshot
)

type AutomationJob struct {
	Key       string
	SessionID int64
	Kind      string
	Mode      JobMode
	Run       func(context.Context) error
	// HealthKinds attributes one composite job to each affected automation
	// setting. Kind remains the scheduling/snapshot identity.
	HealthKinds []string
}

type DispatcherOptions struct {
	Workers   int
	QueueSize int
}

type AutomationDispatchHealth struct {
	Kind        string     `json:"kind"`
	Queued      int        `json:"queued"`
	Running     int        `json:"running"`
	LastSuccess *time.Time `json:"last_success,omitempty"`
	LastError   string     `json:"last_error,omitempty"`
}

type dispatchHealth struct {
	queued              int
	running             int
	lastSuccess         time.Time
	lastRunError        string
	lastSubmissionError string
}

type queuedAutomation struct {
	job         AutomationJob
	identity    string
	healthKinds []string
}

// Dispatcher bounds automation execution while preserving accepted event work
// and serializing mutations that belong to the same session.
type Dispatcher struct {
	ctx    context.Context
	cancel context.CancelFunc

	mu             sync.Mutex
	notify         chan struct{}
	accepting      bool
	forced         bool
	queueSize      int
	queue          []*queuedAutomation
	runningSession map[int64]bool
	health         map[string]*dispatchHealth
	nextKey        uint64

	workers sync.WaitGroup
	done    chan struct{}
}

func NewDispatcher(parent context.Context, options DispatcherOptions) *Dispatcher {
	if parent == nil {
		parent = context.Background()
	}
	workers := options.Workers
	if workers <= 0 {
		workers = defaultAutomationWorkers
	}
	queueSize := options.QueueSize
	if queueSize <= 0 {
		queueSize = defaultAutomationQueueSize
	}
	ctx, cancel := context.WithCancel(parent)
	d := &Dispatcher{
		ctx:            ctx,
		cancel:         cancel,
		notify:         make(chan struct{}),
		accepting:      true,
		queueSize:      queueSize,
		runningSession: make(map[int64]bool),
		health:         make(map[string]*dispatchHealth),
		done:           make(chan struct{}),
	}
	d.workers.Add(workers)
	for i := 0; i < workers; i++ {
		go d.worker()
	}
	go func() {
		d.workers.Wait()
		close(d.done)
	}()
	go func() {
		select {
		case <-parent.Done():
			d.forceCancel()
		case <-d.done:
		}
	}()
	return d
}

// Submit transfers ownership of a job to the dispatcher. Event submission
// waits for capacity; snapshots fail immediately when a pending slot cannot be
// acquired, except that an equivalent pending snapshot may be replaced.
func (d *Dispatcher) Submit(ctx context.Context, job AutomationJob) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if job.Run == nil || job.Kind == "" {
		return errors.New("automation job requires kind and run function")
	}
	if job.Mode != JobModeEvent && job.Mode != JobModeSnapshot {
		return errors.New("invalid automation job mode")
	}
	healthKinds := automationHealthKinds(job)
	job.HealthKinds = append([]string(nil), healthKinds...)

	identity := ""
	if job.Mode == JobModeSnapshot {
		identity = fmt.Sprintf("%d:%s", job.SessionID, job.Kind)
		job.Key = identity
	}
	for {
		d.mu.Lock()
		if err := ctx.Err(); err != nil {
			d.setSubmissionErrorLocked(healthKinds, automationDispatchError(err))
			d.mu.Unlock()
			return err
		}
		if d.ctx.Err() != nil {
			d.setSubmissionErrorLocked(healthKinds, ErrAutomationDispatcherClosed.Error())
			d.mu.Unlock()
			return ErrAutomationDispatcherClosed
		}
		if !d.accepting {
			d.setSubmissionErrorLocked(healthKinds, ErrAutomationDispatcherClosed.Error())
			d.mu.Unlock()
			return ErrAutomationDispatcherClosed
		}
		if job.Mode == JobModeSnapshot {
			for _, pending := range d.queue {
				if pending.job.Mode == JobModeSnapshot && pending.identity == identity {
					d.adjustQueuedLocked(pending.healthKinds, -1)
					pending.job = job
					pending.healthKinds = append(pending.healthKinds[:0], healthKinds...)
					d.adjustQueuedLocked(pending.healthKinds, 1)
					d.setSubmissionErrorLocked(healthKinds, "")
					d.signalLocked()
					d.mu.Unlock()
					return nil
				}
			}
		}
		if len(d.queue) < d.queueSize {
			d.nextKey++
			if job.Mode == JobModeEvent {
				job.Key = fmt.Sprintf("%s#%d", job.Key, d.nextKey)
			}
			d.queue = append(d.queue, &queuedAutomation{job: job, identity: identity, healthKinds: healthKinds})
			d.adjustQueuedLocked(healthKinds, 1)
			d.setSubmissionErrorLocked(healthKinds, "")
			d.signalLocked()
			d.mu.Unlock()
			return nil
		}
		if job.Mode == JobModeSnapshot {
			d.setSubmissionErrorLocked(healthKinds, ErrAutomationQueueFull.Error())
			d.mu.Unlock()
			return ErrAutomationQueueFull
		}
		wait := d.notify
		d.mu.Unlock()

		select {
		case <-ctx.Done():
			d.recordSubmissionError(healthKinds, automationDispatchError(ctx.Err()))
			return ctx.Err()
		case <-d.ctx.Done():
			d.recordSubmissionError(healthKinds, ErrAutomationDispatcherClosed.Error())
			return ErrAutomationDispatcherClosed
		case <-wait:
		}
	}
}

// Shutdown rejects new work, drains accepted jobs while ctx remains live, and
// force-cancels dispatcher-owned work if the deadline expires.
func (d *Dispatcher) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	d.stopAccepting()
	select {
	case <-d.done:
		return nil
	case <-ctx.Done():
		d.forceCancel()
		return ctx.Err()
	}
}

// ForceCancel rejects new jobs and cancels all dispatcher-owned work. It is
// used by abrupt server Close and by root-context cancellation.
func (d *Dispatcher) ForceCancel() {
	d.forceCancel()
}

func (d *Dispatcher) stopAccepting() {
	d.mu.Lock()
	if d.accepting {
		d.accepting = false
		d.signalLocked()
	}
	d.mu.Unlock()
}

func (d *Dispatcher) forceCancel() {
	d.mu.Lock()
	if !d.forced {
		d.accepting = false
		d.forced = true
		for _, pending := range d.queue {
			d.adjustQueuedLocked(pending.healthKinds, -1)
			d.setRunErrorLocked(pending.healthKinds, "automation request canceled")
		}
		d.queue = nil
		d.cancel()
		d.signalLocked()
	}
	d.mu.Unlock()
}

// Snapshot returns independently allocated, kind-sorted dispatcher health.
func (d *Dispatcher) Snapshot() []AutomationDispatchHealth {
	d.mu.Lock()
	defer d.mu.Unlock()
	result := make([]AutomationDispatchHealth, 0, len(d.health))
	for kind, health := range d.health {
		result = append(result, AutomationDispatchHealth{
			Kind:        kind,
			Queued:      health.queued,
			Running:     health.running,
			LastSuccess: copyTime(health.lastSuccess),
			LastError:   joinAutomationErrors(health.lastRunError, health.lastSubmissionError),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Kind < result[j].Kind })
	return result
}

func (d *Dispatcher) worker() {
	defer d.workers.Done()
	for {
		d.mu.Lock()
		job, stop := d.takeLocked()
		if job == nil {
			if stop {
				d.mu.Unlock()
				return
			}
			wait := d.notify
			d.mu.Unlock()
			select {
			case <-wait:
			case <-d.ctx.Done():
			}
			continue
		}
		d.mu.Unlock()

		err := runAutomationJob(d.ctx, job.job.Run)

		d.mu.Lock()
		d.adjustRunningLocked(job.healthKinds, -1)
		if err == nil {
			completedAt := time.Now()
			for _, kind := range job.healthKinds {
				health := d.healthForLocked(kind)
				health.lastSuccess = completedAt
				health.lastRunError = ""
			}
		} else if errors.Is(err, errAutomationJobPanicked) {
			d.setRunErrorLocked(job.healthKinds, errAutomationJobPanicked.Error())
		} else {
			d.setRunErrorLocked(job.healthKinds, sanitizeAutomationError(err))
		}
		if job.job.SessionID != 0 {
			delete(d.runningSession, job.job.SessionID)
		}
		d.signalLocked()
		d.mu.Unlock()
	}
}

func (d *Dispatcher) takeLocked() (*queuedAutomation, bool) {
	if d.forced {
		return nil, true
	}
	for index, pending := range d.queue {
		if pending.job.SessionID != 0 && d.runningSession[pending.job.SessionID] {
			continue
		}
		d.queue = append(d.queue[:index], d.queue[index+1:]...)
		d.adjustQueuedLocked(pending.healthKinds, -1)
		d.adjustRunningLocked(pending.healthKinds, 1)
		if pending.job.SessionID != 0 {
			d.runningSession[pending.job.SessionID] = true
		}
		d.signalLocked()
		return pending, false
	}
	return nil, !d.accepting && len(d.queue) == 0
}

func (d *Dispatcher) healthForLocked(kind string) *dispatchHealth {
	health := d.health[kind]
	if health == nil {
		health = &dispatchHealth{}
		d.health[kind] = health
	}
	return health
}

func (d *Dispatcher) recordSubmissionError(kinds []string, message string) {
	d.mu.Lock()
	d.setSubmissionErrorLocked(kinds, message)
	d.mu.Unlock()
}

func automationHealthKinds(job AutomationJob) []string {
	result := make([]string, 0, 1+len(job.HealthKinds))
	seen := make(map[string]bool, 1+len(job.HealthKinds))
	for _, kind := range append([]string{job.Kind}, job.HealthKinds...) {
		if kind != "" && !seen[kind] {
			seen[kind] = true
			result = append(result, kind)
		}
	}
	return result
}

func (d *Dispatcher) adjustQueuedLocked(kinds []string, delta int) {
	for _, kind := range kinds {
		d.healthForLocked(kind).queued += delta
	}
}

func (d *Dispatcher) adjustRunningLocked(kinds []string, delta int) {
	for _, kind := range kinds {
		d.healthForLocked(kind).running += delta
	}
}

func (d *Dispatcher) setRunErrorLocked(kinds []string, message string) {
	for _, kind := range kinds {
		d.healthForLocked(kind).lastRunError = message
	}
}

func (d *Dispatcher) setSubmissionErrorLocked(kinds []string, message string) {
	for _, kind := range kinds {
		d.healthForLocked(kind).lastSubmissionError = message
	}
}

func runAutomationJob(ctx context.Context, run func(context.Context) error) (err error) {
	defer func() {
		if recover() != nil {
			err = errAutomationJobPanicked
		}
	}()
	return run(ctx)
}

func joinAutomationErrors(values ...string) string {
	result := ""
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		if result != "" {
			result += "; "
		}
		result += value
	}
	return result
}

func automationDispatchError(err error) string {
	switch {
	case errors.Is(err, ErrAutomationQueueFull):
		return ErrAutomationQueueFull.Error()
	case errors.Is(err, ErrAutomationDispatcherClosed):
		return ErrAutomationDispatcherClosed.Error()
	case errors.Is(err, context.DeadlineExceeded):
		return "automation submission timed out"
	case errors.Is(err, context.Canceled):
		return "automation submission canceled"
	default:
		return "automation submission failed"
	}
}

func (d *Dispatcher) signalLocked() {
	close(d.notify)
	d.notify = make(chan struct{})
}
