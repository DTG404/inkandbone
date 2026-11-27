package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCampaignArchiveRoundTrip(t *testing.T) {
	s := newTestServerWithDir(t, t.TempDir())
	cid, sid := seedCampaign(t, s.db)
	exec := func(stmt string, args ...any) {
		t.Helper()
		_, err := s.db.SQL().Exec(stmt, args...)
		require.NoError(t, err)
	}
	exec("INSERT INTO campaigns(name,ruleset_id,gm_notes) SELECT 'Other',ruleset_id,'other campaign secret' FROM campaigns WHERE id=?", cid)
	exec("UPDATE campaigns SET gm_notes = 'secret ending' WHERE id=?", cid)
	exec("INSERT INTO rulebook_chunks(ruleset_id,source,heading,content) SELECT ruleset_id,'Primer','Rules','do not export full rulebook' FROM campaigns WHERE id=?", cid)
	character := createAssetCharacter(t, s, cid, "portraits/hero.png")
	writeAssetFile(t, s.dataDir, "portraits/hero.png", validPNG)
	m := createAssetMap(t, s, cid, "maps/world.png")
	writeAssetFile(t, s.dataDir, "maps/world.png", validPNG)
	exec("INSERT INTO map_tokens(map_id,entity_type,entity_id,x,y) VALUES (?,'character',?,.2,.4)", m, character)
	exec("INSERT INTO map_zones(map_id,name,x,y,width,height,is_revealed) VALUES (?,'vault',0,0,.5,.5,0)", m)
	exec("INSERT INTO messages(session_id,role,content,whisper,character_id) VALUES (?,'user','private move',1,?)", sid, character)
	export := getAsset(t, s, fmt.Sprintf("/api/campaigns/%d/archive", cid))
	require.Equal(t, 200, export.Code)
	require.NotContains(t, export.Body.String(), "do not export full rulebook")
	require.NotContains(t, export.Body.String(), "other campaign secret")
	require.Equal(t, "no-store", export.Header().Get("Cache-Control"))
	post := func(data []byte) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/campaigns/import", bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
		return w
	}
	var manifest map[string]any
	require.NoError(t, json.Unmarshal(export.Body.Bytes(), &manifest))
	tables := manifest["tables"].(map[string]any)
	tables["settings"] = []any{map[string]any{"key": "active_campaign_id", "value": "9"}}
	bad, _ := json.Marshal(manifest)
	require.Equal(t, 400, post(bad).Code)
	delete(tables, "settings")
	mapRow := tables["maps"].([]any)[0].(map[string]any)
	safeKey := mapRow["image_path"]
	mapRow["image_path"] = "../existing-file"
	bad, _ = json.Marshal(manifest)
	require.Equal(t, 400, post(bad).Code)
	mapRow["image_path"] = safeKey
	token := tables["map_tokens"].([]any)[0].(map[string]any)
	token["entity_id"] = 9999999
	bad, _ = json.Marshal(manifest)
	require.Equal(t, 400, post(bad).Code)
	files, err := os.ReadDir(filepath.Join(s.dataDir, "portraits"))
	require.NoError(t, err)
	require.Len(t, files, 1)
	w := post(export.Body.Bytes())
	require.Equal(t, 201, w.Code, w.Body.String())
	var imported struct{ ID int64 }
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &imported))
	require.NotEqual(t, cid, imported.ID)
	var newCharacter, newMap int64
	var secret, whisper, mp, pp string
	err = s.db.SQL().QueryRow("SELECT c.gm_notes,ch.id,m.id,msg.content,m.image_path,ch.portrait_path FROM campaigns c JOIN characters ch ON ch.campaign_id=c.id JOIN maps m ON m.campaign_id=c.id JOIN sessions ss ON ss.campaign_id=c.id JOIN messages msg ON msg.session_id=ss.id AND msg.character_id=ch.id AND msg.whisper=1 JOIN map_tokens tok ON tok.map_id=m.id AND tok.entity_id=ch.id JOIN map_zones z ON z.map_id=m.id AND z.is_revealed=0 WHERE c.id=?", imported.ID).Scan(&secret, &newCharacter, &newMap, &whisper, &mp, &pp)
	require.NoError(t, err)
	require.NotEqual(t, character, newCharacter)
	require.NotEqual(t, m, newMap)
	require.Equal(t, "secret ending", secret)
	require.Equal(t, "private move", whisper)
	require.NotEqual(t, "maps/world.png", mp)
	require.NotEqual(t, "portraits/hero.png", pp)
	for _, p := range []string{mp, pp} {
		data, err := os.ReadFile(filepath.Join(s.dataDir, p))
		require.NoError(t, err)
		require.Equal(t, validPNG, data)
	}
	exec("DELETE FROM rulebook_chunks WHERE source='Primer'")
	require.Equal(t, 422, post(export.Body.Bytes()).Code)
}

