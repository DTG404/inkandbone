package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/digitalghost404/inkandbone/internal/ai"
	"github.com/digitalghost404/inkandbone/internal/db"
)

// Server holds dependencies and registers routes.
type Server struct {
	db               *db.DB
	hub              *Hub
	bus              *Bus
	mux              *http.ServeMux
	dataDir          string
	aiClient         ai.Completer // nil when ANTHROPIC_API_KEY is unset
	xpSuggestCounts  sync.Map     // sessionID int64 → int
	settingCache     sync.Map     // rulesetID int64 → string (cached [SETTING]...[/SETTING] block)
	embCache         sync.Map     // rulesetID int64 → []db.RulebookChunk
	autoFailCount    int32        // incremented on automation failure, reset on success; circuit breaker at 3
}

// NewServer creates the HTTP server. dataDir is the base path for uploaded files
// (e.g. ~/.ttrpg). aiClient may be nil if AI features are disabled.
func NewServer(database *db.DB, dataDir string, aiClient ai.Completer) *Server {
	bus := NewBus()
	hub := NewHub(bus)
	s := &Server{
		db:       database,
		hub:      hub,
		bus:      bus,
		mux:      http.NewServeMux(),
		dataDir:  dataDir,
		aiClient: aiClient,
	}
	s.registerRoutes()
	go hub.Run()
	// Capture the ruleset list before launching the goroutine so that rulesets
	// created after NewServer returns are not included in the startup backfill.
	existingRulesets, _ := database.ListRulesets()
	go s.backfillEmbeddings(existingRulesets)
	return s
}

// Bus returns the event bus so the MCP server can publish events.
func (s *Server) Bus() *Bus { return s.bus }

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Block path traversal attempts on file-serving routes before the mux
	// redirects them (Go's mux normalises .. segments via 307).
	if strings.HasPrefix(r.URL.Path, "/api/files/") && strings.Contains(r.URL.Path, "..") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	s.mux.ServeHTTP(w, r)
}

// ListenAndServe starts the HTTP server on addr (e.g. ":7432").
func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s)
}

// Shutdown is a no-op placeholder.
func (s *Server) Shutdown(_ context.Context) error { return nil }

