package db

import (
	"database/sql"
	"encoding/json"
	"errors"
)

var ErrTurnConflict = errors.New("another action or response is pending")
var ErrActionKeyConflict = errors.New("action key already belongs to a different submission")

type RuleCitation struct {
	ChunkID int64  `json:"chunk_id"`
	Source  string `json:"source"`
	Heading string `json:"heading"`
	Passage string `json:"passage"`
}

// Capture an implicit actor at submission, while the active character and
// session still describe the player's choice. A later retry must never resolve
// the actor from mutable application settings.
const submittedActor = `COALESCE(?, (SELECT c.id FROM characters c JOIN sessions s ON s.campaign_id=c.campaign_id
 JOIN settings active ON active.key='active_character_id' AND active.value=CAST(c.id AS TEXT)
 WHERE s.id=?))`

// SubmitAction deduplicates a browser submission even if its HTTP acknowledgement was lost.
func (d *DB) SubmitAction(sid int64, text string, whisper bool, character *int64, key string, expectGM bool) (int64, bool, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`INSERT INTO messages(session_id,role,content,whisper,character_id,client_action_key)
 SELECT ?, 'user', ?, ?, `+submittedActor+`, ? WHERE ? OR NOT EXISTS (
 SELECT 1 FROM gm_turns t JOIN messages m ON m.id=t.action_message_id WHERE m.session_id=? AND response_message_id IS NULL AND m.id=(SELECT id FROM messages WHERE session_id=m.session_id AND whisper=0 ORDER BY created_at DESC,id DESC LIMIT 1))
 ON CONFLICT DO NOTHING`, sid, text, whisper, character, sid, key, whisper || !expectGM, sid)
	if err != nil {
		return 0, false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, false, err
	}
	var id int64
	var matches bool
	err = tx.QueryRow(`SELECT id, (content=? AND role='user' AND whisper=? AND (? IS NULL OR character_id IS ?)) FROM messages WHERE session_id=? AND client_action_key=?`, text, whisper, character, character, sid, key).Scan(&id, &matches)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, ErrTurnConflict
	}
	if err != nil {
		return 0, false, err
	}
	if !matches {
		return 0, false, ErrActionKeyConflict
	}
	if n == 1 && !whisper && expectGM {
		if _, err = tx.Exec("INSERT INTO gm_turns(action_message_id) VALUES (?)", id); err != nil {
			return 0, false, err
		}
	}
	return id, n == 1, tx.Commit()
}

type TurnClaim struct {
	Preparation sql.NullString
	ReplyID     int64
	Reply       string
}

func (d *DB) ClaimTurn(sid, action int64, token string) (TurnClaim, error) {
	var claim TurnClaim
	tx, err := d.db.Begin()
	if err != nil {
		return claim, err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`INSERT OR IGNORE INTO gm_turns(action_message_id) SELECT id FROM messages WHERE id=? AND session_id=? AND role='user' AND whisper=0 AND id=(SELECT id FROM messages WHERE session_id=? AND whisper=0 ORDER BY created_at DESC,id DESC LIMIT 1)`, action, sid, sid)
	if err != nil {
		return claim, err
	}
	err = tx.QueryRow(`SELECT t.preparation,coalesce(t.response_message_id,0),coalesce(r.content,'') FROM gm_turns t JOIN messages a ON a.id=t.action_message_id LEFT JOIN messages r ON r.id=t.response_message_id WHERE a.id=? AND a.session_id=?`, action, sid).Scan(&claim.Preparation, &claim.ReplyID, &claim.Reply)
	if err != nil {
		return claim, err
	}
	if claim.ReplyID != 0 {
		return claim, tx.Commit()
	}
	res, err := tx.Exec(`UPDATE gm_turns SET lease_token=?,lease_until=unixepoch()+30 WHERE action_message_id=? AND (lease_token IS NULL OR lease_until<=unixepoch()) AND action_message_id=(SELECT id FROM messages WHERE session_id=? AND whisper=0 ORDER BY created_at DESC,id DESC LIMIT 1)`, token, action, sid)
	if err = changedTurn(res, err); err != nil {
		return claim, err
	}
	return claim, tx.Commit()
}
func changedTurn(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrTurnConflict
	}
	return nil
}
func (d *DB) RenewTurn(action int64, token string) error {
	return changedTurn(d.db.Exec("UPDATE gm_turns SET lease_until=unixepoch()+30 WHERE action_message_id=? AND lease_token=? AND response_message_id IS NULL", action, token))
}
func (d *DB) ReleaseTurn(action int64, token string) error {
	_, err := d.db.Exec("UPDATE gm_turns SET lease_token=NULL,lease_until=0 WHERE action_message_id=? AND lease_token=?", action, token)
	return err
}

type TurnDice struct {
	Expression string
	Total      int
	Breakdown  string
}
type TurnCharacterChange struct {
	ID            int64
	Before, After string
}

func (d *DB) SaveTurnPreparation(sid, action int64, token, prepared string, rolls []TurnDice, stats *TurnCharacterChange) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = changedTurn(tx.Exec("UPDATE gm_turns SET preparation=? WHERE action_message_id=? AND lease_token=? AND preparation IS NULL", prepared, action, token)); err != nil {
		return err
	}
	for _, roll := range rolls {
		if _, err = tx.Exec("INSERT INTO dice_rolls(session_id,expression,result,breakdown_json) VALUES(?,?,?,?)", sid, roll.Expression, roll.Total, roll.Breakdown); err != nil {
			return err
		}
	}
	if stats != nil {
		if err = changedTurn(tx.Exec("UPDATE characters SET data_json=? WHERE id=? AND data_json=?", stats.After, stats.ID, stats.Before)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *DB) FinishTurn(sid, action int64, token, reply string, citations []RuleCitation) (int64, error) {
	payload, err := json.Marshal(citations)
	if err != nil {
		return 0, err
	}
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.Exec("INSERT INTO messages(session_id,role,content,rule_citations) VALUES(?,'assistant',?,?)", sid, reply, string(payload))
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err = changedTurn(tx.Exec("UPDATE gm_turns SET response_message_id=?, lease_token=NULL,lease_until=0 WHERE action_message_id=? AND lease_token=? AND response_message_id IS NULL", id, action, token)); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

// AppendTranscriptMessage keeps older clients' unkeyed transcript posts usable while
// preventing them from racing a GM worker or overtaking a submitted browser action.
func (d *DB) AppendTranscriptMessage(sid int64, role, text string, whisper bool, character *int64) (int64, error) {
	if whisper {
		return d.CreateMessage(sid, role, text, true, character)
	}
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO messages(session_id,role,content,character_id)
 SELECT ?,?,?,CASE WHEN ?='user' THEN `+submittedActor+` ELSE ? END WHERE NOT EXISTS (SELECT 1 FROM gm_turns t
 WHERE t.action_message_id=(SELECT id FROM messages WHERE session_id=? AND whisper=0 ORDER BY created_at DESC,id DESC LIMIT 1)
 AND t.response_message_id IS NULL AND (?='user' OR t.lease_until>unixepoch()))`, sid, role, text, role, character, sid, character, sid, role)
	if err = changedTurn(res, err); err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if role == "assistant" {
		_, err = tx.Exec(`UPDATE gm_turns SET response_message_id=?, lease_token=NULL,lease_until=0 WHERE response_message_id IS NULL AND action_message_id=(
  SELECT id FROM messages WHERE session_id=? AND whisper=0 AND id<>? ORDER BY created_at DESC,id DESC LIMIT 1)`, id, sid, id)
		if err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}
