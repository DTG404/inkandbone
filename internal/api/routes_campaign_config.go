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
		"description":            camp.Description,
		"gm_notes":               camp.GmNotes,
		"system_prompt_override": camp.SystemPromptOverride,
		"content_boundaries":     camp.ContentBoundaries,
		"narrative_locale":       camp.NarrativeLocale,
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
		ContentBoundaries    *string `json:"content_boundaries"`
		NarrativeLocale      *string `json:"narrative_locale"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}

	if body.Description == nil && body.GmNotes == nil && body.SystemPromptOverride == nil && body.ContentBoundaries == nil && body.NarrativeLocale == nil {
		respondError(w, "no fields to update", http.StatusBadRequest)
		return
	}
	if body.SystemPromptOverride != nil && len(*body.SystemPromptOverride) > maxNarrativePreferenceBytes {
		respondError(w, "campaign narration guidance exceeds 8 KiB", http.StatusBadRequest)
		return
	}
	if body.ContentBoundaries != nil && len(*body.ContentBoundaries) > maxNarrativePreferenceBytes {
		respondError(w, "content boundaries exceed 8 KiB", http.StatusBadRequest)
		return
	}
	if body.NarrativeLocale != nil && !validNarrativeLocale(*body.NarrativeLocale) {
		respondError(w, "invalid narrative locale", http.StatusBadRequest)
		return
	}

	if err := s.db.UpdateCampaignConfig(id, body.Description, body.GmNotes, body.SystemPromptOverride, body.ContentBoundaries, body.NarrativeLocale); err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, "not found", http.StatusNotFound)
			return
		}
		serverError(w, r, err)
		return
	}

	s.bus.Publish(Event{Type: EventCampaignConfigUpdated, Payload: map[string]any{
		"campaign_id": id,
	}})
	w.WriteHeader(http.StatusNoContent)
}
