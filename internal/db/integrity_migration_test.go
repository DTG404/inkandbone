package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type expectedForeignKey struct {
	table    string
	from     string
	parent   string
	onDelete string
}

func TestForeignKeySchemaDeclaresEveryDeleteAction(t *testing.T) {
	d, err := Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, d.Close()) })

	want := []expectedForeignKey{
		{"adventures", "campaign_id", "campaigns", "CASCADE"},
		{"calendar_events", "campaign_id", "campaigns", "CASCADE"},
		{"calendar_events", "session_id", "sessions", "SET NULL"},
		{"campaigns", "ruleset_id", "rulesets", "RESTRICT"},
		{"character_macros", "character_id", "characters", "CASCADE"},
		{"characters", "campaign_id", "campaigns", "CASCADE"},
		{"combat_encounters", "session_id", "sessions", "CASCADE"},
		{"combatants", "character_id", "characters", "SET NULL"},
		{"combatants", "encounter_id", "combat_encounters", "CASCADE"},
		{"deck_draws", "deck_id", "decks", "CASCADE"},
		{"deck_draws", "session_id", "sessions", "CASCADE"},
		{"decks", "campaign_id", "campaigns", "CASCADE"},
		{"dice_rolls", "session_id", "sessions", "CASCADE"},
		{"factions", "campaign_id", "campaigns", "CASCADE"},
		{"items", "character_id", "characters", "CASCADE"},
		{"map_pins", "map_id", "maps", "CASCADE"},
		{"map_tokens", "map_id", "maps", "CASCADE"},
		{"map_zones", "map_id", "maps", "CASCADE"},
		{"maps", "campaign_id", "campaigns", "CASCADE"},
		{"messages", "character_id", "characters", "SET NULL"},
		{"messages", "session_id", "sessions", "CASCADE"},
		{"npc_stats", "campaign_id", "campaigns", "CASCADE"},
		{"objectives", "campaign_id", "campaigns", "CASCADE"},
		{"objectives", "parent_id", "objectives", "CASCADE"},
		{"oracle_tables", "ruleset_id", "rulesets", "SET NULL"},
		{"relationships", "campaign_id", "campaigns", "CASCADE"},
		{"rulebook_chunks", "ruleset_id", "rulesets", "CASCADE"},
		{"secrets", "campaign_id", "campaigns", "CASCADE"},
		{"secrets", "revealed_at_session_id", "sessions", "SET NULL"},
		{"session_npcs", "session_id", "sessions", "CASCADE"},
		{"sessions", "adventure_id", "adventures", "SET NULL"},
		{"sessions", "campaign_id", "campaigns", "CASCADE"},
		{"world_notes", "campaign_id", "campaigns", "CASCADE"},
		{"xp_log", "session_id", "sessions", "CASCADE"},
	}

	var got []expectedForeignKey
	rows, err := d.db.Query(`
		SELECT m.name, fk."from", fk."table", fk.on_delete
		FROM sqlite_master AS m, pragma_foreign_key_list(m.name) AS fk
		WHERE m.type = 'table'
		ORDER BY m.name, fk."from"`)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var edge expectedForeignKey
		require.NoError(t, rows.Scan(&edge.table, &edge.from, &edge.parent, &edge.onDelete))
		got = append(got, edge)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, want, got)
}

