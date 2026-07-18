# End-to-end coverage matrix

The maintained browser tree is `e2e/tests/`; `cd e2e && npm test` is its only supported command. API-only behavior belongs in Go route/database tests, while browser tests cover user-visible workflows. The inventory below was completed before deleting the former root Playwright scripts.

## Former comprehensive script

| Old assertion/scenario | Maintained coverage | Status and rationale |
|---|---|---|
| 1.1-1.4 campaign, character, stats, session seed | `internal/api/routes_write_test.go`; all maintained specs seed through HTTP | Covered; API creation is deterministic and reused by browser prerequisites. |
| 2.1-2.9 header/navigation controls | `smoke.spec.ts` app load; `workspace-workflows.spec.ts` overlay/export workflows | Covered with semantic names rather than icon/title selectors. |
| 3.1-3.14 character attributes, tracks, inventory, currency, XP entries | `CharacterSheetPanel.test.tsx`; `smoke.spec.ts`; `workspace-workflows.spec.ts` XP workflow | Covered at component and user-workflow levels. |
| 3b.1-3b.2 XP add and amount | `workspace-workflows.spec.ts` XP add/delete | Covered through the visible form and persisted refresh. |
| 4.1-4.5 session title, messages, composer, search, scene tags | `smoke.spec.ts`; `workspace-workflows.spec.ts` story search | Covered with accessible selectors. |
| 5.1 map drawer | `responsive.spec.ts` visual/layout checks | Obsolete as an assertion: the old script clicked open/close and then called `assert(true)` without inspecting state. Current viewport screenshots and overflow checks cover the rendered drawer boundary. |
| 6.1-6.12 legacy sidebar tabs | `smoke.spec.ts` right-panel destinations; `responsive.spec.ts` destination availability | Covered by the current Play/World/GM navigation model; the old twelve-tab layout is obsolete. |
| 7 Notes panel | `smoke.spec.ts` Notes navigation and search | Covered. |
| 8 NPCs panel | `smoke.spec.ts` NPC creation | Covered through API setup plus visible roster. |
| 9 Objectives panel | `smoke.spec.ts` navigation; `routes_write_test.go` objective CRUD | Covered; CRUD semantics are stronger in route tests. |
| 10 Oracle panel | `smoke.spec.ts` navigation; `routes_write_test.go` oracle success/validation | Covered. |
| 11 Relationships panel | `smoke.spec.ts` navigation; `routes_write_test.go` relationship CRUD | Covered. |
| 12 Factions panel | `KeyboardRows.test.tsx`; `routes_factions_test.go` | Covered at keyboard/UI row and full API contract levels. |
| 13 Adventures panel | `KeyboardRows.test.tsx`; `routes_adventures_test.go` | Covered at keyboard/UI row and full API contract levels. |
| 14 NPC Stats panel | `KeyboardRows.test.tsx`; `routes_npc_stats_test.go` | Covered at keyboard/UI row and full API contract levels. |
| 15 Secrets panel | `smoke.spec.ts` reveal/push handout; `routes_secrets_test.go` | Covered. |
| 16 Calendar panel | `KeyboardRows.test.tsx`; `queries_calendar_test.go` | Covered; calendar persistence and deletion are API/database behavior. |
| 17 Journal panel | `workspace-workflows.spec.ts` Journal/Timeline; `JournalPanel.test.tsx` | Covered. |
| 18 GM Tools panel | `GMToolsPanel.test.tsx`; `smoke.spec.ts` navigation registry | Covered. |
| 19.1-19.4 Journal Notes/Timeline/Reanalyze | `workspace-workflows.spec.ts`; `JournalPanel.test.tsx` | Covered; reanalysis failure/success status remains unit-tested to avoid paid AI calls. |
| 20.1-20.4 GM Screen overlay | `workspace-workflows.spec.ts`; `GMScreenPanel.test.tsx` | Covered including dialog semantics. |
| 21.1-21.2 player history | `workspace-workflows.spec.ts` | Covered with visible dialog and seeded action. |
| 22.1 talents overlay | `workspace-workflows.spec.ts`; `XPSuggestionsPanel.test.tsx` | Covered; AI talent generation remains API/unit scoped. |
| 23.1-23.2 theme toggle | `smoke.spec.ts`; four `responsive.spec.ts` visual baselines | Covered. |
| 24.1 audio control | `accessibility.spec.ts` axe scans; `responsive.spec.ts` header viewport checks | Covered for presence/accessibility; actual sound output is not deterministic browser-test behavior. |
| 25 story search | `workspace-workflows.spec.ts` | Covered including exclusion of a non-match. |
| 26 export | `workspace-workflows.spec.ts` | Covered by an actual download and markdown-content assertions. |
| 27 whisper toggle | `workspace-workflows.spec.ts`; `security.spec.ts` | Covered through visible composition, stored flag, provider exclusion, and export exclusion. |
| 28 scene tags | `routes_test.go` scene-tag patch; `App.test.tsx` session rendering | Covered. |
| 29.1-29.6 Manage tabs | `smoke.spec.ts`; `workspace-workflows.spec.ts`; `ManagePanel.test.tsx` | Covered. |
| 30.1-30.5 automation settings/toggle | `workspace-workflows.spec.ts`; `AutomationSettingsPanel.test.tsx`; `automation_dispatcher_test.go` | Covered without triggering provider work. |
| 31.1 character numeric edit | `CharacterSheetPanel.test.tsx`; `routes_character_test.go` | Covered; component test precisely verifies edit semantics. |
| 32.1 inventory add | route/database item tests | Obsolete as a browser assertion: the old script called `assert(true)` after an optional click and never proved item creation. Item mutation remains covered at its authoritative API/database boundary. |
| 33.1-33.4 campaign config read/update | `routes_campaign_config_test.go`; `GMScreenPanel.test.tsx` | Covered. |
| 34.1-34.3 secrets list/reveal/delete | `routes_secrets_test.go`; `smoke.spec.ts` handout | Covered. |
| 35.1-35.4 faction list/update | `routes_factions_test.go`; `queries_factions_test.go` | Covered. |
| 36.1-36.2 adventure list/update | `routes_adventures_test.go`; `queries_adventures_test.go` | Covered. |
| 37.1-37.3 NPC-stat list/update/delete | `routes_npc_stats_test.go` | Covered. |
| 38.1-38.4 calendar read/advance/event delete | `queries_calendar_test.go`; route tests | Covered. |
| 39.1-39.5 automation setting schema/toggle | `AutomationSettingsPanel.test.tsx`; `automation_dispatcher_test.go` | Covered. |
| 40.1-40.2 world-note list/search | `routes_test.go` world-note filters | Covered. |
| 41.1-41.2 objective list/update | `routes_write_test.go`; `queries_objectives_test.go` | Covered. |
| 42.1 session notes update | `routes_test.go::TestPatchSession_ok`; `queries_session_test.go` | Covered. |
| 43.1-43.2 tension read/update | focused tension route/database tests | Covered. |
| 44.1 oracle roll | `routes_write_test.go::TestOracleRoll`; `queries_oracle_test.go` | Covered. |
| 45.1 dice list | `routes_test.go::TestListDiceRolls_empty`; `queries_world_test.go::TestDiceRolls`; `smoke.spec.ts` live feed | Covered. |
| 46.1 timeline list | `routes_test.go::TestGetTimeline_withData`; `queries_timeline_test.go` | Covered. |
| 47.1-47.4 relationship CRUD | `routes_write_test.go`; `queries_relationships_test.go` | Covered. |
| 48.1 chronicle night | VtM route/smoke tests | Covered in the ruleset where the field is valid. |
| 49.1 health | `routes_write_test.go::TestHealthEndpoint`; every lifecycle readiness probe | Covered. |
| 50.1-50.3 active context | `routes_test.go::TestGetContext_withActiveState`; all maintained browser setup | Covered. |
| 51.1 character list | `routes_test.go::TestListCharacters_withData` | Covered. |
| 52.1 session list | `routes_test.go::TestListSessions_withData` | Covered. |
| 53.1-53.2 messages list/roles | route message tests; `smoke.spec.ts`; `workspace-workflows.spec.ts` | Covered. |
| 54.1-54.2 XP list/note | `routes_phase_a_test.go`; `workspace-workflows.spec.ts` | Covered. |
| 55.1-55.3 ruleset list/get | ruleset route tests and every maintained browser seed | Covered. |
| 56.1 character options | `routes_write_test.go::TestGetCharacterOptions` | Covered. |
| 57.1 campaign list | `routes_test.go::TestListCampaigns_withData` | Covered. |
| 58.1 talent description success-or-503 | talent route tests | Covered; the old assertion accepted either result and added no deterministic browser guarantee. |
| 59.1 maps list | map route tests; `security.spec.ts` typed map asset | Covered. |
| 60.1 path traversal | `security.spec.ts`; asset route security tests | Covered with stronger 404/typed-route expectations than the old “any error” assertion. |
| 61.1-61.2 rulebook sources/ingest | `routes_write_test.go` rulebook tests; DB search tests | Covered. |
| 62.1 invalid dice | `routes_write_test.go::TestRollDice_invalidExpression`; `smoke.spec.ts` valid dice set | Covered. |
| 63.1 invalid oracle | `routes_write_test.go::TestOracleRoll_invalidTable` | Covered. |
| 64.1-64.2 campaign deletion | `routes_write_test.go::TestDeleteCampaign`; `integrity_migration_test.go` complete graph cascade | Covered with explicit referential-integrity checks. |

