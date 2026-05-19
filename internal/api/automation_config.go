package api

import (
	"net/http"
)

// Automation config keys -- stored in the settings table.
const (
	settingAutoExtractNPCs     = "auto_extract_npcs"
	settingAutoGenerateMap     = "auto_generate_map"
	settingAutoUpdateStats     = "auto_update_stats"
	settingAutoUpdateRecap     = "auto_update_recap"
	settingAutoDetectObj       = "auto_detect_objectives"
	settingAutoExtractItems    = "auto_extract_items"
	settingAutoCheckRoll       = "auto_check_roll"
	settingAutoUpdateTension   = "auto_update_tension"
	settingAutoUpdateCurrency  = "auto_update_currency"
	settingAutoUpdateSceneTags = "auto_update_scene_tags"
	settingAutoUpdateMasq      = "auto_update_masquerade"
	settingAutoUpdateNight     = "auto_update_chronicle_night"
	settingAutoSuggestXP       = "auto_suggest_xp"
)

// AutomationSetting holds a single automation toggle definition.
type AutomationSetting struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Default bool   `json:"default"`
}

// AllAutomationSettings returns all automation config keys with their defaults.
func AllAutomationSettings() []AutomationSetting {
	return []AutomationSetting{
		{settingAutoExtractNPCs, "Extract NPCs", true},
		{settingAutoGenerateMap, "Generate Maps", true},
		{settingAutoUpdateStats, "Update Character Stats", true},
		{settingAutoUpdateRecap, "Regenerate Session Recap", true},
		{settingAutoDetectObj, "Detect Objectives", true},
		{settingAutoExtractItems, "Extract Items", true},
		{settingAutoCheckRoll, "Enforce Dice Rolls", true},
		{settingAutoUpdateTension, "Update Tension", true},
		{settingAutoUpdateCurrency, "Update Currency", true},
		{settingAutoUpdateSceneTags, "Update Scene Tags", true},
		{settingAutoUpdateMasq, "Update Masquerade (VtM)", true},
		{settingAutoUpdateNight, "Update Chronicle Night (VtM)", true},
		{settingAutoSuggestXP, "Suggest XP Spending", true},
	}
}

// isAutomationEnabled checks if an automation goroutine is enabled via settings.
// Defaults to enabled if no setting is stored.
func (s *Server) isAutomationEnabled(settingKey string) bool {
	val, err := s.db.GetSetting(settingKey)
	if err != nil || val == "" {
		return true // default: enabled
	}
	return val == "1" || val == "true"
}

// handleListAutomationSettings returns all automation settings with their current values.
// GET /api/settings/automations
func (s *Server) handleListAutomationSettings(w http.ResponseWriter, r *http.Request) {
	settings := AllAutomationSettings()
	result := make([]map[string]any, len(settings))
	for i, setting := range settings {
		result[i] = map[string]any{
			"key":     setting.Key,
			"label":   setting.Label,
			"enabled": s.isAutomationEnabled(setting.Key),
		}
	}
	respondJSON(w, result)
}

// handlePatchAutomationSetting toggles a single automation setting.
// Body: {"key":"auto_extract_npcs","enabled":false}
func (s *Server) handlePatchAutomationSetting(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key     string `json:"key"`
		Enabled bool   `json:"enabled"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Key == "" {
		respondError(w, "key and enabled are required", http.StatusBadRequest)
		return
	}
	val := "0"
	if body.Enabled {
		val = "1"
	}
	if err := s.db.SetSetting(body.Key, val); err != nil {
		respondError(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
