package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/digitalghost404/inkandbone/internal/api"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) activeCharacterID(req mcplib.CallToolRequest) (int64, error) {
	if id, ok := optInt64(req, "character_id"); ok {
		return id, nil
	}
	idStr, err := s.db.GetSetting("active_character_id")
	if err != nil || idStr == "" {
		return 0, fmt.Errorf("no active character — call set_active first")
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid active_character_id in settings")
	}
	return id, nil
}

// activeSessionID returns the active session ID from settings.
func (s *Server) activeSessionID() (int64, error) {
	idStr, err := s.db.GetSetting("active_session_id")
	if err != nil || idStr == "" {
		return 0, fmt.Errorf("no active session")
	}
	return strconv.ParseInt(idStr, 10, 64)
}

func (s *Server) handleGetCharacterSheet(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	id, err := s.activeCharacterID(req)
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}
	char, err := s.db.GetCharacter(id)
	if err != nil {
		return mcplib.NewToolResultError("db error: " + err.Error()), nil
	}
	if char == nil {
		return mcplib.NewToolResultError(fmt.Sprintf("character %d not found", id)), nil
	}
	b, err := json.Marshal(char)
	if err != nil {
		return mcplib.NewToolResultError("marshal error: " + err.Error()), nil
	}
	return mcplib.NewToolResultText(string(b)), nil
}

func (s *Server) handleUpdateCharacter(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	charID, err := s.activeCharacterID(req)
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}
	updatesStr, ok := reqStr(req, "updates")
	if !ok {
		return mcplib.NewToolResultError("updates is required"), nil
	}

	char, err := s.db.GetCharacter(charID)
	if err != nil {
		return mcplib.NewToolResultError("db error: " + err.Error()), nil
	}
	if char == nil {
		return mcplib.NewToolResultError(fmt.Sprintf("character %d not found", charID)), nil
	}

	// Merge updates into current data (top-level key merge)
	current := map[string]any{}
	if char.DataJSON != "" && char.DataJSON != "{}" {
		if err := json.Unmarshal([]byte(char.DataJSON), &current); err != nil {
			return mcplib.NewToolResultError("invalid existing character data JSON: " + err.Error()), nil
		}
	}
	updates := map[string]any{}
	if err := json.Unmarshal([]byte(updatesStr), &updates); err != nil {
		return mcplib.NewToolResultError("invalid updates JSON: " + err.Error()), nil
	}
	for k, v := range updates {
		current[k] = v
	}
	merged, err := json.Marshal(current)
	if err != nil {
		return mcplib.NewToolResultError("marshal error: " + err.Error()), nil
	}
	if err := s.db.UpdateCharacterData(charID, string(merged)); err != nil {
		return mcplib.NewToolResultError("update error: " + err.Error()), nil
	}

	sessID, _ := s.activeSessionID()
	s.logNarrative(req, sessID)
	s.bus.Publish(api.Event{Type: api.EventCharacterUpdated, Payload: &api.CharacterUpdatedPayload{ID: api.RealtimeInt64(charID), DataJson: api.RealtimePtr(string(merged))}})
	return mcplib.NewToolResultText(fmt.Sprintf("character %d updated", charID)), nil
}

func (s *Server) handleAddItem(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	charID, err := s.activeCharacterID(req)
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}
	itemName, ok := reqStr(req, "item_name")
	if !ok {
		return mcplib.NewToolResultError("item_name is required"), nil
	}

	char, err := s.db.GetCharacter(charID)
	if err != nil {
		return mcplib.NewToolResultError("db error: " + err.Error()), nil
	}
	if char == nil {
		return mcplib.NewToolResultError(fmt.Sprintf("character %d not found", charID)), nil
	}

	data := map[string]any{}
	if char.DataJSON != "" && char.DataJSON != "{}" {
		if err := json.Unmarshal([]byte(char.DataJSON), &data); err != nil {
			return mcplib.NewToolResultError("invalid character data JSON: " + err.Error()), nil
		}
	}
	inv := []any{}
	if existing, ok := data["inventory"].([]any); ok {
		inv = existing
	}
	inv = append(inv, itemName)
	data["inventory"] = inv

	merged, err := json.Marshal(data)
	if err != nil {
		return mcplib.NewToolResultError("marshal error: " + err.Error()), nil
	}
	if err := s.db.UpdateCharacterData(charID, string(merged)); err != nil {
		return mcplib.NewToolResultError("update error: " + err.Error()), nil
	}

	sessID, _ := s.activeSessionID()
	s.logNarrative(req, sessID)
	s.bus.Publish(api.Event{Type: api.EventCharacterUpdated, Payload: &api.CharacterUpdatedPayload{ID: api.RealtimeInt64(charID), DataJson: api.RealtimePtr(string(merged))}})
	return mcplib.NewToolResultText(fmt.Sprintf("added %q to inventory", itemName)), nil
}

func (s *Server) handleRemoveItem(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	charID, err := s.activeCharacterID(req)
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}
	itemName, ok := reqStr(req, "item_name")
	if !ok {
		return mcplib.NewToolResultError("item_name is required"), nil
	}

	char, err := s.db.GetCharacter(charID)
	if err != nil {
		return mcplib.NewToolResultError("db error: " + err.Error()), nil
	}
	if char == nil {
		return mcplib.NewToolResultError(fmt.Sprintf("character %d not found", charID)), nil
	}

	data := map[string]any{}
	if char.DataJSON != "" && char.DataJSON != "{}" {
		if err := json.Unmarshal([]byte(char.DataJSON), &data); err != nil {
			return mcplib.NewToolResultError("invalid character data JSON: " + err.Error()), nil
		}
	}
	existing, _ := data["inventory"].([]any)
	filtered := make([]any, 0, len(existing))
	removed := false
	for _, v := range existing {
		if !removed && v == itemName {
			removed = true
			continue
		}
		filtered = append(filtered, v)
	}
	if !removed {
		return mcplib.NewToolResultError(fmt.Sprintf("%q not found in inventory", itemName)), nil
	}
	data["inventory"] = filtered

	merged, err := json.Marshal(data)
	if err != nil {
		return mcplib.NewToolResultError("marshal error: " + err.Error()), nil
	}
	if err := s.db.UpdateCharacterData(charID, string(merged)); err != nil {
		return mcplib.NewToolResultError("update error: " + err.Error()), nil
	}

	sessID, _ := s.activeSessionID()
	s.logNarrative(req, sessID)
	s.bus.Publish(api.Event{Type: api.EventCharacterUpdated, Payload: &api.CharacterUpdatedPayload{ID: api.RealtimeInt64(charID), DataJson: api.RealtimePtr(string(merged))}})
	return mcplib.NewToolResultText(fmt.Sprintf("removed %q from inventory", itemName)), nil
}
