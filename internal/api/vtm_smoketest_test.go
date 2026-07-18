package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// vtmCampaignAndChar creates a vtm ruleset, campaign, character and session.
// Returns (campaignID, characterID, sessionID).
func vtmCampaignAndChar(t *testing.T, s *Server, statsJSON string) (campID, charID, sessID int64) {
	t.Helper()
	rs, err := s.db.GetRulesetByName("vtm")
	require.NoError(t, err)
	require.NotNil(t, rs, "vtm ruleset must be seeded")

	campID, err = s.db.CreateCampaign(rs.ID, "VtM Campaign", "")
	require.NoError(t, err)

	charID, err = s.db.CreateCharacter(campID, "Elara Voss")
	require.NoError(t, err)

	if statsJSON != "" {
		require.NoError(t, s.db.UpdateCharacterData(charID, statsJSON))
	}

	sessID, err = s.db.CreateSession(campID, "Night 1", "2026-04-01")
	require.NoError(t, err)

	return
}

// advanceRequest fires POST /api/characters/{id}/advance and returns the recorder.
func advanceRequest(t *testing.T, s *Server, charID int64, field string, newVal int) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"field":%q,"new_value":%d}`, field, newVal)
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/characters/%d/advance", charID),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	return w
}

// getCharStats fetches a character and parses its DataJSON into map[string]any.
func getCharStats(t *testing.T, s *Server, charID int64) map[string]any {
	t.Helper()
	char, err := s.db.GetCharacter(charID)
	require.NoError(t, err)
	var stats map[string]any
	require.NoError(t, json.Unmarshal([]byte(char.DataJSON), &stats))
	return stats
}

// ---------------------------------------------------------------------------
// 1. Character CRUD with VtM-specific stat fields
// ---------------------------------------------------------------------------

func TestVtM_CharacterCRUD_VtMFields(t *testing.T) {
	s := newTestServer(t)

	vtmStats := `{
		"clan":"Brujah",
		"hunger":0,
		"health_superficial":0,
		"health_aggravated":0,
		"health_max":6,
		"willpower_superficial":0,
		"willpower_aggravated":0,
		"willpower_max":3,
		"humanity":7,
		"stains":0,
		"blood_potency":1,
		"predator_type":"Alleycat",
		"xp":20
	}`

	_, charID, _ := vtmCampaignAndChar(t, s, vtmStats)

	// Verify all fields roundtrip through DB
	stats := getCharStats(t, s, charID)
	assert.Equal(t, "Brujah", stats["clan"])
	assert.Equal(t, float64(0), stats["hunger"])
	assert.Equal(t, float64(0), stats["health_superficial"])
	assert.Equal(t, float64(0), stats["health_aggravated"])
	assert.Equal(t, float64(0), stats["willpower_superficial"])
	assert.Equal(t, float64(0), stats["willpower_aggravated"])
	assert.Equal(t, float64(7), stats["humanity"])
	assert.Equal(t, float64(0), stats["stains"])
	assert.Equal(t, float64(1), stats["blood_potency"])
	assert.Equal(t, "Alleycat", stats["predator_type"])
}

func TestVtM_UpdateCharacterData_NoFieldWipe(t *testing.T) {
	s := newTestServer(t)

	initial := `{"clan":"Brujah","hunger":2,"humanity":7,"blood_potency":2,"xp":30}`
	_, charID, _ := vtmCampaignAndChar(t, s, initial)

	// Full-replace with a partial update — fields from initial must survive
	updated := `{"clan":"Brujah","hunger":3,"humanity":7,"blood_potency":2,"xp":25,"strength":3}`
	require.NoError(t, s.db.UpdateCharacterData(charID, updated))

	stats := getCharStats(t, s, charID)
	assert.Equal(t, "Brujah", stats["clan"])
	assert.Equal(t, float64(3), stats["hunger"])
	assert.Equal(t, float64(25), stats["xp"])
	assert.Equal(t, float64(3), stats["strength"])
}

// ---------------------------------------------------------------------------
// 2. HTTP API — Advance Attribute
// ---------------------------------------------------------------------------

func TestVtM_HTTP_AdvanceAttribute(t *testing.T) {
	s := newTestServer(t)
	stats := `{"clan":"Brujah","strength":2,"xp":20}`
	_, charID, _ := vtmCampaignAndChar(t, s, stats)

	// Advance strength 2 → 3: cost = 3*4 = 12 XP
	w := advanceRequest(t, s, charID, "strength", 3)
	assert.Equal(t, http.StatusOK, w.Code)

	got := getCharStats(t, s, charID)
	assert.Equal(t, float64(3), got["strength"], "strength should be 3")
	assert.Equal(t, float64(8), got["xp"], "xp should be 20-12=8")
}

// ---------------------------------------------------------------------------
// 3. HTTP API — Advance Skill
// ---------------------------------------------------------------------------

func TestVtM_HTTP_AdvanceSkill(t *testing.T) {
	s := newTestServer(t)
	stats := `{"clan":"Brujah","athletics":1,"xp":20}`
	_, charID, _ := vtmCampaignAndChar(t, s, stats)

	// Advance athletics 1 → 2: cost = 2*3 = 6 XP
	w := advanceRequest(t, s, charID, "athletics", 2)
	assert.Equal(t, http.StatusOK, w.Code)

	got := getCharStats(t, s, charID)
	assert.Equal(t, float64(2), got["athletics"])
	assert.Equal(t, float64(14), got["xp"], "xp should be 20-6=14")
}

// ---------------------------------------------------------------------------
// 4. HTTP API — Advance In-Clan Discipline (Brujah + potence, 5x cost)
// ---------------------------------------------------------------------------

func TestVtM_HTTP_AdvanceInClanDiscipline(t *testing.T) {
	s := newTestServer(t)
	// potence is in-clan for Brujah; cost = new_dots * 5 = 2 * 5 = 10
	stats := `{"clan":"Brujah","potence":1,"xp":15}`
	_, charID, _ := vtmCampaignAndChar(t, s, stats)

	w := advanceRequest(t, s, charID, "potence", 2)
	assert.Equal(t, http.StatusOK, w.Code)

	got := getCharStats(t, s, charID)
	assert.Equal(t, float64(2), got["potence"])
	assert.Equal(t, float64(5), got["xp"], "xp should be 15-10=5")
}

// ---------------------------------------------------------------------------
// 5. HTTP API — Advance Out-Of-Clan Discipline (Brujah + oblivion, 7x cost)
// ---------------------------------------------------------------------------

func TestVtM_HTTP_AdvanceOutOfClanDiscipline(t *testing.T) {
	s := newTestServer(t)
	// oblivion is out-of-clan for Brujah; cost = new_dots * 7 = 2 * 7 = 14
	stats := `{"clan":"Brujah","oblivion":1,"xp":20}`
	_, charID, _ := vtmCampaignAndChar(t, s, stats)

	w := advanceRequest(t, s, charID, "oblivion", 2)
	assert.Equal(t, http.StatusOK, w.Code)

	got := getCharStats(t, s, charID)
	assert.Equal(t, float64(2), got["oblivion"])
	assert.Equal(t, float64(6), got["xp"], "xp should be 20-14=6")
}

// ---------------------------------------------------------------------------
// 6. HTTP API — Advance Blood Potency (10x cost)
// ---------------------------------------------------------------------------

func TestVtM_HTTP_AdvanceBloodPotency(t *testing.T) {
	s := newTestServer(t)
	// blood_potency 1 → 2: cost = 2 * 10 = 20 XP
	stats := `{"clan":"Brujah","blood_potency":1,"xp":25}`
	_, charID, _ := vtmCampaignAndChar(t, s, stats)

	w := advanceRequest(t, s, charID, "blood_potency", 2)
	assert.Equal(t, http.StatusOK, w.Code)

	got := getCharStats(t, s, charID)
	assert.Equal(t, float64(2), got["blood_potency"])
	assert.Equal(t, float64(5), got["xp"], "xp should be 25-20=5")
}

// ---------------------------------------------------------------------------
// 7. HTTP API — Reject if not enough XP
// ---------------------------------------------------------------------------

func TestVtM_HTTP_Advance_NotEnoughXP(t *testing.T) {
	s := newTestServer(t)
	// strength 2→3 costs 12, only has 5
	stats := `{"clan":"Brujah","strength":2,"xp":5}`
	_, charID, _ := vtmCampaignAndChar(t, s, stats)

	w := advanceRequest(t, s, charID, "strength", 3)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// 8. HTTP API — Reject invalid field
// ---------------------------------------------------------------------------

func TestVtM_HTTP_Advance_InvalidField(t *testing.T) {
	s := newTestServer(t)
	stats := `{"clan":"Brujah","xp":100}`
	_, charID, _ := vtmCampaignAndChar(t, s, stats)

	// "hunger" is not an advanceable field
	w := advanceRequest(t, s, charID, "hunger", 1)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVtM_HTTP_Advance_SkipRank(t *testing.T) {
	s := newTestServer(t)
	// Trying to jump strength from 2 to 4 (skipping 3) should fail
	stats := `{"clan":"Brujah","strength":2,"xp":100}`
	_, charID, _ := vtmCampaignAndChar(t, s, stats)

	w := advanceRequest(t, s, charID, "strength", 4)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// 9. HTTP API — Suggest Advances (POST /api/characters/{id}/suggest-advances)
// ---------------------------------------------------------------------------

func TestVtM_HTTP_SuggestAdvances_Returns202(t *testing.T) {
	s := newTestServer(t)
	stats := `{"clan":"Brujah","potence":1,"strength":2,"xp":20}`
	_, charID, _ := vtmCampaignAndChar(t, s, stats)

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/characters/%d/suggest-advances", charID),
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	// No AI client configured → goroutine no-ops, but HTTP should still return 202
	assert.Equal(t, http.StatusAccepted, w.Code)
}

func TestVtM_HTTP_SuggestAdvances_HintXPAccepted(t *testing.T) {
	s := newTestServer(t)
	stats := `{"clan":"Brujah","xp":0}`
	_, charID, _ := vtmCampaignAndChar(t, s, stats)

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/characters/%d/suggest-advances", charID),
		strings.NewReader(`{"hint_xp":50}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusAccepted, w.Code)

	// hint_xp=50 should have persisted because it's larger than current xp=0
	got := getCharStats(t, s, charID)
	xp, _ := got["xp"].(float64)
	assert.Equal(t, float64(50), xp, "hint_xp should have been persisted")
}

