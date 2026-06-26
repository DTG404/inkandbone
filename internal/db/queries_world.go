package db

import (
	"database/sql"
	"fmt"
)

// --- World Notes ---

type WorldNote struct {
	ID              int64  `json:"id"`
	CampaignID      int64  `json:"campaign_id"`
	Title           string `json:"title"`
	Content         string `json:"content"`
	Category        string `json:"category"`
	TagsJSON        string `json:"tags_json"`
	PersonalityJSON string `json:"personality_json"`
	IsRevealed      bool   `json:"is_revealed"`
	CreatedAt       string `json:"created_at"`
}

func (d *DB) CreateWorldNote(campaignID int64, title, content, category string) (int64, error) {
	res, err := d.db.Exec(
		"INSERT INTO world_notes (campaign_id, title, content, category) VALUES (?, ?, ?, ?)",
		campaignID, title, content, category,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) UpdateWorldNote(id int64, title, content, tagsJSON string) error {
	var res sql.Result
	var err error
	if tagsJSON != "" {
		res, err = d.db.Exec(
			"UPDATE world_notes SET title = ?, content = ?, tags_json = ? WHERE id = ?",
			title, content, tagsJSON, id,
		)
	} else {
		res, err = d.db.Exec(
			"UPDATE world_notes SET title = ?, content = ? WHERE id = ?",
			title, content, id,
		)
	}
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("world note %d not found", id)
	}
	return nil
}

