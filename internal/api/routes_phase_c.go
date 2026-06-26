package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// handleImprovise generates an improvised GM complication or plot twist based
// on recent session messages.
func (s *Server) handleImprovise(w http.ResponseWriter, r *http.Request) {
	if s.aiClient == nil {
		http.Error(w, "AI client not available", http.StatusServiceUnavailable)
		return
	}

	idStr := r.PathValue("id")
	sessionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	messages, err := s.db.ListMessages(sessionID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	var sb strings.Builder
	sb.WriteString("You are a tabletop RPG Game Master. Based on the recent session events, generate an interesting improvised complication, NPC reaction, or plot twist (2-3 sentences):\n\n")
	start := len(messages) - 5
	if start < 0 {
		start = 0
	}
	for _, m := range messages[start:] {
		sb.WriteString(fmt.Sprintf("%s: %s\n", m.Role, m.Content))
	}

	result, err := s.aiClient.Generate(r.Context(), sb.String(), 512)
	if err != nil {
		http.Error(w, "AI error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"result": result}) //nolint:errcheck
}

// handlePreSessionBrief generates a concise pre-session brief from world notes
// and active objectives for the given campaign.
func (s *Server) handlePreSessionBrief(w http.ResponseWriter, r *http.Request) {
	if s.aiClient == nil {
		http.Error(w, "AI client not available", http.StatusServiceUnavailable)
		return
	}

	idStr := r.PathValue("id")
	campaignID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}

	notes, _ := s.db.SearchWorldNotes(campaignID, "", "", "", nil)
	objectives, _ := s.db.ListObjectives(campaignID)

	var sb strings.Builder
	sb.WriteString("You are a tabletop RPG Game Master. Generate a concise pre-session brief (3-5 bullets) summarizing what the players should remember before the next session:\n\n")
	sb.WriteString("World Notes:\n")
	for _, n := range notes {
		sb.WriteString(fmt.Sprintf("- %s (%s): %s\n", n.Title, n.Category, n.Content))
	}
	sb.WriteString("\nActive Objectives:\n")
	for _, o := range objectives {
		if o.Status == "active" {
			sb.WriteString(fmt.Sprintf("- %s\n", o.Title))
		}
	}

	result, err := s.aiClient.Generate(r.Context(), sb.String(), 512)
	if err != nil {
		http.Error(w, "AI error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"result": result}) //nolint:errcheck
}

// handleDetectThreads analyzes the session transcript and identifies unresolved
// narrative threads or loose ends.
func (s *Server) handleDetectThreads(w http.ResponseWriter, r *http.Request) {
	if s.aiClient == nil {
		http.Error(w, "AI client not available", http.StatusServiceUnavailable)
		return
	}

	idStr := r.PathValue("id")
	sessionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	messages, err := s.db.ListMessages(sessionID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	var sb strings.Builder
	sb.WriteString("You are a tabletop RPG Game Master. Analyze the session transcript and identify unresolved narrative threads, loose ends, or plot hooks that could be developed in future sessions (list format):\n\n")
	for _, m := range messages {
		sb.WriteString(fmt.Sprintf("%s: %s\n", m.Role, m.Content))
	}

	result, err := s.aiClient.Generate(r.Context(), sb.String(), 512)
	if err != nil {
		http.Error(w, "AI error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"result": result}) //nolint:errcheck
}

// handleCampaignAsk answers a GM question about the campaign using world notes
// as context.
func (s *Server) handleCampaignAsk(w http.ResponseWriter, r *http.Request) {
	if s.aiClient == nil {
		http.Error(w, "AI client not available", http.StatusServiceUnavailable)
		return
	}

	idStr := r.PathValue("id")
	campaignID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}

	var body struct {
		Question string `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Question == "" {
		http.Error(w, "question required", http.StatusBadRequest)
		return
	}

	notes, _ := s.db.SearchWorldNotes(campaignID, "", "", "", nil)

	var sb strings.Builder
	sb.WriteString("You are a knowledgeable tabletop RPG Game Master. Answer the following question about the campaign based on the world notes:\n\n")
	sb.WriteString("World Notes:\n")
	for _, n := range notes {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", n.Title, n.Content))
	}
	sb.WriteString(fmt.Sprintf("\nQuestion: %s", body.Question))

	result, err := s.aiClient.Generate(r.Context(), sb.String(), 512)
	if err != nil {
		http.Error(w, "AI error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"result": result}) //nolint:errcheck
}

// handleReanalyzeSession retroactively runs objective and NPC analysis over the
// full session history. Fires in the background; responds 202 immediately.
// POST /api/sessions/{id}/reanalyze
func (s *Server) handleReanalyzeSession(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	if s.aiClient == nil {
		http.Error(w, "AI not configured", http.StatusServiceUnavailable)
		return
	}

	messages, err := s.db.ListMessages(id)
	if err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Build corpus from all GM (assistant) messages.
	var sb strings.Builder
	for _, m := range messages {
		if m.Role == "assistant" {
			sb.WriteString(m.Content)
			sb.WriteString("\n\n")
		}
	}
	corpus := strings.TrimSpace(sb.String())
	if corpus == "" {
		http.Error(w, "no GM messages found", http.StatusUnprocessableEntity)
		return
	}

	// Truncate to ~12000 chars (~3000 tokens) — take the tail so we analyze
	// the most recent story state rather than early session setup.
	const maxCorpusChars = 12000
	if len(corpus) > maxCorpusChars {
		corpus = corpus[len(corpus)-maxCorpusChars:]
	}

	go s.autoDetectObjectives(context.Background(), id, corpus)
	go s.extractNPCs(context.Background(), id, corpus)

	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]string{"status": "reanalysis started"})
}
