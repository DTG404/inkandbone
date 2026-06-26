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
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.bus.Publish(Event{Type: EventTurnAdvanced, Payload: map[string]any{
		"encounter_id":      id,
		"active_turn_index": nextIdx,
		"round_number":      roundNumber,
	}})
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
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Note == "" {
		http.Error(w, "note is required", http.StatusBadRequest)
		return
	}
	entry, err := s.db.CreateXP(id, body.Note, body.Amount)
	if err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.bus.Publish(Event{Type: EventXPAdded, Payload: map[string]any{
		"session_id": id,
		"id":         entry.ID,
		"note":       entry.Note,
		"amount":     entry.Amount,
	}})
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
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
