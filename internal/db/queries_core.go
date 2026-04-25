package db

import (
	"database/sql"
	"fmt"
)

// --- Settings ---

func (d *DB) GetSetting(key string) (string, error) {
	var value string
	err := d.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (d *DB) SetSetting(key, value string) error {
	_, err := d.db.Exec(
		"INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
		key, value,
	)
	return err
}

// --- Rulesets ---

type Ruleset struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	SchemaJSON string `json:"schema_json"`
	Version    string `json:"version"`
	GMContext  string `json:"gm_context"`
}

func (d *DB) CreateRuleset(name, schemaJSON, version string) (int64, error) {
	res, err := d.db.Exec(
		"INSERT INTO rulesets (name, schema_json, version) VALUES (?, ?, ?)",
		name, schemaJSON, version,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) GetRulesetByName(name string) (*Ruleset, error) {
	r := &Ruleset{}
	err := d.db.QueryRow(
		"SELECT id, name, schema_json, version, gm_context FROM rulesets WHERE name = ?", name,
	).Scan(&r.ID, &r.Name, &r.SchemaJSON, &r.Version, &r.GMContext)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return r, err
}

func (d *DB) ListRulesets() ([]Ruleset, error) {
	rows, err := d.db.Query("SELECT id, name, schema_json, version, gm_context FROM rulesets ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Ruleset
	for rows.Next() {
		var r Ruleset
		if err := rows.Scan(&r.ID, &r.Name, &r.SchemaJSON, &r.Version, &r.GMContext); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// --- Campaigns ---

type Campaign struct {
	ID                    int64  `json:"id"`
	RulesetID             int64  `json:"ruleset_id"`
	Name                  string `json:"name"`
	Description           string `json:"description"`
	Active                bool   `json:"active"`
	ChronicleNight        int    `json:"chronicle_night"`
	ChronicleNightStartDOW int   `json:"chronicle_night_start_dow"`
	CreatedAt             string `json:"created_at"`
}

func (d *DB) CreateCampaign(rulesetID int64, name, description string) (int64, error) {
	res, err := d.db.Exec(
		"INSERT INTO campaigns (ruleset_id, name, description) VALUES (?, ?, ?)",
		rulesetID, name, description,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) GetCampaign(id int64) (*Campaign, error) {
	c := &Campaign{}
	var active int
	err := d.db.QueryRow(
		"SELECT id, ruleset_id, name, description, active, chronicle_night, chronicle_night_start_dow, created_at FROM campaigns WHERE id = ?", id,
	).Scan(&c.ID, &c.RulesetID, &c.Name, &c.Description, &active, &c.ChronicleNight, &c.ChronicleNightStartDOW, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.Active = active == 1
	return c, nil
}

func (d *DB) ListCampaigns() ([]Campaign, error) {
	rows, err := d.db.Query(
		"SELECT id, ruleset_id, name, description, active, chronicle_night, chronicle_night_start_dow, created_at FROM campaigns ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Campaign
	for rows.Next() {
		var c Campaign
		var active int
		if err := rows.Scan(&c.ID, &c.RulesetID, &c.Name, &c.Description, &active, &c.ChronicleNight, &c.ChronicleNightStartDOW, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.Active = active == 1
		out = append(out, c)
	}
	return out, rows.Err()
}

func (d *DB) UpdateCampaignChronicleNight(id int64, night int) error {
	if night < 1 {
		night = 1
	}
	res, err := d.db.Exec("UPDATE campaigns SET chronicle_night = ? WHERE id = ?", night, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("campaign %d not found", id)
	}
	return nil
}

func (d *DB) SetCampaignChronicleNightStartDOW(id int64, dow int) error {
	_, err := d.db.Exec("UPDATE campaigns SET chronicle_night_start_dow = ? WHERE id = ?", dow, id)
	return err
}

// --- Characters ---

type Character struct {
	ID              int64  `json:"id"`
	CampaignID      int64  `json:"campaign_id"`
	Name            string `json:"name"`
	DataJSON        string `json:"data_json"`
	PortraitPath    string `json:"portrait_path"` // NOT NULL DEFAULT '' in schema; never nil
	CurrencyBalance int64  `json:"currency_balance"`
	CurrencyLabel   string `json:"currency_label"`
	CreatedAt       string `json:"created_at"`
}

func (d *DB) CreateCharacter(campaignID int64, name string) (int64, error) {
	res, err := d.db.Exec(
		"INSERT INTO characters (campaign_id, name) VALUES (?, ?)",
		campaignID, name,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) GetCharacter(id int64) (*Character, error) {
	c := &Character{}
	err := d.db.QueryRow(
		"SELECT id, campaign_id, name, data_json, portrait_path, currency_balance, currency_label, created_at FROM characters WHERE id = ?", id,
	).Scan(&c.ID, &c.CampaignID, &c.Name, &c.DataJSON, &c.PortraitPath, &c.CurrencyBalance, &c.CurrencyLabel, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func (d *DB) UpdateCharacterData(id int64, dataJSON string) error {
	res, err := d.db.Exec("UPDATE characters SET data_json = ? WHERE id = ?", dataJSON, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("character %d not found", id)
	}
	return nil
}

func (d *DB) UpdateCharacterPortrait(id int64, portraitPath string) error {
	res, err := d.db.Exec("UPDATE characters SET portrait_path = ? WHERE id = ?", portraitPath, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("character %d not found", id)
	}
	return nil
}

func (d *DB) UpdateCharacterCurrencyBalance(id int64, balance int64) error {
	res, err := d.db.Exec("UPDATE characters SET currency_balance = ? WHERE id = ?", balance, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("character %d not found", id)
	}
	return nil
}

func (d *DB) UpdateCharacterCurrencyLabel(id int64, label string) error {
	res, err := d.db.Exec("UPDATE characters SET currency_label = ? WHERE id = ?", label, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("character %d not found", id)
	}
	return nil
}

func (d *DB) ListCharacters(campaignID int64) ([]Character, error) {
	rows, err := d.db.Query(
		"SELECT id, campaign_id, name, data_json, portrait_path, currency_balance, currency_label, created_at FROM characters WHERE campaign_id = ? ORDER BY name",
		campaignID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Character
	for rows.Next() {
		var c Character
		if err := rows.Scan(&c.ID, &c.CampaignID, &c.Name, &c.DataJSON, &c.PortraitPath, &c.CurrencyBalance, &c.CurrencyLabel, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// DeleteCharacter removes a character and all its items.
func (d *DB) DeleteCharacter(id int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM items WHERE character_id = ?", id); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM characters WHERE id = ?", id); err != nil {
		return err
	}
	return tx.Commit()
}

// CloseCampaign sets active = 0 for the given campaign.
// Returns an error if the campaign does not exist.
// Idempotent: closing an already-closed campaign is a no-op.
func (d *DB) CloseCampaign(id int64) error {
	res, err := d.db.Exec("UPDATE campaigns SET active = 0 WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("campaign %d not found", id)
	}
	return nil
}

// ReopenCampaign sets active = 1 for the given campaign.
// Returns an error if the campaign does not exist.
// Idempotent: reopening an already-open campaign is a no-op.
func (d *DB) ReopenCampaign(id int64) error {
	res, err := d.db.Exec("UPDATE campaigns SET active = 1 WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("campaign %d not found", id)
	}
	return nil
}

// CampaignStats holds row counts for the confirmation message in delete_campaign.
type CampaignStats struct {
	Sessions   int
	Characters int
	WorldNotes int
	Maps       int
}

func (d *DB) GetCampaignStats(id int64) (CampaignStats, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return CampaignStats{}, err
	}
	defer tx.Rollback()

	var s CampaignStats
	if err := tx.QueryRow("SELECT COUNT(*) FROM sessions WHERE campaign_id = ?", id).Scan(&s.Sessions); err != nil {
		return s, err
	}
	if err := tx.QueryRow("SELECT COUNT(*) FROM characters WHERE campaign_id = ?", id).Scan(&s.Characters); err != nil {
		return s, err
	}
	if err := tx.QueryRow("SELECT COUNT(*) FROM world_notes WHERE campaign_id = ?", id).Scan(&s.WorldNotes); err != nil {
		return s, err
	}
	if err := tx.QueryRow("SELECT COUNT(*) FROM maps WHERE campaign_id = ?", id).Scan(&s.Maps); err != nil {
		return s, err
	}
	return s, tx.Commit()
}

func (d *DB) DeleteCampaign(id int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmts := []string{
		`DELETE FROM dice_rolls WHERE session_id IN (SELECT id FROM sessions WHERE campaign_id = ?)`,
		`DELETE FROM messages WHERE session_id IN (SELECT id FROM sessions WHERE campaign_id = ?)`,
		`DELETE FROM combatants WHERE encounter_id IN (SELECT id FROM combat_encounters WHERE session_id IN (SELECT id FROM sessions WHERE campaign_id = ?))`,
		`DELETE FROM combat_encounters WHERE session_id IN (SELECT id FROM sessions WHERE campaign_id = ?)`,
		`DELETE FROM sessions WHERE campaign_id = ?`,
		`DELETE FROM world_notes WHERE campaign_id = ?`,
		`DELETE FROM map_pins WHERE map_id IN (SELECT id FROM maps WHERE campaign_id = ?)`,
		`DELETE FROM maps WHERE campaign_id = ?`,
		`DELETE FROM characters WHERE campaign_id = ?`,
		`DELETE FROM campaigns WHERE id = ?`,
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt, id); err != nil {
			return fmt.Errorf("delete campaign %d: %w", id, err)
		}
	}
	return tx.Commit()
}
