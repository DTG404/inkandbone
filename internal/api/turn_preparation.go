package api

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/digitalghost404/inkandbone/internal/db"
)

type preparedTurn struct {
	Roll    *rollCheckResult
	Command string
}
type turnEffects struct {
	dice   []db.TurnDice
	stats  *db.TurnCharacterChange
	events []Event
}
type turnEffectsKey struct{}

func (s *Server) recordTurnDice(ctx context.Context, sid int64, expr string, total int, breakdown string) {
	if e, ok := ctx.Value(turnEffectsKey{}).(*turnEffects); ok {
		e.dice = append(e.dice, db.TurnDice{Expression: expr, Total: total, Breakdown: breakdown})
		return
	}
	_, _ = s.db.LogDiceRoll(sid, expr, total, breakdown)
}
func (s *Server) turnEvent(ctx context.Context, event Event) {
	if e, ok := ctx.Value(turnEffectsKey{}).(*turnEffects); ok {
		e.events = append(e.events, event)
		return
	}
	s.bus.Publish(event)
}
func (s *Server) turnCharacter(ctx context.Context, sid, id int64, before, after string) error {
	if e, ok := ctx.Value(turnEffectsKey{}).(*turnEffects); ok {
		e.stats = &db.TurnCharacterChange{ID: id, Before: before, After: after}
		return nil
	}
	if source := turnSource(ctx, sid); source.MessageID > 0 {
		_, err := s.db.ApplyTurnStats(source, id, turnStatReason(ctx, "rouse"), before, after)
		return err
	}
	return s.db.UpdateCharacterData(id, after)
}
func (s *Server) prepareAction(ctx context.Context, sid int64, action db.Message, name, token string, claim db.TurnClaim) (preparedTurn, error) {
	var result preparedTurn
	if claim.Preparation.Valid {
		err := json.Unmarshal([]byte(claim.Preparation.String), &result)
		return result, err
	}
	effects := &turnEffects{}
	ctx = context.WithValue(ctx, turnEffectsKey{}, effects)
	text := strings.ToLower(action.Content)
	if rouseCheckRE.MatchString(text) || bloodSurgeRE.MatchString(text) {
		if sess, err := s.db.GetSession(sid); err == nil && sess != nil {
			if camp, err := s.db.GetCampaign(sess.CampaignID); err == nil && camp != nil {
				if rs, err := s.db.GetRuleset(camp.RulesetID); err == nil && rs != nil && rs.Name == "vtm" {
					if bloodSurgeRE.MatchString(text) {
						result.Command = s.handleVtMBloodSurge(ctx, sid)
					} else {
						result.Command = s.handleVtMRouseCheck(ctx, sid)
					}
				}
			}
		}
	}
	if result.Command == "" {
		result.Roll = s.checkAndExecuteRoll(ctx, sid, playerInputForGM(action.Content), name)
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return result, err
	}
	if err = s.db.SaveTurnPreparation(sid, action.ID, token, string(encoded), effects.dice, effects.stats); err != nil {
		return result, err
	}
	for _, event := range effects.events {
		s.bus.Publish(event)
	}
	return result, nil
}