// RegisterStatic serves the embedded React SPA for all routes not matched by /api/ or /ws.
// index.html is served with Cache-Control: no-cache so browsers always re-validate it
// after a binary update (Vite hashes JS/CSS names; a stale index.html causes blank screens).
func (s *Server) RegisterStatic(fsys http.FileSystem) {
	fileServer := http.FileServer(fsys)
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Strip query string for extension check; assets (*.js, *.css) can be
		// cached by hash. Only index.html needs no-cache.
		path := r.URL.Path
		if path == "/" || strings.HasSuffix(path, ".html") {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		fileServer.ServeHTTP(w, r)
	})
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/ws", s.hub.ServeWS)
	s.mux.HandleFunc("/api/health", s.handleHealth)
	// Existing read routes
	s.mux.HandleFunc("GET /api/campaigns", s.handleListCampaigns)
	s.mux.HandleFunc("GET /api/campaigns/{id}/characters", s.handleListCharacters)
	s.mux.HandleFunc("GET /api/campaigns/{id}/sessions", s.handleListSessions)
	s.mux.HandleFunc("GET /api/campaigns/{id}/world-notes", s.handleListWorldNotes)
	s.mux.HandleFunc("GET /api/sessions/{id}/messages", s.handleListMessages)
	s.mux.HandleFunc("POST /api/sessions/{id}/messages", s.handleCreateMessage)
	s.mux.HandleFunc("POST /api/sessions/{id}/gm-respond", s.handleGMRespond)
	s.mux.HandleFunc("GET /api/sessions/{id}/dice-rolls", s.handleListDiceRolls)
	s.mux.HandleFunc("GET /api/maps/{id}/pins", s.handleListMapPins)
	s.mux.HandleFunc("GET /api/context", s.handleGetContext)
	// Plan 7
	s.mux.HandleFunc("GET /api/sessions/{id}/timeline", s.handleGetTimeline)
	// Plan 8
	s.mux.HandleFunc("GET /api/files/{path...}", s.handleServeFile)
	s.mux.HandleFunc("GET /api/campaigns/{id}/maps", s.handleListMaps)
	s.mux.HandleFunc("POST /api/campaigns/{id}/maps", s.handleUploadMap)
	s.mux.HandleFunc("POST /api/campaigns/{id}/maps/generate", s.handleGenerateMap)
	s.mux.HandleFunc("GET /api/maps/{id}", s.handleGetMap)
	s.mux.HandleFunc("PATCH /api/campaigns/{id}", s.handlePatchCampaign)
	s.mux.HandleFunc("PATCH /api/sessions/{id}", s.handlePatchSession)
	s.mux.HandleFunc("POST /api/sessions/{id}/recap", s.handleGenerateRecap)
	s.mux.HandleFunc("POST /api/campaigns/{id}/world-notes/draft", withMaxBody(4096, s.handleDraftWorldNote))
	s.mux.HandleFunc("PATCH /api/world-notes/{id}", s.handlePatchWorldNote)
	s.mux.HandleFunc("PATCH /api/world-notes/{id}/personality", s.handlePatchWorldNotePersonality)
	s.mux.HandleFunc("PATCH /api/world-notes/{id}/reveal", s.handlePatchWorldNoteRevealed)
	s.mux.HandleFunc("GET /api/rulesets/{id}", s.handleGetRuleset)
	s.mux.HandleFunc("GET /api/rulesets/{id}/character-options", s.handleGetCharacterOptions)
	s.mux.HandleFunc("GET /api/rulesets/{id}/rulebook", s.handleListRulebookSources)
	s.mux.HandleFunc("POST /api/rulesets/{id}/rulebook", withMaxBody(50<<20, s.handleIngestRulebook))
	s.mux.HandleFunc("POST /api/rulesets/{id}/rulebook/search", s.handleSearchRulebook)
	s.mux.HandleFunc("PATCH /api/characters/{id}", s.handlePatchCharacter)
	s.mux.HandleFunc("POST /api/characters/{id}/portrait", s.handleUploadPortrait)
	// Feature 1: Streaming GM
	s.mux.HandleFunc("POST /api/sessions/{id}/gm-respond-stream", s.handleGMRespondStream)
	// Feature 4: Dice roller
	s.mux.HandleFunc("POST /api/sessions/{id}/dice-rolls", s.handleRollDice)
	// Feature 5: Condition badges
	s.mux.HandleFunc("PATCH /api/combatants/{id}", s.handlePatchCombatant)
	// Initiative reorder
	s.mux.HandleFunc("PATCH /api/encounters/{id}/combatants/reorder", s.handleReorderCombatants)
	// Feature 8: Map pins
	s.mux.HandleFunc("POST /api/maps/{id}/pins", s.handleCreateMapPin)
	// Feature 9: NPC roster
	s.mux.HandleFunc("GET /api/sessions/{id}/npcs", s.handleListNPCs)
	s.mux.HandleFunc("POST /api/sessions/{id}/npcs", s.handleCreateNPC)
	s.mux.HandleFunc("PATCH /api/npcs/{id}", s.handlePatchNPC)
	s.mux.HandleFunc("DELETE /api/npcs/{id}", s.handleDeleteNPC)
	// Feature 10: Objectives tracker
	s.mux.HandleFunc("GET /api/campaigns/{id}/objectives", s.handleListObjectives)
	s.mux.HandleFunc("POST /api/campaigns/{id}/objectives", s.handleCreateObjective)
	s.mux.HandleFunc("POST /api/campaigns/{id}/objectives/dedup", s.handleDeduplicateObjectives)
	s.mux.HandleFunc("PATCH /api/objectives/{id}", s.handlePatchObjective)
	s.mux.HandleFunc("DELETE /api/objectives/{id}", s.handleDeleteObjective)
	// Feature 11: Player inventory
	s.mux.HandleFunc("GET /api/characters/{id}/items", s.handleListItems)
	s.mux.HandleFunc("POST /api/characters/{id}/items", s.handleCreateItem)
	s.mux.HandleFunc("PATCH /api/items/{id}", s.handlePatchItem)
	s.mux.HandleFunc("DELETE /api/items/{id}", s.handleDeleteItem)
	// Phase A
	s.mux.HandleFunc("POST /api/combat-encounters/{id}/next-turn", s.handleNextTurn)
	s.mux.HandleFunc("GET /api/sessions/{id}/xp", s.handleListXP)
	s.mux.HandleFunc("POST /api/sessions/{id}/xp", s.handleCreateXP)
	s.mux.HandleFunc("DELETE /api/xp/{id}", s.handleDeleteXP)
	// Phase D: Oracle + tension
	s.mux.HandleFunc("POST /api/oracle/roll", s.handleOracleRoll)
	s.mux.HandleFunc("GET /api/sessions/{id}/tension", s.handleGetTension)
	s.mux.HandleFunc("PATCH /api/sessions/{id}/tension", s.handlePatchTension)
	s.mux.HandleFunc("GET /api/sessions/{id}/masquerade", s.handleGetMasqueradeIntegrity)
	s.mux.HandleFunc("PATCH /api/sessions/{id}/masquerade", s.handlePatchMasqueradeIntegrity)
	// Phase D: Relationships
	s.mux.HandleFunc("POST /api/campaigns/{id}/relationships", s.handleCreateRelationship)
	s.mux.HandleFunc("GET /api/campaigns/{id}/relationships", s.handleListRelationships)
	s.mux.HandleFunc("PATCH /api/relationships/{id}", s.handleUpdateRelationship)
	s.mux.HandleFunc("DELETE /api/relationships/{id}", s.handleDeleteRelationship)
	// Factions
	s.mux.HandleFunc("POST /api/campaigns/{id}/factions", s.handleCreateFaction)
	s.mux.HandleFunc("GET /api/campaigns/{id}/factions", s.handleListFactions)
	s.mux.HandleFunc("GET /api/factions/{id}", s.handleGetFaction)
	s.mux.HandleFunc("PATCH /api/factions/{id}", s.handleUpdateFaction)
	s.mux.HandleFunc("DELETE /api/factions/{id}", s.handleDeleteFaction)
	// Adventures
	s.mux.HandleFunc("POST /api/campaigns/{id}/adventures", s.handleCreateAdventure)
	s.mux.HandleFunc("GET /api/campaigns/{id}/adventures", s.handleListAdventures)
	s.mux.HandleFunc("GET /api/adventures/{id}", s.handleGetAdventure)
	s.mux.HandleFunc("PATCH /api/adventures/{id}", s.handleUpdateAdventure)
	s.mux.HandleFunc("DELETE /api/adventures/{id}", s.handleDeleteAdventure)
	s.mux.HandleFunc("PATCH /api/sessions/{id}/adventure", s.handleSetSessionAdventure)
	// Secrets
	s.mux.HandleFunc("POST /api/campaigns/{id}/secrets", s.handleCreateSecret)
	s.mux.HandleFunc("GET /api/campaigns/{id}/secrets", s.handleListSecrets)
	s.mux.HandleFunc("GET /api/secrets/{id}", s.handleGetSecret)
	s.mux.HandleFunc("PATCH /api/secrets/{id}/reveal", s.handleRevealSecret)
	s.mux.HandleFunc("PATCH /api/secrets/{id}", s.handleUpdateSecret)
	s.mux.HandleFunc("DELETE /api/secrets/{id}", s.handleDeleteSecret)
	// Calendar
	s.mux.HandleFunc("GET /api/campaigns/{id}/calendar", s.handleGetCalendar)
	s.mux.HandleFunc("PATCH /api/campaigns/{id}/calendar", s.handlePatchCalendar)
	s.mux.HandleFunc("POST /api/campaigns/{id}/calendar-events", s.handleCreateCalendarEvent)
	s.mux.HandleFunc("GET /api/campaigns/{id}/calendar-events", s.handleListCalendarEvents)
	s.mux.HandleFunc("DELETE /api/calendar-events/{id}", s.handleDeleteCalendarEvent)
	// NPC Stat Blocks
	s.mux.HandleFunc("POST /api/campaigns/{id}/npc-stats", s.handleCreateNpcStat)
	s.mux.HandleFunc("GET /api/campaigns/{id}/npc-stats", s.handleListNpcStats)
	s.mux.HandleFunc("GET /api/npc-stats/{id}", s.handleGetNpcStat)
	s.mux.HandleFunc("PATCH /api/npc-stats/{id}", s.handleUpdateNpcStat)
	s.mux.HandleFunc("DELETE /api/npc-stats/{id}", s.handleDeleteNpcStat)
	// Phase C: GM tools
	s.mux.HandleFunc("POST /api/sessions/{id}/improvise", s.handleImprovise)
	s.mux.HandleFunc("POST /api/campaigns/{id}/pre-session-brief", s.handlePreSessionBrief)
	s.mux.HandleFunc("POST /api/sessions/{id}/detect-threads", s.handleDetectThreads)
	s.mux.HandleFunc("POST /api/campaigns/{id}/ask", s.handleCampaignAsk)
	s.mux.HandleFunc("POST /api/sessions/{id}/reanalyze", s.handleReanalyzeSession)
	// Management UI routes
	s.mux.HandleFunc("GET /api/rulesets", s.handleListRulesets)
	s.mux.HandleFunc("POST /api/campaigns", s.handleCreateCampaign)
	s.mux.HandleFunc("DELETE /api/campaigns/{id}", s.handleDeleteCampaign)
	s.mux.HandleFunc("POST /api/campaigns/{id}/characters", s.handleCreateCharacter)
	s.mux.HandleFunc("DELETE /api/characters/{id}", s.handleDeleteCharacter)
	s.mux.HandleFunc("POST /api/campaigns/{id}/sessions", s.handleCreateSession)
	s.mux.HandleFunc("DELETE /api/sessions/{id}", s.handleDeleteSession)
	s.mux.HandleFunc("PATCH /api/settings", s.handlePatchSettings)
	// Automation settings
	s.mux.HandleFunc("GET /api/settings/automations", s.handleListAutomationSettings)
	s.mux.HandleFunc("PATCH /api/settings/automations", s.handlePatchAutomationSetting)
	// XP advancement
	s.mux.HandleFunc("POST /api/characters/{id}/advance", s.handleAdvanceCharacter)
	s.mux.HandleFunc("POST /api/characters/{id}/suggest-advances", s.handleSuggestAdvances)
	s.mux.HandleFunc("GET /api/talent-description", s.handleTalentDescription)
	// Campaign config (GM Screen)
	s.mux.HandleFunc("GET /api/campaigns/{id}/config", s.handleGetCampaignConfig)
	s.mux.HandleFunc("PATCH /api/campaigns/{id}/config", s.handlePatchCampaignConfig)
	// Multiplayer: typing indicator for player agents
	s.mux.HandleFunc("POST /api/sessions/{id}/typing", s.handleTyping)
	// Macro quick-bar
	s.mux.HandleFunc("GET /api/characters/{id}/macros", s.handleListMacros)
	s.mux.HandleFunc("POST /api/characters/{id}/macros", s.handleCreateMacro)
	s.mux.HandleFunc("PATCH /api/macros/{id}", s.handlePatchMacro)
	s.mux.HandleFunc("DELETE /api/macros/{id}", s.handleDeleteMacro)
	s.mux.HandleFunc("PATCH /api/characters/{id}/macros/reorder", s.handleReorderMacros)
	// Decks
	s.mux.HandleFunc("GET /api/campaigns/{id}/decks", s.handleListDecks)
	s.mux.HandleFunc("POST /api/campaigns/{id}/decks", s.handleCreateDeck)
	s.mux.HandleFunc("DELETE /api/decks/{id}", s.handleDeleteDeck)
	s.mux.HandleFunc("POST /api/decks/{id}/shuffle", s.handleShuffleDeck)
	s.mux.HandleFunc("POST /api/decks/{id}/draw", s.handleDrawCard)
	s.mux.HandleFunc("GET /api/sessions/{id}/deck-draws", s.handleListDeckDraws)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":     "ok",
		"ai_enabled": s.aiClient != nil,
	})
}

// backfillEmbeddings runs at startup to embed any chunks that have no embedding yet.
func (s *Server) backfillEmbeddings(rulesets []db.Ruleset) {
	ctx := context.Background()
	for _, rs := range rulesets {
		chunks, err := s.db.ListChunksForEmbedding(rs.ID)
		if err != nil {
			continue
		}
		for _, c := range chunks {
			emb, err := ai.EmbedText(ctx, c.Content)
			if err != nil {
				log.Printf("backfillEmbeddings: embed chunk %d: %v", c.ID, err)
				continue
			}
			if err := s.db.UpsertChunkEmbedding(c.ID, emb); err != nil {
				log.Printf("backfillEmbeddings: store chunk %d: %v", c.ID, err)
			}
		}
		s.embCache.Delete(rs.ID)
	}
}