func TestDeleteGraphUsesDeclaredRelationships(t *testing.T) {
	d, err := Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, d.Close()) })

	rulesetID := mustInsertID(t, d.db, `INSERT INTO rulesets(name, schema_json) VALUES ('custom_delete_graph', '[]')`)
	campaignID := mustInsertID(t, d.db, `INSERT INTO campaigns(ruleset_id, name) VALUES (?, 'graph')`, rulesetID)
	characterID := mustInsertID(t, d.db, `INSERT INTO characters(campaign_id, name) VALUES (?, 'hero')`, campaignID)
	adventureID := mustInsertID(t, d.db, `INSERT INTO adventures(campaign_id, title) VALUES (?, 'arc')`, campaignID)
	sessionID := mustInsertID(t, d.db, `INSERT INTO sessions(campaign_id, title, date, adventure_id) VALUES (?, 'night', '2026-07-17', ?)`, campaignID, adventureID)
	encounterID := mustInsertID(t, d.db, `INSERT INTO combat_encounters(session_id, name) VALUES (?, 'fight')`, sessionID)
	mapID := mustInsertID(t, d.db, `INSERT INTO maps(campaign_id, name, image_path) VALUES (?, 'map', 'maps/graph.png')`, campaignID)
	deckID := mustInsertID(t, d.db, `INSERT INTO decks(campaign_id, name) VALUES (?, 'deck')`, campaignID)
	parentObjectiveID := mustInsertID(t, d.db, `INSERT INTO objectives(campaign_id, title) VALUES (?, 'parent')`, campaignID)

	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO messages(session_id, role, content, character_id) VALUES (?, 'user', 'history', ?)`, []any{sessionID, characterID}},
		{`INSERT INTO session_npcs(session_id, name) VALUES (?, 'guide')`, []any{sessionID}},
		{`INSERT INTO combatants(encounter_id, character_id, name) VALUES (?, ?, 'hero')`, []any{encounterID, characterID}},
		{`INSERT INTO dice_rolls(session_id, expression, result) VALUES (?, '1d20', 10)`, []any{sessionID}},
		{`INSERT INTO xp_log(session_id, note) VALUES (?, 'xp')`, []any{sessionID}},
		{`INSERT INTO items(character_id, name) VALUES (?, 'sword')`, []any{characterID}},
		{`INSERT INTO character_macros(character_id, label, action_text) VALUES (?, 'Attack', 'swing')`, []any{characterID}},
		{`INSERT INTO world_notes(campaign_id, title, category) VALUES (?, 'handout', 'other')`, []any{campaignID}},
		{`INSERT INTO map_pins(map_id, x, y) VALUES (?, .5, .5)`, []any{mapID}},
		{`INSERT INTO map_tokens(map_id, entity_type, entity_id, x, y) VALUES (?, 'character', ?, .5, .5)`, []any{mapID, characterID}},
		{`INSERT INTO map_zones(map_id, name, x, y, width, height) VALUES (?, 'zone', 0, 0, 1, 1)`, []any{mapID}},
		{`INSERT INTO objectives(campaign_id, title, parent_id) VALUES (?, 'child', ?)`, []any{campaignID, parentObjectiveID}},
		{`INSERT INTO relationships(campaign_id, from_name, to_name) VALUES (?, 'a', 'b')`, []any{campaignID}},
		{`INSERT INTO factions(campaign_id, name) VALUES (?, 'guild')`, []any{campaignID}},
		{`INSERT INTO npc_stats(campaign_id, name) VALUES (?, 'guard')`, []any{campaignID}},
		{`INSERT INTO secrets(campaign_id, title, content, revealed_at_session_id) VALUES (?, 'secret', 'text', ?)`, []any{campaignID, sessionID}},
		{`INSERT INTO calendar_events(campaign_id, title, session_id) VALUES (?, 'event', ?)`, []any{campaignID, sessionID}},
		{`INSERT INTO deck_draws(session_id, deck_id, card_json) VALUES (?, ?, '{}')`, []any{sessionID, deckID}},
	}
	for _, statement := range statements {
		_, err := d.db.Exec(statement.query, statement.args...)
		require.NoError(t, err, statement.query)
	}
	var sessionNPCID int64
	require.NoError(t, d.db.QueryRow(`SELECT id FROM session_npcs WHERE session_id = ?`, sessionID).Scan(&sessionNPCID))
	_, err = d.db.Exec(`INSERT INTO map_tokens(map_id, entity_type, entity_id, x, y) VALUES (?, 'npc', ?, .25, .25)`, mapID, sessionNPCID)
	require.NoError(t, err)

	_, err = d.db.Exec(`DELETE FROM rulesets WHERE id = ?`, rulesetID)
	require.ErrorContains(t, err, "FOREIGN KEY constraint failed")

	require.NoError(t, d.DeleteCharacter(characterID))
	assertZeroCount(t, d.db, "items")
	assertZeroCount(t, d.db, "character_macros")
	assertNullReference(t, d.db, "combatants", "character_id")
	assertNullReference(t, d.db, "messages", "character_id")
	assertZeroCountWhere(t, d.db, "map_tokens", `entity_type = 'character'`)

	require.NoError(t, d.DeleteAdventure(adventureID))
	assertNullReference(t, d.db, "sessions", "adventure_id")

	require.NoError(t, d.DeleteSession(sessionID))
	for _, table := range []string{"messages", "session_npcs", "combatants", "combat_encounters", "dice_rolls", "xp_log", "deck_draws"} {
		assertZeroCount(t, d.db, table)
	}
	assertNullReference(t, d.db, "secrets", "revealed_at_session_id")
	assertNullReference(t, d.db, "calendar_events", "session_id")
	assertZeroCountWhere(t, d.db, "map_tokens", `entity_type = 'npc'`)

	require.NoError(t, d.DeleteCampaign(campaignID))
	for _, table := range []string{"campaigns", "objectives", "world_notes", "maps", "map_pins", "map_tokens", "map_zones", "relationships", "factions", "adventures", "npc_stats", "secrets", "calendar_events", "decks"} {
		assertZeroCount(t, d.db, table)
	}
	require.NoError(t, foreignKeyCheck(d.db))
}

func TestDeleteCampaignCascadesCompleteOwnedGraph(t *testing.T) {
	d, err := Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, d.Close()) })

	rulesetID := mustInsertID(t, d.db, `INSERT INTO rulesets(name, schema_json) VALUES ('campaign_graph_ruleset', '[]')`)
	campaignID := mustInsertID(t, d.db, `INSERT INTO campaigns(ruleset_id, name) VALUES (?, 'complete graph')`, rulesetID)
	characterID := mustInsertID(t, d.db, `INSERT INTO characters(campaign_id, name) VALUES (?, 'hero')`, campaignID)
	adventureID := mustInsertID(t, d.db, `INSERT INTO adventures(campaign_id, title) VALUES (?, 'arc')`, campaignID)
	sessionID := mustInsertID(t, d.db, `INSERT INTO sessions(campaign_id, title, date, adventure_id) VALUES (?, 'night', '2026-07-17', ?)`, campaignID, adventureID)
	encounterID := mustInsertID(t, d.db, `INSERT INTO combat_encounters(session_id, name) VALUES (?, 'fight')`, sessionID)
	mapID := mustInsertID(t, d.db, `INSERT INTO maps(campaign_id, name, image_path) VALUES (?, 'map', 'maps/full.png')`, campaignID)
	deckID := mustInsertID(t, d.db, `INSERT INTO decks(campaign_id, name) VALUES (?, 'deck')`, campaignID)
	objectiveID := mustInsertID(t, d.db, `INSERT INTO objectives(campaign_id, title) VALUES (?, 'parent')`, campaignID)
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO messages(session_id, role, content, character_id) VALUES (?, 'user', 'history', ?)`, []any{sessionID, characterID}},
		{`INSERT INTO session_npcs(session_id, name) VALUES (?, 'guide')`, []any{sessionID}},
		{`INSERT INTO combatants(encounter_id, character_id, name) VALUES (?, ?, 'hero')`, []any{encounterID, characterID}},
		{`INSERT INTO dice_rolls(session_id, expression, result) VALUES (?, '1d20', 10)`, []any{sessionID}},
		{`INSERT INTO xp_log(session_id, note) VALUES (?, 'xp')`, []any{sessionID}},
		{`INSERT INTO items(character_id, name) VALUES (?, 'sword')`, []any{characterID}},
		{`INSERT INTO character_macros(character_id, label, action_text) VALUES (?, 'Attack', 'swing')`, []any{characterID}},
		{`INSERT INTO world_notes(campaign_id, title, category, is_revealed) VALUES (?, 'handout', 'other', 1)`, []any{campaignID}},
		{`INSERT INTO map_pins(map_id, x, y) VALUES (?, .5, .5)`, []any{mapID}},
		{`INSERT INTO map_tokens(map_id, entity_type, entity_id, x, y) VALUES (?, 'character', ?, .5, .5)`, []any{mapID, characterID}},
		{`INSERT INTO map_zones(map_id, name, x, y, width, height) VALUES (?, 'zone', 0, 0, 1, 1)`, []any{mapID}},
		{`INSERT INTO objectives(campaign_id, title, parent_id) VALUES (?, 'child', ?)`, []any{campaignID, objectiveID}},
		{`INSERT INTO relationships(campaign_id, from_name, to_name) VALUES (?, 'a', 'b')`, []any{campaignID}},
		{`INSERT INTO factions(campaign_id, name) VALUES (?, 'guild')`, []any{campaignID}},
		{`INSERT INTO npc_stats(campaign_id, name) VALUES (?, 'guard')`, []any{campaignID}},
		{`INSERT INTO secrets(campaign_id, title, content, revealed_at_session_id) VALUES (?, 'secret', 'text', ?)`, []any{campaignID, sessionID}},
		{`INSERT INTO calendar_events(campaign_id, title, session_id) VALUES (?, 'event', ?)`, []any{campaignID, sessionID}},
		{`INSERT INTO deck_draws(session_id, deck_id, card_json) VALUES (?, ?, '{}')`, []any{sessionID, deckID}},
	}
	for _, statement := range statements {
		_, err := d.db.Exec(statement.query, statement.args...)
		require.NoError(t, err, statement.query)
	}

	require.NoError(t, d.DeleteCampaign(campaignID))
	for _, table := range []string{
		"campaigns", "characters", "sessions", "messages", "session_npcs",
		"combat_encounters", "combatants", "dice_rolls", "xp_log", "items",
		"character_macros", "world_notes", "maps", "map_pins", "map_tokens",
		"map_zones", "objectives", "relationships", "factions", "adventures",
		"npc_stats", "secrets", "calendar_events", "decks", "deck_draws",
	} {
		assertZeroCount(t, d.db, table)
	}
	var rulesetCount int
	require.NoError(t, d.db.QueryRow(`SELECT count(*) FROM rulesets WHERE id = ?`, rulesetID).Scan(&rulesetCount))
	assert.Equal(t, 1, rulesetCount)
	require.NoError(t, foreignKeyCheck(d.db))
}

