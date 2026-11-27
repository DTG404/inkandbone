package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type archiveIDs map[string]map[int64]int64

func (s *Server) handleImportCampaign(w http.ResponseWriter, r *http.Request) {
	var a campaignArchive
	if err := decodeJSON(w, r, &a, archiveLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	id, err := s.importCampaign(r.Context(), a)
	switch {
	case errors.Is(err, errInvalidArchive):
		respondError(w, errInvalidArchive.Error(), 400)
		return
	case errors.Is(err, errArchiveSize):
		respondError(w, errArchiveSize.Error(), 413)
		return
	case errors.Is(err, errArchiveDependencies):
		respondError(w, errArchiveDependencies.Error(), 422)
		return
	case err != nil:
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventCampaignCreated, Payload: &CampaignCreatedPayload{CampaignID: RealtimeInt64(id), Name: RealtimePtr(archiveString(a.Tables["campaigns"][0]["name"]))}})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]int64{"id": id})
}

func (s *Server) importCampaign(ctx context.Context, a campaignArchive) (int64, error) {
	if a.Format != "inkandbone-campaign" || a.Version != archiveVersion || a.Ruleset == "" || len(a.Tables["campaigns"]) != 1 {
		return 0, errInvalidArchive
	}
	tx, err := s.db.SQL().BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var rulesetID int64
	err = tx.QueryRowContext(ctx, "SELECT id FROM rulesets WHERE name=? AND version=?", a.Ruleset, a.RulesetVersion).Scan(&rulesetID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errArchiveDependencies
	}
	if err != nil {
		return 0, err
	}
	books, err := archiveBooks(ctx, tx, rulesetID)
	if err != nil {
		return 0, err
	}
	installed := make(map[string]bool, len(books))
	for _, book := range books {
		installed[book] = true
	}
	for _, book := range a.Books {
		if !installed[book] {
			return 0, errArchiveDependencies
		}
		delete(installed, book)
	}
	schema := map[string][]archiveColumn{}
	ids := archiveIDs{}
	count := 0
	// Reserve fresh identifiers for the whole graph before inserting. Parents
	// such as a reassigned objective can have a higher ID than their children.
	for _, t := range campaignTables {
		rows, exists := a.Tables[t.name]
		if !exists {
			continue
		}
		count += len(rows)
		if count > archiveRowLimit {
			return 0, errArchiveSize
		}
		cols, err := archiveColumns(ctx, tx, t.name)
		if err != nil {
			return 0, err
		}
		if len(cols) == 0 {
			return 0, errInvalidArchive
		}
		schema[t.name] = cols
		ids[t.name] = map[int64]int64{}
		var seq int64
		if t.name != "gm_turns" {
			if err = tx.QueryRowContext(ctx, "SELECT coalesce((SELECT seq FROM sqlite_sequence WHERE name=?),0)", t.name).Scan(&seq); err != nil {
				return 0, err
			}
		}
		for _, row := range rows {
			if len(row) != len(cols) {
				return 0, errInvalidArchive
			}
			for _, col := range cols {
				if _, ok := row[col.name]; !ok {
					return 0, errInvalidArchive
				}
			}

			old, err := archiveID(row[archivePK(t.name)])
			if err != nil {
				return 0, err
			}
			if _, exists := ids[t.name][old]; exists || seq == math.MaxInt64 {
				return 0, errInvalidArchive
			}
			seq++
			ids[t.name][old] = seq
		}
	}
	if len(schema) != len(a.Tables) {
		return 0, errInvalidArchive
	}
	if err := validateArchiveTurns(a.Tables); err != nil {
		return 0, err
	}
	if _, err = tx.ExecContext(ctx, "PRAGMA defer_foreign_keys=ON"); err != nil {
		return 0, err
	}

	var created []string
	committed := false
	defer func() {
		if !committed {
			for _, path := range created {
				_ = removeStoredAsset(filepath.Dir(path), filepath.Base(path))
			}
		}
	}()
	for _, t := range campaignTables {
		if t.name == "turn_changes" {
			if err := reserveRemovedTargets(ctx, tx, a.Tables[t.name], ids); err != nil {
				return 0, err
			}
		}
		for _, row := range a.Tables[t.name] {
			old, err := archiveID(row[archivePK(t.name)])
			if err != nil {
				return 0, err
			}
			path, err := s.importArchiveAsset(t.name, old, row, a.Assets)
			if err != nil {
				return 0, err
			}
			if path != "" {
				created = append(created, path)
			}
			if err = remapArchiveRow(t.name, row, ids, rulesetID); err != nil {
				return 0, err
			}
			cols := make([]string, 0, len(row))
			args := make([]any, 0, len(row))
			params := make([]string, 0, len(row))
			for _, col := range schema[t.name] {
				raw, ok := row[col.name]
				if !ok {
					return 0, errInvalidArchive
				}
				v, err := archiveValue(col.kind, raw)
				if err != nil {
					return 0, err
				}
				cols = append(cols, col.name)
				args = append(args, v)
				params = append(params, "?")
			}
			_, err = tx.ExecContext(ctx, "INSERT INTO "+t.name+" ("+strings.Join(cols, ",")+") VALUES ("+strings.Join(params, ",")+")", args...)
			if err != nil {
				return 0, archiveInsertError(err)
			}
		}
	}
	if len(a.Assets) != 0 {
		return 0, errInvalidArchive
	}
	if err = tx.Commit(); err != nil {
		return 0, archiveInsertError(err)
	}
	committed = true
	id, err := archiveID(a.Tables["campaigns"][0]["id"])
	return id, err
}

