package mcp

import (
	"context"
	"fmt"
	"strconv"

	"github.com/digitalghost404/inkandbone/internal/api"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) handleSetActive(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	if id, ok := optInt64(req, "campaign_id"); ok && id > 0 {
		campaign, err := s.db.GetCampaign(id)
		if err != nil {
			return mcplib.NewToolResultError("get campaign: " + err.Error()), nil
		}
		if campaign == nil {
			return mcplib.NewToolResultError(fmt.Sprintf("campaign %d not found", id)), nil
		}
		if !campaign.Active {
			if err := s.db.ReopenCampaign(id); err != nil {
				return mcplib.NewToolResultError("reopen campaign: " + err.Error()), nil
			}
			if err := s.db.SetSetting("active_campaign_id", strconv.FormatInt(id, 10)); err != nil {
				return mcplib.NewToolResultError("set campaign: " + err.Error()), nil
			}
			s.bus.Publish(api.Event{Type: api.EventCampaignReopened, Payload: map[string]any{"campaign_id": id}})
		} else if err := s.db.SetSetting("active_campaign_id", strconv.FormatInt(id, 10)); err != nil {
			return mcplib.NewToolResultError("set campaign: " + err.Error()), nil
		}
	}
	if id, ok := optInt64(req, "session_id"); ok && id > 0 {
		if err := s.db.SetSetting("active_session_id", strconv.FormatInt(id, 10)); err != nil {
			return mcplib.NewToolResultError("set session: " + err.Error()), nil
		}
	}
	if id, ok := optInt64(req, "character_id"); ok && id > 0 {
		if err := s.db.SetSetting("active_character_id", strconv.FormatInt(id, 10)); err != nil {
			return mcplib.NewToolResultError("set character: " + err.Error()), nil
		}
	}
	return mcplib.NewToolResultText("active context updated"), nil
}

func (s *Server) handleStartSession(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	campIDStr, err := s.db.GetSetting("active_campaign_id")
	if err != nil || campIDStr == "" {
		return mcplib.NewToolResultError("no active campaign — call set_active first"), nil
	}
	campID, err := strconv.ParseInt(campIDStr, 10, 64)
	if err != nil {
		return mcplib.NewToolResultError("invalid active_campaign_id in settings"), nil
	}

	title, ok := reqStr(req, "title")
	if !ok {
		return mcplib.NewToolResultError("title is required"), nil
	}
	date, ok := reqStr(req, "date")
	if !ok {
		return mcplib.NewToolResultError("date is required"), nil
	}

	sessID, err := s.db.CreateSession(campID, title, date)
	if err != nil {
		return mcplib.NewToolResultError("create session: " + err.Error()), nil
	}
	if err := s.db.SetSetting("active_session_id", strconv.FormatInt(sessID, 10)); err != nil {
		return mcplib.NewToolResultError("set active session: " + err.Error()), nil
	}

	s.logNarrative(req, sessID)
	s.bus.Publish(api.Event{Type: api.EventSessionStarted, Payload: map[string]any{"session_id": sessID, "title": title}})
	return mcplib.NewToolResultText(fmt.Sprintf("session %d started: %s", sessID, title)), nil
}

func (s *Server) handleEndSession(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	sessIDStr, err := s.db.GetSetting("active_session_id")
	if err != nil || sessIDStr == "" {
		return mcplib.NewToolResultError("no active session"), nil
	}
	sessID, err := strconv.ParseInt(sessIDStr, 10, 64)
	if err != nil {
		return mcplib.NewToolResultError("invalid active_session_id in settings"), nil
	}

	summary, ok := reqStr(req, "summary")
	if !ok {
		return mcplib.NewToolResultError("summary is required"), nil
	}
	if err := s.db.UpdateSessionSummary(sessID, summary); err != nil {
		return mcplib.NewToolResultError("update summary: " + err.Error()), nil
	}
	_ = s.db.SetSetting("active_session_id", "")

	s.logNarrative(req, sessID)
	s.bus.Publish(api.Event{Type: api.EventSessionEnded, Payload: map[string]any{"session_id": sessID}})
	return mcplib.NewToolResultText(fmt.Sprintf("session %d ended", sessID)), nil
}

// logNarrative saves the optional "narrative" parameter as an assistant message.
// Silently skips if no narrative or no valid session ID is provided.
func (s *Server) logNarrative(req mcplib.CallToolRequest, sessionID int64) {
	if n := optStr(req, "narrative"); n != "" && sessionID != 0 {
		if _, err := s.db.CreateMessage(sessionID, "assistant", n, false, nil); err == nil {
			s.bus.Publish(api.Event{Type: api.EventMessageCreated, Payload: map[string]any{"session_id": sessionID, "content": n}})
		}
	}
}
