# API and campaign storage

The Go server serves the embedded browser UI and the HTTP API. GM narration streams over server-sent events. WebSocket events tell the dashboard when a roll, sheet or other piece of game state changes. [Registered HTTP routes](../internal/api/server.go) · [Realtime event definitions](../contracts/realtime.json)

## Frequently used routes

| Task | Method and path |
| --- | --- |
| List campaigns or rulesets | `GET /api/campaigns` or `GET /api/rulesets` |
| Read or append to the transcript | `GET /api/sessions/{id}/messages` or `POST /api/sessions/{id}/messages` |
| Stream a GM response | `POST /api/sessions/{id}/gm-respond-stream` |
| List or generate campaign maps | `GET /api/campaigns/{id}/maps` or `POST /api/campaigns/{id}/maps/generate` |
| Prepare a session or find loose ends | `POST /api/campaigns/{id}/pre-session-brief` or `POST /api/sessions/{id}/detect-threads` |
| Improvise a complication or ask about the campaign | `POST /api/sessions/{id}/improvise` or `POST /api/campaigns/{id}/ask` |
| Roll on an oracle table | `POST /api/oracle/roll` |
| Read or update automation settings | `GET /api/settings/automations` or `PATCH /api/settings/automations` |

### Import and search a book on loopback

Use `GET /api/rulesets` to find the numeric ID, then substitute it below. Each source label identifies one book within a ruleset. On a private HTTPS table, API clients must also authenticate. [Authentication](security.md#listener-and-authentication-modes)

```bash
base=http://127.0.0.1:7432
ruleset_id=1
book="$base/api/rulesets/$ruleset_id/rulebook"
curl --fail-with-body "$book"
curl --fail-with-body "$book?source=Harbor+gazetteer" \
  -H 'Content-Type: text/plain' --data-binary @harbor.txt
curl --fail-with-body "$book/search" \
  -H 'Content-Type: application/json' -d '{"query":"ferryman"}'
```

For a PDF, submit a multipart form with a `rulebook` file field and a `source` label. The [import handler](../internal/api/routes_rulebook.go) covers accepted formats and the search response.

## Where state lives

| Scope | Stored records |
| --- | --- |
| Ruleset | Sheet schema, GM guidance and searchable rulebook chunks grouped by source. |
| Campaign | Characters, world notes, relationships, factions, adventures, calendar and maps. |
| Session | Messages, dice history, NPC roster and combat encounters. |
| Map | Pins, tokens and revealable zones. The image is served through the map’s ID. |

The default SQLite file is `~/.ttrpg/ttrpg.db`. Use `-db` to choose a different path. Uploaded assets live beside the database. The [ordered SQL migrations](../internal/db/migrations) define the tables and indexes. See [data recovery](data-recovery.md) before copying or restoring a table.

## Add a ruleset

Copy the [ruleset SQL template](ruleset-template.sql) into a new, incrementally numbered migration. Give the system a unique name, GM guidance and a field schema. Fields can specify types, categories, options, defaults, conditional visibility and computed values. A system with special stat-generation rules may also need [ruleset-specific Go logic](../internal/ruleset). Check both a fresh database and an upgrade from an existing one, then create a campaign and verify the sheet and a first roll. [Development checks](development.md)
