package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/digitalghost404/inkandbone/internal/db"
	"github.com/stretchr/testify/require"
)

func TestAutomationTurnChangesHTTPAndActorScope(t *testing.T) {
	stub := &stubCompleter{}
	s := newTestServerWithAI(t, stub)
	campaign, session := seedCampaign(t, s.db)
	actor, err := s.db.CreateCharacter(campaign, "Sol")
	require.NoError(t, err)
	other, err := s.db.CreateCharacter(campaign, "Luna")
	require.NoError(t, err)
	require.NoError(t, s.db.SetSetting("active_character_id", fmt.Sprint(other)))
	require.NoError(t, s.db.UpdateCharacterData(actor, `{"hp":10}`))
	msg, err := s.db.CreateMessage(session, "assistant", "Ada pays you five gold.", false, nil)
	require.NoError(t, err)
	ctx := withTurnMessage(t.Context(), msg, actor)
	stub.response = `{"add":[{"name":"Ada","note":"Pilot"}],"remove":[]}`
	s.extractNPCs(ctx, session, "Ada signs the contract.")
	stub.response = `{"new":[{"title":"Escort Ada","description":"Walk to the gate"}]}`
	s.autoDetectObjectives(ctx, session, "Ada gives you a quest.")
	stub.response = `{"gained":[{"name":"Brass key","quantity":1}]}`
	s.autoExtractItems(ctx, session, "Ada hands you a key; you keep it.")
	stub.response = `{"delta":5}`
	s.autoUpdateCurrency(ctx, session, "Ada pays you 5 gold.")
	s.autoUpdateCurrency(ctx, session, "Ada pays you 5 gold.")
	stub.response = `{"hp":8}`
	s.autoUpdateCharacterStats(ctx, session, "I take damage.", "You take two damage.")
	path := fmt.Sprintf("/api/sessions/%d/changes", session)
	call := func(method, url string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest(method, url, nil))
		return w
	}
	resp := call("GET", path)
	require.Equal(t, 200, resp.Code)
	var changes []db.TurnChange
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &changes))
	require.Len(t, changes, 5)
	ids := map[string]int64{}
	for _, c := range changes {
		require.Equal(t, msg, c.MessageID)
		ids[c.Kind] = c.ID
		require.True(t, c.CanUndo)
	}
	require.Len(t, ids, 5)
	correct := func(kind string) *httptest.ResponseRecorder {
		return call("POST", fmt.Sprintf("%s/%d/undo", path, ids[kind]))
	}
	require.Equal(t, http.StatusNotFound, call("POST", fmt.Sprintf("/api/sessions/%d/changes/%d/undo", session+1, ids["currency"])).Code)
	require.Equal(t, http.StatusOK, correct("currency").Code)
	require.Equal(t, http.StatusOK, correct("currency").Code)
	require.NoError(t, s.db.UpdateCharacterData(actor, `{"hp":7}`))
	require.Equal(t, http.StatusConflict, correct("stat").Code)
	persisted, err := s.db.GetCharacter(actor)
	require.NoError(t, err)
	require.Zero(t, persisted.CurrencyBalance)
	require.JSONEq(t, `{"hp":7}`, persisted.DataJSON)
	items, err := s.db.ListItems(actor)
	require.NoError(t, err)
	require.Len(t, items, 1)
	items, err = s.db.ListItems(other)
	require.NoError(t, err)
	require.Empty(t, items)
}

func TestStreamedReplyAttachesChangeOnceAcrossRetry(t *testing.T) {
	s := newTestServerWithAI(t, &stubCompleterStreamer{generateResp: `{"delta":6}`, streamResp: "The courier gives you 6 gold."})
	campaign, session := seedCampaign(t, s.db)
	actor, err := s.db.CreateCharacter(campaign, "Courier")
	require.NoError(t, err)
	for _, cfg := range AllAutomationSettings() {
		if cfg.Key != settingAutoUpdateCurrency {
			require.NoError(t, s.db.SetSetting(cfg.Key, "0"))
		}
	}
	action, err := s.db.CreateMessage(session, "user", "I claim the payment.", false, &actor)
	require.NoError(t, err)
	run := func() {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("POST", fmt.Sprintf("/api/sessions/%d/gm-respond-stream?action_id=%d", session, action), nil))
		require.Equal(t, 200, w.Code)
		require.Contains(t, w.Body.String(), `"type":"done"`)
	}
	run()
	require.Eventually(t, func() bool { rows, err := s.db.ListTurnChanges(session); return err == nil && len(rows) == 1 }, 2*time.Second, 10*time.Millisecond)
	messages, err := s.db.ListMessages(session)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	rows, err := s.db.ListTurnChanges(session)
	require.NoError(t, err)
	require.Equal(t, messages[1].ID, rows[0].MessageID)
	run()
	rows, err = s.db.ListTurnChanges(session)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	ch, err := s.db.GetCharacter(actor)
	require.NoError(t, err)
	require.EqualValues(t, 6, ch.CurrencyBalance)
}
