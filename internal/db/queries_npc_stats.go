package db

// NpcStat represents a reusable NPC combat stat block within a campaign.
type NpcStat struct {
	ID            int64  `json:"id"`
	CampaignID    int64  `json:"campaign_id"`
	Name          string `json:"name"`
	Role          string `json:"role"`
	DataJSON      string `json:"data_json"`
	HPMax         int    `json:"hp_max"`
	ArmorClass    *int   `json:"armor_class"`
	InitiativeMod int    `json:"initiative_mod"`
	Skills        string `json:"skills"`
	Abilities     string `json:"abilities"`
	Loot          string `json:"loot"`
	Notes         string `json:"notes"`
	CreatedAt     string `json:"created_at"`
}

// CreateNpcStat inserts a new NPC stat block and returns its ID.
func (d *DB) CreateNpcStat(campaignID int64, name, role, dataJSON string, hpMax int, armorClass *int, initiativeMod int, skills, abilities, loot, notes string) (int64, error) {
	res, err := d.db.Exec(
		`INSERT INTO npc_stats (campaign_id, name, role, data_json, hp_max, armor_class, initiative_mod, skills, abilities, loot, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		campaignID, name, role, dataJSON, hpMax, armorClass, initiativeMod, skills, abilities, loot, notes,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetNpcStat returns a single NPC stat block by ID.
func (d *DB) GetNpcStat(id int64) (*NpcStat, error) {
	row := d.db.QueryRow(
		`SELECT id, campaign_id, name, role, data_json, hp_max, armor_class, initiative_mod, skills, abilities, loot, notes, created_at
		 FROM npc_stats WHERE id = ?`, id,
	)
	var n NpcStat
	if err := row.Scan(&n.ID, &n.CampaignID, &n.Name, &n.Role, &n.DataJSON, &n.HPMax, &n.ArmorClass, &n.InitiativeMod, &n.Skills, &n.Abilities, &n.Loot, &n.Notes, &n.CreatedAt); err != nil {
		return nil, err
	}
	return &n, nil
}

// ListNpcStats returns all NPC stat blocks for a campaign.
func (d *DB) ListNpcStats(campaignID int64) ([]NpcStat, error) {
	rows, err := d.db.Query(
		`SELECT id, campaign_id, name, role, data_json, hp_max, armor_class, initiative_mod, skills, abilities, loot, notes, created_at
		 FROM npc_stats WHERE campaign_id = ? ORDER BY name ASC`,
		campaignID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []NpcStat
	for rows.Next() {
		var n NpcStat
		if err := rows.Scan(&n.ID, &n.CampaignID, &n.Name, &n.Role, &n.DataJSON, &n.HPMax, &n.ArmorClass, &n.InitiativeMod, &n.Skills, &n.Abilities, &n.Loot, &n.Notes, &n.CreatedAt); err != nil {
			return nil, err
		}
		stats = append(stats, n)
	}
	return stats, rows.Err()
}

// UpdateNpcStats updates an existing NPC stat block's fields.
func (d *DB) UpdateNpcStats(id int64, name, role, dataJSON string, hpMax int, armorClass *int, initiativeMod int, skills, abilities, loot, notes string) error {
	_, err := d.db.Exec(
		`UPDATE npc_stats SET name = ?, role = ?, data_json = ?, hp_max = ?, armor_class = ?, initiative_mod = ?, skills = ?, abilities = ?, loot = ?, notes = ? WHERE id = ?`,
		name, role, dataJSON, hpMax, armorClass, initiativeMod, skills, abilities, loot, notes, id,
	)
	return err
}

// DeleteNpcStat removes an NPC stat block by ID.
func (d *DB) DeleteNpcStat(id int64) error {
	_, err := d.db.Exec(`DELETE FROM npc_stats WHERE id = ?`, id)
	return err
}
