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

func TestHandoutRevealRoute(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)

	noteID, err := s.db.CreateWorldNote(campID, "The Vault", "Hidden place", "location")
	require.NoError(t, err)

	// reveal
	body := `{"is_revealed":true}`
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/world-notes/%d/reveal", noteID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// list with revealed=true returns it
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/campaigns/%d/world-notes?revealed=true", campID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var notes []db.WorldNote
	require.NoError(t, json.NewDecoder(w.Body).Decode(&notes))
	require.Len(t, notes, 1)
	assert.True(t, notes[0].IsRevealed)
}
