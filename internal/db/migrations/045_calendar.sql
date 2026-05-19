ALTER TABLE campaigns ADD COLUMN in_game_year INTEGER NOT NULL DEFAULT 1;
ALTER TABLE campaigns ADD COLUMN in_game_month INTEGER NOT NULL DEFAULT 1;
ALTER TABLE campaigns ADD COLUMN in_game_day INTEGER NOT NULL DEFAULT 1;
ALTER TABLE campaigns ADD COLUMN calendar_config TEXT NOT NULL DEFAULT '{}';

CREATE TABLE IF NOT EXISTS calendar_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    in_game_year INTEGER NOT NULL DEFAULT 1,
    in_game_month INTEGER NOT NULL DEFAULT 1,
    in_game_day INTEGER NOT NULL DEFAULT 1,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    event_type TEXT NOT NULL DEFAULT 'note',
    session_id INTEGER REFERENCES sessions(id),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_calendar_events_campaign ON calendar_events(campaign_id);
