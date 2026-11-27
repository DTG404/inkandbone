// The turn journal and the state it describes are written in the same
// transaction. Unattributed jobs (such as a manual reanalysis) still use the
// same scoped mutations, but do not attach changes to an unrelated GM reply.
package db

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
)

var ErrChangeConflict = errors.New("a later edit prevents undoing this change")

type TurnSource struct{ SessionID, MessageID int64 }

type TurnChange struct {
	ID          int64           `json:"id"`
	MessageID   int64           `json:"message_id"`
	Kind        string          `json:"kind"`
	OwnerID     *int64          `json:"-"`
	EntityID    *int64          `json:"-"`
	Field       string          `json:"field,omitempty"`
	Operation   string          `json:"operation"`
	Label       string          `json:"label"`
	Before      json.RawMessage `json:"before,omitempty"`
	After       json.RawMessage `json:"after,omitempty"`
	CorrectedAt *string         `json:"corrected_at,omitempty"`
	Superseded  bool            `json:"superseded"`
	CanUndo     bool            `json:"can_undo"`
}

const changeColumns = "c.id,c.message_id,c.kind,c.owner_id,c.entity_id,c.field,c.operation,c.label,c.before_json,c.after_json,c.corrected_at,c.superseded"

func readTurnChange(row interface{ Scan(...any) error }) (TurnChange, error) {
	var c TurnChange
	var before, after []byte
	err := row.Scan(&c.ID, &c.MessageID, &c.Kind, &c.OwnerID, &c.EntityID, &c.Field, &c.Operation, &c.Label, &before, &after, &c.CorrectedAt, &c.Superseded)
	c.Before, c.After = before, after
	c.CanUndo = c.OwnerID != nil && c.EntityID != nil && !c.Superseded && c.CorrectedAt == nil
	return c, err
}

