package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	ruleset "github.com/digitalghost404/inkandbone/internal/ruleset"
	"github.com/digitalghost404/inkandbone/internal/ai"
)

func (s *Server) handleAdvanceCharacter(w http.ResponseWriter, r *http.Request) {
	charID, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid character id", http.StatusBadRequest)
		return
	}

	var body struct {
		Field    string `json:"field"`
		NewValue int    `json:"new_value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Field == "" {
		http.Error(w, "field and new_value required", http.StatusBadRequest)
		return
	}

	// Load character.
	char, err := s.db.GetCharacter(charID)
	if err != nil || char == nil {
		http.NotFound(w, r)
		return
	}

	// Load campaign -> ruleset.
	camp, err := s.db.GetCampaign(char.CampaignID)
	if err != nil || camp == nil {
		http.Error(w, "campaign not found", http.StatusInternalServerError)
		return
	}
	rs, err := s.db.GetRuleset(camp.RulesetID)
	if err != nil || rs == nil {
		http.Error(w, "ruleset not found", http.StatusInternalServerError)
		return
	}
	system := rs.Name

	// Parse current stats.
	var stats map[string]any
	if char.DataJSON != "" && char.DataJSON != "{}" {
		if err := json.Unmarshal([]byte(char.DataJSON), &stats); err != nil {
			http.Error(w, "invalid stats JSON", http.StatusInternalServerError)
			return
		}
	}
	if stats == nil {
		stats = map[string]any{}
	}

	xpKey := ruleset.XPKey(system)
	currentXP := 0
	if v, ok := stats[xpKey].(float64); ok {
		currentXP = int(v)
	}

	field := body.Field
	newVal := body.NewValue

	// ── Validate ────────────────────────────────────────────────────────────

	if strings.HasPrefix(field, "talent:") {
		if system != "wrath_glory" {
			http.Error(w, "talent advances only supported for wrath_glory", http.StatusBadRequest)
			return
		}
		talentName := strings.TrimPrefix(field, "talent:")

		ownedStr, _ := stats["talents"].(string)
		archetypeName, _ := stats["archetype"].(string)
		owned := isWGTalentOwned(ownedStr, archetypeName, talentName)

		if newVal <= 1 {
			// Initial purchase: must not already own it.
			if owned {
				http.Error(w, "talent already owned", http.StatusBadRequest)
				return
			}
		} else {
			// Upgrade: must already own it at rank newVal-1.
			if !owned {
				http.Error(w, "talent not yet owned; purchase at rank 1 first", http.StatusBadRequest)
				return
			}
			currentRank := wgTalentRank(stats, talentName)
			if newVal != currentRank+1 {
				http.Error(w, "new_value must be current rank + 1", http.StatusBadRequest)
				return
			}
		}

		cost := ruleset.WGTalentCost(talentName)
		if currentXP < cost {
			http.Error(w, "not enough XP", http.StatusBadRequest)
			return
		}

	} else if system == "dnd5e" && field == "level" {
		currentLevel := 1
		if v, ok := stats["level"].(float64); ok {
			currentLevel = int(v)
		}
		if newVal != currentLevel+1 {
			http.Error(w, "new_value must be current level + 1", http.StatusBadRequest)
			return
		}
		dnd5eThresholds := []int{
			0, 300, 900, 2700, 6500, 14000, 23000, 34000,
			48000, 64000, 85000, 100000, 120000, 140000,
			165000, 195000, 225000, 265000, 305000, 355000,
		}
		if currentLevel >= len(dnd5eThresholds)-1 {
			http.Error(w, "already max level", http.StatusBadRequest)
			return
		}
		if currentXP < dnd5eThresholds[currentLevel] {
			http.Error(w, "not enough XP to level up", http.StatusBadRequest)
			return
		}

	} else {
		validFields := ruleset.ValidFields(system)
		validSet := make(map[string]bool, len(validFields))
		for _, f := range validFields {
			validSet[f] = true
		}
		if !validSet[field] {
			http.Error(w, "field not advanceable for this system", http.StatusBadRequest)
			return
		}

		currentVal := 0
		if v, ok := stats[field].(float64); ok {
			currentVal = int(v)
		} else if s, ok := stats[field].(string); ok {
			if n, err := strconv.Atoi(s); err == nil {
				currentVal = n
			}
		}
		if newVal != currentVal+1 {
			http.Error(w, "new_value must be current value + 1", http.StatusBadRequest)
			return
		}
		if system == "blades" {
			cost := ruleset.XPCostFor(system, field, newVal, "")
			if currentXP < cost {
				http.Error(w, fmt.Sprintf("not enough XP (need %d)", cost), http.StatusBadRequest)
				return
			}
		} else if system != "dnd5e" {
			cost := ruleset.XPCostFor(system, field, newVal, char.DataJSON)
			if currentXP < cost {
				http.Error(w, "not enough XP", http.StatusBadRequest)
				return
			}
		}
	}

	// ── Apply ────────────────────────────────────────────────────────────────

	switch {
	case strings.HasPrefix(field, "talent:"):
		talentName := strings.TrimPrefix(field, "talent:")
		cost := ruleset.WGTalentCost(talentName)
		stats[xpKey] = float64(currentXP - cost)
		if newVal <= 1 {
			// Initial purchase: add to owned talents string.
			ownedStr, _ := stats["talents"].(string)
			if ownedStr == "" {
				stats["talents"] = talentName
			} else {
				stats["talents"] = ownedStr + "|" + talentName
			}
		}
		// Track rank in talent_ranks map.
		setWGTalentRank(stats, talentName, newVal)

	case system == "blades":
		stats[xpKey] = float64(0)
		currentVal := 0
		if v, ok := stats[field].(float64); ok {
			currentVal = int(v)
		}
		stats[field] = float64(currentVal + 1)

	case system == "dnd5e" && field == "level":
		currentLevel := 1
		if v, ok := stats["level"].(float64); ok {
			currentLevel = int(v)
		}
		newLevel := currentLevel + 1
		stats["level"] = float64(newLevel)
		currentHP := 0
		if v, ok := stats["hp"].(float64); ok {
			currentHP = int(v)
		}
		stats["hp"] = float64(currentHP + 5)
		stats["proficiency_bonus"] = float64(math.Floor(float64(newLevel-1)/4) + 2)

	default:
		cost := ruleset.XPCostFor(system, field, newVal, char.DataJSON)
		stats[xpKey] = float64(currentXP - cost)
		stats[field] = float64(newVal)

		if system == "wrath_glory" {
			ruleset.WGRecalcDerived(stats, field)
		}
	}

	// Persist.
	updated, err := json.Marshal(stats)
	if err != nil {
		http.Error(w, "marshal error", http.StatusInternalServerError)
		return
	}
	if err := s.db.UpdateCharacterData(charID, string(updated)); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	s.bus.Publish(Event{Type: EventCharacterUpdated, Payload: map[string]any{
		"id":           charID,
		"character_id": charID,
		"data_json":    string(updated),
	}})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"data_json": string(updated)}) //nolint:errcheck
}

// isWGTalentOwned returns true if talentName is in the pipe-delimited ownedStr.
func isWGTalentOwned(ownedStr, _ /*archetypeName*/ string, talentName string) bool {
	for _, t := range strings.Split(ownedStr, "|") {
		if strings.TrimSpace(t) == talentName {
			return true
		}
	}
	return false
}

// wgTalentRank returns the current rank of a talent from stats["talent_ranks"].
// Returns 1 if owned but no rank recorded (legacy), 0 if not owned.
func wgTalentRank(stats map[string]any, talentName string) int {
	ranks, _ := stats["talent_ranks"].(map[string]any)
	if ranks == nil {
		return 1 // owned but no rank map — treat as rank 1
	}
	if v, ok := ranks[talentName].(float64); ok {
		return int(v)
	}
	return 1
}

// setWGTalentRank persists the talent rank into stats["talent_ranks"].
func setWGTalentRank(stats map[string]any, talentName string, rank int) {
	ranks, _ := stats["talent_ranks"].(map[string]any)
	if ranks == nil {
		ranks = map[string]any{}
	}
	ranks[talentName] = float64(rank)
	stats["talent_ranks"] = ranks
}

// handleSuggestAdvances triggers the XP spend suggestion goroutine on demand
// (bypassing the per-session cap). The result arrives as a xp_spend_suggestions WS event.
// POST /api/characters/{id}/suggest-advances
func (s *Server) handleSuggestAdvances(w http.ResponseWriter, r *http.Request) {
	charID, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid character id", http.StatusBadRequest)
		return
	}

	char, err := s.db.GetCharacter(charID)
	if err != nil || char == nil {
		http.NotFound(w, r)
		return
	}

	camp, err := s.db.GetCampaign(char.CampaignID)
	if err != nil || camp == nil {
		http.Error(w, "campaign not found", http.StatusInternalServerError)
		return
	}

	rs, err := s.db.GetRuleset(camp.RulesetID)
	if err != nil || rs == nil {
		http.Error(w, "ruleset not found", http.StatusInternalServerError)
		return
	}

	var body struct {
		HintXP int `json:"hint_xp"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body) // ignore decode errors — hint is optional

	var stats map[string]any
	if char.DataJSON != "" && char.DataJSON != "{}" {
		if err := json.Unmarshal([]byte(char.DataJSON), &stats); err != nil {
			http.Error(w, "invalid stats JSON", http.StatusInternalServerError)
			return
		}
	}
	if stats == nil {
		stats = map[string]any{}
	}

	xpKey := ruleset.XPKey(rs.Name)
	currentXP := 0
	if v, ok := stats[xpKey].(float64); ok {
		currentXP = int(v)
	}
	// Use the frontend hint if it's larger (guards against DB corruption wiping XP).
	if body.HintXP > currentXP {
		currentXP = body.HintXP
		stats[xpKey] = float64(currentXP)
		// Persist the corrected XP so the advance endpoint reads the right value.
		if fixed, err := json.Marshal(stats); err == nil {
			_ = s.db.UpdateCharacterData(charID, string(fixed))
			char.DataJSON = string(fixed)
		}
	}

	// sessionID = 0 signals "manual trigger" — the goroutine skips the per-session cap.
	go s.autoSuggestXPSpend(0, charID, char, rs, stats, currentXP)

	w.WriteHeader(http.StatusAccepted)
}

