package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/digitalghost404/inkandbone/internal/ai"
	"github.com/digitalghost404/inkandbone/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedCampaign creates a ruleset, campaign, and session for use in route tests.
// It uses t.Name() as the ruleset name to avoid conflicts with pre-seeded rulesets.
func seedCampaign(t *testing.T, d *db.DB) (campID, sessID int64) {
	t.Helper()
	rsID, err := d.CreateRuleset(t.Name(), `{}`, "test")
	require.NoError(t, err)
	campID, err = d.CreateCampaign(rsID, "Test Campaign", "")
	require.NoError(t, err)
	sessID, err = d.CreateSession(campID, "S1", "2026-04-03")
	require.NoError(t, err)
	return
}

func TestListCampaigns_empty(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var campaigns []db.Campaign
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &campaigns))
	assert.Empty(t, campaigns)
}

func TestListCampaigns_withData(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var campaigns []db.Campaign
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &campaigns))
	require.Len(t, campaigns, 1)
	assert.Equal(t, campID, campaigns[0].ID)
}

func TestListCharacters_empty(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/characters", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var chars []db.Character
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &chars))
	assert.Empty(t, chars)
}

func TestListCharacters_withData(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)
	charID, err := s.db.CreateCharacter(campID, "Kael")
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/characters", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var chars []db.Character
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &chars))
	require.Len(t, chars, 1)
	assert.Equal(t, charID, chars[0].ID)
}

func TestListSessions_withData(t *testing.T) {
	s := newTestServer(t)
	campID, sessID := seedCampaign(t, s.db)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/sessions", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var sessions []db.Session
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sessions))
	require.Len(t, sessions, 1)
	assert.Equal(t, sessID, sessions[0].ID)
}

func TestListMessages_empty(t *testing.T) {
	s := newTestServer(t)
	_, sessID := seedCampaign(t, s.db)
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+strconv.FormatInt(sessID, 10)+"/messages", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var msgs []db.Message
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &msgs))
	assert.Empty(t, msgs)
}

func TestListDiceRolls_empty(t *testing.T) {
	s := newTestServer(t)
	_, sessID := seedCampaign(t, s.db)
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+strconv.FormatInt(sessID, 10)+"/dice-rolls", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var rolls []db.DiceRoll
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rolls))
	assert.Empty(t, rolls)
}

func TestListMapPins_empty(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)
	mapID, err := s.db.CreateMap(campID, "World Map", "")
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "/api/maps/"+strconv.FormatInt(mapID, 10)+"/pins", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var pins []db.MapPin
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pins))
	assert.Empty(t, pins)
}

func TestGetContext_empty(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/context", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp contextResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Nil(t, resp.Campaign)
	assert.Nil(t, resp.Character)
	assert.Nil(t, resp.Session)
	assert.Empty(t, resp.RecentMessages)
	assert.Nil(t, resp.ActiveCombat)
}

func TestListWorldNotes_empty(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/world-notes", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var notes []db.WorldNote
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &notes))
	assert.Empty(t, notes)
}

func TestListWorldNotes_withData(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)
	_, err := s.db.CreateWorldNote(campID, "Tavern", "A seedy place.", "location")
	require.NoError(t, err)
	_, err = s.db.CreateWorldNote(campID, "Dragon", "Ancient red dragon.", "npc")
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/world-notes", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var notes []db.WorldNote
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &notes))
	require.Len(t, notes, 2)
}

func TestListWorldNotes_searchFilter(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)
	_, err := s.db.CreateWorldNote(campID, "Tavern", "A seedy place.", "location")
	require.NoError(t, err)
	_, err = s.db.CreateWorldNote(campID, "Dragon", "Ancient red dragon.", "npc")
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/world-notes?q=tavern", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var notes []db.WorldNote
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &notes))
	require.Len(t, notes, 1)
	assert.Equal(t, "Tavern", notes[0].Title)
}

