package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/digitalghost404/inkandbone/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRuleset(t *testing.T) {
	s := newTestServer(t)
	// seedCampaign creates a ruleset; find it via ListRulesets
	seedCampaign(t, s.db)
	rulesets, err := s.db.ListRulesets()
	require.NoError(t, err)
	require.NotEmpty(t, rulesets)
	rs := rulesets[0]

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/rulesets/%d", rs.ID), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var got db.Ruleset
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, rs.ID, got.ID)
	assert.Equal(t, rs.Name, got.Name)
}

func TestGetRuleset_notFound(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/rulesets/9999", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPatchCharacter(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)
	charID, err := s.db.CreateCharacter(campID, "Kael")
	require.NoError(t, err)

	// Subscribe before the request so we capture the event
	ch := s.bus.SubscribeContext(t.Context())

	body := `{"data_json":"{\"hp\":10}"}`
	req := httptest.NewRequest(http.MethodPatch,
		fmt.Sprintf("/api/characters/%d", charID),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify DB updated
	char, err := s.db.GetCharacter(charID)
	require.NoError(t, err)
	assert.Equal(t, `{"hp":10}`, char.DataJSON)

	// Verify event published
	var got Event
	select {
	case got = <-ch:
	default:
		t.Fatal("expected character_updated event, got none")
	}
	assert.Equal(t, EventCharacterUpdated, got.Type)
	payload, ok := got.Payload.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, charID, payload["id"])
}

func TestPatchCharacter_currency(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)
	charID, err := s.db.CreateCharacter(campID, "Kael")
	require.NoError(t, err)

	ch := s.bus.SubscribeContext(t.Context())

	body := `{"currency_balance":75,"currency_label":"Coin"}`
	req := httptest.NewRequest(http.MethodPatch,
		fmt.Sprintf("/api/characters/%d", charID),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	char, err := s.db.GetCharacter(charID)
	require.NoError(t, err)
	assert.Equal(t, int64(75), char.CurrencyBalance)
	assert.Equal(t, "Coin", char.CurrencyLabel)

	var got Event
	select {
	case got = <-ch:
	default:
		t.Fatal("expected character_updated event")
	}
	assert.Equal(t, EventCharacterUpdated, got.Type)
	payload := got.Payload.(map[string]any)
	assert.Equal(t, charID, payload["id"])
}

