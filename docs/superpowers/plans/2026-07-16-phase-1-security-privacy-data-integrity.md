# Phase 1 Security, Privacy, and Data Integrity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Ink & Bone loopback-safe by default, authenticated and encrypted when exposed, incapable of serving arbitrary files, whisper-private across every AI workflow, and referentially correct under SQLite foreign-key enforcement.

**Architecture:** Add opt-in security configuration around the existing `Server`, centralize authenticated request handling and AI-visible message reads, replace filesystem paths with typed asset IDs, and apply incremental backup-guarded integrity migrations. Preserve `NewServer` as the loopback/test convenience constructor and add explicit options for the binary.

**Tech Stack:** Go 1.26.5+, `net/http`, `crypto/rand`, `crypto/subtle`, SQLite via `modernc.org/sqlite`, React 19, Vitest.

## Global Constraints

- Default listen address is exactly `127.0.0.1:7432`.
- Non-loopback direct access requires a 32-byte-or-longer secret, TLS certificate/key, and allowed origins.
- The configured authentication secret is never stored in a cookie, database, response, or log.
- Existing migrations remain unchanged; new migrations begin after `055_`.
- Repair work creates and validates a database backup before modifying existing relational data.
- Whisper content never enters AI-visible queries.
- Existing modifications to `.claude/settings.local.json` and `ttrpg-e2e` remain untouched.

---

### Task 1: Validate the listen security boundary

**Files:**
- Create: `internal/api/security_config.go`
- Create: `internal/api/security_config_test.go`
- Modify: `cmd/ttrpg/main.go`

**Interfaces:**
- Produces: `api.ListenSecurityConfig`, `api.ValidateListenSecurity(addr string, cfg ListenSecurityConfig) error`, flags `-listen`, `-tls-cert`, `-tls-key`, and `-allowed-origin`.
- Consumes: `TTRPG_AUTH_SECRET` from the process environment.

- [ ] **Step 1: Write failing table tests for loopback and non-loopback validation**

```go
func TestValidateListenSecurity(t *testing.T) {
	tests := []struct {
		name string
		addr string
		cfg  ListenSecurityConfig
		wantErr string
	}{
		{name: "loopback needs no auth", addr: "127.0.0.1:7432"},
		{name: "ipv6 loopback needs no auth", addr: "[::1]:7432"},
		{name: "public needs secret", addr: "0.0.0.0:7432", wantErr: "TTRPG_AUTH_SECRET"},
		{name: "public needs tls", addr: "0.0.0.0:7432", cfg: ListenSecurityConfig{AuthSecret: strings.Repeat("x", 32)}, wantErr: "TLS"},
		{name: "public needs origin", addr: "0.0.0.0:7432", cfg: ListenSecurityConfig{AuthSecret: strings.Repeat("x", 32), TLSCertFile: "cert.pem", TLSKeyFile: "key.pem"}, wantErr: "allowed origin"},
		{name: "public complete", addr: "0.0.0.0:7432", cfg: ListenSecurityConfig{AuthSecret: strings.Repeat("x", 32), TLSCertFile: "cert.pem", TLSKeyFile: "key.pem", AllowedOrigins: []string{"https://table.example"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateListenSecurity(tt.addr, tt.cfg)
			if tt.wantErr == "" { require.NoError(t, err); return }
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
```

- [ ] **Step 2: Run the test and verify RED**

Run: `go test ./internal/api -run TestValidateListenSecurity -v`

Expected: compile failure because `ListenSecurityConfig` and `ValidateListenSecurity` do not exist.

- [ ] **Step 3: Implement address classification and fail-closed validation**

```go
type ListenSecurityConfig struct {
	AuthSecret    string
	TLSCertFile   string
	TLSKeyFile    string
	AllowedOrigins []string
}

func ValidateListenSecurity(addr string, cfg ListenSecurityConfig) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil { return fmt.Errorf("listen address: %w", err) }
	ip := net.ParseIP(host)
	if host == "localhost" || (ip != nil && ip.IsLoopback()) { return nil }
	if len(cfg.AuthSecret) < 32 { return errors.New("non-loopback listen requires TTRPG_AUTH_SECRET of at least 32 bytes") }
	if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" { return errors.New("non-loopback listen requires TLS certificate and key") }
	if len(cfg.AllowedOrigins) == 0 { return errors.New("non-loopback listen requires at least one allowed origin") }
	return nil
}
```

