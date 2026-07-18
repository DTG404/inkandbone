package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

// handleNextTurn advances the active turn index for a combat encounter (wraps around).
// POST /api/combat-encounters/{id}/next-turn
func (s *Server) handleNextTurn(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid encounter id", http.StatusBadRequest)
		return
	}
	nextIdx, roundNumber, err := s.db.AdvanceTurn(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, "not found", http.StatusNotFound)
			return
		}
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventTurnAdvanced, Payload: &TurnAdvancedPayload{EncounterID: RealtimeInt64(id), ActiveTurnIndex: RealtimeInt64(nextIdx), RoundNumber: RealtimeInt64(roundNumber)}})
	w.WriteHeader(http.StatusNoContent)
}

// handleListXP returns XP log entries for a session.
// GET /api/sessions/{id}/xp
func (s *Server) handleListXP(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	entries, err := s.db.ListXP(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if entries == nil {
		writeJSON(w, []struct{}{})
		return
	}
	writeJSON(w, entries)
}

// handleCreateXP adds an XP log entry to a session.
// POST /api/sessions/{id}/xp
// Body: {"note":"...","amount":100}  (amount is optional)
func (s *Server) handleCreateXP(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	var body struct {
		Note   string `json:"note"`
		Amount *int   `json:"amount"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.Note == "" {
		http.Error(w, "note is required", http.StatusBadRequest)
		return
	}
	entry, err := s.db.CreateXP(id, body.Note, body.Amount)
	if err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventXPAdded, Payload: &XPAddedPayload{SessionID: RealtimeInt64(id), ID: RealtimeInt64(entry.ID), Note: RealtimePtr(entry.Note), Amount: entry.Amount}})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(entry) //nolint:errcheck
}

// handleDeleteXP removes an XP log entry.
// DELETE /api/xp/{id}
func (s *Server) handleDeleteXP(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid xp id", http.StatusBadRequest)
		return
	}
	if err := s.db.DeleteXP(id); err != nil {
		serverError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
