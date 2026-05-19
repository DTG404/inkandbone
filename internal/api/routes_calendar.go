package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/digitalghost404/inkandbone/internal/db"
)

func (s *Server) handleGetCalendar(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	campaignID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	info, err := s.db.GetCampaignDate(campaignID)
	if err != nil {
		http.Error(w, "campaign not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info) //nolint:errcheck
}

func (s *Server) handlePatchCalendar(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	campaignID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	var body struct {
		InGameYear    *int    `json:"in_game_year"`
		InGameMonth   *int    `json:"in_game_month"`
		InGameDay     *int    `json:"in_game_day"`
		CalendarConfig *string `json:"calendar_config"`
		AdvanceDays   *int    `json:"advance_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	info, err := s.db.GetCampaignDate(campaignID)
	if err != nil {
		http.Error(w, "campaign not found", http.StatusNotFound)
		return
	}

	year := info.InGameYear
	month := info.InGameMonth
	day := info.InGameDay
	cfg := info.CalendarConfig

	if body.AdvanceDays != nil && *body.AdvanceDays > 0 {
		days := *body.AdvanceDays
		day += days
		for day > 30 {
			day -= 30
			month++
			if month > 12 {
				month = 1
				year++
			}
		}
	}
	if body.InGameYear != nil {
		year = *body.InGameYear
	}
	if body.InGameMonth != nil {
		month = *body.InGameMonth
	}
	if body.InGameDay != nil {
		day = *body.InGameDay
	}
	if body.CalendarConfig != nil {
		cfg = *body.CalendarConfig
	}

	if err := s.db.UpdateCampaignDate(campaignID, year, month, day, &cfg); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	s.bus.Publish(Event{Type: EventCalendarUpdated, Payload: map[string]any{"campaign_id": campaignID}})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"in_game_year":   year,
		"in_game_month":  month,
		"in_game_day":    day,
		"calendar_config": cfg,
	}) //nolint:errcheck
}

func (s *Server) handleCreateCalendarEvent(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	campaignID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	var body struct {
		InGameYear  int    `json:"in_game_year"`
		InGameMonth int    `json:"in_game_month"`
		InGameDay   int    `json:"in_game_day"`
		Title       string `json:"title"`
		Description string `json:"description"`
		EventType   string `json:"event_type"`
		SessionID   *int64 `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Title == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}
	eventType := body.EventType
	if eventType == "" {
		eventType = "note"
	}
	id, err := s.db.CreateCalendarEvent(campaignID, body.InGameYear, body.InGameMonth, body.InGameDay, body.Title, body.Description, eventType, body.SessionID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.bus.Publish(Event{Type: EventCalendarUpdated, Payload: map[string]any{"campaign_id": campaignID}})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id}) //nolint:errcheck
}

func (s *Server) handleListCalendarEvents(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	campaignID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	events, err := s.db.ListCalendarEvents(campaignID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if events == nil {
		events = []db.CalendarEvent{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events) //nolint:errcheck
}

func (s *Server) handleDeleteCalendarEvent(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.db.DeleteCalendarEvent(id); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
