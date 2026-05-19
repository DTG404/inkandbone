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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Title == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}
	status := body.Status
	if status == "" {
		status = "upcoming"
	}

	id, err := s.db.CreateAdventure(campaignID, body.Title, body.Description, status, body.SortOrder)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	s.bus.Publish(Event{Type: EventAdventureUpdated, Payload: map[string]any{"campaign_id": campaignID}})

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
		http.Error(w, "db error", http.StatusInternalServerError)
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
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
	if err := s.db.UpdateAdventure(id, body.Title, body.Description, status, body.SortOrder); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.bus.Publish(Event{Type: EventAdventureUpdated, Payload: map[string]any{"id": id}})
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDeleteAdventure(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.db.DeleteAdventure(id); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := s.db.SetSessionAdventure(sessionID, body.AdventureID); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.bus.Publish(Event{Type: EventAdventureUpdated, Payload: map[string]any{"session_id": sessionID}})
	w.WriteHeader(http.StatusOK)
}
