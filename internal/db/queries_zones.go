package db

type MapZone struct {
	ID         int64   `json:"id"`
	MapID      int64   `json:"map_id"`
	Name       string  `json:"name"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
	IsRevealed bool    `json:"is_revealed"`
}

func (d *DB) ListMapZones(mapID int64) ([]MapZone, error) {
	rows, err := d.db.Query(
		"SELECT id, map_id, name, x, y, width, height, is_revealed FROM map_zones WHERE map_id = ? ORDER BY id",
		mapID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MapZone
	for rows.Next() {
		var z MapZone
		if err := rows.Scan(&z.ID, &z.MapID, &z.Name, &z.X, &z.Y, &z.Width, &z.Height, &z.IsRevealed); err != nil {
			return nil, err
		}
		out = append(out, z)
	}
	return out, rows.Err()
}

func (d *DB) ListUnrevealedZones(mapID int64) ([]MapZone, error) {
	rows, err := d.db.Query(
		"SELECT id, map_id, name, x, y, width, height, is_revealed FROM map_zones WHERE map_id = ? AND is_revealed = 0",
		mapID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MapZone
	for rows.Next() {
		var z MapZone
		if err := rows.Scan(&z.ID, &z.MapID, &z.Name, &z.X, &z.Y, &z.Width, &z.Height, &z.IsRevealed); err != nil {
			return nil, err
		}
		out = append(out, z)
	}
	return out, rows.Err()
}

func (d *DB) CreateMapZone(mapID int64, name string, x, y, w, h float64) (int64, error) {
	res, err := d.db.Exec(
		"INSERT INTO map_zones (map_id, name, x, y, width, height) VALUES (?, ?, ?, ?, ?, ?)",
		mapID, name, x, y, w, h,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) UpdateMapZone(id int64, name string, x, y, w, h float64) error {
	_, err := d.db.Exec(
		"UPDATE map_zones SET name = ?, x = ?, y = ?, width = ?, height = ? WHERE id = ?",
		name, x, y, w, h, id,
	)
	return err
}

func (d *DB) DeleteMapZone(id int64) error {
	_, err := d.db.Exec("DELETE FROM map_zones WHERE id = ?", id)
	return err
}

func (d *DB) RevealZone(id int64, revealed bool) error {
	val := 0
	if revealed {
		val = 1
	}
	_, err := d.db.Exec("UPDATE map_zones SET is_revealed = ? WHERE id = ?", val, id)
	return err
}

func (d *DB) GetMapZone(id int64) (*MapZone, error) {
	var z MapZone
	err := d.db.QueryRow(
		"SELECT id, map_id, name, x, y, width, height, is_revealed FROM map_zones WHERE id = ?", id,
	).Scan(&z.ID, &z.MapID, &z.Name, &z.X, &z.Y, &z.Width, &z.Height, &z.IsRevealed)
	if err != nil {
		return nil, err
	}
	return &z, nil
}