func TestArchivePreservesTurnHistoryAndRemovedEntityUndo(t *testing.T) {
	s := newTestServerWithDir(t, t.TempDir())
	cid, sid := seedCampaign(t, s.db)
	insert := func(q string, args ...any) int64 {
		t.Helper()
		result, err := s.db.SQL().Exec(q, args...)
		require.NoError(t, err)
		id, err := result.LastInsertId()
		require.NoError(t, err)
		return id
	}
	character, err := s.db.CreateCharacter(cid, "Scout")
	require.NoError(t, err)
	action := insert("INSERT INTO messages(session_id,role,content,client_action_key) VALUES (?,'user','Look around','action-1')", sid)
	response := insert("INSERT INTO messages(session_id,role,content,rule_citations) VALUES (?,'assistant','A clue','[{\"chunk_id\":91,\"source\":\"Primer\",\"heading\":\"Search\",\"passage\":\"Look closely\"}]')", sid)
	insert("INSERT INTO gm_turns(action_message_id,response_message_id,preparation,lease_token,lease_until) VALUES (?,?, '{}','private-live-lease',999999)", action, response)
	insert("INSERT INTO turn_changes(message_id,kind,owner_id,entity_id,operation,dedup_key,label,before_json) VALUES (?,'item',?,9876,'remove','lost','Rope',?)", response, character, `{"name":"Rope","quantity":1,"created_at":"2026-01-01"}`)
	w := getAsset(t, s, fmt.Sprintf("/api/campaigns/%d/archive", cid))
	require.Equal(t, 200, w.Code)
	require.NotContains(t, w.Body.String(), "private-live-lease")
	// An imported whisper cannot be substituted for a public GM response.
	var bad campaignArchive
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &bad))
	bad.Tables["messages"][1]["whisper"] = archiveJSON(1)
	_, err = s.importCampaign(t.Context(), bad)
	require.ErrorIs(t, err, errInvalidArchive)

	req := httptest.NewRequest(http.MethodPost, "/api/campaigns/import", w.Body)
	req.Header.Set("Content-Type", "application/json")
	out := httptest.NewRecorder()
	s.ServeHTTP(out, req)
	require.Equal(t, 201, out.Code, out.Body.String())
	var dest struct{ ID int64 }
	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &dest))
	var session, change, restoredCharacter int64
	err = s.db.SQL().QueryRow("SELECT m.session_id,c.id,ch.id FROM turn_changes c JOIN messages m ON m.id=c.message_id JOIN gm_turns g ON g.response_message_id=m.id JOIN messages a ON a.id=g.action_message_id AND a.session_id=m.session_id JOIN sessions s ON s.id=m.session_id JOIN characters ch ON ch.id=c.owner_id WHERE s.campaign_id=? AND a.client_action_key='action-1' AND g.lease_token IS NULL AND g.lease_until=0 AND json_extract(m.rule_citations,'$[0].passage')='Look closely'", dest.ID).Scan(&session, &change, &restoredCharacter)
	require.NoError(t, err)
	// A later ordinary inventory insert cannot take the deleted item's reserved ID.
	insert("INSERT INTO items(character_id,name) VALUES (?,'Torch')", restoredCharacter)
	undo := httptest.NewRecorder()
	s.ServeHTTP(undo, httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/sessions/%d/changes/%d/undo", session, change), nil))
	require.Equal(t, 200, undo.Code, undo.Body.String())
	var count int
	require.NoError(t, s.db.SQL().QueryRow("SELECT count(*) FROM items WHERE character_id=? AND name='Rope'", restoredCharacter).Scan(&count))
	require.Equal(t, 1, count)
}