func (d *DB) GetWorldNote(id int64) (*WorldNote, error) {
	var n WorldNote
	err := d.db.QueryRow(
		"SELECT id, campaign_id, title, content, category, tags_json, personality_json, is_revealed, created_at FROM world_notes WHERE id = ?",
		id,
	).Scan(&n.ID, &n.CampaignID, &n.Title, &n.Content, &n.Category, &n.TagsJSON, &n.PersonalityJSON, &n.IsRevealed, &n.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (d *DB) SearchWorldNotes(campaignID int64, query, category, tag string, revealed *bool) ([]WorldNote, error) {
	q := "SELECT id, campaign_id, title, content, category, tags_json, personality_json, is_revealed, created_at FROM world_notes WHERE campaign_id = ?"
	args := []any{campaignID}
	if query != "" {
		q += " AND (title LIKE ? OR content LIKE ?)"
		like := "%" + query + "%"
		args = append(args, like, like)
	}
	if category != "" {
		q += " AND category = ?"
		args = append(args, category)
	}
	if tag != "" {
		q += " AND tags_json LIKE ?"
		args = append(args, `%"`+tag+`"%`)
	}
	if revealed != nil {
		if *revealed {
			q += " AND is_revealed = 1"
		} else {
			q += " AND is_revealed = 0"
		}
	}
	q += " ORDER BY title"
	rows, err := d.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WorldNote
	for rows.Next() {
		var n WorldNote
		if err := rows.Scan(&n.ID, &n.CampaignID, &n.Title, &n.Content, &n.Category, &n.TagsJSON, &n.PersonalityJSON, &n.IsRevealed, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// ListRecentWorldNotes returns the most recent n world notes for a campaign, ordered by created_at DESC.
func (d *DB) ListRecentWorldNotes(campaignID int64, limit int) ([]WorldNote, error) {
	rows, err := d.db.Query(
		`SELECT id, campaign_id, title, content, category, tags_json, personality_json, is_revealed, created_at
		 FROM world_notes WHERE campaign_id = ? ORDER BY created_at DESC LIMIT ?`,
		campaignID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WorldNote
	for rows.Next() {
		var n WorldNote
		if err := rows.Scan(&n.ID, &n.CampaignID, &n.Title, &n.Content, &n.Category, &n.TagsJSON, &n.PersonalityJSON, &n.IsRevealed, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (d *DB) FindWorldNoteByTitle(campaignID int64, title string) (*WorldNote, error) {
	row := d.db.QueryRow(
		`SELECT id, campaign_id, title, content, category, tags_json, personality_json, is_revealed, created_at
		 FROM world_notes WHERE campaign_id = ? AND LOWER(title) = LOWER(?) LIMIT 1`,
		campaignID, title,
	)
	var wn WorldNote
	err := row.Scan(&wn.ID, &wn.CampaignID, &wn.Title, &wn.Content, &wn.Category, &wn.TagsJSON, &wn.PersonalityJSON, &wn.IsRevealed, &wn.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &wn, nil
}

func (d *DB) UpdateWorldNotePersonality(noteID int64, personalityJSON string) error {
	_, err := d.db.Exec(
		`UPDATE world_notes SET personality_json = ? WHERE id = ?`,
		personalityJSON, noteID,
	)
	return err
}

func (d *DB) PatchWorldNoteRevealed(id int64, revealed bool) error {
	val := 0
	if revealed {
		val = 1
	}
	_, err := d.db.Exec("UPDATE world_notes SET is_revealed = ? WHERE id = ?", val, id)
	return err
}

// --- Maps ---

type Map struct {
	ID         int64  `json:"id"`
	CampaignID int64  `json:"campaign_id"`
	Name       string `json:"name"`
	ImagePath  string `json:"image_path"`
	CreatedAt  string `json:"created_at"`
}

func (d *DB) CreateMap(campaignID int64, name, imagePath string) (int64, error) {
	res, err := d.db.Exec(
		"INSERT INTO maps (campaign_id, name, image_path) VALUES (?, ?, ?)",
		campaignID, name, imagePath,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) GetMap(id int64) (*Map, error) {
	m := &Map{}
	err := d.db.QueryRow(
		"SELECT id, campaign_id, name, image_path, created_at FROM maps WHERE id = ?", id,
	).Scan(&m.ID, &m.CampaignID, &m.Name, &m.ImagePath, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

func (d *DB) ListMaps(campaignID int64) ([]Map, error) {
	rows, err := d.db.Query(
		"SELECT id, campaign_id, name, image_path, created_at FROM maps WHERE campaign_id = ? ORDER BY created_at",
		campaignID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Map
	for rows.Next() {
		var m Map
		if err := rows.Scan(&m.ID, &m.CampaignID, &m.Name, &m.ImagePath, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (d *DB) GetLatestMap(campaignID int64) (*Map, error) {
	m := &Map{}
	err := d.db.QueryRow(
		"SELECT id, campaign_id, name, image_path, created_at FROM maps WHERE campaign_id = ? ORDER BY id DESC LIMIT 1",
		campaignID,
	).Scan(&m.ID, &m.CampaignID, &m.Name, &m.ImagePath, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

// --- Map Pins ---

type MapPin struct {
	ID        int64   `json:"id"`
	MapID     int64   `json:"map_id"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Label     string  `json:"label"`
	Note      string  `json:"note"`
	Color     string  `json:"color"`
	CreatedAt string  `json:"created_at"`
}

func (d *DB) AddMapPin(mapID int64, x, y float64, label, note, color string) (int64, error) {
	res, err := d.db.Exec(
		"INSERT INTO map_pins (map_id, x, y, label, note, color) VALUES (?, ?, ?, ?, ?, ?)",
		mapID, x, y, label, note, color,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) ListMapPins(mapID int64) ([]MapPin, error) {
	rows, err := d.db.Query(
		"SELECT id, map_id, x, y, label, note, color, created_at FROM map_pins WHERE map_id = ?",
		mapID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MapPin
	for rows.Next() {
		var p MapPin
		if err := rows.Scan(&p.ID, &p.MapID, &p.X, &p.Y, &p.Label, &p.Note, &p.Color, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// --- Dice Rolls ---

type DiceRoll struct {
	ID            int64  `json:"id"`
	SessionID     int64  `json:"session_id"`
	Expression    string `json:"expression"`
	Result        int    `json:"result"`
	BreakdownJSON string `json:"breakdown_json"`
	CreatedAt     string `json:"created_at"`
}

func (d *DB) LogDiceRoll(sessionID int64, expression string, result int, breakdownJSON string) (int64, error) {
	res, err := d.db.Exec(
		"INSERT INTO dice_rolls (session_id, expression, result, breakdown_json) VALUES (?, ?, ?, ?)",
		sessionID, expression, result, breakdownJSON,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) ListDiceRolls(sessionID int64) ([]DiceRoll, error) {
	rows, err := d.db.Query(
		"SELECT id, session_id, expression, result, breakdown_json, created_at FROM dice_rolls WHERE session_id = ? ORDER BY created_at DESC",
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DiceRoll
	for rows.Next() {
		var r DiceRoll
		if err := rows.Scan(&r.ID, &r.SessionID, &r.Expression, &r.Result, &r.BreakdownJSON, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
