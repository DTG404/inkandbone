package db

import "database/sql"

type MapToken struct {
	ID         int64   `json:"id"`
	MapID      int64   `json:"map_id"`
	EntityType string  `json:"entity_type"`
	EntityID   int64   `json:"entity_id"`
	Name       string  `json:"name"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
}

func (d *DB) ListMapTokens(mapID int64) ([]MapToken, error) {
	rows, err := d.db.Query(`
		SELECT mt.id, mt.map_id, mt.entity_type, mt.entity_id, mt.x, mt.y,
		  COALESCE(
		    CASE WHEN mt.entity_type = 'character' THEN c.name ELSE sn.name END,
		    ''
		  ) AS name
		FROM map_tokens mt
		LEFT JOIN characters c ON mt.entity_type = 'character' AND c.id = mt.entity_id
		LEFT JOIN session_npcs sn ON mt.entity_type = 'npc' AND sn.id = mt.entity_id
		WHERE mt.map_id = ?`, mapID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MapToken
	for rows.Next() {
		var t MapToken
		if err := rows.Scan(&t.ID, &t.MapID, &t.EntityType, &t.EntityID, &t.X, &t.Y, &t.Name); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (d *DB) PlaceToken(mapID int64, entityType string, entityID int64, x, y float64) (int64, error) {
	res, err := d.db.Exec(
		"INSERT INTO map_tokens (map_id, entity_type, entity_id, x, y) VALUES (?, ?, ?, ?, ?)",
		mapID, entityType, entityID, x, y,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) MoveToken(id int64, x, y float64) error {
	_, err := d.db.Exec("UPDATE map_tokens SET x = ?, y = ? WHERE id = ?", x, y, id)
	return err
}

func (d *DB) RemoveToken(id int64) error {
	_, err := d.db.Exec("DELETE FROM map_tokens WHERE id = ?", id)
	return err
}

func (d *DB) GetToken(id int64) (*MapToken, error) {
	var t MapToken
	err := d.db.QueryRow(`
		SELECT mt.id, mt.map_id, mt.entity_type, mt.entity_id, mt.x, mt.y,
		  COALESCE(
		    CASE WHEN mt.entity_type = 'character' THEN c.name ELSE sn.name END,
		    ''
		  ) AS name
		FROM map_tokens mt
		LEFT JOIN characters c ON mt.entity_type = 'character' AND c.id = mt.entity_id
		LEFT JOIN session_npcs sn ON mt.entity_type = 'npc' AND sn.id = mt.entity_id
		WHERE mt.id = ?`, id,
	).Scan(&t.ID, &t.MapID, &t.EntityType, &t.EntityID, &t.X, &t.Y, &t.Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &t, err
}
