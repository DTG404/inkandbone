# ink & bone

A private, local AI Game Master for 14 tabletop RPG systems. Single-binary Go server (HTTP, WebSocket, SQLite) with embedded React/TypeScript frontend. Streams GM responses via SSE. Automation goroutines handle NPC extraction, map generation, stat updates, recap regeneration, objective detection, and item tracking. Schema-driven character sheets with computed fields and conditional visibility.

## Tech Stack

**Backend:**
- Go 1.22+
- SQLite (persisted to `~/.ttrpg`)
- HTTP + WebSocket + SSE
- AI clients: DeepSeek Flash (primary), Anthropic Claude Haiku (fallback), Ollama (local)

**Frontend:**
- React 18 + TypeScript
- Vite (development and bundled into binary)
- WebSocket client for live updates
- "Worn Grimoire" dark theme (parchment + gold) + light theme toggle

**Tools & Libraries:**
- Dice roller (expression parsing: `1d20`, `2d6+3`, etc.)
- SVG map generator (AI-generated location maps)
- Ruleset schema validator (character sheet generation)
- Rulebook indexing (PDF/text upload for rules lookup)

## Project Structure

```
cmd/ttrpg/            - Binary entrypoint
internal/
  api/                - HTTP handlers (decomposed: routes, automations, vtm, factions, etc.), WebSocket hub, event bus, validation middleware
  db/                 - SQLite layer, 46 migrations
  ai/                 - AI client implementations (DeepSeek, Anthropic Claude, Ollama, Hybrid, Dual), system prompt injection, SSE streaming
  mcp/                - MCP server for AI coding assistant integration (optional)
  ruleset/            - Ruleset-specific logic (advancement, random stats, character options, VtM, W&G)
  dice/               - Dice roller, expression parsing
web/                  - React frontend (SessionView, panels, hooks)
Makefile              - build, install, dev, test, clean
```

## AI Client Configuration

The binary detects and configures the AI client based on environment variables (checked in this order):

| Config | Env Vars | Behavior |
|--------|----------|----------|
| **Hybrid** | `OLLAMA_GM_MODEL` + `ANTHROPIC_API_KEY` | Ollama for GM narration, Claude Haiku for automation |
| **DeepSeek** | `DEEPSEEK_API_KEY` | DeepSeek V4 Flash for everything (default) |
| **Dual DeepSeek** | `DEEPSEEK_API_KEY` + `DEEPSEEK_AUTO_MODEL` | DeepSeek Flash for GM, DeepSeek Pro for automation |
| **OpenRouter** | `OPENROUTER_API_KEY` | Any model via OpenRouter proxy |
| **Claude** | `ANTHROPIC_API_KEY` only | Claude Haiku for everything |
| **Dual Ollama** | `OLLAMA_GM_MODEL` + `OLLAMA_AI_MODEL` | Two local models |
| **Single Ollama** | `OLLAMA_MODEL` only | Single local model |
| **Disabled** | None set | AI client is nil; endpoints fail gracefully |

Wrapper at `~/bin/ttrpg` pulls `DEEPSEEK_API_KEY` from pass store and launches the binary.

## Key Database Tables

- `campaigns` (ruleset reference, `chronicle_night` INTEGER DEFAULT 1 for VtM night tracking; `gm_notes`, `system_prompt_override` for GM screen; `in_game_*` for calendar)
- `rulesets` (name, schema as JSON, `gm_context` TEXT per-system narrative guidance — 14 rulesets)
- `characters` (stats as JSON, portrait path, `currency_balance`, `currency_label`)
- `sessions` (title, date, summary, `tension_level` 1-10, `masquerade_integrity`, `adventure_id`)
- `messages` (full conversation history)
- `session_npcs` (named NPCs per session)
- `world_notes` (tagged lore; `personality_json` for NPC traits)
- `maps` (uploaded or AI-generated SVG)
- `map_pins` (coordinate + label + notes)
- `objectives` (active/completed/failed)
- `items` (character inventory)
- `combat_encounters` + `combatants` (turn tracking; VtM includes damage and willpower tracks)
- `dice_rolls` (expression + result)
- `oracle_tables` (action/theme + VtM clan compulsion tables)
- `relationships` (from/to names, type, description)
- `scene_tags` (session scene tags: tavern, dungeon, forest, etc.)
- `factions` (campaign-level factions with influence, type, resources, color)
- `adventures` (campaign arcs/chapters; sessions linked via `adventure_id`)
- `npc_stats` (reusable NPC stat blocks: HP, AC, initiative, skills, abilities, loot)
- `secrets` (GM secrets/handouts with hidden/revealed state, linked to sessions)
- `calendar_events` (in-game calendar: year/month/day, type, linked to sessions)