// Cached replies can be replayed directly to a player. A pair must link a
// public user action to a public assistant reply in that same session.
func validateArchiveTurns(tables map[string][]archiveRow) error {
	messages := map[int64]archiveRow{}
	for _, row := range tables["messages"] {
		id, err := archiveID(row["id"])
		if err != nil {
			return err
		}
		messages[id] = row
	}
	for _, turn := range tables["gm_turns"] {
		actionID, err := archiveID(turn["action_message_id"])
		if err != nil {
			return err
		}
		action := messages[actionID]
		if archiveString(action["role"]) != "user" || string(action["whisper"]) != "0" {
			return errInvalidArchive
		}
		if bytes.Equal(turn["response_message_id"], []byte("null")) {
			continue
		}
		replyID, err := archiveID(turn["response_message_id"])
		if err != nil {
			return err
		}
		reply := messages[replyID]
		if archiveString(reply["role"]) != "assistant" || string(reply["whisper"]) != "0" || !bytes.Equal(action["session_id"], reply["session_id"]) {
			return errInvalidArchive
		}
	}
	for _, change := range tables["turn_changes"] {
		id, err := archiveID(change["message_id"])
		if err != nil {
			return err
		}
		if archiveString(messages[id]["role"]) != "assistant" {
			return errInvalidArchive
		}
		for _, col := range []string{"before_json", "after_json"} {
			raw := change[col]
			if !bytes.Equal(raw, []byte("null")) && !json.Valid([]byte(archiveString(raw))) {
				return errInvalidArchive
			}
		}
	}
	return nil
}

// A removal journal can restore a deleted item or NPC at its former ID. Reserve
// fresh destination IDs for those tombstones so future auto-inserts cannot take them.
func reserveRemovedTargets(ctx context.Context, tx *sql.Tx, changes []archiveRow, ids archiveIDs) error {
	for _, target := range []struct{ kind, table string }{{"item", "items"}, {"npc", "session_npcs"}} {
		var missing []int64
		seen := make(map[int64]bool)
		for _, row := range changes {
			if archiveString(row["kind"]) != target.kind || archiveString(row["operation"]) != "remove" {
				continue
			}
			raw := row["entity_id"]
			if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
				continue
			}
			old, err := archiveID(raw)
			if err != nil {
				return err
			}
			if ids[target.table][old] == 0 && !seen[old] {
				missing = append(missing, old)
				seen[old] = true
			}
		}
		if len(missing) == 0 {
			continue
		}
		var seq int64
		err := tx.QueryRowContext(ctx, "SELECT seq FROM sqlite_sequence WHERE name=?", target.table).Scan(&seq)
		if errors.Is(err, sql.ErrNoRows) {
			_, err = tx.ExecContext(ctx, "INSERT INTO sqlite_sequence(name,seq) VALUES (?,0)", target.table)
		}
		if err != nil {
			return err
		}
		if int64(len(missing)) > math.MaxInt64-seq {
			return errInvalidArchive
		}
		if ids[target.table] == nil {
			ids[target.table] = make(map[int64]int64)
		}
		for _, old := range missing {
			seq++
			ids[target.table][old] = seq
		}
		if _, err = tx.ExecContext(ctx, "UPDATE sqlite_sequence SET seq=? WHERE name=?", seq, target.table); err != nil {
			return err
		}
	}
	return nil
}

