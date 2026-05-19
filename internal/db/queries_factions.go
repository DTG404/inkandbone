package db

import "time"

// Faction represents a named faction within a campaign.
type Faction struct {
	ID            int64     `json:"id"`
	CampaignID    int64     `json:"campaign_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	FactionType   string    `json:"faction_type"`
	Influence     int       `json:"influence"`
	ResourcesJSON string    `json:"resources_json"`
	Color         string    `json:"color"`
	CreatedAt     time.Time `json:"created_at"`
}

// CreateFaction inserts a new faction and returns its ID.
func (db *DB) CreateFaction(campaignID int64, name, description, factionType string, influence int, resourcesJSON, color string) (int64, error) {
	res, err := db.db.Exec(
		`INSERT INTO factions (campaign_id, name, description, faction_type, influence, resources_json, color)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		campaignID, name, description, factionType, influence, resourcesJSON, color,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetFaction returns a single faction by ID.
func (db *DB) GetFaction(id int64) (*Faction, error) {
	row := db.db.QueryRow(
		`SELECT id, campaign_id, name, description, faction_type, influence, resources_json, color, created_at
		 FROM factions WHERE id = ?`, id,
	)
	var f Faction
	if err := row.Scan(&f.ID, &f.CampaignID, &f.Name, &f.Description, &f.FactionType, &f.Influence, &f.ResourcesJSON, &f.Color, &f.CreatedAt); err != nil {
		return nil, err
	}
	return &f, nil
}

// ListFactions returns all factions for a campaign.
func (db *DB) ListFactions(campaignID int64) ([]Faction, error) {
	rows, err := db.db.Query(
		`SELECT id, campaign_id, name, description, faction_type, influence, resources_json, color, created_at
		 FROM factions WHERE campaign_id = ? ORDER BY name ASC`,
		campaignID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var factions []Faction
	for rows.Next() {
		var f Faction
		if err := rows.Scan(&f.ID, &f.CampaignID, &f.Name, &f.Description, &f.FactionType, &f.Influence, &f.ResourcesJSON, &f.Color, &f.CreatedAt); err != nil {
			return nil, err
		}
		factions = append(factions, f)
	}
	return factions, rows.Err()
}

// UpdateFaction updates an existing faction's fields.
func (db *DB) UpdateFaction(id int64, name, description, factionType string, influence int, resourcesJSON, color string) error {
	_, err := db.db.Exec(
		`UPDATE factions SET name = ?, description = ?, faction_type = ?, influence = ?, resources_json = ?, color = ? WHERE id = ?`,
		name, description, factionType, influence, resourcesJSON, color, id,
	)
	return err
}

// DeleteFaction removes a faction by ID.
func (db *DB) DeleteFaction(id int64) error {
	_, err := db.db.Exec(`DELETE FROM factions WHERE id = ?`, id)
	return err
}