For non-loopback configuration, also load the certificate/key with `tls.LoadX509KeyPair` and reject unreadable or mismatched files during validation. In every mode, WebSocket origin validation requires the request origin host to match the HTTP host or the explicit allowlist; `CheckOrigin: true` is removed.

- [ ] **Step 4: Add flags and validate before opening the database or socket**

Use repeatable comma-separated origins, default `-listen` to `127.0.0.1:7432`, construct `ListenSecurityConfig` from flags plus `os.Getenv("TTRPG_AUTH_SECRET")`, and call `ValidateListenSecurity` immediately after `flag.Parse()`.

- [ ] **Step 5: Run focused and package tests**

Run: `gofmt -w internal/api/security_config.go internal/api/security_config_test.go cmd/ttrpg/main.go && go test ./internal/api ./cmd/ttrpg`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add cmd/ttrpg/main.go internal/api/security_config.go internal/api/security_config_test.go
git commit -m "fix: fail closed for non-loopback serving"
```

### Task 2: Add server-side login sessions, CSRF, and origin enforcement

**Files:**
- Create: `internal/api/auth.go`
- Create: `internal/api/auth_test.go`
- Modify: `internal/api/server.go`
- Modify: `internal/api/ws.go`
- Modify: `cmd/ttrpg/main.go`
- Create: `web/src/LoginScreen.tsx`
- Create: `web/src/LoginScreen.test.tsx`
- Modify: `web/src/App.tsx`

**Interfaces:**
- Produces: `ServerOptions{Security ListenSecurityConfig}`, `NewServerWithOptions`, `POST /api/auth/login`, `POST /api/auth/logout`, `GET /api/auth/session`, `SessionInfo{Authenticated, CSRFToken}`.
- Preserves: `NewServer(database, dataDir, aiClient)` for loopback tests.

- [ ] **Step 1: Write failing handler tests**

Cover invalid secret, valid constant-time login, rate-limited repeated failures, `HttpOnly`/`SameSite=Strict` cookie flags, CSRF rejection on mutation, bearer-token API access, logout, expiry via injected clock, and WebSocket origin rejection.

```go
func TestAuthLoginCreatesOpaqueSession(t *testing.T) {
	s := newSecureTestServer(t, strings.Repeat("s", 32), "https://table.example")
	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"secret":"`+strings.Repeat("s", 32)+`"}`))
	r.RemoteAddr = "192.0.2.4:1234"
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	require.Equal(t, http.StatusNoContent, w.Code)
	cookie := requireAuthCookie(t, w.Result().Cookies())
	assert.True(t, cookie.HttpOnly)
	assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
	assert.NotEqual(t, strings.Repeat("s", 32), cookie.Value)
}
```

- [ ] **Step 2: Run and verify RED**

Run: `go test ./internal/api -run 'TestAuth|TestWebSocketOrigin' -v`

Expected: compile failure for the secure server/session interfaces.

- [ ] **Step 3: Implement an in-memory session manager**

```go
type sessionRecord struct { csrf string; createdAt, lastSeen time.Time }
type sessionManager struct {
	mu sync.Mutex
	secret string
	sessions map[string]sessionRecord
	now func() time.Time
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil { return "", err }
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func secretMatches(got, want string) bool {
	if len(got) != len(want) { return false }
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
```

Use a 30-minute idle lifetime, 12-hour absolute lifetime, a five-attempt-per-minute remote-address limiter, and restart-time revocation by keeping sessions only in memory.

- [ ] **Step 4: Wrap protected routes and validate CSRF for cookie-authenticated mutations**

Keep `/api/health`, `/api/auth/login`, and static login assets public. Require authentication for `/api/*` and `/ws` only when security is enabled. Accept `Authorization: Bearer <master-secret>` for API clients without CSRF; require `X-CSRF-Token` for non-safe cookie requests.

- [ ] **Step 5: Implement TLS serving and secure shutdown compatibility**

