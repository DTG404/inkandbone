package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

func scoreEnabled(campaignID int64) bool {
	scope := os.Getenv("INK_SCORE_CAMPAIGN_ID")
	return campaignID > 0 && (scope == "all" || scope == strconv.FormatInt(campaignID, 10))
}

// Prepare cues while the GM still owns the inference slot. Post-turn background
// jobs can then use the model without competing with location and sound selection.
func (s *Server) prepareLiveScore(r *http.Request, sessionID int64, narration string) {
	url := os.Getenv("INK_SCORE_PREPARE_URL")
	if url == "" || r.Header.Get("X-Ink-Live-Score") != "1" {
		return
	}
	session, err := s.db.GetSession(sessionID)
	if err != nil || session == nil || !scoreEnabled(session.CampaignID) {
		return
	}
	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		return
	}
	body, err := json.Marshal(map[string]any{"session": sessionID, "digest": fmt.Sprintf("%x", sha256.Sum256([]byte(narration)))})
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", r.Header.Get("Origin"))
	req.Header.Set("X-CSRF-Token", r.Header.Get("X-CSRF-Token"))
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8192))
	}
}
