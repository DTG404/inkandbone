package api

import (
	"net/http"

	"github.com/digitalghost404/inkandbone/internal/db"
)

func (s *Server) handleListTokens(w http.ResponseWriter, r *http.Request) {
	mapID, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid map id", http.StatusBadRequest)
		return
	}
	tokens, err := s.db.ListMapTokens(mapID)
	if err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if tokens == nil {
		tokens = []db.MapToken{}
	}
	writeJSON(w, tokens)
}

func (s *Server) handlePlaceToken(w http.ResponseWriter, r *http.Request) {
	mapID, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid map id", http.StatusBadRequest)
		return
	}
	var body struct {
		EntityType string  `json:"entity_type"`
		EntityID   int64   `json:"entity_id"`
		X          float64 `json:"x"`
		Y          float64 `json:"y"`
	}
	if err := decodeJSON(r, &body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.EntityType != "character" && body.EntityType != "npc" {
		http.Error(w, "entity_type must be character or npc", http.StatusBadRequest)
		return
	}
	tokenID, err := s.db.PlaceToken(mapID, body.EntityType, body.EntityID, body.X, body.Y)
	if err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusConflict)
		return
	}
	token, err := s.db.GetToken(tokenID)
	if err != nil || token == nil {
		http.Error(w, "fetch token", http.StatusInternalServerError)
		return
	}
	s.bus.Publish(Event{Type: EventTokenPlaced, Payload: map[string]any{
		"map_id": mapID,
		"token":  token,
	}})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, token)
}

func (s *Server) handleMoveToken(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid token id", http.StatusBadRequest)
		return
	}
	var body struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}
	if err := decodeJSON(r, &body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	token, err := s.db.GetToken(id)
	if err != nil || token == nil {
		http.Error(w, "token not found", http.StatusNotFound)
		return
	}
	if err := s.db.MoveToken(id, body.X, body.Y); err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.bus.Publish(Event{Type: EventTokenMoved, Payload: map[string]any{
		"map_id":   token.MapID,
		"token_id": id,
		"x":        body.X,
		"y":        body.Y,
	}})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRemoveToken(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid token id", http.StatusBadRequest)
		return
	}
	token, err := s.db.GetToken(id)
	if err != nil || token == nil {
		http.Error(w, "token not found", http.StatusNotFound)
		return
	}
	if err := s.db.RemoveToken(id); err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.bus.Publish(Event{Type: EventTokenRemoved, Payload: map[string]any{
		"map_id":   token.MapID,
		"token_id": id,
	}})
	w.WriteHeader(http.StatusNoContent)
}