Pass `ServerOptions` from `main`, select `ListenAndServeTLS` for configured non-loopback operation, and ensure cookies use `Secure` whenever TLS is configured.

- [ ] **Step 6: Add the login UI and fetch CSRF integration**

`LoginScreen` submits the secret once, clears the input immediately, and never stores it. Add `web/src/transport.ts` with `setCSRFToken` and `request` so mutations automatically send `X-CSRF-Token` after `/api/auth/session` succeeds.

Run `rg -n 'fetch\\(' web/src` and migrate every non-GET browser request to `request` in this task. Add transport tests for POST, PATCH, and DELETE proving the CSRF header is present; remote authenticated mode must not leave existing mutations broken.

- [ ] **Step 7: Verify backend and frontend behavior**

Run: `go test ./internal/api ./cmd/ttrpg && cd web && npm test -- --run LoginScreen`

Expected: PASS with no secret value in snapshots or logs.

- [ ] **Step 8: Commit**

```bash
git add cmd/ttrpg/main.go internal/api/auth.go internal/api/auth_test.go internal/api/server.go internal/api/ws.go web/src/App.tsx web/src/LoginScreen.tsx web/src/LoginScreen.test.tsx web/src/transport.ts
git commit -m "feat: authenticate intentional network access"
```

### Task 3: Replace arbitrary file serving with typed assets

**Files:**
- Create: `internal/api/routes_assets.go`
- Create: `internal/api/routes_assets_test.go`
- Modify: `internal/api/server.go`
- Modify: `internal/api/routes.go`
- Modify: `internal/db/queries_world.go`
- Modify: `internal/db/queries_core.go`
- Modify: `web/src/api.ts`
- Modify callers containing `/api/files/` under `web/src/`

**Interfaces:**
- Produces: `GET /api/assets/maps/{id}`, `GET /api/assets/portraits/{id}`, `db.GetMap`, `db.GetMapByImagePath`, and existing `db.GetCharacter` lookups as the only path sources.
- Produces temporarily: a database-backed compatibility redirect for existing `/api/files/maps/{filename}` message links; it resolves only an exact `maps.image_path` row and cannot serve other files.
- Removes immediately: arbitrary `GET /api/files/{path...}` behavior.

- [ ] **Step 1: Replace old file-route tests with failing typed-asset tests**

Test successful referenced map/portrait delivery, missing database record, absolute path, `..`, symlink escape, wrong extension, mismatched MIME, `nosniff`, and failed attempts to request the database filename.

```go
func TestAssetRouteCannotServeDatabase(t *testing.T) {
	s := newTestServerWithDir(t, t.TempDir())
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/files/ttrpg.db", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}
```

- [ ] **Step 2: Run and verify RED**

Run: `go test ./internal/api -run 'TestAsset|TestServeFile' -v`

Expected: old generic-file success tests fail the new requirement.

- [ ] **Step 3: Implement canonical allowlisted resolution**

```go
func resolveAsset(baseDir, relative string, allowedExt map[string]string) (string, string, error) {
	if filepath.IsAbs(relative) { return "", "", errInvalidAsset }
	base, err := filepath.EvalSymlinks(baseDir); if err != nil { return "", "", err }
	full, err := filepath.EvalSymlinks(filepath.Join(baseDir, filepath.Clean(relative))); if err != nil { return "", "", err }
	rel, err := filepath.Rel(base, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) { return "", "", errInvalidAsset }
	mimeType, ok := allowedExt[strings.ToLower(filepath.Ext(full))]
	if !ok { return "", "", errInvalidAsset }
	return full, mimeType, nil
}
```

Maps permit sanitized `.svg`, `.png`, `.jpg`, `.jpeg`, and `.webp`; portraits permit raster formats only. Set `Content-Security-Policy: sandbox; default-src 'none'`, `X-Content-Type-Options: nosniff`, and `Content-Disposition: inline` with a sanitized filename.

- [ ] **Step 4: Register typed routes and delete the generic handler**

