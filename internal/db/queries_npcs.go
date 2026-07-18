package db

import "database/sql"

// SessionNPC represents an NPC tracked within a session.
type SessionNPC struct {
	ID        int64  `json:"id"`
	SessionID int64  `json:"session_id"`
	Name      string `json:"name"`
	Note      string `json:"note"`
	CreatedAt string `json:"created_at"`
}

func (d *DB) GetSessionNPC(id int64) (*SessionNPC, error) {
	var n SessionNPC
	err := d.db.QueryRow(
		"SELECT id, session_id, name, note, created_at FROM session_npcs WHERE id = ?", id,
	).Scan(&n.ID, &n.SessionID, &n.Name, &n.Note, &n.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (d *DB) ListSessionNPCs(sessionID int64) ([]SessionNPC, error) {
	rows, err := d.db.Query(
		"SELECT id, session_id, name, note, created_at FROM session_npcs WHERE session_id = ? ORDER BY created_at",
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SessionNPC
	for rows.Next() {
		var n SessionNPC
		if err := rows.Scan(&n.ID, &n.SessionID, &n.Name, &n.Note, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (d *DB) CreateSessionNPC(sessionID int64, name, note string) (SessionNPC, error) {
	res, err := d.db.Exec(
		"INSERT INTO session_npcs (session_id, name, note) VALUES (?, ?, ?)",
		sessionID, name, note,
	)
	if err != nil {
		return SessionNPC{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return SessionNPC{}, err
	}
	var n SessionNPC
	err = d.db.QueryRow(
		"SELECT id, session_id, name, note, created_at FROM session_npcs WHERE id = ?", id,
	).Scan(&n.ID, &n.SessionID, &n.Name, &n.Note, &n.CreatedAt)
	return n, err
}

func (d *DB) UpdateSessionNPC(id int64, note string) error {
	_, err := d.db.Exec("UPDATE session_npcs SET note = ? WHERE id = ?", note, id)
	return err
}

func (d *DB) DeleteSessionNPC(id int64) error {
	_, err := d.db.Exec("DELETE FROM session_npcs WHERE id = ?", id)
	return err
}