// handleTalentDescription returns a 1-2 sentence description for any talent
	// or power name, generated by the AI when not in the static lookup.
// GET /api/talent-description?name=Unnatural+Awareness&system=wrath_glory
func (s *Server) handleTalentDescription(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	system := r.URL.Query().Get("system")
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if system == "" {
		system = "wrath_glory"
	}

	completer, ok := s.aiClient.(ai.Completer)
	if !ok {
		http.Error(w, "AI not available", http.StatusServiceUnavailable)
		return
	}

	systemLabel := map[string]string{
		"wrath_glory": "Wrath & Glory (Warhammer 40,000)",
		"dnd5e":       "Dungeons & Dragons 5th Edition",
		"blades":      "Blades in the Dark",
		"vtm":         "Vampire: The Masquerade",
		"cthulhu":     "Call of Cthulhu",
	}[system]
	if systemLabel == "" {
		systemLabel = system
	}

	prompt := fmt.Sprintf(
		`You are a rules expert for %s. Describe the talent or special ability "%s" in exactly 1–2 sentences. Focus on what it allows the character to do mechanically and narratively. Be concise and specific. Output only the description — no title, no bullet points, no extra text.`,
		systemLabel, name,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	desc, err := completer.Generate(ctx, prompt, 120)
	if err != nil {
		http.Error(w, "AI error", http.StatusInternalServerError)
		return
	}
	desc = strings.TrimSpace(desc)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"description": desc}) //nolint:errcheck
}
