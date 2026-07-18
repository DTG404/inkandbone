package mcp

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/digitalghost404/inkandbone/internal/api"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mcpStubCompleter struct {
	response string
	prompt   string
}

func (s *mcpStubCompleter) Generate(_ context.Context, prompt string, _ int) (string, error) {
	s.prompt = prompt
	return s.response, nil
}

func TestGenerateSessionRecap(t *testing.T) {
	stub := &mcpStubCompleter{response: "The heroes fought valiantly."}
	s := newTestMCPWithAI(t, stub)

	// Seed campaign and session
	rs, err := s.db.GetRulesetByName("dnd5e")
	require.NoError(t, err)
	require.NotNil(t, rs)
	campID, err := s.db.CreateCampaign(rs.ID, "Test Campaign", "")
	require.NoError(t, err)
	sessID, err := s.db.CreateSession(campID, "S1", "2026-04-03")
	require.NoError(t, err)
	require.NoError(t, s.db.SetSetting("active_session_id", strconv.FormatInt(sessID, 10)))
	_, err = s.db.CreateMessage(sessID, "user", "PUBLIC_SENTINEL", false, nil)
	require.NoError(t, err)
	_, err = s.db.CreateMessage(sessID, "user", "PRIVATE_SENTINEL", true, nil)
	require.NoError(t, err)

	// Collect WS events
	ch := s.bus.SubscribeContext(t.Context())

	req := mcplib.CallToolRequest{}
	req.Params.Arguments = map[string]any{"session_id": float64(sessID)}
	result, err := s.handleGenerateSessionRecap(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	// Drain one event
	var e api.Event
	select {
	case e = <-ch:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for session_updated event")
	}
	assert.Equal(t, api.EventSessionUpdated, e.Type)
	payload, ok := e.Payload.(map[string]any)
	require.True(t, ok, "expected map[string]any payload")
	assert.Equal(t, sessID, payload["session_id"])
	assert.Equal(t, "The heroes fought valiantly.", payload["summary"])

	// Verify DB updated
	sess, err := s.db.GetSession(sessID)
	require.NoError(t, err)
	assert.Equal(t, "The heroes fought valiantly.", sess.Summary)
	assert.Contains(t, stub.prompt, "PUBLIC_SENTINEL")
	assert.NotContains(t, stub.prompt, "PRIVATE_SENTINEL")
}

func TestGenerateSessionRecap_noAI(t *testing.T) {
	s := newTestMCP(t) // aiClient is nil
	req := mcplib.CallToolRequest{}
	req.Params.Arguments = map[string]any{"session_id": float64(1)}
	result, err := s.handleGenerateSessionRecap(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}
