package db

// Adventure represents a story arc/chapter within a campaign.
type Adventure struct {
	ID          int64  `json:"id"`
	CampaignID  int64  `json:"campaign_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	SortOrder   int    `json:"sort_order"`
	CreatedAt   string `json:"created_at"`
}

// CreateAdventure inserts a new adventure and returns its ID.
func (d *DB) CreateAdventure(campaignID int64, title, description, status string, sortOrder int) (int64, error) {
	if status == "" {
		status = "upcoming"
	}
	res, err := d.db.Exec(
		`INSERT INTO adventures (campaign_id, title, description, status, sort_order)
		 VALUES (?, ?, ?, ?, ?)`,
		campaignID, title, description, status, sortOrder,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetAdventure returns a single adventure by ID.
func (d *DB) GetAdventure(id int64) (*Adventure, error) {
	row := d.db.QueryRow(
		`SELECT id, campaign_id, title, description, status, sort_order, created_at
		 FROM adventures WHERE id = ?`, id,
	)
	var a Adventure
	if err := row.Scan(&a.ID, &a.CampaignID, &a.Title, &a.Description, &a.Status, &a.SortOrder, &a.CreatedAt); err != nil {
		return nil, err
	}
	return &a, nil
}

// ListAdventures returns all adventures for a campaign, ordered by sort_order.
func (d *DB) ListAdventures(campaignID int64) ([]Adventure, error) {
	rows, err := d.db.Query(
		`SELECT id, campaign_id, title, description, status, sort_order, created_at
		 FROM adventures WHERE campaign_id = ? ORDER BY sort_order ASC, id ASC`,
		campaignID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var adventures []Adventure
	for rows.Next() {
		var a Adventure
		if err := rows.Scan(&a.ID, &a.CampaignID, &a.Title, &a.Description, &a.Status, &a.SortOrder, &a.CreatedAt); err != nil {
			return nil, err
		}
		adventures = append(adventures, a)
	}
	return adventures, rows.Err()
}

// UpdateAdventure updates an existing adventure's fields.
func (d *DB) UpdateAdventure(id int64, title, description, status string, sortOrder int) error {
	_, err := d.db.Exec(
		`UPDATE adventures SET title = ?, description = ?, status = ?, sort_order = ? WHERE id = ?`,
		title, description, status, sortOrder, id,
	)
	return err
}

// DeleteAdventure removes an adventure by ID.
func (d *DB) DeleteAdventure(id int64) error {
	// Set adventure_id to NULL for all sessions in this adventure before deleting
	_, err := d.db.Exec(`UPDATE sessions SET adventure_id = NULL WHERE adventure_id = ?`, id)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(`DELETE FROM adventures WHERE id = ?`, id)
	return err
}

// ListSessionsByAdventure returns sessions for a given adventure.
func (d *DB) ListSessionsByAdventure(adventureID int64) ([]Session, error) {
	rows, err := d.db.Query(
		`SELECT id, campaign_id, title, date, summary, notes, scene_tags, adventure_id, created_at
		 FROM sessions WHERE adventure_id = ? ORDER BY date ASC, id ASC`,
		adventureID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.CampaignID, &s.Title, &s.Date, &s.Summary, &s.Notes, &s.SceneTags, &s.AdventureID, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// SetSessionAdventure updates the adventure_id for a session.
// Pass nil to remove the session from an adventure.
func (d *DB) SetSessionAdventure(sessionID int64, adventureID *int64) error {
	_, err := d.db.Exec(
		`UPDATE sessions SET adventure_id = ? WHERE id = ?`,
		adventureID, sessionID,
	)
	return err
}
