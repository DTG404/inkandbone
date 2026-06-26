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
