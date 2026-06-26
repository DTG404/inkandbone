package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/digitalghost404/inkandbone/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestRulebook_textPlain(t *testing.T) {
	s := newTestServer(t)
	rulesets, err := s.db.ListRulesets()
	require.NoError(t, err)
	require.NotEmpty(t, rulesets, "expected at least one seeded ruleset")
	rsID := rulesets[0].ID

	body := "# Chapter 1\nThis is the first chapter content.\n# Chapter 2\nThis is the second chapter content.\n"
	req := httptest.NewRequest(http.MethodPost,
		"/api/rulesets/"+strconv.FormatInt(rsID, 10)+"/rulebook",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		ChunksCreated int `json:"chunks_created"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.ChunksCreated)
}

func TestIngestRulebook_noHeadings(t *testing.T) {
	s := newTestServer(t)
	rulesets, err := s.db.ListRulesets()
	require.NoError(t, err)
	require.NotEmpty(t, rulesets, "expected at least one seeded ruleset")
	rsID := rulesets[0].ID

	body := "This is plain text without any headings. It should become one chunk.\n"
	req := httptest.NewRequest(http.MethodPost,
		"/api/rulesets/"+strconv.FormatInt(rsID, 10)+"/rulebook",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		ChunksCreated int `json:"chunks_created"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.ChunksCreated)
}

func TestIngestRulebook_replacesExisting(t *testing.T) {
	s := newTestServer(t)
	rulesets, err := s.db.ListRulesets()
	require.NoError(t, err)
	require.NotEmpty(t, rulesets)
	rsID := rulesets[0].ID
	url := "/api/rulesets/" + strconv.FormatInt(rsID, 10) + "/rulebook"

	// First ingest
	req := httptest.NewRequest(http.MethodPost, url, strings.NewReader("# Old\nOld content.\n"))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Second ingest with different content
	req2 := httptest.NewRequest(http.MethodPost, url, strings.NewReader("# New\nNew content.\n"))
	req2.Header.Set("Content-Type", "text/plain")
	w2 := httptest.NewRecorder()
	s.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	// Old chunk must be gone; only new chunk exists
	chunks, err := s.db.SearchRulebookChunks(rsID, "Old")
	require.NoError(t, err)
	assert.Empty(t, chunks, "old chunks should be deleted on re-ingest")

	chunks, err = s.db.SearchRulebookChunks(rsID, "New")
	require.NoError(t, err)
	assert.Len(t, chunks, 1)
}

func TestIngestRulebook_pdf(t *testing.T) {
	// TODO: PDF integration test requires fixture file
	t.Skip("PDF integration test requires a valid PDF fixture file")
}

func TestSearchRulebook_keywordFallback(t *testing.T) {
	s := newTestServer(t)
	rsID, err := s.db.CreateRuleset("test", "{}", "test")
	require.NoError(t, err)

	chunks := []db.RulebookChunk{{Source: "Core", Heading: "Combat", Content: "Roll dice to attack."}}
	require.NoError(t, s.db.CreateRulebookChunks(rsID, chunks))

	body := `{"query":"dice"}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/rulesets/%d/rulebook/search", rsID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Results []struct {
			Heading string `json:"heading"`
			Content string `json:"content"`
			Source  string `json:"source"`
		} `json:"results"`
		Mode string `json:"mode"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "keyword", resp.Mode)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, "Combat", resp.Results[0].Heading)
}
