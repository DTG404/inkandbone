# Phase 4 Maintainability, Delivery, and Documentation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the remediated application reproducible in local development and CI, consolidate browser coverage, prevent realtime contract drift, reduce oversized file responsibilities without behavior changes, and make documentation match the product.

**Architecture:** Establish one pinned verification toolchain first, then consolidate tests and generate realtime types from a small canonical contract. Refactor large files only behind characterization tests and keep file-movement commits separate from behavior changes.

**Tech Stack:** Go 1.26.5, Node 24.18.0 LTS, npm lockfiles, GitHub Actions, Playwright 1.61.1, JSON, React 19, TypeScript.

## Global Constraints

- CI failures are never converted to warnings with `|| echo`.
- GitHub workflow permissions remain read-only unless a job demonstrates a narrower write requirement.
- All dependency installations use lockfiles and `npm ci` in CI.
- The maintained E2E suite lives only under `e2e/` after coverage migration.
- Realtime generation uses a repository-owned Go standard-library program.
- Refactor commits do not change observable behavior.
- README and contributor guidance do not duplicate independently editable version facts.

---

### Task 1: Pin and patch the supported dependency baseline

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Create: `.nvmrc`
- Modify: `web/package.json`
- Modify: `web/package-lock.json`
- Modify: `e2e/package.json`
- Modify: `e2e/package-lock.json`

**Interfaces:**
- Establishes: Go `1.26.5`, Node `24.18.0`, `golang.org/x/image v0.43.0`, Playwright `1.61.1`, synchronized npm trees.

- [ ] **Step 1: Record the vulnerable baseline as a failing verification**

Run: `govulncheck ./...`.

Expected: reachable findings include GO-2026-5856 and GO-2026-5061.

Run: `cd web && npm audit`.

Expected: the Babel source-map advisory is reported.

- [ ] **Step 2: Upgrade the Go language/toolchain and image dependency**

Run:

```bash
go mod edit -go=1.26.5
go get golang.org/x/image@v0.43.0
go mod tidy
```

Verify `go version` is at least 1.26.5 before claiming the TLS runtime finding fixed.

- [ ] **Step 3: Repair frontend lockfile drift and the Babel advisory**

Run `cd web && npm ci` first. If the declared tree cannot install cleanly, use `npm install` once to synchronize the lockfile, then run `npm audit fix` without `--force`. Review the lockfile diff and reject major framework downgrades or unrelated package additions.

- [ ] **Step 4: Pin Node and synchronize E2E dependencies**

Write `24.18.0` to `.nvmrc`; set `@playwright/test` to exactly `1.61.1`; run `cd e2e && npm install`; then confirm the manifest and lockfile resolve exactly one Playwright version.

- [ ] **Step 5: Verify patched status**

Run:

```bash
go test ./...
govulncheck ./...
cd web && npm test -- --run && npm audit
cd ../e2e && npm audit
```

