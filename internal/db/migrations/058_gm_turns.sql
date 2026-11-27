ALTER TABLE messages ADD COLUMN client_action_key TEXT;
ALTER TABLE messages ADD COLUMN rule_citations TEXT NOT NULL DEFAULT '[]';
CREATE UNIQUE INDEX messages_client_action_key ON messages(session_id, client_action_key) WHERE client_action_key IS NOT NULL;
CREATE TABLE gm_turns (
    action_message_id INTEGER PRIMARY KEY REFERENCES messages(id) ON DELETE CASCADE,
    response_message_id INTEGER UNIQUE REFERENCES messages(id) ON DELETE SET NULL,
    preparation TEXT,
    lease_token TEXT,
    lease_until INTEGER NOT NULL DEFAULT 0
);
