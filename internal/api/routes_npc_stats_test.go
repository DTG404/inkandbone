package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/digitalghost404/inkandbone/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNpcStatCRUD_API(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	// Create
	body := `{"name":"Goblin Scout","role":"scout","data_json":"{\"str\":8,\"dex\":14}","hp_max":7,"armor_class":15,"initiative_mod":2,"skills":"[\"Stealth +4\"]","abilities":"[\"Nimble Escape\"]","loot":"[\"Shortbow\",\"Dagger\"]","notes":"A nimble goblin lookout"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/campaigns/%d/npc-stats", campaignID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	npcStatID := int64(created["id"].(float64))

	// List
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/campaigns/%d/npc-stats", campaignID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var list []db.NpcStat
	require.NoError(t, json.NewDecoder(w.Body).Decode(&list))
	require.Len(t, list, 1)
	assert.Equal(t, "Goblin Scout", list[0].Name)
	assert.Equal(t, 7, list[0].HPMax)
	require.NotNil(t, list[0].ArmorClass)
	assert.Equal(t, 15, *list[0].ArmorClass)

	// Get single
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/npc-stats/%d", npcStatID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var single db.NpcStat
	require.NoError(t, json.NewDecoder(w.Body).Decode(&single))
	assert.Equal(t, "Goblin Scout", single.Name)

	// Update
	updateBody := `{"name":"Goblin Scout (Elite)","role":"scout","data_json":"{}","hp_max":14,"armor_class":17,"initiative_mod":3,"skills":"[\"Stealth +6\"]","abilities":"[\"Nimble Escape\"]","loot":"[]","notes":""}`
	req = httptest.NewRequest("PATCH", fmt.Sprintf("/api/npc-stats/%d", npcStatID), strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify update
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/npc-stats/%d", npcStatID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.NoError(t, json.NewDecoder(w.Body).Decode(&single))
	assert.Equal(t, "Goblin Scout (Elite)", single.Name)
	assert.Equal(t, 14, single.HPMax)
	require.NotNil(t, single.ArmorClass)
	assert.Equal(t, 17, *single.ArmorClass)

	// Delete
	req = httptest.NewRequest("DELETE", fmt.Sprintf("/api/npc-stats/%d", npcStatID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	stats, _ := s.db.ListNpcStats(campaignID)
	assert.Len(t, stats, 0)
}

func TestCreateNpcStat_requiresName(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/campaigns/%d/npc-stats", campaignID), strings.NewReader(`{"hp_max":10}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetNpcStat_notFound(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/npc-stats/99999", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateNpcStat_requiresName(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	id, err := s.db.CreateNpcStat(campaignID, "Test", "", "{}", 5, nil, 0, "[]", "[]", "[]", "")
	require.NoError(t, err)

	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/npc-stats/%d", id), strings.NewReader(`{"name":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNpcStatList_defaults(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	body := `{"name":"Gelatinous Cube"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/campaigns/%d/npc-stats", campaignID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	npcStatID := int64(created["id"].(float64))

	n, err := s.db.GetNpcStat(npcStatID)
	require.NoError(t, err)
	assert.Equal(t, 1, n.HPMax)
	assert.Nil(t, n.ArmorClass)
	assert.Equal(t, 0, n.InitiativeMod)
}
