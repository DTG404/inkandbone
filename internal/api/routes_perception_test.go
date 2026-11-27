package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCharacterReferenceUsesOnlyWitnessedGMNarration(t *testing.T) {
	stub := &stubCompleter{response: `{"answer":"Two bells ring.","location":"Beyond the hills.","decision":"Two bells ring.","leads":"imaginary knight"}`}
	s := newTestServerWithAI(t, stub)
	campaign, session := seedCampaign(t, s.db)
	a, err := s.db.CreateCharacter(campaign, "Aster")
	require.NoError(t, err)
	b, err := s.db.CreateCharacter(campaign, "Briar")
	require.NoError(t, err)
	add := func(role, text string, whisper bool, actor *int64) {
		_, err := s.db.CreateMessage(session, role, text, whisper, actor)
		require.NoError(t, err)
	}
	add("assistant", "UNKNOWN_WITNESS", false, nil)
	add("user", "PLAYER_GUESS", false, &a)
	add("assistant", "Beyond the hills. Two bells ring.", false, nil)
	add("assistant", "PRIVATE_WHISPER", true, &a)
	add("user", "listen", false, &b)
	add("assistant", "ANOTHER_CHARACTER_SECRET", false, nil)
	before, err := s.db.ListMessages(session)
	require.NoError(t, err)
	for _, endpoint := range []string{"clarify", "player-brief"} {
		path := fmt.Sprintf("/api/sessions/%d/%s", session, endpoint)
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(fmt.Sprintf(`{"character_id":%d,"question":"What can I hear?"}`, a)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
		require.Equal(t, 200, w.Code, w.Body.String())
		require.Contains(t, w.Body.String(), "Two bells ring.")
		require.NotContains(t, w.Body.String(), "imaginary knight")
	}
	prompt := stub.capturedPrompts()
	for _, private := range []string{"UNKNOWN_WITNESS", "PLAYER_GUESS", "PRIVATE_WHISPER", "ANOTHER_CHARACTER_SECRET"} {
		require.NotContains(t, prompt, private)
	}
	after, err := s.db.ListMessages(session)
	require.NoError(t, err)
	require.Equal(t, before, after)
}