func TestMapTokenPolymorphicIntegrityTriggers(t *testing.T) {
	d, err := Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, d.Close()) })

	rulesetID := mustInsertID(t, d.db, `INSERT INTO rulesets(name, schema_json) VALUES ('token_ruleset', '[]')`)
	campaignID := mustInsertID(t, d.db, `INSERT INTO campaigns(ruleset_id, name) VALUES (?, 'tokens')`, rulesetID)
	characterID := mustInsertID(t, d.db, `INSERT INTO characters(campaign_id, name) VALUES (?, 'hero')`, campaignID)
	sessionID := mustInsertID(t, d.db, `INSERT INTO sessions(campaign_id, title, date) VALUES (?, 'night', '2026-07-17')`, campaignID)
	npcID := mustInsertID(t, d.db, `INSERT INTO session_npcs(session_id, name) VALUES (?, 'guide')`, sessionID)
	mapID := mustInsertID(t, d.db, `INSERT INTO maps(campaign_id, name) VALUES (?, 'map')`, campaignID)

	characterTokenID := mustInsertID(t, d.db, `INSERT INTO map_tokens(map_id, entity_type, entity_id, x, y) VALUES (?, 'character', ?, .2, .2)`, mapID, characterID)
	npcTokenID := mustInsertID(t, d.db, `INSERT INTO map_tokens(map_id, entity_type, entity_id, x, y) VALUES (?, 'npc', ?, .3, .3)`, mapID, npcID)

	_, err = d.db.Exec(`INSERT INTO map_tokens(map_id, entity_type, entity_id, x, y) VALUES (?, 'character', 999999, .4, .4)`, mapID)
	require.ErrorContains(t, err, "map token character does not exist")
	_, err = d.db.Exec(`INSERT INTO map_tokens(map_id, entity_type, entity_id, x, y) VALUES (?, 'npc', 999999, .4, .4)`, mapID)
	require.ErrorContains(t, err, "map token npc does not exist")
	_, err = d.db.Exec(`UPDATE map_tokens SET entity_id = 999999 WHERE id = ?`, characterTokenID)
	require.ErrorContains(t, err, "map token character does not exist")
	_, err = d.db.Exec(`UPDATE map_tokens SET entity_type = 'npc', entity_id = 999999 WHERE id = ?`, characterTokenID)
	require.ErrorContains(t, err, "map token npc does not exist")

	_, err = d.db.Exec(`DELETE FROM characters WHERE id = ?`, characterID)
	require.NoError(t, err)
	assertZeroCountWhere(t, d.db, "map_tokens", "id = ?", characterTokenID)
	_, err = d.db.Exec(`DELETE FROM session_npcs WHERE id = ?`, npcID)
	require.NoError(t, err)
	assertZeroCountWhere(t, d.db, "map_tokens", "id = ?", npcTokenID)
	require.NoError(t, foreignKeyCheck(d.db))
}

func TestTemplateRulesetRemovedWithoutDeletingCustomRulesets(t *testing.T) {
	d, err := Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, d.Close()) })

	var placeholderCount int
	require.NoError(t, d.db.QueryRow(`SELECT count(*) FROM rulesets WHERE name = 'my_ruleset'`).Scan(&placeholderCount))
	assert.Zero(t, placeholderCount)

	_, err = d.CreateRuleset("my_real_ruleset", `[{"key":"focus"}]`, "2.0")
	require.NoError(t, err)
	var customCount int
	require.NoError(t, d.db.QueryRow(`SELECT count(*) FROM rulesets WHERE name = 'my_real_ruleset'`).Scan(&customCount))
	assert.Equal(t, 1, customCount)

	template, err := os.ReadFile(filepath.Join("..", "..", "docs", "ruleset-template.sql"))
	require.NoError(t, err)
	assert.Contains(t, string(template), "INSERT OR IGNORE INTO rulesets")
}

