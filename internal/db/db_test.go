package db

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen_CreatesTablesOnFirstRun(t *testing.T) {
	d, err := Open(":memory:")
	require.NoError(t, err)
	defer d.Close()

	tables := []string{
		"settings", "rulesets", "campaigns", "characters",
		"sessions", "messages", "combat_encounters", "combatants",
		"world_notes", "maps", "map_pins", "dice_rolls",
	}
	for _, table := range tables {
		var name string
		err := d.db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		require.NoError(t, err, "table %s should exist", table)
	}
}

func TestOpen_IdempotentMigrations(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "test.db")
	d1, err := Open(tmp)
	require.NoError(t, err)
	d1.Close()

	d2, err := Open(tmp)
	require.NoError(t, err)
	defer d2.Close()
	// Reaching here means migrations ran twice without error (idempotent)
}

func TestRulesets_SeededByMigration(t *testing.T) {
	d := newTestDB(t)
	list, err := d.ListRulesets()
	require.NoError(t, err)
	names := make([]string, len(list))
	for i, r := range list {
		names[i] = r.Name
	}
	assert.ElementsMatch(t, []string{"dnd5e", "ironsworn", "vtm", "coc", "cyberpunk", "shadowrun", "wfrp", "starwars", "l5r", "theonering", "wrath_glory", "blades", "paranoia", "my_ruleset", "dune"}, names)
}
