package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

const archiveVersion = 1
const archiveLimit = 96 << 20
const archiveRowLimit = 100000

var errInvalidArchive = errors.New("invalid campaign archive")
var errArchiveSize = errors.New("campaign archive exceeds size limit")
var errArchiveDependencies = errors.New("install the matching ruleset version and referenced rulebooks before importing")

type archiveRow map[string]json.RawMessage
type campaignArchive struct {
	Format         string                  `json:"format"`
	Version        int                     `json:"version"`
	Ruleset        string                  `json:"ruleset"`
	RulesetVersion string                  `json:"ruleset_version"`
	Books          []string                `json:"book_sources"`
	Tables         map[string][]archiveRow `json:"tables"`
	Assets         map[string][]byte       `json:"assets"`
}

// Every export query follows the ownership chain to one campaign. Identifiers
// for queries and inserts come from this registry and the installed schema.
type archiveTable struct{ name, owner, parent string }

var campaignTables = []archiveTable{
	{"campaigns", "", ""}, {"adventures", "campaign_id", "campaigns"}, {"characters", "campaign_id", "campaigns"},
	{"sessions", "campaign_id", "campaigns"}, {"messages", "session_id", "sessions"}, {"session_npcs", "session_id", "sessions"},
	{"maps", "campaign_id", "campaigns"}, {"map_pins", "map_id", "maps"}, {"map_tokens", "map_id", "maps"}, {"map_zones", "map_id", "maps"},
	{"world_notes", "campaign_id", "campaigns"}, {"objectives", "campaign_id", "campaigns"}, {"items", "character_id", "characters"},
	{"character_macros", "character_id", "characters"}, {"relationships", "campaign_id", "campaigns"}, {"factions", "campaign_id", "campaigns"},
	{"npc_stats", "campaign_id", "campaigns"}, {"secrets", "campaign_id", "campaigns"}, {"calendar_events", "campaign_id", "campaigns"},
	{"combat_encounters", "session_id", "sessions"}, {"combatants", "encounter_id", "combat_encounters"}, {"dice_rolls", "session_id", "sessions"},
	{"xp_log", "session_id", "sessions"}, {"decks", "campaign_id", "campaigns"}, {"deck_draws", "session_id", "sessions"},
	{"gm_turns", "action_message_id", "messages"}, {"turn_changes", "message_id", "messages"},
}

type archiveColumn struct{ name, kind string }

