package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/digitalghost404/inkandbone/internal/api"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) handleGenerateSessionRecap(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	if s.aiClient == nil {
		return mcplib.NewToolResultError("AI not configured — set DEEPSEEK_API_KEY, ANTHROPIC_API_KEY, or OLLAMA_MODEL"), nil
	}

	sessID, ok := optInt64(req, "session_id")
	if !ok {
		// Fall back to active session
		var err error
		sessID, err = s.activeSessionID()
		if err != nil {
			return mcplib.NewToolResultError("session_id required or set active session first"), nil
		}
	}

	msgs, err := s.db.ListMessages(sessID)
	if err != nil {
		return mcplib.NewToolResultError("list messages: " + err.Error()), nil
	}
	rolls, err := s.db.ListDiceRolls(sessID)
	if err != nil {
		return mcplib.NewToolResultError("list rolls: " + err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("Write a 2-3 sentence narrative recap of this TTRPG session.\n\nMessages:\n")
	for _, m := range msgs {
		fmt.Fprintf(&sb, "[%s]: %s\n", m.Role, m.Content)
	}
	sb.WriteString("\nDice rolls:\n")
	for _, r := range rolls {
		fmt.Fprintf(&sb, "%s = %d\n", r.Expression, r.Result)
	}

	if sb.Len() > 32000 {
		return mcplib.NewToolResultError("session transcript too long for recap — try end_session with a manual summary"), nil
	}

	summary, err := s.aiClient.Generate(ctx, sb.String(), 200)
	if err != nil {
		return mcplib.NewToolResultError("AI error: " + err.Error()), nil
	}

	if err := s.db.UpdateSessionSummary(sessID, summary); err != nil {
		return mcplib.NewToolResultError("update session: " + err.Error()), nil
	}
	s.bus.Publish(api.Event{Type: api.EventSessionUpdated, Payload: map[string]any{
		"session_id": sessID,
		"summary":    summary,
	}})
	return mcplib.NewToolResultText(fmt.Sprintf("session %d recap saved: %s", sessID, summary)), nil
}
