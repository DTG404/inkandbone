# Complete Project Remediation Design

**Date:** 2026-07-16

**Status:** Approved design

## Objective

Remediate the complete project audit without replacing Ink & Bone's Go, React, SQLite, or single-binary architecture. The work covers security, privacy, data integrity, runtime reliability, realtime protocols, responsive UX, accessibility, dependency management, CI, maintainability, and documentation.

The application remains local-first. It binds only to loopback by default. Intentional non-loopback access is authenticated and encrypted.

## Delivery strategy

Work is divided into four independently testable phases:

1. Security, privacy, and data integrity.
2. Runtime reliability and protocol correctness.
3. Responsive UI, accessibility, and interaction quality.
4. Maintainability, reproducible delivery, and documentation.

Each phase must leave the application buildable and usable. Behavior-changing work follows test-driven development. Refactoring begins only after characterization tests cover the behavior being moved.

## Global constraints

- Preserve the worn-grimoire visual identity and existing game-system functionality.
- Preserve unrelated user changes, including the existing modifications to `.claude/settings.local.json` and `ttrpg-e2e`.
- Do not rewrite the application in a different framework or database.
- Do not modify historical migrations. Add incremental migrations.
- Back up and validate an existing database before any repair that can remove or restructure data.
- Keep secrets process-scoped. Never log, persist, or return an authentication secret.
- Fail closed when security configuration is incomplete.
- Make privacy exclusions a data-layer rule rather than a caller convention.
- Keep generated protocol artifacts checked in and reproducible.
- Keep all existing supported rulesets and workflows unless a test proves an entry is an accidental placeholder.

## Phase 1: Security, privacy, and data integrity

### Network boundary

The command accepts a listen address and defaults to `127.0.0.1:7432`. Loopback operation does not require login. A non-loopback address requires:

- A configured authentication secret of at least 32 random bytes.
- A TLS certificate and key for direct access.
- Explicit allowed browser origins.

Startup fails before opening a socket when any non-loopback requirement is missing. Reverse-proxy deployments may keep Ink & Bone loopback-bound and terminate authentication/TLS at the proxy; direct non-loopback plaintext HTTP is not supported.

### Authentication and sessions

The configured master secret is compared in constant time and never copied into a browser cookie. A successful login creates a cryptographically random, server-side session with:

- An opaque `HttpOnly` cookie.
- `SameSite=Strict`.
- `Secure` when TLS is active.
- A finite idle and absolute lifetime.
- Explicit logout and restart-time revocation.

Login attempts are rate limited per remote address. Browser mutations require a CSRF token tied to the session. Non-browser API clients may authenticate with the master secret as a bearer token. WebSocket upgrades use the authenticated browser session and enforce the configured origin list.

Authentication state is held in memory because the application is single-process and single-user. Restarting intentionally invalidates all sessions.

### Asset isolation

The generic `/api/files/{path}` route is removed. Public asset handlers accept a typed resource and database ID, load its stored record, and resolve the corresponding file inside an allowlisted directory. Resolution must reject:

- Absolute paths.
- Parent traversal.
- Symlink escapes.
- Unknown extensions or MIME types.
- Files not referenced by the requested database record.

Responses set an exact content type, `X-Content-Type-Options: nosniff`, a restrictive content-security policy where applicable, and a safe disposition. SVG is never treated as active HTML.

### Message privacy

The database exposes a single AI-visible message query that excludes whispers. Every GM-context, automation, recap, export, indexing, search, and provider request that is not explicitly a private whisper operation consumes this query.

Tests pass a unique sentinel whisper through each AI-related workflow and assert that the sentinel is absent from provider requests, recaps, exports, and indexes. Privacy is not considered covered by tests that only inspect the normal GM response path.

Documentation states which data each provider receives. Only Ollama operation is described as fully local; DeepSeek, Anthropic, and OpenRouter receive the prompt context needed for configured work.

### Database migration safety

SQLite connections enable foreign keys through connection configuration so every connection has the same behavior. Before the first integrity-repair migration, startup:

1. Creates a consistent SQLite backup beside the database.
2. Opens and validates the backup.
3. Runs `PRAGMA integrity_check` on the source.
4. Stops without changing data if either check fails.

The migration records orphaned rows in a recovery table containing source table, original primary key, JSON payload, detection time, and repair reason. It then rebuilds affected tables where necessary and assigns explicit relationship behavior:

- `CASCADE` when a child has no independent meaning after its parent is deleted.
- `SET NULL` when a record remains meaningful without the optional parent.
- `RESTRICT` when deletion would destroy meaningful independent records.

After migration, startup runs `PRAGMA foreign_key_check` and stops on any violation. Campaign, session, character, and adventure deletion receive integration tests covering every dependent table.

The executable migration placeholder is removed through a new migration. The reusable ruleset template moves outside the embedded `.sql` directory and cannot be executed by the migration runner.

### API boundary

Common middleware provides:

