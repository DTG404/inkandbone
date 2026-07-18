package mcp

import (
	"context"
	"encoding/json"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRollDice(t *testing.T) {
	s := newTestMCP(t)
	sessID := setupActiveSession(t, s)
	events := s.bus.SubscribeContext(t.Context())

	req := mcplib.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"expression": "2d6+3",
	}
	result, err := s.handleRollDice(context.Background(), req)
	require.NoError(t, err)
	require.False(t, result.IsError)
	tc, ok := result.Content[0].(mcplib.TextContent)
	require.True(t, ok)
	assert.Contains(t, tc.Text, "2d6+3")

	// Verify DB log
	rolls, err := s.db.ListDiceRolls(sessID)
	require.NoError(t, err)
	require.Len(t, rolls, 1)
	assert.Equal(t, "2d6+3", rolls[0].Expression)
	assert.GreaterOrEqual(t, rolls[0].Result, 5) // min: 1+1+3
	assert.LessOrEqual(t, rolls[0].Result, 15)   // max: 6+6+3

	event := <-events
	require.Equal(t, "dice_rolled", string(event.Type))
	payload := eventPayload(t, event)
	assert.EqualValues(t, sessID, payload["session_id"])
	assert.EqualValues(t, rolls[0].Result, payload["result"])
	assert.Equal(t, "2d6+3", payload["expression"])
	assert.Equal(t, rolls[0].BreakdownJSON, mustJSON(t, payload["breakdown"]))
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	return string(encoded)
}

func TestRollDice_invalidExpression(t *testing.T) {
	s := newTestMCP(t)
	setupActiveSession(t, s)

	req := mcplib.CallToolRequest{}
	req.Params.Arguments = map[string]any{"expression": "notdice"}
	result, err := s.handleRollDice(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestRollDice_noSession(t *testing.T) {
	s := newTestMCP(t)
	// No active session set
	req := mcplib.CallToolRequest{}
	req.Params.Arguments = map[string]any{"expression": "d20"}
	result, err := s.handleRollDice(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}
