# ink & bone

A private, local AI Game Master for 13 tabletop RPG systems. Single-binary Go server (HTTP, WebSocket, SQLite) with embedded React/TypeScript frontend. Streams GM responses via SSE. Automation goroutines handle NPC extraction, map generation, stat updates, recap regeneration, objective detection, and item tracking.

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
  api/                - HTTP handlers, WebSocket hub, event bus, automation goroutines
  db/                 - SQLite layer, migrations
  ai/                 - AI client implementations (DeepSeek, Anthropic Claude, Ollama, Hybrid, Dual), system prompt injection, SSE streaming
  mcp/                - MCP server for AI coding assistant integration (optional)
  utils/              - Dice roller, validation, rulebook parsing
web/                  - React frontend (src/components, src/pages, src/hooks)
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

- `campaigns` (ruleset reference, `chronicle_night` INTEGER DEFAULT 1 for VtM night tracking)
- `rulesets` (name, schema as JSON, `gm_context` TEXT per-system narrative guidance — 13 rulesets)
- `characters` (stats as JSON, portrait path, `currency_balance`, `currency_label`)
- `sessions` (title, date, summary, `tension_level` 1-10, `masquerade_integrity`)
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

## Routes

**Read:** campaigns, characters, sessions, messages, timeline, world-notes, dice-rolls, maps, map-pins, NPCs, objectives, items, tension, relationships
**Write:** messages, GM-stream, dice-rolls, world-notes, maps, objectives, items, improvise, pre-session-brief, detect-threads, campaign-ask, oracle-roll, relationships
**Patches:** campaign, session, character, world-note, objective, item, combatant, tension, relationship, personality, masquerade-integrity
**Delete:** relationships
**WebSocket:** `/ws` — live dashboard updates

## Automation Goroutines

All fire after every GM response via `handleGMRespondStream`:

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

## Key Implementation Details

- **System Prompt Injection:** Character name, active objectives, NPC personality cards, and rulebook chunks injected per turn
- **GM Response Length:** Hard-enforced at 4-5 paragraphs via system prompt + per-ruleset directives + [REMINDER] block
- **Em-dash Strip:** All output em-dashes programmatically stripped before display
- **SSE Streaming:** GM responses stream character-by-character
- **WebSocket Hub:** Event bus broadcasts state changes in real time
- **Rulebook Chunks:** PDF/text upload for rules lookup during play
- **MCP Layer:** Optional MCP stdio server for AI coding assistant integration (tools: get_context, set_active, start/end session, create/list characters, roll dice, combat management, world notes, maps, rulebook search, session recap)
- **30 database migrations** covering core schema through VtM V5 features

## Notes for Contributors

1. **All routes tested?** Check both request/response and database mutations.
2. **Automation goroutine?** Handle nil AI client gracefully and log errors.
3. **New feature = new migration.** Don't modify existing migrations; add incrementals.
4. **Frontend updates?** Hot reload via Vite during `make dev`; rebuild with `make build`.
5. **Ruleset additions?** Add JSON schema to `rulesets` table; test with character creation.
6. **AI thinking mode:** DeepSeek defaults to thinking mode. The DeepSeek client explicitly disables it with `"thinking":{"type":"disabled"}` on every request. If switching models, check for equivalent defaults.