func (d *DB) ListTurnChanges(session int64) ([]TurnChange, error) {
	rows, err := d.db.Query("SELECT "+changeColumns+" FROM turn_changes c JOIN messages m ON m.id=c.message_id WHERE m.session_id=? AND m.role='assistant' AND m.whisper=0 ORDER BY c.id", session)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TurnChange{}
	for rows.Next() {
		c, err := readTurnChange(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func validateTurnOwner(tx *sql.Tx, src TurnSource, kind string, owner int64) error {
	var check string
	switch kind {
	case "stat", "currency", "item":
		check = "EXISTS (SELECT 1 FROM characters ch WHERE ch.id=? AND ch.campaign_id=s.campaign_id)"
	case "objective":
		check = "s.campaign_id=?"
	case "npc":
		check = "s.id=?"
	default:
		return fmt.Errorf("unknown change kind %q", kind)
	}
	var ok int
	return tx.QueryRow("SELECT 1 FROM sessions s WHERE s.id=? AND "+check+" AND (?=0 OR EXISTS (SELECT 1 FROM messages m WHERE m.session_id=s.id AND m.id=? AND m.role='assistant' AND m.whisper=0))", src.SessionID, owner, src.MessageID, src.MessageID).Scan(&ok)
}

func withTurnTx[T any](d *DB, src TurnSource, kind string, owner int64, fn func(*sql.Tx) (T, error)) (T, error) {
	var zero T
	tx, err := d.db.Begin()
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	if err := validateTurnOwner(tx, src, kind, owner); err != nil {
		return zero, err
	}
	v, err := fn(tx)
	if err != nil {
		return zero, err
	}
	return v, tx.Commit()
}

func alreadyApplied(tx *sql.Tx, src TurnSource, kind, op, key string) (bool, error) {
	if src.MessageID == 0 {
		return false, nil
	}
	var exists bool
	err := tx.QueryRow("SELECT EXISTS (SELECT 1 FROM turn_changes WHERE message_id=? AND kind=? AND operation=? AND dedup_key=?)", src.MessageID, kind, op, key).Scan(&exists)
	return exists, err
}

func recordChange(tx *sql.Tx, src TurnSource, kind string, owner, entity int64, field, op, key, label string, before, after json.RawMessage) error {
	if src.MessageID == 0 {
		return nil
	}
	asText := func(value json.RawMessage) any {
		if value == nil {
			return nil
		}
		return string(value)
	}
	_, err := tx.Exec("INSERT INTO turn_changes(message_id,kind,owner_id,entity_id,field,operation,dedup_key,label,before_json,after_json) VALUES(?,?,?,?,?,?,?,?,?,?)", src.MessageID, kind, owner, entity, field, op, key, label, asText(before), asText(after))
	return err
}

type entitySpec struct{ table, owner, name, snapshot string }

var turnEntities = map[string]entitySpec{
	"item":      {"items", "character_id", "name", "json_object('name',name,'description',description,'quantity',quantity,'equipped',equipped,'created_at',created_at)"},
	"npc":       {"session_npcs", "session_id", "name", "json_object('name',name,'note',note,'created_at',created_at)"},
	"objective": {"objectives", "campaign_id", "title", "json_object('title',title,'description',description,'status',status,'created_at',created_at)"},
}

func (s entitySpec) snapshotRow(tx *sql.Tx, owner, id int64) (json.RawMessage, error) {
	var v []byte
	err := tx.QueryRow("SELECT "+s.snapshot+" FROM "+s.table+" WHERE "+s.owner+"=? AND id=?", owner, id).Scan(&v)
	return v, err
}
func (s entitySpec) named(tx *sql.Tx, owner int64, label string) (bool, error) {
	var exists bool
	err := tx.QueryRow("SELECT EXISTS (SELECT 1 FROM "+s.table+" WHERE "+s.owner+"=? AND lower(trim("+s.name+"))=lower(trim(?)))", owner, label).Scan(&exists)
	return exists, err
}

func (d *DB) AddTurnEntity(src TurnSource, kind string, owner int64, label, detail string, quantity int) (int64, error) {
	spec, ok := turnEntities[kind]
	if !ok {
		return 0, errors.New("unknown turn entity")
	}
	return withTurnTx(d, src, kind, owner, func(tx *sql.Tx) (int64, error) {
		key := strings.ToLower(strings.TrimSpace(label))
		if done, err := alreadyApplied(tx, src, kind, "add", key); err != nil || done {
			return 0, err
		}
		if exists, err := spec.named(tx, owner, label); err != nil || exists {
			return 0, err
		}
		var res sql.Result
		var err error
		switch kind {
		case "item":
			res, err = tx.Exec("INSERT INTO items(character_id,name,description,quantity) VALUES(?,?,?,?)", owner, label, detail, quantity)
		case "objective":
			res, err = tx.Exec("INSERT INTO objectives(campaign_id,title,description) VALUES(?,?,?)", owner, label, detail)
		case "npc":
			res, err = tx.Exec("INSERT INTO session_npcs(session_id,name,note) VALUES(?,?,?)", owner, label, detail)
		}
		if err != nil {
			return 0, err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return 0, err
		}
		after, err := spec.snapshotRow(tx, owner, id)
		if err != nil {
			return 0, err
		}
		return id, recordChange(tx, src, kind, owner, id, "", "add", key, label, nil, after)
	})
}

func (d *DB) RemoveTurnEntity(src TurnSource, kind string, owner, id int64, label string) error {
	if kind != "item" && kind != "npc" {
		return errors.New("unsupported removal")
	}
	spec := turnEntities[kind]
	_, err := withTurnTx(d, src, kind, owner, func(tx *sql.Tx) (bool, error) {
		key := strings.ToLower(strings.TrimSpace(label))
		if done, err := alreadyApplied(tx, src, kind, "remove", key); err != nil || done {
			return false, err
		}
		before, err := spec.snapshotRow(tx, owner, id)
		if err != nil {
			return false, err
		}
		var name struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(before, &name); err != nil {
			return false, err
		}
		if name.Name != label {
			return false, ErrChangeConflict
		}
		if _, err = tx.Exec("DELETE FROM "+spec.table+" WHERE id=?", id); err != nil {
			return false, err
		}
		return true, recordChange(tx, src, kind, owner, id, "", "remove", key, label, before, nil)
	})
	return err
}

func (d *DB) SetTurnObjectiveStatus(src TurnSource, owner, id int64, title, status string) error {
	_, err := withTurnTx(d, src, "objective", owner, func(tx *sql.Tx) (bool, error) {
		var old, label string
		if err := tx.QueryRow("SELECT status,title FROM objectives WHERE campaign_id=? AND id=?", owner, id).Scan(&old, &label); err != nil {
			return false, err
		}
		if old != "active" || old == status || label != title {
			return false, nil
		}
		key := strings.ToLower(strings.TrimSpace(label))
		if done, err := alreadyApplied(tx, src, "objective", "set", key); err != nil || done {
			return false, err
		}
		if _, err := tx.Exec("UPDATE objectives SET status=? WHERE id=?", status, id); err != nil {
			return false, err
		}
		before, _ := json.Marshal(old)
		after, _ := json.Marshal(status)
		return true, recordChange(tx, src, "objective", owner, id, "status", "set", key, label, before, after)
	})
	return err
}

type TurnCurrency struct {
	Balance, Delta, ChangeID int64
	Label                    string
}

func (d *DB) ApplyTurnCurrency(src TurnSource, char int64, delta int64) (TurnCurrency, error) {
	return withTurnTx(d, src, "currency", char, func(tx *sql.Tx) (TurnCurrency, error) {
		var result TurnCurrency
		if err := tx.QueryRow("SELECT currency_balance,currency_label FROM characters WHERE id=?", char).Scan(&result.Balance, &result.Label); err != nil {
			return result, err
		}
		if done, err := alreadyApplied(tx, src, "currency", "set", "balance"); err != nil || done {
			return result, err
		}
		if delta > 0 && result.Balance > math.MaxInt64-delta {
			return result, errors.New("currency exceeds supported balance")
		}
		previous := result.Balance
		result.Balance = max(0, previous+delta)
		result.Delta = result.Balance - previous
		if result.Delta == 0 {
			return result, nil
		}
		if _, err := tx.Exec("UPDATE characters SET currency_balance=? WHERE id=?", result.Balance, char); err != nil {
			return result, err
		}
		before, _ := json.Marshal(previous)
		after, _ := json.Marshal(result.Balance)
		err := recordChange(tx, src, "currency", char, char, "balance", "set", "balance", result.Label, before, after)
		if err == nil && src.MessageID != 0 {
			err = tx.QueryRow("SELECT last_insert_rowid()").Scan(&result.ChangeID)
		}
		return result, err
	})
}

func (d *DB) ApplyTurnStats(src TurnSource, char int64, reason, initial, target string) (string, error) {
	var before, after map[string]json.RawMessage
	if err := json.Unmarshal([]byte(initial), &before); err != nil {
		return "", err
	}
	if err := json.Unmarshal([]byte(target), &after); err != nil {
		return "", err
	}
	return withTurnTx(d, src, "stat", char, func(tx *sql.Tx) (string, error) {
		var raw string
		if err := tx.QueryRow("SELECT data_json FROM characters WHERE id=?", char).Scan(&raw); err != nil {
			return "", err
		}
		var live map[string]json.RawMessage
		if err := json.Unmarshal([]byte(raw), &live); err != nil {
			return "", err
		}
		if live == nil {
			return "", errors.New("character data must be an object")
		}
		var changed []string
		for _, key := range slices.Sorted(maps.Keys(after)) {
			if bytes.Equal(before[key], after[key]) || !bytes.Equal(live[key], before[key]) {
				continue
			}
			done, err := alreadyApplied(tx, src, "stat", "set", reason+"/"+key)
			if err != nil {
				return "", err
			}
			if done {
				continue
			}
			live[key] = after[key]
			changed = append(changed, key)
		}
		if len(changed) == 0 {
			return raw, nil
		}
		updated, err := json.Marshal(live)
		if err != nil {
			return "", err
		}
		if _, err = tx.Exec("UPDATE characters SET data_json=? WHERE id=?", string(updated), char); err != nil {
			return "", err
		}
		for _, k := range changed {
			if err := recordChange(tx, src, "stat", char, char, k, "set", reason+"/"+k, k, before[k], after[k]); err != nil {
				return "", err
			}
		}
		return string(updated), nil
	})
}