func TestListWorldNotes_categoryFilter(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)
	_, err := s.db.CreateWorldNote(campID, "Tavern", "A seedy place.", "location")
	require.NoError(t, err)
	_, err = s.db.CreateWorldNote(campID, "Dragon", "Ancient red dragon.", "npc")
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/world-notes?category=location", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var notes []db.WorldNote
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &notes))
	require.Len(t, notes, 1)
	assert.Equal(t, "Tavern", notes[0].Title)
}

func TestGetContext_withActiveState(t *testing.T) {
	s := newTestServer(t)
	campID, sessID := seedCampaign(t, s.db)
	charID, err := s.db.CreateCharacter(campID, "Arin")
	require.NoError(t, err)
	require.NoError(t, s.db.SetSetting("active_campaign_id", strconv.FormatInt(campID, 10)))
	require.NoError(t, s.db.SetSetting("active_character_id", strconv.FormatInt(charID, 10)))
	require.NoError(t, s.db.SetSetting("active_session_id", strconv.FormatInt(sessID, 10)))

	req := httptest.NewRequest(http.MethodGet, "/api/context", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp contextResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Campaign)
	assert.Equal(t, "Test Campaign", resp.Campaign.Name)
	require.NotNil(t, resp.Character)
	assert.Equal(t, "Arin", resp.Character.Name)
	require.NotNil(t, resp.Session)
	assert.Equal(t, "S1", resp.Session.Title)
	assert.Nil(t, resp.ActiveCombat)
}

func TestListWorldNotes_tagFilter(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)
	noteID, err := s.db.CreateWorldNote(campID, "Shrine", "Ancient shrine.", "location")
	require.NoError(t, err)
	require.NoError(t, s.db.UpdateWorldNote(noteID, "Shrine", "Ancient shrine.", `["dungeon"]`))
	_, err = s.db.CreateWorldNote(campID, "Merchant", "Sells goods.", "npc")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/world-notes?tag=dungeon", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var notes []db.WorldNote
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &notes))
	require.Len(t, notes, 1)
	assert.Equal(t, "Shrine", notes[0].Title)
}

