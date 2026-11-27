package db

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
)

func (d *DB) UndoTurnChange(session, id int64) (TurnChange, error) {
	var empty TurnChange
	tx, err := d.db.Begin()
	if err != nil {
		return empty, err
	}
	defer tx.Rollback()
	c, err := readTurnChange(tx.QueryRow("SELECT "+changeColumns+" FROM turn_changes c JOIN messages m ON m.id=c.message_id WHERE c.id=? AND m.session_id=? AND m.role='assistant' AND m.whisper=0", id, session))
	if err != nil {
		return empty, err
	}
	if c.CorrectedAt != nil {
		return c, nil
	}
	if !c.CanUndo {
		return empty, ErrChangeConflict
	}
	if err := validateTurnOwner(tx, TurnSource{session, c.MessageID}, c.Kind, *c.OwnerID); err != nil {
		return empty, err
	}
	if err := reverseChange(tx, c); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return empty, ErrChangeConflict
		}
		return empty, err
	}
	var stamp string
	if err = tx.QueryRow("UPDATE turn_changes SET corrected_at=CURRENT_TIMESTAMP,superseded=0 WHERE id=? RETURNING corrected_at", id).Scan(&stamp); err != nil {
		return empty, err
	}
	c.CanUndo, c.CorrectedAt = false, &stamp
	return c, tx.Commit()
}

func reverseChange(tx *sql.Tx, c TurnChange) error {
	owner, id := *c.OwnerID, *c.EntityID
	if c.Kind == "stat" || c.Kind == "currency" {
		if owner != id || c.Operation != "set" {
			return ErrChangeConflict
		}
	}
	switch c.Kind {
	case "stat":
		var raw string
		if err := tx.QueryRow("SELECT data_json FROM characters WHERE id=?", id).Scan(&raw); err != nil {
			return err
		}
		var live map[string]json.RawMessage
		if err := json.Unmarshal([]byte(raw), &live); err != nil {
			return err
		}
		if !bytes.Equal(live[c.Field], c.After) {
			return ErrChangeConflict
		}
		if c.Before == nil {
			delete(live, c.Field)
		} else {
			live[c.Field] = c.Before
		}
		dst, err := json.Marshal(live)
		if err != nil {
			return err
		}
		_, err = tx.Exec("UPDATE characters SET data_json=? WHERE id=?", string(dst), id)
		return err
	case "currency":
		var before, after int64
		if json.Unmarshal(c.Before, &before) != nil || json.Unmarshal(c.After, &after) != nil {
			return ErrChangeConflict
		}
		return expectOne(tx.Exec("UPDATE characters SET currency_balance=? WHERE id=? AND currency_balance=?", before, id, after))
	}
	spec, ok := turnEntities[c.Kind]
	if !ok {
		return ErrChangeConflict
	}
	switch c.Operation {
	case "set":
		if c.Kind != "objective" || c.Field != "status" {
			return ErrChangeConflict
		}
		var before, after string
		if json.Unmarshal(c.Before, &before) != nil || json.Unmarshal(c.After, &after) != nil {
			return ErrChangeConflict
		}
		return expectOne(tx.Exec("UPDATE objectives SET status=? WHERE campaign_id=? AND id=? AND status=?", before, owner, id, after))
	case "add":
		live, err := spec.snapshotRow(tx, owner, id)
		if err != nil {
			return err
		}
		if !bytes.Equal(live, c.After) {
			return ErrChangeConflict
		}
		if c.Kind == "objective" {
			var dependents bool
			if err := tx.QueryRow("SELECT EXISTS (SELECT 1 FROM objectives WHERE parent_id=?)", id).Scan(&dependents); err != nil {
				return err
			}
			if dependents {
				return ErrChangeConflict
			}
		}
		return expectOne(tx.Exec("DELETE FROM "+spec.table+" WHERE id=?", id))
	case "remove":
		var taken bool
		if err := tx.QueryRow("SELECT EXISTS (SELECT 1 FROM "+spec.table+" WHERE id=?)", id).Scan(&taken); err != nil {
			return err
		}
		if taken {
			return ErrChangeConflict
		}
		exists, err := spec.named(tx, owner, c.Label)
		if err != nil {
			return err
		}
		if exists {
			return ErrChangeConflict
		}
		var row struct {
			Name, Description, Note string
			Quantity, Equipped      int
			CreatedAt               string `json:"created_at"`
		}
		if err := json.Unmarshal(c.Before, &row); err != nil {
			return err
		}
		switch c.Kind {
		case "item":
			_, err = tx.Exec("INSERT INTO items(id,character_id,name,description,quantity,equipped,created_at) VALUES(?,?,?,?,?,?,?)", id, owner, row.Name, row.Description, row.Quantity, row.Equipped, row.CreatedAt)
		case "npc":
			_, err = tx.Exec("INSERT INTO session_npcs(id,session_id,name,note,created_at) VALUES(?,?,?,?,?)", id, owner, row.Name, row.Note, row.CreatedAt)
		default:
			return ErrChangeConflict
		}
		return err
	}
	return ErrChangeConflict
}
func expectOne(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrChangeConflict
	}
	return nil
}
