# Group B Interactivity — Design Spec

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add five table-enhancing features to inkandbone: handouts panel, compendium quick-reference, card/deck system, interactive token map, and fog of war.

**Architecture:** Each feature follows the Group A pattern — Go migration + queries + routes, then React frontend. Features are independent except fog of war, which depends on the token map's map layer work being in place first.

**Tech Stack:** Go 1.22, SQLite (modernc.org/sqlite), React 18 + TypeScript, Vite, WebSocket event bus (existing Hub/Bus), Ollama (nomic-embed-text, localhost:11434) for compendium embeddings.

## Global Constraints

- All coordinates normalized 0.0–1.0 (same system as existing pins and tokens).
- WebSocket event names: snake_case strings matching existing convention (`world_note_revealed`, `card_drawn`, `token_placed`, `token_moved`, `token_removed`, `zone_revealed`).
- No AI key required for handouts, card/deck, or token map. Compendium embedding requires Ollama running locally. Fog of war zone auto-reveal runs in the existing automation goroutine.
- SQLite migrations follow the existing numbered filename pattern (`050_`, `051_`, etc. — check the highest existing number before naming).
- No hand management for cards, no player-held cards, no per-player visibility for handouts.
- Fog of war only functions on maps that have zones defined; maps without zones show no fog overlay.
- Desktop-only: no touch event handling needed.
- YAGNI: no features beyond what is specified here.

---

## Feature 1: Handouts Panel

### Overview

World notes already exist per campaign. This feature adds a `is_revealed` flag so the GM can mark specific notes as player-visible. A new `HandoutsPanel` tab shows revealed notes read-only. The `WorldNotesPanel` gets a Reveal toggle and a visual indicator on already-revealed notes.

### DB

Migration: `ALTER TABLE world_notes ADD COLUMN is_revealed INTEGER NOT NULL DEFAULT 0`

No new table. Revealed state is campaign-level (persists across sessions).

### Backend

**Queries (`internal/db/queries_world.go`):**
- `PatchWorldNoteRevealed(id int64, revealed bool) error` — UPDATE world_notes SET is_revealed = ? WHERE id = ?
- `ListWorldNotes` gains an optional `revealed *bool` filter: when non-nil, adds `AND is_revealed = ?` to the query

**Routes:**
- `PATCH /api/world-notes/{id}/reveal` — body `{"is_revealed": true|false}`, calls `PatchWorldNoteRevealed`, broadcasts `world_note_revealed` WS event with `{id, is_revealed}`
- `GET /api/campaigns/{id}/world-notes?revealed=true` — existing handler extended to pass revealed filter

**WebSocket event:** `world_note_revealed` → `{id: int, is_revealed: bool}`

### Frontend

**`WorldNotesPanel.tsx`:**
- Add `is_revealed: boolean` to the `WorldNote` TypeScript type
- Each note card gets a Reveal toggle button (📤 unrevealed / ✅ revealed) in the card header
- Already-revealed notes show a muted `"Revealed"` badge below the title so the GM can track what the player sees at a glance
- Clicking the toggle calls `PATCH /api/world-notes/{id}/reveal`
- Subscribe to `world_note_revealed` WS event to update local state

**`HandoutsPanel.tsx` (new file):**
- Fetches `GET /api/campaigns/{id}/world-notes?revealed=true`
- Renders notes read-only: title, category badge, tags, content (markdown rendered). No edit controls, no personality editor, no draft button.
- Re-fetches on `world_note_revealed` WS event
- Empty state: `"No handouts revealed yet."`

**`SessionView.tsx`:**
- Add `handouts` tab as the first tab in the right sidebar (before `notes`)
- Render `<HandoutsPanel campaignId={ctx.campaign.id} />`

---

## Feature 2: Compendium Quick-Reference

### Overview

Rulebook chunks are already ingested and stored in SQLite. This feature adds semantic search via Ollama (nomic-embed-text) so players can query the rulebook inline during play. Embeddings are stored as BLOBs on `rulebook_chunks` and generated at ingest time. A keyword fallback activates when Ollama is unavailable.

### DB

Migration: `ALTER TABLE rulebook_chunks ADD COLUMN embedding BLOB`

No new table. Embeddings are 768-dimensional float32 vectors stored as raw binary (768 × 4 = 3072 bytes per chunk).

### Backend

**Ollama client (`internal/ai/ollama.go`, new file):**
```go
func EmbedText(ctx context.Context, text string) ([]float32, error)
// POST http://localhost:11434/api/embeddings {"model":"nomic-embed-text","prompt":text}
// Returns []float32 decoded from response embedding array.
// Returns error if Ollama is unreachable.
```

