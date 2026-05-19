package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/digitalghost404/inkandbone/internal/db"
)

func (s *Server) handleCreateSecret(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	campaignID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	var body struct {
		Title    string `json:"title"`
		Content  string `json:"content"`
		Category string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Title == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}
	category := body.Category
	if category == "" {
		category = "secret"
	}

	id, err := s.db.CreateSecret(campaignID, body.Title, body.Content, category)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	s.bus.Publish(Event{Type: EventSecretsUpdated, Payload: map[string]any{"campaign_id": campaignID}})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id}) //nolint:errcheck
}

func (s *Server) handleListSecrets(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	campaignID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	secrets, err := s.db.ListSecretsByCampaign(campaignID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if secrets == nil {
		secrets = []db.Secret{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(secrets) //nolint:errcheck
}

func (s *Server) handleGetSecret(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	secret, err := s.db.GetSecret(id)
	if err != nil {
		http.Error(w, "secret not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(secret) //nolint:errcheck
}

func (s *Server) handleRevealSecret(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body struct {
		SessionID int64 `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SessionID == 0 {
		http.Error(w, "session_id required", http.StatusBadRequest)
		return
	}
	if err := s.db.RevealSecret(id, body.SessionID); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.bus.Publish(Event{Type: EventSecretsUpdated, Payload: map[string]any{"id": id}})
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleUpdateSecret(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body struct {
		Title    string `json:"title"`
		Content  string `json:"content"`
		Category string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Title == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}
	category := body.Category
	if category == "" {
		category = "secret"
	}
	if err := s.db.UpdateSecret(id, body.Title, body.Content, category); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.bus.Publish(Event{Type: EventSecretsUpdated, Payload: map[string]any{"id": id}})
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDeleteSecret(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.db.DeleteSecret(id); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