// Constraint errors describe malformed input. Operational failures are reported
// through the standard server error path.
func archiveInsertError(err error) error {
	var e interface{ Code() int }
	if errors.As(err, &e) && e.Code()&255 == 19 {
		return errInvalidArchive
	}
	return err
}
func archiveValue(kind string, raw json.RawMessage) (any, error) {
	if bytes.Equal(raw, []byte("null")) {
		return nil, nil
	}
	switch kind {
	case "INTEGER":
		var n int64
		if json.Unmarshal(raw, &n) == nil {
			return n, nil
		}
	case "REAL":
		var n float64
		if json.Unmarshal(raw, &n) == nil {
			return n, nil
		}
	case "TEXT", "DATETIME", "TIMESTAMP":
		var text string
		if json.Unmarshal(raw, &text) == nil {
			return text, nil
		}
	}
	return nil, errInvalidArchive
}

var archiveReferences = map[string]string{
	"campaign_id": "campaigns", "session_id": "sessions", "revealed_at_session_id": "sessions",
	"adventure_id": "adventures", "character_id": "characters", "encounter_id": "combat_encounters",
	"map_id": "maps", "parent_id": "objectives", "deck_id": "decks", "message_id": "messages",
	"action_message_id": "messages", "response_message_id": "messages",
}

func remapArchiveKey(row archiveRow, key, table string, ids archiveIDs, allowMissing bool) error {
	v, ok := row[key]
	if !ok || bytes.Equal(v, []byte("null")) {
		return nil
	}
	old, err := archiveID(v)
	if err != nil {
		return err
	}
	fresh, ok := ids[table][old]
	if !ok {
		if allowMissing {
			row[key] = archiveJSON(nil)
			return nil
		}
		return errInvalidArchive
	}
	row[key] = archiveJSON(fresh)
	return nil
}
func remapArchiveRow(table string, row archiveRow, ids archiveIDs, rulesetID int64) error {
	if err := remapArchiveKey(row, "id", table, ids, false); err != nil {
		return err
	}
	for key, parent := range archiveReferences {
		if err := remapArchiveKey(row, key, parent, ids, false); err != nil {
			return err
		}
	}
	switch table {
	case "campaigns":
		row["ruleset_id"] = archiveJSON(rulesetID)
	case "map_tokens":
		parent := map[string]string{"character": "characters", "npc": "session_npcs"}[archiveString(row["entity_type"])]
		return remapArchiveKey(row, "entity_id", parent, ids, false)
	case "gm_turns":
		row["lease_token"] = archiveJSON(nil)
		row["lease_until"] = archiveJSON(0)
	case "turn_changes":
		targets := map[string][2]string{"stat": {"characters", "characters"}, "currency": {"characters", "characters"}, "item": {"characters", "items"}, "npc": {"sessions", "session_npcs"}, "objective": {"campaigns", "objectives"}}
		target, ok := targets[archiveString(row["kind"])]
		if !ok {
			return errInvalidArchive
		}
		if err := remapArchiveKey(row, "owner_id", target[0], ids, true); err != nil {
			return err
		}
		return remapArchiveKey(row, "entity_id", target[1], ids, true)
	}
	return nil
}
func (s *Server) importArchiveAsset(table string, id int64, row archiveRow, assets map[string][]byte) (string, error) {
	col, dir, types := archiveAssetFields(table)
	if col == "" {
		return "", nil
	}
	key := archiveString(row[col])
	if key == "" {
		return "", nil
	}
	ext := filepath.Ext(key)
	if _, ok := types[ext]; !ok || key != fmt.Sprintf("%s/%d%s", dir, id, ext) {
		return "", errInvalidArchive
	}
	data, ok := assets[key]
	if !ok || len(data) == 0 || int64(len(data)) > imageUploadLimit {
		return "", errInvalidArchive
	}
	base := filepath.Join(s.dataDir, dir)
	if err := os.MkdirAll(base, 0750); err != nil {
		return "", err
	}
	filename := "import-" + randomHex(16) + ext
	if err := writeValidatedUpload(base, filename, bytes.NewReader(data), types); err != nil {
		if errors.Is(err, errInvalidAsset) {
			return "", errInvalidArchive
		}
		return "", err
	}
	delete(assets, key)
	row[col] = archiveJSON(dir + "/" + filename)
	return filepath.Join(base, filename), nil
}
