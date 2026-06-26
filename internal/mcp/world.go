package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/digitalghost404/inkandbone/internal/api"
	"github.com/digitalghost404/inkandbone/internal/db"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) activeCampaignID() (int64, error) {
	idStr, err := s.db.GetSetting("active_campaign_id")
	if err != nil || idStr == "" {
		return 0, fmt.Errorf("no active campaign — call set_active first")
	}
	return strconv.ParseInt(idStr, 10, 64)
}

func (s *Server) handleCreateWorldNote(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	campID, err := s.activeCampaignID()
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}
	title, ok := reqStr(req, "title")
	if !ok {
		return mcplib.NewToolResultError("title is required"), nil
	}
	content, ok := reqStr(req, "content")
	if !ok {
		return mcplib.NewToolResultError("content is required"), nil
	}
	category, ok := reqStr(req, "category")
	if !ok {
		return mcplib.NewToolResultError("category is required"), nil
	}

	noteID, err := s.db.CreateWorldNote(campID, title, content, category)
	if err != nil {
		return mcplib.NewToolResultError("create note: " + err.Error()), nil
	}

	sessID, _ := s.activeSessionID()
	s.logNarrative(req, sessID)
	s.bus.Publish(api.Event{Type: api.EventWorldNoteCreated, Payload: map[string]any{"note_id": noteID, "title": title}})
	return mcplib.NewToolResultText(fmt.Sprintf("world note %d created: %s", noteID, title)), nil
}

func (s *Server) handleUpdateWorldNote(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	noteID, ok := optInt64(req, "note_id")
	if !ok {
		return mcplib.NewToolResultError("note_id is required"), nil
	}
	title, ok := reqStr(req, "title")
	if !ok {
		return mcplib.NewToolResultError("title is required"), nil
	}
	content, ok := reqStr(req, "content")
	if !ok {
		return mcplib.NewToolResultError("content is required"), nil
	}

	tagsJSON := ""
	if tagsRaw := optStr(req, "tags"); tagsRaw != "" {
		var tags []string
		if err := json.Unmarshal([]byte(tagsRaw), &tags); err != nil {
			return mcplib.NewToolResultError("tags must be a JSON array of strings"), nil
		}
		b, _ := json.Marshal(tags)
		tagsJSON = string(b)
	}

	if err := s.db.UpdateWorldNote(noteID, title, content, tagsJSON); err != nil {
		return mcplib.NewToolResultError("update note: " + err.Error()), nil
	}

	sessID, _ := s.activeSessionID()
	s.logNarrative(req, sessID)
	s.bus.Publish(api.Event{Type: api.EventWorldNoteUpdated, Payload: map[string]any{"note_id": noteID}})
	return mcplib.NewToolResultText(fmt.Sprintf("world note %d updated", noteID)), nil
}

func (s *Server) handleSearchWorldNotes(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	campID, err := s.activeCampaignID()
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}
	query := optStr(req, "query")
	category := optStr(req, "category")

	notes, err := s.db.SearchWorldNotes(campID, query, category, "", nil)
	if err != nil {
		return mcplib.NewToolResultError("search error: " + err.Error()), nil
	}
	if notes == nil {
		notes = []db.WorldNote{}
	}
	b, err := json.Marshal(notes)
	if err != nil {
		return mcplib.NewToolResultError("marshal error: " + err.Error()), nil
	}
	return mcplib.NewToolResultText(string(b)), nil
}
