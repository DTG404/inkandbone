package mcp

import (
	"github.com/digitalghost404/inkandbone/internal/ai"
	"github.com/digitalghost404/inkandbone/internal/api"
	"github.com/digitalghost404/inkandbone/internal/db"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server and holds shared dependencies.
type Server struct {
	db       *db.DB
	bus      *api.Bus
	srv      *server.MCPServer
	aiClient ai.Completer // nil when no AI key is configured
}

// New creates the MCP server and registers all tools.
func New(database *db.DB, bus *api.Bus, aiClient ai.Completer) *Server {
	s := &Server{
		db:       database,
		bus:      bus,
		srv:      server.NewMCPServer("ink & bone", "1.0.0"),
		aiClient: aiClient,
	}
	s.registerTools()
	return s
}

// Start runs the MCP stdio transport. Blocks until stdin closes.
func (s *Server) Start() error {
	return server.ServeStdio(s.srv)
}

func (s *Server) registerTools() {
	// Context
	s.srv.AddTool(mcplib.NewTool("get_context",
		mcplib.WithDescription("Get a full game-state snapshot: active campaign, character, session, recent messages, and active combat. Call at the start of every session."),
	), s.handleGetContext)

	// Campaign / session
	s.srv.AddTool(mcplib.NewTool("set_active",
		mcplib.WithDescription("Set the active campaign, session, or character by ID. All parameters are optional; provide at least one."),
		mcplib.WithNumber("campaign_id", mcplib.Description("Campaign ID to make active")),
		mcplib.WithNumber("session_id", mcplib.Description("Session ID to make active")),
		mcplib.WithNumber("character_id", mcplib.Description("Character ID to make active")),
	), s.handleSetActive)

	s.srv.AddTool(mcplib.NewTool("start_session",
		mcplib.WithDescription("Create a new play session under the active campaign and make it active."),
		mcplib.WithString("title", mcplib.Required(), mcplib.Description("Session title")),
		mcplib.WithString("date", mcplib.Required(), mcplib.Description("Session date (YYYY-MM-DD)")),
		mcplib.WithString("narrative", mcplib.Description("Optional opening narrative to log")),
	), s.handleStartSession)

	s.srv.AddTool(mcplib.NewTool("end_session",
		mcplib.WithDescription("Close the active session and save a summary."),
		mcplib.WithString("summary", mcplib.Required(), mcplib.Description("Session summary")),
		mcplib.WithString("narrative", mcplib.Description("Optional closing narrative to log")),
	), s.handleEndSession)

	// Character
	s.srv.AddTool(mcplib.NewTool("get_character_sheet",
		mcplib.WithDescription("Read the full character sheet for the active (or specified) character."),
		mcplib.WithNumber("character_id", mcplib.Description("Character ID (defaults to active)")),
	), s.handleGetCharacterSheet)

	s.srv.AddTool(mcplib.NewTool("update_character",
		mcplib.WithDescription("Patch character fields. 'updates' is a JSON object — top-level keys are merged into the character data."),
		mcplib.WithString("updates", mcplib.Required(), mcplib.Description(`JSON object of fields to update, e.g. {"hp":15,"level":3}`)),
		mcplib.WithNumber("character_id", mcplib.Description("Character ID (defaults to active)")),
		mcplib.WithString("narrative", mcplib.Description("Optional narrative to log")),
	), s.handleUpdateCharacter)

	s.srv.AddTool(mcplib.NewTool("add_item",
		mcplib.WithDescription("Add an item to the character's inventory."),
		mcplib.WithString("item_name", mcplib.Required(), mcplib.Description("Item to add")),
		mcplib.WithNumber("character_id", mcplib.Description("Character ID (defaults to active)")),
		mcplib.WithString("narrative", mcplib.Description("Optional narrative to log")),
	), s.handleAddItem)

	s.srv.AddTool(mcplib.NewTool("remove_item",
		mcplib.WithDescription("Remove the first matching item from the character's inventory."),
		mcplib.WithString("item_name", mcplib.Required(), mcplib.Description("Item to remove")),
		mcplib.WithNumber("character_id", mcplib.Description("Character ID (defaults to active)")),
		mcplib.WithString("narrative", mcplib.Description("Optional narrative to log")),
	), s.handleRemoveItem)

	// Combat
	s.srv.AddTool(mcplib.NewTool("start_combat",
		mcplib.WithDescription("Start a combat encounter in the active session. Deactivates any existing encounter."),
		mcplib.WithString("name", mcplib.Required(), mcplib.Description("Encounter name")),
		mcplib.WithString("combatants", mcplib.Required(), mcplib.Description(`JSON array of combatants, e.g. [{"name":"Goblin","initiative":14,"hp_max":7,"is_player":false}]`)),
		mcplib.WithString("narrative", mcplib.Description("Optional narrative to log")),
	), s.handleStartCombat)

	s.srv.AddTool(mcplib.NewTool("update_combatant",
		mcplib.WithDescription("Update a combatant's HP and/or conditions."),
		mcplib.WithNumber("combatant_id", mcplib.Required(), mcplib.Description("Combatant ID")),
		mcplib.WithNumber("hp_current", mcplib.Description("New current HP")),
		mcplib.WithString("conditions", mcplib.Description(`JSON array of condition strings, e.g. ["poisoned","prone"]`)),
		mcplib.WithString("narrative", mcplib.Description("Optional narrative to log")),
	), s.handleUpdateCombatant)

	s.srv.AddTool(mcplib.NewTool("end_combat",
		mcplib.WithDescription("Close the active combat encounter."),
		mcplib.WithString("narrative", mcplib.Description("Optional narrative to log")),
	), s.handleEndCombat)

	// World
	s.srv.AddTool(mcplib.NewTool("create_world_note",
		mcplib.WithDescription("Create a world note (NPC, location, faction, or item) in the active campaign."),
		mcplib.WithString("title", mcplib.Required(), mcplib.Description("Note title")),
		mcplib.WithString("content", mcplib.Required(), mcplib.Description("Note content")),
		mcplib.WithString("category", mcplib.Required(), mcplib.Description("One of: npc, location, faction, item")),
		mcplib.WithString("narrative", mcplib.Description("Optional narrative to log")),
	), s.handleCreateWorldNote)

	s.srv.AddTool(mcplib.NewTool("update_world_note",
		mcplib.WithDescription("Edit an existing world note."),
		mcplib.WithNumber("note_id", mcplib.Required(), mcplib.Description("World note ID")),
		mcplib.WithString("title", mcplib.Required(), mcplib.Description("New title")),
		mcplib.WithString("content", mcplib.Required(), mcplib.Description("New content")),
		mcplib.WithString("tags", mcplib.Description(`JSON array of tag strings, e.g. ["npc","villain"]`)),
		mcplib.WithString("narrative", mcplib.Description("Optional narrative to log")),
	), s.handleUpdateWorldNote)

	s.srv.AddTool(mcplib.NewTool("search_world_notes",
		mcplib.WithDescription("Search world notes by title/content keyword and/or category."),
		mcplib.WithString("query", mcplib.Description("Text to search in title and content")),
		mcplib.WithString("category", mcplib.Description("Filter by category (npc, location, faction, item)")),
	), s.handleSearchWorldNotes)

	// Dice
	s.srv.AddTool(mcplib.NewTool("roll_dice",
		mcplib.WithDescription("Roll dice using standard notation and log the result to the session."),
		mcplib.WithString("expression", mcplib.Required(), mcplib.Description(`Dice expression e.g. "2d6+3", "d20", "1d8-1"`)),
		mcplib.WithString("narrative", mcplib.Description("Optional narrative to log")),
	), s.handleRollDice)

	// Lifecycle — campaign & character creation
	s.srv.AddTool(mcplib.NewTool("create_campaign",
		mcplib.WithDescription("Create a new campaign under a ruleset and make it active. Call once at the very start of a new game."),
		mcplib.WithString("ruleset", mcplib.Required(), mcplib.Description("Ruleset name: dnd5e, ironsworn, vtm, coc, or cyberpunk")),
		mcplib.WithString("name", mcplib.Required(), mcplib.Description("Campaign name")),
		mcplib.WithString("description", mcplib.Description("Optional campaign description")),
	), s.handleCreateCampaign)

	s.srv.AddTool(mcplib.NewTool("list_campaigns",
		mcplib.WithDescription("List all campaigns."),
	), s.handleListCampaigns)

	s.srv.AddTool(mcplib.NewTool("close_campaign",
		mcplib.WithDescription("Soft-close a campaign (marks inactive, preserves data). Errors if a session is currently open."),
		mcplib.WithNumber("campaign_id", mcplib.Description("Campaign ID (defaults to active campaign)")),
	), s.handleCloseCampaign)

	s.srv.AddTool(mcplib.NewTool("delete_campaign",
		mcplib.WithDescription("Permanently delete a campaign and all its data. Requires confirm: true."),
		mcplib.WithNumber("campaign_id", mcplib.Required(), mcplib.Description("Campaign ID")),
		mcplib.WithBoolean("confirm", mcplib.Required(), mcplib.Description("Must be true to proceed with deletion")),
	), s.handleDeleteCampaign)

	s.srv.AddTool(mcplib.NewTool("create_character",
		mcplib.WithDescription("Create a new player character in the active campaign and make them active. Starting stats are automatically rolled for the campaign's ruleset."),
		mcplib.WithString("name", mcplib.Required(), mcplib.Description("Character name")),
		mcplib.WithNumber("campaign_id", mcplib.Description("Campaign ID (defaults to active campaign)")),
	), s.handleCreateCharacter)

	s.srv.AddTool(mcplib.NewTool("list_characters",
		mcplib.WithDescription("List all characters in the active (or specified) campaign."),
		mcplib.WithNumber("campaign_id", mcplib.Description("Campaign ID (defaults to active campaign)")),
	), s.handleListCharacters)

	s.srv.AddTool(mcplib.NewTool("list_sessions",
		mcplib.WithDescription("List all sessions for the active (or specified) campaign, newest first."),
		mcplib.WithNumber("campaign_id", mcplib.Description("Campaign ID (defaults to active campaign)")),
	), s.handleListSessions)

	// Maps
	s.srv.AddTool(mcplib.NewTool("add_map_pin",
		mcplib.WithDescription("Pin a location on the active campaign map."),
		mcplib.WithNumber("map_id", mcplib.Required(), mcplib.Description("Map ID")),
		mcplib.WithNumber("x", mcplib.Required(), mcplib.Description("X position as fraction 0.0–1.0")),
		mcplib.WithNumber("y", mcplib.Required(), mcplib.Description("Y position as fraction 0.0–1.0")),
		mcplib.WithString("label", mcplib.Required(), mcplib.Description("Pin label")),
		mcplib.WithString("note", mcplib.Description("Pin note text")),
		mcplib.WithString("color", mcplib.Description("Pin color (hex or name)")),
	), s.handleAddMapPin)

	// Plan 8
	s.srv.AddTool(mcplib.NewTool("generate_session_recap",
		mcplib.WithDescription("Generate a narrative recap for a session using AI. Reads messages and dice rolls, writes the summary, and fires a session_updated event."),
		mcplib.WithNumber("session_id", mcplib.Description("Session ID (defaults to active session)")),
	), s.handleGenerateSessionRecap)

	// Plan 9
	s.srv.AddTool(mcplib.NewTool("search_rulebook",
		mcplib.WithDescription("Search rulebook chunks by keyword. Returns up to 3 matching sections (heading + content). Call proactively when adjudicating rules."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Keyword to search in rulebook headings and content")),
		mcplib.WithNumber("ruleset_id", mcplib.Description("Ruleset ID (defaults to active campaign's ruleset)")),
	), s.handleSearchRulebook)
}

