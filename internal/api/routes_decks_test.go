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

func TestDeckAPI(t *testing.T) {
	s := newTestServer(t)
	campID, sessID := seedCampaign(t, s.db)

	// Create
	body := fmt.Sprintf(`{"name":"Oracle","cards":[{"front":"Moon","back":"Change"},{"front":"Sun"}]}`)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/campaigns/%d/decks", campID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	deckID := int64(created["id"].(float64))

	// Shuffle
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/decks/%d/shuffle", deckID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Draw
	drawBody := fmt.Sprintf(`{"session_id":%d}`, sessID)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/decks/%d/draw", deckID), strings.NewReader(drawBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var drawResp map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&drawResp))
	assert.NotNil(t, drawResp["card"])
	assert.Nil(t, drawResp["exhausted"])

	// Draw history
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/sessions/%d/deck-draws", sessID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Delete
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/decks/%d", deckID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}