func TestIntegrityMigrationPreservesAutoincrementHighWaterMarks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sequences.db")
	seed, err := sql.Open("sqlite", sqliteDSN(path))
	require.NoError(t, err)
	seed.SetMaxOpenConns(1)
	_, err = runMigrationsFromFS(seed, migrationsThrough055(t), "migrations", migrationRunOptions{databasePath: path})
	require.NoError(t, err)

	var rulesetID int64
	require.NoError(t, seed.QueryRow(`SELECT id FROM rulesets WHERE name = 'dnd5e'`).Scan(&rulesetID))
	_, err = seed.Exec(`
		INSERT INTO campaigns(id, ruleset_id, name) VALUES (100, ?, 'sequence root');
		INSERT INTO characters(id, campaign_id, name) VALUES (101, 100, 'sequence hero');
		INSERT INTO sessions(id, campaign_id, title, date) VALUES (102, 100, 'sequence session', 'x');
		INSERT INTO combat_encounters(id, session_id, name) VALUES (103, 102, 'sequence encounter');
		INSERT INTO maps(id, campaign_id, name) VALUES (104, 100, 'sequence map');

		INSERT INTO campaigns(id, ruleset_id, name) VALUES (60001, ?, 'deleted campaign');
		INSERT INTO characters(id, campaign_id, name) VALUES (60002, 100, 'deleted character');
		INSERT INTO sessions(id, campaign_id, title, date) VALUES (60003, 100, 'deleted session', 'x');
		INSERT INTO messages(id, session_id, role, content) VALUES (60004, 102, 'user', 'deleted message');
		INSERT INTO combat_encounters(id, session_id, name) VALUES (60005, 102, 'deleted encounter');
		INSERT INTO combatants(id, encounter_id, name) VALUES (60006, 103, 'deleted combatant');
		INSERT INTO world_notes(id, campaign_id, title) VALUES (60007, 100, 'deleted note');
		INSERT INTO maps(id, campaign_id, name) VALUES (60008, 100, 'deleted map');
		INSERT INTO map_pins(id, map_id, x, y) VALUES (60009, 104, .5, .5);
		INSERT INTO dice_rolls(id, session_id, expression, result) VALUES (60010, 102, '1d20', 1);
		INSERT INTO rulebook_chunks(id, ruleset_id, heading, content) VALUES (60011, ?, 'deleted rules', 'body');
		INSERT INTO session_npcs(id, session_id, name) VALUES (60012, 102, 'deleted npc');
		INSERT INTO objectives(id, campaign_id, title) VALUES (60013, 100, 'deleted objective');
		INSERT INTO items(id, character_id, name) VALUES (60014, 101, 'deleted item');
		INSERT INTO secrets(id, campaign_id, title, content) VALUES (60015, 100, 'deleted secret', 'body');
		INSERT INTO calendar_events(id, campaign_id, title) VALUES (60016, 100, 'deleted event');

		DELETE FROM calendar_events WHERE id = 60016;
		DELETE FROM secrets WHERE id = 60015;
		DELETE FROM items WHERE id = 60014;
		DELETE FROM objectives WHERE id = 60013;
		DELETE FROM session_npcs WHERE id = 60012;
		DELETE FROM rulebook_chunks WHERE id = 60011;
		DELETE FROM dice_rolls WHERE id = 60010;
		DELETE FROM map_pins WHERE id = 60009;
		DELETE FROM maps WHERE id = 60008;
		DELETE FROM world_notes WHERE id = 60007;
		DELETE FROM combatants WHERE id = 60006;
		DELETE FROM combat_encounters WHERE id = 60005;
		DELETE FROM messages WHERE id = 60004;
		DELETE FROM sessions WHERE id = 60003;
		DELETE FROM characters WHERE id = 60002;
		DELETE FROM campaigns WHERE id = 60001;
	`, rulesetID, rulesetID, rulesetID)
	require.NoError(t, err)
	require.NoError(t, seed.Close())

	d, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, d.Close()) })
	wantSequences := map[string]int64{
		"campaigns": 60001, "characters": 60002, "sessions": 60003,
		"messages": 60004, "combat_encounters": 60005, "combatants": 60006,
		"world_notes": 60007, "maps": 60008, "map_pins": 60009,
		"dice_rolls": 60010, "rulebook_chunks": 60011, "session_npcs": 60012,
		"objectives": 60013, "items": 60014, "secrets": 60015,
		"calendar_events": 60016,
	}
	for table, want := range wantSequences {
		var got int64
		require.NoError(t, d.db.QueryRow(`SELECT seq FROM sqlite_sequence WHERE name = ?`, table).Scan(&got), table)
		assert.GreaterOrEqual(t, got, want, table)
	}
	nextCharacterID, err := d.CreateCharacter(100, "after repair")
	require.NoError(t, err)
	assert.Greater(t, nextCharacterID, int64(60002))
}

