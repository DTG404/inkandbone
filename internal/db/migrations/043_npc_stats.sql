CREATE TABLE IF NOT EXISTS npc_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT '',
    data_json TEXT NOT NULL DEFAULT '{}',
    hp_max INTEGER NOT NULL DEFAULT 1,
    armor_class INTEGER DEFAULT NULL,
    initiative_mod INTEGER NOT NULL DEFAULT 0,
    skills TEXT NOT NULL DEFAULT '[]',
    abilities TEXT NOT NULL DEFAULT '[]',
    loot TEXT NOT NULL DEFAULT '[]',
    notes TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_npc_stats_campaign ON npc_stats(campaign_id);
