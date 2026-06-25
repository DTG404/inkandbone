package db

import (
	"database/sql"
	"errors"
	"fmt"
)

type CombatEncounter struct {
	ID               int64  `json:"id"`
	SessionID        int64  `json:"session_id"`
	Name             string `json:"name"`
	Active           bool   `json:"active"`
	ActiveTurnIndex  int    `json:"active_turn_index"`
	CreatedAt        string `json:"created_at"`
}

func (d *DB) CreateEncounter(sessionID int64, name string) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec("UPDATE combat_encounters SET active = 0 WHERE session_id = ? AND active = 1", sessionID); err != nil {
		return 0, fmt.Errorf("deactivate existing encounter: %w", err)
	}
	res, err := tx.Exec(
		"INSERT INTO combat_encounters (session_id, name) VALUES (?, ?)",
		sessionID, name,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (d *DB) GetActiveEncounter(sessionID int64) (*CombatEncounter, error) {
	e := &CombatEncounter{}
	var active int
	err := d.db.QueryRow(
		"SELECT id, session_id, name, active, active_turn_index, created_at FROM combat_encounters WHERE session_id = ? AND active = 1",
		sessionID,
	).Scan(&e.ID, &e.SessionID, &e.Name, &active, &e.ActiveTurnIndex, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	e.Active = active == 1
	return e, nil
}

func (d *DB) EndEncounter(id int64) error {
	_, err := d.db.Exec("UPDATE combat_encounters SET active = 0 WHERE id = ?", id)
	return err
}

// AdvanceTurn increments active_turn_index modulo the number of combatants.
// Returns the new index. Returns 0 if the encounter has no combatants.
// Returns an error containing "not found" if the encounter does not exist.
func (d *DB) AdvanceTurn(encounterID int64) (int, error) {
	var current int
	err := d.db.QueryRow("SELECT active_turn_index FROM combat_encounters WHERE id = ?", encounterID).Scan(&current)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("combat encounter %d not found", encounterID)
	}
	if err != nil {
		return 0, err
	}
	var count int
	if err := d.db.QueryRow("SELECT COUNT(*) FROM combatants WHERE encounter_id = ?", encounterID).Scan(&count); err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, nil
	}
	next := (current + 1) % count
	_, err = d.db.Exec("UPDATE combat_encounters SET active_turn_index = ? WHERE id = ?", next, encounterID)
	return next, err
}

// --- Combatants ---

type Combatant struct {
	ID                   int64  `json:"id"`
	EncounterID          int64  `json:"encounter_id"`
	CharacterID          *int64 `json:"character_id"`
	Name                 string `json:"name"`
	Initiative           int    `json:"initiative"`
	HPCurrent            int    `json:"hp_current"`
	HPMax                int    `json:"hp_max"`
	ConditionsJSON       string `json:"conditions_json"`
	IsPlayer             bool   `json:"is_player"`
	DamageSuperficial    int    `json:"damage_superficial"`
	DamageAggravated     int    `json:"damage_aggravated"`
	WillpowerSuperficial int    `json:"willpower_superficial"`
	WillpowerAggravated  int    `json:"willpower_aggravated"`
	Hunger               int    `json:"hunger"`
	SortOrder            int    `json:"sort_order"`
}

func (d *DB) AddCombatant(encounterID int64, name string, initiative, hpMax int, isPlayer bool, characterID *int64) (int64, error) {
	var maxSort int
	_ = d.db.QueryRow(
		"SELECT COALESCE(MAX(sort_order), -1) FROM combatants WHERE encounter_id = ?",
		encounterID,
	).Scan(&maxSort)
	res, err := d.db.Exec(
		`INSERT INTO combatants
		 (encounter_id, character_id, name, initiative, hp_current, hp_max, is_player, sort_order)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		encounterID, characterID, name, initiative, hpMax, hpMax, boolToInt(isPlayer), maxSort+1,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) UpdateCombatant(id int64, hpCurrent int, conditionsJSON string) error {
	_, err := d.db.Exec(
		"UPDATE combatants SET hp_current = ?, conditions_json = ? WHERE id = ?",
		hpCurrent, conditionsJSON, id,
	)
	return err
}

func (d *DB) PatchCombatantInitiative(id int64, initiative int) error {
	_, err := d.db.Exec("UPDATE combatants SET initiative = ? WHERE id = ?", initiative, id)
	return err
}

func (d *DB) ReorderCombatants(encounterID int64, ids []int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	for i, id := range ids {
		if _, err := tx.Exec(
			"UPDATE combatants SET sort_order = ? WHERE id = ? AND encounter_id = ?",
			i, id, encounterID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *DB) ListCombatants(encounterID int64) ([]Combatant, error) {
	rows, err := d.db.Query(
		`SELECT id, encounter_id, character_id, name, initiative, hp_current, hp_max,
		        conditions_json, is_player,
		        damage_superficial, damage_aggravated, willpower_superficial, willpower_aggravated,
		        hunger, sort_order
		 FROM combatants WHERE encounter_id = ? ORDER BY sort_order ASC`,
		encounterID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Combatant
	for rows.Next() {
		var c Combatant
		var isPlayer int
		if err := rows.Scan(
			&c.ID, &c.EncounterID, &c.CharacterID, &c.Name,
			&c.Initiative, &c.HPCurrent, &c.HPMax, &c.ConditionsJSON, &isPlayer,
			&c.DamageSuperficial, &c.DamageAggravated,
			&c.WillpowerSuperficial, &c.WillpowerAggravated,
			&c.Hunger, &c.SortOrder,
		); err != nil {
			return nil, err
		}
		c.IsPlayer = isPlayer == 1
		out = append(out, c)
	}
	return out, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// UpdateCombatantVtMDamage applies VtM V5 damage to a combatant.
// isVampire=true halves superficial damage (round up).
// Superficial overflow beyond HPMax converts to aggravated.
func (d *DB) UpdateCombatantVtMDamage(id int64, superficialIn, aggravatedIn int, isVampire bool) error {
	var cur Combatant
	var isPlayer int
	err := d.db.QueryRow(
		`SELECT id, encounter_id, character_id, name, initiative, hp_current, hp_max,
		        conditions_json, is_player,
		        damage_superficial, damage_aggravated, willpower_superficial, willpower_aggravated, hunger
		 FROM combatants WHERE id = ?`, id,
	).Scan(
		&cur.ID, &cur.EncounterID, &cur.CharacterID, &cur.Name,
		&cur.Initiative, &cur.HPCurrent, &cur.HPMax, &cur.ConditionsJSON, &isPlayer,
		&cur.DamageSuperficial, &cur.DamageAggravated,
		&cur.WillpowerSuperficial, &cur.WillpowerAggravated, &cur.Hunger,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("combatant %d not found", id)
		}
		return err
	}
	cur.IsPlayer = isPlayer == 1

	applied := superficialIn
	if isVampire && applied > 0 {
		applied = (applied + 1) / 2
	}
	newSuperficial := cur.DamageSuperficial + applied

	newAggravated := cur.DamageAggravated + aggravatedIn
	if newSuperficial > cur.HPMax {
		overflow := newSuperficial - cur.HPMax
		newSuperficial = cur.HPMax
		newAggravated += overflow
	}
	if newAggravated > cur.HPMax {
		newAggravated = cur.HPMax
	}

	_, err = d.db.Exec(
		`UPDATE combatants SET damage_superficial = ?, damage_aggravated = ? WHERE id = ?`,
		newSuperficial, newAggravated, id,
	)
	return err
}
