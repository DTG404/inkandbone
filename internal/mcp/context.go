package mcp

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/digitalghost404/inkandbone/internal/db"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

type combatSnapshot struct {
	Encounter  *db.CombatEncounter `json:"encounter"`
	Combatants []db.Combatant      `json:"combatants"`
}

type contextSnapshot struct {
	Campaign       *db.Campaign    `json:"campaign"`
	Character      *db.Character   `json:"character"`
	Session        *db.Session     `json:"session"`
	RecentMessages []db.Message    `json:"recent_messages"`
	ActiveCombat   *combatSnapshot `json:"active_combat"`
}

func (s *Server) handleGetContext(_ context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	snap := contextSnapshot{
		RecentMessages: []db.Message{},
	}

	// active campaign
	if campIDStr, err := s.db.GetSetting("active_campaign_id"); err == nil && campIDStr != "" {
		if campID, err := strconv.ParseInt(campIDStr, 10, 64); err == nil {
			snap.Campaign, _ = s.db.GetCampaign(campID)
		}
	}

	// active character
	if charIDStr, err := s.db.GetSetting("active_character_id"); err == nil && charIDStr != "" {
		if charID, err := strconv.ParseInt(charIDStr, 10, 64); err == nil {
			snap.Character, _ = s.db.GetCharacter(charID)
		}
	}

	// active session + recent messages + active combat
	if sessIDStr, err := s.db.GetSetting("active_session_id"); err == nil && sessIDStr != "" {
		if sessID, err := strconv.ParseInt(sessIDStr, 10, 64); err == nil {
			snap.Session, _ = s.db.GetSession(sessID)

			if msgs, err := s.db.ListAIVisibleMessages(sessID); err == nil {
				if len(msgs) > 20 {
					msgs = msgs[len(msgs)-20:]
				}
				snap.RecentMessages = msgs
			}

			if enc, err := s.db.GetActiveEncounter(sessID); err == nil && enc != nil {
				cs := &combatSnapshot{Encounter: enc}
				if combatants, err := s.db.ListCombatants(enc.ID); err == nil {
					cs.Combatants = combatants
				}
				snap.ActiveCombat = cs
			}
		}
	}

	b, err := json.Marshal(snap)
	if err != nil {
		return mcplib.NewToolResultError("failed to marshal context: " + err.Error()), nil
	}
	return mcplib.NewToolResultText(string(b)), nil
}
