package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/digitalghost404/inkandbone/internal/ruleset"
)

// handleListRulesets returns all rulesets.
func (s *Server) handleListRulesets(w http.ResponseWriter, r *http.Request) {
	rulesets, err := s.db.ListRulesets()
	if err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, rulesets)
}

// handleCreateCampaign creates a campaign.
// Body: {"name":"…","description":"…","ruleset_id":1}
func (s *Server) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		RulesetID   int64  `json:"ruleset_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		respondError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		respondError(w, "name is required", http.StatusBadRequest)
		return
	}
	if body.RulesetID == 0 {
		respondError(w, "ruleset_id is required", http.StatusBadRequest)
		return
	}
	id, err := s.db.CreateCampaign(body.RulesetID, body.Name, body.Description)
	if err != nil {
		respondError(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.bus.Publish(Event{Type: EventCampaignCreated, Payload: map[string]any{"campaign_id": id, "name": body.Name}})
	w.WriteHeader(http.StatusCreated)
	respondJSON(w, map[string]any{"id": id})
}

// handleDeleteCampaign deletes a campaign and all its data.
func (s *Server) handleDeleteCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	// Clear active_campaign_id only if it points to this campaign.
	if campIDStr, _ := s.db.GetSetting("active_campaign_id"); campIDStr == strconv.FormatInt(id, 10) {
		_ = s.db.SetSetting("active_campaign_id", "")
	}
	// Clear active_session_id only if the session belongs to this campaign.
	if sessIDStr, _ := s.db.GetSetting("active_session_id"); sessIDStr != "" {
		if sessID, err := strconv.ParseInt(sessIDStr, 10, 64); err == nil && sessID > 0 {
			sess, err := s.db.GetSession(sessID)
			if err == nil && (sess == nil || sess.CampaignID == id) {
				_ = s.db.SetSetting("active_session_id", "")
			}
		}
	}
	// Clear active_character_id only if the character belongs to this campaign.
	if charIDStr, _ := s.db.GetSetting("active_character_id"); charIDStr != "" {
		if charID, err := strconv.ParseInt(charIDStr, 10, 64); err == nil && charID > 0 {
			char, err := s.db.GetCharacter(charID)
			if err == nil && (char == nil || char.CampaignID == id) {
				_ = s.db.SetSetting("active_character_id", "")
			}
		}
	}
	if err := s.db.DeleteCampaign(id); err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.bus.Publish(Event{Type: EventCampaignDeleted, Payload: map[string]any{"campaign_id": id}})
	w.WriteHeader(http.StatusNoContent)
}

// handleGetCharacterOptions returns the chooseable field options for a ruleset.
// GET /api/rulesets/{id}/character-options
func (s *Server) handleGetCharacterOptions(w http.ResponseWriter, r *http.Request) {
	rulesetID, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid ruleset id", http.StatusBadRequest)
		return
	}
	rs, err := s.db.GetRuleset(rulesetID)
	if err != nil || rs == nil {
		http.Error(w, "ruleset not found", http.StatusNotFound)
		return
	}
	opts := ruleset.CharacterOptions(rs.Name)
	if opts == nil {
		opts = map[string][]string{}
	}
	writeJSON(w, opts)
}

// handleCreateCharacter creates a character in a campaign.
// Body: {"name":"…","overrides":{"archetype":"Astra Militarum","faction":""}}
// Fields in overrides with non-empty values replace the random roll for that field.
func (s *Server) handleCreateCharacter(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	var body struct {
		Name      string            `json:"name"`
		Overrides map[string]string `json:"overrides"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	id, err := s.db.CreateCharacter(campaignID, body.Name)
	if err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Auto-roll stats based on the campaign's ruleset system.
	camp, err := s.db.GetCampaign(campaignID)
	if err == nil && camp != nil {
		rs, err := s.db.GetRuleset(camp.RulesetID)
		if err == nil && rs != nil {
			stats := ruleset.RollStats(rs.Name, body.Overrides["archetype"])
			// If RollStats doesn't know this system, try schema_json defaults.
			if len(stats) == 0 && rs.SchemaJSON != "" {
				stats = ruleset.RollStatsFromSchema(rs.SchemaJSON)
			}
			if len(stats) > 0 {
				// Apply user-selected overrides (non-empty values only).
				for k, v := range body.Overrides {
					if v != "" {
						stats[k] = v
					}
				}
				if dataJSON, err := json.Marshal(stats); err == nil {
					_ = s.db.UpdateCharacterData(id, string(dataJSON))
				}
			}
		}
	}

	char, err := s.db.GetCharacter(id)
	if err != nil || char == nil {
		http.Error(w, "db: could not retrieve character", http.StatusInternalServerError)
		return
	}
	s.bus.Publish(Event{Type: EventCharacterCreated, Payload: map[string]any{"character_id": id, "name": body.Name}})
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, char)
}

