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

func TestAdventureCRUD_API(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	// Create
	body := `{"title":"The Lost Library","description":"Search for ancient knowledge","status":"active","sort_order":0}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/campaigns/%d/adventures", campaignID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	adventureID := int64(created["id"].(float64))

	// List
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/campaigns/%d/adventures", campaignID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var list []db.Adventure
	require.NoError(t, json.NewDecoder(w.Body).Decode(&list))
	require.Len(t, list, 1)
	assert.Equal(t, "The Lost Library", list[0].Title)
	assert.Equal(t, "active", list[0].Status)

	// Get single
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/adventures/%d", adventureID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var single db.Adventure
	require.NoError(t, json.NewDecoder(w.Body).Decode(&single))
	assert.Equal(t, "The Lost Library", single.Title)

	// Update
	updateBody := `{"title":"The Lost Library (Revised)","description":"Updated","status":"completed","sort_order":1}`
	req = httptest.NewRequest("PATCH", fmt.Sprintf("/api/adventures/%d", adventureID), strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify update
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/adventures/%d", adventureID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.NoError(t, json.NewDecoder(w.Body).Decode(&single))
	assert.Equal(t, "The Lost Library (Revised)", single.Title)
	assert.Equal(t, "completed", single.Status)
	assert.Equal(t, 1, single.SortOrder)

	// Delete
	req = httptest.NewRequest("DELETE", fmt.Sprintf("/api/adventures/%d", adventureID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	adventures, _ := s.db.ListAdventures(campaignID)
	assert.Len(t, adventures, 0)
}

func TestCreateAdventure_requiresTitle(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/campaigns/%d/adventures", campaignID), strings.NewReader(`{"description":"no title"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetAdventure_notFound(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/adventures/99999", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateAdventure_requiresTitle(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	id, err := s.db.CreateAdventure(campaignID, "Test", "", "", 0)
	require.NoError(t, err)

	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/adventures/%d", id), strings.NewReader(`{"title":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdventureList_defaults(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	// Create with minimal fields
	body := `{"title":"Mystery of the Depths"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/campaigns/%d/adventures", campaignID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	adventureID := int64(created["id"].(float64))

	a, err := s.db.GetAdventure(adventureID)
	require.NoError(t, err)
	assert.Equal(t, "upcoming", a.Status)
	assert.Equal(t, 0, a.SortOrder)
}

func TestSetSessionAdventure_API(t *testing.T) {
	s := newTestServer(t)
	campaignID, sessID := seedCampaign(t, s.db)

	// Create an adventure
	advID, err := s.db.CreateAdventure(campaignID, "Arc 1", "", "active", 0)
	require.NoError(t, err)

	// Assign session to adventure
	body := fmt.Sprintf(`{"adventure_id":%d}`, advID)
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/sessions/%d/adventure", sessID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify adventure_id set via DB
	sess, err := s.db.GetSession(sessID)
	require.NoError(t, err)
	require.NotNil(t, sess.AdventureID)
	assert.Equal(t, advID, *sess.AdventureID)

	// Clear adventure assignment
	body = `{"adventure_id":null}`
	req = httptest.NewRequest("PATCH", fmt.Sprintf("/api/sessions/%d/adventure", sessID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	sess, _ = s.db.GetSession(sessID)
	assert.Nil(t, sess.AdventureID)
}