func TestPatchWorldNote_updatesNote(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)
	noteID, err := s.db.CreateWorldNote(campID, "Old Title", "Old content", "npc")
	require.NoError(t, err)

	body := `{"title":"New Title","content":"New content","tags_json":"[\"ally\"]"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/world-notes/"+strconv.FormatInt(noteID, 10), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	notes, err := s.db.SearchWorldNotes(campID, "New Title", "", "")
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, "New content", notes[0].Content)
	assert.Contains(t, notes[0].TagsJSON, "ally")
}

func TestPatchWorldNote_invalidID(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodPatch, "/api/world-notes/abc", strings.NewReader(`{"title":"x","content":"y"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPatchWorldNote_missingTitle(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)
	noteID, err := s.db.CreateWorldNote(campID, "A Note", "Content.", "npc")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPatch, "/api/world-notes/"+strconv.FormatInt(noteID, 10), strings.NewReader(`{"content":"y"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServeFile_notFound(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/files/portraits/nonexistent.jpg", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestServeFile_traversalBlocked(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/files/../../etc/passwd", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestGetTimeline_empty(t *testing.T) {
	s := newTestServer(t)
	_, sessID := seedCampaign(t, s.db)
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+strconv.FormatInt(sessID, 10)+"/timeline", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var entries []db.TimelineEntry
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &entries))
	assert.Empty(t, entries)
}

func TestGetTimeline_withData(t *testing.T) {
	s := newTestServer(t)
	_, sessID := seedCampaign(t, s.db)
	_, err := s.db.CreateMessage(sessID, "user", "A brave move.", false, nil)
	require.NoError(t, err)
	_, err = s.db.LogDiceRoll(sessID, "2d6", 9, "[4,5]")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+strconv.FormatInt(sessID, 10)+"/timeline", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var entries []db.TimelineEntry
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &entries))
	assert.Len(t, entries, 2)
}

func TestGetTimeline_invalidID(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/abc/timeline", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServeFile_ok(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("world"), 0600))
	s := newTestServerWithDir(t, dir)
	req := httptest.NewRequest(http.MethodGet, "/api/files/hello.txt", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "world", w.Body.String())
}

func TestServeFile_traversal(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/files/../etc/passwd", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestListMaps_empty(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/maps", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var maps []db.Map
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &maps))
	assert.Empty(t, maps)
}

func TestGetMap_found(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)
	mapID, err := s.db.CreateMap(campID, "World", "maps/test.jpg")
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "/api/maps/"+strconv.FormatInt(mapID, 10), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var m db.Map
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &m))
	assert.Equal(t, mapID, m.ID)
	assert.Equal(t, "World", m.Name)
}

func TestGetMap_notFound(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/maps/9999", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPatchSession_ok(t *testing.T) {
	s := newTestServer(t)
	_, sessID := seedCampaign(t, s.db)
	body := `{"summary":"Session went great"}`
	req := httptest.NewRequest(http.MethodPatch,
		"/api/sessions/"+strconv.FormatInt(sessID, 10),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
	sess, err := s.db.GetSession(sessID)
	require.NoError(t, err)
	assert.Equal(t, "Session went great", sess.Summary)
}

func TestDraftWorldNote_ok(t *testing.T) {
	stub := &stubCompleter{response: "Title: Zara the Smith\nContent: A dwarven blacksmith known for fine steel."}
	s := newTestServerWithAI(t, stub)
	campID, _ := seedCampaign(t, s.db)

	body := `{"hint":"Dwarven blacksmith NPC"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/campaigns/"+strconv.FormatInt(campID, 10)+"/world-notes/draft",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	var note db.WorldNote
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &note))
	assert.NotZero(t, note.ID)
	assert.Equal(t, campID, note.CampaignID)
	assert.Equal(t, "Zara the Smith", note.Title)
}

func TestDraftWorldNote_noAI(t *testing.T) {
	s := newTestServer(t) // aiClient is nil
	campID, _ := seedCampaign(t, s.db)
	body := `{"hint":"test"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/campaigns/"+strconv.FormatInt(campID, 10)+"/world-notes/draft",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGenerateRecap_ok(t *testing.T) {
	stub := &stubCompleter{response: "The party defeated the goblin horde."}
	s := newTestServerWithAI(t, stub)
	_, sessID := seedCampaign(t, s.db)

	req := httptest.NewRequest(http.MethodPost,
		"/api/sessions/"+strconv.FormatInt(sessID, 10)+"/recap",
		nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Summary string `json:"summary"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "The party defeated the goblin horde.", resp.Summary)
	// Verify DB was updated
	sess, err := s.db.GetSession(sessID)
	require.NoError(t, err)
	assert.Equal(t, "The party defeated the goblin horde.", sess.Summary)
}

