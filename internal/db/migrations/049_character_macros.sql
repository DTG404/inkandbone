CREATE TABLE character_macros (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  label        TEXT NOT NULL,
  action_text  TEXT NOT NULL,
  color        TEXT NOT NULL DEFAULT 'var(--gold)',
  sort_order   INTEGER NOT NULL DEFAULT 0,
  created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
)
