package api

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func scopedEvent(t *testing.T, events <-chan Event, eventType EventType, scope string, id int64) {
	t.Helper()
	event := <-events
	require.Equal(t, eventType, event.Type)
	payload, ok := event.Payload.(map[string]any)
	require.True(t, ok)
	require.Equal(t, id, payload[scope])
}

func handlerRequest(t *testing.T, method, body, id string) (*httptest.ResponseRecorder, *http.Request) {
	t.Helper()
	req := httptest.NewRequest(method, "/", bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.SetPathValue("id", id)
	return httptest.NewRecorder(), req
}

func TestPanelMutationEventsIncludeOwningScope(t *testing.T) {
	s := newTestServer(t)
	campaignID, sessionID := seedCampaign(t, s.db)
	events := s.bus.SubscribeContext(t.Context())

	noteID, err := s.db.CreateWorldNote(campaignID, "Note", "Text", "other")
	require.NoError(t, err)
	w, req := handlerRequest(t, http.MethodPatch, `{"title":"Changed","content":"Text"}`, fmt.Sprint(noteID))
	s.handlePatchWorldNote(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
	scopedEvent(t, events, EventWorldNoteUpdated, "campaign_id", campaignID)

	npc, err := s.db.CreateSessionNPC(sessionID, "Keeper", "")
	require.NoError(t, err)
	w, req = handlerRequest(t, http.MethodPatch, `{"note":"changed"}`, fmt.Sprint(npc.ID))
	s.handlePatchNPC(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
	scopedEvent(t, events, EventNPCUpdated, "session_id", sessionID)

	objective, err := s.db.CreateObjective(campaignID, "Goal", "", nil)
	require.NoError(t, err)
	w, req = handlerRequest(t, http.MethodPatch, `{"status":"completed"}`, fmt.Sprint(objective.ID))
	s.handlePatchObjective(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
	scopedEvent(t, events, EventObjectiveUpdated, "campaign_id", campaignID)

	adventureID, err := s.db.CreateAdventure(campaignID, "Arc", "", "active", 0)
	require.NoError(t, err)
	w, req = handlerRequest(t, http.MethodPut, `{"title":"Changed","status":"active"}`, fmt.Sprint(adventureID))
	s.handleUpdateAdventure(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	scopedEvent(t, events, EventAdventureUpdated, "campaign_id", campaignID)

	secretID, err := s.db.CreateSecret(campaignID, "Secret", "Text", "secret")
	require.NoError(t, err)
	w, req = handlerRequest(t, http.MethodPut, `{"title":"Changed","content":"Text","category":"secret"}`, fmt.Sprint(secretID))
	s.handleUpdateSecret(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	scopedEvent(t, events, EventSecretsUpdated, "campaign_id", campaignID)

	relationshipID, err := s.db.CreateRelationship(campaignID, "A", "B", "ally", "")
	require.NoError(t, err)
	w, req = handlerRequest(t, http.MethodPatch, `{"relationship_type":"rival"}`, fmt.Sprint(relationshipID))
	s.handleUpdateRelationship(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	scopedEvent(t, events, EventRelationshipUpdated, "campaign_id", campaignID)

	factionID, err := s.db.CreateFaction(campaignID, "Guild", "", "guild", 5, "{}", "#fff")
	require.NoError(t, err)
	w, req = handlerRequest(t, http.MethodPut, `{"name":"Guild","faction_type":"guild","influence":6,"resources_json":"{}","color":"#fff"}`, fmt.Sprint(factionID))
	s.handleUpdateFaction(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	scopedEvent(t, events, EventFactionUpdated, "campaign_id", campaignID)

	npcStatID, err := s.db.CreateNpcStat(campaignID, "Guard", "brute", "{}", 10, nil, 0, "", "", "", "")
	require.NoError(t, err)
	w, req = handlerRequest(t, http.MethodPut, `{"name":"Guard","role":"brute","data_json":"{}","hp_max":12}`, fmt.Sprint(npcStatID))
	s.handleUpdateNpcStat(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	scopedEvent(t, events, EventNpcStatUpdated, "campaign_id", campaignID)
}

func TestPanelDeleteEventsIncludeOwningScope(t *testing.T) {
	s := newTestServer(t)
	campaignID, sessionID := seedCampaign(t, s.db)
	events := s.bus.SubscribeContext(t.Context())

	npc, err := s.db.CreateSessionNPC(sessionID, "Keeper", "")
	require.NoError(t, err)
	w, req := handlerRequest(t, http.MethodDelete, "", fmt.Sprint(npc.ID))
	s.handleDeleteNPC(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
	scopedEvent(t, events, EventNPCUpdated, "session_id", sessionID)

	objective, err := s.db.CreateObjective(campaignID, "Goal", "", nil)
	require.NoError(t, err)
	w, req = handlerRequest(t, http.MethodDelete, "", fmt.Sprint(objective.ID))
	s.handleDeleteObjective(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
	scopedEvent(t, events, EventObjectiveUpdated, "campaign_id", campaignID)

	adventureID, err := s.db.CreateAdventure(campaignID, "Arc", "", "active", 0)
	require.NoError(t, err)
	w, req = handlerRequest(t, http.MethodDelete, "", fmt.Sprint(adventureID))
	s.handleDeleteAdventure(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
	scopedEvent(t, events, EventAdventureUpdated, "campaign_id", campaignID)

	secretID, err := s.db.CreateSecret(campaignID, "Secret", "Text", "secret")
	require.NoError(t, err)
	w, req = handlerRequest(t, http.MethodDelete, "", fmt.Sprint(secretID))
	s.handleDeleteSecret(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
	scopedEvent(t, events, EventSecretsUpdated, "campaign_id", campaignID)

	relationshipID, err := s.db.CreateRelationship(campaignID, "A", "B", "ally", "")
	require.NoError(t, err)
	w, req = handlerRequest(t, http.MethodDelete, "", fmt.Sprint(relationshipID))
	s.handleDeleteRelationship(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
	scopedEvent(t, events, EventRelationshipUpdated, "campaign_id", campaignID)

	factionID, err := s.db.CreateFaction(campaignID, "Guild", "", "guild", 5, "{}", "#fff")
	require.NoError(t, err)
	w, req = handlerRequest(t, http.MethodDelete, "", fmt.Sprint(factionID))
	s.handleDeleteFaction(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
	scopedEvent(t, events, EventFactionUpdated, "campaign_id", campaignID)

	npcStatID, err := s.db.CreateNpcStat(campaignID, "Guard", "brute", "{}", 10, nil, 0, "", "", "", "")
	require.NoError(t, err)
	w, req = handlerRequest(t, http.MethodDelete, "", fmt.Sprint(npcStatID))
	s.handleDeleteNpcStat(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
	scopedEvent(t, events, EventNpcStatUpdated, "campaign_id", campaignID)
}

func TestSecretRevealEventsIncludeCampaignAndSessionScope(t *testing.T) {
	s := newTestServer(t)
	campaignID, sessionID := seedCampaign(t, s.db)
	events := s.bus.SubscribeContext(t.Context())
	secretID, err := s.db.CreateSecret(campaignID, "Secret", "Text", "secret")
	require.NoError(t, err)

	w, req := handlerRequest(t, http.MethodPost, fmt.Sprintf(`{"session_id":%d}`, sessionID), fmt.Sprint(secretID))
	s.handleRevealSecret(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	event := <-events
	require.Equal(t, EventSecretRevealed, event.Type)
	payload := event.Payload.(map[string]any)
	require.Equal(t, campaignID, payload["campaign_id"])
	require.Equal(t, sessionID, payload["session_id"])
	scopedEvent(t, events, EventSecretsUpdated, "campaign_id", campaignID)
}
