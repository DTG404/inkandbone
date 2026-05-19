# Scripts

## Comprehensive E2E Test

`node scripts/e2e-comprehensive.mjs`

Tests **every feature** of ink & bone: all 12 sidebar tabs, all 5 Manage panel tabs, all CRUD API endpoints, edge cases, UI interactions, and entity systems. Currently **187 assertions** covering the full application surface.

### Prerequisites

- Node.js 18+
- Playwright: `npm install playwright` (run from project root)
- Chromium: `npx playwright install chromium`

### Running

1. Start the server with a clean database:
   ```bash
   rm -f ~/.ttrpg
   ttrpg -db /tmp/e2e-test.db
   ```

2. In another terminal:
   ```bash
   node scripts/e2e-comprehensive.mjs
   ```

3. Expected result: **187 passed, 0 failed**

### What It Tests

| Section | Coverage |
|---------|----------|
| 1 | Data seeding via HTTP API (campaign, character, session, messages, dice, factions, adventures, NPC stats, secrets, calendar, XP, maps) |
| 2-5 | App navigation, header, character sheet, XP log CRUD, session view |
| 6-18 | All 12 right sidebar tabs (Notes → GM Tools) rendered with content |
| 19 | Journal sub-tabs (Notes ↔ Timeline), Reanalyze button |
| 20-22 | GM Screen overlay, Player History overlay, Talents overlay |
| 23-28 | Theme toggle, audio, story search, export, whisper, scene tags |
| 29-30 | Manage panel with 5 tabs, Automation toggle settings |
| 31-32 | Character sheet inline editing, inventory add via UI |
| 33-64 | Full CRUD API tests: campaign config, secrets, factions, adventures, NPC stats, calendar, automations, world notes, objectives, sessions, tension, oracle, dice, timeline, relationships, health, context, rulesets, character options, talent descriptions, maps, file serving, rulebook ingest, dice validation, oracle validation, campaign deletion |
