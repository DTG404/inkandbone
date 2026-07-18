package api

import (
	"encoding/json"
	mathrand "math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/digitalghost404/inkandbone/internal/db"
)

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func parsePathID(r *http.Request, key string) (int64, bool) {
	s := r.PathValue(key)
	id, err := strconv.ParseInt(s, 10, 64)
	return id, err == nil && id > 0
}

func (s *Server) handleListCampaigns(w http.ResponseWriter, r *http.Request) {
	campaigns, err := s.db.ListCampaigns()
	if err != nil {
		serverError(w, r, err)
		return
	}
	if campaigns == nil {
		campaigns = []db.Campaign{}
	}
	writeJSON(w, campaigns)
}

func (s *Server) handleListCharacters(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	characters, err := s.db.ListCharacters(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if characters == nil {
		characters = []db.Character{}
	}
	writeJSON(w, characters)
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	sessions, err := s.db.ListSessions(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if sessions == nil {
		sessions = []db.Session{}
	}
	writeJSON(w, sessions)
}

func (s *Server) handleListDiceRolls(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	rolls, err := s.db.ListDiceRolls(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if rolls == nil {
		rolls = []db.DiceRoll{}
	}
	writeJSON(w, rolls)
}

type contextCombatSnapshot struct {
	Encounter  *db.CombatEncounter `json:"encounter"`
	Combatants []db.Combatant      `json:"combatants"`
}

type contextResponse struct {
	Campaign       *db.Campaign           `json:"campaign"`
	Character      *db.Character          `json:"character"`
	Session        *db.Session            `json:"session"`
	RecentMessages []db.Message           `json:"recent_messages"`
	ActiveCombat   *contextCombatSnapshot `json:"active_combat"`
}

func (s *Server) handlePatchCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		respondError(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	var body struct {
		Active         *bool `json:"active"`
		ChronicleNight *int  `json:"chronicle_night"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.Active == nil && body.ChronicleNight == nil {
		respondError(w, "active or chronicle_night is required", http.StatusBadRequest)
		return
	}
	if body.ChronicleNight != nil {
		if err := s.db.UpdateCampaignChronicleNight(id, *body.ChronicleNight); err != nil {
			if strings.Contains(err.Error(), "not found") {
				respondError(w, "not found", http.StatusNotFound)
				return
			}
			serverError(w, r, err)
			return
		}
		campaign, err := s.db.GetCampaign(id)
		if err != nil || campaign == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		s.bus.Publish(Event{Type: EventCampaignUpdated, Payload: &CampaignUpdatedPayload{CampaignID: RealtimeInt64(id), ChronicleNight: RealtimeInt64(*body.ChronicleNight)}})
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var err error
	if *body.Active {
		err = s.db.ReopenCampaign(id)
	} else {
		err = s.db.CloseCampaign(id)
		if err == nil {
			// Clear active context settings so the UI resets to blank.
			for _, key := range []string{"active_campaign_id", "active_character_id", "active_session_id"} {
				_ = s.db.SetSetting(key, "")
			}
		}
	}
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, "not found", http.StatusNotFound)
			return
		}
		serverError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePatchSession(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	var body struct {
		Summary   *string `json:"summary"`
		Notes     *string `json:"notes"`
		SceneTags *string `json:"scene_tags"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	payload := &SessionUpdatedPayload{SessionID: RealtimeInt64(id)}
	if body.Summary != nil {
		if err := s.db.UpdateSessionSummary(id, *body.Summary); err != nil {
			if strings.Contains(err.Error(), "not found") {
				respondError(w, "not found", http.StatusNotFound)
				return
			}
			serverError(w, r, err)
			return
		}
		payload.Summary = body.Summary
	}
	if body.Notes != nil {
		if err := s.db.UpdateSessionNotes(id, *body.Notes); err != nil {
			if strings.Contains(err.Error(), "not found") {
				respondError(w, "not found", http.StatusNotFound)
				return
			}
			serverError(w, r, err)
			return
		}
		payload.Notes = body.Notes
	}
	if body.SceneTags != nil {
		if err := s.db.UpdateSceneTags(id, *body.SceneTags); err != nil {
			serverError(w, r, err)
			return
		}
		payload.SceneTags = body.SceneTags
	}
	s.bus.Publish(Event{Type: EventSessionUpdated, Payload: payload})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGenerateRecap(w http.ResponseWriter, r *http.Request) {
	if s.aiClient == nil {
		http.Error(w, "AI not configured — set ANTHROPIC_API_KEY", http.StatusServiceUnavailable)
		return
	}
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	summary, err := s.buildRecap(r.Context(), id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if err := s.db.UpdateSessionSummary(id, summary); err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, "not found", http.StatusNotFound)
			return
		}
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventSessionUpdated, Payload: &SessionUpdatedPayload{SessionID: RealtimeInt64(id), Summary: RealtimePtr(summary)}})
	writeJSON(w, map[string]string{"summary": summary})
}

func (s *Server) handleGetTimeline(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	entries, err := s.db.GetSessionTimeline(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if entries == nil {
		entries = []db.TimelineEntry{}
	}
	writeJSON(w, entries)
}

func (s *Server) handleGetContext(w http.ResponseWriter, _ *http.Request) {
	resp := contextResponse{RecentMessages: []db.Message{}}

	if campIDStr, err := s.db.GetSetting("active_campaign_id"); err == nil && campIDStr != "" {
		if campID, err := strconv.ParseInt(campIDStr, 10, 64); err == nil {
			resp.Campaign, _ = s.db.GetCampaign(campID)
		}
	}
	if charIDStr, err := s.db.GetSetting("active_character_id"); err == nil && charIDStr != "" {
		if charID, err := strconv.ParseInt(charIDStr, 10, 64); err == nil {
			resp.Character, _ = s.db.GetCharacter(charID)
		}
	}
	if sessIDStr, err := s.db.GetSetting("active_session_id"); err == nil && sessIDStr != "" {
		if sessID, err := strconv.ParseInt(sessIDStr, 10, 64); err == nil {
			resp.Session, _ = s.db.GetSession(sessID)
			if msgs, err := s.db.ListAIVisibleMessages(sessID); err == nil {
				if len(msgs) > 20 {
					msgs = msgs[len(msgs)-20:]
				}
				resp.RecentMessages = msgs
			}
			if enc, err := s.db.GetActiveEncounter(sessID); err == nil && enc != nil {
				cs := &contextCombatSnapshot{Encounter: enc, Combatants: []db.Combatant{}}
				if combatants, err := s.db.ListCombatants(enc.ID); err == nil {
					cs.Combatants = combatants
				}
				resp.ActiveCombat = cs
			}
		}
	}

	writeJSON(w, resp)
}

// --- Feature 1: Streaming GM Text (SSE) ---

// handleTyping broadcasts a typing indicator from a player agent (e.g. Nyx).
// POST /api/sessions/{id}/typing  {"character_id":5,"status":"thinking"}
func (s *Server) handleTyping(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	var body struct {
		CharacterID int64  `json:"character_id"`
		Status      string `json:"status"` // "thinking" or "done"
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.CharacterID == 0 {
		http.Error(w, "character_id required", http.StatusBadRequest)
		return
	}
	if body.Status == "" {
		body.Status = "thinking"
	}
	s.bus.Publish(Event{Type: EventTyping, Payload: &TypingPayload{SessionID: RealtimeInt64(sessionID), CharacterID: RealtimeInt64(body.CharacterID), Status: RealtimePtr(body.Status)}})
	w.WriteHeader(http.StatusNoContent)
}

// autoSuggestXPSpend fires when XP increases after a GM response.
// It calls the AI to generate 2–3 ranked advancement suggestions and pushes
// them to the frontend as an xp_spend_suggestions WebSocket event.
// A per-session cap of 20 suggestions is enforced to avoid spam.

// --- Feature 4: In-Browser Dice Roller ---

func (s *Server) handleRollDice(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	var body struct {
		Expression    string `json:"expression"`
		CharacterName string `json:"character_name"`
		Hidden        bool   `json:"hidden"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.Expression == "" {
		http.Error(w, "expression is required", http.StatusBadRequest)
		return
	}

	expr := strings.ToLower(strings.TrimSpace(body.Expression))
	count := 1
	sides := 0
	modifier := 0
	if idx := strings.Index(expr, "d"); idx >= 0 {
		if idx > 0 {
			n, err := strconv.Atoi(expr[:idx])
			if err != nil || n < 1 {
				http.Error(w, "invalid dice count", http.StatusBadRequest)
				return
			}
			count = n
		}
		rest := expr[idx+1:]
		// Parse optional +M or -M modifier after the die sides.
		sideStr := rest
		if plus := strings.IndexAny(rest, "+-"); plus >= 0 {
			m, err := strconv.Atoi(rest[plus:])
			if err != nil {
				http.Error(w, "invalid modifier", http.StatusBadRequest)
				return
			}
			modifier = m
			sideStr = rest[:plus]
		}
		s2, err := strconv.Atoi(sideStr)
		if err != nil || s2 < 1 {
			http.Error(w, "invalid die sides", http.StatusBadRequest)
			return
		}
		sides = s2
	} else {
		http.Error(w, "expression must be dX or NdX format", http.StatusBadRequest)
		return
	}

	rolls := make([]int, count)
	total := modifier
	for i := range rolls {
		roll := mathrand.Intn(sides) + 1
		rolls[i] = roll
		total += roll
	}

	breakdownBytes, _ := json.Marshal(rolls)
	_, err := s.db.LogDiceRoll(id, body.Expression, total, string(breakdownBytes))
	if err != nil {
		serverError(w, r, err)
		return
	}

	s.bus.Publish(Event{Type: EventDiceRolled, Payload: &DiceRolledPayload{SessionID: RealtimeInt64(id), Expression: RealtimePtr(body.Expression), Result: RealtimeInt64(total), CharacterName: RealtimePtr(body.CharacterName), Hidden: RealtimePtr(body.Hidden)}})

	writeJSON(w, map[string]any{
		"expression": body.Expression,
		"result":     total,
		"rolls":      rolls,
	})
}

// --- Feature 5: Condition Badges (PATCH combatant) ---

func (s *Server) handlePatchCombatant(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid combatant id", http.StatusBadRequest)
		return
	}
	var body struct {
		HPCurrent      *int   `json:"hp_current"`
		ConditionsJSON string `json:"conditions_json"`
		Initiative     *int   `json:"initiative"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}

	var currentHP int
	var currentConditions string
	err := s.db.SQL().QueryRowContext(r.Context(),
		"SELECT hp_current, conditions_json FROM combatants WHERE id = ?", id,
	).Scan(&currentHP, &currentConditions)
	if err != nil {
		http.Error(w, "combatant not found", http.StatusNotFound)
		return
	}

	hp := currentHP
	if body.HPCurrent != nil {
		hp = *body.HPCurrent
	}
	conditions := currentConditions
	if body.ConditionsJSON != "" {
		conditions = body.ConditionsJSON
	}

	if err := s.db.UpdateCombatant(id, hp, conditions); err != nil {
		serverError(w, r, err)
		return
	}
	if body.Initiative != nil {
		if err := s.db.PatchCombatantInitiative(id, *body.Initiative); err != nil {
			serverError(w, r, err)
			return
		}
	}
	s.bus.Publish(Event{Type: EventCombatantUpdated, Payload: &CombatantUpdatedPayload{CombatantID: RealtimeInt64(id)}})
	w.WriteHeader(http.StatusNoContent)
}

// --- Feature 8: Map Pins from Chat ---

// --- Feature 9: NPC Roster ---

// --- Feature 10: Objectives Tracker ---

// --- Feature 11: Player Inventory ---

func (s *Server) handleReorderCombatants(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid encounter id", http.StatusBadRequest)
		return
	}
	var body struct {
		IDs []int64 `json:"ids"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if len(body.IDs) == 0 {
		http.Error(w, "ids required", http.StatusBadRequest)
		return
	}
	if err := s.db.ReorderCombatants(id, body.IDs); err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventCombatantUpdated, Payload: &CombatantUpdatedPayload{EncounterID: RealtimeInt64(id)}})
	w.WriteHeader(http.StatusNoContent)
}
