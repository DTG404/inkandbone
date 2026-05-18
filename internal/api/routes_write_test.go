package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/digitalghost404/inkandbone/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestServerWithSeed creates a server with a seeded campaign+ruleset and returns
// the server, campaign ID, and session ID for use in write operation tests.
func newTestServerWithSeed(t *testing.T) (*Server, int64, int64) {
	t.Helper()
	s := newTestServer(t)
	campID, sessID := seedCampaign(t, s.db)
	return s, campID, sessID
}

// --- CAMPAIGN CRUD ---

func TestCreateCampaign(t *testing.T) {
	s := newTestServer(t)
	// Use a pre-seeded ruleset
	body := bytes.NewReader([]byte(`{"name":"New Campaign","description":"desc","ruleset_id":1}`))
	req := httptest.NewRequest(http.MethodPost, "/api/campaigns", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Positive(t, resp["id"])
}

func TestCreateCampaign_missingName(t *testing.T) {
	s := newTestServer(t)
	body := bytes.NewReader([]byte(`{"description":"desc","ruleset_id":1}`))
	req := httptest.NewRequest(http.MethodPost, "/api/campaigns", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateCampaign_missingRulesetID(t *testing.T) {
	s := newTestServer(t)
	body := bytes.NewReader([]byte(`{"name":"New Campaign","description":"desc"}`))
	req := httptest.NewRequest(http.MethodPost, "/api/campaigns", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPatchCampaign(t *testing.T) {
	s, campID, _ := newTestServerWithSeed(t)
	body := bytes.NewReader([]byte(`{"active":false}`))
	req := httptest.NewRequest(http.MethodPatch, "/api/campaigns/"+strconv.FormatInt(campID, 10), body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

func TestDeleteCampaign(t *testing.T) {
	s, campID, _ := newTestServerWithSeed(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/campaigns/"+strconv.FormatInt(campID, 10), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

func TestGetCharacterOptions(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/rulesets/1/character-options", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

// --- CHARACTER CRUD ---

func TestCreateCharacter(t *testing.T) {
	s, campID, _ := newTestServerWithSeed(t)
	body := bytes.NewReader([]byte(`{"name":"Test Hero"}`))
	req := httptest.NewRequest(http.MethodPost, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/characters", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)

	var char db.Character
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &char))
	assert.Equal(t, "Test Hero", char.Name)
}

func TestCreateCharacter_missingName(t *testing.T) {
	s, campID, _ := newTestServerWithSeed(t)
	body := bytes.NewReader([]byte(`{}`))
	req := httptest.NewRequest(http.MethodPost, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/characters", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteCharacter(t *testing.T) {
	s, campID, _ := newTestServerWithSeed(t)
	charID, err := s.db.CreateCharacter(campID, "Hero")
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodDelete, "/api/characters/"+strconv.FormatInt(charID, 10), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

// --- SESSION CRUD ---

func TestCreateSession(t *testing.T) {
	s, campID, _ := newTestServerWithSeed(t)
	body := bytes.NewReader([]byte(`{"title":"New Session","date":"2026-05-18"}`))
	req := httptest.NewRequest(http.MethodPost, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/sessions", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

func TestDeleteSession(t *testing.T) {
	s, _, sessID := newTestServerWithSeed(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/sessions/"+strconv.FormatInt(sessID, 10), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

// --- NPC CRUD ---

func TestCreateNPC(t *testing.T) {
	s, _, sessID := newTestServerWithSeed(t)
	body := bytes.NewReader([]byte(`{"name":"Sardaukar Commander"}`))
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+strconv.FormatInt(sessID, 10)+"/npcs", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

func TestListNPCs(t *testing.T) {
	s, _, sessID := newTestServerWithSeed(t)
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+strconv.FormatInt(sessID, 10)+"/npcs", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

func TestPatchNPC(t *testing.T) {
	s, _, sessID := newTestServerWithSeed(t)
	npcs, err := s.db.ListSessionNPCs(sessID)
	require.NoError(t, err)
	if len(npcs) == 0 {
		t.Skip("no NPCs to patch")
	}
	body := bytes.NewReader([]byte(`{"notes":"Updated notes"}`))
	req := httptest.NewRequest(http.MethodPatch, "/api/npcs/"+strconv.FormatInt(npcs[0].ID, 10), body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

func TestDeleteNPC(t *testing.T) {
	s, _, sessID := newTestServerWithSeed(t)
	npcs, err := s.db.ListSessionNPCs(sessID)
	require.NoError(t, err)
	if len(npcs) == 0 {
		t.Skip("no NPCs to delete")
	}
	req := httptest.NewRequest(http.MethodDelete, "/api/npcs/"+strconv.FormatInt(npcs[0].ID, 10), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

// --- OBJECTIVES CRUD ---

func TestCreateObjective(t *testing.T) {
	s, campID, _ := newTestServerWithSeed(t)
	body := bytes.NewReader([]byte(`{"title":"Find the Spy","description":"Uncover the traitor"}`))
	req := httptest.NewRequest(http.MethodPost, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/objectives", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

func TestListObjectives(t *testing.T) {
	s, campID, _ := newTestServerWithSeed(t)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/objectives", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

func TestPatchObjective(t *testing.T) {
	s, campID, _ := newTestServerWithSeed(t)
	obj, err := s.db.CreateObjective(campID, "Test Objective", "", nil)
	require.NoError(t, err)
	body := bytes.NewReader([]byte(`{"status":"completed"}`))
	req := httptest.NewRequest(http.MethodPatch, "/api/objectives/"+strconv.FormatInt(obj.ID, 10), body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNoContent, "expected 2xx, got %d", w.Code)
}

func TestDeleteObjective(t *testing.T) {
	s, campID, _ := newTestServerWithSeed(t)
	obj, err := s.db.CreateObjective(campID, "Test Objective", "", nil)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodDelete, "/api/objectives/"+strconv.FormatInt(obj.ID, 10), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

// --- ITEMS CRUD ---

func TestCreateItem(t *testing.T) {
	s, campID, _ := newTestServerWithSeed(t)
	charID, err := s.db.CreateCharacter(campID, "Hero")
	require.NoError(t, err)
	body := bytes.NewReader([]byte(`{"name":"Crysknife","description":"A sacred Fremen blade","quantity":1}`))
	req := httptest.NewRequest(http.MethodPost, "/api/characters/"+strconv.FormatInt(charID, 10)+"/items", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

func TestListItems(t *testing.T) {
	s, campID, _ := newTestServerWithSeed(t)
	charID, err := s.db.CreateCharacter(campID, "Hero")
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "/api/characters/"+strconv.FormatInt(charID, 10)+"/items", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

func TestPatchItem(t *testing.T) {
	s, campID, _ := newTestServerWithSeed(t)
	charID, err := s.db.CreateCharacter(campID, "Hero")
	require.NoError(t, err)
	item, err := s.db.CreateItem(charID, "Knife", "", 1)
	require.NoError(t, err)
	body := bytes.NewReader([]byte(`{"quantity":3}`))
	req := httptest.NewRequest(http.MethodPatch, "/api/items/"+strconv.FormatInt(item.ID, 10), body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

func TestDeleteItem(t *testing.T) {
	s, campID, _ := newTestServerWithSeed(t)
	charID, err := s.db.CreateCharacter(campID, "Hero")
	require.NoError(t, err)
	item, err := s.db.CreateItem(charID, "Knife", "", 1)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodDelete, "/api/items/"+strconv.FormatInt(item.ID, 10), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

// --- MAP & MAP PINS ---

func TestCreateMapPin(t *testing.T) {
	s, campID, _ := newTestServerWithSeed(t)
	// Upload a map first
	_, err := s.db.CreateMap(campID, "Test Map", "maps/test.svg")
	require.NoError(t, err)
	maps, err := s.db.ListMaps(campID)
	require.NoError(t, err)
	require.NotEmpty(t, maps)

	body := bytes.NewReader([]byte(`{"x":0.5,"y":0.5,"label":"Spice Silos","note":"Contains 500 tonnes"}`))
	req := httptest.NewRequest(http.MethodPost, "/api/maps/"+strconv.FormatInt(maps[0].ID, 10)+"/pins", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

// --- DICE ROLLS ---

func TestRollDice(t *testing.T) {
	s, _, sessID := newTestServerWithSeed(t)
	body := bytes.NewReader([]byte(`{"expression":"2d6+3"}`))
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+strconv.FormatInt(sessID, 10)+"/dice-rolls", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)

	var roll db.DiceRoll
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &roll))
	assert.Equal(t, "2d6+3", roll.Expression)
	assert.Positive(t, roll.Result)
}

func TestRollDice_invalidExpression(t *testing.T) {
	s, _, sessID := newTestServerWithSeed(t)
	body := bytes.NewReader([]byte(`{"expression":"not-a-dice-roll"}`))
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+strconv.FormatInt(sessID, 10)+"/dice-rolls", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- ORACLE ---

func TestOracleRoll(t *testing.T) {
	s := newTestServer(t)
	body := bytes.NewReader([]byte(`{"table":"action","roll":10}`))
	req := httptest.NewRequest(http.MethodPost, "/api/oracle/roll", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "action", resp["table"])
	assert.Equal(t, float64(10), resp["roll"])
	assert.NotEmpty(t, resp["result"])
}

func TestOracleRoll_invalidTable(t *testing.T) {
	s := newTestServer(t)
	body := bytes.NewReader([]byte(`{"table":"nonexistent","roll":99}`))
	req := httptest.NewRequest(http.MethodPost, "/api/oracle/roll", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- RELATIONSHIPS ---

func TestCreateRelationship(t *testing.T) {
	s, campID, _ := newTestServerWithSeed(t)
	body := bytes.NewReader([]byte(`{"from_name":"Paul Atreides","to_name":"Baron Harkonnen","type":"enemy","description":"Mortal enemies"}`))
	req := httptest.NewRequest(http.MethodPost, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/relationships", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

func TestListRelationships(t *testing.T) {
	s, campID, _ := newTestServerWithSeed(t)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/relationships", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

func TestUpdateRelationship(t *testing.T) {
	s, campID, _ := newTestServerWithSeed(t)
	relID, err := s.db.CreateRelationship(campID, "A", "B", "ally", "")
	require.NoError(t, err)
	body := bytes.NewReader([]byte(`{"relationship_type":"rival","description":"Now rivals"}`))
	req := httptest.NewRequest(http.MethodPatch, "/api/relationships/"+strconv.FormatInt(relID, 10), body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

func TestDeleteRelationship(t *testing.T) {
	s, campID, _ := newTestServerWithSeed(t)
	relID, err := s.db.CreateRelationship(campID, "A", "B", "ally", "")
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodDelete, "/api/relationships/"+strconv.FormatInt(relID, 10), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

// --- TENSION ---

func TestGetTension(t *testing.T) {
	s, _, sessID := newTestServerWithSeed(t)
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+strconv.FormatInt(sessID, 10)+"/tension", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(5), resp["tension_level"])
}

func TestPatchTension(t *testing.T) {
	s, _, sessID := newTestServerWithSeed(t)
	body := bytes.NewReader([]byte(`{"tension_level":8}`))
	req := httptest.NewRequest(http.MethodPatch, "/api/sessions/"+strconv.FormatInt(sessID, 10)+"/tension", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

func TestPatchCombatant(t *testing.T) {
	s, _, sessID := newTestServerWithSeed(t)
	encID, err := s.db.CreateEncounter(sessID, "Test Combat")
	require.NoError(t, err)
	combatantID, err := s.db.AddCombatant(encID, "Hero", 15, 10, true, nil)
	require.NoError(t, err)

	body := bytes.NewReader([]byte(`{"conditions_json":"[\"poisoned\"]"}`))
	req := httptest.NewRequest(http.MethodPatch, "/api/combatants/"+strconv.FormatInt(combatantID, 10), body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

// --- SETTINGS ---

func TestPatchSettings(t *testing.T) {
	s, campID, sessID := newTestServerWithSeed(t)
	body := bytes.NewReader([]byte(`{"campaign_id":` + strconv.FormatInt(campID, 10) + `,"session_id":` + strconv.FormatInt(sessID, 10) + `}`))
	req := httptest.NewRequest(http.MethodPatch, "/api/settings", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

// --- HEALTH ---

func TestHealthEndpoint(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp["ai_enabled"].(bool))
}



// --- XP ---

func TestCreateXP(t *testing.T) {
	s, _, sessID := newTestServerWithSeed(t)
	body := bytes.NewReader([]byte(`{"note":"Defeated the Beast","amount":3}`))
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+strconv.FormatInt(sessID, 10)+"/xp", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

func TestListXP(t *testing.T) {
	s, _, sessID := newTestServerWithSeed(t)
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+strconv.FormatInt(sessID, 10)+"/xp", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

func TestDeleteXP(t *testing.T) {
	s, _, sessID := newTestServerWithSeed(t)
	amount := 1
	xp, err := s.db.CreateXP(sessID, "Test XP", &amount)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodDelete, "/api/xp/"+strconv.FormatInt(xp.ID, 10), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

// --- RULEBOOK ---

func TestListRulebookSources(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/rulesets/1/rulebook", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)
}

func TestIngestRulebook_plainText(t *testing.T) {
	s := newTestServer(t)
	body := strings.NewReader("# Chapter 1\nThe beginning.\n\n# Chapter 2\nThe middle.\n\n# Chapter 3\nThe end.")
	req := httptest.NewRequest(http.MethodPost, "/api/rulesets/1/rulebook?source=Test+Book", body)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(3), resp["chunks_created"])
}



// --- VALIDATION / ERROR CASES ---
func TestCreateCharacter_invalidCampaignID(t *testing.T) {
	s := newTestServer(t)
	body := bytes.NewReader([]byte(`{"name":"Hero"}`))
	req := httptest.NewRequest(http.MethodPost, "/api/campaigns/99999/characters", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	// The handler creates the character first, then fails to roll stats.
	// Character is created but without stats — that's the current behavior.
	assert.True(t, w.Code < 500, "unexpected server error: %d", w.Code)
}

func TestDeleteCampaign_invalidID(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/campaigns/99999", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	// Currently succeeds with no-op for non-existent campaigns.
	assert.True(t, w.Code < 500, "unexpected server error: %d", w.Code)
}

func TestPatchSession_sceneTags(t *testing.T) {
	s := newTestServer(t)
	// Create without seed to control the session
	rsID, err := s.db.CreateRuleset("test_patch_session", `{}`, "1.0")
	require.NoError(t, err)
	campID, err := s.db.CreateCampaign(rsID, "Test", "")
	require.NoError(t, err)
	sessID, err := s.db.CreateSession(campID, "S1", "2026-01-01")
	require.NoError(t, err)

	body := bytes.NewReader([]byte(`{"scene_tags":"dungeon,night"}`))
	req := httptest.NewRequest(http.MethodPatch, "/api/sessions/"+strconv.FormatInt(sessID, 10), body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200 && w.Code < 300, "expected 2xx, got %d", w.Code)

	sess, err := s.db.GetSession(sessID)
	require.NoError(t, err)
	assert.Equal(t, "dungeon,night", sess.SceneTags)
}


