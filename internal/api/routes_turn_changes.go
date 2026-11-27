package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/digitalghost404/inkandbone/internal/db"
)

type turnAttribution struct{ message, actor int64 }
type turnActorKey struct{}
type turnReasonKey struct{}

func withTurnMessage(ctx context.Context, message int64, actor ...int64) context.Context {
	a := turnAttribution{message: message}
	if len(actor) > 0 {
		a.actor = actor[0]
	}
	return context.WithValue(ctx, turnActorKey{}, a)
}
func turnSource(ctx context.Context, session int64) db.TurnSource {
	a, _ := ctx.Value(turnActorKey{}).(turnAttribution)
	return db.TurnSource{SessionID: session, MessageID: a.message}
}
func turnStatReason(ctx context.Context, fallback string) string {
	if reason, ok := ctx.Value(turnReasonKey{}).(string); ok {
		return reason
	}
	return fallback
}
func (s *Server) turnActorID(ctx context.Context) (int64, error) {
	if turn, ok := ctx.Value(turnActorKey{}).(turnAttribution); ok {
		return turn.actor, nil
	}
	value, err := s.db.GetSetting("active_character_id")
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(value, 10, 64)
}

func (s *Server) registerTurnChangeRoutes() {
	s.mux.HandleFunc("GET /api/sessions/{id}/changes", s.handleListTurnChanges)
	s.mux.HandleFunc("POST /api/sessions/{id}/changes/{change}/undo", s.handleUndoTurnChange)
}
func (s *Server) handleListTurnChanges(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "Invalid session", http.StatusBadRequest)
		return
	}
	changes, err := s.db.ListTurnChanges(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, changes)
}
func (s *Server) handleUndoTurnChange(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	change, valid := parsePathID(r, "change")
	if !ok || !valid {
		http.Error(w, "Invalid change", http.StatusBadRequest)
		return
	}
	c, err := s.db.UndoTurnChange(id, change)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if errors.Is(err, db.ErrChangeConflict) {
		http.Error(w, "Later edits prevent undoing this change.", http.StatusConflict)
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	if c.OwnerID != nil {
		owner := *c.OwnerID
		switch c.Kind {
		case "stat", "currency":
			payload := &CharacterUpdatedPayload{ID: RealtimeInt64(owner), CharacterID: RealtimeInt64(owner), SessionID: RealtimeInt64(id)}
			if char, err := s.db.GetCharacter(owner); err == nil && char != nil {
				payload.DataJson = RealtimePtr(char.DataJSON)
				payload.CurrencyBalance = RealtimeInt64(char.CurrencyBalance)
				payload.CurrencyLabel = RealtimePtr(char.CurrencyLabel)
			}
			s.bus.Publish(Event{Type: EventCharacterUpdated, Payload: payload})
		case "npc":
			s.bus.Publish(Event{Type: EventNPCUpdated, Payload: &NPCUpdatedPayload{SessionID: RealtimeInt64(id)}})
		case "item":
			s.bus.Publish(Event{Type: EventItemUpdated, Payload: &ItemUpdatedPayload{CharacterID: RealtimeInt64(owner)}})
		case "objective":
			s.bus.Publish(Event{Type: EventObjectiveUpdated, Payload: &ObjectiveUpdatedPayload{CampaignID: RealtimeInt64(owner)}})
		}
	}
	writeJSON(w, c)
}
