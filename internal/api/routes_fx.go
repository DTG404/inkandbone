package api

import "net/http"

// handleTriggerMapFX publishes a map_fx event without persisting to DB.
// POST /api/maps/{id}/fx
// Body: {"effect":"fire","x":0.3,"y":0.5,"duration_ms":1500}
func (s *Server) handleTriggerMapFX(w http.ResponseWriter, r *http.Request) {
	mapID, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid map id", http.StatusBadRequest)
		return
	}
	var body struct {
		Effect     string  `json:"effect"`
		X          float64 `json:"x"`
		Y          float64 `json:"y"`
		DurationMS int     `json:"duration_ms"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.Effect == "" {
		http.Error(w, "effect required", http.StatusBadRequest)
		return
	}
	if body.X < 0 || body.X > 1 || body.Y < 0 || body.Y > 1 {
		http.Error(w, "x/y must be 0–1", http.StatusBadRequest)
		return
	}
	if body.DurationMS <= 0 {
		body.DurationMS = 1000
	}
	s.bus.Publish(Event{Type: EventMapFX, Payload: map[string]any{
		"map_id":      mapID,
		"effect":      body.Effect,
		"x":           body.X,
		"y":           body.Y,
		"duration_ms": body.DurationMS,
	}})
	w.WriteHeader(http.StatusNoContent)
}