func TestGenerateRecap_noAI(t *testing.T) {
	s := newTestServer(t)
	_, sessID := seedCampaign(t, s.db)
	req := httptest.NewRequest(http.MethodPost,
		"/api/sessions/"+strconv.FormatInt(sessID, 10)+"/recap",
		nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestBuildWorldContext_ActiveObjectives(t *testing.T) {
	s := newTestServer(t)
	campID, sessID := seedCampaign(t, s.db)

	_, err := s.db.CreateObjective(campID, "Find the lost sword", "main quest", nil)
	require.NoError(t, err)

	ctx := s.buildWorldContext(t.Context(), sessID)
	assert.Contains(t, ctx, "Find the lost sword")
	assert.Contains(t, ctx, "OBJECTIVES")
}

func TestBuildWorldContext_NPCPersonality(t *testing.T) {
	s := newTestServer(t)
	campID, sessID := seedCampaign(t, s.db)

	_, err := s.db.CreateWorldNote(campID, "Elara", "A skilled merchant", "npc")
	require.NoError(t, err)
	note, err := s.db.FindWorldNoteByTitle(campID, "Elara")
	require.NoError(t, err)
	require.NotNil(t, note)
	err = s.db.UpdateWorldNotePersonality(note.ID, `{"traits":["cunning"],"motivation":"profit"}`)
	require.NoError(t, err)

	ctx := s.buildWorldContext(t.Context(), sessID)
	assert.Contains(t, ctx, "Elara")
	assert.Contains(t, ctx, "cunning")
}

func TestBuildWorldContext_Setting(t *testing.T) {
	s := newTestServer(t)

	// Use a seeded ruleset that has gm_context populated by migration 018.
	rs, err := s.db.GetRulesetByName("dnd5e")
	require.NoError(t, err)
	require.NotNil(t, rs)
	require.NotEmpty(t, rs.GMContext, "dnd5e gm_context must be set by migration 018")

	campID, err := s.db.CreateCampaign(rs.ID, "Test Campaign", "")
	require.NoError(t, err)
	sessID, err := s.db.CreateSession(campID, "S1", "2026-04-06")
	require.NoError(t, err)

	ctx := s.buildWorldContext(t.Context(), sessID)
	assert.Contains(t, ctx, "[SETTING]")
	assert.Contains(t, ctx, "[/SETTING]")
	// Verify the actual ruleset content is present (dnd5e context mentions high fantasy).
	assert.Contains(t, ctx, rs.GMContext)
}

func TestBuildWorldContext_SettingAbsentForCustomRuleset(t *testing.T) {
	s := newTestServer(t)

	// seedCampaign creates a custom ruleset with empty gm_context — no [SETTING] block expected.
	_, sessID := seedCampaign(t, s.db)

	ctx := s.buildWorldContext(t.Context(), sessID)
	assert.NotContains(t, ctx, "[SETTING]")
}

func TestPatchWorldNotePersonality(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)

	// Create a world note
	_, err := s.db.CreateWorldNote(campID, "Elara", "A skilled merchant", "npc")
	require.NoError(t, err)
	note, err := s.db.FindWorldNoteByTitle(campID, "Elara")
	require.NoError(t, err)

	personality := `{"traits":["cunning"],"motivation":"profit"}`
	body := map[string]string{"personality_json": personality}
	bodyJSON, _ := json.Marshal(body)

	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/world-notes/%d/personality", note.ID), bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	updated, err := s.db.GetWorldNote(note.ID)
	require.NoError(t, err)
	assert.Equal(t, personality, updated.PersonalityJSON)
}

// stubCompleterStreamer implements both ai.Completer and ai.Streamer for testing
// handleGMRespondStream. Generate returns a fixed response; StreamRespond
// captures the system prompt and writes a minimal SSE response.
type stubCompleterStreamer struct {
	generateResp string
	capturedSys  string
	streamResp   string
}

func (s *stubCompleterStreamer) Generate(_ context.Context, _ string, _ int) (string, error) {
	return s.generateResp, nil
}

func (s *stubCompleterStreamer) StreamRespond(_ context.Context, system string, _ []ai.ChatMessage, _ int, w http.ResponseWriter) (string, error) {
	s.capturedSys = system
	fmt.Fprintf(w, "data: %s\n\n", s.streamResp)
	return s.streamResp, nil
}

func TestHandleGMRespondStream_FailureDirection(t *testing.T) {
	// The stub's Generate returns a roll requiring DC 100 — guaranteed failure
	// on any dice roll. StreamRespond captures the system prompt.
	stub := &stubCompleterStreamer{
		generateResp: `{"required":true,"expression":"1d20","attribute":"Strength","dc":100,"reason":"forcing a door"}`,
		streamResp:   "You fail to open the door.",
	}
	s := newTestServerWithAI(t, stub)
	_, sessID := seedCampaign(t, s.db)

	// Seed a user message so handleGMRespondStream has something to respond to.
	_, err := s.db.CreateMessage(sessID, "user", "I try to kick down the door.", false, nil)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost,
		"/api/sessions/"+strconv.FormatInt(sessID, 10)+"/gm-respond-stream",
		nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, stub.capturedSys, "[GM DIRECTION]")
	assert.Contains(t, stub.capturedSys, "FAILED")
}