## Other former root scripts

| Old script/scenario | Maintained coverage | Status and rationale |
|---|---|---|
| `group-a-interactivity-e2e.mjs` A.1-A.10 click-to-roll | `workspace-workflows.spec.ts` click-to-roll; `CharacterSheetPanel.test.tsx` schema rendering | Covered. The maintained test asserts the persisted action shape and hint state without a fixed browser binary path. |
| `group-a-interactivity-e2e.mjs` B.1-B.12 macros | `workspace-workflows.spec.ts` macro workflow; `queries_macros_test.go` | Covered across visible fire/reorder/delete/cap behavior and database ordering. |
| `group-a-interactivity-e2e.mjs` C.1-C.10 combat order | `CombatPanel.test.tsx`; `UserActionFeedback.test.tsx`; `queries_combat_test.go` | Covered. The former script inserted directly through `node:sqlite`, tied itself to `/tmp/e2e-group-a.db`, and bypassed the API prerequisite rule; deterministic component/database characterization replaces it. |
| `multiplayer-e2e.mjs` seed/context/message ownership | context/message route tests; `App.test.tsx` character selection | Covered. |
| `multiplayer-e2e.mjs` ten live GM turns and name mentions | none | Obsolete: model wording is nondeterministic, requires external provider credentials, and can incur paid requests; it is not a reliable product contract. Prompt/context inclusion is verified by deterministic backend tests. |
| `multiplayer-e2e.mjs` browser story/selector | `App.test.tsx`; maintained smoke/session workflow specs | Covered deterministically. |
| `multiplayer-e2e-deep.mjs` forty live GM turns | none | Obsolete load/demo script: paid-provider, wording, and timing dependence make it unsuitable for CI. Dispatcher saturation/recovery is covered by `reliability.spec.ts` and Go race tests. |
| `multiplayer-e2e-deep.mjs` identity persistence and story display | route/database character tests; `App.test.tsx`; maintained smoke/session workflow specs | Covered. |

## Maintained suite boundaries

- Browser: `accessibility.spec.ts`, `reliability.spec.ts`, `responsive.spec.ts`, `security.spec.ts`, `smoke.spec.ts`, and `workspace-workflows.spec.ts`.
- Lifecycle: `e2e/lifecycle.test.mjs` proves exact process and temporary-directory cleanup on success, failure, and signals.
- Backend/API-only contracts: Go tests remain authoritative where a browser adds no user-visible assertion.
- External model quality and prose wording are deliberately outside deterministic CI.