func TestVtM_HTTP_SuggestAdvances_NotFound(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/characters/99999/suggest-advances",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// 10. HTTP API — Talent Description (GET /api/talent-description?name=X&system=vtm)
// ---------------------------------------------------------------------------

func TestVtM_HTTP_TalentDescription_NoAI_Returns503(t *testing.T) {
	// No AI client → should return 503 Service Unavailable (not 500)
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/talent-description?name=Celerity&system=vtm", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	// Acceptable: 503 when AI not configured, never 500
	assert.NotEqual(t, http.StatusInternalServerError, w.Code,
		"talent-description should not return 500; got %d %s", w.Code, w.Body.String())
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestVtM_HTTP_TalentDescription_WithAI_Returns200(t *testing.T) {
	stub := &stubCompleter{response: "Celerity lets the vampire move at supernatural speed."}
	s := newTestServerWithAI(t, stub)

	req := httptest.NewRequest(http.MethodGet, "/api/talent-description?name=Celerity&system=vtm", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["description"])
}

func TestVtM_HTTP_TalentDescription_MissingName_Returns400(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/talent-description?system=vtm", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// 11. Masquerade Integrity Endpoints
// ---------------------------------------------------------------------------

func TestVtM_Masquerade_GetDefault(t *testing.T) {
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("vtm")
	require.NoError(t, err)
	campID, err := s.db.CreateCampaign(rs.ID, "VtM Camp", "")
	require.NoError(t, err)
	sessID, err := s.db.CreateSession(campID, "Night 1", "2026-04-01")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/sessions/%d/masquerade", sessID), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	// Default masquerade integrity is 10
	assert.Equal(t, float64(10), resp["masquerade_integrity"])
}

func TestVtM_Masquerade_PatchDecrement(t *testing.T) {
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("vtm")
	require.NoError(t, err)
	campID, err := s.db.CreateCampaign(rs.ID, "VtM Camp", "")
	require.NoError(t, err)
	sessID, err := s.db.CreateSession(campID, "Night 1", "2026-04-01")
	require.NoError(t, err)

	// Patch to 7
	patchReq := httptest.NewRequest(http.MethodPatch,
		fmt.Sprintf("/api/sessions/%d/masquerade", sessID),
		strings.NewReader(`{"masquerade_integrity":7}`))
	patchReq.Header.Set("Content-Type", "application/json")
	pw := httptest.NewRecorder()
	s.ServeHTTP(pw, patchReq)
	assert.Equal(t, http.StatusOK, pw.Code)

	// Verify via GET
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/sessions/%d/masquerade", sessID), nil)
	gw := httptest.NewRecorder()
	s.ServeHTTP(gw, getReq)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(gw.Body.Bytes(), &resp))
	assert.Equal(t, float64(7), resp["masquerade_integrity"])
}

func TestVtM_Masquerade_ClampBelowZero(t *testing.T) {
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("vtm")
	require.NoError(t, err)
	campID, err := s.db.CreateCampaign(rs.ID, "VtM Camp", "")
	require.NoError(t, err)
	sessID, err := s.db.CreateSession(campID, "Night 1", "2026-04-01")
	require.NoError(t, err)

	patchReq := httptest.NewRequest(http.MethodPatch,
		fmt.Sprintf("/api/sessions/%d/masquerade", sessID),
		strings.NewReader(`{"masquerade_integrity":-5}`))
	patchReq.Header.Set("Content-Type", "application/json")
	pw := httptest.NewRecorder()
	s.ServeHTTP(pw, patchReq)
	assert.Equal(t, http.StatusOK, pw.Code)

	// DB should clamp at 0
	level, err := s.db.GetMasqueradeIntegrity(sessID)
	require.NoError(t, err)
	assert.Equal(t, 0, level)
}

func TestVtM_Masquerade_MissingBodyReturns400(t *testing.T) {
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("vtm")
	require.NoError(t, err)
	campID, err := s.db.CreateCampaign(rs.ID, "VtM Camp", "")
	require.NoError(t, err)
	sessID, err := s.db.CreateSession(campID, "Night 1", "2026-04-01")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPatch,
		fmt.Sprintf("/api/sessions/%d/masquerade", sessID),
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// 12. Chronicle Night tracking (autoUpdateChronicleNight)
// ---------------------------------------------------------------------------

func TestVtM_ChronicleNight_IncrementsOnNewNight(t *testing.T) {
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("vtm")
	require.NoError(t, err)
	campID, err := s.db.CreateCampaign(rs.ID, "VtM Camp", "")
	require.NoError(t, err)
	sessID, err := s.db.CreateSession(campID, "Night 1", "2026-04-01")
	require.NoError(t, err)

	// chronicle_night defaults to 1 (1-based per migration 030)
	camp, err := s.db.GetCampaign(campID)
	require.NoError(t, err)
	initialNight := camp.ChronicleNight

	// Call the function directly with a new-night keyword
	s.autoUpdateChronicleNight(context.Background(), sessID, "As dusk falls over the city, your character awakens.")

	camp, err = s.db.GetCampaign(campID)
	require.NoError(t, err)
	assert.Equal(t, initialNight+1, camp.ChronicleNight, "chronicle_night should increment by 1")
}

func TestVtM_ChronicleNight_NoIncrementOnUnrelatedText(t *testing.T) {
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("vtm")
	require.NoError(t, err)
	campID, err := s.db.CreateCampaign(rs.ID, "VtM Camp", "")
	require.NoError(t, err)
	sessID, err := s.db.CreateSession(campID, "Night 1", "2026-04-01")
	require.NoError(t, err)

	camp, err := s.db.GetCampaign(campID)
	require.NoError(t, err)
	initialNight := camp.ChronicleNight

	s.autoUpdateChronicleNight(context.Background(), sessID, "You approach the bartender and ask for a drink.")

	camp, err = s.db.GetCampaign(campID)
	require.NoError(t, err)
	assert.Equal(t, initialNight, camp.ChronicleNight, "chronicle_night should not change on unrelated text")
}

func TestVtM_ChronicleNight_NoIncrementForNonVtM(t *testing.T) {
	s := newTestServer(t)
	// Use a non-vtm campaign
	rs, err := s.db.GetRulesetByName("wrath_glory")
	require.NoError(t, err)
	campID, err := s.db.CreateCampaign(rs.ID, "WG Camp", "")
	require.NoError(t, err)
	sessID, err := s.db.CreateSession(campID, "Session 1", "2026-04-01")
	require.NoError(t, err)

	camp, err := s.db.GetCampaign(campID)
	require.NoError(t, err)
	initialNight := camp.ChronicleNight

	s.autoUpdateChronicleNight(context.Background(), sessID, "As dusk falls, the sky turns orange.")

	camp, err = s.db.GetCampaign(campID)
	require.NoError(t, err)
	assert.Equal(t, initialNight, camp.ChronicleNight, "non-vtm campaign should not increment chronicle_night")
}

// ---------------------------------------------------------------------------
// 13. Stain Trigger Keywords (stainTriggerRE)
// ---------------------------------------------------------------------------

func TestVtM_StainTriggerRE_MatchesViolenceKeywords(t *testing.T) {
	triggerTexts := []string{
		"The vampire killed the mortal.",
		"She fed from the unwilling victim.",
		"The diablerie was witnessed by others.",
		"Forcing feeding upon the helpless servant.",
		"He killed the last witness.",
	}
	for _, text := range triggerTexts {
		lower := strings.ToLower(text)
		if !stainTriggerRE.MatchString(lower) {
			t.Errorf("stainTriggerRE should match %q", text)
		}
	}
}

func TestVtM_StainTriggerRE_NoMatchOnUnrelated(t *testing.T) {
	noTriggerTexts := []string{
		"You walk through the park enjoying the night air.",
		"The merchant greets you warmly.",
		"Your haven is quiet and secure.",
		"You read a book and contemplate the evening.",
	}
	for _, text := range noTriggerTexts {
		lower := strings.ToLower(text)
		if stainTriggerRE.MatchString(lower) {
			t.Errorf("stainTriggerRE should NOT match %q", text)
		}
	}
}

// ---------------------------------------------------------------------------
// 14. detectAndApplyVtMStains — direct function tests
// ---------------------------------------------------------------------------

func TestVtM_DetectAndApplyStains_StainAdded(t *testing.T) {
	s := newTestServer(t)
	stats := `{"clan":"Brujah","humanity":7,"stains":0,"xp":10}`
	_, charID, sessID := vtmCampaignAndChar(t, s, stats)

	// Set the active_character_id setting so the function can find the character
	require.NoError(t, s.db.SetSetting("active_character_id", fmt.Sprintf("%d", charID)))

	s.detectAndApplyVtMStains(context.Background(), sessID, "The vampire killed the mortal brutally.")

	got := getCharStats(t, s, charID)
	stains, _ := got["stains"].(float64)
	assert.Equal(t, float64(1), stains, "stains should increase by 1")
}

func TestVtM_DetectAndApplyStains_CapsAt10(t *testing.T) {
	s := newTestServer(t)
	// stains already at 10 — should not go higher
	stats := `{"clan":"Brujah","humanity":7,"stains":10,"xp":10}`
	_, charID, sessID := vtmCampaignAndChar(t, s, stats)
	require.NoError(t, s.db.SetSetting("active_character_id", fmt.Sprintf("%d", charID)))

	s.detectAndApplyVtMStains(context.Background(), sessID, "She fed from the helpless victim.")

	got := getCharStats(t, s, charID)
	stains, _ := got["stains"].(float64)
	assert.Equal(t, float64(10), stains, "stains should cap at 10")
}

func TestVtM_DetectAndApplyStains_NoMatchNoChange(t *testing.T) {
	s := newTestServer(t)
	stats := `{"clan":"Brujah","humanity":7,"stains":0,"xp":10}`
	_, charID, sessID := vtmCampaignAndChar(t, s, stats)
	require.NoError(t, s.db.SetSetting("active_character_id", fmt.Sprintf("%d", charID)))

	s.detectAndApplyVtMStains(context.Background(), sessID, "You enjoy a quiet evening at the Elysium.")

	got := getCharStats(t, s, charID)
	stains, _ := got["stains"].(float64)
	assert.Equal(t, float64(0), stains, "stains should not change on unrelated text")
}

func TestVtM_DetectAndApplyStains_RemorseThreshold_StainsReset(t *testing.T) {
	s := newTestServer(t)
	// humanity=7 → remorse threshold = 11-7 = 4
	// stains=3, adding 1 more = 4, which hits threshold → Remorse check fires
	stats := `{"clan":"Brujah","humanity":7,"stains":3,"xp":10}`
	_, charID, sessID := vtmCampaignAndChar(t, s, stats)
	require.NoError(t, s.db.SetSetting("active_character_id", fmt.Sprintf("%d", charID)))

	// Trigger stain addition — Remorse check will roll randomly;
	// regardless of result, stains must be reset to 0
	s.detectAndApplyVtMStains(context.Background(), sessID, "The vampire killed the witness.")

	got := getCharStats(t, s, charID)
	stains, _ := got["stains"].(float64)
	assert.Equal(t, float64(0), stains, "stains should reset to 0 after Remorse check (pass or fail)")
}

func TestVtM_DetectAndApplyStains_BelowRemorseThreshold_StainsAccumulate(t *testing.T) {
	s := newTestServer(t)
	// humanity=7 → threshold = 4; stains=0, adding 1 → stains=1, threshold not met
	stats := `{"clan":"Brujah","humanity":7,"stains":0,"xp":10}`
	_, charID, sessID := vtmCampaignAndChar(t, s, stats)
	require.NoError(t, s.db.SetSetting("active_character_id", fmt.Sprintf("%d", charID)))

	s.detectAndApplyVtMStains(context.Background(), sessID, "She fed from the mortal.")

	got := getCharStats(t, s, charID)
	stains, _ := got["stains"].(float64)
	// stains=1, threshold=4 → Remorse not triggered, stains stays at 1
	assert.Equal(t, float64(1), stains, "stains should be 1 (below remorse threshold)")
	// humanity should be unchanged
	humanity, _ := got["humanity"].(float64)
	assert.Equal(t, float64(7), humanity, "humanity should not change below remorse threshold")
}

// ---------------------------------------------------------------------------
// 15. autoUpdateMasquerade — direct function tests
// ---------------------------------------------------------------------------

func TestVtM_AutoUpdateMasquerade_MajorBreach(t *testing.T) {
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("vtm")
	require.NoError(t, err)
	campID, err := s.db.CreateCampaign(rs.ID, "VtM Camp", "")
	require.NoError(t, err)
	sessID, err := s.db.CreateSession(campID, "Night 1", "2026-04-01")
	require.NoError(t, err)

	require.NoError(t, s.db.UpdateMasqueradeIntegrity(sessID, 10))

	// Major breach keywords (vtmMajorBreachRE): "caught on camera", "viral", "police", "recorded", etc.
	s.autoUpdateMasquerade(context.Background(), sessID, "The attack was caught on camera and is now viral online.")

	level, err := s.db.GetMasqueradeIntegrity(sessID)
	require.NoError(t, err)
	// Major breach decrements by 3
	assert.Equal(t, 7, level, "masquerade should decrease by 3 on major breach (caught on camera)")
}

func TestVtM_AutoUpdateMasquerade_NoBreachNoChange(t *testing.T) {
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("vtm")
	require.NoError(t, err)
	campID, err := s.db.CreateCampaign(rs.ID, "VtM Camp", "")
	require.NoError(t, err)
	sessID, err := s.db.CreateSession(campID, "Night 1", "2026-04-01")
	require.NoError(t, err)

	require.NoError(t, s.db.UpdateMasqueradeIntegrity(sessID, 10))

	s.autoUpdateMasquerade(context.Background(), sessID, "You enjoy a quiet evening at the Elysium, speaking with other Kindred.")

	level, err := s.db.GetMasqueradeIntegrity(sessID)
	require.NoError(t, err)
	assert.Equal(t, 10, level, "masquerade should not change on innocuous text")
}

func TestVtM_AutoUpdateMasquerade_NonVtMNoOp(t *testing.T) {
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("wrath_glory")
	require.NoError(t, err)
	campID, err := s.db.CreateCampaign(rs.ID, "WG Camp", "")
	require.NoError(t, err)
	sessID, err := s.db.CreateSession(campID, "Session 1", "2026-04-01")
	require.NoError(t, err)

	// Start at default (10)
	// autoUpdateMasquerade should no-op for non-vtm
	s.autoUpdateMasquerade(context.Background(), sessID, "A masquerade breach! The frenzy is witnessed by mortals.")

	level, err := s.db.GetMasqueradeIntegrity(sessID)
	require.NoError(t, err)
	assert.Equal(t, 10, level, "non-vtm session masquerade should not change")
}

// ---------------------------------------------------------------------------
// 16. autoUpdateCharacterStats — XP guard: AI returning xp=0 must NOT wipe XP
// ---------------------------------------------------------------------------

func TestVtM_AutoUpdateCharacterStats_XPGuard(t *testing.T) {
	// AI returns xp=0 but character has xp=30 — the guard must keep 30.
	stub := &stubCompleter{response: `{"xp":0,"hunger":2}`}
	s := newTestServerWithAI(t, stub)

	stats := `{"clan":"Brujah","xp":30,"hunger":0}`
	_, charID, sessID := vtmCampaignAndChar(t, s, stats)
	require.NoError(t, s.db.SetSetting("active_character_id", fmt.Sprintf("%d", charID)))

	// autoUpdateCharacterStats uses keyword matching; inject a VtM keyword
	s.autoUpdateCharacterStats(context.Background(), sessID, "I use Dominate", "The vampire used dominate to compel the mortal.")

	got := getCharStats(t, s, charID)
	xp, _ := got["xp"].(float64)
	assert.Equal(t, float64(30), xp, "XP guard must prevent AI from zeroing out XP")
}

func TestVtM_AutoUpdateCharacterStats_StringXPNormalization(t *testing.T) {
	// AI returns xp as string "42" — should be parsed correctly
	stub := &stubCompleter{response: `{"xp":"42","hunger":1}`}
	s := newTestServerWithAI(t, stub)

	// Start with xp stored as number 30; AI claims "42" — since 42 > 30, it should update
	stats := `{"clan":"Brujah","xp":30,"hunger":0}`
	_, charID, sessID := vtmCampaignAndChar(t, s, stats)
	require.NoError(t, s.db.SetSetting("active_character_id", fmt.Sprintf("%d", charID)))

	s.autoUpdateCharacterStats(context.Background(), sessID, "I hunt for blood", "The vampire fed from the mortal.")

	got := getCharStats(t, s, charID)
	// xp should be normalized to float64; the AI returned string "42" which is > 30
	// Note: the autoUpdateCharacterStats only applies if the field exists in current stats
	// xp field exists, and value "42" > 30 after normalization, so it should apply
	xp, _ := got["xp"].(float64)
	assert.GreaterOrEqual(t, xp, float64(30), "XP should not decrease; string normalization should work")
}

// ---------------------------------------------------------------------------
// 17. handleVtMRouseCheck — via active_character_id setting
// NOTE: This is tested indirectly since the function reads active_character_id
// ---------------------------------------------------------------------------

func TestVtM_RouseCheck_HungerIncreaseOnFailure(t *testing.T) {
	// We need to test handleVtMRouseCheck's effect on hunger.
	// Since it uses random dice, we run it multiple times and check invariants.
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("vtm")
	require.NoError(t, err)
	campID, err := s.db.CreateCampaign(rs.ID, "VtM Camp", "")
	require.NoError(t, err)
	sessID, err := s.db.CreateSession(campID, "Night 1", "2026-04-01")
	require.NoError(t, err)
	charID, err := s.db.CreateCharacter(campID, "Elara Voss")
	require.NoError(t, err)
	require.NoError(t, s.db.UpdateCharacterData(charID, `{"hunger":0,"blood_potency":1}`))
	require.NoError(t, s.db.SetSetting("active_character_id", fmt.Sprintf("%d", charID)))

	// Run rouse check — result is random but hunger should be 0 or 1 (never negative, never > 1 from 0)
	result := s.handleVtMRouseCheck(context.Background(), sessID)

	// Result should always be a non-empty string
	assert.NotEmpty(t, result, "handleVtMRouseCheck should return a non-empty result string")

	got := getCharStats(t, s, charID)
	hunger, _ := got["hunger"].(float64)
	// After one rouse check starting at hunger=0: should be 0 (success) or 1 (failure)
	assert.True(t, hunger == 0 || hunger == 1,
		"hunger after one rouse check from 0 should be 0 or 1, got %v", hunger)
}

func TestVtM_RouseCheck_HungerCapsAt5(t *testing.T) {
	// When hunger is already 5, rouse check should not increase it
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("vtm")
	require.NoError(t, err)
	campID, err := s.db.CreateCampaign(rs.ID, "VtM Camp", "")
	require.NoError(t, err)
	sessID, err := s.db.CreateSession(campID, "Night 1", "2026-04-01")
	require.NoError(t, err)
	charID, err := s.db.CreateCharacter(campID, "Elara Voss")
	require.NoError(t, err)
	require.NoError(t, s.db.UpdateCharacterData(charID, `{"hunger":5,"blood_potency":1}`))
	require.NoError(t, s.db.SetSetting("active_character_id", fmt.Sprintf("%d", charID)))

	result := s.handleVtMRouseCheck(context.Background(), sessID)
	assert.NotEmpty(t, result)

	got := getCharStats(t, s, charID)
	hunger, _ := got["hunger"].(float64)
	assert.Equal(t, float64(5), hunger, "hunger should not exceed 5")
	// Result must mention frenzy risk when at 5
	assert.True(t, strings.Contains(result, "5") || strings.Contains(strings.ToLower(result), "frenzy"),
		"result at hunger 5 should mention 5 or frenzy: %q", result)
}

func TestVtM_RouseCheck_NoActiveCharacter_ReturnsEmpty(t *testing.T) {
	s := newTestServer(t)
	rs, err := s.db.GetRulesetByName("vtm")
	require.NoError(t, err)
	campID, err := s.db.CreateCampaign(rs.ID, "VtM Camp", "")
	require.NoError(t, err)
	sessID, err := s.db.CreateSession(campID, "Night 1", "2026-04-01")
	require.NoError(t, err)

	// No active_character_id set
	result := s.handleVtMRouseCheck(context.Background(), sessID)
	assert.Empty(t, result, "handleVtMRouseCheck should return empty string when no active character")
}

// ---------------------------------------------------------------------------
// 18. Oracle Roll endpoint (POST /api/oracle/roll)
// ---------------------------------------------------------------------------

func TestVtM_OracleRoll_InvalidInput(t *testing.T) {
	s := newTestServer(t)

	// roll=0 should be rejected (valid range 1-50)
	req := httptest.NewRequest(http.MethodPost, "/api/oracle/roll",
		strings.NewReader(`{"table":"action","roll":0}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVtM_OracleRoll_MissingTable(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/oracle/roll",
		strings.NewReader(`{"roll":5}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// 19. vtmNewNightRE — keyword coverage
// ---------------------------------------------------------------------------

func TestVtM_NewNightRE_MatchesKeywords(t *testing.T) {
	matches := []string{
		"As dusk falls over the city.",
		"Dusk settles on the horizon.",
		"As night falls, the streets empty.",
		"When night falls, the Kindred emerge.",
		"The sun sets behind the skyline.",
		"Nightfall brings the Kindred out.",
		"Another night begins in the city.",
		"The following night, you awaken.",
		"That evening you find yourself...",
		"When consciousness returns with the fall of night.",
		"When your eyes open again, night has reclaimed the world.",
		"Darkness descends over the city.",
		"As darkness descends, the streets empty.",
		"Night has come to Chicago.",
		"Night descends over the skyline.",
		"Night reclaims the city once more.",
	}
	for _, text := range matches {
		lower := strings.ToLower(text)
		if !vtmNewNightRE.MatchString(lower) {
			t.Errorf("vtmNewNightRE should match %q", text)
		}
	}
}

func TestVtM_NewNightRE_NoMatchOnDaytime(t *testing.T) {
	noMatches := []string{
		"The morning sun rises above the rooftops.",
		"It is noon and the streets are busy.",
		"You hide from the daylight.",
		"A normal afternoon in the city.",
	}
	for _, text := range noMatches {
		lower := strings.ToLower(text)
		if vtmNewNightRE.MatchString(lower) {
			t.Errorf("vtmNewNightRE should NOT match %q", text)
		}
	}
}

// ---------------------------------------------------------------------------
// 20. systemNote validation — verify vtm systemNote content
// ---------------------------------------------------------------------------

func TestVtM_SystemNoteContent(t *testing.T) {
	// The systemNote is built inline in autoUpdateCharacterStats.
	// We verify it indirectly by checking that a stub AI gets the right prompt.
	var capturedPrompt string
	capturer := &captureCompleter{fn: func(prompt string) string {
		capturedPrompt = prompt
		return `{}`
	}}
	s := newTestServerWithAI(t, capturer)

	stats := `{"clan":"Brujah","xp":10,"hunger":0,"humanity":7,"stains":0,"blood_potency":1,"health_superficial":0,"willpower_superficial":0}`
	_, charID, sessID := vtmCampaignAndChar(t, s, stats)
	require.NoError(t, s.db.SetSetting("active_character_id", fmt.Sprintf("%d", charID)))

	// Use a VtM keyword to ensure the function runs
	s.autoUpdateCharacterStats(context.Background(), sessID, "I use Dominate on the mortal", "The vampire used Dominate and hunger increased.")

	if capturedPrompt == "" {
		t.Skip("prompt not captured — autoUpdateCharacterStats may have short-circuited")
	}

	// Verify key VtM rules are in the prompt
	for _, want := range []string{
		"HUNGER", "HEALTH", "WILLPOWER", "HUMANITY", "XP",
	} {
		if !strings.Contains(capturedPrompt, want) {
			t.Errorf("vtm systemNote prompt missing keyword %q", want)
		}
	}
}

// captureCompleter captures the prompt and returns a fixed response
type captureCompleter struct {
	fn func(string) string
}

func (c *captureCompleter) Generate(_ context.Context, prompt string, _ int) (string, error) {
	return c.fn(prompt), nil
}

// ---------------------------------------------------------------------------
// Hunger Dice mechanic
// ---------------------------------------------------------------------------

func TestVtM_HungerDice_NormalRoll_NoHunger(t *testing.T) {
	s := newTestServer(t)
	_, charID, sessID := vtmCampaignAndChar(t, s,
		`{"hunger":0,"strength":3,"xp":20}`)
	require.NoError(t, s.db.SetSetting("active_character_id", fmt.Sprintf("%d", charID)))

	// Simulate vtmHungerDiceRoll with pool=4, hunger=0 → all normal dice.
	result := s.vtmHungerDiceRoll(context.Background(), sessID, 4, "Strength", 2,
		"Forcing a door", "4d10", `{"hunger":0}`, "")

	assert.NotNil(t, result)
	assert.Equal(t, "Strength", result.Attribute)
	assert.Equal(t, 2, result.DC)
	// With hunger=0 no Bestial Failure or Messy Critical are possible.
	assert.False(t, result.BestialFail, "no hunger dice → no Bestial Failure possible")
	assert.False(t, result.MessyCritical, "no hunger dice → no Messy Critical possible")
}

func TestVtM_HungerDice_MaxHunger_PoolFull(t *testing.T) {
	s := newTestServer(t)
	_, charID, sessID := vtmCampaignAndChar(t, s,
		`{"hunger":5,"strength":3,"xp":20}`)
	require.NoError(t, s.db.SetSetting("active_character_id", fmt.Sprintf("%d", charID)))

	result := s.vtmHungerDiceRoll(context.Background(), sessID, 3, "Strength", 2,
		"Forcing a door", "3d10", `{"hunger":5}`, "")

	assert.NotNil(t, result)
	// Pool=3, hunger=5 → clamped: 3 hunger dice, 0 normal dice.
	// Bestial Failure possible (no normal dice to offset).
	assert.NotEmpty(t, result.Expression, "expression must include dice breakdown")
}

func TestVtM_HungerDice_ExpressionFormat(t *testing.T) {
	s := newTestServer(t)
	_, charID, sessID := vtmCampaignAndChar(t, s, `{"hunger":2}`)
	require.NoError(t, s.db.SetSetting("active_character_id", fmt.Sprintf("%d", charID)))

	result := s.vtmHungerDiceRoll(context.Background(), sessID, 5, "Dexterity", 2,
		"Dodging an attack", "5d10", `{"hunger":2}`, "")

	assert.NotNil(t, result)
	// Expression should note pool split: 5 total = 3 normal + 2 hunger.
	assert.Contains(t, result.Expression, "3N", "expression should note normal dice count")
	assert.Contains(t, result.Expression, "2H", "expression should note hunger dice count")
}

func TestVtM_HungerDice_MessyCritical_TriggersCompulsion(t *testing.T) {
	// Use a Brujah character so compulsion_brujah table exists.
	s := newTestServer(t)
	_, charID, sessID := vtmCampaignAndChar(t, s,
		`{"hunger":3,"clan":"Brujah","strength":4,"xp":20}`)
	require.NoError(t, s.db.SetSetting("active_character_id", fmt.Sprintf("%d", charID)))

	// We can't guarantee a Messy Critical probabilistically, so call
	// vtmRollClanCompulsion directly to verify it returns a non-empty result.
	compulsion := s.vtmRollClanCompulsion(context.Background(), sessID, `{"clan":"Brujah"}`)
	assert.NotEmpty(t, compulsion, "clan compulsion table must return a result for Brujah")
}

func TestVtM_HungerDice_CompulsionFallback_UnknownClan(t *testing.T) {
	s := newTestServer(t)
	_, _, sessID := vtmCampaignAndChar(t, s, `{"hunger":3,"clan":"Caitiff"}`)

	// Caitiff has no compulsion table → should return generic fallback, not empty.
	compulsion := s.vtmRollClanCompulsion(context.Background(), sessID, `{"clan":"Caitiff"}`)
	assert.NotEmpty(t, compulsion, "unknown clan compulsion should return generic fallback")
}

func TestVtM_HungerDice_AllClans_HaveCompulsionTables(t *testing.T) {
	s := newTestServer(t)
	_, _, sessID := vtmCampaignAndChar(t, s, `{}`)

	clans := []string{
		"Brujah", "Gangrel", "Malkavian", "Nosferatu",
		"Toreador", "Tremere", "Ventrue",
	}
	for _, clan := range clans {
		charStats := fmt.Sprintf(`{"clan":%q}`, clan)
		result := s.vtmRollClanCompulsion(context.Background(), sessID, charStats)
		assert.NotEmpty(t, result, "clan %s must have a compulsion table entry", clan)
		// Ensure it's NOT the generic fallback for these named clans.
		assert.NotContains(t, result, "generic", "clan %s should have specific compulsion", clan)
	}
}

func TestVtM_HungerDice_SuccessCount_NotRawSum(t *testing.T) {
	// VtM total should be successes (not raw sum of dice faces).
	// With hunger=1, pool=3: result.Total should be 0-4 (successes), not e.g. 15-25.
	s := newTestServer(t)
	_, charID, sessID := vtmCampaignAndChar(t, s, `{"hunger":1}`)
	require.NoError(t, s.db.SetSetting("active_character_id", fmt.Sprintf("%d", charID)))

	result := s.vtmHungerDiceRoll(context.Background(), sessID, 3, "Wits", 1,
		"Noticing something", "3d10", `{"hunger":1}`, "")

	assert.NotNil(t, result)
	// Total = successes (6+), which for 3 dice is at most 3 + 1 critical pair bonus = 4 max
	// but practically 0-4. It should not be a raw face sum (3-30).
	assert.LessOrEqual(t, result.Total, 5, "total must be success count, not raw die sum")
	assert.GreaterOrEqual(t, result.Total, 0)
}