func archiveColumns(ctx context.Context, tx *sql.Tx, table string) ([]archiveColumn, error) {
	rows, err := tx.QueryContext(ctx, `SELECT name,upper(type) FROM pragma_table_info(?) ORDER BY cid`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []archiveColumn
	for rows.Next() {
		var c archiveColumn
		if err = rows.Scan(&c.name, &c.kind); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}
func archivePK(table string) string {
	if table == "gm_turns" {
		return "action_message_id"
	}
	return "id"
}
func archiveID(v json.RawMessage) (int64, error) {
	var id int64
	if json.Unmarshal(v, &id) != nil || id <= 0 {
		return 0, errInvalidArchive
	}
	return id, nil
}
func archiveString(v json.RawMessage) string { var s string; _ = json.Unmarshal(v, &s); return s }
func archiveJSON(v any) json.RawMessage      { data, _ := json.Marshal(v); return data }

func (s *Server) registerCampaignArchiveRoutes() {
	s.mux.HandleFunc("GET /api/campaigns/{id}/archive", s.handleExportCampaign)
	s.mux.HandleFunc("POST /api/campaigns/import", s.handleImportCampaign)
}
func (s *Server) handleExportCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		respondError(w, "invalid campaign id", 400)
		return
	}
	a, err := s.exportCampaign(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		respondError(w, "campaign not found", 404)
		return
	}
	if errors.Is(err, errArchiveSize) {
		respondError(w, err.Error(), 413)
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	data, err := json.Marshal(a)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if len(data) > archiveLimit {
		respondError(w, errArchiveSize.Error(), 413)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="campaign-%d.inkandbone.json"`, id))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}
func (s *Server) exportCampaign(ctx context.Context, id int64) (*campaignArchive, error) {
	tx, err := s.db.SQL().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	a := &campaignArchive{Format: "inkandbone-campaign", Version: archiveVersion, Books: []string{}, Tables: map[string][]archiveRow{}, Assets: map[string][]byte{}}
	var rulesetID int64
	err = tx.QueryRowContext(ctx, `SELECT r.id,r.name,r.version FROM campaigns c JOIN rulesets r ON r.id=c.ruleset_id WHERE c.id=?`, id).Scan(&rulesetID, &a.Ruleset, &a.RulesetVersion)
	if err != nil {
		return nil, err
	}
	a.Books, err = archiveBooks(ctx, tx, rulesetID)
	if err != nil {
		return nil, err
	}
	filters := map[string]string{}
	size, count := 0, 0
	for _, t := range campaignTables {
		cols, err := archiveColumns(ctx, tx, t.name)
		if err != nil {
			return nil, err
		}
		if len(cols) == 0 {
			continue
		}
		filter := "id=?"
		if t.owner != "" {
			filter = t.owner + " IN (SELECT " + archivePK(t.parent) + " FROM " + t.parent + " WHERE " + filters[t.parent] + ")"
		}
		filters[t.name] = filter
		selectCols := make([]string, 0, len(cols))
		for _, c := range cols {
			expr := c.name
			if c.kind == "TEXT" || c.kind == "DATETIME" || c.kind == "TIMESTAMP" {
				expr = "CAST(" + c.name + " AS TEXT)"
			}
			selectCols = append(selectCols, expr)
		}
		rows, err := tx.QueryContext(ctx, "SELECT "+strings.Join(selectCols, ",")+" FROM "+t.name+" WHERE "+filter+" ORDER BY "+archivePK(t.name), id)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			count++
			if count > archiveRowLimit {
				rows.Close()
				return nil, errArchiveSize
			}
			values := make([]any, len(cols))
			slots := make([]any, len(cols))
			for i := range slots {
				slots[i] = &values[i]
			}
			if err = rows.Scan(slots...); err != nil {
				rows.Close()
				return nil, err
			}
			row := archiveRow{}
			for i, c := range cols {
				v := archiveJSON(values[i])
				size += len(v) + len(c.name) + 4
				row[c.name] = v
			}
			if t.name == "gm_turns" {
				row["lease_token"] = archiveJSON(nil)
				row["lease_until"] = archiveJSON(0)
			}
			n, err := s.exportArchiveAsset(t.name, row, a.Assets)
			if err != nil {
				rows.Close()
				return nil, err
			}
			size += n
			if size > archiveLimit {
				rows.Close()
				return nil, errArchiveSize
			}
			a.Tables[t.name] = append(a.Tables[t.name], row)
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	return a, tx.Commit()
}
func archiveBooks(ctx context.Context, tx *sql.Tx, rulesetID int64) ([]string, error) {
	rows, err := tx.QueryContext(ctx, "SELECT DISTINCT source FROM rulebook_chunks WHERE ruleset_id=? ORDER BY source", rulesetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	books := []string{}
	for rows.Next() {
		var book string
		if err = rows.Scan(&book); err != nil {
			return nil, err
		}
		books = append(books, book)
	}
	return books, rows.Err()
}

func archiveAssetFields(table string) (col, dir string, types map[string]string) {
	switch table {
	case "maps":
		return "image_path", "maps", mapAssetTypes
	case "characters":
		return "portrait_path", "portraits", portraitAssetTypes
	}
	return "", "", nil
}
func (s *Server) exportArchiveAsset(table string, row archiveRow, assets map[string][]byte) (int, error) {
	col, dir, types := archiveAssetFields(table)
	path := archiveString(row[col])
	if col == "" || path == "" {
		return 0, nil
	}
	relative, err := storedAssetRelative(path, dir)
	if err != nil {
		return 0, err
	}
	f, _, err := openValidatedAsset(filepath.Join(s.dataDir, dir), relative, types)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, imageUploadLimit+1))
	if err != nil {
		return 0, err
	}
	if len(data) > int(imageUploadLimit) {
		return 0, errArchiveSize
	}
	id, err := archiveID(row["id"])
	if err != nil {
		return 0, err
	}
	key := fmt.Sprintf("%s/%d%s", dir, id, strings.ToLower(filepath.Ext(relative)))
	assets[key] = data
	row[col] = archiveJSON(key)
	return (len(data) + 2) / 3 * 4, nil
}