func TestIntegrityMigrationRewritesLegacyMapURLsWithinCampaignOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "map-links.db")
	seed, err := sql.Open("sqlite", sqliteDSN(path))
	require.NoError(t, err)
	seed.SetMaxOpenConns(1)
	_, err = runMigrationsFromFS(seed, migrationsThrough055(t), "migrations", migrationRunOptions{databasePath: path})
	require.NoError(t, err)
	var rulesetID int64
	require.NoError(t, seed.QueryRow(`SELECT id FROM rulesets WHERE name = 'dnd5e'`).Scan(&rulesetID))

	campaignA := mustInsertID(t, seed, `INSERT INTO campaigns(ruleset_id, name) VALUES (?, 'A')`, rulesetID)
	campaignB := mustInsertID(t, seed, `INSERT INTO campaigns(ruleset_id, name) VALUES (?, 'B')`, rulesetID)
	campaignC := mustInsertID(t, seed, `INSERT INTO campaigns(ruleset_id, name) VALUES (?, 'C')`, rulesetID)
	campaignD := mustInsertID(t, seed, `INSERT INTO campaigns(ruleset_id, name) VALUES (?, 'D')`, rulesetID)
	sessionA := mustInsertID(t, seed, `INSERT INTO sessions(campaign_id, title, date) VALUES (?, 'A', 'x')`, campaignA)
	sessionB := mustInsertID(t, seed, `INSERT INTO sessions(campaign_id, title, date) VALUES (?, 'B', 'x')`, campaignB)
	sessionC := mustInsertID(t, seed, `INSERT INTO sessions(campaign_id, title, date) VALUES (?, 'C', 'x')`, campaignC)
	sessionD := mustInsertID(t, seed, `INSERT INTO sessions(campaign_id, title, date) VALUES (?, 'D', 'x')`, campaignD)
	sharedA := mustInsertID(t, seed, `INSERT INTO maps(campaign_id, name, image_path) VALUES (?, 'shared A', 'maps/shared.svg')`, campaignA)
	firstA := mustInsertID(t, seed, `INSERT INTO maps(campaign_id, name, image_path) VALUES (?, 'first A', 'maps/first.svg')`, campaignA)
	secondA := mustInsertID(t, seed, `INSERT INTO maps(campaign_id, name, image_path) VALUES (?, 'second A', 'maps/second.svg')`, campaignA)
	shortA := mustInsertID(t, seed, `INSERT INTO maps(campaign_id, name, image_path) VALUES (?, 'short overlap', 'maps/overlap')`, campaignA)
	longA := mustInsertID(t, seed, `INSERT INTO maps(campaign_id, name, image_path) VALUES (?, 'long overlap', 'maps/overlap.svg')`, campaignA)
	sharedB := mustInsertID(t, seed, `INSERT INTO maps(campaign_id, name, image_path) VALUES (?, 'shared B', 'maps/shared.svg')`, campaignB)
	mustInsertID(t, seed, `INSERT INTO maps(campaign_id, name, image_path) VALUES (?, 'ambiguous one', 'maps/ambiguous.svg')`, campaignC)
	mustInsertID(t, seed, `INSERT INTO maps(campaign_id, name, image_path) VALUES (?, 'ambiguous two', 'maps/ambiguous.svg')`, campaignC)
	mustInsertID(t, seed, `INSERT INTO maps(campaign_id, name, image_path) VALUES (?, 'short only', 'maps/overlap')`, campaignD)
	mustInsertID(t, seed, `INSERT INTO messages(session_id, role, content) VALUES (?, 'assistant', 'A /api/files/maps/shared.svg /api/files/maps/first.svg /api/files/maps/second.svg /api/files/maps/overlap.svg /api/files/maps/overlap')`, sessionA)
	mustInsertID(t, seed, `INSERT INTO messages(session_id, role, content) VALUES (?, 'assistant', 'B /api/files/maps/shared.svg')`, sessionB)
	mustInsertID(t, seed, `INSERT INTO messages(session_id, role, content) VALUES (?, 'assistant', 'C /api/files/maps/ambiguous.svg')`, sessionC)
	mustInsertID(t, seed, `INSERT INTO messages(session_id, role, content) VALUES (?, 'assistant', 'D /api/files/maps/overlap.svg')`, sessionD)
	require.NoError(t, seed.Close())

	d, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, d.Close()) })
	var gotA, gotB, gotC, gotD string
	require.NoError(t, d.db.QueryRow(`SELECT content FROM messages WHERE session_id = ?`, sessionA).Scan(&gotA))
	require.NoError(t, d.db.QueryRow(`SELECT content FROM messages WHERE session_id = ?`, sessionB).Scan(&gotB))
	require.NoError(t, d.db.QueryRow(`SELECT content FROM messages WHERE session_id = ?`, sessionC).Scan(&gotC))
	require.NoError(t, d.db.QueryRow(`SELECT content FROM messages WHERE session_id = ?`, sessionD).Scan(&gotD))
	assert.Equal(t, fmt.Sprintf("A /api/assets/maps/%d /api/assets/maps/%d /api/assets/maps/%d /api/assets/maps/%d /api/assets/maps/%d", sharedA, firstA, secondA, longA, shortA), gotA)
	assert.Equal(t, fmt.Sprintf("B /api/assets/maps/%d", sharedB), gotB)
	assert.Equal(t, "C /api/files/maps/ambiguous.svg", gotC)
	assert.Equal(t, "D /api/files/maps/overlap.svg", gotD)
}

