package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleImprovise_ok verifies 200 + {"result": "..."} when AI is available.
func TestHandleImprovise_ok(t *testing.T) {
	stub := &stubCompleter{response: "A mysterious figure enters the tavern."}
	s := newTestServerWithAI(t, stub)
	campID, sessID := seedCampaign(t, s.db)
	_ = campID

	req := httptest.NewRequest(http.MethodPost,
		"/api/sessions/"+strconv.FormatInt(sessID, 10)+"/improvise",
		nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Result string `json:"result"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "A mysterious figure enters the tavern.", resp.Result)
}

// TestHandleImprovise_noAI verifies 503 when aiClient is nil.
func TestHandleImprovise_noAI(t *testing.T) {
	s := newTestServer(t) // aiClient is nil
	_, sessID := seedCampaign(t, s.db)

	req := httptest.NewRequest(http.MethodPost,
		"/api/sessions/"+strconv.FormatInt(sessID, 10)+"/improvise",
		nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// TestHandleImprovise_invalidID verifies 400 on bad session ID.
func TestHandleImprovise_invalidID(t *testing.T) {
	stub := &stubCompleter{response: "something"}
	s := newTestServerWithAI(t, stub)

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/abc/improvise", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandlePreSessionBrief_ok verifies 200 + {"result": "..."} when AI is available.
func TestHandlePreSessionBrief_ok(t *testing.T) {
	stub := &stubCompleter{response: "• Last session the party found the map."}
	s := newTestServerWithAI(t, stub)
	campID, _ := seedCampaign(t, s.db)
	_, err := s.db.CreateWorldNote(campID, "Ancient Ruins", "The site of an old battle.", "location")
	require.NoError(t, err)
	_, err = s.db.CreateObjective(campID, "Find the artifact", "main quest", nil)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost,
		"/api/campaigns/"+strconv.FormatInt(campID, 10)+"/pre-session-brief",
		nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Result string `json:"result"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "• Last session the party found the map.", resp.Result)
}

// TestHandlePreSessionBrief_noAI verifies 503 when aiClient is nil.
func TestHandlePreSessionBrief_noAI(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)

	req := httptest.NewRequest(http.MethodPost,
		"/api/campaigns/"+strconv.FormatInt(campID, 10)+"/pre-session-brief",
		nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// TestHandlePreSessionBrief_invalidID verifies 400 on bad campaign ID.
func TestHandlePreSessionBrief_invalidID(t *testing.T) {
	stub := &stubCompleter{response: "something"}
	s := newTestServerWithAI(t, stub)

	req := httptest.NewRequest(http.MethodPost, "/api/campaigns/xyz/pre-session-brief", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandleDetectThreads_ok verifies 200 + {"result": "..."} when AI is available.
func TestHandleDetectThreads_ok(t *testing.T) {
	stub := &stubCompleter{response: "1. The stolen relic has not been recovered.\n2. The innkeeper's secret remains unknown."}
	s := newTestServerWithAI(t, stub)
	_, sessID := seedCampaign(t, s.db)
	_, err := s.db.CreateMessage(sessID, "user", "We investigate the burned-down library.", false, nil)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost,
		"/api/sessions/"+strconv.FormatInt(sessID, 10)+"/detect-threads",
		nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Result string `json:"result"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp.Result, "relic")
}

// TestHandleDetectThreads_noAI verifies 503 when aiClient is nil.
func TestHandleDetectThreads_noAI(t *testing.T) {
	s := newTestServer(t)
	_, sessID := seedCampaign(t, s.db)

	req := httptest.NewRequest(http.MethodPost,
		"/api/sessions/"+strconv.FormatInt(sessID, 10)+"/detect-threads",
		nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// TestHandleDetectThreads_invalidID verifies 400 on bad session ID.
func TestHandleDetectThreads_invalidID(t *testing.T) {
	stub := &stubCompleter{response: "something"}
	s := newTestServerWithAI(t, stub)

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/bad/detect-threads", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandleCampaignAsk_ok verifies 200 + {"result": "..."} when AI is available.
func TestHandleCampaignAsk_ok(t *testing.T) {
	stub := &stubCompleter{response: "The dragon lives in the northern mountains."}
	s := newTestServerWithAI(t, stub)
	campID, _ := seedCampaign(t, s.db)
	_, err := s.db.CreateWorldNote(campID, "Ignarok", "An ancient red dragon.", "npc")
	require.NoError(t, err)

	body := `{"question":"Where does the dragon live?"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/campaigns/"+strconv.FormatInt(campID, 10)+"/ask",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Result string `json:"result"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "The dragon lives in the northern mountains.", resp.Result)
}

// TestHandleCampaignAsk_noAI verifies 503 when aiClient is nil.
func TestHandleCampaignAsk_noAI(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)

	body := `{"question":"test?"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/campaigns/"+strconv.FormatInt(campID, 10)+"/ask",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// TestHandleCampaignAsk_noQuestion verifies 400 when question is missing.
func TestHandleCampaignAsk_noQuestion(t *testing.T) {
	stub := &stubCompleter{response: "answer"}
	s := newTestServerWithAI(t, stub)
	campID, _ := seedCampaign(t, s.db)

	req := httptest.NewRequest(http.MethodPost,
		"/api/campaigns/"+strconv.FormatInt(campID, 10)+"/ask",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandleCampaignAsk_invalidID verifies 400 on bad campaign ID.
func TestHandleCampaignAsk_invalidID(t *testing.T) {
	stub := &stubCompleter{response: "answer"}
	s := newTestServerWithAI(t, stub)

	req := httptest.NewRequest(http.MethodPost,
		"/api/campaigns/nope/ask",
		strings.NewReader(`{"question":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
