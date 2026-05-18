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

func (s *Server) handleCreateCampaign(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	rulesetName, ok := reqStr(req, "ruleset")
	if !ok {
		return mcplib.NewToolResultError("ruleset is required"), nil
	}
	name, ok := reqStr(req, "name")
	if !ok {
		return mcplib.NewToolResultError("name is required"), nil
	}
	description := optStr(req, "description")

	rs, err := s.db.GetRulesetByName(rulesetName)
	if err != nil {
		return mcplib.NewToolResultError("db error: " + err.Error()), nil
	}
	if rs == nil {
		return mcplib.NewToolResultError(fmt.Sprintf("unknown ruleset %q — valid: dnd5e, ironsworn, vtm, coc, cyberpunk", rulesetName)), nil
	}

	campID, err := s.db.CreateCampaign(rs.ID, name, description)
	if err != nil {
		return mcplib.NewToolResultError("create campaign: " + err.Error()), nil
	}
	if err := s.db.SetSetting("active_campaign_id", strconv.FormatInt(campID, 10)); err != nil {
		return mcplib.NewToolResultError("set active campaign: " + err.Error()), nil
	}

	s.bus.Publish(api.Event{Type: api.EventCampaignCreated, Payload: map[string]any{"campaign_id": campID, "name": name}})
	return mcplib.NewToolResultText(fmt.Sprintf("campaign %d created and activated: %s (%s)", campID, name, rulesetName)), nil
}

func (s *Server) handleListCampaigns(_ context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	campaigns, err := s.db.ListCampaigns()
	if err != nil {
		return mcplib.NewToolResultError("db error: " + err.Error()), nil
	}
	if campaigns == nil {
		campaigns = []db.Campaign{}
	}
	b, err := json.Marshal(campaigns)
	if err != nil {
		return mcplib.NewToolResultError("marshal error: " + err.Error()), nil
	}
	return mcplib.NewToolResultText(string(b)), nil
}

func (s *Server) handleCreateCharacter(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	name, ok := reqStr(req, "name")
	if !ok {
		return mcplib.NewToolResultError("name is required"), nil
	}

	var campID int64
	if id, ok := optInt64(req, "campaign_id"); ok && id > 0 {
		campID = id
	} else {
		var err error
		campID, err = s.activeCampaignID()
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
	}

	charID, err := s.db.CreateCharacter(campID, name)
	if err != nil {
		return mcplib.NewToolResultError("create character: " + err.Error()), nil
	}
	if err := s.db.SetSetting("active_character_id", strconv.FormatInt(charID, 10)); err != nil {
		return mcplib.NewToolResultError("set active character: " + err.Error()), nil
	}

	msg := fmt.Sprintf("character %d created and activated: %s", charID, name)

	camp, err := s.db.GetCampaign(campID)
	if err != nil || camp == nil {
		return mcplib.NewToolResultError("could not load campaign for random stats"), nil
	}
	rs, err := s.db.GetRuleset(camp.RulesetID)
	if err != nil || rs == nil {
		return mcplib.NewToolResultError("could not load ruleset for random stats"), nil
	}
	stats := rollStats(rs.Name, "")
	if len(stats) > 0 {
		dataJSON, err := json.Marshal(stats)
		if err != nil {
			return mcplib.NewToolResultError("marshal stats: " + err.Error()), nil
		}
		if err := s.db.UpdateCharacterData(charID, string(dataJSON)); err != nil {
			return mcplib.NewToolResultError("save stats: " + err.Error()), nil
		}
		msg += fmt.Sprintf(" (stats rolled for %s)", rs.Name)
	}

	s.bus.Publish(api.Event{Type: api.EventCharacterCreated, Payload: map[string]any{"character_id": charID, "name": name}})
	return mcplib.NewToolResultText(msg), nil
}

func (s *Server) handleListCharacters(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	var campID int64
	if id, ok := optInt64(req, "campaign_id"); ok && id > 0 {
		campID = id
	} else {
		var err error
		campID, err = s.activeCampaignID()
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
	}

	characters, err := s.db.ListCharacters(campID)
	if err != nil {
		return mcplib.NewToolResultError("db error: " + err.Error()), nil
	}
	if characters == nil {
		characters = []db.Character{}
	}
	b, err := json.Marshal(characters)
	if err != nil {
		return mcplib.NewToolResultError("marshal error: " + err.Error()), nil
	}
	return mcplib.NewToolResultText(string(b)), nil
}

func (s *Server) handleListSessions(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	var campID int64
	if id, ok := optInt64(req, "campaign_id"); ok && id > 0 {
		campID = id
	} else {
		var err error
		campID, err = s.activeCampaignID()
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
	}

	sessions, err := s.db.ListSessions(campID)
	if err != nil {
		return mcplib.NewToolResultError("db error: " + err.Error()), nil
	}
	if sessions == nil {
		sessions = []db.Session{}
	}
	b, err := json.Marshal(sessions)
	if err != nil {
		return mcplib.NewToolResultError("marshal error: " + err.Error()), nil
	}
	return mcplib.NewToolResultText(string(b)), nil
}

