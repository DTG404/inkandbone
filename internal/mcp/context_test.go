package mcp

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetContext_empty(t *testing.T) {
	s := newTestMCP(t)
	result, err := s.handleGetContext(context.Background(), mcplib.CallToolRequest{})
	require.NoError(t, err)
	require.False(t, result.IsError)
	tc, ok := result.Content[0].(mcplib.TextContent)
	require.True(t, ok)
	var snap map[string]any
	require.NoError(t, json.Unmarshal([]byte(tc.Text), &snap))
	assert.Nil(t, snap["campaign"])
	assert.Nil(t, snap["character"])
	assert.Nil(t, snap["session"])
	assert.Nil(t, snap["active_combat"])
}

func TestGetContext_withActiveState(t *testing.T) {
	s := newTestMCP(t)
	d := s.db

	rs, err := d.GetRulesetByName("dnd5e")
	require.NoError(t, err)
	require.NotNil(t, rs, "dnd5e ruleset must be seeded by migration 002")
	campID, err := d.CreateCampaign(rs.ID, "Test Campaign", "")
	require.NoError(t, err)
	charID, err := d.CreateCharacter(campID, "Arin")
	require.NoError(t, err)
	sessID, err := d.CreateSession(campID, "Session 1", "2026-04-01")
	require.NoError(t, err)

	require.NoError(t, d.SetSetting("active_campaign_id", strconv.FormatInt(campID, 10)))
	require.NoError(t, d.SetSetting("active_character_id", strconv.FormatInt(charID, 10)))
	require.NoError(t, d.SetSetting("active_session_id", strconv.FormatInt(sessID, 10)))

	result, err := s.handleGetContext(context.Background(), mcplib.CallToolRequest{})
	require.NoError(t, err)
	require.False(t, result.IsError)
	tc, ok := result.Content[0].(mcplib.TextContent)
	require.True(t, ok)

	var snap contextSnapshot
	require.NoError(t, json.Unmarshal([]byte(tc.Text), &snap))
	require.NotNil(t, snap.Campaign)
	assert.Equal(t, "Test Campaign", snap.Campaign.Name)
	require.NotNil(t, snap.Character)
	assert.Equal(t, "Arin", snap.Character.Name)
	require.NotNil(t, snap.Session)
	assert.Equal(t, "Session 1", snap.Session.Title)
	assert.Nil(t, snap.ActiveCombat)
}

func TestGetContextExcludesWhispers(t *testing.T) {
	s := newTestMCP(t)
	d := s.db

	rs, err := d.GetRulesetByName("dnd5e")
	require.NoError(t, err)
	require.NotNil(t, rs)
	campID, err := d.CreateCampaign(rs.ID, "Test Campaign", "")
	require.NoError(t, err)
	sessID, err := d.CreateSession(campID, "Session 1", "2026-04-01")
	require.NoError(t, err)
	require.NoError(t, d.SetSetting("active_session_id", strconv.FormatInt(sessID, 10)))
	_, err = d.CreateMessage(sessID, "user", "PUBLIC_SENTINEL", false, nil)
	require.NoError(t, err)
	_, err = d.CreateMessage(sessID, "user", "PRIVATE_SENTINEL", true, nil)
	require.NoError(t, err)

	result, err := s.handleGetContext(context.Background(), mcplib.CallToolRequest{})
	require.NoError(t, err)
	require.False(t, result.IsError)
	tc, ok := result.Content[0].(mcplib.TextContent)
	require.True(t, ok)

	var snap contextSnapshot
	require.NoError(t, json.Unmarshal([]byte(tc.Text), &snap))
	require.Len(t, snap.RecentMessages, 1)
	assert.Equal(t, "PUBLIC_SENTINEL", snap.RecentMessages[0].Content)
}

func TestGetContext_withActiveCombat(t *testing.T) {
	s := newTestMCP(t)
	d := s.db

	rs2, err := d.GetRulesetByName("dnd5e")
	require.NoError(t, err)
	require.NotNil(t, rs2, "dnd5e ruleset must be seeded by migration 002")
	campID, err := d.CreateCampaign(rs2.ID, "Test Campaign", "")
	require.NoError(t, err)
	charID, err := d.CreateCharacter(campID, "Arin")
	require.NoError(t, err)
	sessID, err := d.CreateSession(campID, "Session 1", "2026-04-01")
	require.NoError(t, err)

	require.NoError(t, d.SetSetting("active_campaign_id", strconv.FormatInt(campID, 10)))
	require.NoError(t, d.SetSetting("active_character_id", strconv.FormatInt(charID, 10)))
	require.NoError(t, d.SetSetting("active_session_id", strconv.FormatInt(sessID, 10)))

	encID, err := d.CreateEncounter(sessID, "Battle")
	require.NoError(t, err)
	_, err = d.AddCombatant(encID, "Goblin", 12, 7, false, nil)
	require.NoError(t, err)

	result, err := s.handleGetContext(context.Background(), mcplib.CallToolRequest{})
	require.NoError(t, err)
	require.False(t, result.IsError)
	tc, ok := result.Content[0].(mcplib.TextContent)
	require.True(t, ok)

	var snap contextSnapshot
	require.NoError(t, json.Unmarshal([]byte(tc.Text), &snap))
	require.NotNil(t, snap.ActiveCombat)
	require.NotNil(t, snap.ActiveCombat.Encounter)
	assert.Equal(t, "Battle", snap.ActiveCombat.Encounter.Name)
	require.NotEmpty(t, snap.ActiveCombat.Combatants)
	assert.Equal(t, "Goblin", snap.ActiveCombat.Combatants[0].Name)
}
