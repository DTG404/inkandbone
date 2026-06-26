package api

import (
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/digitalghost404/inkandbone/internal/db"
)

func (s *Server) handleListDecks(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	decks, err := s.db.ListDecks(campaignID)
	if err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if decks == nil {
		decks = []db.Deck{}
	}
	writeJSON(w, decks)
}

func (s *Server) handleCreateDeck(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	var body struct {
		Name  string            `json:"name"`
		Cards []json.RawMessage `json:"cards"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if len(body.Cards) == 0 {
		http.Error(w, "at least 1 card required", http.StatusBadRequest)
		return
	}
	if len(body.Cards) > 200 {
		http.Error(w, "maximum 200 cards", http.StatusBadRequest)
		return
	}
	cardsJSON, err := json.Marshal(body.Cards)
	if err != nil {
		http.Error(w, "encode cards: "+err.Error(), http.StatusBadRequest)
		return
	}
	id, err := s.db.CreateDeck(campaignID, body.Name, string(cardsJSON))
	if err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id}) //nolint:errcheck
}

func (s *Server) handleDeleteDeck(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid deck id", http.StatusBadRequest)
		return
	}
	if err := s.db.DeleteDeck(id); err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleShuffleDeck(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid deck id", http.StatusBadRequest)
		return
	}
	deck, err := s.db.GetDeck(id)
	if err != nil || deck == nil {
		http.Error(w, "deck not found", http.StatusNotFound)
		return
	}
	var cards []json.RawMessage
	if err := json.Unmarshal([]byte(deck.CardsJSON), &cards); err != nil {
		http.Error(w, "deck cards malformed", http.StatusInternalServerError)
		return
	}
	order := rand.Perm(len(cards))
	orderJSON, _ := json.Marshal(order)
	if err := s.db.ShuffleDeck(id, string(orderJSON)); err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDrawCard(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid deck id", http.StatusBadRequest)
		return
	}
	var body struct {
		SessionID int64 `json:"session_id"`
	}
	if err := decodeJSON(r, &body); err != nil || body.SessionID == 0 {
		http.Error(w, "session_id required", http.StatusBadRequest)
		return
	}
	deck, err := s.db.GetDeck(id)
	if err != nil || deck == nil {
		http.Error(w, "deck not found", http.StatusNotFound)
		return
	}
	var cards []json.RawMessage
	if err := json.Unmarshal([]byte(deck.CardsJSON), &cards); err != nil {
		http.Error(w, "deck malformed", http.StatusInternalServerError)
		return
	}
	var order []int
	if err := json.Unmarshal([]byte(deck.ShuffledOrderJSON), &order); err != nil || len(order) == 0 {
		writeJSON(w, map[string]any{"exhausted": true})
		return
	}
	if deck.DrawIndex >= len(order) {
		writeJSON(w, map[string]any{"exhausted": true})
		return
	}
	cardIdx := order[deck.DrawIndex]
	if cardIdx >= len(cards) {
		http.Error(w, "card index out of range", http.StatusInternalServerError)
		return
	}
	cardJSON := string(cards[cardIdx])
	if err := s.db.DrawCard(id, deck.DrawIndex, deck.DrawIndex+1, body.SessionID, cardJSON); err != nil {
		if errors.Is(err, db.ErrDrawConflict) {
			http.Error(w, "draw conflict: try again", http.StatusConflict)
			return
		}
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var card map[string]any
	json.Unmarshal([]byte(cardJSON), &card) //nolint:errcheck
	s.bus.Publish(Event{Type: EventCardDrawn, Payload: map[string]any{
		"deck_id":    id,
		"deck_name":  deck.Name,
		"card":       card,
		"draw_index": deck.DrawIndex + 1,
		"total":      len(order),
	}})
	writeJSON(w, map[string]any{"card": card, "draw_index": deck.DrawIndex + 1, "total": len(order)})
}

func (s *Server) handleListDeckDraws(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	sessionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	draws, err := s.db.ListDeckDraws(sessionID)
	if err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if draws == nil {
		draws = []db.DeckDraw{}
	}
	writeJSON(w, draws)
}