- Per-route request body limits.
- Strict JSON decoding with unknown-field rejection.
- Consistent content-type validation.
- Opaque error responses containing a request ID.
- Detailed server-side structured logs without secret values.
- Security response headers.

Existing domain validation remains in its domain packages; the middleware standardizes transport behavior rather than duplicating business rules.

## Phase 2: Runtime reliability and protocol correctness

### Automation dispatcher

Post-response automation runs through a bounded dispatcher instead of unconstrained goroutines. Each job identifies its campaign, session, automation type, deadline, and cancellation context.

Snapshot-based jobs such as recap regeneration or roster extraction may coalesce so only the newest pending snapshot runs. Event-based jobs involving currency, items, dice, advancement, or other irreversible deltas never coalesce or silently drop. Ordering-sensitive work executes sequentially for the same session.

On shutdown, the dispatcher stops accepting work, drains for a bounded interval, then cancels remaining jobs. Queue saturation is visible as a structured error and UI health state; it is never silently ignored.

### Circuit breakers and health

Circuit breakers are keyed by provider and automation. Each breaker has closed, open, and half-open states, a failure threshold, timed cooldown, and a single probe after cooldown. Success resets only the relevant breaker.

Automation health exposes queued, running, cooling-down, last-success, and sanitized last-error information through the settings API and Manage panel.

### Timeouts and lifecycle

The application owns a root context cancelled by operating-system signals. The HTTP server uses read-header, idle, and maximum-header limits and performs graceful shutdown.

Provider clients use connection and response-header timeouts. Automation requests use bounded task deadlines. GM streaming uses an inactivity watchdog that resets when data arrives plus an absolute upper limit, preventing both indefinite hangs and premature termination of healthy streams.

### Server-Sent Events

SSE uses typed JSON payloads:

- `delta` with a text fragment.
- `complete` with final metadata.
- `error` with a safe user-facing code and request ID.

The client parser buffers incomplete frames and preserves embedded newlines, UTF-8 boundaries, multiple frames per read, and partial reads. Tests cover each case. Reconnection must not resubmit a completed player action.

### WebSockets

Realtime events include a monotonically increasing in-process sequence number. The client detects gaps or reconnects and reloads authoritative state rather than attempting durable event replay. Reconnection uses exponential backoff with jitter and exposes offline/reconnecting state.

The browser chooses `ws://` or `wss://` from `location.protocol`. Event-channel overflow is logged and triggers reconciliation instead of silent divergence.

### Prompt override

Campaign-specific GM guidance is inserted in a delimited, size-limited section after ruleset guidance and before per-turn reminders. It customizes narration but cannot replace mandatory privacy, protocol, or system-integrity instructions. The UI copy reflects this precedence accurately.

### Generated SVG

Generated maps pass through a strict XML sanitizer that allowlists required SVG elements and attributes. Scripts, event handlers, `foreignObject`, external references, CSS imports, and unsupported constructs are rejected or removed. Sanitization fails closed. Asset responses add restrictive headers even after sanitization.

## Phase 3: Responsive UI, accessibility, and interaction quality

### Workspace layout

Ink & Bone retains an adaptive three-pane desktop workspace:

- Left and right panels are resizable within tested minimum and maximum widths.
- Panels can collapse without hiding the control that restores them.
- Width and collapsed state persist locally.
- A visible command resets the workspace layout.

At tablet widths, side panels become modal or non-modal drawers as appropriate. On phones, bottom navigation exposes Story, Character, World, and GM directly. General settings remain available from the application header.

The shell uses dynamic viewport units and safe-area insets. All features remain reachable at 320 CSS pixels without two-dimensional page scrolling.

### Navigation and visual hierarchy

The clipped right-tab row becomes grouped Play, World, and GM navigation with visible labels and overflow-safe behavior. Desktop navigation remains efficient for frequent switching; mobile navigation prioritizes the four primary destinations.

The worn-grimoire palette and typography remain. Theme-token changes provide:

- At least 4.5:1 contrast for normal text.
- At least 3:1 for large text and meaningful non-text controls.
- Larger default labels and controls.
- Clearer surface separation.
- Gold as accessible emphasis and active state rather than low-contrast body text.

Interactive touch targets are at least 44 by 44 CSS pixels where layout permits. Narrative content remains the dominant visual surface.

### Accessible primitives

Reusable Dialog, Drawer, Tabs, IconButton, interactive-row, form-field, and status/toast components provide consistent:

- Native button semantics or equivalent keyboard activation.
- Visible `:focus-visible` treatment.
- Accessible names and state relationships.
- Dialog focus trapping, Escape handling, focus restoration, and scroll management.
- Tab roles, selected state, and controlled-panel relationships.
- Reduced-motion alternatives.

Existing clickable non-semantic elements are migrated to these primitives. Removing a browser outline without an equivalent focus indicator is prohibited.

### Async interaction states

