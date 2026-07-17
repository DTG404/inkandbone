# Phase 2 Runtime Reliability and Protocol Correctness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bound background work, make automation failures recoverable and observable, give every network operation a lifecycle, and make SSE/WebSocket/SVG/prompt behavior exact and testable.

**Architecture:** Introduce a root application context, bounded automation dispatcher, independent timed circuit breakers, typed realtime frames, and strict prompt/SVG boundaries around existing domain functions. Prefer authoritative state reconciliation over durable event replay.

**Tech Stack:** Go 1.26.5+, `context`, `net/http`, `encoding/json`, `encoding/xml`, React 19, TypeScript, Vitest, WebSocket, SSE.

## Global Constraints

- Snapshot jobs may coalesce; event/delta jobs may not coalesce or drop.
- GM streams use inactivity and absolute deadlines, not one short total timeout.
- Circuit-breaker state is keyed by provider and automation.
- WebSocket gaps reload authoritative state; no durable event log is introduced.
- Campaign prompt guidance cannot replace privacy, transport, or integrity instructions.
- SVG sanitization fails closed.

---

### Task 1: Give the server and AI clients explicit lifecycles

**Files:**
- Modify: `internal/api/server.go`
- Create: `internal/api/lifecycle_test.go`
- Create: `internal/ai/http_client.go`
- Modify: `internal/ai/client.go`
- Modify: `internal/ai/deepseek.go`
- Modify: `internal/ai/openrouter.go`
- Modify: `internal/ai/ollama.go`
- Modify: `internal/ai/embed.go`
- Modify: `cmd/ttrpg/main.go`

**Interfaces:**
- Produces: `Server.Start(addr, certFile, keyFile string) error`, working `Server.Shutdown(context.Context) error`, `ai.NewHTTPClient()`, and request-context deadlines.

- [ ] **Step 1: Write failing lifecycle and timeout tests**

Use an `httptest.Server` that withholds response headers and another that streams periodic bytes. Assert provider header timeout, cancellation propagation, GM stream survival while bytes arrive, absolute stream timeout, and `Shutdown` waiting for an in-flight handler before returning.

- [ ] **Step 2: Run and verify RED**

Run: `go test ./internal/ai ./internal/api -run 'Timeout|Lifecycle|Shutdown' -v`

Expected: existing clients hang until the test context ends and `Shutdown` returns without controlling a server.

- [ ] **Step 3: Add a shared transport**

```go
func NewHTTPClient() *http.Client {
	return &http.Client{Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout: 90 * time.Second,
		MaxIdleConns: 32,
	}}
}
```

Do not set `http.Client.Timeout` for streaming clients. Use request contexts for automation and absolute GM limits. Replace every `&http.Client{}` and `http.DefaultClient` call.

- [ ] **Step 4: Own an `http.Server` and root context**

Store `rootCtx`, `cancel`, and `httpServer` on `Server`. Configure `ReadHeaderTimeout: 10s`, `IdleTimeout: 120s`, `MaxHeaderBytes: 1<<20`; use signal cancellation in `main`; call `Shutdown` with a 15-second context; close the dispatcher and hub in later tasks through the same lifecycle.

- [ ] **Step 5: Verify**

