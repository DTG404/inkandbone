package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/digitalghost404/inkandbone/internal/ai"
	"github.com/digitalghost404/inkandbone/internal/db"
)

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	messages, err := s.db.ListMessages(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if messages == nil {
		messages = []db.Message{}
	}
	writeJSON(w, messages)
}

func (s *Server) handleCreateMessage(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		respondError(w, "invalid session id", http.StatusBadRequest)
		return
	}
	var body struct {
		ClientActionKey string `json:"client_action_key"`
		Role            string `json:"role"`
		Content         string `json:"content"`
		Whisper         bool   `json:"whisper"`
		CharacterID     *int64 `json:"character_id"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.Role != "user" && body.Role != "assistant" {
		respondError(w, "role must be user or assistant", http.StatusBadRequest)
		return
	}
	if body.Content == "" {
		respondError(w, "content is required", http.StatusBadRequest)
		return
	}
	if body.CharacterID != nil {
		sess, err := s.db.GetSession(id)
		if err != nil || sess == nil {
			respondError(w, "session not found", http.StatusNotFound)
			return
		}
		char, err := s.db.GetCharacter(*body.CharacterID)
		if err != nil || char == nil || char.CampaignID != sess.CampaignID {
			respondError(w, "character not found or not in this campaign", http.StatusBadRequest)
			return
		}
	}
	var msgID int64
	var err error
	created := true
	if body.ClientActionKey != "" {
		if len(body.ClientActionKey) > 100 || body.Role != "user" {
			respondError(w, "invalid action key", 400)
			return
		}
		_, canStream := s.aiClient.(ai.Streamer)
		msgID, created, err = s.db.SubmitAction(id, body.Content, body.Whisper, body.CharacterID, body.ClientActionKey, canStream)
	} else {
		msgID, err = s.db.AppendTranscriptMessage(id, body.Role, body.Content, body.Whisper, body.CharacterID)
	}
	if errors.Is(err, db.ErrTurnConflict) || errors.Is(err, db.ErrActionKeyConflict) {
		respondError(w, err.Error(), 409)
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	w.Header().Set("X-Message-ID", strconv.FormatInt(msgID, 10))
	if created {
		s.bus.Publish(Event{Type: EventMessageCreated, Payload: &MessageCreatedPayload{SessionID: RealtimeInt64(id), MessageID: RealtimeInt64(msgID), Role: RealtimePtr(body.Role), CharacterID: body.CharacterID}})
		w.WriteHeader(http.StatusCreated)
	}
}

func (s *Server) handleGMRespond(w http.ResponseWriter, r *http.Request) {
	s.respondToAction(w, r, false)
}
func (s *Server) handleGMRespondStream(w http.ResponseWriter, r *http.Request) {
	s.respondToAction(w, r, true)
}

func (s *Server) respondToAction(w http.ResponseWriter, r *http.Request, streaming bool) {
	streamer, canStream := s.aiClient.(ai.Streamer)
	responder, canRespond := s.aiClient.(ai.Responder)
	unavailable := func() bool {
		switch {
		case s.aiClient == nil:
			http.Error(w, "AI not configured — set ANTHROPIC_API_KEY", 503)
		case !streaming && !canRespond:
			http.Error(w, "AI client does not support chat", 503)
		case streaming && !canStream:
			http.Error(w, "AI client does not support streaming", 503)
		default:
			return false
		}
		return true
	}
	if r.URL.Query().Get("action_id") == "" && unavailable() {
		return
	}
	id, ok2 := parsePathID(r, "id")
	if !ok2 {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	msgs, err := s.db.ListAIVisibleMessages(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	var actionID int64
	if raw := r.URL.Query().Get("action_id"); raw != "" {
		actionID, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || actionID <= 0 {
			respondError(w, "invalid action id", 400)
			return
		}
	} else if len(msgs) > 0 && msgs[len(msgs)-1].Role == "user" {
		actionID = msgs[len(msgs)-1].ID
	} else {
		http.Error(w, "no player message to respond to", 400)
		return
	}
	token := rand.Text()
	claim, err := s.db.ClaimTurn(id, actionID, token)
	if errors.Is(err, db.ErrTurnConflict) {
		respondError(w, "GM response is already in progress; retry shortly", 409)
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		respondError(w, "action unavailable", 404)
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	if claim.ReplyID != 0 {
		if streaming {
			w.Header().Set("Content-Type", "text/event-stream")
			_ = ai.WriteSSE(w, ai.SSEEvent{Type: SSEEventDelta, Delta: claim.Reply})
			_ = ai.WriteSSE(w, ai.SSEEvent{Type: SSEEventDone})
		}
		return
	}
	turnCtx, cancel := context.WithCancel(r.Context())
	defer func() { cancel(); _ = s.db.ReleaseTurn(actionID, token) }()
	if unavailable() {
		return
	}
	go func() {
		ticker := time.NewTicker(8 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-turnCtx.Done():
				return
			case <-ticker.C:
				if s.db.RenewTurn(actionID, token) != nil {
					cancel()
					return
				}
			}
		}
	}()
	r = r.WithContext(turnCtx)
	if len(msgs) == 0 || msgs[len(msgs)-1].ID != actionID {
		respondError(w, "the active action changed", 409)
		return
	}

	charNameMap := make(map[int64]string)
	for _, m := range msgs {
		if m.CharacterID != nil {
			if _, ok := charNameMap[*m.CharacterID]; !ok {
				if char, err := s.db.GetCharacter(*m.CharacterID); err == nil && char != nil {
					charNameMap[*m.CharacterID] = char.Name
				}
			}
		}
	}

	var history []ai.ChatMessage
	for _, m := range msgs {
		if m.Whisper {
			continue
		}
		content := m.Content
		if m.Role == "user" && m.CharacterID != nil {
			if name, ok := charNameMap[*m.CharacterID]; ok {
				content = "[" + name + "] " + content
			}
		}
		history = append(history, ai.ChatMessage{Role: m.Role, Content: content})
	}
	// Cap history to the last 30 messages to bound input token cost.
	// The session summary in buildWorldContext covers long-term memory.
	const historyWindow = 30
	if len(history) > historyWindow {
		history = history[len(history)-historyWindow:]
		for len(history) > 0 && history[0].Role != "user" {
			history = history[1:]
		}
	}

	// Determine if multi-character session and get last speaker name and ID.
	// Must be resolved before checkAndExecuteRoll so characterName can be passed.
	var lastSpeakerName string
	var lastSpeakerID int64
	if len(charNameMap) > 1 {
		for i := len(msgs) - 1; i >= 0; i-- {
			if msgs[i].Role == "user" && !msgs[i].Whisper && msgs[i].CharacterID != nil {
				if name, ok := charNameMap[*msgs[i].CharacterID]; ok {
					lastSpeakerName = name
					lastSpeakerID = *msgs[i].CharacterID
					break
				}
			}
		}
	}

	// Check if the player's action requires a dice roll under the active ruleset.
	// Do this before building the system prompt so the result can be injected.
	lastPlayerMsg := msgs[len(msgs)-1].Content
	var actorID int64
	if actor := msgs[len(msgs)-1].CharacterID; actor != nil {
		actorID = *actor
	}
	prepared, err := s.prepareAction(withTurnMessage(r.Context(), 0, actorID), id, msgs[len(msgs)-1], lastSpeakerName, token, claim)
	if err != nil {
		serverError(w, r, err)
		return
	}
	roll := prepared.Roll

	worldCtx := s.buildWorldContext(r.Context(), id)
	citations := s.appendRulebookContext(r.Context(), id, lastPlayerMsg, &worldCtx)
	s.appendNPCDisambiguation(r.Context(), id, lastPlayerMsg, &worldCtx)
	if roll != nil {
		outcome := "FAILURE"
		if roll.Success {
			outcome = "SUCCESS"
		}
		dcNote := ""
		if roll.DC > 0 {
			dcNote = fmt.Sprintf(" against DC %d", roll.DC)
		}
		worldCtx += fmt.Sprintf(
			"\n[DICE ROLL]\nAction required a %s check%s.\nReason: %s\nRoll: %s = %d — %s\n[/DICE ROLL]",
			roll.Attribute, dcNote, roll.Reason, roll.Expression, roll.Total, outcome,
		)
		if roll.MessyCritical && roll.Compulsion != "" {
			worldCtx += fmt.Sprintf(
				"\n[MESSY CRITICAL]\nThe character succeeded — but the Beast stirred. This is a Messy Critical. Their Clan Compulsion activates:\n%s\nNarrate the success with a dark, beast-driven complication woven in.\n[/MESSY CRITICAL]",
				roll.Compulsion,
			)
		} else if roll.MessyCritical {
			worldCtx += "\n[MESSY CRITICAL]\nThe character succeeded but the Beast stirred. Narrate success with a dark, uncontrolled complication.\n[/MESSY CRITICAL]"
		}
		if roll.BestialFail {
			worldCtx += "\n[BESTIAL FAILURE]\nThe character failed AND a Hunger die showed a 1. The Beast acted. Narrate the failure as an instinctive, animalistic reaction — the character does something they immediately regret.\n[/BESTIAL FAILURE]"
		} else if !roll.Success {
			worldCtx += "\n[GM DIRECTION]\nThe player's action FAILED. Narrate a setback, complication, or consequence. Do not give them what they wanted. Make failure interesting.\n[/GM DIRECTION]"
		}
	}

	if prepared.Command != "" {
		worldCtx += "\n" + prepared.Command
	}
	var reminder string
	if lastSpeakerName != "" {
		reminder = fmt.Sprintf("[REMINDER] Your response must be exactly 4-5 paragraphs. Count them. End with **What do you do, %s?** on its own line.", safePromptDisplayName(lastSpeakerName))
	} else {
		reminder = "[REMINDER] Your response must be exactly 4-5 paragraphs. Count them. Do not write a sixth paragraph. End with **What do you do?** on its own line."
	}
	var multiPrompt string
	if len(charNameMap) > 1 {
		multiPrompt = fmt.Sprintf(`
[MULTI-CHARACTER SESSION]
CRITICAL: This is a MULTI-PLAYER session. Every character listed below is a PLAYER CHARACTER controlled by a real person (or AI agent). They are NOT NPCs.

The base prompt above says "the player controls only their character" — that referred to a single-player assumption. OVERRIDE IT. In this session:
- MULTIPLE player characters are active. Each one is controlled by an independent actor.
- Characters present: %s
- NARRATE IN THIRD PERSON using each character's name. DO NOT use "you" narration.
- When a player character's name appears in the conversation history prefixed with [Name], that character just acted. Respond to their action.
- DO NOT narrate what a player character does, thinks, or feels unless that player's message describes it.
- When a player character speaks, the dialogue comes from that player's own message — do not put words in their mouth.
- When the text says [Nyx] I cast detect magic, that means Nyx THE PLAYER chose to do that — narrate the result, do not have Nyx do something else.
- ROTATE FOCUS naturally. Give each player character moments of narrative attention.
- If a character is NOT present in the current scene, do not narrate their actions.
- CRITICAL: Follow the [REMINDER] at the bottom of this prompt exactly.
`, formatCharNames(charNameMap))
	}
	systemPrompt, err := s.buildGMSystemPrompt(id, worldCtx+multiPrompt, reminder)
	if err != nil {
		if errors.Is(err, errPromptSessionNotFound) || errors.Is(err, errPromptCampaignNotFound) {
			respondError(w, "session or campaign not found", http.StatusNotFound)
			return
		}
		serverError(w, r, err)
		return
	}

	var fullText string
	if streaming {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Accel-Buffering", "no")

		fullText, err = streamer.StreamRespond(r.Context(), systemPrompt, history, 2048, w)
		if err != nil {
			// Headers already sent; can't send HTTP error status, just log and return
			log.Printf("gm-respond-stream: StreamRespond error (session %d): %v", id, err)
			_ = ai.WriteSSE(w, ai.SSEEvent{Type: SSEEventError, Code: "gm_failed", RequestID: requestID(r)})
			return
		}

		if fullText == "" {
			log.Printf("gm-respond-stream: empty response from model (session %d) — model may have refused or produced only a think block", id)
			fallback := "The GM pauses, seeming lost in thought. **What do you do?**"
			if err := ai.WriteSSE(w, ai.SSEEvent{Type: SSEEventDelta, Delta: fallback}); err != nil {
				return
			}
			fullText = fallback
		}

	} else {
		fullText, err = responder.Respond(r.Context(), systemPrompt, history, 2048)
		if err != nil {
			serverError(w, r, err)
			return
		}
	}
	msgID, err := s.db.FinishTurn(id, actionID, token, fullText, citations)
	if err != nil {
		if !streaming {
			serverError(w, r, err)
			return
		}
		log.Printf("gm-respond-stream: persist assistant message failed (session %d): %v", id, err)
		_ = ai.WriteSSE(w, ai.SSEEvent{Type: SSEEventError, Code: "persistence_failed", RequestID: requestID(r)})
		return
	}
	if streaming {
		s.prepareLiveScore(r, id, fullText)
		if err := ai.WriteSSE(w, ai.SSEEvent{Type: SSEEventDone}); err != nil {
			log.Printf("gm-respond-stream: completion write failed after persistence (session %d): %v", id, err)
		}
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	s.bus.Publish(Event{Type: EventMessageCreated, Payload: &MessageCreatedPayload{SessionID: RealtimeInt64(id), MessageID: RealtimeInt64(msgID), Role: RealtimePtr("assistant")}})
	if lastSpeakerID > 0 {
		s.bus.Publish(Event{Type: EventExpectedAction, Payload: &ExpectedActionPayload{SessionID: RealtimeInt64(id), CharacterID: RealtimeInt64(lastSpeakerID), CharacterName: RealtimePtr(lastSpeakerName)}})
	}

	if !streaming {
		return
	}
	s.autoRevealZones(withTurnMessage(r.Context(), msgID, actorID), id, fullText)
	sessionID := id
	gmText := fullText
	playerAction := lastPlayerMsg
	s.submitPostStreamAutomation(sessionID, settingAutoExtractNPCs, JobModeEvent, func(ctx context.Context) error {
		ctx = withTurnMessage(ctx, msgID, actorID)
		s.extractNPCs(ctx, sessionID, gmText)
		return nil
	})
	s.submitPostStreamAutomation(sessionID, settingAutoGenerateMap, JobModeEvent, func(ctx context.Context) error {
		ctx = withTurnMessage(ctx, msgID, actorID)
		s.autoGenerateMap(ctx, sessionID, gmText)
		return nil
	})
	s.submitPostStreamAutomation(sessionID, settingAutoUpdateStats, JobModeEvent, func(ctx context.Context) error {
		ctx = withTurnMessage(ctx, msgID, actorID)
		s.autoUpdateCharacterStats(ctx, sessionID, playerAction, gmText)
		return nil
	})
	s.submitPostStreamAutomation(sessionID, settingAutoUpdateRecap, JobModeSnapshot, func(ctx context.Context) error {
		ctx = withTurnMessage(ctx, msgID, actorID)
		s.autoUpdateRecap(ctx, sessionID)
		return nil
	})
	s.submitPostStreamAutomation(sessionID, settingAutoDetectObj, JobModeEvent, func(ctx context.Context) error {
		ctx = withTurnMessage(ctx, msgID, actorID)
		s.autoDetectObjectives(ctx, sessionID, gmText)
		return nil
	})
	s.submitPostStreamAutomation(sessionID, settingAutoExtractItems, JobModeEvent, func(ctx context.Context) error {
		ctx = withTurnMessage(ctx, msgID, actorID)
		s.autoExtractItems(ctx, sessionID, gmText)
		return nil
	})
	s.submitPostStreamAutomation(sessionID, settingAutoUpdateCurrency, JobModeEvent, func(ctx context.Context) error {
		ctx = withTurnMessage(ctx, msgID, actorID)
		s.autoUpdateCurrency(ctx, sessionID, gmText)
		return nil
	})
	tensionText := fullText
	if roll != nil && !roll.Success {
		tensionText = "critical failure " + fullText
	}
	immutableTensionText := tensionText
	s.submitPostStreamAutomation(sessionID, settingAutoUpdateTension, JobModeEvent, func(context.Context) error {
		s.autoUpdateTension(sessionID, immutableTensionText)
		return nil
	})
	s.submitPostStreamAutomation(sessionID, settingAutoUpdateMasq, JobModeEvent, func(ctx context.Context) error {
		ctx = withTurnMessage(ctx, msgID, actorID)
		s.autoUpdateMasquerade(ctx, sessionID, gmText)
		return nil
	})
	s.submitPostStreamAutomation(sessionID, settingAutoUpdateSceneTags, JobModeEvent, func(ctx context.Context) error {
		ctx = withTurnMessage(ctx, msgID, actorID)
		s.autoUpdateSceneTags(ctx, sessionID, gmText)
		return nil
	})
	s.submitPostStreamAutomation(sessionID, settingAutoUpdateNight, JobModeEvent, func(ctx context.Context) error {
		ctx = withTurnMessage(ctx, msgID, actorID)
		s.autoUpdateChronicleNight(ctx, sessionID, gmText)
		return nil
	})
	s.submitPostStreamAutomation(sessionID, settingAutoUpdateStats, JobModeEvent, func(ctx context.Context) error {
		ctx = withTurnMessage(ctx, msgID, actorID)
		s.autoVtMDisciplineRouseChecks(ctx, sessionID, playerAction, gmText)
		return nil
	})
	s.submitPostStreamAutomation(sessionID, settingAutoUpdateStats, JobModeEvent, func(ctx context.Context) error {
		ctx = withTurnMessage(ctx, msgID, actorID)
		s.autoDetectVtMEmbrace(ctx, sessionID, gmText)
		return nil
	})
	s.submitPostStreamAutomation(sessionID, settingAutoUpdateNight, JobModeEvent, func(ctx context.Context) error {
		ctx = withTurnMessage(ctx, msgID, actorID)
		s.autoDetectVtMNightDOW(ctx, sessionID, gmText)
		return nil
	})
}

func (s *Server) submitPostStreamAutomation(sessionID int64, kind string, mode JobMode, run func(context.Context) error) {
	job := AutomationJob{
		Key:          fmt.Sprintf("%d:%s", sessionID, kind),
		SessionID:    sessionID,
		Kind:         kind,
		Mode:         mode,
		Run:          run,
		illustration: s.mapSocket != "" && kind == settingAutoGenerateMap,
	}
	if err := s.automations.Submit(s.rootCtx, job); err != nil {
		log.Printf("automation dispatch rejected (session %d, kind %s): %s", sessionID, kind, automationDispatchError(err))
	}
}