// handleDeleteCharacter deletes a character and its items.
func (s *Server) handleDeleteCharacter(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid character id", http.StatusBadRequest)
		return
	}
	// Clear settings if this character is active
	activeChar, _ := s.db.GetSetting("active_character_id")
	if activeChar == strconv.FormatInt(id, 10) {
		_ = s.db.SetSetting("active_character_id", "")
	}
	if err := s.db.DeleteCharacter(id); err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleCreateSession creates a session in a campaign.
// Body: {"title":"…","date":"2025-01-01"}
func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	var body struct {
		Title string `json:"title"`
		Date  string `json:"date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}
	if body.Date == "" {
		body.Date = time.Now().Format("2006-01-02")
	}
	id, err := s.db.CreateSession(campaignID, body.Title, body.Date)
	if err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	sess, err := s.db.GetSession(id)
	if err != nil || sess == nil {
		http.Error(w, "db: could not retrieve session", http.StatusInternalServerError)
		return
	}
	s.bus.Publish(Event{Type: EventSessionStarted, Payload: map[string]any{"session_id": id, "title": body.Title}})
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, sess)
}

// handleDeleteSession deletes a session and all its messages, dice rolls, combat data.
func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	// Clear settings if this session is active
	activeSess, _ := s.db.GetSetting("active_session_id")
	if activeSess == strconv.FormatInt(id, 10) {
		_ = s.db.SetSetting("active_session_id", "")
	}
	if err := s.db.DeleteSession(id); err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.bus.Publish(Event{Type: EventSessionDeleted, Payload: map[string]any{"session_id": id}})
	w.WriteHeader(http.StatusNoContent)
}

// handlePatchSettings sets one or more active-context settings.
// Body: {"campaign_id":1,"character_id":2,"session_id":3}  (any subset)
func (s *Server) handlePatchSettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CampaignID  *int64 `json:"campaign_id"`
		CharacterID *int64 `json:"character_id"`
		SessionID   *int64 `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.CampaignID != nil {
		val := ""
		if *body.CampaignID != 0 {
			val = strconv.FormatInt(*body.CampaignID, 10)
			// Auto-reopen closed campaigns, mirroring MCP set_active behaviour.
			campaign, err := s.db.GetCampaign(*body.CampaignID)
			if err != nil {
				http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
				return
			}
			if campaign != nil && !campaign.Active {
				if err := s.db.ReopenCampaign(*body.CampaignID); err != nil {
					http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
					return
				}
				s.bus.Publish(Event{Type: EventCampaignReopened, Payload: map[string]any{"campaign_id": *body.CampaignID}})
			}
		}
		if err := s.db.SetSetting("active_campaign_id", val); err != nil {
			http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if body.CharacterID != nil {
		val := ""
		if *body.CharacterID != 0 {
			val = strconv.FormatInt(*body.CharacterID, 10)
		}
		if err := s.db.SetSetting("active_character_id", val); err != nil {
			http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if body.SessionID != nil {
		val := ""
		if *body.SessionID != 0 {
			val = strconv.FormatInt(*body.SessionID, 10)
		}
		if err := s.db.SetSetting("active_session_id", val); err != nil {
			http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	ctxPayload := map[string]any{}
	if body.CampaignID != nil {
		ctxPayload["campaign_id"] = *body.CampaignID
	}
	if body.CharacterID != nil {
		ctxPayload["character_id"] = *body.CharacterID
	}
	if body.SessionID != nil {
		ctxPayload["session_id"] = *body.SessionID
	}
	s.bus.Publish(Event{Type: EventContextUpdated, Payload: ctxPayload})
	w.WriteHeader(http.StatusNoContent)
}

