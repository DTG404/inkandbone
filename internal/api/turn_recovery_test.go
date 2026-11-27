package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/digitalghost404/inkandbone/internal/ai"
	"github.com/digitalghost404/inkandbone/internal/db"
	"github.com/stretchr/testify/require"
)

func TestTurnReplayKeepsOneActionRollAndReply(t *testing.T) {
	s := newTestServerWithAI(t, &stubCompleterStreamer{generateResp: `{"required":true,"expression":"1d6","attribute":"Strength"}`, streamResp: "The door opens."})
	_, sid := seedCampaign(t, s.db)
	for _, setting := range AllAutomationSettings() {
		require.NoError(t, s.db.SetSetting(setting.Key, "0"))
	}
	require.NoError(t, s.db.SetSetting(settingAutoCheckRoll, "1"))
	post := func(path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/sessions/%d/", sid)+path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		return w
	}
	for range 2 {
		w := post("messages", `{"role":"user","content":"Open the door","client_action_key":"test-action-123"}`)
		require.Contains(t, []int{200, 201}, w.Code)
	}
	msgs, err := s.db.ListMessages(sid)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	target := fmt.Sprintf("gm-respond-stream?action_id=%d", msgs[0].ID)
	_, err = s.db.SQL().Exec("CREATE TRIGGER fail_reply BEFORE INSERT ON messages WHEN NEW.role='assistant' BEGIN SELECT RAISE(ABORT, 'fixture'); END")
	require.NoError(t, err)
	require.Contains(t, post(target, "").Body.String(), "persistence_failed")
	_, err = s.db.SQL().Exec("DROP TRIGGER fail_reply")
	require.NoError(t, err)
	for range 2 {
		require.Contains(t, post(target, "").Body.String(), `"type":"done"`)
	}
	msgs, err = s.db.ListMessages(sid)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	rolls, err := s.db.ListDiceRolls(sid)
	require.NoError(t, err)
	require.Len(t, rolls, 1)
}

func TestTurnCitationsSnapshotOnlyPromptExcerpts(t *testing.T) {
	stub := &stubCompleterStreamer{generateResp: `{"required":false}`, streamResp: "I invented a citation to Book X."}
	s := newTestServerWithAI(t, stub)
	cid, sid := seedCampaign(t, s.db)
	campaign, err := s.db.GetCampaign(cid)
	require.NoError(t, err)
	for _, setting := range AllAutomationSettings() {
		require.NoError(t, s.db.SetSetting(setting.Key, "0"))
	}
	text := "vaulting keyword " + strings.Repeat("棚", 500)
	require.NoError(t, s.db.CreateRulebookChunks(campaign.RulesetID, []db.RulebookChunk{{Source: "Guide", Heading: "Vaulting", Content: text}}))
	aid, err := s.db.CreateMessage(sid, "user", "vaulting", false, nil)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/sessions/%d/gm-respond-stream?action_id=%d", sid, aid), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.Contains(t, w.Body.String(), `"type":"done"`)
	require.NoError(t, s.db.DeleteRulebookChunksBySource(campaign.RulesetID, "Guide"))
	messages, err := s.db.ListMessages(sid)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Len(t, messages[1].RuleCitations, 1)
	c := messages[1].RuleCitations[0]
	require.Equal(t, "Guide", c.Source)
	require.Equal(t, "Vaulting", c.Heading)
	require.NotEqual(t, text, c.Passage)
	require.Contains(t, stub.capturedSys, c.Passage)
}

func TestTurnClaimFencesConcurrentAndExpiredAttempts(t *testing.T) {
	s := newTestServer(t)
	_, sid := seedCampaign(t, s.db)
	aid, _, err := s.db.SubmitAction(sid, "Listen", false, nil, "uuid", true)
	require.NoError(t, err)
	_, err = s.db.ClaimTurn(sid, aid, "worker-1")
	require.NoError(t, err)
	_, err = s.db.ClaimTurn(sid, aid, "worker-2")
	require.ErrorIs(t, err, db.ErrTurnConflict)
	_, _, err = s.db.SubmitAction(sid, "Run", false, nil, "another", true)
	require.ErrorIs(t, err, db.ErrTurnConflict)
	_, err = s.db.SQL().Exec("UPDATE gm_turns SET lease_until=0 WHERE action_message_id=?", aid)
	require.NoError(t, err)
	_, err = s.db.ClaimTurn(sid, aid, "worker-2")
	require.NoError(t, err)
	_, err = s.db.FinishTurn(sid, aid, "worker-1", "late stale answer", nil)
	require.ErrorIs(t, err, db.ErrTurnConflict)
	_, err = s.db.FinishTurn(sid, aid, "worker-2", "recovered answer", nil)
	require.NoError(t, err)
	ms, err := s.db.ListMessages(sid)
	require.NoError(t, err)
	require.Len(t, ms, 2)
	require.Equal(t, "recovered answer", ms[1].Content)
}

