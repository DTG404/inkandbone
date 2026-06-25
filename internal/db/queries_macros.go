package db

type Macro struct {
	ID          int64  `json:"id"`
	CharacterID int64  `json:"character_id"`
	Label       string `json:"label"`
	ActionText  string `json:"action_text"`
	Color       string `json:"color"`
	SortOrder   int    `json:"sort_order"`
	CreatedAt   string `json:"created_at"`
}

func (d *DB) ListMacros(characterID int64) ([]Macro, error) {
	rows, err := d.db.Query(
		`SELECT id, character_id, label, action_text, color, sort_order, created_at
		 FROM character_macros WHERE character_id = ? ORDER BY sort_order ASC`,
		characterID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Macro
	for rows.Next() {
		var m Macro
		if err := rows.Scan(&m.ID, &m.CharacterID, &m.Label, &m.ActionText, &m.Color, &m.SortOrder, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (d *DB) CreateMacro(characterID int64, label, actionText, color string) (int64, error) {
	var maxSort int
	_ = d.db.QueryRow(
		"SELECT COALESCE(MAX(sort_order), -1) FROM character_macros WHERE character_id = ?",
		characterID,
	).Scan(&maxSort)
	res, err := d.db.Exec(
		`INSERT INTO character_macros (character_id, label, action_text, color, sort_order)
		 VALUES (?, ?, ?, ?, ?)`,
		characterID, label, actionText, color, maxSort+1,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) UpdateMacro(id int64, label, actionText, color string) error {
	_, err := d.db.Exec(
		"UPDATE character_macros SET label = ?, action_text = ?, color = ? WHERE id = ?",
		label, actionText, color, id,
	)
	return err
}

func (d *DB) DeleteMacro(id int64) error {
	_, err := d.db.Exec("DELETE FROM character_macros WHERE id = ?", id)
	return err
}

func (d *DB) ReorderMacros(characterID int64, ids []int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	for i, id := range ids {
		if _, err := tx.Exec(
			"UPDATE character_macros SET sort_order = ? WHERE id = ? AND character_id = ?",
			i, id, characterID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}
