package mcp

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/digitalghost404/inkandbone/internal/db"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupActiveCampaign(t *testing.T, s *Server) int64 {
	t.Helper()
	rs, err := s.db.GetRulesetByName("dnd5e")
	require.NoError(t, err)
	require.NotNil(t, rs, "dnd5e ruleset must be seeded by migration 002")
	campID, err := s.db.CreateCampaign(rs.ID, "Camp", "")
	require.NoError(t, err)
	require.NoError(t, s.db.SetSetting("active_campaign_id", strconv.FormatInt(campID, 10)))
	return campID
}

func TestCreateWorldNote(t *testing.T) {
	s := newTestMCP(t)
	setupActiveCampaign(t, s)

	req := mcplib.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"title":    "Mira",
		"content":  "A mysterious merchant.",
		"category": "npc",
	}
	result, err := s.handleCreateWorldNote(context.Background(), req)
	require.NoError(t, err)
	require.False(t, result.IsError)

	campIDStr, _ := s.db.GetSetting("active_campaign_id")
	campID, _ := strconv.ParseInt(campIDStr, 10, 64)
	notes, err := s.db.SearchWorldNotes(campID, "Mira", "", "", nil)
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, "Mira", notes[0].Title)
}

func TestUpdateWorldNote(t *testing.T) {
	s := newTestMCP(t)
	campID := setupActiveCampaign(t, s)

	noteID, err := s.db.CreateWorldNote(campID, "Old Title", "Old content", "location")
	require.NoError(t, err)

	req := mcplib.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"note_id": float64(noteID),
		"title":   "New Title",
		"content": "New content",
	}
	result, err := s.handleUpdateWorldNote(context.Background(), req)
	require.NoError(t, err)
	require.False(t, result.IsError)

	notes, err := s.db.SearchWorldNotes(campID, "New Title", "", "", nil)
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, "New content", notes[0].Content)
}

func TestUpdateWorldNote_withTags(t *testing.T) {
	s := newTestMCP(t)
	campID := setupActiveCampaign(t, s)

	noteID, err := s.db.CreateWorldNote(campID, "Old", "Old content", "npc")
	require.NoError(t, err)

	req := mcplib.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"note_id": float64(noteID),
		"title":   "New",
		"content": "New content",
		"tags":    `["boss","undead"]`,
	}
	result, err := s.handleUpdateWorldNote(context.Background(), req)
	require.NoError(t, err)
	require.False(t, result.IsError)

	notes, err := s.db.SearchWorldNotes(campID, "New", "", "boss", nil)
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Contains(t, notes[0].TagsJSON, "boss")
}

func TestSearchWorldNotes(t *testing.T) {
	s := newTestMCP(t)
	campID := setupActiveCampaign(t, s)

	_, err := s.db.CreateWorldNote(campID, "Iron Gate", "A fortified gate.", "location")
	require.NoError(t, err)
	_, err = s.db.CreateWorldNote(campID, "Dagger", "Sharp blade.", "item")
	require.NoError(t, err)

	req := mcplib.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": "gate"}
	result, err := s.handleSearchWorldNotes(context.Background(), req)
	require.NoError(t, err)
	require.False(t, result.IsError)
	tc, ok := result.Content[0].(mcplib.TextContent)
	require.True(t, ok)
	var notes []db.WorldNote
	require.NoError(t, json.Unmarshal([]byte(tc.Text), &notes))
	require.Len(t, notes, 1)
	assert.Equal(t, "Iron Gate", notes[0].Title)
}
