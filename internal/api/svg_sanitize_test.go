package api

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeSVGRejectsActiveAmbiguousAndMalformedContent(t *testing.T) {
	fixtures := []string{
		`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><foreignObject><p>html</p></foreignObject></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"><rect/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><rect onclick="alert(1)"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><use href="javascript:alert(1)"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><use href="data:image/svg+xml,bad"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><use href="https://evil.invalid/a.svg#x"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><use xlink:href="#x"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><rect style="fill:url(https://evil.invalid/x)"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><rect style="fill:url( #safe )"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><rect style="fill:expression(alert(1))"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><rect style="@import:https://evil.invalid/x"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><style>@import url(https://evil.invalid/x)</style></svg>`,
		`<!DOCTYPE svg [<!ENTITY xxe SYSTEM "file:///etc/passwd">]><svg xmlns="http://www.w3.org/2000/svg"><text>&xxe;</text></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg" xmlns:evil="https://evil.invalid"><evil:x/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><rect xmlns="https://evil.invalid"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><image href="#x"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><rect filter="url(#x)"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><rect id="9bad"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><rect fill="url(#bad id)"/></svg>`,
		`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"/>`,
		`<svg xmlns="http://www.w3.org/2000/svg"/><svg xmlns="http://www.w3.org/2000/svg"/>`,
		`<svg xmlns="http://www.w3.org/2000/svg"/>trailing`,
		`<svg xmlns="http://www.w3.org/2000/svg"><path d="M0 0"/></sv`,
		`<html><svg xmlns="http://www.w3.org/2000/svg"/></html>`,
	}
	for _, fixture := range fixtures {
		_, err := SanitizeSVG(fixture)
		assert.ErrorIs(t, err, ErrUnsafeSVG, fixture)
		assert.NotContains(t, err.Error(), fixture)
	}
}

