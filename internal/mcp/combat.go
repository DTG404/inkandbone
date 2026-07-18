package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalghost404/inkandbone/internal/api"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

type combatantInput struct {
	Name        string `json:"name"`
	Initiative  int    `json:"initiative"`
	HPMax       int    `json:"hp_max"`
	IsPlayer    bool   `json:"is_player"`
	CharacterID *int64 `json:"character_id"`
}

func (s *Server) handleStartCombat(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	sessID, err := s.activeSessionID()
	if err != nil {
		return mcplib.NewToolResultError("no active session — call start_session first"), nil
	}
	name, ok := reqStr(req, "name")
	if !ok {
		return mcplib.NewToolResultError("name is required"), nil
	}
	combatantsStr, ok := reqStr(req, "combatants")
	if !ok {
		return mcplib.NewToolResultError("combatants is required"), nil
	}
	var inputs []combatantInput
	if err := json.Unmarshal([]byte(combatantsStr), &inputs); err != nil {
		return mcplib.NewToolResultError("invalid combatants JSON: " + err.Error()), nil
	}

	encID, err := s.db.CreateEncounter(sessID, name)
	if err != nil {
		return mcplib.NewToolResultError("create encounter: " + err.Error()), nil
	}
	for _, c := range inputs {
		if _, err := s.db.AddCombatant(encID, c.Name, c.Initiative, c.HPMax, c.IsPlayer, c.CharacterID); err != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("add combatant %q: %v", c.Name, err)), nil
		}
	}

	s.logNarrative(req, sessID)
	s.bus.Publish(api.Event{Type: api.EventCombatStarted, Payload: map[string]any{"session_id": sessID, "encounter_id": encID, "name": name}})
	return mcplib.NewToolResultText(fmt.Sprintf("combat %q started (encounter %d, %d combatants)", name, encID, len(inputs))), nil
}

func (s *Server) handleUpdateCombatant(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	combID, ok := optInt64(req, "combatant_id")
	if !ok {
		return mcplib.NewToolResultError("combatant_id is required"), nil
	}

	hpRaw, hpOk := optInt64(req, "hp_current")
	if !hpOk {
		return mcplib.NewToolResultError("hp_current is required"), nil
	}
	hpCurrent := int(hpRaw)
	conditions := optStr(req, "conditions")
	if conditions == "" {
		conditions = "[]"
	}

	if err := s.db.UpdateCombatant(combID, hpCurrent, conditions); err != nil {
		return mcplib.NewToolResultError("update combatant: " + err.Error()), nil
	}

	sessID, _ := s.activeSessionID()
	s.logNarrative(req, sessID)
	s.bus.Publish(api.Event{Type: api.EventCombatantUpdated, Payload: map[string]any{"session_id": sessID, "combatant_id": combID}})
	return mcplib.NewToolResultText(fmt.Sprintf("combatant %d updated", combID)), nil
}

func (s *Server) handleEndCombat(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	sessID, err := s.activeSessionID()
	if err != nil {
		return mcplib.NewToolResultError("no active session"), nil
	}
	enc, err := s.db.GetActiveEncounter(sessID)
	if err != nil {
		return mcplib.NewToolResultError("db error: " + err.Error()), nil
	}
	if enc == nil {
		return mcplib.NewToolResultError("no active combat encounter"), nil
	}
	if err := s.db.EndEncounter(enc.ID); err != nil {
		return mcplib.NewToolResultError("end encounter: " + err.Error()), nil
	}

	s.logNarrative(req, sessID)
	s.bus.Publish(api.Event{Type: api.EventCombatEnded, Payload: map[string]any{"session_id": sessID, "encounter_id": enc.ID}})
	return mcplib.NewToolResultText(fmt.Sprintf("combat encounter %d ended", enc.ID)), nil
}