**Cosine similarity utility (`internal/ai/cosine.go`, new file):**
```go
func CosineSimilarity(a, b []float32) float32
func TopK(chunks []db.RulebookChunk, queryEmb []float32, k int) []db.RulebookChunk
// Brute-force: iterate all chunks with embeddings, rank by cosine similarity, return top k.
```

**DB queries (`internal/db/queries_rulebook.go`, modify):**
- `UpsertChunkEmbedding(id int64, emb []float32) error` — encode []float32 to []byte, UPDATE rulebook_chunks SET embedding = ? WHERE id = ?
- `ListChunksForEmbedding(rulesetID int64) ([]RulebookChunk, error)` — SELECT id, content WHERE embedding IS NULL
- `ListAllChunks(rulesetID int64) ([]RulebookChunk, error)` — SELECT id, heading, content, source, embedding for cosine scan

**Embedding at ingest (`internal/api/routes_rulebook.go`):**
- After `CreateRulebookChunks`, iterate chunks and call `EmbedText` for each, storing via `UpsertChunkEmbedding`. Ingest continues even if Ollama fails (chunks stored without embedding).

**Startup goroutine (`internal/api/server.go`):**
- On `NewServer`, launch a goroutine that calls `ListChunksForEmbedding` for all rulesets and embeds any missing embeddings. Runs once on startup; errors are logged but do not crash the server.

**Search endpoint:**
- `POST /api/rulesets/{id}/rulebook/search` — body `{"query": string}`
- Tries semantic: embed query via Ollama, load all chunks for ruleset, run `TopK(chunks, queryEmb, 8)`
- Falls back to keyword on Ollama error: `SELECT * FROM rulebook_chunks WHERE ruleset_id = ? AND (heading LIKE ? OR content LIKE ?) LIMIT 8`
- Response: `{"results": [{heading, content, source}], "mode": "semantic"|"keyword"}`

### Frontend

**`api.ts`:** Add `searchRulebook(rulesetId: number, query: string): Promise<{results: RulebookResult[], mode: string}>`

**`CompendiumPanel.tsx` (new file):**
- Props: `rulesetId: number`
- Search input with 400ms debounce + Enter key submit
- Panel opens blank — no results until user types
- Results list: up to 8 items showing heading (bold) + content excerpt (first 200 chars) + source filename (muted)
- When `mode === "keyword"`, show subtle banner: `"Semantic search unavailable — showing keyword results"`
- Loading spinner during search; `"No results found"` empty state

**`SessionView.tsx`:**
- Add `compendium` tab to right sidebar
- Render `<CompendiumPanel rulesetId={ctx.campaign.ruleset_id} />`

---

## Feature 3: Card/Deck System

### Overview

A flip/reveal card mechanic for oracle decks, WFRP crit tables, and GM inspiration decks. Decks are defined per campaign (named list of cards with front + optional back text). The GM shuffles and draws; drawn cards are broadcast to all connected windows. Draw history is recorded per session.

### DB

Migration (two tables):

