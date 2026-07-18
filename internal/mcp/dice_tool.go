package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/digitalghost404/inkandbone/internal/api"
	"github.com/digitalghost404/inkandbone/internal/dice"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) handleRollDice(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	sessID, err := s.activeSessionID()
	if err != nil {
		return mcplib.NewToolResultError("no active session — start a session before rolling"), nil
	}
	expr, ok := reqStr(req, "expression")
	if !ok {
		return mcplib.NewToolResultError("expression is required"), nil
	}

	total, breakdown, err := dice.Roll(expr)
	if err != nil {
		return mcplib.NewToolResultError("invalid dice expression: " + err.Error()), nil
	}

	breakdownJSON, _ := json.Marshal(breakdown)
	if _, err := s.db.LogDiceRoll(sessID, expr, total, string(breakdownJSON)); err != nil {
		return mcplib.NewToolResultError("log roll: " + err.Error()), nil
	}

	// Format breakdown as [d1, d2, ...]
	parts := make([]string, len(breakdown))
	for i, d := range breakdown {
		parts[i] = fmt.Sprintf("%d", d)
	}
	summary := fmt.Sprintf("Rolled %s: **%d** [%s]", expr, total, strings.Join(parts, ", "))

	s.logNarrative(req, sessID)
	s.bus.Publish(api.Event{Type: api.EventDiceRolled, Payload: &api.DiceRolledPayload{SessionID: api.RealtimeInt64(sessID), Expression: api.RealtimePtr(expr), Result: api.RealtimeInt64(total), Breakdown: api.RealtimeArray(breakdown)}})
	return mcplib.NewToolResultText(summary), nil
}