func (s *Server) handleCloseCampaign(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	// Resolve campaign ID.
	var id int64
	if paramID, ok := optInt64(req, "campaign_id"); ok && paramID > 0 {
		id = paramID
	} else {
		var err error
		id, err = s.activeCampaignID()
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
	}

	// Load the campaign.
	campaign, err := s.db.GetCampaign(id)
	if err != nil {
		return mcplib.NewToolResultError("db error: " + err.Error()), nil
	}
	if campaign == nil {
		return mcplib.NewToolResultError(fmt.Sprintf("campaign %d not found", id)), nil
	}

	// Guard checks the active_session_id setting only; does not scan all sessions in the DB.
	sessIDStr, _ := s.db.GetSetting("active_session_id")
	if sessIDStr != "" {
		sessID, parseErr := strconv.ParseInt(sessIDStr, 10, 64)
		if parseErr == nil && sessID > 0 {
			sess, sessErr := s.db.GetSession(sessID)
			if sessErr != nil {
				return mcplib.NewToolResultError("db error: " + sessErr.Error()), nil
			}
			if sess != nil && sess.CampaignID == id && sess.Summary == "" {
				return mcplib.NewToolResultError("end your current session before closing the campaign"), nil
			}
		}
	}

	// Close the campaign.
	if err := s.db.CloseCampaign(id); err != nil {
		return mcplib.NewToolResultError("close campaign: " + err.Error()), nil
	}

	// Clear active_campaign_id if it points to this campaign.
	if campIDStr, _ := s.db.GetSetting("active_campaign_id"); campIDStr != "" {
		if activeCampID, parseErr := strconv.ParseInt(campIDStr, 10, 64); parseErr == nil && activeCampID == id {
			_ = s.db.SetSetting("active_campaign_id", "")
		}
	}

	// Clear active_session_id if the session belongs to this campaign.
	if sessIDStr != "" {
		if sessID, parseErr := strconv.ParseInt(sessIDStr, 10, 64); parseErr == nil && sessID > 0 {
			sess, sessErr := s.db.GetSession(sessID)
			if sessErr == nil && sess != nil && sess.CampaignID == id {
				_ = s.db.SetSetting("active_session_id", "")
			}
		}
	}

	// Clear active_character_id if the character belongs to this campaign.
	if charIDStr, _ := s.db.GetSetting("active_character_id"); charIDStr != "" {
		if charID, parseErr := strconv.ParseInt(charIDStr, 10, 64); parseErr == nil && charID > 0 {
			char, charErr := s.db.GetCharacter(charID)
			if charErr == nil && char != nil && char.CampaignID == id {
				_ = s.db.SetSetting("active_character_id", "")
			}
		}
	}

	s.bus.Publish(api.Event{Type: api.EventCampaignClosed, Payload: map[string]any{"campaign_id": id}})
	return mcplib.NewToolResultText(fmt.Sprintf("campaign %d closed: %s", id, campaign.Name)), nil
}

func (s *Server) handleDeleteCampaign(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	// campaign_id is required.
	id, ok := optInt64(req, "campaign_id")
	if !ok || id == 0 {
		return mcplib.NewToolResultError("campaign_id is required"), nil
	}

	// Read confirm bool.
	args := req.GetArguments()
	confirm, _ := args["confirm"].(bool)

	// Load the campaign.
	campaign, err := s.db.GetCampaign(id)
	if err != nil {
		return mcplib.NewToolResultError("db error: " + err.Error()), nil
	}
	if campaign == nil {
		return mcplib.NewToolResultError(fmt.Sprintf("campaign %d not found", id)), nil
	}

	if !confirm {
		stats, statsErr := s.db.GetCampaignStats(id)
		if statsErr != nil {
			return mcplib.NewToolResultError("db error: " + statsErr.Error()), nil
		}
		msg := fmt.Sprintf(
			"campaign %d %q and all its data will be permanently deleted:\n  - %d sessions, %d characters, %d world notes, %d maps\ncall delete_campaign again with confirm: true to proceed",
			id, campaign.Name, stats.Sessions, stats.Characters, stats.WorldNotes, stats.Maps,
		)
		return mcplib.NewToolResultError(msg), nil
	}

	// confirmed — delete.
	if err := s.db.DeleteCampaign(id); err != nil {
		return mcplib.NewToolResultError("delete campaign: " + err.Error()), nil
	}

	// Clear settings that belonged to this campaign.
	if campIDStr, _ := s.db.GetSetting("active_campaign_id"); campIDStr != "" {
		if activeCampID, parseErr := strconv.ParseInt(campIDStr, 10, 64); parseErr == nil && activeCampID == id {
			_ = s.db.SetSetting("active_campaign_id", "")
		}
	}
	// active_session_id — clear only if the session belonged to this campaign.
	if sessIDStr, _ := s.db.GetSetting("active_session_id"); sessIDStr != "" {
		if sessID, parseErr := strconv.ParseInt(sessIDStr, 10, 64); parseErr == nil && sessID > 0 {
			sess, sessErr := s.db.GetSession(sessID)
			if sessErr == nil && (sess == nil || sess.CampaignID == id) {
				_ = s.db.SetSetting("active_session_id", "")
			}
		}
	}
	// active_character_id — clear only if the character belonged to this campaign.
	if charIDStr, _ := s.db.GetSetting("active_character_id"); charIDStr != "" {
		if charID, parseErr := strconv.ParseInt(charIDStr, 10, 64); parseErr == nil && charID > 0 {
			char, charErr := s.db.GetCharacter(charID)
			if charErr == nil && (char == nil || char.CampaignID == id) {
				_ = s.db.SetSetting("active_character_id", "")
			}
		}
	}

	s.bus.Publish(api.Event{Type: api.EventCampaignDeleted, Payload: map[string]any{"campaign_id": id}})
	return mcplib.NewToolResultText(fmt.Sprintf("campaign %d deleted", id)), nil
}