func TestIntegrityMigrationQuarantinesOrphansAndRewritesLegacyMapURLs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orphaned.db")
	preRepairFS := migrationsThrough055(t)
	seed, err := sql.Open("sqlite", sqliteDSN(path))
	require.NoError(t, err)
	seed.SetMaxOpenConns(1)
	_, err = runMigrationsFromFS(seed, preRepairFS, "migrations", migrationRunOptions{databasePath: path})
	require.NoError(t, err)

	_, err = seed.Exec(`PRAGMA foreign_keys=OFF`)
	require.NoError(t, err)
	var rulesetID int64
	require.NoError(t, seed.QueryRow(`SELECT id FROM rulesets WHERE name = 'dnd5e'`).Scan(&rulesetID))
	var templateRulesetID int64
	require.NoError(t, seed.QueryRow(`SELECT id FROM rulesets WHERE name = 'my_ruleset'`).Scan(&templateRulesetID))
	_, err = seed.Exec(`
		INSERT INTO campaigns(id, ruleset_id, name, chronicle_night, in_game_year, gm_notes) VALUES (9800, 999999, 'orphan campaign', 7, 42, 'recover me');
		INSERT INTO rulebook_chunks(id, ruleset_id, heading, content, source, embedding) VALUES (9801, 999999, 'orphan rules', 'body', 'Lost Book', X'0102FF');
		INSERT INTO oracle_tables(id, ruleset_id, table_name, roll_min, roll_max, result) VALUES (9802, 999999, 'lost oracle', 1, 1, 'result');

		INSERT INTO campaigns(id, ruleset_id, name) VALUES (9700, ?, 'valid root');
		INSERT INTO characters(id, campaign_id, name) VALUES (9701, 9700, 'valid hero');
		INSERT INTO adventures(id, campaign_id, title) VALUES (9702, 9700, 'valid arc');
		INSERT INTO sessions(id, campaign_id, title, date, adventure_id) VALUES (9703, 9700, 'valid session', '2026-07-17', 9702);
		INSERT INTO combat_encounters(id, session_id, name) VALUES (9704, 9703, 'valid encounter');
		INSERT INTO maps(id, campaign_id, name) VALUES (9705, 9700, 'valid map');
		INSERT INTO decks(id, campaign_id, name) VALUES (9706, 9700, 'valid deck');

		INSERT INTO characters(id, campaign_id, name) VALUES (9803, 999999, 'orphan character');
		INSERT INTO sessions(id, campaign_id, title, date) VALUES (9804, 999999, 'orphan session', 'x');
		INSERT INTO world_notes(id, campaign_id, title) VALUES (9805, 999999, 'orphan note');
		INSERT INTO maps(id, campaign_id, name) VALUES (9806, 999999, 'orphan map');
		INSERT INTO objectives(id, campaign_id, title) VALUES (9807, 999999, 'orphan objective');
		INSERT INTO relationships(id, campaign_id, from_name, to_name) VALUES (9808, 999999, 'a', 'b');
		INSERT INTO factions(id, campaign_id, name) VALUES (9809, 999999, 'orphan faction');
		INSERT INTO adventures(id, campaign_id, title) VALUES (9810, 999999, 'orphan adventure');
		INSERT INTO npc_stats(id, campaign_id, name) VALUES (9811, 999999, 'orphan npc stat');
		INSERT INTO secrets(id, campaign_id, title, content) VALUES (9812, 999999, 'orphan secret', 'secret payload');
		INSERT INTO calendar_events(id, campaign_id, in_game_year, in_game_month, in_game_day, title) VALUES (9813, 999999, 88, 9, 10, 'orphan calendar');
		INSERT INTO decks(id, campaign_id, name) VALUES (9814, 999999, 'orphan deck');

		INSERT INTO items(id, character_id, name) VALUES (9815, 999999, 'orphan item');
		INSERT INTO character_macros(id, character_id, label, action_text) VALUES (9816, 999999, 'orphan macro', 'act');
		INSERT INTO messages(id, session_id, role, content, character_id) VALUES (9817, 9703, 'user', 'optional character', 999999);
		INSERT INTO combatants(id, encounter_id, character_id, name, damage_aggravated, willpower_superficial, hunger) VALUES (9818, 9704, 999999, 'optional character', 2, 3, 4);
		INSERT INTO sessions(id, campaign_id, title, date, adventure_id, tension_level, scene_tags, masquerade_integrity) VALUES (9819, 9700, 'optional adventure', 'x', 999999, 9, 'crypt', 2);

		INSERT INTO messages(id, session_id, role, content) VALUES (9820, 999999, 'user', 'orphan session message');
		INSERT INTO session_npcs(id, session_id, name) VALUES (9821, 999999, 'orphan npc');
		INSERT INTO combat_encounters(id, session_id, name) VALUES (9822, 999999, 'orphan encounter');
		INSERT INTO dice_rolls(id, session_id, expression, result) VALUES (9823, 999999, '1d20', 1);
		INSERT INTO xp_log(id, session_id, note) VALUES (9824, 999999, 'orphan xp');
		INSERT INTO secrets(id, campaign_id, title, content, revealed_at_session_id) VALUES (9825, 9700, 'optional session', 'keep', 999999);
		INSERT INTO calendar_events(id, campaign_id, title, session_id) VALUES (9826, 9700, 'optional session', 999999);
		INSERT INTO deck_draws(id, session_id, deck_id, card_json) VALUES (9827, 999999, 9706, '{}');
		INSERT INTO deck_draws(id, session_id, deck_id, card_json) VALUES (9828, 9703, 999999, '{}');
		INSERT INTO combatants(id, encounter_id, name) VALUES (9829, 999999, 'orphan encounter combatant');
		INSERT INTO map_pins(id, map_id, x, y, label) VALUES (9830, 999999, .5, .5, 'orphan pin');
		INSERT INTO map_tokens(id, map_id, entity_type, entity_id, x, y) VALUES (9831, 999999, 'npc', 1, .5, .5);
		INSERT INTO map_zones(id, map_id, name, x, y, width, height) VALUES (9832, 999999, 'orphan zone', 0, 0, 1, 1);
		INSERT INTO objectives(id, campaign_id, title, parent_id) VALUES (9833, 9700, 'orphan parent', 999999);
		INSERT INTO map_tokens(id, map_id, entity_type, entity_id, x, y) VALUES (9835, 9705, 'character', 999999, .5, .5);
		INSERT INTO map_tokens(id, map_id, entity_type, entity_id, x, y) VALUES (9836, 9705, 'npc', 999999, .5, .5);
		INSERT INTO objectives(id, campaign_id, title, description, parent_id) VALUES (9840, 9700, 'missing root', 'root payload', 999999);
		INSERT INTO objectives(id, campaign_id, title, description, parent_id) VALUES (9841, 9700, 'orphan child', 'child payload', 9840);
		INSERT INTO objectives(id, campaign_id, title, description, parent_id) VALUES (9842, 9700, 'orphan grandchild', 'grandchild payload', 9841);
	`, rulesetID)
	require.NoError(t, err)
	_, err = seed.Exec(`INSERT INTO campaigns(id, ruleset_id, name) VALUES (9834, ?, 'uses historical template')`, templateRulesetID)
	require.NoError(t, err)

	campaignID := mustInsertID(t, seed, `INSERT INTO campaigns(ruleset_id, name) VALUES (?, 'legacy url')`, rulesetID)
	sessionID := mustInsertID(t, seed, `INSERT INTO sessions(campaign_id, title, date) VALUES (?, 'history', '2026-07-17')`, campaignID)
	mapID := mustInsertID(t, seed, `INSERT INTO maps(campaign_id, name, image_path) VALUES (?, 'old map', 'maps/old.svg')`, campaignID)
	mustInsertID(t, seed, `INSERT INTO messages(session_id, role, content) VALUES (?, 'assistant', 'See /api/files/maps/old.svg and keep text.')`, sessionID)
	require.NoError(t, seed.Close())

	d, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, d.Close()) })
	require.NoError(t, foreignKeyCheck(d.db))

	requiredOrphans := []struct {
		table string
		id    int64
	}{
		{"campaigns", 9800}, {"rulebook_chunks", 9801},
		{"characters", 9803}, {"sessions", 9804}, {"world_notes", 9805},
		{"maps", 9806}, {"objectives", 9807}, {"relationships", 9808},
		{"factions", 9809}, {"adventures", 9810}, {"npc_stats", 9811},
		{"secrets", 9812}, {"calendar_events", 9813}, {"decks", 9814},
		{"items", 9815}, {"character_macros", 9816},
		{"messages", 9820}, {"session_npcs", 9821}, {"combat_encounters", 9822},
		{"dice_rolls", 9823}, {"xp_log", 9824}, {"deck_draws", 9827},
		{"deck_draws", 9828}, {"combatants", 9829}, {"map_pins", 9830},
		{"map_tokens", 9831}, {"map_zones", 9832}, {"objectives", 9833},
		{"map_tokens", 9835}, {"map_tokens", 9836},
		{"objectives", 9840}, {"objectives", 9841}, {"objectives", 9842},
	}
	for _, orphan := range requiredOrphans {
		var payload, reason string
		require.NoError(t, d.db.QueryRow(`SELECT payload_json, reason FROM orphaned_records WHERE source_table = ? AND source_id = ?`, orphan.table, orphan.id).Scan(&payload, &reason))
		var decoded map[string]any
		require.NoError(t, json.Unmarshal([]byte(payload), &decoded))
		assert.Equal(t, float64(orphan.id), decoded["id"])
		assert.Contains(t, reason, "missing")
		assertZeroCountWhere(t, d.db, orphan.table, "id = ?", orphan.id)
	}

	optionalOrphans := []struct {
		table  string
		id     int64
		column string
	}{
		{"oracle_tables", 9802, "ruleset_id"},
		{"messages", 9817, "character_id"},
		{"combatants", 9818, "character_id"},
		{"sessions", 9819, "adventure_id"},
		{"secrets", 9825, "revealed_at_session_id"},
		{"calendar_events", 9826, "session_id"},
	}
	for _, orphan := range optionalOrphans {
		var payload, reason string
		require.NoError(t, d.db.QueryRow(`SELECT payload_json, reason FROM orphaned_records WHERE source_table = ? AND source_id = ?`, orphan.table, orphan.id).Scan(&payload, &reason))
		assert.Contains(t, reason, "reference cleared")
		var decoded map[string]any
		require.NoError(t, json.Unmarshal([]byte(payload), &decoded))
		assert.Equal(t, float64(orphan.id), decoded["id"])
		var nonNull int
		require.NoError(t, d.db.QueryRow(`SELECT count(*) FROM `+orphan.table+` WHERE id = ? AND `+orphan.column+` IS NOT NULL`, orphan.id).Scan(&nonNull))
		assert.Zero(t, nonNull)
	}

	var campaignPayload map[string]any
	var campaignPayloadJSON string
	require.NoError(t, d.db.QueryRow(`SELECT payload_json FROM orphaned_records WHERE source_table = 'campaigns' AND source_id = 9800`).Scan(&campaignPayloadJSON))
	require.NoError(t, json.Unmarshal([]byte(campaignPayloadJSON), &campaignPayload))
	assert.Equal(t, float64(7), campaignPayload["chronicle_night"])
	assert.Equal(t, float64(42), campaignPayload["in_game_year"])
	assert.Equal(t, "recover me", campaignPayload["gm_notes"])

	var rulebookPayload map[string]any
	var rulebookPayloadJSON string
	require.NoError(t, d.db.QueryRow(`SELECT payload_json FROM orphaned_records WHERE source_table = 'rulebook_chunks' AND source_id = 9801`).Scan(&rulebookPayloadJSON))
	require.NoError(t, json.Unmarshal([]byte(rulebookPayloadJSON), &rulebookPayload))
	assert.Equal(t, "0102FF", rulebookPayload["embedding_hex"])
	assert.Equal(t, "hex", rulebookPayload["embedding_encoding"])

	var sessionPayload map[string]any
	var sessionPayloadJSON string
	require.NoError(t, d.db.QueryRow(`SELECT payload_json FROM orphaned_records WHERE source_table = 'sessions' AND source_id = 9819`).Scan(&sessionPayloadJSON))
	require.NoError(t, json.Unmarshal([]byte(sessionPayloadJSON), &sessionPayload))
	assert.Equal(t, float64(9), sessionPayload["tension_level"])
	assert.Equal(t, "crypt", sessionPayload["scene_tags"])
	assert.Equal(t, float64(2), sessionPayload["masquerade_integrity"])

	var combatantPayload map[string]any
	var combatantPayloadJSON string
	require.NoError(t, d.db.QueryRow(`SELECT payload_json FROM orphaned_records WHERE source_table = 'combatants' AND source_id = 9818`).Scan(&combatantPayloadJSON))
	require.NoError(t, json.Unmarshal([]byte(combatantPayloadJSON), &combatantPayload))
	assert.Equal(t, float64(2), combatantPayload["damage_aggravated"])
	assert.Equal(t, float64(3), combatantPayload["willpower_superficial"])
	assert.Equal(t, float64(4), combatantPayload["hunger"])

	var calendarPayload map[string]any
	var calendarPayloadJSON string
	require.NoError(t, d.db.QueryRow(`SELECT payload_json FROM orphaned_records WHERE source_table = 'calendar_events' AND source_id = 9813`).Scan(&calendarPayloadJSON))
	require.NoError(t, json.Unmarshal([]byte(calendarPayloadJSON), &calendarPayload))
	assert.Equal(t, float64(88), calendarPayload["in_game_year"])
	assert.Equal(t, float64(9), calendarPayload["in_game_month"])
	assert.Equal(t, float64(10), calendarPayload["in_game_day"])

	var templateCount int
	require.NoError(t, d.db.QueryRow(`SELECT count(*) FROM rulesets WHERE id = ? AND name = 'my_ruleset'`, templateRulesetID).Scan(&templateCount))
	assert.Equal(t, 1, templateCount, "referenced historical template row must be preserved")

	var content string
	require.NoError(t, d.db.QueryRow(`SELECT content FROM messages WHERE session_id = ?`, sessionID).Scan(&content))
	assert.Equal(t, fmt.Sprintf("See /api/assets/maps/%d and keep text.", mapID), content)
}

