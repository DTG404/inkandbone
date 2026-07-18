package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/digitalghost404/inkandbone/internal/db"
)

func (s *Server) handleCreateAdventure(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	campaignID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.Title == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}
	status := body.Status
	if status == "" {
		status = "upcoming"
	}

	id, err := s.db.CreateAdventure(campaignID, body.Title, body.Description, status, body.SortOrder)
	if err != nil {
		serverError(w, r, err)
		return
	}

	s.bus.Publish(Event{Type: EventAdventureUpdated, Payload: &AdventureUpdatedPayload{CampaignID: RealtimeInt64(campaignID)}})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id}) //nolint:errcheck
}

func (s *Server) handleListAdventures(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	campaignID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	adventures, err := s.db.ListAdventures(campaignID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if adventures == nil {
		adventures = []db.Adventure{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(adventures) //nolint:errcheck
}

func (s *Server) handleGetAdventure(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	a, err := s.db.GetAdventure(id)
	if err != nil {
		http.Error(w, "adventure not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a) //nolint:errcheck
}

func (s *Server) handleUpdateAdventure(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.Title == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}
	adventure, err := s.db.GetAdventure(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if adventure == nil {
		http.Error(w, "adventure not found", http.StatusNotFound)
		return
	}
	status := body.Status
	if status == "" {
		status = "upcoming"
	}
	if err := s.db.UpdateAdventure(id, body.Title, body.Description, status, body.SortOrder); err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventAdventureUpdated, Payload: &AdventureUpdatedPayload{CampaignID: RealtimeInt64(adventure.CampaignID), ID: RealtimeInt64(id)}})
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDeleteAdventure(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	adventure, err := s.db.GetAdventure(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if adventure == nil {
		http.Error(w, "adventure not found", http.StatusNotFound)
		return
	}
	if err := s.db.DeleteAdventure(id); err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventAdventureUpdated, Payload: &AdventureUpdatedPayload{CampaignID: RealtimeInt64(adventure.CampaignID), ID: RealtimeInt64(id)}})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSetSessionAdventure(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	sessionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	var body struct {
		AdventureID *int64 `json:"adventure_id"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	session, err := s.db.GetSession(sessionID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if session == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if err := s.db.SetSessionAdventure(sessionID, body.AdventureID); err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventAdventureUpdated, Payload: &AdventureUpdatedPayload{CampaignID: RealtimeInt64(session.CampaignID), SessionID: RealtimeInt64(sessionID)}})
	w.WriteHeader(http.StatusOK)
}