func TestPatchCharacter_currencyBalanceOnly(t *testing.T) {
	s := newTestServer(t)
	campID, _ := seedCampaign(t, s.db)
	charID, err := s.db.CreateCharacter(campID, "Kael")
	require.NoError(t, err)

	body := `{"currency_balance":30}`
	req := httptest.NewRequest(http.MethodPatch,
		fmt.Sprintf("/api/characters/%d", charID),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	char, err := s.db.GetCharacter(charID)
	require.NoError(t, err)
	assert.Equal(t, int64(30), char.CurrencyBalance)
	assert.Equal(t, "Gold", char.CurrencyLabel) // label unchanged
}

func TestUploadPortrait(t *testing.T) {
	dir := t.TempDir()
	s := newTestServerWithDir(t, dir)
	campID, _ := seedCampaign(t, s.db)
	charID, err := s.db.CreateCharacter(campID, "Mira")
	require.NoError(t, err)

	// Subscribe before the request so we capture the event
	ch := s.bus.SubscribeContext(t.Context())

	// Build multipart body
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("portrait", "avatar.jpg")
	require.NoError(t, err)
	_, err = fw.Write(validJPEG)
	require.NoError(t, err)
	mw.Close()

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/characters/%d/portrait", charID),
		&body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		PortraitPath string `json:"portrait_path"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.PortraitPath)
	assert.True(t, strings.HasPrefix(resp.PortraitPath, "portraits/"))

	// Verify DB updated
	char, err := s.db.GetCharacter(charID)
	require.NoError(t, err)
	assert.Equal(t, resp.PortraitPath, char.PortraitPath)

	// Verify event published
	var got Event
	select {
	case got = <-ch:
	default:
		t.Fatal("expected character_updated event, got none")
	}
	assert.Equal(t, EventCharacterUpdated, got.Type)
	payload, ok := got.Payload.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, charID, payload["id"])
	assert.Equal(t, resp.PortraitPath, payload["portrait_path"])
}

func uploadPortraitRequest(t *testing.T, s *Server, charID int64, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("portrait", filename)
	require.NoError(t, err)
	_, err = fw.Write(content)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/characters/%d/portrait", charID), &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	return w
}

func portraitPathFromResponse(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var resp struct {
		PortraitPath string `json:"portrait_path"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp.PortraitPath
}

func TestUploadPortraitDBFailurePreservesExistingPortrait(t *testing.T) {
	dir := t.TempDir()
	s := newTestServerWithDir(t, dir)
	campID, _ := seedCampaign(t, s.db)
	charID, err := s.db.CreateCharacter(campID, "Mira")
	require.NoError(t, err)

	oldRelative := fmt.Sprintf("portraits/%d_avatar.jpg", charID)
	oldBytes := append(append([]byte{}, validJPEG...), []byte("-old")...)
	oldPath := writeAssetFile(t, dir, oldRelative, oldBytes)
	require.NoError(t, s.db.UpdateCharacterPortrait(charID, oldRelative))
	_, err = s.db.SQL().Exec(fmt.Sprintf(`
		CREATE TRIGGER fail_portrait_update
		BEFORE UPDATE OF portrait_path ON characters
		WHEN OLD.id = %d
		BEGIN
			SELECT RAISE(ABORT, 'forced portrait update failure');
		END`, charID))
	require.NoError(t, err)

	newBytes := append(append([]byte{}, validJPEG...), []byte("-new")...)
	w := uploadPortraitRequest(t, s, charID, "avatar.jpg", newBytes)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	character, err := s.db.GetCharacter(charID)
	require.NoError(t, err)
	assert.Equal(t, oldRelative, character.PortraitPath)
	gotOldBytes, err := os.ReadFile(oldPath)
	require.NoError(t, err)
	assert.Equal(t, oldBytes, gotOldBytes)
	entries, err := os.ReadDir(filepath.Join(dir, "portraits"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, filepath.Base(oldRelative), entries[0].Name())
}

func TestUploadPortraitSameOriginalNameCreatesUniqueStoredFiles(t *testing.T) {
	dir := t.TempDir()
	s := newTestServerWithDir(t, dir)
	campID, _ := seedCampaign(t, s.db)
	charID, err := s.db.CreateCharacter(campID, "Mira")
	require.NoError(t, err)

	firstBytes := append(append([]byte{}, validJPEG...), []byte("-first")...)
	first := uploadPortraitRequest(t, s, charID, "avatar.jpg", firstBytes)
	require.Equal(t, http.StatusOK, first.Code)
	firstPath := portraitPathFromResponse(t, first)

	secondBytes := append(append([]byte{}, validJPEG...), []byte("-second")...)
	second := uploadPortraitRequest(t, s, charID, "avatar.jpg", secondBytes)
	require.Equal(t, http.StatusOK, second.Code)
	secondPath := portraitPathFromResponse(t, second)

	assert.NotEqual(t, firstPath, secondPath)
	gotFirst, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(firstPath)))
	require.NoError(t, err)
	assert.Equal(t, firstBytes, gotFirst)
	gotSecond, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(secondPath)))
	require.NoError(t, err)
	assert.Equal(t, secondBytes, gotSecond)
	character, err := s.db.GetCharacter(charID)
	require.NoError(t, err)
	assert.Equal(t, secondPath, character.PortraitPath)
}

func TestUploadPortraitRejectsMismatchedContentWithoutSideEffects(t *testing.T) {
	dir := t.TempDir()
	s := newTestServerWithDir(t, dir)
	campID, _ := seedCampaign(t, s.db)
	charID, err := s.db.CreateCharacter(campID, "Mira")
	require.NoError(t, err)
	ch := s.bus.SubscribeContext(t.Context())

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("portrait", "avatar.jpg")
	require.NoError(t, err)
	_, err = fw.Write(validPNG)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/characters/%d/portrait", charID), &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	character, err := s.db.GetCharacter(charID)
	require.NoError(t, err)
	assert.Empty(t, character.PortraitPath)
	files, err := filepath.Glob(filepath.Join(dir, "portraits", "*"))
	require.NoError(t, err)
	assert.Empty(t, files)
	select {
	case event := <-ch:
		t.Fatalf("unexpected success event: %#v", event)
	default:
	}
}
