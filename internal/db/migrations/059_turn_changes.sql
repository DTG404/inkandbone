CREATE TABLE turn_changes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK(kind IN ('stat','currency','item','npc','objective')),
    owner_id INTEGER,
    entity_id INTEGER,
    field TEXT NOT NULL DEFAULT '',
    operation TEXT NOT NULL CHECK(operation IN ('add','remove','set')),
    dedup_key TEXT NOT NULL,
    label TEXT NOT NULL,
    before_json TEXT,
    after_json TEXT,
    corrected_at TEXT,
    superseded INTEGER NOT NULL DEFAULT 0,
    UNIQUE(message_id,kind,operation,dedup_key)
);
CREATE INDEX turn_changes_target ON turn_changes(kind,entity_id) WHERE corrected_at IS NULL AND superseded=0;

-- A later edit claims the field even when someone subsequently returns it to
-- the same value. These guards also cover writes from MCP and older clients.
CREATE TRIGGER turn_changes_stats AFTER UPDATE OF data_json ON characters
WHEN OLD.data_json IS NOT NEW.data_json
BEGIN
    UPDATE turn_changes SET superseded=1 WHERE kind='stat' AND entity_id=NEW.id
      AND superseded=0 AND corrected_at IS NULL AND (
        NOT json_valid(OLD.data_json) OR NOT json_valid(NEW.data_json) OR
        (SELECT json_array(type,value) FROM json_each(OLD.data_json) WHERE key=field)
        IS NOT (SELECT json_array(type,value) FROM json_each(NEW.data_json) WHERE key=field)
      );
END;
CREATE TRIGGER turn_changes_currency AFTER UPDATE OF currency_balance,currency_label ON characters
WHEN OLD.currency_balance IS NOT NEW.currency_balance OR OLD.currency_label IS NOT NEW.currency_label
BEGIN
    UPDATE turn_changes SET superseded=1 WHERE kind='currency' AND entity_id=NEW.id AND corrected_at IS NULL AND superseded=0;
END;
CREATE TRIGGER turn_changes_item_edit AFTER UPDATE ON items
WHEN (OLD.name,OLD.description,OLD.quantity,OLD.equipped,OLD.character_id) IS NOT (NEW.name,NEW.description,NEW.quantity,NEW.equipped,NEW.character_id)
BEGIN
    UPDATE turn_changes SET superseded=1 WHERE kind='item' AND entity_id=NEW.id AND corrected_at IS NULL AND superseded=0;
END;
CREATE TRIGGER turn_changes_item_delete AFTER DELETE ON items BEGIN
    UPDATE turn_changes SET superseded=1 WHERE kind='item' AND entity_id=OLD.id AND corrected_at IS NULL AND superseded=0;
END;
CREATE TRIGGER turn_changes_npc_edit AFTER UPDATE ON session_npcs
WHEN (OLD.name,OLD.note,OLD.session_id) IS NOT (NEW.name,NEW.note,NEW.session_id)
BEGIN
    UPDATE turn_changes SET superseded=1 WHERE kind='npc' AND entity_id=NEW.id AND corrected_at IS NULL AND superseded=0;
END;
CREATE TRIGGER turn_changes_npc_delete AFTER DELETE ON session_npcs BEGIN
    UPDATE turn_changes SET superseded=1 WHERE kind='npc' AND entity_id=OLD.id AND corrected_at IS NULL AND superseded=0;
END;
CREATE TRIGGER turn_changes_obj_edit AFTER UPDATE ON objectives
WHEN (OLD.title,OLD.description,OLD.status,OLD.parent_id,OLD.campaign_id) IS NOT (NEW.title,NEW.description,NEW.status,NEW.parent_id,NEW.campaign_id)
BEGIN
    UPDATE turn_changes SET superseded=1 WHERE kind='objective' AND entity_id=NEW.id AND corrected_at IS NULL AND superseded=0
      AND (operation='add' OR OLD.status IS NOT NEW.status OR OLD.campaign_id IS NOT NEW.campaign_id);
END;
CREATE TRIGGER turn_changes_obj_delete AFTER DELETE ON objectives BEGIN
    UPDATE turn_changes SET superseded=1 WHERE kind='objective' AND entity_id=OLD.id AND corrected_at IS NULL AND superseded=0;
END;
CREATE TRIGGER turn_changes_npc_return AFTER INSERT ON session_npcs BEGIN
    UPDATE turn_changes SET superseded=1 WHERE kind='npc' AND operation='remove' AND owner_id=NEW.session_id
      AND (entity_id=NEW.id OR lower(trim(label))=lower(trim(NEW.name))) AND corrected_at IS NULL AND superseded=0;
END;
CREATE TRIGGER turn_changes_item_return AFTER INSERT ON items BEGIN
    UPDATE turn_changes SET superseded=1 WHERE kind='item' AND operation='remove' AND owner_id=NEW.character_id
      AND (entity_id=NEW.id OR lower(trim(label))=lower(trim(NEW.name))) AND corrected_at IS NULL AND superseded=0;
END;
