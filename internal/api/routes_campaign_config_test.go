package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCampaignConfig_empty(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/999/config", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetCampaignConfig_withData(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)

	// Create a character and session for stats
	_, err := s.db.CreateCharacter(campID, "Kael")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/config", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "", resp["description"])
	assert.Equal(t, "", resp["gm_notes"])
	assert.Equal(t, "", resp["system_prompt_override"])
	assert.Equal(t, float64(1), resp["character_count"])
	assert.Equal(t, float64(1), resp["session_count"])
	assert.NotEmpty(t, resp["ruleset_name"])
}

func TestGetCampaignConfig_invalidID(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/abc/config", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPatchCampaignConfig_all(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)

	desc := "Updated description"
	gmNotes := "GM secret notes"
	promptOverride := "You are a dark fantasy GM."

	body, _ := json.Marshal(map[string]any{
		"description":             desc,
		"gm_notes":               gmNotes,
		"system_prompt_override": promptOverride,
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify via GET
	getReq := httptest.NewRequest(http.MethodGet, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/config", nil)
	getW := httptest.NewRecorder()
	s.ServeHTTP(getW, getReq)
	assert.Equal(t, http.StatusOK, getW.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(getW.Body.Bytes(), &resp))
	assert.Equal(t, desc, resp["description"])
	assert.Equal(t, gmNotes, resp["gm_notes"])
	assert.Equal(t, promptOverride, resp["system_prompt_override"])
}

func TestPatchCampaignConfig_partial(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)

	gmNotes := "Just the notes"
	body, _ := json.Marshal(map[string]any{
		"gm_notes": gmNotes,
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify only gm_notes changed
	getReq := httptest.NewRequest(http.MethodGet, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/config", nil)
	getW := httptest.NewRecorder()
	s.ServeHTTP(getW, getReq)
	assert.Equal(t, http.StatusOK, getW.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(getW.Body.Bytes(), &resp))
	assert.Equal(t, gmNotes, resp["gm_notes"])
	assert.Equal(t, "", resp["system_prompt_override"])
}

func TestPatchCampaignConfig_emptyBody(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)

	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest(http.MethodPatch, "/api/campaigns/"+strconv.FormatInt(campID, 10)+"/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPatchCampaignConfig_notFound(t *testing.T) {
	s := newTestServer(t)

	gmNotes := "notes"
	body, _ := json.Marshal(map[string]any{
		"gm_notes": gmNotes,
	})
	req := httptest.NewRequest(http.MethodPatch, "/api/campaigns/999/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPatchCampaignConfig_invalidID(t *testing.T) {
	s := newTestServer(t)

	body, _ := json.Marshal(map[string]any{
		"gm_notes": "notes",
	})
	req := httptest.NewRequest(http.MethodPatch, "/api/campaigns/abc/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