Remove the arbitrary `/api/files/{path...}` handler and its traversal special case in `ServeHTTP`. Register a temporary exact legacy-map handler that accepts only `maps/<base filename>`, looks up the complete stored path through `db.GetMapByImagePath`, and redirects to `/api/assets/maps/{id}`. It returns 404 for database names, portraits, unknown rows, subdirectories, and traversal. Update new map and portrait URLs returned or constructed by the frontend to use record IDs.

- [ ] **Step 5: Verify**

Run: `go test ./internal/api ./internal/db && cd web && npm test -- --run`

Expected: typed asset tests pass; the only source match for `/api/files/` is the temporary exact legacy-map compatibility route and its tests.

- [ ] **Step 6: Commit**

```bash
git add internal/api/routes_assets.go internal/api/routes_assets_test.go internal/api/server.go internal/api/routes.go internal/db/queries_world.go internal/db/queries_core.go web/src
git commit -m "fix: restrict asset delivery to database records"
```

### Task 4: Enforce whisper privacy at the database boundary

**Files:**
- Modify: `internal/db/queries_session.go`
- Modify: `internal/db/queries_session_test.go`
- Modify: `internal/api/automation.go`
- Modify: `internal/api/routes.go`
- Modify: `internal/api/routes_test.go`

**Interfaces:**
- Produces: `ListAIVisibleMessages(sessionID int64) ([]Message, error)`.
- Preserves: `ListMessages` for authorized display of all messages.

- [ ] **Step 1: Write a failing data-layer privacy test**

```go
func TestListAIVisibleMessagesExcludesWhispers(t *testing.T) {
	d := newTestDB(t); campaignID := setupCampaign(t, d); sessionID := setupSession(t, d, campaignID)
	_, _ = d.CreateMessage(sessionID, "user", "PUBLIC_SENTINEL", false, nil)
	_, _ = d.CreateMessage(sessionID, "user", "PRIVATE_SENTINEL", true, nil)
	messages, err := d.ListAIVisibleMessages(sessionID)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, "PUBLIC_SENTINEL", messages[0].Content)
}
```

- [ ] **Step 2: Run and verify RED**

Run: `go test ./internal/db -run TestListAIVisibleMessagesExcludesWhispers -v`

Expected: compile failure because the query does not exist.

- [ ] **Step 3: Implement the centralized query**

Use the same selected columns and ordering as `ListMessages`, adding `WHERE session_id = ? AND whisper = 0`.

- [ ] **Step 4: Add provider-sentinel tests for recap and GM context**

Extend the existing stub completer/streamer to capture prompts. Seed both sentinel messages, run `handleGenerateRecap`, `autoUpdateRecap`, and the GM stream context path, then assert `PRIVATE_SENTINEL` is absent and `PUBLIC_SENTINEL` is present.

- [ ] **Step 5: Replace every AI-context message read**

Run `rg -n 'ListMessages\(' internal/api internal/ai internal/mcp` and classify each caller. Provider, recap, automation, indexing, and export callers use `ListAIVisibleMessages`; only authenticated message-display handlers retain `ListMessages`.

- [ ] **Step 6: Verify all privacy tests**

Run: `go test ./internal/db ./internal/api -run 'AIVisible|Whisper|Recap|GMRespond' -v`

Expected: PASS and no captured provider prompt contains the private sentinel.

- [ ] **Step 7: Commit**

```bash
git add internal/db/queries_session.go internal/db/queries_session_test.go internal/api/automation.go internal/api/routes.go internal/api/routes_test.go
git commit -m "fix: exclude whispers from all AI context"
```

### Task 5: Enable foreign keys and backup-guarded migration validation

**Files:**
- Create: `internal/db/integrity.go`
- Create: `internal/db/integrity_test.go`
- Modify: `internal/db/db.go`
- Modify: `internal/db/db_test.go`

**Interfaces:**
- Produces: `OpenOptions{BackupBeforeRepair bool}`, `OpenWithOptions(path string, opts OpenOptions)`, internal `backupDatabase`, `integrityCheck`, and `foreignKeyCheck`.
- Preserves: `Open(path)` delegating to production-safe defaults.

- [ ] **Step 1: Write failing tests for per-connection foreign keys and validated backup**

Open a file database, assert `PRAGMA foreign_keys` equals `1`, create multiple connections through the pool and repeat, corrupt a copied fixture and assert startup refuses it, and verify a valid pre-repair backup can be independently opened.

