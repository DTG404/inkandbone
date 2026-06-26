CREATE TABLE map_tokens (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  map_id      INTEGER NOT NULL REFERENCES maps(id) ON DELETE CASCADE,
  entity_type TEXT NOT NULL CHECK(entity_type IN ('character','npc')),
  entity_id   INTEGER NOT NULL,
  x           REAL NOT NULL,
  y           REAL NOT NULL,
  UNIQUE(map_id, entity_type, entity_id)
);
