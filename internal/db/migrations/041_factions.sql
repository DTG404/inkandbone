CREATE TABLE IF NOT EXISTS factions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    faction_type TEXT NOT NULL DEFAULT 'faction',
    influence INTEGER NOT NULL DEFAULT 5,
    resources_json TEXT NOT NULL DEFAULT '{}',
    color TEXT NOT NULL DEFAULT '#c9a84c',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_factions_campaign ON factions(campaign_id);