- [ ] **Step 2: Run and verify RED**

Run: `go test ./internal/db -run 'ForeignKeys|Integrity|Backup' -v`

Expected: foreign-key assertion reports `0` and backup interfaces are missing.

- [ ] **Step 3: Configure SQLite pragmas in the DSN**

Build the modernc SQLite DSN with `_pragma=foreign_keys(1)`, `_pragma=busy_timeout(5000)`, and existing single-writer settings. Do not rely on executing the pragma once after opening the pool.

- [ ] **Step 4: Implement consistent backup and validation**

Use SQLite `VACUUM INTO ?` or the driver's backup support while the source is open, validate the resulting file with `integrity_check`, use mode `0600`, and return the backup path in migration errors. Skip filesystem backup only for `:memory:` tests.

Add a repair-migration registry keyed by migration filename. `runMigrations` checks whether a registered repair migration is pending and invokes the validated backup before beginning that migration. Ordinary additive migrations do not create a backup on every startup.

- [ ] **Step 5: Replace semicolon splitting with transactional whole-file execution**

Add a migration-runner test containing a quoted semicolon and a trigger body. Execute each migration file as one driver script inside its migration transaction rather than using `strings.Split(sql, ";")`; assert both the trigger and quoted value work.

- [ ] **Step 6: Run focused and complete DB tests**