type heldTurnStream struct {
	*stubCompleterStreamer
	started chan struct{}
	resume  chan struct{}
	once    sync.Once
}

func (stub *heldTurnStream) StreamRespond(ctx context.Context, prompt string, history []ai.ChatMessage, maxTokens int, w http.ResponseWriter) (string, error) {
	stub.once.Do(func() { close(stub.started) })
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-stub.resume:
	}
	return stub.stubCompleterStreamer.StreamRespond(ctx, prompt, history, maxTokens, w)
}
func TestConcurrentTurnHTTPRequestsDoNotDoubleRollOrReply(t *testing.T) {
	held := &heldTurnStream{stubCompleterStreamer: &stubCompleterStreamer{generateResp: `{"required":true,"expression":"1d6"}`, streamResp: "One reply."}, started: make(chan struct{}), resume: make(chan struct{})}
	s := newTestServerWithAI(t, held)
	_, sid := seedCampaign(t, s.db)
	for _, setting := range AllAutomationSettings() {
		require.NoError(t, s.db.SetSetting(setting.Key, "0"))
	}
	require.NoError(t, s.db.SetSetting(settingAutoCheckRoll, "1"))
	aid, _, err := s.db.SubmitAction(sid, "Race the clock", false, nil, "same", true)
	require.NoError(t, err)
	url := fmt.Sprintf("/api/sessions/%d/gm-respond-stream?action_id=%d", sid, aid)
	call := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest(http.MethodPost, url, nil))
		return w
	}
	finished := make(chan *httptest.ResponseRecorder, 1)
	go func() { finished <- call() }()
	select {
	case <-held.started:
	case <-time.After(5 * time.Second):
		close(held.resume)
		t.Fatal("provider never started")
	}
	conflict := call()
	close(held.resume)
	require.Equal(t, 409, conflict.Code)
	require.Contains(t, (<-finished).Body.String(), `"type":"done"`)
	require.Contains(t, call().Body.String(), "One reply.")
	msgs, err := s.db.ListMessages(sid)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	rolls, err := s.db.ListDiceRolls(sid)
	require.NoError(t, err)
	require.Len(t, rolls, 1)
}

func TestSavedActionKeepsOriginalActorWhenTableSwitchesCharacter(t *testing.T) {
	stub := &stubCompleterStreamer{generateResp: `{"required":false}`, streamResp: "The Beast stirs."}
	s := newTestServerWithAI(t, stub)
	cid, sid := seedCampaign(t, s.db)
	_, err := s.db.SQL().Exec("UPDATE campaigns SET ruleset_id=(SELECT id FROM rulesets WHERE name='vtm') WHERE id=?", cid)
	require.NoError(t, err)
	for _, setting := range AllAutomationSettings() {
		require.NoError(t, s.db.SetSetting(setting.Key, "0"))
	}
	a, err := s.db.CreateCharacter(cid, "Kestrel")
	require.NoError(t, err)
	b, err := s.db.CreateCharacter(cid, "Crow")
	require.NoError(t, err)
	require.NoError(t, s.db.UpdateCharacterData(a, `{"hunger":1,"blood_potency":1}`))
	require.NoError(t, s.db.UpdateCharacterData(b, `{"hunger":3,"blood_potency":8}`))
	require.NoError(t, s.db.SetSetting("active_character_id", strconv.FormatInt(a, 10)))
	aid, _, err := s.db.SubmitAction(sid, "/surge", false, nil, "surge-actor", true)
	require.NoError(t, err)
	require.NoError(t, s.db.SetSetting("active_character_id", strconv.FormatInt(b, 10)))
	same, created, err := s.db.SubmitAction(sid, "/surge", false, nil, "surge-actor", true)
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, aid, same)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/sessions/%d/gm-respond-stream?action_id=%d", sid, aid), nil))
	require.Contains(t, w.Body.String(), `"type":"done"`)
	require.Contains(t, stub.capturedSys, "Blood Potency 1")
	require.NotContains(t, stub.capturedSys, "Blood Potency 8")
	other, err := s.db.GetCharacter(b)
	require.NoError(t, err)
	require.JSONEq(t, `{"hunger":3,"blood_potency":8}`, other.DataJSON)
	rolls, err := s.db.ListDiceRolls(sid)
	require.NoError(t, err)
	require.Len(t, rolls, 1)
}