func TestIntegrityIndexesServeHighFrequencyQueries(t *testing.T) {
	d, err := Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, d.Close()) })

	queries := []struct {
		name  string
		query string
		args  []any
		index string
		avoid string
	}{
		{name: "messages session order", query: `SELECT id FROM messages WHERE session_id = ? ORDER BY created_at, id`, args: []any{1}, index: "idx_messages_session_order"},
		{name: "sessions campaign", query: `SELECT id FROM sessions WHERE campaign_id = ? ORDER BY date DESC`, args: []any{1}, index: "idx_sessions_campaign_date"},
		{name: "characters campaign", query: `SELECT id FROM characters WHERE campaign_id = ? ORDER BY name`, args: []any{1}, index: "idx_characters_campaign_name"},
		{
			name: "objectives production list",
			query: `SELECT id, campaign_id, title, description, status, parent_id, created_at
				FROM objectives WHERE campaign_id = ? ORDER BY created_at DESC`,
			args: []any{1}, index: "idx_objectives_campaign_created", avoid: "TEMP B-TREE",
		},
		{name: "maps campaign", query: `SELECT id FROM maps WHERE campaign_id = ? ORDER BY created_at`, args: []any{1}, index: "idx_maps_campaign_created"},
		{name: "map pins", query: `SELECT id FROM map_pins WHERE map_id = ?`, args: []any{1}, index: "idx_map_pins_map"},
		{name: "map tokens", query: `SELECT id FROM map_tokens WHERE map_id = ?`, args: []any{1}, index: "sqlite_autoindex_map_tokens_1"},
		{name: "map zones", query: `SELECT id FROM map_zones WHERE map_id = ?`, args: []any{1}, index: "idx_map_zones_map"},
		{name: "combatant order", query: `SELECT id FROM combatants WHERE encounter_id = ? ORDER BY sort_order, id`, args: []any{1}, index: "idx_combatants_encounter_order"},
	}
	for _, tt := range queries {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := d.db.Query("EXPLAIN QUERY PLAN "+tt.query, tt.args...)
			require.NoError(t, err)
			defer rows.Close()
			var details []string
			for rows.Next() {
				var id, parent, unused int
				var detail string
				require.NoError(t, rows.Scan(&id, &parent, &unused, &detail))
				details = append(details, detail)
			}
			require.NoError(t, rows.Err())
			plan := strings.Join(details, "\n")
			assert.Contains(t, plan, tt.index)
			if tt.avoid != "" {
				assert.NotContains(t, plan, tt.avoid)
			}
		})
	}
}

