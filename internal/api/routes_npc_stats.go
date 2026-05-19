package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/digitalghost404/inkandbone/internal/db"
)

func (s *Server) handleCreateNpcStat(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	campaignID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	var body struct {
		Name          string `json:"name"`
		Role          string `json:"role"`
		DataJSON      string `json:"data_json"`
		HPMax         int    `json:"hp_max"`
		ArmorClass    *int   `json:"armor_class"`
		InitiativeMod int    `json:"initiative_mod"`
		Skills        string `json:"skills"`
		Abilities     string `json:"abilities"`
		Loot          string `json:"loot"`
		Notes         string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	hpMax := body.HPMax
	if hpMax < 1 {
		hpMax = 1
	}

	createdID, err := s.db.CreateNpcStat(campaignID, body.Name, body.Role, body.DataJSON, hpMax, body.ArmorClass, body.InitiativeMod, body.Skills, body.Abilities, body.Loot, body.Notes)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	s.bus.Publish(Event{Type: EventNpcStatUpdated, Payload: map[string]any{"campaign_id": campaignID}})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": createdID}) //nolint:errcheck
}

func (s *Server) handleListNpcStats(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	campaignID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	stats, err := s.db.ListNpcStats(campaignID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if stats == nil {
		stats = []db.NpcStat{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats) //nolint:errcheck
}

func (s *Server) handleGetNpcStat(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	n, err := s.db.GetNpcStat(id)
	if err != nil {
		http.Error(w, "npc stat not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(n) //nolint:errcheck
}

func (s *Server) handleUpdateNpcStat(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body struct {
		Name          string `json:"name"`
		Role          string `json:"role"`
		DataJSON      string `json:"data_json"`
		HPMax         int    `json:"hp_max"`
		ArmorClass    *int   `json:"armor_class"`
		InitiativeMod int    `json:"initiative_mod"`
		Skills        string `json:"skills"`
		Abilities     string `json:"abilities"`
		Loot          string `json:"loot"`
		Notes         string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	hpMax := body.HPMax
	if hpMax < 1 {
		hpMax = 1
	}

	if err := s.db.UpdateNpcStats(id, body.Name, body.Role, body.DataJSON, hpMax, body.ArmorClass, body.InitiativeMod, body.Skills, body.Abilities, body.Loot, body.Notes); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.bus.Publish(Event{Type: EventNpcStatUpdated, Payload: map[string]any{"id": id}})
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDeleteNpcStat(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.db.DeleteNpcStat(id); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
