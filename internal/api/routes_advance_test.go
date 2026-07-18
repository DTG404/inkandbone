package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoSuggestXPSpend_noopForCoC(t *testing.T) {
	s := newTestServer(t)

	rs, err := s.db.GetRulesetByName("coc")
	require.NoError(t, err)
	require.NotNil(t, rs)

	campID, err := s.db.CreateCampaign(rs.ID, "CoC Campaign", "")
	require.NoError(t, err)
	charID, err := s.db.CreateCharacter(campID, "Investigator")
	require.NoError(t, err)
	_ = s.db.UpdateCharacterData(charID, `{"xp":50}`)

	char, err := s.db.GetCharacter(charID)
	require.NoError(t, err)

	ch := s.bus.SubscribeContext(t.Context())

	// Should no-op immediately for CoC.
	go s.autoSuggestXPSpend(context.Background(), 1, charID, char, rs, map[string]any{"xp": float64(50)}, 50)

	// Wait briefly — no xp_spend_suggestions event should arrive.
	select {
	case ev := <-ch:
		assert.NotEqual(t, EventXPSpendSuggestions, ev.Type,
			"CoC should not emit xp_spend_suggestions")
	case <-time.After(200 * time.Millisecond):
		// correct: nothing emitted
	}
}

func TestAdvancementConfig(t *testing.T) {
	s := newTestServer(t)
	for _, test := range []struct {
		name    string
		minimum int
	}{
		{"vtm", 3},
		{"cyberpunk", 10},
		{"wrath_glory", 4},
	} {
		t.Run(test.name, func(t *testing.T) {
			ruleset, err := s.db.GetRulesetByName(test.name)
			require.NoError(t, err)
			require.NotNil(t, ruleset)

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/rulesets/%d/advancement-config", ruleset.ID), nil)
			w := httptest.NewRecorder()
			s.ServeHTTP(w, req)
			require.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			var response struct {
				MinimumXP int  `json:"minimum_xp"`
				Supported bool `json:"supported"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			assert.Equal(t, test.minimum, response.MinimumXP)
			assert.True(t, response.Supported)
		})
	}
}

func TestAutoSuggestXPSpend_sessionCap(t *testing.T) {
	s := newTestServer(t)
	const sessionID = int64(42)

	// Pre-fill the cap.
	s.xpSuggestCounts.Store(sessionID, 20)

	rs, err := s.db.GetRulesetByName("wrath_glory")
	require.NoError(t, err)
	require.NotNil(t, rs)

	campID, err := s.db.CreateCampaign(rs.ID, "WG Campaign", "")
	require.NoError(t, err)
	charID, err := s.db.CreateCharacter(campID, "Brother Cato")
	require.NoError(t, err)
	_ = s.db.UpdateCharacterData(charID, `{"xp":50}`)

	char, err := s.db.GetCharacter(charID)
	require.NoError(t, err)

	ch := s.bus.SubscribeContext(t.Context())

	// Session cap reached — should no-op.
	go s.autoSuggestXPSpend(context.Background(), sessionID, charID, char, rs, map[string]any{"xp": float64(50)}, 50)

	select {
	case ev := <-ch:
		assert.NotEqual(t, EventXPSpendSuggestions, ev.Type,
			"capped session should not emit xp_spend_suggestions")
	case <-time.After(200 * time.Millisecond):
		// correct: nothing emitted
	}
}

func TestHandleAdvanceCharacter_wgAttribute(t *testing.T) {
	s := newTestServer(t)

	rs, err := s.db.GetRulesetByName("wrath_glory")
	require.NoError(t, err)
	require.NotNil(t, rs)
	campID, err := s.db.CreateCampaign(rs.ID, "WG Campaign", "")
	require.NoError(t, err)
	charID, err := s.db.CreateCharacter(campID, "Brother Cato")
	require.NoError(t, err)

	statsJSON := `{"archetype":"Imperial Guardsman","tier":1,"strength":2,"agility":2,"toughness":4,"intellect":2,"willpower":2,"fellowship":2,"initiative":2,"ws":0,"bs":0,"athletics":0,"awareness":0,"cunning":0,"deception":0,"fortitude":0,"insight":0,"intimidation":0,"investigation":0,"leadership":0,"medicae":0,"persuasion":0,"pilot":0,"psychic_mastery":0,"scholar":0,"stealth":0,"survival":0,"tech":0,"xp":20,"wounds":9,"resilience":5,"determination":4,"shock":3,"resolve":1,"conviction":2,"influence":1,"defence":1,"talents":""}`
	require.NoError(t, s.db.UpdateCharacterData(charID, statsJSON))

	ch := s.bus.SubscribeContext(t.Context())

	body := `{"field":"toughness","new_value":5}`
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/characters/%d/advance", charID),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify XP deducted (20 - 20 = 0; new toughness=5, cost=5*4=20)
	char, err := s.db.GetCharacter(charID)
	require.NoError(t, err)
	var stats map[string]any
	require.NoError(t, json.Unmarshal([]byte(char.DataJSON), &stats))
	assert.Equal(t, float64(0), stats["xp"])
	assert.Equal(t, float64(5), stats["toughness"])

	// Derived stats recalculated: wounds = (tier=1)*2 + 5 = 7, resilience = 6, determination = 5
	assert.Equal(t, float64(7), stats["wounds"])
	assert.Equal(t, float64(6), stats["resilience"])
	assert.Equal(t, float64(5), stats["determination"])

	// Verify character_updated event published
	var got Event
	select {
	case got = <-ch:
	default:
		t.Fatal("expected character_updated event")
	}
	assert.Equal(t, EventCharacterUpdated, got.Type)
}

func TestHandleAdvanceCharacter_wgTalent(t *testing.T) {
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("wrath_glory")
	require.NoError(t, err)
	require.NotNil(t, rs)
	campID, err := s.db.CreateCampaign(rs.ID, "WG Campaign", "")
	require.NoError(t, err)
	charID, err := s.db.CreateCharacter(campID, "Kael")
	require.NoError(t, err)

	statsJSON := `{"archetype":"Imperial Guardsman","strength":2,"agility":2,"toughness":2,"intellect":2,"willpower":2,"fellowship":2,"initiative":2,"ws":0,"bs":0,"athletics":0,"awareness":0,"cunning":0,"deception":0,"fortitude":0,"insight":0,"intimidation":0,"investigation":0,"leadership":0,"medicae":0,"persuasion":0,"pilot":0,"psychic_mastery":0,"scholar":0,"stealth":0,"survival":0,"tech":0,"xp":20,"talents":"Look Out, Sir!"}`
	require.NoError(t, s.db.UpdateCharacterData(charID, statsJSON))

	body := `{"field":"talent:Iron Will","new_value":1}`
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/characters/%d/advance", charID),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	char, err := s.db.GetCharacter(charID)
	require.NoError(t, err)
	var stats map[string]any
	require.NoError(t, json.Unmarshal([]byte(char.DataJSON), &stats))
	assert.Equal(t, float64(0), stats["xp"])
	talents, _ := stats["talents"].(string)
	assert.Contains(t, talents, "Iron Will")
}

func TestHandleAdvanceCharacter_wgTalent_alreadyOwned(t *testing.T) {
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("wrath_glory")
	require.NoError(t, err)
	require.NotNil(t, rs)
	campID, err := s.db.CreateCampaign(rs.ID, "WG Campaign", "")
	require.NoError(t, err)
	charID, err := s.db.CreateCharacter(campID, "Kael")
	require.NoError(t, err)

	statsJSON := `{"archetype":"Imperial Guardsman","xp":20,"talents":"Iron Will|Look Out, Sir!"}`
	require.NoError(t, s.db.UpdateCharacterData(charID, statsJSON))

	body := `{"field":"talent:Iron Will","new_value":1}`
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/characters/%d/advance", charID),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAdvanceCharacter_wgTalent_archetypeAbility(t *testing.T) {
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("wrath_glory")
	require.NoError(t, err)
	require.NotNil(t, rs)
	campID, err := s.db.CreateCampaign(rs.ID, "WG Campaign", "")
	require.NoError(t, err)
	charID, err := s.db.CreateCharacter(campID, "Kael")
	require.NoError(t, err)

	statsJSON := `{"archetype":"Imperial Guardsman","xp":50,"talents":"Look Out, Sir!"}`
	require.NoError(t, s.db.UpdateCharacterData(charID, statsJSON))

	// "Look Out, Sir!" is the archetype ability — cannot purchase even if XP present
	body := `{"field":"talent:Look Out, Sir!","new_value":1}`
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/characters/%d/advance", charID),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAdvanceCharacter_notEnoughXP(t *testing.T) {
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("wrath_glory")
	require.NoError(t, err)
	require.NotNil(t, rs)
	campID, err := s.db.CreateCampaign(rs.ID, "WG Campaign", "")
	require.NoError(t, err)
	charID, err := s.db.CreateCharacter(campID, "Kael")
	require.NoError(t, err)

	statsJSON := `{"archetype":"Imperial Guardsman","toughness":4,"xp":5}`
	require.NoError(t, s.db.UpdateCharacterData(charID, statsJSON))

	body := `{"field":"toughness","new_value":5}` // costs 20 XP, has only 5
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/characters/%d/advance", charID),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAdvanceCharacter_skipRank(t *testing.T) {
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("wrath_glory")
	require.NoError(t, err)
	require.NotNil(t, rs)
	campID, err := s.db.CreateCampaign(rs.ID, "WG Campaign", "")
	require.NoError(t, err)
	charID, err := s.db.CreateCharacter(campID, "Kael")
	require.NoError(t, err)

	statsJSON := `{"archetype":"Imperial Guardsman","toughness":3,"xp":100}`
	require.NoError(t, s.db.UpdateCharacterData(charID, statsJSON))

	// Trying to jump from 3 to 5 (skipping 4)
	body := `{"field":"toughness","new_value":5}`
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/characters/%d/advance", charID),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAdvanceCharacter_bladesXPReset(t *testing.T) {
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("blades")
	require.NoError(t, err)
	require.NotNil(t, rs)
	campID, err := s.db.CreateCampaign(rs.ID, "Blades Campaign", "")
	require.NoError(t, err)
	charID, err := s.db.CreateCharacter(campID, "Swick")
	require.NoError(t, err)

	statsJSON := `{"xp":10,"action:Hunt":0}`
	require.NoError(t, s.db.UpdateCharacterData(charID, statsJSON))

	body := `{"field":"action:Hunt","new_value":1}`
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/characters/%d/advance", charID),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	char, err := s.db.GetCharacter(charID)
	require.NoError(t, err)
	var stats map[string]any
	require.NoError(t, json.Unmarshal([]byte(char.DataJSON), &stats))
	// Blades XP resets to 0, not subtracts 8
	assert.Equal(t, float64(0), stats["xp"])
}

func TestHandleAdvanceCharacter_dnd5eLevel(t *testing.T) {
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("dnd5e")
	require.NoError(t, err)
	require.NotNil(t, rs)
	campID, err := s.db.CreateCampaign(rs.ID, "DnD Campaign", "")
	require.NoError(t, err)
	charID, err := s.db.CreateCharacter(campID, "Aria")
	require.NoError(t, err)

	// Level 1, 300 XP (threshold for level 2)
	statsJSON := `{"level":1,"hp":10,"xp":300,"proficiency_bonus":2}`
	require.NoError(t, s.db.UpdateCharacterData(charID, statsJSON))

	body := `{"field":"level","new_value":2}`
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/characters/%d/advance", charID),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	char, err := s.db.GetCharacter(charID)
	require.NoError(t, err)
	var stats map[string]any
	require.NoError(t, json.Unmarshal([]byte(char.DataJSON), &stats))
	assert.Equal(t, float64(2), stats["level"])
	assert.Equal(t, float64(15), stats["hp"])               // +5
	assert.Equal(t, float64(300), stats["xp"])              // XP NOT subtracted for dnd5e
	assert.Equal(t, float64(2), stats["proficiency_bonus"]) // floor((2-1)/4)+2 = 2
}

func TestHandleAdvanceCharacter_notFound(t *testing.T) {
	s := newTestServer(t)
	body := `{"field":"toughness","new_value":3}`
	req := httptest.NewRequest(http.MethodPost, "/api/characters/9999/advance",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAdvanceCharacter_invalidField(t *testing.T) {
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("wrath_glory")
	require.NoError(t, err)
	require.NotNil(t, rs)
	campID, err := s.db.CreateCampaign(rs.ID, "WG Campaign", "")
	require.NoError(t, err)
	charID, err := s.db.CreateCharacter(campID, "Brother Cato")
	require.NoError(t, err)

	// Give plenty of XP so the rejection is definitely about field validity, not XP.
	statsJSON := `{"archetype":"Imperial Guardsman","xp":100}`
	require.NoError(t, s.db.UpdateCharacterData(charID, statsJSON))

	// "xp" is not an advanceable field — it should be rejected with 400.
	body := `{"field":"xp","new_value":11}`
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/characters/%d/advance", charID),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAdvanceCharacter_bladesNotEnoughXP(t *testing.T) {
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("blades")
	require.NoError(t, err)
	campID, err := s.db.CreateCampaign(rs.ID, "Blades Campaign", "")
	require.NoError(t, err)
	charID, err := s.db.CreateCharacter(campID, "Nix")
	require.NoError(t, err)

	// 5 XP — less than the required 8 for a Blades advance
	statsJSON := `{"xp":5,"action:Hunt":0}`
	require.NoError(t, s.db.UpdateCharacterData(charID, statsJSON))

	body := `{"field":"action:Hunt","new_value":1}`
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/characters/%d/advance", charID),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
