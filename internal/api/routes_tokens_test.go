package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenAPI(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)

	mapID, err := s.db.CreateMap(campID, "Dungeon", "maps/a.svg")
	require.NoError(t, err)
	charID, err := s.db.CreateCharacter(campID, "Hero")
	require.NoError(t, err)

	// Place
	body := fmt.Sprintf(`{"entity_type":"character","entity_id":%d,"x":0.5,"y":0.5}`, charID)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/maps/%d/tokens", mapID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	var token map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&token))
	tokenID := int64(token["id"].(float64))

	// List
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/maps/%d/tokens", mapID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Move
	req = httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/map-tokens/%d", tokenID), strings.NewReader(`{"x":0.8,"y":0.2}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Remove
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/map-tokens/%d", tokenID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}
