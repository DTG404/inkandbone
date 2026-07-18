package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggerMapFX(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)

	// Create a map
	mapID, err := s.db.CreateMap(campID, "Dungeon", "dungeon.png")
	require.NoError(t, err)

	ch := s.bus.SubscribeContext(t.Context())

	body := `{"effect":"fire","x":0.3,"y":0.5,"duration_ms":1500}`
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/maps/%d/fx", mapID),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	var got Event
	select {
	case got = <-ch:
	default:
		t.Fatal("expected map_fx event")
	}
	assert.Equal(t, EventMapFX, got.Type)
	payload := got.Payload.(map[string]any)
	assert.Equal(t, mapID, payload["map_id"])
	assert.Equal(t, "fire", payload["effect"])
}