Run: `go test ./internal/db -v`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/db/db.go internal/db/db_test.go internal/db/integrity.go internal/db/integrity_test.go
git commit -m "fix: enforce sqlite integrity on every connection"
```

### Task 6: Repair relationships, quarantine orphans, and remove the template ruleset

**Files:**
- Create: `internal/db/migrations/056_integrity_repair.sql`
- Create: `internal/db/integrity_migration_test.go`
- Create: `docs/ruleset-template.sql`
- Modify: `internal/db/queries_core.go`
- Modify: `internal/db/queries_session.go`
- Modify: `internal/db/queries_adventures.go`
- Modify: deletion tests under `internal/db/`.

**Interfaces:**
- Produces: `orphaned_records(source_table, source_id, payload_json, reason, quarantined_at)` and explicit cascade/set-null/restrict behavior for every relationship.

- [ ] **Step 1: Write a schema/deletion matrix test that fails on every orphan**

Seed one campaign with character, session, messages, objectives, items, combat, XP, notes, maps/pins/tokens/zones, relationships, factions, adventures, NPC stats, secrets, calendar events, macros, decks/draws, and handouts. Delete campaign and assert every `CASCADE` child is gone, every `SET NULL` reference is null, restricted records block deletion, and `foreign_key_check` is empty.

- [ ] **Step 2: Run and verify RED**

Run: `go test ./internal/db -run 'DeleteGraph|ForeignKeySchema|TemplateRuleset' -v`

Expected: orphan counts remain and `my_ruleset` exists.

- [ ] **Step 3: Inventory actual foreign-key clauses before writing migration 056**

Run: `sqlite3 /tmp/inkandbone-schema.db '.schema'` after opening it through a small existing test helper, and record each child-parent relationship in the test table. Assign `CASCADE`, `SET NULL`, or `RESTRICT` according to the approved design; do not infer behavior from table names alone.

- [ ] **Step 4: Implement the incremental repair migration**

Create `orphaned_records`; insert each orphan as `json_object(...)`; remove quarantined child rows; rebuild only tables whose existing foreign keys have missing or incorrect actions; delete only the placeholder row where `name = 'my_ruleset'`; and leave real user-created rulesets intact. Rewrite stored generated-map message URLs from `/api/files/<map.image_path>` to `/api/assets/maps/<map.id>` so existing campaign history remains renderable. After verifying the rewrite, remove the temporary database-backed legacy-map handler and assert every `/api/files/` request returns 404.

Recreate existing indexes and add measured composite indexes for the actual high-frequency query shapes, including messages by session/order, sessions by campaign, characters by campaign, objectives by campaign/status, maps by campaign, map children by map, and combatants by encounter/order. Add `EXPLAIN QUERY PLAN` tests that assert those reads use the intended indexes rather than full table scans.

- [ ] **Step 5: Move the ruleset template out of embedded migrations**

Place the documented sample at `docs/ruleset-template.sql`. Leave `038_template_new_ruleset.sql` untouched as immutable migration history. Migration 056 removes the accidental seeded row, and the new documentation file is the only template contributors should copy going forward.

- [ ] **Step 6: Simplify deletion methods to trust declared relationship behavior**

Use a transaction for each parent deletion and delete the parent after any required domain-specific `RESTRICT` checks. Remove incomplete manually maintained delete lists only when the matrix test proves equivalent behavior.

- [ ] **Step 7: Verify migration from both clean and orphaned fixtures**

Run: `go test ./internal/db -run 'Migration|DeleteGraph|TemplateRuleset|ForeignKey' -v && go test ./internal/api -run 'DeleteCampaign|DeleteCharacter|DeleteSession' -v`

Expected: PASS; `foreign_key_check` returns no rows; quarantined fixture records retain JSON payloads.

- [ ] **Step 8: Commit**

```bash
git add internal/db docs/ruleset-template.sql
git commit -m "fix: repair relational integrity and placeholder data"
```

### Task 7: Standardize request limits and opaque API errors

**Files:**
- Modify: `internal/api/middleware.go`
- Create: `internal/api/middleware_test.go`
- Modify: `internal/api/server.go`
- Mechanically update handlers under `internal/api/routes*.go` to use the helpers.

**Interfaces:**
- Produces: `decodeJSON(w, r, dst, maxBytes) error`, `requestID(r) string`, `serverError(w, r, err)`, and middleware security headers.

- [ ] **Step 1: Write failing transport tests**

Cover oversized JSON (`413`), wrong content type (`415`), unknown field (`400`), trailing JSON (`400`), internal error response containing only `error` and `request_id`, request ID propagation, and security headers.

- [ ] **Step 2: Run and verify RED**

Run: `go test ./internal/api -run 'Middleware|DecodeJSON|OpaqueError|SecurityHeaders' -v`

- [ ] **Step 3: Implement strict decoding and opaque errors**

```go
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any, max int64) error {
	if ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); ct != "application/json" { return errUnsupportedMediaType }
	r.Body = http.MaxBytesReader(w, r.Body, max)
	dec := json.NewDecoder(r.Body); dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil { return err }
	if err := dec.Decode(&struct{}{}); err != io.EOF { return errTrailingJSON }
	return nil
}
```

Generate request IDs with `crypto/rand`, place them in context and `X-Request-ID`, log detailed errors with the ID, and return `{"error":"internal server error","request_id":"..."}`.

- [ ] **Step 4: Apply explicit limits by request family**

Use 64 KiB for ordinary JSON, 4 KiB for short AI hints, 10 MiB for images including multipart overhead, and 50 MiB for rulebooks. Preserve stricter existing limits.

- [ ] **Step 5: Verify no raw internal errors remain**

Run: `rg -n 'http\.Error\(w, err\.Error\(\)' internal/api`.

Expected: no matches.

Run: `go test ./internal/api ./internal/db && go vet ./...`.

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/api
git commit -m "fix: harden api request and error boundaries"
```

### Task 8: Phase 1 end-to-end verification

**Files:**
- Create: `e2e/tests/security.spec.ts`
- Modify: `README.md` security/data-flow sections only.

**Interfaces:**
- Consumes: built binary, disposable database, test TLS certificate generated under the test output directory.

- [ ] **Step 1: Add browser/API security scenarios**

Test loopback startup without login, failed public startup without secret/TLS/origin, authenticated TLS login, CSRF rejection, WebSocket origin rejection, typed asset success, database download failure, and whisper sentinel privacy.

- [ ] **Step 2: Run the phase verification suite**

Run:

```bash
gofmt -w cmd internal
go test ./...
go test -race ./internal/db ./internal/api
go vet ./...
make build
cd web && npm test -- --run && npm run lint
cd ../e2e && npm test
```

Expected: every command exits 0; browser security tests pass with a disposable database.

- [ ] **Step 3: Commit**

```bash
git add e2e/tests/security.spec.ts README.md
git commit -m "test: verify phase one security boundaries"
```
