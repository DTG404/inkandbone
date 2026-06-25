package api

import (
	"encoding/json"
	"net/http"

	"github.com/digitalghost404/inkandbone/internal/db"
)

func (s *Server) handleListMacros(w http.ResponseWriter, r *http.Request) {
	charID, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid character id", http.StatusBadRequest)
		return
	}
	macros, err := s.db.ListMacros(charID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if macros == nil {
		macros = []db.Macro{}
	}
	writeJSON(w, macros)
}

func (s *Server) handleCreateMacro(w http.ResponseWriter, r *http.Request) {
	charID, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid character id", http.StatusBadRequest)
		return
	}

	// Enforce 10-macro cap
	existing, err := s.db.ListMacros(charID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(existing) >= 10 {
		http.Error(w, "maximum 10 macros per character", http.StatusUnprocessableEntity)
		return
	}

	var body struct {
		Label      string `json:"label"`
		ActionText string `json:"action_text"`
		Color      string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Label == "" || body.ActionText == "" {
		http.Error(w, "label and action_text are required", http.StatusBadRequest)
		return
	}
	color := body.Color
	if color == "" {
		color = "var(--gold)"
	}
	id, err := s.db.CreateMacro(charID, body.Label, body.ActionText, color)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]any{"id": id})
}

func (s *Server) handlePatchMacro(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid macro id", http.StatusBadRequest)
		return
	}
	var body struct {
		Label      string `json:"label"`
		ActionText string `json:"action_text"`
		Color      string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Label == "" || body.ActionText == "" {
		http.Error(w, "label and action_text are required", http.StatusBadRequest)
		return
	}
	if err := s.db.UpdateMacro(id, body.Label, body.ActionText, body.Color); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteMacro(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid macro id", http.StatusBadRequest)
		return
	}
	if err := s.db.DeleteMacro(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleReorderMacros(w http.ResponseWriter, r *http.Request) {
	charID, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid character id", http.StatusBadRequest)
		return
	}
	var body struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.IDs) == 0 {
		http.Error(w, "ids required", http.StatusBadRequest)
		return
	}
	if err := s.db.ReorderMacros(charID, body.IDs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
