package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/digitalghost404/inkandbone/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCalendar_Defaults(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/campaigns/%d/calendar", campaignID), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var info struct {
		InGameYear     int    `json:"in_game_year"`
		InGameMonth    int    `json:"in_game_month"`
		InGameDay      int    `json:"in_game_day"`
		CalendarConfig string `json:"calendar_config"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&info))
	assert.Equal(t, 1, info.InGameYear)
	assert.Equal(t, 1, info.InGameMonth)
	assert.Equal(t, 1, info.InGameDay)
	assert.Equal(t, "{}", info.CalendarConfig)
}

func TestPatchCalendar(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	body := `{"in_game_year":1247,"in_game_month":3,"in_game_day":15,"calendar_config":"{\"months\":12}"}`
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/campaigns/%d/calendar", campaignID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, float64(1247), resp["in_game_year"])
	assert.Equal(t, float64(3), resp["in_game_month"])
	assert.Equal(t, float64(15), resp["in_game_day"])
}

func TestPatchCalendar_AdvanceDays(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	body := `{"advance_days":5}`
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/campaigns/%d/calendar", campaignID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, float64(1), resp["in_game_year"])
	assert.Equal(t, float64(1), resp["in_game_month"])
	assert.Equal(t, float64(6), resp["in_game_day"])
}

func TestPatchCalendar_AdvanceToNextMonth(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	body := `{"advance_days":30}`
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/campaigns/%d/calendar", campaignID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, float64(1), resp["in_game_year"])
	assert.Equal(t, float64(2), resp["in_game_month"])
	assert.Equal(t, float64(1), resp["in_game_day"])
}

func TestPatchCalendar_AdvanceToNextYear(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	body := `{"advance_days":365}`
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/campaigns/%d/calendar", campaignID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, float64(2), resp["in_game_year"])
	assert.Equal(t, float64(1), resp["in_game_month"])
	assert.Equal(t, float64(6), resp["in_game_day"])
}

func TestCreateCalendarEvent(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	body := `{"in_game_year":1247,"in_game_month":3,"in_game_day":15,"title":"Battle of Iron Pass","description":"A decisive battle","event_type":"battle"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/campaigns/%d/calendar-events", campaignID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	eventID := int64(created["id"].(float64))

	e, err := s.db.GetCalendarEvent(eventID)
	require.NoError(t, err)
	assert.Equal(t, "Battle of Iron Pass", e.Title)
	assert.Equal(t, 1247, e.InGameYear)
	assert.Equal(t, 3, e.InGameMonth)
	assert.Equal(t, 15, e.InGameDay)
	assert.Equal(t, "battle", e.EventType)
}

func TestCreateCalendarEvent_requiresTitle(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/campaigns/%d/calendar-events", campaignID), strings.NewReader(`{"description":"no title"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListCalendarEvents(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	_, err := s.db.CreateCalendarEvent(campaignID, 1, 1, 1, "Event 1", "", "note", nil)
	require.NoError(t, err)
	_, err = s.db.CreateCalendarEvent(campaignID, 1, 1, 2, "Event 2", "", "note", nil)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/campaigns/%d/calendar-events", campaignID), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var events []db.CalendarEvent
	require.NoError(t, json.NewDecoder(w.Body).Decode(&events))
	require.Len(t, events, 2)
}

func TestListCalendarEvents_empty(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/campaigns/%d/calendar-events", campaignID), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var events []db.CalendarEvent
	require.NoError(t, json.NewDecoder(w.Body).Decode(&events))
	assert.Empty(t, events)
}

func TestDeleteCalendarEvent(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	eid, err := s.db.CreateCalendarEvent(campaignID, 1, 1, 1, "To Delete", "", "note", nil)
	require.NoError(t, err)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/calendar-events/%d", eid), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	_, err = s.db.GetCalendarEvent(eid)
	require.Error(t, err)
}

func TestPatchCalendar_InvalidCampaign(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest("PATCH", "/api/campaigns/99999/calendar", strings.NewReader(`{"in_game_year":2}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPatchCalendar_InvalidBody(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/campaigns/%d/calendar", campaignID), strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
