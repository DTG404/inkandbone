package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/digitalghost404/inkandbone/internal/db"
)

func (s *Server) handleCreateFaction(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	campaignID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	var body struct {
		Name          string `json:"name"`
		Description   string `json:"description"`
		FactionType   string `json:"faction_type"`
		Influence     int    `json:"influence"`
		ResourcesJSON string `json:"resources_json"`
		Color         string `json:"color"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	factionType := body.FactionType
	if factionType == "" {
		factionType = "faction"
	}
	influence := body.Influence
	if influence < 1 {
		influence = 5
	}
	color := body.Color
	if color == "" {
		color = "#c9a84c"
	}

	id, err := s.db.CreateFaction(campaignID, body.Name, body.Description, factionType, influence, body.ResourcesJSON, color)
	if err != nil {
		serverError(w, r, err)
		return
	}

	s.bus.Publish(Event{Type: EventFactionUpdated, Payload: &FactionUpdatedPayload{CampaignID: RealtimeInt64(campaignID)}})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id}) //nolint:errcheck
}

func (s *Server) handleListFactions(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	campaignID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	factions, err := s.db.ListFactions(campaignID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if factions == nil {
		factions = []db.Faction{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(factions) //nolint:errcheck
}

func (s *Server) handleGetFaction(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	f, err := s.db.GetFaction(id)
	if err != nil {
		http.Error(w, "faction not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(f) //nolint:errcheck
}

func (s *Server) handleUpdateFaction(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body struct {
		Name          string `json:"name"`
		Description   string `json:"description"`
		FactionType   string `json:"faction_type"`
		Influence     int    `json:"influence"`
		ResourcesJSON string `json:"resources_json"`
		Color         string `json:"color"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	factionType := body.FactionType
	if factionType == "" {
		factionType = "faction"
	}
	influence := body.Influence
	if influence < 1 {
		influence = 5
	}
	color := body.Color
	if color == "" {
		color = "#c9a84c"
	}
	faction, err := s.db.GetFaction(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if faction == nil {
		http.Error(w, "faction not found", http.StatusNotFound)
		return
	}
	if err := s.db.UpdateFaction(id, body.Name, body.Description, factionType, influence, body.ResourcesJSON, color); err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventFactionUpdated, Payload: &FactionUpdatedPayload{CampaignID: RealtimeInt64(faction.CampaignID), ID: RealtimeInt64(id)}})
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDeleteFaction(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	faction, err := s.db.GetFaction(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if faction == nil {
		http.Error(w, "faction not found", http.StatusNotFound)
		return
	}
	if err := s.db.DeleteFaction(id); err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventFactionUpdated, Payload: &FactionUpdatedPayload{CampaignID: RealtimeInt64(faction.CampaignID), ID: RealtimeInt64(id)}})
	w.WriteHeader(http.StatusNoContent)
}
