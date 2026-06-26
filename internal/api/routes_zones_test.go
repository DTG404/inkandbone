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

func TestZoneAPI(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)
	mapID, err := s.db.CreateMap(campID, "Dungeon", "maps/a.svg")
	require.NoError(t, err)

	// Create zone
	body := `{"name":"throne room","x":0.1,"y":0.2,"width":0.3,"height":0.25}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/maps/%d/zones", mapID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	zoneID := int64(created["id"].(float64))

	// List
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/maps/%d/zones", mapID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var zones []db.MapZone
	require.NoError(t, json.NewDecoder(w.Body).Decode(&zones))
	require.Len(t, zones, 1)
	assert.False(t, zones[0].IsRevealed)

	// Reveal via PATCH
	req = httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/map-zones/%d", zoneID), strings.NewReader(`{"is_revealed":true}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Delete
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/map-zones/%d", zoneID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}
