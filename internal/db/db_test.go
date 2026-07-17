package db

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

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

func TestMigrationRunnerExecutesWholeTransactionalScript(t *testing.T) {
	db, err := sql.Open("sqlite", sqliteDSN(":memory:"))
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)

	migrationFiles := fstest.MapFS{
		"migrations/001_script.sql": {Data: []byte(`
			CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT NOT NULL);
			CREATE TABLE note_events (body TEXT NOT NULL);
			CREATE TRIGGER capture_note AFTER INSERT ON notes BEGIN
				INSERT INTO note_events (body) VALUES ('trigger;' || NEW.body);
				UPDATE notes SET body = body WHERE id = NEW.id;
			END;
			INSERT INTO notes (body) VALUES ('quoted;value');
		`)},
	}
	_, err = runMigrationsFromFS(db, migrationFiles, "migrations", migrationRunOptions{})
	require.NoError(t, err)

	var note, event string
	require.NoError(t, db.QueryRow("SELECT body FROM notes").Scan(&note))
	require.NoError(t, db.QueryRow("SELECT body FROM note_events").Scan(&event))
	assert.Equal(t, "quoted;value", note)
	assert.Equal(t, "trigger;quoted;value", event)
}

func TestMigrationRunnerRollsBackWholeScriptOnFailure(t *testing.T) {
	db, err := sql.Open("sqlite", sqliteDSN(":memory:"))
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)

	migrationFiles := fstest.MapFS{
		"migrations/001_broken.sql": {Data: []byte(`
			CREATE TABLE should_rollback (id INTEGER PRIMARY KEY);
			INSERT INTO missing_table VALUES (1);
		`)},
	}
	_, err = runMigrationsFromFS(db, migrationFiles, "migrations", migrationRunOptions{})
	require.Error(t, err)

	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'should_rollback'").Scan(&count))
	assert.Zero(t, count)
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = '001_broken'").Scan(&count))
	assert.Zero(t, count)
}

func TestMigrationRunnerBacksUpOnlyPendingRegisteredRepair(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "campaign.db")
	db, err := sql.Open("sqlite", sqliteDSN(path))
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)

	additive := fstest.MapFS{
		"migrations/001_additive.sql": {Data: []byte("CREATE TABLE records (value TEXT NOT NULL); INSERT INTO records VALUES ('before repair');")},
	}
	opts := migrationRunOptions{
		databasePath:       path,
		backupBeforeRepair: true,
		repairMigrations:   map[string]struct{}{"002_repair.sql": {}},
	}
	_, err = runMigrationsFromFS(db, additive, "migrations", opts)
	require.NoError(t, err)
	assert.Empty(t, backupFiles(t, path))

	withRepair := fstest.MapFS{
		"migrations/001_additive.sql": additive["migrations/001_additive.sql"],
		"migrations/002_repair.sql":   {Data: []byte("UPDATE records SET value = 'after repair';")},
	}
	_, err = runMigrationsFromFS(db, withRepair, "migrations", opts)
	require.NoError(t, err)
	backups := backupFiles(t, path)
	require.Len(t, backups, 1)
	_, err = runMigrationsFromFS(db, withRepair, "migrations", opts)
	require.NoError(t, err)
	assert.Len(t, backupFiles(t, path), 1, "an already-applied repair must not create another backup")

	backup, err := sql.Open("sqlite", sqliteDSN(backups[0]))
	require.NoError(t, err)
	defer backup.Close()
	require.NoError(t, integrityCheck(backup))
	var value string
	require.NoError(t, backup.QueryRow("SELECT value FROM records").Scan(&value))
	assert.Equal(t, "before repair", value)
}

func TestMigrationRepairErrorIncludesValidatedBackupPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "campaign.db")
	db, err := sql.Open("sqlite", sqliteDSN(path))
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)

	migrationFiles := fstest.MapFS{
		"migrations/001_base.sql":   {Data: []byte("CREATE TABLE records (value TEXT NOT NULL);")},
		"migrations/002_repair.sql": {Data: []byte("THIS IS NOT SQL;")},
	}
	_, err = runMigrationsFromFS(db, migrationFiles, "migrations", migrationRunOptions{
		databasePath:       path,
		backupBeforeRepair: true,
		repairMigrations:   map[string]struct{}{"002_repair.sql": {}},
	})
	require.Error(t, err)
	backups := backupFiles(t, path)
	require.Len(t, backups, 1)
	assert.Contains(t, err.Error(), backups[0])

	backup, openErr := sql.Open("sqlite", sqliteDSN(backups[0]))
	require.NoError(t, openErr)
	defer backup.Close()
	require.NoError(t, integrityCheck(backup))
}

func TestMigrationRunnerRefusesFileRepairWithoutBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "campaign.db")
	db, err := sql.Open("sqlite", sqliteDSN(path))
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)

	migrationFiles := fstest.MapFS{
		"migrations/001_repair.sql": {Data: []byte("CREATE TABLE repaired (id INTEGER PRIMARY KEY);")},
	}
	_, err = runMigrationsFromFS(db, migrationFiles, "migrations", migrationRunOptions{
		databasePath:     path,
		repairMigrations: map[string]struct{}{"001_repair.sql": {}},
	})
	require.ErrorContains(t, err, "requires a validated backup")
	assert.Empty(t, backupFiles(t, path))
}

func TestPostRepairForeignKeyFailureIncludesValidatedBackupPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "campaign.db")
	seed, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	_, err = seed.Exec(`
		CREATE TABLE parents (id INTEGER PRIMARY KEY);
		CREATE TABLE children (id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES parents(id));
		INSERT INTO children (id, parent_id) VALUES (1, 999);
	`)
	require.NoError(t, err)
	require.NoError(t, seed.Close())

	db, err := sql.Open("sqlite", sqliteDSN(path))
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)
	migrationFiles := fstest.MapFS{
		"migrations/001_repair.sql": {Data: []byte("SELECT 1;")},
	}
	backupPath, err := runMigrationsFromFS(db, migrationFiles, "migrations", migrationRunOptions{
		databasePath:       path,
		backupBeforeRepair: true,
		repairMigrations:   map[string]struct{}{"001_repair.sql": {}},
	})
	require.NoError(t, err)
	require.NotEmpty(t, backupPath)

	err = postMigrationForeignKeyCheck(db, backupPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), backupPath)
}

func backupFiles(t *testing.T, databasePath string) []string {
	t.Helper()
	matches, err := filepath.Glob(databasePath + ".backup-*")
	require.NoError(t, err)
	filtered := matches[:0]
	for _, match := range matches {
		if !strings.HasSuffix(match, ".tmp") {
			filtered = append(filtered, match)
		}
	}
	return filtered
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
