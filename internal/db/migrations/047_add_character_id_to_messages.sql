ALTER TABLE messages ADD COLUMN character_id INTEGER REFERENCES characters(id);
CREATE INDEX IF NOT EXISTS idx_messages_character_id ON messages(character_id);
