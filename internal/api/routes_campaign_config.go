package api

import (
	"net/http"
	"strings"
)

func (s *Server) handleGetCampaignConfig(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		respondError(w, "invalid campaign id", http.StatusBadRequest)
		return
	}

	camp, err := s.db.GetCampaign(id)
	if err != nil || camp == nil {
		respondError(w, "not found", http.StatusNotFound)
		return
	}

	// Fetch stats
	chars, _ := s.db.ListCharacters(id)
	sessions, _ := s.db.ListSessions(id)

	// Fetch ruleset name
	rulesetName := ""
	if rs, err := s.db.GetRuleset(camp.RulesetID); err == nil && rs != nil {
		rulesetName = rs.Name
	}

	respondJSON(w, map[string]any{
		"description":             camp.Description,
		"gm_notes":               camp.GmNotes,
		"system_prompt_override": camp.SystemPromptOverride,
		"character_count":        len(chars),
		"session_count":          len(sessions),
		"ruleset_name":           rulesetName,
	})
}

func (s *Server) handlePatchCampaignConfig(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		respondError(w, "invalid campaign id", http.StatusBadRequest)
		return
	}

	var body struct {
		Description          *string `json:"description"`
		GmNotes              *string `json:"gm_notes"`
		SystemPromptOverride *string `json:"system_prompt_override"`
	}
	if err := decodeJSON(r, &body); err != nil {
		respondError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if body.Description == nil && body.GmNotes == nil && body.SystemPromptOverride == nil {
		respondError(w, "no fields to update", http.StatusBadRequest)
		return
	}

	if err := s.db.UpdateCampaignConfig(id, body.Description, body.GmNotes, body.SystemPromptOverride); err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, err.Error(), http.StatusNotFound)
			return
		}
		respondError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.bus.Publish(Event{Type: EventCampaignConfigUpdated, Payload: map[string]any{
		"campaign_id": id,
	}})
	w.WriteHeader(http.StatusNoContent)
}