Run: `gofmt -w cmd internal/ai internal/api && go test ./internal/ai ./internal/api ./cmd/ttrpg`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add cmd/ttrpg/main.go internal/ai internal/api/server.go internal/api/lifecycle_test.go
git commit -m "fix: bound server and provider lifecycles"
```

### Task 2: Replace the global permanent circuit breaker

**Files:**
- Create: `internal/api/breaker.go`
- Create: `internal/api/breaker_test.go`
- Modify: `internal/api/server.go`
- Modify: `internal/api/automation.go`
- Create: `internal/ai/provider.go`
- Modify: provider implementations under `internal/ai/*.go`.

**Interfaces:**
- Produces: `ai.ProviderNamer{ProviderName() string}`, `BreakerRegistry.Allow(key string) bool`, `Success(key string)`, `Failure(key string, err error)`, `Snapshot() []AutomationHealth`.

- [ ] **Step 1: Write deterministic breaker-state tests with an injected clock**

```go
func TestBreakerRecoversThroughHalfOpenProbe(t *testing.T) {
	now := time.Unix(100, 0)
	r := NewBreakerRegistry(func() time.Time { return now }, 3, time.Minute)
	for range 3 { require.True(t, r.Allow("deepseek:recap")); r.Failure("deepseek:recap", errors.New("down")) }
	assert.False(t, r.Allow("deepseek:recap"))
	assert.True(t, r.Allow("deepseek:npcs"), "unrelated automation remains available")
	now = now.Add(time.Minute)
	assert.True(t, r.Allow("deepseek:recap"))
	assert.False(t, r.Allow("deepseek:recap"), "only one half-open probe")
	r.Success("deepseek:recap")
	assert.True(t, r.Allow("deepseek:recap"))
}
```

- [ ] **Step 2: Run and verify RED**

Run: `go test ./internal/api -run TestBreaker -v`

- [ ] **Step 3: Implement keyed closed/open/half-open state**

Protect state with a mutex. Store failure count, state, opened time, probe flag, last success, and sanitized last error. A success resets only its key. The registry returns copies for API serialization.

Each concrete AI client and dual/hybrid wrapper implements `ProviderName() string`. Use `unknown` only for test doubles that do not implement `ProviderNamer`; breaker keys remain `<provider-name>:<automation-setting-key>`.

- [ ] **Step 4: Replace `autoFailCount`, `canRunAutomation`, `recordAutoSuccess`, and `recordAutoFailure`**

Every guarded automation supplies a stable key of `<provider>:<automation-setting-key>`. Remove the global atomic counter from `Server`.

- [ ] **Step 5: Verify and commit**

Run: `go test ./internal/api -run 'Breaker|Automation' -v`

```bash
git add internal/api/breaker.go internal/api/breaker_test.go internal/api/server.go internal/api/automation.go
git commit -m "fix: make automation circuit breakers recoverable"
```

### Task 3: Dispatch automation through a bounded, typed queue

**Files:**
- Create: `internal/api/automation_dispatcher.go`
- Create: `internal/api/automation_dispatcher_test.go`
- Modify: `internal/api/server.go`
- Modify: `internal/api/routes.go`
- Modify: `internal/api/automation_config.go`

**Interfaces:**
- Produces: `AutomationJob{Key, SessionID, Kind, Mode, Run}`, `JobModeSnapshot`, `JobModeEvent`, `Dispatcher.Submit`, `Dispatcher.Shutdown`, health fields `queued` and `running`.

- [ ] **Step 1: Write failing queue-behavior tests**

Test maximum concurrency, same-session sequential execution, replacement of pending snapshot jobs, preservation of every event job, snapshot saturation reporting, event-job backpressure, cancellation deadline, drain-on-shutdown, and forced cancellation after drain timeout.

```go
func TestDispatcherNeverCoalescesEventJobs(t *testing.T) {
	d := NewDispatcher(context.Background(), DispatcherOptions{Workers: 1, QueueSize: 8})
	defer d.Shutdown(context.Background())
	var ran atomic.Int32
	block := make(chan struct{})
	require.NoError(t, d.Submit(eventJob("currency-1", func(context.Context) error { <-block; ran.Add(1); return nil })))
	require.NoError(t, d.Submit(eventJob("currency-2", func(context.Context) error { ran.Add(1); return nil })))
	close(block)
	require.Eventually(t, func() bool { return ran.Load() == 2 }, time.Second, time.Millisecond)
}
```

- [ ] **Step 2: Run and verify RED**

Run: `go test ./internal/api -run TestDispatcher -v`

- [ ] **Step 3: Implement dispatcher state and job classification**

Use four workers and a queue of 64 by default. Snapshot keys are `<session-id>:<automation-kind>` and may replace only a pending snapshot with the same key. Event jobs always receive unique keys and block submission until queue capacity exists or the server root context is cancelled; they never return a successful submission without durable ownership by a worker/queue. Snapshot saturation returns `ErrAutomationQueueFull` and is surfaced in health state.

- [ ] **Step 4: Replace post-GM goroutines with submitted jobs**

Start conservatively: only recap regeneration is a snapshot job because it recomputes from authoritative message history. NPC extraction, map generation, character changes, objective detection, scene tags, tension, Masquerade, chronicle-night changes, currency, inventory, XP, dice, rouse/stains, and every other job derived from one specific GM response are event jobs until a test proves a rewritten job recomputes equivalent state from authoritative history. Keep `checkAndExecuteRoll` synchronous before GM response.

- [ ] **Step 5: Add automation-health response fields**

Extend `GET /api/settings/automations` without breaking existing setting fields. Include `status`, `queued`, `running`, `last_success`, and `last_error` from dispatcher and breaker snapshots.

- [ ] **Step 6: Verify under race detector**

Run: `go test -race ./internal/api -run 'Dispatcher|Automation' -v`

Expected: PASS with no race reports.

- [ ] **Step 7: Commit**

```bash
git add internal/api/automation_dispatcher.go internal/api/automation_dispatcher_test.go internal/api/server.go internal/api/routes.go internal/api/automation_config.go
git commit -m "feat: bound and observe automation work"
```

### Task 4: Encode and parse typed SSE frames exactly

**Files:**
- Create: `internal/ai/sse.go`
- Create: `internal/ai/sse_test.go`
- Modify: stream implementations in `internal/ai/*.go`
- Create: `web/src/sse.ts`
- Create: `web/src/sse.test.ts`
- Modify: `web/src/api.ts`

**Interfaces:**
- Produces: Go `SSEEvent{Type, Delta, Code, RequestID}`, `WriteSSE`, and TypeScript `parseSSE(response, onEvent)`.

- [ ] **Step 1: Write failing Go framing tests**

```go
func TestWriteSSEPreservesNewlines(t *testing.T) {
	var b bytes.Buffer
	require.NoError(t, WriteSSE(&b, SSEEvent{Type: "delta", Delta: "first\nsecond ☃"}))
	assert.Equal(t, "data: {\"type\":\"delta\",\"delta\":\"first\\nsecond ☃\"}\n\n", b.String())
}
```

- [ ] **Step 2: Write failing browser parser tests**

Feed a `ReadableStream<Uint8Array>` with splits inside UTF-8, JSON, `data:`, and the blank-line delimiter. Assert exact output for embedded newlines, two frames in one chunk, completion, and error events.

- [ ] **Step 3: Run and verify RED**

Run: `go test ./internal/ai -run TestWriteSSE -v && cd web && npm test -- --run sse.test.ts`

- [ ] **Step 4: Implement one JSON object per SSE frame**

`WriteSSE` JSON-encodes the event, writes exactly one `data: ` line plus two LF bytes, checks write errors, and flushes when supported. Provider streamers call it for every delta and emit one terminal event.

- [ ] **Step 5: Implement buffered browser parsing**

Use one streaming `TextDecoder`, accumulate text until `\n\n`, join all `data:` lines in a frame, parse JSON, and dispatch by the discriminated `type` field. Flush the decoder and final buffer at EOF.

- [ ] **Step 6: Verify exact persisted versus streamed text**

Add an API test whose stub streams `"first\nsecond ☃"` in adversarial chunks and assert browser callback concatenation equals the stored assistant message.

- [ ] **Step 7: Commit**

```bash
git add internal/ai web/src/sse.ts web/src/sse.test.ts web/src/api.ts internal/api/routes_test.go
git commit -m "fix: preserve exact text across sse streaming"
```

### Task 5: Detect WebSocket gaps and reconcile authoritative state

**Files:**
- Modify: `internal/api/events.go`
- Modify: `internal/api/ws.go`
- Modify: `internal/api/ws_test.go`
- Modify: `web/src/useWebSocket.ts`
- Modify: `web/src/useWebSocket.test.tsx`
- Modify: `web/src/App.tsx`

**Interfaces:**
- Produces: event field `sequence`, hook result `{lastEvent, status, needsReconcile}`, protocol-derived URL, exponential reconnect.

- [ ] **Step 1: Write failing server sequence and overflow tests**

Publish consecutive events and assert strictly increasing sequence numbers. Fill a client channel and assert overflow is logged/marked for reconciliation instead of being indistinguishable from success.

- [ ] **Step 2: Write failing hook tests with deterministic timers/randomness**

Assert `wss://` under HTTPS, statuses `connecting/open/reconnecting/offline`, backoff capped at 30 seconds, jitter injection, reset after a successful open, and `needsReconcile` after sequence gap or reconnect.

- [ ] **Step 3: Run and verify RED**

Run: `go test ./internal/api -run 'Hub|Sequence|Overflow' -v && cd web && npm test -- --run useWebSocket.test.tsx`

- [ ] **Step 4: Implement server sequencing and explicit resync signal**

Assign sequence in `Bus.Publish` or one central hub boundary. When a per-client buffer is full, send/mark a `resync_required` event as soon as capacity returns and log the client identifier and lost sequence range.

- [ ] **Step 5: Implement client state machine**

Derive URL from `window.location.protocol`; use delays `min(1000 * 2^attempt, 30000)` multiplied by injected jitter `0.8..1.2`; set `needsReconcile` on reconnect or nonconsecutive sequence; have `App` call its authoritative `loadContext` once and acknowledge reconciliation. Remove the unconditional `loadContext()` at the start of every event callback. Known events update their owned local slice or let the relevant panel refetch; only reconnects, sequence gaps, and unknown state-invalidating events reload the full context. Add a fetch-count test proving a normal typing event does not reload `/api/context`.

- [ ] **Step 6: Verify and commit**

Run: `go test -race ./internal/api -run 'Hub|Bus' -v && cd web && npm test -- --run useWebSocket.test.tsx App.test.tsx`

```bash
git add internal/api/events.go internal/api/ws.go internal/api/ws_test.go web/src/useWebSocket.ts web/src/useWebSocket.test.tsx web/src/App.tsx
git commit -m "fix: reconcile websocket gaps and reconnects"
```

### Task 6: Make campaign guidance and content boundaries functional

**Files:**
- Create: `internal/api/prompt.go`
- Create: `internal/api/prompt_test.go`
- Create: `internal/db/migrations/057_campaign_narrative_preferences.sql`
- Modify: `internal/db/queries_core.go`
- Modify: `internal/api/routes.go`
- Modify: `internal/api/routes_campaign_config.go`
- Modify: `web/src/GMScreenPanel.tsx`

**Interfaces:**
- Produces: `BuildSystemPrompt(base, ruleset, campaignGuidance, contentBoundaries, narrativeLocale, reminder string) string`, maximum campaign-guidance/content-boundary lengths of 8 KiB, and campaign settings `content_boundaries` plus `narrative_locale` defaulting to `en`.

- [ ] **Step 1: Write failing precedence and truncation tests**

Assert exact section order, delimiter presence, 8-KiB bounds, locale validation, empty-section omission, and immutable mandatory privacy/protocol sections when campaign guidance contains text such as “ignore previous instructions.” Assert the default prompt no longer assumes consent or instructs the model to produce explicit non-consensual sexual content.

- [ ] **Step 2: Run and verify RED**

Run: `go test ./internal/api -run TestBuildSystemPrompt -v`

- [ ] **Step 3: Implement the prompt composer**

Use fixed section labels and treat campaign guidance/content boundaries as quoted campaign configuration, not as replacement text. The safe default avoids explicit sexual violence and does not assert facts about participant age or consent. Validate locale as a short BCP-47-style tag and instruct output in that locale while keeping protocol tokens stable. Build the prompt through this function in both streaming and non-streaming GM handlers.

- [ ] **Step 4: Correct UI copy and verify**

Add labelled GM Screen controls for campaign narration guidance, content boundaries, and narrative locale. Explain that these fields customize narration but cannot override privacy, provider, or protocol constraints. Run `go test ./internal/db ./internal/api && cd web && npm test -- --run`.

- [ ] **Step 5: Commit**

```bash
git add internal/db internal/api/prompt.go internal/api/prompt_test.go internal/api/routes.go internal/api/routes_campaign_config.go web/src/GMScreenPanel.tsx
git commit -m "fix: apply safe campaign narrative preferences"
```

### Task 7: Sanitize generated SVG with a strict allowlist

**Files:**
- Create: `internal/api/svg_sanitize.go`
- Create: `internal/api/svg_sanitize_test.go`
- Modify: `internal/api/automation.go`
- Modify: `internal/api/routes.go`

**Interfaces:**
- Produces: `SanitizeSVG(input string) (string, error)` and `ErrUnsafeSVG`.

- [ ] **Step 1: Write failing malicious and valid fixture tests**

Reject or remove `script`, `foreignObject`, `onload`, `onclick`, `javascript:`, external `href`, external CSS URLs, XML entities, and unsupported namespaces. Preserve basic `svg`, `g`, `path`, `rect`, `circle`, `line`, `polyline`, `polygon`, `text`, gradients, safe IDs, transforms, fills, strokes, and viewBox.

- [ ] **Step 2: Run and verify RED**

Run: `go test ./internal/api -run TestSanitizeSVG -v`

- [ ] **Step 3: Implement token-based XML rewriting**

Use `encoding/xml.Decoder` with `Strict = true`; maintain allowed-element and allowed-attribute maps; reject entity declarations and non-SVG roots; validate numeric/style values; allow fragment-only `url(#id)` references; write a normalized document with `encoding/xml.Encoder`. Return an error if unsafe active content is encountered rather than trying to preserve it.

- [ ] **Step 4: Apply sanitizer before every generated SVG write**

Sanitize AI map output before `CreateMap` and filesystem persistence. Delete partial files on failure and return a safe generation error.

- [ ] **Step 5: Verify and commit**

Run: `go test ./internal/api -run 'SVG|GenerateMap' -v`

```bash
git add internal/api/svg_sanitize.go internal/api/svg_sanitize_test.go internal/api/automation.go internal/api/routes.go
git commit -m "fix: reject active content in generated maps"
```

### Task 8: Phase 2 verification

**Files:**
- Create: `e2e/tests/reliability.spec.ts`

**Interfaces:**
- Verifies: queue health, recoverable breaker status, exact multiline streaming, reconnect UI, prompt guidance, and rejected unsafe SVG.

- [ ] **Step 1: Add maintained reliability scenarios**

Use disposable provider stubs and deterministic failures; do not call paid external providers.

- [ ] **Step 2: Run full verification**

Run:

```bash
gofmt -w cmd internal
go test ./...
go test -race ./internal/api ./internal/db ./internal/ai
go vet ./...
make build
cd web && npm test -- --run && npm run lint
cd ../e2e && npm test
```

Expected: every command exits 0 with no race report or unhandled frontend error.

- [ ] **Step 3: Commit**

```bash
git add e2e/tests/reliability.spec.ts
git commit -m "test: verify runtime reliability boundaries"
```