func TestSanitizeSVGPreservesNormalizedSafePrimitives(t *testing.T) {
	input := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 600"><defs><linearGradient id="gold"><stop offset="0%" stop-color="#fff"/><stop offset="100%" stop-color="#000"/></linearGradient></defs><g transform="translate(10 20)"><rect id="room" x="1" y="2" width="30" height="40" fill="url(#gold)" stroke="#c9a84c"/><use href="#room" x="40" y="0"/><circle cx="5" cy="6" r="2"/><path d="M 0 0 L 4 4 Z"/><text x="4" y="8" font-family="serif" font-size="11">Inn &amp; Bone</text></g></svg>`
	got, err := SanitizeSVG(input)
	require.NoError(t, err)
	assert.Contains(t, got, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 600">`)
	assert.Contains(t, got, `<linearGradient id="gold">`)
	assert.Contains(t, got, `fill="url(#gold)"`)
	assert.Contains(t, got, `href="#room"`)
	assert.Contains(t, got, `Inn &amp; Bone`)
	_, err = SanitizeSVG(got)
	require.NoError(t, err)
}

func TestGeneratedMapRejectsUnsafeSVGWithoutFileOrRecord(t *testing.T) {
	var capturedLogs bytes.Buffer
	previousLogOutput := log.Writer()
	log.SetOutput(&capturedLogs)
	t.Cleanup(func() { log.SetOutput(previousLogOutput) })
	responses := []string{
		`<svg xmlns="http://www.w3.org/2000/svg"><script>HOSTILE_PAYLOAD</script></svg>`,
		`<!DOCTYPE svg [<!ENTITY xxe SYSTEM "file:///HOSTILE_PAYLOAD">]><svg xmlns="http://www.w3.org/2000/svg"><text>&xxe;</text></svg>`,
	}
	for index, response := range responses {
		stub := &stubCompleter{response: response}
		s := newTestServerWithAI(t, stub)
		campaignID, _ := seedCampaign(t, s.db)
		body := bytes.NewBufferString(`{"name":"Unsafe","context":"test"}`)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/campaigns/%d/maps/generate", campaignID), body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code, index)
		assert.NotContains(t, rec.Body.String(), "HOSTILE_PAYLOAD")
		assert.NotContains(t, capturedLogs.String(), "HOSTILE_PAYLOAD")
		maps, err := s.db.ListMaps(campaignID)
		require.NoError(t, err)
		assert.Empty(t, maps)
		files, err := filepath.Glob(filepath.Join(s.dataDir, "maps", "*"))
		require.NoError(t, err)
		assert.Empty(t, files)
	}
}

func TestGeneratedMapDBFailureRemovesPublishedFile(t *testing.T) {
	stub := &stubCompleter{response: `<svg xmlns="http://www.w3.org/2000/svg"><rect x="0" y="0" width="10" height="10"/></svg>`}
	s := newTestServerWithAI(t, stub)
	campaignID, _ := seedCampaign(t, s.db)
	_, err := s.db.SQL().Exec(`CREATE TRIGGER reject_generated_map BEFORE INSERT ON maps BEGIN SELECT RAISE(ABORT, 'forced'); END`)
	require.NoError(t, err)
	body := bytes.NewBufferString(`{"name":"DB failure","context":"test"}`)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/campaigns/%d/maps/generate", campaignID), body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	files, err := filepath.Glob(filepath.Join(s.dataDir, "maps", "*"))
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestGeneratedMapPersistsOnlySanitizedSVG(t *testing.T) {
	stub := &stubCompleter{response: "```svg\n" + `<svg viewBox="0 0 10 10"><rect id="room" x="0" y="0" width="10" height="10" fill="#fff"/></svg>` + "\n```"}
	s := newTestServerWithAI(t, stub)
	campaignID, _ := seedCampaign(t, s.db)
	body := bytes.NewBufferString(`{"name":"Safe","context":"test"}`)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/campaigns/%d/maps/generate", campaignID), body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
	maps, err := s.db.ListMaps(campaignID)
	require.NoError(t, err)
	require.Len(t, maps, 1)
	stored, err := os.ReadFile(filepath.Join(s.dataDir, maps[0].ImagePath))
	require.NoError(t, err)
	assert.Contains(t, string(stored), `xmlns="http://www.w3.org/2000/svg"`)
	_, err = SanitizeSVG(string(stored))
	require.NoError(t, err)
}

type detectThenUnsafeSVGCompleter struct{ calls atomic.Int32 }

func (c *detectThenUnsafeSVGCompleter) Generate(_ context.Context, prompt string, _ int) (string, error) {
	c.calls.Add(1)
	if strings.Contains(prompt, "map assistant") {
		return `{"new_location":true,"name":"Ashen Tower","context":"ruin"}`, nil
	}
	return `<!DOCTYPE svg [<!ENTITY xxe SYSTEM "file:///HOSTILE_PAYLOAD">]><svg xmlns="http://www.w3.org/2000/svg"><text>&xxe;</text></svg>`, nil
}

type detectThenValidSVGCompleter struct{}

func (detectThenValidSVGCompleter) Generate(_ context.Context, prompt string, _ int) (string, error) {
	if strings.Contains(prompt, "map assistant") {
		return `{"new_location":true,"name":"Ashen Tower","context":"ruin"}`, nil
	}
	return `<svg xmlns="http://www.w3.org/2000/svg"><rect x="0" y="0" width="10" height="10"/></svg>`, nil
}

func TestAutoGenerateMapUnsafeSVGCountsBreakerFailureAndLeavesNoArtifacts(t *testing.T) {
	completer := &detectThenUnsafeSVGCompleter{}
	s := newTestServerWithAI(t, completer)
	campaignID, sessionID := seedCampaign(t, s.db)
	for range 3 {
		s.autoGenerateMap(t.Context(), sessionID, "The party enters Ashen Tower.")
	}
	health := healthByKey(s.breakers.Snapshot())["unknown:"+settingAutoGenerateMap]
	assert.Equal(t, BreakerOpen, health.Status)
	assert.Equal(t, 3, health.FailureCount)
	maps, err := s.db.ListMaps(campaignID)
	require.NoError(t, err)
	assert.Empty(t, maps)
	files, err := filepath.Glob(filepath.Join(s.dataDir, "maps", "*"))
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestAutoGenerateMapDBFailureCountsBreakerFailureAndRemovesFile(t *testing.T) {
	s := newTestServerWithAI(t, detectThenValidSVGCompleter{})
	campaignID, sessionID := seedCampaign(t, s.db)
	_, err := s.db.SQL().Exec(`CREATE TRIGGER reject_automated_map BEFORE INSERT ON maps BEGIN SELECT RAISE(ABORT, 'forced'); END`)
	require.NoError(t, err)
	s.autoGenerateMap(t.Context(), sessionID, "The party enters Ashen Tower.")
	health := healthByKey(s.breakers.Snapshot())["unknown:"+settingAutoGenerateMap]
	assert.Equal(t, 1, health.FailureCount)
	maps, err := s.db.ListMaps(campaignID)
	require.NoError(t, err)
	assert.Empty(t, maps)
	files, err := filepath.Glob(filepath.Join(s.dataDir, "maps", "*"))
	require.NoError(t, err)
	assert.Empty(t, files)
}