User-triggered requests show loading, success, empty, recoverable error, offline, and reconnecting states. Failures are not limited to console logging. Destructive actions retain confirmation and identify the affected record.

Automation health from Phase 2 is shown in Manage with understandable status labels and recovery guidance.

### Frontend tests

The existing computed-field test waits for the observable computed update. Map tests provide token and zone responses and a `ResizeObserver` implementation. New tests cover:

- Keyboard-only primary workflows.
- Dialog and drawer focus behavior.
- Navigation reachability.
- Responsive layouts at 1440, 1024, 768, 390, and 320 CSS pixels.
- 200 percent zoom and reflow.
- Reduced motion.
- Reconnection and recoverable errors.
- Automated accessibility scanning.

Automated accessibility results are supplemented by a release checklist for keyboard navigation, screen-reader spot checks, contrast, zoom/reflow, and reduced motion.

## Phase 4: Maintainability, delivery, and documentation

### Reproducible toolchain and CI

The supported Go release is upgraded to a version containing the identified TLS fix. `golang.org/x/image` and affected frontend dependencies are upgraded to patched versions.

CI uses pinned Go, Node, linter, vulnerability scanner, secret scanner, Playwright, and GitHub Action versions. It uses `npm ci`, grants read-only permissions unless a job proves it requires more, and fails on:

- Formatting or lint errors.
- Go or frontend unit-test failures.
- Reachable vulnerability findings covered by the project's policy.
- Production build failures.
- Maintained browser-test failures.
- Stale generated realtime contracts.

`make verify` runs formatting checks, lint, audits, unit tests, and production builds. `make verify-e2e` creates a disposable database, starts the built server, runs Playwright, and cleans up reliably. CI runs both.

### Playwright consolidation

All browser tests and their browser dependency live under `e2e/`. Before removing the older comprehensive runner, a coverage matrix maps each valuable scenario to a maintained test or records why the scenario is obsolete. The root Playwright dependency and obsolete runner are removed only after this review is complete.

### File boundaries

Large files are split only after relevant characterization tests pass:

- `internal/api/routes.go`: transport helpers, assets, GM/message streaming, and remaining resource-specific handlers.
- `internal/api/automation.go`: dispatcher, breaker/health, prompts, and individual automation jobs.
- `web/src/api.ts`: shared transport/SSE plus domain-specific clients.
- `web/src/SessionView.tsx`: workspace shell, narrative stream, grouped navigation, and responsive panel composition.
- `web/src/App.css`: theme tokens, shell, primitives, panel groups, and responsive rules.

File moves and behavior changes are separate commits. The split follows current responsibilities and does not introduce a new web framework or styling system.

### Realtime contract

A small canonical JSON contract defines WebSocket and SSE event names and payload shapes. A repository-owned generator using the Go standard library emits Go constants/types and TypeScript types. Generated files are checked in. CI regenerates them and fails if the worktree changes.

This contract covers realtime protocols only. A full OpenAPI conversion is outside scope.

### Documentation

Documentation is updated to match the supported Go and React versions, actual migration and ruleset counts, test commands, provider data flows, network/authentication behavior, database backup and recovery, accessibility expectations, and new module boundaries.

One contributor-facing toolchain section is canonical. Other documents link to it rather than independently duplicating version facts.

## Error handling principles

- User-facing errors explain what action failed and whether retry is safe.
- Internal errors include request/job IDs and structured context, never secrets or full private prompts.
- Security configuration errors stop startup with actionable messages.
- Data migration errors leave the original database and validated backup recoverable.
- Background job failures remain isolated to their automation and cannot permanently disable unrelated work.
- Protocol errors trigger safe reconciliation rather than silent state divergence.

## Verification and completion criteria

The remediation is complete only when:

- The server cannot bind non-loopback without the required authentication and TLS configuration.
- Arbitrary files and the SQLite database cannot be retrieved through HTTP.
- Sentinel whispers never reach providers, recaps, exports, search, or indexes.
- Foreign keys are enabled on every connection and `foreign_key_check` is clean.
- Parent deletion tests cover all dependent tables without unexplained orphans.
- Automation queues are bounded, event jobs are not lost, and breakers recover after cooldown.
- SSE preserves exact streamed text across newline, UTF-8, and chunk-boundary tests.
- WebSocket reconnects reconcile authoritative state and display connection status.
- Generated SVG rejects active content.
- All UI functionality is reachable by keyboard and at the specified viewport widths.
- Automated accessibility scans and the manual accessibility checklist pass.
- Go tests, race tests, frontend tests, lint, vulnerability policy, builds, and maintained E2E tests pass in CI.
- Documentation accurately describes current behavior and supported tooling.

## Explicit non-goals

- Replacing SQLite, React, Vite, or the Go HTTP server.
- Building multi-user accounts, permissions, or cloud synchronization.
- Durable WebSocket event replay.
- Full OpenAPI generation for every endpoint.
- A wholesale visual redesign.
- Adding unrelated game systems or gameplay features during remediation.
