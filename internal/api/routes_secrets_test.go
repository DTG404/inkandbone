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

func TestSecretCRUD_API(t *testing.T) {
	s := newTestServer(t)
	campaignID, sessionID := seedCampaign(t, s.db)

	body := `{"title":"The Lost Vault","content":"Hidden beneath the old temple.","category":"secret"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/campaigns/%d/secrets", campaignID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	secretID := int64(created["id"].(float64))

	req = httptest.NewRequest("GET", fmt.Sprintf("/api/campaigns/%d/secrets", campaignID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var list []db.Secret
	require.NoError(t, json.NewDecoder(w.Body).Decode(&list))
	require.Len(t, list, 1)
	assert.Equal(t, "The Lost Vault", list[0].Title)
	assert.False(t, list[0].Revealed)

	req = httptest.NewRequest("GET", fmt.Sprintf("/api/secrets/%d", secretID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var single db.Secret
	require.NoError(t, json.NewDecoder(w.Body).Decode(&single))
	assert.Equal(t, "The Lost Vault", single.Title)
	assert.Equal(t, "secret", single.Category)

	revealBody := fmt.Sprintf(`{"session_id":%d}`, sessionID)
	req = httptest.NewRequest("PATCH", fmt.Sprintf("/api/secrets/%d/reveal", secretID), strings.NewReader(revealBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest("GET", fmt.Sprintf("/api/secrets/%d", secretID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.NoError(t, json.NewDecoder(w.Body).Decode(&single))
	assert.True(t, single.Revealed)

	updateBody := `{"title":"The Hidden Vault","content":"Moved.","category":"clue"}`
	req = httptest.NewRequest("PATCH", fmt.Sprintf("/api/secrets/%d", secretID), strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest("GET", fmt.Sprintf("/api/secrets/%d", secretID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.NoError(t, json.NewDecoder(w.Body).Decode(&single))
	assert.Equal(t, "The Hidden Vault", single.Title)
	assert.Equal(t, "clue", single.Category)

	req = httptest.NewRequest("DELETE", fmt.Sprintf("/api/secrets/%d", secretID), nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	secrets, _ := s.db.ListSecretsByCampaign(campaignID)
	assert.Len(t, secrets, 0)
}

func TestCreateSecret_requiresTitle(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/campaigns/%d/secrets", campaignID), strings.NewReader(`{"content":"no title"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetSecret_notFound(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/secrets/99999", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRevealSecret_requiresSessionID(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	id, err := s.db.CreateSecret(campaignID, "Test", "Content", "secret")
	require.NoError(t, err)

	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/secrets/%d/reveal", id), strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSecret_requiresTitle(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	id, err := s.db.CreateSecret(campaignID, "Test", "Content", "secret")
	require.NoError(t, err)

	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/secrets/%d", id), strings.NewReader(`{"title":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateSecret_defaultsCategory(t *testing.T) {
	s := newTestServer(t)
	campaignID, _ := seedCampaign(t, s.db)

	body := `{"title":"Mystery"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/campaigns/%d/secrets", campaignID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	secretID := int64(created["id"].(float64))

	secret, err := s.db.GetSecret(secretID)
	require.NoError(t, err)
	assert.Equal(t, "secret", secret.Category)
}

func TestRevealSecretPublishesHandoutEvent(t *testing.T) {
	s := newTestServer(t)
	campID, sessID := seedCampaign(t, s.db)

	// Create a secret
	secretID, err := s.db.CreateSecret(campID, "Lost Map", "A map to the dungeon", "handout")
	require.NoError(t, err)

	ch := s.bus.SubscribeContext(t.Context())

	body := fmt.Sprintf(`{"session_id":%d}`, sessID)
	req := httptest.NewRequest(http.MethodPatch,
		fmt.Sprintf("/api/secrets/%d/reveal", secretID),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Drain channel looking for secret_revealed event
	var foundReveal bool
	for len(ch) > 0 {
		ev := <-ch
		if ev.Type == EventSecretRevealed {
			foundReveal = true
			payload := ev.Payload.(map[string]any)
			assert.Equal(t, "Lost Map", payload["title"])
			assert.Equal(t, "A map to the dungeon", payload["content"])
		}
	}
	assert.True(t, foundReveal, "expected secret_revealed event")
}
