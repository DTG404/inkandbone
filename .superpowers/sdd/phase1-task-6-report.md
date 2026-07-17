# Phase 1 Task 6 Implementation Report

## Status

DONE

## Commit scope

- Added incremental migration `056_integrity_repair.sql`; historical migrations, including `038_template_new_ruleset.sql`, remain unchanged.
- Added the canonical contributor template at `docs/ruleset-template.sql` and routed `AGENTS.md` to it.
- Declared all 34 SQLite foreign-key delete actions: ownership edges cascade, optional independent references set null, and campaigns restrict ruleset deletion.
- Added backup-guarded parent-table rebuild support that disables FK enforcement only for the exact registered rebuild migration, on the held exclusive connection, around one transaction. Enforcement is verified off before execution, restored and verified on after success/script failure/ledger failure, and final `foreign_key_check` remains inside the exclusive startup boundary.
- Added top-down orphan quarantine for every historical FK edge. Required orphans are removed after full-row JSON capture; optional-reference orphans are captured and preserved with the reference cleared. Every source column is represented, and rulebook embedding BLOBs are encoded as uppercase hex with an explicit `hex` encoding marker.
- Added trigger-backed integrity for the polymorphic `map_tokens.entity_type/entity_id` reference: invalid insert/update rejection and delete cascades for character and session-NPC targets. Historical missing targets are quarantined.
- Rewrote stored generated-map message links to typed map asset IDs before removing the legacy route. Every `/api/files` request now returns 404 before ServeMux canonical redirects.
- Removed the temporary legacy map lookup/handler and its data-layer query.
- Removed only an unreferenced `my_ruleset` seed. A referenced historical row is preserved to prevent campaign loss; similarly named real custom rulesets are untouched.
- Added measured query indexes and `EXPLAIN QUERY PLAN` proofs. The existing map-token unique index is reused rather than duplicated.
- Simplified campaign, character, session, adventure, and objective deletion to trust the tested relationship graph. Objective cascade covers arbitrary nesting.

## TDD evidence

### RED

Command:

`GOCACHE=/tmp/inkandbone-task6-gocache go test ./internal/db -run 'DeleteGraph|ForeignKeySchema|TemplateRuleset|IntegrityMigration|IntegrityIndexes|RepairMigrationTemporarily' -v`

Observed failures before implementation:

- 18 relationships reported `NO ACTION` instead of the required explicit action.
- Character deletion failed with `FOREIGN KEY constraint failed`.
- `my_ruleset` remained seeded and `docs/ruleset-template.sql` did not exist.
- Three representative orphan fixtures survived and caused final `foreign_key_check` failure.
- All nine query-plan cases scanned or sorted without the intended index.
- The representative parent-table rebuild failed because foreign keys were still enabled.

Additional RED after the recovery matrix was expanded:

- Full recovery payload tests showed missing chronicle/calendar/GM fields and missing rulebook embedding encoding.
- Referenced `my_ruleset` was initially removed, proving the data-loss edge.
- Polymorphic token tests accepted missing character/NPC targets and left tokens after direct parent deletion.
- Expanded historical orphan fixtures had no quarantine rows for missing polymorphic targets.

### GREEN

Focused integrity command:

`GOCACHE=/tmp/inkandbone-task6-gocache go test ./internal/db -run 'MapTokenPolymorphic|IntegrityMigrationQuarantinesOrphans|DeleteObjectiveCascadesNested|DeleteGraph|DeleteCampaignCascadesComplete' -v -count=1 -timeout=30s`

Result: PASS, 5 top-level tests, 0 failures, 0.262s.

Broader migration/deletion command:

`GOCACHE=/tmp/inkandbone-task6-gocache go test ./internal/db -run 'Migration|DeleteGraph|DeleteCampaignCascades|MapToken|TemplateRuleset|ForeignKey|Objective' -v -count=1 -timeout=60s`

Result after the legacy token fixture was corrected to use a real character: PASS in the subsequent full DB run.

## Verification evidence

- `go test ./internal/db -count=1 -timeout=90s` — PASS, 0 failures, 4.677s.
- `go test -race ./internal/db -run 'DeleteGraph|MapTokenPolymorphic|IntegrityMigrationQuarantinesOrphans|RepairMigration' -v -count=1 -timeout=60s` — PASS, 0 failures, 5.651s.
- `go test ./internal/api -run 'AssetLegacyFileSurface|ServeFile|DeleteCampaign|DeleteCharacter|DeleteSession' -v -count=1 -timeout=60s` — PASS, 9 top-level tests, 0 failures, 0.445s.
- `go vet ./...` — PASS before the later test-only polymorphic fixture additions; no production Go changed afterward except the tested objective simplification and migration SQL.
- `GOOS=windows GOARCH=amd64 go test -c ./internal/db` — PASS; Windows test binary produced.
- `go build -o /tmp/inkandbone-task6-bin ./cmd/ttrpg` — PASS; binary produced.
- `npm test -- --run` — PASS, 16 files and 144 tests.
- `npm run lint` — PASS.
- `npm run build` — PASS, TypeScript and Vite production build completed.
- `git diff --check` — PASS.
- Source search after removal found no runtime `/api/files/` reference outside migration/test fixtures.

## Environment-limited checks

- A full sandboxed API/all-package run reaches existing `httptest` socket tests and panics because the sandbox cannot open loopback listeners. The bounded affected API suite above is green; controller verification should run the unrestricted full suite.
- A later redundant post-frontend Go rebuild attempt failed because `/tmp` had only 170 MB free (`no space left on device`). This does not replace the earlier successful Go build evidence; controller verification should use a fresh cache location with available space.
- `golangci-lint` could not initialize in the restricted environment because Go attempted to write its module stat cache under the read-only module cache. Frontend ESLint and Go vet were green; controller verification should rerun golangci-lint with its established unrestricted/cache setup.

## Self-review

- Confirmed migration `038_template_new_ruleset.sql` is unmodified.
- Confirmed parent rebuilds use create/copy/drop/rename (not rename-old), avoiding SQLite rewriting child references to a temporary parent name.
- Confirmed FK restoration is asserted after successful repair, script failure, and migration-ledger failure.
- Confirmed campaign cascade and direct optional-parent deletion use separate fixtures.
- Confirmed all required and optional historical relationship classes have quarantine fixtures, including both deck-draw parents and both polymorphic token targets.
- Confirmed recovery payload assertions include late-added campaign/session/combat/calendar fields and deterministic BLOB encoding.
