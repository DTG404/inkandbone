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

func TestNextTurn(t *testing.T) {
	s := newTestServer(t)
	campID, sessID := seedCampaign(t, s.db)
	_ = campID
	encID, err := s.db.CreateEncounter(sessID, "Forest Ambush")
	require.NoError(t, err)
	s.db.AddCombatant(encID, "Fighter", 16, 30, true, nil)
	s.db.AddCombatant(encID, "Goblin", 10, 8, false, nil)

	ch := s.bus.SubscribeContext(t.Context())

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/combat-encounters/%d/next-turn", encID), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	var got Event
	select {
	case got = <-ch:
	default:
		t.Fatal("expected turn_advanced event")
	}
	assert.Equal(t, EventTurnAdvanced, got.Type)
	payload := eventPayload(t, got)
	assert.EqualValues(t, encID, payload["encounter_id"])
	assert.EqualValues(t, 1, payload["active_turn_index"])
	assert.EqualValues(t, 1, payload["round_number"]) // first advance, no wrap — still round 1
}

func TestNextTurnNotFound(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/combat-encounters/99999/next-turn", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListCreateDeleteXP(t *testing.T) {
	s := newTestServer(t)
	_, sessID := seedCampaign(t, s.db)

	// List — empty
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/sessions/%d/xp", sessID), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var list []db.XPEntry
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	assert.Empty(t, list)

	// Create
	ch := s.bus.SubscribeContext(t.Context())
	body := `{"note":"Solved the riddle","amount":100}`
	req = httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/sessions/%d/xp", sessID),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var entry db.XPEntry
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &entry))
	assert.Equal(t, "Solved the riddle", entry.Note)
	require.NotNil(t, entry.Amount)
	assert.Equal(t, 100, *entry.Amount)

	var got Event
	select {
	case got = <-ch:
	default:
		t.Fatal("expected xp_added event")
	}
	assert.Equal(t, EventXPAdded, got.Type)

	// Delete
	req = httptest.NewRequest(http.MethodDelete,
		fmt.Sprintf("/api/xp/%d", entry.ID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}