```sql
CREATE TABLE decks (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  campaign_id     INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  name            TEXT NOT NULL,
  cards_json      TEXT NOT NULL DEFAULT '[]',
  shuffled_order_json TEXT NOT NULL DEFAULT '[]',
  draw_index      INTEGER NOT NULL DEFAULT 0,
  created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE deck_draws (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id  INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  deck_id     INTEGER NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
  card_json   TEXT NOT NULL,
  drawn_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

`cards_json`: `[{"front": "...", "back": "..."}]` — `back` is optional.
`shuffled_order_json`: `[2, 0, 4, 1, 3]` — indices into cards_json representing the current shuffle order.

### Backend

**Queries (`internal/db/queries_decks.go`, new file):**
- `ListDecks(campaignID int64) ([]Deck, error)`
- `CreateDeck(campaignID int64, name string, cardsJSON string) (int64, error)`
- `DeleteDeck(id int64) error`
- `ShuffleDeck(id int64, shuffledOrderJSON string) error` — UPDATE shuffled_order_json + reset draw_index = 0
- `DrawCard(id int64, newDrawIndex int, sessionID int64, cardJSON string) error` — transaction: UPDATE draw_index, INSERT deck_draws
- `GetDeck(id int64) (*Deck, error)`
- `ListDeckDraws(sessionID int64) ([]DeckDraw, error)`

**Routes (`internal/api/routes_decks.go`, new file):**
- `GET /api/campaigns/{id}/decks` → ListDecks
- `POST /api/campaigns/{id}/decks` — body `{name, cards: [{front, back?}]}`, validates len(cards) ≥ 1 and len(cards) ≤ 200 (returns 400 if exceeded)
- `DELETE /api/decks/{id}`
- `POST /api/decks/{id}/shuffle` — generates a random permutation of [0..n-1] using Go `math/rand`, calls ShuffleDeck
- `POST /api/decks/{id}/draw` — body `{session_id: int}`. Reads deck, checks draw_index < len(shuffled_order). Returns card at shuffled_order[draw_index], increments draw_index, records to deck_draws, broadcasts `card_drawn` WS event. If draw_index == len(shuffled_order), returns `{"exhausted": true}` without drawing.
- `GET /api/sessions/{id}/deck-draws` → ListDeckDraws for session

**WebSocket event:** `card_drawn` → `{deck_id, deck_name, card: {front, back?}, draw_index, total}`

### Frontend

**Types (`web/src/types.ts`):**
```typescript
interface Deck { id: number; campaign_id: number; name: string; cards: DeckCard[]; draw_index: number; total: number }
interface DeckCard { front: string; back?: string }
interface DeckDraw { id: number; deck_id: number; card: DeckCard; drawn_at: string }
```

**`api.ts`:** Add `listDecks`, `createDeck`, `deleteDeck`, `shuffleDeck`, `drawCard`, `listDeckDraws`.

**`DecksPanel.tsx` (new file):**
- Two modes toggled by a ⚙ gear button: **Browse** and **Edit**.

**Browse mode:**
- Lists all campaign decks. Each deck shows name, card count, draw progress (`3 / 12`), and two buttons: `Shuffle ↺` and `Draw ▶`.
- Last drawn card displayed prominently below the deck list: card front text large, back text (if any) smaller below. Deck name shown as subtitle.
- When deck is exhausted, Draw button is disabled and shows `"Deck empty — reshuffle"`.
- Last 5 draws for the session shown as a compact history list below the drawn card.
- WS `card_drawn` event updates the displayed card and history in all windows.

**Edit mode:**
- Create deck form: name input + dynamic card list (rows with front/back text inputs, `+` to add row, `×` to remove). Submit creates the deck. Max 200 cards enforced client-side.
- Existing decks listed with a `×` delete button (confirm dialog: "Delete [name]?").

**`SessionView.tsx`:** Add `decks` tab to right sidebar, render `<DecksPanel campaignId={ctx.campaign.id} sessionId={ctx.session.id} />`.

---

## Feature 4: Interactive Token Map

### Overview

Tokens representing campaign characters and NPCs can be dragged onto any map and repositioned. Token positions persist per map. All connected windows stay in sync via WebSocket events.

### DB

Migration:

```sql
CREATE TABLE map_tokens (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  map_id      INTEGER NOT NULL REFERENCES maps(id) ON DELETE CASCADE,
  entity_type TEXT NOT NULL CHECK(entity_type IN ('character','npc')),
  entity_id   INTEGER NOT NULL,
  x           REAL NOT NULL,
  y           REAL NOT NULL,
  UNIQUE(map_id, entity_type, entity_id)
);
```

The UNIQUE constraint prevents placing the same entity twice on one map.

### Backend

**Queries (`internal/db/queries_tokens.go`, new file):**
- `ListMapTokens(mapID int64) ([]MapToken, error)`
- `PlaceToken(mapID int64, entityType string, entityID int64, x, y float64) (int64, error)`
- `MoveToken(id int64, x, y float64) error`
- `RemoveToken(id int64) error`

**Validation:** Before INSERT, verify entity_id exists in `characters` (if type=character) or the NPC table (if type=npc) within the same campaign as the map — implementer should verify the NPC table name from existing queries in `queries_combat.go` or `queries_world.go`. Return 409 if already placed.

**Routes (`internal/api/routes_tokens.go`, new file):**
- `GET /api/maps/{id}/tokens` → ListMapTokens; each token includes entity name and type (JOIN or separate lookup)
- `POST /api/maps/{id}/tokens` — body `{entity_type, entity_id, x, y}`; broadcasts `token_placed`
- `PATCH /api/map-tokens/{id}` — body `{x, y}`; broadcasts `token_moved`
- `DELETE /api/map-tokens/{id}` — broadcasts `token_removed`

**WebSocket events:**
- `token_placed` → `{map_id, token: {id, entity_type, entity_id, name, x, y}}`
- `token_moved` → `{map_id, token_id, x, y}`
- `token_removed` → `{map_id, token_id}`

### Frontend

**`MapPanel.tsx` changes:**
- Fetch tokens via `GET /api/maps/{id}/tokens` when active map changes
- Token layer rendered above pins: 32px circles, gold border for characters (`entity_type === 'character'`), red border for NPCs, first character of entity name centered in white (`?` if name is empty)
- Drag to reposition: `onMouseDown` on token starts drag, `onMouseMove` on map updates position locally, `onMouseUp` calls `PATCH /api/map-tokens/{id}` with final normalized coords
- On token hover, a small `×` button appears (top-right of the circle); clicking it calls `DELETE /api/map-tokens/{id}` (same hover-reveal pattern as condition badge removal in CombatPanel)
- Subscribe to `token_placed`, `token_moved`, `token_removed` WS events

**Token palette:**
- Collapsible panel below the map (toggle button: "🎭 Tokens")
- Lists all campaign characters and NPCs not yet placed on the active map
- Each entry is a draggable chip (name + type badge); dragging onto the map image calls `POST /api/maps/{id}/tokens` with drop coordinates
- Already-placed entities shown greyed-out in the palette (not draggable)

---

## Feature 5: Fog of War

### Overview

Named rectangular zones are defined on each map. Zones start hidden (fog). The automation goroutine reveals zones automatically when the AI GM narrates entering them by name. The GM can manually toggle any zone at any time.

### DB

Migration:

```sql
CREATE TABLE map_zones (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  map_id      INTEGER NOT NULL REFERENCES maps(id) ON DELETE CASCADE,
  name        TEXT NOT NULL,
  x           REAL NOT NULL,
  y           REAL NOT NULL,
  width       REAL NOT NULL,
  height      REAL NOT NULL,
  is_revealed INTEGER NOT NULL DEFAULT 0
);
```

### Backend

**Queries (`internal/db/queries_zones.go`, new file):**
- `ListMapZones(mapID int64) ([]MapZone, error)`
- `CreateMapZone(mapID int64, name string, x, y, w, h float64) (int64, error)`
- `UpdateMapZone(id int64, name string, x, y, w, h float64) error`
- `DeleteMapZone(id int64) error`
- `RevealZone(id int64, revealed bool) error`
- `ListUnrevealedZones(mapID int64) ([]MapZone, error)`

**Routes (`internal/api/routes_zones.go`, new file):**
- `GET /api/maps/{id}/zones` → ListMapZones
- `POST /api/maps/{id}/zones` — body `{name, x, y, width, height}`
- `PATCH /api/map-zones/{id}` — body `{name?, x?, y?, width?, height?, is_revealed?}` (partial update)
- `DELETE /api/map-zones/{id}`

**WebSocket event:** `zone_revealed` → `{map_id, zone_id, zone_name, is_revealed}`

**AI map generation (`internal/api/routes.go`, `handleGenerateMap`):**
- Updated system prompt instructs the AI to append a `[ZONES]...[/ZONES]` block after the SVG containing a JSON array: `[{"name":"throne room","x":0.1,"y":0.2,"width":0.3,"height":0.25}]`
- Server strips and parses this block, calls `CreateMapZone` for each entry before returning the map response
- If the block is absent or malformed, map creation succeeds without zones (not an error)

**Fog reveal in automation goroutine (`internal/api/automation.go`):**
- After each GM response completes (streaming finished), if the session has an active map:
  1. Fetch all unrevealed zones for that map
  2. For each zone, do a case-insensitive `strings.Contains(narrativeText, zone.Name)` check
  3. On match: call `RevealZone`, broadcast `zone_revealed` WS event
- This runs synchronously after streaming completes — no separate goroutine needed

**Manual reveal:** Handled by `PATCH /api/map-zones/{id}` with `{is_revealed: true|false}`.

### Frontend

**`MapPanel.tsx` changes:**

**Fog layer:** Rendered above tokens and pins. For each unrevealed zone, render a dark semi-transparent `<div>` (rgba(0,0,0,0.75)) absolutely positioned using normalized coords × img dimensions. Revealed zones render nothing (fully transparent). Zone reveal animates: CSS `transition: opacity 0.6s ease-out`.

**Zone-edit mode** (toggle button "⬜ Zones" in map toolbar):
- Fog rects become outlined (2px dashed gold border, 20% opacity background) instead of opaque — so GM can see zones without blinding themselves
- Click + drag on map image to define a new zone rectangle: `onMouseDown` records start coords, `onMouseMove` shows preview rect, `onMouseUp` prompts for zone name (inline input) then calls `POST /api/maps/{id}/zones`
- Click existing zone in edit mode: shows delete button (×) and name input for rename

**Zone list panel** (shown when zone-edit mode is active, below token palette):
- Lists all zones for the active map with name, revealed status, and two buttons: reveal/hide toggle + delete
- Reveal/hide toggle calls `PATCH /api/map-zones/{id}` with `{is_revealed: !current}`

**Subscribe to `zone_revealed` WS event:** update zone state and trigger CSS transition.

---

## Implementation Order

Tasks (10 total, suitable for SDD):

1. Handouts — DB + Backend
2. Handouts — Frontend
3. Compendium — Embedding infrastructure + Backend (Ollama client, cosine util, search endpoint)
4. Compendium — Frontend
5. Card/Deck — DB + Backend
6. Card/Deck — Frontend
7. Token Map — DB + Backend
8. Token Map — Frontend
9. Fog of War — DB + Backend (zones API + AI map generation + automation goroutine reveal)
10. Fog of War — Frontend (fog layer + zone editor)
