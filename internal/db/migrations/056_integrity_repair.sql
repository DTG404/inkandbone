-- Repair historical foreign-key gaps without modifying immutable migrations.
-- The migration runner disables FK enforcement only for this script, while
-- retaining its exclusive database lock and transaction, then restores and
-- validates enforcement before releasing the lock.

CREATE TABLE orphaned_records (
    source_table TEXT NOT NULL,
    source_id INTEGER NOT NULL,
    payload_json TEXT NOT NULL,
    reason TEXT NOT NULL,
    quarantined_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_orphaned_records_source ON orphaned_records(source_table, source_id);

-- Ruleset-owned records. A campaign with a missing ruleset cannot be made
-- meaningful automatically, so quarantine it before walking its descendants.
INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'campaigns', c.id,
       json_object('id', c.id, 'ruleset_id', c.ruleset_id, 'name', c.name,
                   'description', c.description, 'active', c.active,
                   'created_at', c.created_at, 'chronicle_night', c.chronicle_night,
                   'chronicle_night_start_dow', c.chronicle_night_start_dow,
                   'in_game_year', c.in_game_year, 'in_game_month', c.in_game_month,
                   'in_game_day', c.in_game_day, 'calendar_config', c.calendar_config,
                   'gm_notes', c.gm_notes, 'system_prompt_override', c.system_prompt_override),
       'missing rulesets parent'
FROM campaigns c LEFT JOIN rulesets r ON r.id = c.ruleset_id
WHERE r.id IS NULL;
DELETE FROM campaigns WHERE NOT EXISTS (SELECT 1 FROM rulesets WHERE rulesets.id = campaigns.ruleset_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'rulebook_chunks', c.id,
       json_object('id', c.id, 'ruleset_id', c.ruleset_id, 'heading', c.heading,
                   'content', c.content, 'created_at', c.created_at, 'source', c.source,
                   'embedding_hex', CASE WHEN c.embedding IS NULL THEN NULL ELSE hex(c.embedding) END,
                   'embedding_encoding', CASE WHEN c.embedding IS NULL THEN NULL ELSE 'hex' END),
       'missing rulesets parent'
FROM rulebook_chunks c LEFT JOIN rulesets r ON r.id = c.ruleset_id
WHERE r.id IS NULL;
DELETE FROM rulebook_chunks WHERE NOT EXISTS (SELECT 1 FROM rulesets WHERE rulesets.id = rulebook_chunks.ruleset_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'oracle_tables', o.id,
       json_object('id', o.id, 'ruleset_id', o.ruleset_id, 'table_name', o.table_name,
                   'roll_min', o.roll_min, 'roll_max', o.roll_max, 'result', o.result),
       'missing optional rulesets parent; reference cleared'
FROM oracle_tables o LEFT JOIN rulesets r ON r.id = o.ruleset_id
WHERE o.ruleset_id IS NOT NULL AND r.id IS NULL;
UPDATE oracle_tables SET ruleset_id = NULL
WHERE ruleset_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM rulesets WHERE rulesets.id = oracle_tables.ruleset_id);

-- Campaign-owned records, including descendants of quarantined campaigns.
INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'characters', x.id,
       json_object('id', x.id, 'campaign_id', x.campaign_id, 'name', x.name,
                   'data_json', x.data_json, 'portrait_path', x.portrait_path,
                   'currency_balance', x.currency_balance, 'currency_label', x.currency_label,
                   'created_at', x.created_at),
       'missing campaigns parent'
FROM characters x LEFT JOIN campaigns c ON c.id = x.campaign_id WHERE c.id IS NULL;
DELETE FROM characters WHERE NOT EXISTS (SELECT 1 FROM campaigns WHERE campaigns.id = characters.campaign_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'sessions', x.id,
       json_object('id', x.id, 'campaign_id', x.campaign_id, 'title', x.title,
                   'date', x.date, 'summary', x.summary, 'notes', x.notes,
                   'tension_level', x.tension_level, 'scene_tags', x.scene_tags,
                   'masquerade_integrity', x.masquerade_integrity,
                   'adventure_id', x.adventure_id, 'created_at', x.created_at),
       'missing campaigns parent'
FROM sessions x LEFT JOIN campaigns c ON c.id = x.campaign_id WHERE c.id IS NULL;
DELETE FROM sessions WHERE NOT EXISTS (SELECT 1 FROM campaigns WHERE campaigns.id = sessions.campaign_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'world_notes', x.id,
       json_object('id', x.id, 'campaign_id', x.campaign_id, 'title', x.title,
                   'content', x.content, 'category', x.category, 'tags_json', x.tags_json,
                   'personality_json', x.personality_json, 'is_revealed', x.is_revealed,
                   'created_at', x.created_at),
       'missing campaigns parent'
FROM world_notes x LEFT JOIN campaigns c ON c.id = x.campaign_id WHERE c.id IS NULL;
DELETE FROM world_notes WHERE NOT EXISTS (SELECT 1 FROM campaigns WHERE campaigns.id = world_notes.campaign_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'maps', x.id,
       json_object('id', x.id, 'campaign_id', x.campaign_id, 'name', x.name,
                   'image_path', x.image_path, 'created_at', x.created_at),
       'missing campaigns parent'
FROM maps x LEFT JOIN campaigns c ON c.id = x.campaign_id WHERE c.id IS NULL;
DELETE FROM maps WHERE NOT EXISTS (SELECT 1 FROM campaigns WHERE campaigns.id = maps.campaign_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'objectives', x.id,
       json_object('id', x.id, 'campaign_id', x.campaign_id, 'title', x.title,
                   'description', x.description, 'status', x.status,
                   'parent_id', x.parent_id, 'created_at', x.created_at),
       'missing campaigns parent'
FROM objectives x LEFT JOIN campaigns c ON c.id = x.campaign_id WHERE c.id IS NULL;
DELETE FROM objectives WHERE NOT EXISTS (SELECT 1 FROM campaigns WHERE campaigns.id = objectives.campaign_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'relationships', x.id,
       json_object('id', x.id, 'campaign_id', x.campaign_id, 'from_name', x.from_name,
                   'to_name', x.to_name, 'relationship_type', x.relationship_type,
                   'description', x.description, 'created_at', x.created_at),
       'missing campaigns parent'
FROM relationships x LEFT JOIN campaigns c ON c.id = x.campaign_id WHERE c.id IS NULL;
DELETE FROM relationships WHERE NOT EXISTS (SELECT 1 FROM campaigns WHERE campaigns.id = relationships.campaign_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'factions', x.id,
       json_object('id', x.id, 'campaign_id', x.campaign_id, 'name', x.name,
                   'description', x.description, 'faction_type', x.faction_type,
                   'influence', x.influence, 'resources_json', x.resources_json,
                   'color', x.color, 'created_at', x.created_at),
       'missing campaigns parent'
FROM factions x LEFT JOIN campaigns c ON c.id = x.campaign_id WHERE c.id IS NULL;
DELETE FROM factions WHERE NOT EXISTS (SELECT 1 FROM campaigns WHERE campaigns.id = factions.campaign_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'adventures', x.id,
       json_object('id', x.id, 'campaign_id', x.campaign_id, 'title', x.title,
                   'description', x.description, 'status', x.status,
                   'sort_order', x.sort_order, 'created_at', x.created_at),
       'missing campaigns parent'
FROM adventures x LEFT JOIN campaigns c ON c.id = x.campaign_id WHERE c.id IS NULL;
DELETE FROM adventures WHERE NOT EXISTS (SELECT 1 FROM campaigns WHERE campaigns.id = adventures.campaign_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'npc_stats', x.id,
       json_object('id', x.id, 'campaign_id', x.campaign_id, 'name', x.name,
                   'role', x.role, 'data_json', x.data_json, 'hp_max', x.hp_max,
                   'armor_class', x.armor_class, 'initiative_mod', x.initiative_mod,
                   'skills', x.skills, 'abilities', x.abilities, 'loot', x.loot,
                   'notes', x.notes, 'created_at', x.created_at),
       'missing campaigns parent'
FROM npc_stats x LEFT JOIN campaigns c ON c.id = x.campaign_id WHERE c.id IS NULL;
DELETE FROM npc_stats WHERE NOT EXISTS (SELECT 1 FROM campaigns WHERE campaigns.id = npc_stats.campaign_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'secrets', x.id,
       json_object('id', x.id, 'campaign_id', x.campaign_id, 'title', x.title,
                   'content', x.content, 'category', x.category, 'revealed', x.revealed,
                   'revealed_at_session_id', x.revealed_at_session_id, 'created_at', x.created_at),
       'missing campaigns parent'
FROM secrets x LEFT JOIN campaigns c ON c.id = x.campaign_id WHERE c.id IS NULL;
DELETE FROM secrets WHERE NOT EXISTS (SELECT 1 FROM campaigns WHERE campaigns.id = secrets.campaign_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'calendar_events', x.id,
       json_object('id', x.id, 'campaign_id', x.campaign_id, 'title', x.title,
                   'in_game_year', x.in_game_year, 'in_game_month', x.in_game_month,
                   'in_game_day', x.in_game_day,
                   'description', x.description, 'event_type', x.event_type,
                   'session_id', x.session_id, 'created_at', x.created_at),
       'missing campaigns parent'
FROM calendar_events x LEFT JOIN campaigns c ON c.id = x.campaign_id WHERE c.id IS NULL;
DELETE FROM calendar_events WHERE NOT EXISTS (SELECT 1 FROM campaigns WHERE campaigns.id = calendar_events.campaign_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'decks', x.id,
       json_object('id', x.id, 'campaign_id', x.campaign_id, 'name', x.name,
                   'cards_json', x.cards_json, 'shuffled_order_json', x.shuffled_order_json,
                   'draw_index', x.draw_index, 'created_at', x.created_at),
       'missing campaigns parent'
FROM decks x LEFT JOIN campaigns c ON c.id = x.campaign_id WHERE c.id IS NULL;
DELETE FROM decks WHERE NOT EXISTS (SELECT 1 FROM campaigns WHERE campaigns.id = decks.campaign_id);

-- Character-owned records and optional character references.
INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'items', x.id,
       json_object('id', x.id, 'character_id', x.character_id, 'name', x.name,
                   'description', x.description, 'quantity', x.quantity,
                   'equipped', x.equipped, 'created_at', x.created_at),
       'missing characters parent'
FROM items x LEFT JOIN characters c ON c.id = x.character_id WHERE c.id IS NULL;
DELETE FROM items WHERE NOT EXISTS (SELECT 1 FROM characters WHERE characters.id = items.character_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'character_macros', x.id,
       json_object('id', x.id, 'character_id', x.character_id, 'label', x.label,
                   'action_text', x.action_text, 'color', x.color,
                   'sort_order', x.sort_order, 'created_at', x.created_at),
       'missing characters parent'
FROM character_macros x LEFT JOIN characters c ON c.id = x.character_id WHERE c.id IS NULL;
DELETE FROM character_macros WHERE NOT EXISTS (SELECT 1 FROM characters WHERE characters.id = character_macros.character_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'messages', x.id,
       json_object('id', x.id, 'session_id', x.session_id, 'role', x.role,
                   'content', x.content, 'whisper', x.whisper,
                   'character_id', x.character_id, 'created_at', x.created_at),
       'missing optional characters parent; reference cleared'
FROM messages x LEFT JOIN characters c ON c.id = x.character_id
WHERE x.character_id IS NOT NULL AND c.id IS NULL;
UPDATE messages SET character_id = NULL
WHERE character_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM characters WHERE characters.id = messages.character_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'combatants', x.id,
       json_object('id', x.id, 'encounter_id', x.encounter_id, 'character_id', x.character_id,
                   'name', x.name, 'initiative', x.initiative, 'hp_current', x.hp_current,
                   'hp_max', x.hp_max, 'conditions_json', x.conditions_json,
                   'is_player', x.is_player, 'damage_superficial', x.damage_superficial,
                   'damage_aggravated', x.damage_aggravated,
                   'willpower_superficial', x.willpower_superficial,
                   'willpower_aggravated', x.willpower_aggravated,
                   'hunger', x.hunger, 'sort_order', x.sort_order),
       'missing optional characters parent; reference cleared'
FROM combatants x LEFT JOIN characters c ON c.id = x.character_id
WHERE x.character_id IS NOT NULL AND c.id IS NULL;
UPDATE combatants SET character_id = NULL
WHERE character_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM characters WHERE characters.id = combatants.character_id);

-- Adventure references are optional and survive adventure deletion.
INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'sessions', x.id,
       json_object('id', x.id, 'campaign_id', x.campaign_id, 'title', x.title,
                   'date', x.date, 'summary', x.summary, 'notes', x.notes,
                   'tension_level', x.tension_level, 'scene_tags', x.scene_tags,
                   'masquerade_integrity', x.masquerade_integrity,
                   'adventure_id', x.adventure_id, 'created_at', x.created_at),
       'missing optional adventures parent; reference cleared'
FROM sessions x LEFT JOIN adventures a ON a.id = x.adventure_id
WHERE x.adventure_id IS NOT NULL AND a.id IS NULL;
UPDATE sessions SET adventure_id = NULL
WHERE adventure_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM adventures WHERE adventures.id = sessions.adventure_id);

-- Session-owned records and optional session references.
INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'messages', x.id,
       json_object('id', x.id, 'session_id', x.session_id, 'role', x.role,
                   'content', x.content, 'whisper', x.whisper,
                   'character_id', x.character_id, 'created_at', x.created_at),
       'missing sessions parent'
FROM messages x LEFT JOIN sessions s ON s.id = x.session_id WHERE s.id IS NULL;
DELETE FROM messages WHERE NOT EXISTS (SELECT 1 FROM sessions WHERE sessions.id = messages.session_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'session_npcs', x.id,
       json_object('id', x.id, 'session_id', x.session_id, 'name', x.name,
                   'note', x.note, 'created_at', x.created_at),
       'missing sessions parent'
FROM session_npcs x LEFT JOIN sessions s ON s.id = x.session_id WHERE s.id IS NULL;
DELETE FROM session_npcs WHERE NOT EXISTS (SELECT 1 FROM sessions WHERE sessions.id = session_npcs.session_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'combat_encounters', x.id,
       json_object('id', x.id, 'session_id', x.session_id, 'name', x.name,
                   'active', x.active, 'active_turn_index', x.active_turn_index,
                   'round_number', x.round_number, 'created_at', x.created_at),
       'missing sessions parent'
FROM combat_encounters x LEFT JOIN sessions s ON s.id = x.session_id WHERE s.id IS NULL;
DELETE FROM combat_encounters WHERE NOT EXISTS (SELECT 1 FROM sessions WHERE sessions.id = combat_encounters.session_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'dice_rolls', x.id,
       json_object('id', x.id, 'session_id', x.session_id, 'expression', x.expression,
                   'result', x.result, 'breakdown_json', x.breakdown_json,
                   'created_at', x.created_at),
       'missing sessions parent'
FROM dice_rolls x LEFT JOIN sessions s ON s.id = x.session_id WHERE s.id IS NULL;
DELETE FROM dice_rolls WHERE NOT EXISTS (SELECT 1 FROM sessions WHERE sessions.id = dice_rolls.session_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'xp_log', x.id,
       json_object('id', x.id, 'session_id', x.session_id, 'note', x.note,
                   'amount', x.amount, 'created_at', x.created_at),
       'missing sessions parent'
FROM xp_log x LEFT JOIN sessions s ON s.id = x.session_id WHERE s.id IS NULL;
DELETE FROM xp_log WHERE NOT EXISTS (SELECT 1 FROM sessions WHERE sessions.id = xp_log.session_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'secrets', x.id,
       json_object('id', x.id, 'campaign_id', x.campaign_id, 'title', x.title,
                   'content', x.content, 'category', x.category, 'revealed', x.revealed,
                   'revealed_at_session_id', x.revealed_at_session_id,
                   'created_at', x.created_at),
       'missing optional sessions parent; reference cleared'
FROM secrets x LEFT JOIN sessions s ON s.id = x.revealed_at_session_id
WHERE x.revealed_at_session_id IS NOT NULL AND s.id IS NULL;
UPDATE secrets SET revealed_at_session_id = NULL
WHERE revealed_at_session_id IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM sessions WHERE sessions.id = secrets.revealed_at_session_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'calendar_events', x.id,
       json_object('id', x.id, 'campaign_id', x.campaign_id, 'title', x.title,
                   'in_game_year', x.in_game_year, 'in_game_month', x.in_game_month,
                   'in_game_day', x.in_game_day, 'description', x.description,
                   'session_id', x.session_id, 'event_type', x.event_type,
                   'created_at', x.created_at),
       'missing optional sessions parent; reference cleared'
FROM calendar_events x LEFT JOIN sessions s ON s.id = x.session_id
WHERE x.session_id IS NOT NULL AND s.id IS NULL;
UPDATE calendar_events SET session_id = NULL
WHERE session_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM sessions WHERE sessions.id = calendar_events.session_id);

-- Deck draws depend on both parents and combatants depend on encounters.
INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'deck_draws', x.id,
       json_object('id', x.id, 'session_id', x.session_id, 'deck_id', x.deck_id,
                   'card_json', x.card_json, 'drawn_at', x.drawn_at),
       CASE WHEN s.id IS NULL THEN 'missing sessions parent' ELSE 'missing decks parent' END
FROM deck_draws x
LEFT JOIN sessions s ON s.id = x.session_id
LEFT JOIN decks d ON d.id = x.deck_id
WHERE s.id IS NULL OR d.id IS NULL;
DELETE FROM deck_draws
WHERE NOT EXISTS (SELECT 1 FROM sessions WHERE sessions.id = deck_draws.session_id)
   OR NOT EXISTS (SELECT 1 FROM decks WHERE decks.id = deck_draws.deck_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'combatants', x.id,
       json_object('id', x.id, 'encounter_id', x.encounter_id, 'character_id', x.character_id,
                   'name', x.name, 'initiative', x.initiative, 'hp_current', x.hp_current,
                   'hp_max', x.hp_max, 'conditions_json', x.conditions_json,
                   'is_player', x.is_player, 'damage_superficial', x.damage_superficial,
                   'damage_aggravated', x.damage_aggravated,
                   'willpower_superficial', x.willpower_superficial,
                   'willpower_aggravated', x.willpower_aggravated,
                   'hunger', x.hunger, 'sort_order', x.sort_order),
       'missing combat_encounters parent'
FROM combatants x LEFT JOIN combat_encounters e ON e.id = x.encounter_id WHERE e.id IS NULL;
DELETE FROM combatants WHERE NOT EXISTS (SELECT 1 FROM combat_encounters WHERE combat_encounters.id = combatants.encounter_id);

-- Map-owned records.
INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'map_pins', x.id,
       json_object('id', x.id, 'map_id', x.map_id, 'x', x.x, 'y', x.y,
                   'label', x.label, 'note', x.note, 'color', x.color,
                   'created_at', x.created_at),
       'missing maps parent'
FROM map_pins x LEFT JOIN maps m ON m.id = x.map_id WHERE m.id IS NULL;
DELETE FROM map_pins WHERE NOT EXISTS (SELECT 1 FROM maps WHERE maps.id = map_pins.map_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'map_tokens', x.id,
       json_object('id', x.id, 'map_id', x.map_id, 'entity_type', x.entity_type,
                   'entity_id', x.entity_id, 'x', x.x, 'y', x.y),
       'missing maps parent'
FROM map_tokens x LEFT JOIN maps m ON m.id = x.map_id WHERE m.id IS NULL;
DELETE FROM map_tokens WHERE NOT EXISTS (SELECT 1 FROM maps WHERE maps.id = map_tokens.map_id);

INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'map_zones', x.id,
       json_object('id', x.id, 'map_id', x.map_id, 'name', x.name,
                   'x', x.x, 'y', x.y, 'width', x.width, 'height', x.height,
                   'is_revealed', x.is_revealed),
       'missing maps parent'
FROM map_zones x LEFT JOIN maps m ON m.id = x.map_id WHERE m.id IS NULL;
DELETE FROM map_zones WHERE NOT EXISTS (SELECT 1 FROM maps WHERE maps.id = map_zones.map_id);

-- map_tokens uses a polymorphic entity reference that SQLite cannot express as
-- a conventional foreign key. Quarantine missing targets before installing
-- equivalent trigger-backed validation and delete behavior below.
INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'map_tokens', x.id,
       json_object('id', x.id, 'map_id', x.map_id, 'entity_type', x.entity_type,
                   'entity_id', x.entity_id, 'x', x.x, 'y', x.y),
       CASE WHEN x.entity_type = 'character'
            THEN 'missing characters polymorphic parent'
            ELSE 'missing session_npcs polymorphic parent' END
FROM map_tokens x
LEFT JOIN characters c ON x.entity_type = 'character' AND c.id = x.entity_id
LEFT JOIN session_npcs n ON x.entity_type = 'npc' AND n.id = x.entity_id
WHERE (x.entity_type = 'character' AND c.id IS NULL)
   OR (x.entity_type = 'npc' AND n.id IS NULL);
DELETE FROM map_tokens
WHERE (entity_type = 'character' AND NOT EXISTS (SELECT 1 FROM characters WHERE characters.id = map_tokens.entity_id))
   OR (entity_type = 'npc' AND NOT EXISTS (SELECT 1 FROM session_npcs WHERE session_npcs.id = map_tokens.entity_id));

-- Objective children have no independent meaning without their parent.
INSERT INTO orphaned_records(source_table, source_id, payload_json, reason)
SELECT 'objectives', x.id,
       json_object('id', x.id, 'campaign_id', x.campaign_id, 'title', x.title,
                   'description', x.description, 'status', x.status,
                   'parent_id', x.parent_id, 'created_at', x.created_at),
       'missing objectives parent'
FROM objectives x LEFT JOIN objectives p ON p.id = x.parent_id
WHERE x.parent_id IS NOT NULL AND p.id IS NULL;
DELETE FROM objectives
WHERE parent_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM objectives p WHERE p.id = objectives.parent_id);

-- Rewrite every known generated-map link before removing HTTP compatibility.
WITH RECURSIVE
map_rewrites AS (
    SELECT id, image_path, row_number() OVER (ORDER BY id) AS step
    FROM maps WHERE image_path <> ''
),
rewritten(message_id, step, content) AS (
    SELECT id, 0, content FROM messages
    UNION ALL
    SELECT r.message_id, r.step + 1,
           replace(r.content, '/api/files/' || m.image_path, '/api/assets/maps/' || m.id)
    FROM rewritten r JOIN map_rewrites m ON m.step = r.step + 1
)
UPDATE messages
SET content = (
    SELECT content FROM rewritten r
    WHERE r.message_id = messages.id
    ORDER BY r.step DESC LIMIT 1
)
WHERE instr(content, '/api/files/') > 0;

-- Rebuild only tables whose historical action is absent or incorrect.
CREATE TABLE campaigns_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ruleset_id INTEGER NOT NULL REFERENCES rulesets(id) ON DELETE RESTRICT,
    name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    chronicle_night INTEGER NOT NULL DEFAULT 1, chronicle_night_start_dow INTEGER NOT NULL DEFAULT -1,
    in_game_year INTEGER NOT NULL DEFAULT 1, in_game_month INTEGER NOT NULL DEFAULT 1,
    in_game_day INTEGER NOT NULL DEFAULT 1, calendar_config TEXT NOT NULL DEFAULT '{}',
    gm_notes TEXT NOT NULL DEFAULT '', system_prompt_override TEXT NOT NULL DEFAULT ''
);
INSERT INTO campaigns_new SELECT * FROM campaigns;
DROP TABLE campaigns;
ALTER TABLE campaigns_new RENAME TO campaigns;

CREATE TABLE characters_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name TEXT NOT NULL, data_json TEXT NOT NULL DEFAULT '{}', portrait_path TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    currency_balance INTEGER NOT NULL DEFAULT 0, currency_label TEXT NOT NULL DEFAULT 'Gold'
);
INSERT INTO characters_new SELECT * FROM characters;
DROP TABLE characters;
ALTER TABLE characters_new RENAME TO characters;

CREATE TABLE sessions_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    title TEXT NOT NULL, date TEXT NOT NULL, summary TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, notes TEXT NOT NULL DEFAULT '',
    tension_level INTEGER NOT NULL DEFAULT 5, scene_tags TEXT NOT NULL DEFAULT '',
    masquerade_integrity INTEGER NOT NULL DEFAULT 10,
    adventure_id INTEGER REFERENCES adventures(id) ON DELETE SET NULL
);
INSERT INTO sessions_new SELECT * FROM sessions;
DROP TABLE sessions;
ALTER TABLE sessions_new RENAME TO sessions;

CREATE TABLE messages_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK(role IN ('user', 'assistant')), content TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    whisper INTEGER NOT NULL DEFAULT 0,
    character_id INTEGER REFERENCES characters(id) ON DELETE SET NULL
);
INSERT INTO messages_new SELECT * FROM messages;
DROP TABLE messages;
ALTER TABLE messages_new RENAME TO messages;

CREATE TABLE combat_encounters_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    name TEXT NOT NULL, active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    active_turn_index INTEGER NOT NULL DEFAULT 0, round_number INTEGER NOT NULL DEFAULT 1
);
INSERT INTO combat_encounters_new SELECT * FROM combat_encounters;
DROP TABLE combat_encounters;
ALTER TABLE combat_encounters_new RENAME TO combat_encounters;

CREATE TABLE combatants_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    encounter_id INTEGER NOT NULL REFERENCES combat_encounters(id) ON DELETE CASCADE,
    character_id INTEGER REFERENCES characters(id) ON DELETE SET NULL,
    name TEXT NOT NULL, initiative INTEGER NOT NULL DEFAULT 0,
    hp_current INTEGER NOT NULL DEFAULT 0, hp_max INTEGER NOT NULL DEFAULT 0,
    conditions_json TEXT NOT NULL DEFAULT '[]', is_player INTEGER NOT NULL DEFAULT 0,
    damage_superficial INTEGER NOT NULL DEFAULT 0, damage_aggravated INTEGER NOT NULL DEFAULT 0,
    willpower_superficial INTEGER NOT NULL DEFAULT 0, willpower_aggravated INTEGER NOT NULL DEFAULT 0,
    hunger INTEGER NOT NULL DEFAULT 0, sort_order INTEGER NOT NULL DEFAULT 0
);
INSERT INTO combatants_new SELECT * FROM combatants;
DROP TABLE combatants;
ALTER TABLE combatants_new RENAME TO combatants;

CREATE TABLE world_notes_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    title TEXT NOT NULL, content TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT 'other' CHECK(category IN ('npc','location','faction','item','other')),
    tags_json TEXT NOT NULL DEFAULT '[]', created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    personality_json TEXT NOT NULL DEFAULT '', is_revealed INTEGER NOT NULL DEFAULT 0
);
INSERT INTO world_notes_new SELECT * FROM world_notes;
DROP TABLE world_notes;
ALTER TABLE world_notes_new RENAME TO world_notes;

CREATE TABLE maps_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name TEXT NOT NULL, image_path TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO maps_new SELECT * FROM maps;
DROP TABLE maps;
ALTER TABLE maps_new RENAME TO maps;

CREATE TABLE map_pins_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    map_id INTEGER NOT NULL REFERENCES maps(id) ON DELETE CASCADE,
    x REAL NOT NULL CHECK(x >= 0.0 AND x <= 1.0), y REAL NOT NULL CHECK(y >= 0.0 AND y <= 1.0),
    label TEXT NOT NULL DEFAULT '', note TEXT NOT NULL DEFAULT '', color TEXT NOT NULL DEFAULT '#ff0000',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO map_pins_new SELECT * FROM map_pins;
DROP TABLE map_pins;
ALTER TABLE map_pins_new RENAME TO map_pins;

CREATE TABLE dice_rolls_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    expression TEXT NOT NULL, result INTEGER NOT NULL,
    breakdown_json TEXT NOT NULL DEFAULT '{}', created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO dice_rolls_new SELECT * FROM dice_rolls;
DROP TABLE dice_rolls;
ALTER TABLE dice_rolls_new RENAME TO dice_rolls;

CREATE TABLE rulebook_chunks_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ruleset_id INTEGER NOT NULL REFERENCES rulesets(id) ON DELETE CASCADE,
    heading TEXT NOT NULL, content TEXT NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    source TEXT NOT NULL DEFAULT 'Core Rulebook', embedding BLOB
);
INSERT INTO rulebook_chunks_new SELECT * FROM rulebook_chunks;
DROP TABLE rulebook_chunks;
ALTER TABLE rulebook_chunks_new RENAME TO rulebook_chunks;

CREATE TABLE session_npcs_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    name TEXT NOT NULL, note TEXT NOT NULL DEFAULT '', created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO session_npcs_new SELECT * FROM session_npcs;
DROP TABLE session_npcs;
ALTER TABLE session_npcs_new RENAME TO session_npcs;

CREATE TABLE objectives_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    title TEXT NOT NULL, description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','completed','failed')),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    parent_id INTEGER REFERENCES objectives_new(id) ON DELETE CASCADE
);
INSERT INTO objectives_new SELECT * FROM objectives;
DROP TABLE objectives;
ALTER TABLE objectives_new RENAME TO objectives;

CREATE TABLE items_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', quantity INTEGER NOT NULL DEFAULT 1,
    equipped INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
INSERT INTO items_new SELECT * FROM items;
DROP TABLE items;
ALTER TABLE items_new RENAME TO items;

CREATE TABLE secrets_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    title TEXT NOT NULL, content TEXT NOT NULL, category TEXT NOT NULL DEFAULT 'secret',
    revealed INTEGER NOT NULL DEFAULT 0,
    revealed_at_session_id INTEGER REFERENCES sessions(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO secrets_new SELECT * FROM secrets;
DROP TABLE secrets;
ALTER TABLE secrets_new RENAME TO secrets;

CREATE TABLE calendar_events_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    in_game_year INTEGER NOT NULL DEFAULT 1, in_game_month INTEGER NOT NULL DEFAULT 1,
    in_game_day INTEGER NOT NULL DEFAULT 1, title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '', event_type TEXT NOT NULL DEFAULT 'note',
    session_id INTEGER REFERENCES sessions(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
INSERT INTO calendar_events_new SELECT * FROM calendar_events;
DROP TABLE calendar_events;
ALTER TABLE calendar_events_new RENAME TO calendar_events;

-- Recreate historical indexes removed with rebuilt tables and add indexes for
-- measured high-frequency query shapes.
CREATE INDEX idx_messages_character_id ON messages(character_id);
CREATE INDEX idx_messages_session_order ON messages(session_id, created_at, id);
CREATE INDEX idx_sessions_campaign_date ON sessions(campaign_id, date DESC, id);
CREATE INDEX idx_characters_campaign_name ON characters(campaign_id, name, id);
CREATE INDEX idx_objectives_campaign_status ON objectives(campaign_id, status, id);
CREATE INDEX idx_maps_campaign_created ON maps(campaign_id, created_at, id);
CREATE INDEX idx_map_pins_map ON map_pins(map_id, id);
CREATE INDEX idx_map_zones_map ON map_zones(map_id, id);
CREATE INDEX idx_combatants_encounter_order ON combatants(encounter_id, sort_order, id);
CREATE INDEX idx_secrets_campaign ON secrets(campaign_id);
CREATE INDEX idx_calendar_events_campaign ON calendar_events(campaign_id);

CREATE TRIGGER validate_map_token_entity_insert
BEFORE INSERT ON map_tokens
BEGIN
    SELECT CASE
        WHEN NEW.entity_type = 'character'
         AND NOT EXISTS (SELECT 1 FROM characters WHERE id = NEW.entity_id)
        THEN RAISE(ABORT, 'map token character does not exist')
        WHEN NEW.entity_type = 'npc'
         AND NOT EXISTS (SELECT 1 FROM session_npcs WHERE id = NEW.entity_id)
        THEN RAISE(ABORT, 'map token npc does not exist')
    END;
END;

CREATE TRIGGER validate_map_token_entity_update
BEFORE UPDATE OF entity_type, entity_id ON map_tokens
BEGIN
    SELECT CASE
        WHEN NEW.entity_type = 'character'
         AND NOT EXISTS (SELECT 1 FROM characters WHERE id = NEW.entity_id)
        THEN RAISE(ABORT, 'map token character does not exist')
        WHEN NEW.entity_type = 'npc'
         AND NOT EXISTS (SELECT 1 FROM session_npcs WHERE id = NEW.entity_id)
        THEN RAISE(ABORT, 'map token npc does not exist')
    END;
END;

CREATE TRIGGER cascade_character_map_tokens
AFTER DELETE ON characters
BEGIN
    DELETE FROM map_tokens WHERE entity_type = 'character' AND entity_id = OLD.id;
END;

CREATE TRIGGER cascade_session_npc_map_tokens
AFTER DELETE ON session_npcs
BEGIN
    DELETE FROM map_tokens WHERE entity_type = 'npc' AND entity_id = OLD.id;
END;

-- Remove only the unreferenced accidental template seed. If a historical
-- campaign or other record deliberately references it, preserve that data.
DELETE FROM rulesets
WHERE name = 'my_ruleset'
  AND NOT EXISTS (SELECT 1 FROM campaigns WHERE campaigns.ruleset_id = rulesets.id)
  AND NOT EXISTS (SELECT 1 FROM rulebook_chunks WHERE rulebook_chunks.ruleset_id = rulesets.id)
  AND NOT EXISTS (SELECT 1 FROM oracle_tables WHERE oracle_tables.ruleset_id = rulesets.id);
