package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/digitalghost404/inkandbone/internal/db"
)

func (s *Server) handleOracleRoll(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Table     string `json:"table"`
		Roll      int    `json:"roll"`
		RulesetID *int64 `json:"ruleset_id"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.Table == "" || body.Roll < 1 || body.Roll > 50 {
		http.Error(w, "table and roll (1-50) required", http.StatusBadRequest)
		return
	}

	result, err := s.db.RollOracle(body.RulesetID, body.Table, body.Roll)
	if err != nil {
		serverError(w, r, err)
		return
	}

	s.bus.Publish(Event{Type: EventOracleRolled, Payload: map[string]any{
		"table": body.Table, "roll": body.Roll, "result": result,
	}})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
		"result": result,
		"table":  body.Table,
		"roll":   body.Roll,
	})
}

func (s *Server) handleGetTension(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	sessionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	level, err := s.db.GetTension(sessionID)
	if err != nil {
		serverError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"tension_level": level}) //nolint:errcheck
}

func (s *Server) handleCreateRelationship(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	campaignID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	var body struct {
		FromName         string `json:"from_name"`
		ToName           string `json:"to_name"`
		RelationshipType string `json:"relationship_type"`
		Description      string `json:"description"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.FromName == "" || body.ToName == "" {
		http.Error(w, "from_name and to_name required", http.StatusBadRequest)
		return
	}
	relType := body.RelationshipType
	if relType == "" {
		relType = "neutral"
	}

	id, err := s.db.CreateRelationship(campaignID, body.FromName, body.ToName, relType, body.Description)
	if err != nil {
		serverError(w, r, err)
		return
	}

	s.bus.Publish(Event{Type: EventRelationshipUpdated, Payload: map[string]any{"campaign_id": campaignID}})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id}) //nolint:errcheck
}

func (s *Server) handleListRelationships(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	campaignID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	rels, err := s.db.ListRelationships(campaignID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if rels == nil {
		rels = []db.Relationship{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rels) //nolint:errcheck
}

func (s *Server) handleUpdateRelationship(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body struct {
		RelationshipType string `json:"relationship_type"`
		Description      string `json:"description"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.RelationshipType == "" {
		body.RelationshipType = "neutral"
	}
	if err := s.db.UpdateRelationship(id, body.RelationshipType, body.Description); err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventRelationshipUpdated, Payload: map[string]any{"id": id}})
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDeleteRelationship(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.db.DeleteRelationship(id); err != nil {
		serverError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetMasqueradeIntegrity(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	sessionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	level, err := s.db.GetMasqueradeIntegrity(sessionID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"masquerade_integrity": level}) //nolint:errcheck
}

func (s *Server) handlePatchMasqueradeIntegrity(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	sessionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	var body struct {
		MasqueradeIntegrity *int `json:"masquerade_integrity"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.MasqueradeIntegrity == nil {
		http.Error(w, "masquerade_integrity required", http.StatusBadRequest)
		return
	}
	if err := s.db.UpdateMasqueradeIntegrity(sessionID, *body.MasqueradeIntegrity); err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventSessionUpdated, Payload: map[string]any{
		"session_id":           sessionID,
		"masquerade_integrity": *body.MasqueradeIntegrity,
	}})
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handlePatchTension(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	sessionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	var body struct {
		TensionLevel *int `json:"tension_level"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.TensionLevel == nil {
		http.Error(w, "tension_level required", http.StatusBadRequest)
		return
	}

	if err := s.db.UpdateTension(sessionID, *body.TensionLevel); err != nil {
		serverError(w, r, err)
		return
	}

	s.bus.Publish(Event{Type: EventTensionUpdated, Payload: map[string]any{
		"session_id": sessionID, "tension_level": body.TensionLevel,
	}})

	w.WriteHeader(http.StatusOK)
}