func TestRepairMigrationTemporarilyDisablesForeignKeysAndRestoresThem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repair.db")
	db, err := sql.Open("sqlite", sqliteDSN(path))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	db.SetMaxOpenConns(1)

	migrations := fstest.MapFS{
		"migrations/001_base.sql": {Data: []byte(`
			CREATE TABLE parents(id INTEGER PRIMARY KEY);
			CREATE TABLE children(id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES parents(id));
			INSERT INTO parents VALUES (1); INSERT INTO children VALUES (1, 1);
		`)},
		"migrations/002_repair.sql": {Data: []byte(`
			CREATE TABLE parents_new(id INTEGER PRIMARY KEY);
			INSERT INTO parents_new SELECT * FROM parents;
			DROP TABLE parents;
			ALTER TABLE parents_new RENAME TO parents;
		`)},
	}
	_, err = runMigrationsFromFS(db, migrations, "migrations", migrationRunOptions{
		databasePath:       path,
		backupBeforeRepair: true,
		repairMigrations:   map[string]struct{}{"002_repair.sql": {}},
		foreignKeyRebuilds: map[string]struct{}{"002_repair.sql": {}},
	})
	require.NoError(t, err)
	require.NoError(t, foreignKeyCheck(db))
	var enabled int
	require.NoError(t, db.QueryRow(`PRAGMA foreign_keys`).Scan(&enabled))
	assert.Equal(t, 1, enabled)
}

func TestRepairMigrationRestoresForeignKeysAfterScriptAndLedgerFailures(t *testing.T) {
	tests := []struct {
		name       string
		baseScript string
		repair     string
	}{
		{
			name:       "script failure",
			baseScript: `CREATE TABLE parents(id INTEGER PRIMARY KEY);`,
			repair:     `DROP TABLE parents; SELECT * FROM missing_table;`,
		},
		{
			name: "ledger failure",
			baseScript: `
				CREATE TABLE parents(id INTEGER PRIMARY KEY);
				CREATE TRIGGER reject_repair_ledger BEFORE INSERT ON schema_migrations
				WHEN NEW.version = '002_repair' BEGIN SELECT RAISE(FAIL, 'ledger rejected'); END;
			`,
			repair: `DROP TABLE parents; CREATE TABLE parents(id INTEGER PRIMARY KEY);`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "repair.db")
			db, err := sql.Open("sqlite", sqliteDSN(path))
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, db.Close()) })
			db.SetMaxOpenConns(1)

			base := fstest.MapFS{"migrations/001_base.sql": {Data: []byte(tt.baseScript)}}
			_, err = runMigrationsFromFS(db, base, "migrations", migrationRunOptions{databasePath: path})
			require.NoError(t, err)
			all := fstest.MapFS{
				"migrations/001_base.sql":   {Data: []byte(tt.baseScript)},
				"migrations/002_repair.sql": {Data: []byte(tt.repair)},
			}
			_, err = runMigrationsFromFS(db, all, "migrations", migrationRunOptions{
				databasePath:       path,
				backupBeforeRepair: true,
				repairMigrations:   map[string]struct{}{"002_repair.sql": {}},
				foreignKeyRebuilds: map[string]struct{}{"002_repair.sql": {}},
			})
			require.Error(t, err)

			var enabled int
			require.NoError(t, db.QueryRow(`PRAGMA foreign_keys`).Scan(&enabled))
			assert.Equal(t, 1, enabled)
			var parentCount int
			require.NoError(t, db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'parents'`).Scan(&parentCount))
			assert.Equal(t, 1, parentCount, "failed repair must roll back its schema changes")
		})
	}
}

func migrationsThrough055(t *testing.T) fs.FS {
	t.Helper()
	files := fstest.MapFS{}
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	require.NoError(t, err)
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") || entry.Name() >= "056_" {
			continue
		}
		contents, err := fs.ReadFile(migrationsFS, "migrations/"+entry.Name())
		require.NoError(t, err)
		files["migrations/"+entry.Name()] = &fstest.MapFile{Data: contents}
	}
	return files
}

func mustInsertID(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	result, err := db.Exec(query, args...)
	require.NoError(t, err)
	id, err := result.LastInsertId()
	require.NoError(t, err)
	return id
}

func assertZeroCount(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	assertZeroCountWhere(t, db, table, "1 = 1")
}

func assertZeroCountWhere(t *testing.T, db *sql.DB, table, where string, args ...any) {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM `+table+` WHERE `+where, args...).Scan(&count))
	assert.Zero(t, count, table)
}

func assertNullReference(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()
	var nonNull int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM `+table+` WHERE `+column+` IS NOT NULL`).Scan(&nonNull))
	assert.Zero(t, nonNull, "%s.%s", table, column)
}
