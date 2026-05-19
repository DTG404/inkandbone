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

func TestFactionCRUD_API(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	// Create
	body := `{"name":"The Iron Legion","description":"A militaristic order","faction_type":"faction","influence":8,"resources_json":"{\"gold\":500}","color":"#c9a84c"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/campaigns/%d/factions", campaignID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	factionID := int64(created["id"].(float64))

	// List
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/campaigns/%d/factions", campaignID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var list []db.Faction
	require.NoError(t, json.NewDecoder(w.Body).Decode(&list))
	require.Len(t, list, 1)
	assert.Equal(t, "The Iron Legion", list[0].Name)
	assert.Equal(t, 8, list[0].Influence)

	// Get single
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/factions/%d", factionID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var single db.Faction
	require.NoError(t, json.NewDecoder(w.Body).Decode(&single))
	assert.Equal(t, "The Iron Legion", single.Name)

	// Update
	updateBody := `{"name":"The Iron Legion (Reform)","description":"Reformed order","faction_type":"guild","influence":5,"resources_json":"{}","color":"#ff0000"}`
	req = httptest.NewRequest("PATCH", fmt.Sprintf("/api/factions/%d", factionID), strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify update
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/factions/%d", factionID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.NoError(t, json.NewDecoder(w.Body).Decode(&single))
	assert.Equal(t, "The Iron Legion (Reform)", single.Name)
	assert.Equal(t, "guild", single.FactionType)
	assert.Equal(t, "#ff0000", single.Color)

	// Delete
	req = httptest.NewRequest("DELETE", fmt.Sprintf("/api/factions/%d", factionID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	factions, _ := s.db.ListFactions(campaignID)
	assert.Len(t, factions, 0)
}

func TestCreateFaction_requiresName(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/campaigns/%d/factions", campaignID), strings.NewReader(`{"description":"no name"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetFaction_notFound(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/factions/99999", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateFaction_requiresName(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	// Create one first
	id, err := s.db.CreateFaction(campaignID, "Test", "", "faction", 5, "{}", "#c9a84c")
	require.NoError(t, err)

	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/factions/%d", id), strings.NewReader(`{"name":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFactionList_defaults(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	// Create with minimal fields
	body := `{"name":"Cult of Shadows"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/campaigns/%d/factions", campaignID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	factionID := int64(created["id"].(float64))

	f, err := s.db.GetFaction(factionID)
	require.NoError(t, err)
	assert.Equal(t, "faction", f.FactionType)
	assert.Equal(t, 5, f.Influence)
	assert.Equal(t, "#c9a84c", f.Color)
}
