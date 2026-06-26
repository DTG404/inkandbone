package db

import (
	"database/sql"
	"math"
	"time"
)

// RulebookChunk represents a parsed chunk of rulebook text for a given ruleset.
type RulebookChunk struct {
	ID        int64     `json:"id"`
	RulesetID int64     `json:"ruleset_id"`
	Source    string    `json:"source"`
	Heading   string    `json:"heading"`
	Content   string    `json:"content"`
	Embedding []byte    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

// RulebookSource summarises one uploaded book for a ruleset.
type RulebookSource struct {
	Source string `json:"source"`
	Chunks int    `json:"chunks"`
}

// GetRuleset returns a single ruleset by ID.
func (d *DB) GetRuleset(id int64) (*Ruleset, error) {
	var r Ruleset
	err := d.db.QueryRow(
		"SELECT id, name, schema_json, version, gm_context FROM rulesets WHERE id = ?", id,
	).Scan(&r.ID, &r.Name, &r.SchemaJSON, &r.Version, &r.GMContext)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// DeleteRulebookChunks removes all chunks for a ruleset.
func (d *DB) DeleteRulebookChunks(rulesetID int64) error {
	_, err := d.db.Exec("DELETE FROM rulebook_chunks WHERE ruleset_id = ?", rulesetID)
	return err
}

// DeleteRulebookChunksBySource removes chunks for a specific source book within a ruleset.
func (d *DB) DeleteRulebookChunksBySource(rulesetID int64, source string) error {
	_, err := d.db.Exec("DELETE FROM rulebook_chunks WHERE ruleset_id = ? AND source = ?", rulesetID, source)
	return err
}

// ListRulebookSources returns each distinct source uploaded for a ruleset with its chunk count.
func (d *DB) ListRulebookSources(rulesetID int64) ([]RulebookSource, error) {
	rows, err := d.db.Query(
		`SELECT source, COUNT(*) FROM rulebook_chunks WHERE ruleset_id = ? GROUP BY source ORDER BY source`,
		rulesetID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sources []RulebookSource
	for rows.Next() {
		var s RulebookSource
		if err := rows.Scan(&s.Source, &s.Chunks); err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return sources, rows.Err()
}

// CreateRulebookChunks inserts multiple chunks for a ruleset in a single transaction.
func (d *DB) CreateRulebookChunks(rulesetID int64, chunks []RulebookChunk) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(
		"INSERT INTO rulebook_chunks (ruleset_id, source, heading, content) VALUES (?, ?, ?, ?)",
	)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, c := range chunks {
		if _, err := stmt.Exec(rulesetID, c.Source, c.Heading, c.Content); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SearchRulebookChunks returns up to 5 chunks matching query in heading or content (LIKE search).
func (d *DB) SearchRulebookChunks(rulesetID int64, query string) ([]RulebookChunk, error) {
	like := "%" + query + "%"
	rows, err := d.db.Query(
		`SELECT id, ruleset_id, source, heading, content, created_at
		 FROM rulebook_chunks
		 WHERE ruleset_id = ? AND (heading LIKE ? OR content LIKE ?)
		 LIMIT 5`,
		rulesetID, like, like,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var chunks []RulebookChunk
	for rows.Next() {
		var c RulebookChunk
		if err := rows.Scan(&c.ID, &c.RulesetID, &c.Source, &c.Heading, &c.Content, &c.CreatedAt); err != nil {
			return nil, err
		}
		chunks = append(chunks, c)
	}
	return chunks, rows.Err()
}

// encodeFloat32Slice encodes a float32 slice as little-endian bytes.
func encodeFloat32Slice(emb []float32) []byte {
	b := make([]byte, len(emb)*4)
	for i, f := range emb {
		bits := math.Float32bits(f)
		b[i*4] = byte(bits)
		b[i*4+1] = byte(bits >> 8)
		b[i*4+2] = byte(bits >> 16)
		b[i*4+3] = byte(bits >> 24)
	}
	return b
}

// UpsertChunkEmbedding stores the embedding for a chunk by ID.
func (d *DB) UpsertChunkEmbedding(id int64, emb []float32) error {
	_, err := d.db.Exec(
		"UPDATE rulebook_chunks SET embedding = ? WHERE id = ?",
		encodeFloat32Slice(emb), id,
	)
	return err
}

// ListChunksForEmbedding returns chunks that have no embedding yet for a given ruleset.
func (d *DB) ListChunksForEmbedding(rulesetID int64) ([]RulebookChunk, error) {
	rows, err := d.db.Query(
		"SELECT id, ruleset_id, source, heading, content, created_at FROM rulebook_chunks WHERE ruleset_id = ? AND embedding IS NULL",
		rulesetID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RulebookChunk
	for rows.Next() {
		var c RulebookChunk
		if err := rows.Scan(&c.ID, &c.RulesetID, &c.Source, &c.Heading, &c.Content, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListAllChunks returns all chunks for a ruleset, including those with embeddings.
func (d *DB) ListAllChunks(rulesetID int64) ([]RulebookChunk, error) {
	rows, err := d.db.Query(
		"SELECT id, ruleset_id, source, heading, content, embedding, created_at FROM rulebook_chunks WHERE ruleset_id = ?",
		rulesetID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RulebookChunk
	for rows.Next() {
		var c RulebookChunk
		if err := rows.Scan(&c.ID, &c.RulesetID, &c.Source, &c.Heading, &c.Content, &c.Embedding, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
