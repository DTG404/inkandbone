package api

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouteContractMessageAndWorldReads(t *testing.T) {
	s := newTestServer(t)
	campaignID, sessionID := seedCampaign(t, s.db)
	for _, test := range []struct {
		name string
		path string
	}{
		{"messages", fmt.Sprintf("/api/sessions/%d/messages", sessionID)},
		{"maps", fmt.Sprintf("/api/campaigns/%d/maps", campaignID)},
		{"world notes", fmt.Sprintf("/api/campaigns/%d/world-notes", campaignID)},
		{"npcs", fmt.Sprintf("/api/sessions/%d/npcs", sessionID)},
		{"objectives", fmt.Sprintf("/api/campaigns/%d/objectives", campaignID)},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			s.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			assert.Equal(t, http.StatusOK, response.Code)
			assert.Equal(t, "application/json", response.Header().Get("Content-Type"))
			assert.True(t, strings.HasPrefix(strings.TrimSpace(response.Body.String()), "["))
		})
	}
}

func TestRouteContractCreateMessagePublishesGeneratedEvent(t *testing.T) {
	s := newTestServer(t)
	_, sessionID := seedCampaign(t, s.db)
	events := s.bus.SubscribeContext(t.Context())
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/sessions/%d/messages", sessionID), strings.NewReader(`{"role":"user","content":"hello"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	s.ServeHTTP(response, request)
	require.Equal(t, http.StatusCreated, response.Code)
	event := <-events
	require.Equal(t, EventMessageCreated, event.Type)
	require.IsType(t, &MessageCreatedPayload{}, event.Payload)
}

func TestPromptGoldenCoreDirectives(t *testing.T) {
	assert.Equal(t, "c18fb36f92eee5d5c7786195c219b77ece95cc7358a55ae872aa3aaefa8e348f", fmt.Sprintf("%x", sha256.Sum256([]byte(gmSystemPrompt))))
	assert.Equal(t, "8aefb56d58e9bd67636764476b3f978a57c20ada491931a5bb81269421a2f13b", fmt.Sprintf("%x", sha256.Sum256([]byte(mapSystemPrompt))))
	assert.Equal(t, "The Gate", func() string { title, _ := parseGeneratedNote("Title: The Gate\nContent: Old stone."); return title }())
}

func TestPromptGoldenEmptyWorldContext(t *testing.T) {
	s := newTestServer(t)
	_, sessionID := seedCampaign(t, s.db)
	world := s.buildWorldContext(t.Context(), sessionID)
	assert.Equal(t, "caf421376668c51d0f865ea5d8105a097e58b81502f501ea5b0ad0da9193a1a6", fmt.Sprintf("%x", sha256.Sum256([]byte(world))))
}

func TestAutomationJobSceneTags(t *testing.T) {
	s := newTestServer(t)
	_, sessionID := seedCampaign(t, s.db)
	s.autoUpdateSceneTags(t.Context(), sessionID, "The party enters a torchlit dungeon corridor.")
	session, err := s.db.GetSession(sessionID)
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Contains(t, session.SceneTags, "dungeon")
}