## Routes

**Read:** campaigns, characters, sessions, messages, timeline, world-notes, dice-rolls, maps, map-pins, NPCs, objectives, items, tension, relationships, factions, adventures, npc-stats, secrets, calendar, calendar-events, settings/automations, campaign-config
**Write:** messages, GM-stream, dice-rolls, world-notes, maps, objectives, items, improvise, pre-session-brief, detect-threads, campaign-ask, oracle-roll, relationships, factions, adventures, npc-stats, secrets, calendar-events, settings/automations
**Patches:** campaign, session, character, world-note, objective, item, combatant, tension, relationship, personality, masquerade-integrity, campaign-config, faction, adventure, secret, npc-stat, calendar, session/adventure
**Delete:** relationships, factions, adventures, npc-stats, secrets, calendar-events
**WebSocket:** `/ws` — live dashboard updates

## Automation Goroutines

All fire after every GM response via `handleGMRespondStream`. Each can be individually toggled on/off via `PATCH /api/settings/automations` or the Automation tab in the Manage panel. AI-powered automations use retry logic (1 retry with backoff) and a circuit breaker (skips after 3 consecutive failures).

1. **extractNPCs** — AI extracts NPC names, adds/removes from roster
2. **autoGenerateMap** — Detect new location names, generate SVG
3. **autoUpdateCharacterStats** — Apply story-driven stat changes
4. **autoUpdateRecap** — Every 4 GM messages, regenerate session journal
5. **autoDetectObjectives** — Detect story goals
6. **autoExtractItems** — Parse items gained/lost
7. **checkAndExecuteRoll** — Enforce dice rolls before GM responds
8. **autoUpdateTension** — Crisis keyword scanning (zero AI cost)
9. **autoUpdateCurrency** — Parse currency transactions (zero AI cost)
10. **autoUpdateSceneTags** — Keyword-match environment terms (zero AI cost)
11. **autoUpdateMasquerade** — VtM-only: Masquerade breach scanning
12. **autoUpdateChronicleNight** — VtM-only: night counter advancement
13. **autoSuggestXPSpend** — XP advancement suggestions

## E2E Testing

A comprehensive end-to-end test suite lives at `scripts/e2e-comprehensive.mjs`. It tests every feature through both browser (Playwright) and API calls — 187 assertions covering all UI panels, CRUD endpoints, and edge cases.

```bash
# Start server with a fresh DB
ttrpg -db /tmp/e2e-test.db

# Run tests (in another terminal)
node scripts/e2e-comprehensive.mjs
# Expected: 187 passed, 0 failed
```

Requires: `npm install playwright` + `npx playwright install chromium`

## Build & Deploy

```bash
make build   # React (Vite) → Go binary (embed dist/)
make install # Binary to ~/bin/ttrpg-bin
make dev     # Hot reload: air (Go) + Vite (React) concurrently
make test    # Run all Go tests
make lint    # Go (golangci-lint) + web (ESLint)
make audit   # Go (govulncheck) + web (npm audit)
make secrets-scan  # gitleaks (skips if not installed locally)
```

Wrapper at `~/bin/ttrpg` passes AI configuration to the binary.

## CI Pipeline

GitHub Actions workflow at `.github/workflows/ci.yml`. Runs on push/PR to `main`:

1. **Lint** — `golangci-lint` (Go) + `eslint` (web)
2. **Audit** — `govulncheck` (Go) + `npm audit` (web)
3. **Test** — `go test ./... -v`
4. **Secrets scan** — `gitleaks-action`
5. **Build** — `make build`

## Schema System

Every ruleset has a `schema_json` column that defines its character sheet fields. The schema uses a structured array format:

```json
[
  {"key":"str","label":"STR","type":"number","category":"attribute","min":1,"max":30,"default":"10"},
  {"key":"class","label":"Class","type":"text","category":"identity","options":["Fighter","Mage","Rogue"],"default":"Fighter"},
  {"key":"hp","label":"HP","type":"number","category":"track","max":999},
  {"key":"notes","label":"Notes","type":"textarea","category":"notes"}
]
```

### Field Properties

| Property | Type | Purpose |
|----------|------|---------|
| `key` | string | Field identifier (lowercase_with_underscores) |
| `label` | string | Display label in the UI |
| `type` | `text` / `number` / `textarea` | Input type |
| `category` | `attribute` / `track` / `skill` / `identity` / `resource` / `notes` | Controls rendering style |
| `min` | number (optional) | Minimum value (number fields + attribute pips) |
| `max` | number (optional) | Maximum value (number fields + track bar segments + attribute pip count) |
| `options` | string[] (optional) | Renders as `<select>` dropdown instead of text input |
| `default` | string (optional) | Starting value; used by `RollStatsFromSchema` fallback |
| `computed` | string (optional) | Formula expression for derived fields (e.g. `floor((level-1)/4)+2`); field becomes read-only |
| `computed_display` | string (optional) | Display label for a computed field |
| `condition` | object (optional) | Visibility condition, e.g. `{"field":"character_type","equals":"vampire"}` |