func TestUploadMap_ok(t *testing.T) {
	dir := t.TempDir()
	s := newTestServerWithDir(t, dir)
	campID, _ := seedCampaign(t, s.db)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("image", "map.png")
	require.NoError(t, err)
	_, err = io.WriteString(fw, "fake-image-data")
	require.NoError(t, err)
	require.NoError(t, mw.WriteField("name", "World Map"))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost,
		"/api/campaigns/"+strconv.FormatInt(campID, 10)+"/maps",
		&body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	var m db.Map
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &m))
	assert.Equal(t, campID, m.CampaignID)
	assert.Equal(t, "World Map", m.Name)
	assert.True(t, strings.HasPrefix(m.ImagePath, "maps/"))
	assert.FileExists(t, filepath.Join(dir, "maps", filepath.Base(m.ImagePath)))
}

func TestHandlePatchSession_SceneTags(t *testing.T) {
	s := newTestServer(t)
	_, sessID := seedCampaign(t, s.db)

	body := `{"scene_tags":"dungeon,battle"}`
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/sessions/%d", sessID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	sess, err := s.db.GetSession(sessID)
	require.NoError(t, err)
	assert.Equal(t, "dungeon,battle", sess.SceneTags)
}

func TestAutoUpdateSceneTags_setsTag(t *testing.T) {
	s := newTestServer(t)
	_, sessID := seedCampaign(t, s.db)

	s.autoUpdateSceneTags(context.Background(), sessID, "You descend into the stone corridor.")

	sess, err := s.db.GetSession(sessID)
	require.NoError(t, err)
	assert.Equal(t, "dungeon", sess.SceneTags)
}

func TestAutoUpdateSceneTags_stability_noOpWhenSameTag(t *testing.T) {
	s := newTestServer(t)
	_, sessID := seedCampaign(t, s.db)
	require.NoError(t, s.db.UpdateSceneTags(sessID, "dungeon"))

	s.autoUpdateSceneTags(context.Background(), sessID, "The dungeon corridor stretches ahead.")

	sess, err := s.db.GetSession(sessID)
	require.NoError(t, err)
	assert.Equal(t, "dungeon", sess.SceneTags)
}

func TestAutoUpdateSceneTags_noKeywordMatch_noChange(t *testing.T) {
	s := newTestServer(t)
	_, sessID := seedCampaign(t, s.db)

	s.autoUpdateSceneTags(context.Background(), sessID, "You board the vessel.")

	sess, err := s.db.GetSession(sessID)
	require.NoError(t, err)
	assert.Equal(t, "", sess.SceneTags)
}

func TestAutoUpdateSceneTags_battleKeyword(t *testing.T) {
	s := newTestServer(t)
	_, sessID := seedCampaign(t, s.db)

	s.autoUpdateSceneTags(context.Background(), sessID, "The enemy attacks and combat breaks out.")

	sess, err := s.db.GetSession(sessID)
	require.NoError(t, err)
	assert.Equal(t, "battle", sess.SceneTags)
}

func TestAutoUpdateSceneTags_emptyText_noOp(t *testing.T) {
	s := newTestServer(t)
	_, sessID := seedCampaign(t, s.db)

	s.autoUpdateSceneTags(context.Background(), sessID, "")

	sess, err := s.db.GetSession(sessID)
	require.NoError(t, err)
	assert.Equal(t, "", sess.SceneTags)
}