Expected: no reachable GO-2026-5856/GO-2026-5061 finding and no GHSA-4x5r-pxfx-6jf8 finding.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum .nvmrc web/package.json web/package-lock.json e2e/package.json e2e/package-lock.json
git commit -m "chore: patch and pin supported dependencies"
```

### Task 2: Make local verification and CI identical

**Files:**
- Modify: `Makefile`
- Modify: `.github/workflows/ci.yml`

**Interfaces:**
- Produces: `make verify`, `make verify-e2e`, deterministic CI jobs, pinned tool variables.

- [ ] **Step 1: Define local verification targets**

`make verify` runs, in order: `gofmt -l` check, `golangci-lint run ./...`, `go vet ./...`, `go test ./...`, `govulncheck ./...`, web `npm ci`, unit tests, ESLint, npm audit, production build, and gitleaks when installed. It must stop on the first failure.

`make verify-e2e` builds a disposable binary under `/tmp`, allocates a temporary database and port, starts the server, waits on `/api/health`, runs `npm ci` and Playwright under `e2e/`, and terminates the exact child PID through a shell trap.

- [ ] **Step 2: Rewrite CI as pinned, least-privilege jobs**

Set workflow `permissions: contents: read`; use Node 24.18.0 and Go 1.26.5; pin action releases or commit SHAs; pin golangci-lint v2.12.2 and a reviewed govulncheck version; use `npm ci`; run `make verify` and `make verify-e2e`; remove redundant frontend builds and every `|| echo` bypass.

- [ ] **Step 3: Exercise both targets locally**

Run: `make verify && make verify-e2e`.

Expected: both exit 0 and leave no server process or disposable database in the repository.

- [ ] **Step 4: Commit**

```bash
git add Makefile .github/workflows/ci.yml
git commit -m "ci: enforce reproducible full verification"
```

### Task 3: Consolidate Playwright coverage under `e2e/`

**Files:**
- Create: `docs/testing/e2e-coverage-matrix.md`
- Create focused specs under `e2e/tests/` for uncovered valuable scenarios.
- Modify: `e2e/package.json`
- Modify: `e2e/package-lock.json`
- Delete: `scripts/e2e-comprehensive.mjs`
- Delete: root `package.json`
- Delete: root `package-lock.json`

**Interfaces:**
- Produces: one Playwright installation and one maintained test command, `cd e2e && npm test`.

- [ ] **Step 1: Inventory every old assertion before deletion**

Read `scripts/e2e-comprehensive.mjs` section by section. In the coverage matrix, record old scenario name, current maintained spec/test, status `covered` or `obsolete`, and a concrete explanation for obsolete behavior. No row may use an unexplained status.

- [ ] **Step 2: Write failing maintained specs for uncovered current behavior**

Group scenarios by user workflow rather than creating one 187-assertion test. Use API setup for prerequisites and browser assertions for user-visible behavior. Run each new spec and confirm it fails because coverage-required behavior or selector support is absent.

- [ ] **Step 3: Make only the product/testability corrections required by those specs**

Prefer role/name selectors and stable accessible labels established in Phase 3. Do not add production-only test IDs when semantic selectors are sufficient.

- [ ] **Step 4: Prove coverage and remove duplication**

Run the full `e2e` suite twice against fresh disposable databases. Then delete the old script and root Node manifests. Run `rg -n 'e2e-comprehensive|node scripts/e2e' . --glob '!docs/testing/e2e-coverage-matrix.md'` and update active references.

- [ ] **Step 5: Verify and commit**

Run: `make verify-e2e`.

```bash
git add docs/testing e2e scripts/e2e-comprehensive.mjs package.json package-lock.json Makefile README.md
git commit -m "test: consolidate maintained browser coverage"
```

### Task 4: Generate Go and TypeScript realtime contracts

**Files:**
- Create: `contracts/realtime.json`
- Create: `internal/cmd/genrealtime/main.go`
- Create: `internal/cmd/genrealtime/main_test.go`
- Create: `internal/api/realtime_gen.go`
- Create: `web/src/realtime.gen.ts`
- Create: `scripts/verify-generated.sh`
- Modify: `internal/api/events.go`
- Modify: `web/src/types.ts`
- Modify: `scripts/verify-generated.sh`
- Modify: `Makefile`

**Interfaces:**
- Produces: `go generate ./internal/api`, generated Go event constants/payload structs, generated TypeScript discriminated unions, `make generate`.

- [ ] **Step 1: Write a minimal canonical contract**

The JSON contains a version, SSE event definitions, and every WebSocket event currently found through `rg 'EventType =|Type:' internal/api`. Each event declares a stable snake-case name and object properties with `string`, `number`, `boolean`, `object`, `array`, or nullable variants.

- [ ] **Step 2: Write failing generator golden tests**

Feed a three-event fixture to a pure `Generate(io.Reader) (goSource, tsSource, error)` function and assert exact formatted Go and TypeScript output. Add failures for duplicate names, unsupported types, invalid identifiers, and missing payload definitions.

- [ ] **Step 3: Run and verify RED**

Run: `go test ./internal/cmd/genrealtime -v`.

Expected: compile failure because the generator does not exist.

- [ ] **Step 4: Implement the standard-library generator**

Decode with `encoding/json` and `DisallowUnknownFields`, validate deterministically, render through `text/template`, format Go with `go/format`, sort every generated declaration, and write files only when bytes differ.

- [ ] **Step 5: Replace handwritten realtime names and unsafe frontend casts**

Have backend publishers use generated constants/payload structs. Have `useWebSocket` parse to the generated `RealtimeEvent` union after structural validation. Remove literal event names and duplicate frontend event interfaces.

- [ ] **Step 6: Complete generation verification**

`make generate` runs the generator. `scripts/verify-generated.sh` copies current generated files, runs generation, and fails with a diff if content changes. CI runs it in `make verify`.

- [ ] **Step 7: Verify and commit**

Run: `make generate && go test ./... && cd web && npm test -- --run && npm run build && cd .. && scripts/verify-generated.sh`.

```bash
git add contracts internal/cmd/genrealtime internal/api web/src scripts/verify-generated.sh Makefile
git commit -m "feat: generate typed realtime contracts"
```

### Task 5: Remove duplicated frontend advancement rules

**Files:**
- Modify: `internal/ruleset/advancement.go`
- Modify: `internal/ruleset/advancement_test.go`
- Modify: `internal/api/routes_advance.go`
- Modify: `internal/api/routes_advance_test.go`
- Modify: `internal/api/server.go`
- Modify: `web/src/api.ts`
- Modify: `web/src/App.tsx`
- Modify: `web/src/App.test.tsx`

**Interfaces:**
- Produces: `ruleset.MinimumXPCost(name string) (int, bool)` and `GET /api/rulesets/{id}/advancement-config` returning `{"minimum_xp": number, "supported": boolean}`.
- Removes: frontend `MIN_XP_TO_ADVANCE` domain table.

- [ ] **Step 1: Write failing backend parity tests**

For every ruleset currently listed in the frontend constant, assert `MinimumXPCost` equals the true cheapest legal advancement derived from the existing `XPCostFor` rules. Assert unknown/no-advancement systems return `false`. Add an API test resolving ruleset ID through the database.

- [ ] **Step 2: Run and verify RED**

Run: `go test ./internal/ruleset ./internal/api -run 'MinimumXPCost|AdvancementConfig' -v`.

Expected: compile failure because the backend interface and route do not exist.

- [ ] **Step 3: Implement the backend source of truth**

Keep the mapping beside `XPCostFor`, document each minimum through the rule it represents, return a typed response from the new read route, and register it with the existing ruleset routes.

- [ ] **Step 4: Write a failing frontend test and remove the duplicate**

Mock `/api/rulesets/{id}/advancement-config`, assert the XP suggestion control follows `minimum_xp`, then delete `MIN_XP_TO_ADVANCE` from `App.tsx` and consume the API value with a safe unsupported fallback.

- [ ] **Step 5: Verify and commit**

Run: `go test ./internal/ruleset ./internal/api && cd web && npm test -- --run App.test.tsx api.test.ts`.

```bash
git add internal/ruleset internal/api/routes_advance.go internal/api/routes_advance_test.go internal/api/server.go web/src/api.ts web/src/App.tsx web/src/App.test.tsx
git commit -m "fix: centralize advancement eligibility rules"
```

### Task 6: Split backend files behind characterization tests

**Files:**
- Create: `internal/api/routes_messages.go`
- Modify: `internal/api/routes_assets.go`
- Create: `internal/api/routes_world.go`
- Create: `internal/api/automation_jobs.go`
- Create: `internal/api/automation_prompts.go`
- Modify: `internal/api/routes.go`
- Modify: `internal/api/automation.go`
- Add characterization tests in `internal/api/routes_test.go` and focused new test files.

**Interfaces:**
- Preserves: all existing handler method names, route paths, JSON shapes, event types, and automation job behavior.

- [ ] **Step 1: Freeze behavior before moving code**

Add table-driven route contract tests for status, content type, JSON keys, and event publication for every handler being moved. Add prompt golden tests and automation job tests for every function moving out of `automation.go`.

- [ ] **Step 2: Run characterization tests and verify GREEN before refactor**

Run: `go test ./internal/api -run 'RouteContract|PromptGolden|AutomationJob' -v`.

Expected: PASS against the existing organization.

- [ ] **Step 3: Move message and GM-stream handlers without editing bodies**

Move `handleListMessages`, `handleCreateMessage`, `handleGMRespond`, `handleGMRespondStream`, and directly private helpers to `routes_messages.go`. Run focused tests immediately.

- [ ] **Step 4: Move world/resource handlers by current responsibility**

Move maps/pins/world notes/NPCs/objectives/items into `routes_world.go` only when they still share cohesive helpers; leave already decomposed route files unchanged. Keep asset code in `routes_assets.go`.

- [ ] **Step 5: Split automation orchestration from jobs and prompts**

Keep dispatcher/breaker orchestration in `automation.go`, move domain job bodies to `automation_jobs.go`, and prompt templates/composers to `automation_prompts.go`. Do not rename public behavior during the move.

- [ ] **Step 6: Verify no behavior diff**

Run: `gofmt -w internal/api && go test -race ./internal/api && go vet ./internal/api`.

Expected: PASS and route/prompt golden output unchanged.

- [ ] **Step 7: Commit file movement separately**

```bash
git add internal/api
git commit -m "refactor: separate api and automation responsibilities"
```

### Task 7: Split frontend transport, workspace, and styles behind tests

**Files:**
- Create: `web/src/api/transport.ts`
- Create domain clients under `web/src/api/`.
- Create: `web/src/session/NarrativeStream.tsx`
- Create: `web/src/session/PlayerComposer.tsx`
- Create: `web/src/session/RightWorkspace.tsx`
- Modify: `web/src/api.ts`
- Modify: `web/src/SessionView.tsx`
- Split `web/src/App.css` into files under `web/src/styles/`.
- Add characterization tests beside extracted modules.

**Interfaces:**
- Preserves: existing exported API function names through temporary re-exports, SessionView props, rendered accessible names, CSS class behavior, and route URLs.

- [ ] **Step 1: Add transport and SessionView characterization tests**

For every `api.ts` export, assert method, URL, headers, body, success decode, and non-2xx error behavior. For SessionView, snapshot accessible roles/names and verify story, composer, map drawer, and selected right panel callbacks.

- [ ] **Step 2: Run characterization tests and verify GREEN**

Run: `cd web && npm test -- --run api SessionView`.

- [ ] **Step 3: Extract transport and domain clients**

Move shared fetch/CSRF/error/SSE behavior to `api/transport.ts`; group clients by campaigns, characters, sessions, world, maps, combat, automation, and GM tools. Keep `api.ts` as re-exports until all importers are migrated, then remove it in a separate mechanical step.

- [ ] **Step 4: Extract SessionView components**

Move story rendering, input composer, and right workspace into named components with explicit props. Keep state at the lowest shared owner and avoid new global context unless three or more distant branches truly require the same mutable state.

- [ ] **Step 5: Split CSS by approved boundaries**

Import tokens, primitives, layout, navigation, narrative, panels, and responsive sheets from one small `App.css` entrypoint. Move rules without changing declarations, run screenshot tests, then remove duplicate selectors.

- [ ] **Step 6: Verify and commit**

Run: `cd web && npm test -- --run && npm run lint && npm run build && cd ../e2e && npm test`.

```bash
git add web/src e2e
git commit -m "refactor: separate frontend transport and workspace modules"
```

### Task 8: Correct and centralize product documentation

**Files:**
- Modify: `README.md`
- Modify: `AGENTS.md` if it is stored in the repository.
- Create: `docs/security.md`
- Create: `docs/data-recovery.md`
- Create: `docs/development.md`
- Modify: other tracked documentation found by stale-fact searches.

**Interfaces:**
- Establishes: `docs/development.md` as the canonical supported-toolchain and verification source.

- [ ] **Step 1: Generate a stale-fact inventory**

Run:

```bash
rg -n 'Go 1\.22|React 18|30 migrations|46 migrations|13 systems|14 systems|187 assertions|:7432|/api/files|private.*local|nothing leaves' README.md docs AGENTS.md
```

Record every match in the plan execution notes before editing.

- [ ] **Step 2: Write security and provider data-flow documentation**

Document loopback defaults, intentional TLS/auth exposure, session lifetime, CSRF, allowed origins, typed assets, cloud-provider prompt data, whisper exclusions, and Ollama-local behavior without marketing ambiguity.

- [ ] **Step 3: Write backup and recovery documentation**

Document automatic pre-repair backup naming, integrity failure behavior, `orphaned_records`, restoration steps using a copied database, and how to verify `integrity_check`/`foreign_key_check` without exposing the live database through HTTP.

- [ ] **Step 4: Centralize development versions and commands**

Use values from `go.mod`, `.nvmrc`, package manifests, migration file count, and live ruleset query/tests. README and contributor docs link to `docs/development.md` for version/tool commands.

- [ ] **Step 5: Verify documentation claims mechanically**

Run the stale-fact search again, `go test ./...`, `make verify`, and `make verify-e2e`. Confirm every command shown in documentation exists and succeeds.

- [ ] **Step 6: Commit**

```bash
git add README.md AGENTS.md docs
git commit -m "docs: align security recovery and development guidance"
```

### Task 9: Final remediation verification and review preparation

**Files:**
- Create: `docs/testing/remediation-verification.md`

**Interfaces:**
- Records: exact commands, dates, versions, pass/fail counts, and any explicitly accepted non-reachable advisory.

- [ ] **Step 1: Run the complete clean verification from a fresh install state**

Run:

```bash
go clean -testcache
npm ci --prefix web
npm ci --prefix e2e
make verify
make verify-e2e
go test -race ./internal/api ./internal/db ./internal/ai
```

Expected: every command exits 0 from lockfile-clean dependency installations.

- [ ] **Step 2: Re-run the original exploit and integrity reproductions**

Using a disposable database, prove the SQLite file returns 404, a sentinel whisper is absent from captured provider prompts, `foreign_keys` equals 1, campaign deletion leaves no unexplained child, malicious SVG is rejected, and a breaker recovers after cooldown.

- [ ] **Step 3: Complete the accessibility release checklist**

Record browser, viewport, theme, keyboard, screen-reader, zoom, contrast, and reduced-motion results in `docs/testing/remediation-verification.md`; do not mark unperformed manual checks as passed.

- [ ] **Step 4: Request code review before integration**

Use `superpowers:requesting-code-review` against the approved design and all four plans. Address findings through test-first corrections.

- [ ] **Step 5: Commit verification evidence**

```bash
git add docs/testing/remediation-verification.md
git commit -m "docs: record complete remediation verification"
```
