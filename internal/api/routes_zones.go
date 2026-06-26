package api

import (
	"net/http"

	"github.com/digitalghost404/inkandbone/internal/db"
)

func (s *Server) handleListZones(w http.ResponseWriter, r *http.Request) {
	mapID, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid map id", http.StatusBadRequest)
		return
	}
	zones, err := s.db.ListMapZones(mapID)
	if err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if zones == nil {
		zones = []db.MapZone{}
	}
	writeJSON(w, zones)
}

func (s *Server) handleCreateZone(w http.ResponseWriter, r *http.Request) {
	mapID, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid map id", http.StatusBadRequest)
		return
	}
	var body struct {
		Name   string  `json:"name"`
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
		Width  float64 `json:"width"`
		Height float64 `json:"height"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if body.X < 0 || body.X > 1 || body.Y < 0 || body.Y > 1 || body.Width <= 0 || body.Height <= 0 {
		http.Error(w, "x/y must be 0–1, width/height must be positive", http.StatusBadRequest)
		return
	}
	id, err := s.db.CreateMapZone(mapID, body.Name, body.X, body.Y, body.Width, body.Height)
	if err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]any{"id": id})
}

func (s *Server) handlePatchZone(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid zone id", http.StatusBadRequest)
		return
	}
	var body struct {
		Name       *string  `json:"name"`
		X          *float64 `json:"x"`
		Y          *float64 `json:"y"`
		Width      *float64 `json:"width"`
		Height     *float64 `json:"height"`
		IsRevealed *bool    `json:"is_revealed"`
	}
	if err := decodeJSON(r, &body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	existing, err := s.db.GetMapZone(id)
	if err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.Error(w, "zone not found", http.StatusNotFound)
		return
	}
	name := existing.Name
	x, y, w2, h := existing.X, existing.Y, existing.Width, existing.Height
	if body.Name != nil {
		name = *body.Name
	}
	if body.X != nil {
		x = *body.X
	}
	if body.Y != nil {
		y = *body.Y
	}
	if body.Width != nil {
		w2 = *body.Width
	}
	if body.Height != nil {
		h = *body.Height
	}
	if x < 0 || x > 1 || y < 0 || y > 1 || w2 <= 0 || h <= 0 {
		http.Error(w, "x/y must be 0–1, width/height must be positive", http.StatusBadRequest)
		return
	}
	if err := s.db.UpdateMapZone(id, name, x, y, w2, h); err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if body.IsRevealed != nil {
		if err := s.db.RevealZone(id, *body.IsRevealed); err != nil {
			http.Error(w, "db reveal: "+err.Error(), http.StatusInternalServerError)
			return
		}
		s.bus.Publish(Event{Type: EventZoneRevealed, Payload: map[string]any{
			"map_id":      existing.MapID,
			"zone_id":     id,
			"zone_name":   name,
			"is_revealed": *body.IsRevealed,
		}})
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteZone(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid zone id", http.StatusBadRequest)
		return
	}
	if err := s.db.DeleteMapZone(id); err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