### Category Rendering

| Category | Renders As |
|----------|-----------|
| `attribute` | Pip dots (count driven by `max`, default 5) |
| `track` | Segmented bar (segments driven by `max`, default 5) |
| `skill` | Number input in 2-column grid |
| `resource` | Number input in 2-column grid |
| `identity` | Text input (or `<select>` if `options` present) |
| `notes` | Textarea |

### Schema-Driven Character Creation

Adding a new ruleset now requires **zero Go code** if you provide `default` and `options` in the schema:

- **`default`** — `RollStats()` falls back to `RollStatsFromSchema()` which reads `default` values from the schema. Dice expressions are supported as literal strings.
- **`options`** — Fields with `options` render as `<select>` dropdowns in the character sheet and are used for random selection during auto-generation.

If you need complex stat logic (e.g. D&D's 4d6-drop-lowest, VtM's attribute pools), add a `case "myname":` to `RollStats()` in `internal/ruleset/random_stats.go` and a `case "myname":` to `CharacterOptions()` in `internal/ruleset/options.go`.

### Current Rulesets (14)

| ID | Name | Fields | Rulebooks |
|----|------|--------|-----------|
| 1 | dnd5e | 16 | Core Rulebook |
| 2 | ironsworn | 13 | — |
| 3 | vtm | 77 (V5) | Core Rulebook, Player's Guide |
| 4 | coc | 17 | — |
| 5 | cyberpunk | 16 | — |
| 6 | shadowrun | 20 | — |
| 7 | wfrp | 21 | — |
| 8 | starwars | 20 | — |
| 9 | l5r | 18 | — |
| 10 | theonering | 16 | — |
| 11 | wrath_glory | 48 | Core Rulebook |
| 12 | blades | 25 | — |
| 13 | paranoia | 15 | — |
| 15 | dune | 24 | Core Rulebook, Masters of Dune, Sand and Dust, Power and Pawns |

## Key Implementation Details

- **System Prompt Injection:** Character name, active objectives, NPC personality cards, and rulebook chunks injected per turn
- **GM Response Length:** Hard-enforced at 4-5 paragraphs via system prompt + per-ruleset directives + [REMINDER] block
- **Em-dash Strip:** All output em-dashes programmatically stripped before display
- **SSE Streaming:** GM responses stream character-by-character
- **WebSocket Hub:** Event bus broadcasts state changes in real time
- **Rulebook Chunks:** PDF/text upload for rules lookup during play. Text uploads via `curl -X POST /api/rulesets/{id}/rulebook?source=Name -H "Content-Type: text/plain" --data-binary @file.txt`. Paragraph-based chunking for VtM and Dune; markdown-heading chunking for all others.
- **MCP Layer:** Optional MCP stdio server for AI coding assistant integration (tools: get_context, set_active, start/end session, create/list characters, roll dice, combat management, world notes, maps, rulebook search, session recap)
- **46 database migrations** covering core schema through VtM V5, Dune ruleset, and schema enhancements

## Notes for Contributors

1. **All routes tested?** Check both request/response and database mutations.
2. **Automation goroutine?** Handle nil AI client gracefully and log errors.
3. **New feature = new migration.** Don't modify existing migrations; add incrementals (files are sorted alphabetically, so prefix with `NNN_`).
4. **Frontend updates?** Hot reload via Vite during `make dev`; rebuild with `make build`.
5. **Ruleset additions?** Use the template at `internal/db/migrations/038_template_new_ruleset.sql`. See **Schema System** section above for field properties. Most rulesets need zero Go code — just fill in `default` and `options` in the schema JSON.
6. **AI thinking mode:** DeepSeek defaults to thinking mode. The DeepSeek client explicitly disables it with `"thinking":{"type":"disabled"}` on every request. If switching models, check for equivalent defaults.
7. **RollStats fallback:** If you add a ruleset with only schema defaults (no Go `case`), the API automatically calls `RollStatsFromSchema()`. You only need to add Go code for complex stat generation logic.
8. **Character options in schema:** Fields with `"options":[...]` render as dropdowns in the UI. The `CharacterOptions()` Go function is still used for character-creation defaults, but the UI will show dropdowns regardless.
