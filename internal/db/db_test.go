package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

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

func TestSQLiteFileURIPathShapes(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "POSIX special characters", path: "/tmp/Table Top/game?#.db", want: "file:///tmp/Table%20Top/game%3F%23.db"},
		{name: "POSIX backslash is data", path: `/tmp/Table\Top/game.db`, want: "file:///tmp/Table%5CTop/game.db"},
		{name: "Windows drive", path: `C:\Users\Table Top\game.db`, want: "file:///C:/Users/Table%20Top/game.db"},
		{name: "Windows UNC", path: `\\server\share\Table Top\game.db`, want: "file://server/share/Table%20Top/game.db"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sqliteFileURI(tt.path))
		})
	}
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
	var schemaMigrationTables int
	require.NoError(t, backup.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations'").Scan(&schemaMigrationTables))
	assert.Zero(t, schemaMigrationTables, "backup must precede migration-runner writes")
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
	require.Error(t, err)
	require.NotEmpty(t, backupPath)
	assert.Contains(t, err.Error(), backupPath)
}

func TestMigrationRunnerRetainsExclusiveLockThroughBackupMigrationsAndValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "campaign.db")
	db, err := sql.Open("sqlite", sqliteDSN(path))
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)

	base := fstest.MapFS{
		"migrations/001_base.sql": {Data: []byte("CREATE TABLE records (value TEXT NOT NULL); INSERT INTO records VALUES ('before repair');")},
	}
	_, err = runMigrationsFromFS(db, base, "migrations", migrationRunOptions{databasePath: path})
	require.NoError(t, err)

	writer, err := sql.Open("sqlite", sqliteDSN(path))
	require.NoError(t, err)
	defer writer.Close()
	writer.SetMaxOpenConns(1)
	_, err = writer.Exec("PRAGMA busy_timeout=0")
	require.NoError(t, err)
	var initial string
	require.NoError(t, writer.QueryRow("SELECT value FROM records").Scan(&initial))
	require.Equal(t, "before repair", initial)

	type pause struct {
		phase      string
		backupPath string
	}
	type result struct {
		backupPath string
		err        error
	}
	pauses := make(chan pause)
	resume := make(chan struct{})
	results := make(chan result, 1)
	withRepair := fstest.MapFS{
		"migrations/001_base.sql":   base["migrations/001_base.sql"],
		"migrations/002_repair.sql": {Data: []byte("UPDATE records SET value = 'after repair';")},
		"migrations/003_later.sql":  {Data: []byte("UPDATE records SET value = 'after later migration';")},
	}
	go func() {
		backupPath, runErr := runMigrationsFromFS(db, withRepair, "migrations", migrationRunOptions{
			databasePath:       path,
			backupBeforeRepair: true,
			repairMigrations:   map[string]struct{}{"002_repair.sql": {}},
			afterValidatedBackup: func(path string) {
				pauses <- pause{phase: "backup", backupPath: path}
				<-resume
			},
			afterMigration: func(filename string) {
				if filename == "002_repair.sql" || filename == "003_later.sql" {
					pauses <- pause{phase: filename}
					<-resume
				}
			},
		})
		results <- result{backupPath: backupPath, err: runErr}
	}()

	for _, wantPhase := range []string{"backup", "002_repair.sql", "003_later.sql"} {
		var got pause
		select {
		case got = <-pauses:
		case <-time.After(5 * time.Second):
			t.Fatal("migration runner did not reach pause")
		}
		_, writeErr := writer.Exec("UPDATE records SET value = 'intruder'")
		resume <- struct{}{}
		assert.Equal(t, wantPhase, got.phase)
		assert.Error(t, writeErr, "noncooperating writer committed during %s", got.phase)
		if got.phase == "backup" {
			backup, openErr := sql.Open("sqlite", sqliteDSN(got.backupPath))
			require.NoError(t, openErr)
			var value string
			require.NoError(t, backup.QueryRow("SELECT value FROM records").Scan(&value))
			require.NoError(t, backup.Close())
			assert.Equal(t, "before repair", value)
		}
	}

	var runResult result
	select {
	case runResult = <-results:
	case <-time.After(5 * time.Second):
		t.Fatal("migration runner did not finish")
	}
	require.NoError(t, runResult.err)
	require.NotEmpty(t, runResult.backupPath)

	var repaired string
	require.NoError(t, db.QueryRow("SELECT value FROM records").Scan(&repaired))
	assert.Equal(t, "after later migration", repaired)
	_, err = writer.Exec("UPDATE records SET value = 'writer after release'")
	require.NoError(t, err, "writer should commit after the migration connection releases its exclusive lock")
	require.NoError(t, db.QueryRow("SELECT value FROM records").Scan(&repaired))
	assert.Equal(t, "writer after release", repaired)
}

func TestBackupDatabaseStagesPrivatelyAndCleansUp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "campaign.db")
	d, err := Open(path)
	require.NoError(t, err)
	defer d.Close()

	var stageDir, stagedDatabase string
	backupPath, err := backupDatabaseWithHooks(d.SQL(), path, backupHooks{
		afterStageReady: func(dir, database string) error {
			stageDir, stagedDatabase = dir, database
			dirInfo, statErr := os.Stat(dir)
			require.NoError(t, statErr)
			fileInfo, statErr := os.Stat(database)
			require.NoError(t, statErr)
			assert.Equal(t, os.FileMode(0700), dirInfo.Mode().Perm())
			assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())
			return nil
		},
	})
	require.NoError(t, err)
	assert.FileExists(t, backupPath)
	assert.NoDirExists(t, stageDir)
	assert.NoFileExists(t, stagedDatabase)
}

func TestBackupDatabaseCleansPrivateStageAfterValidationFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "campaign.db")
	d, err := Open(path)
	require.NoError(t, err)
	defer d.Close()

	var stageDir string
	_, err = backupDatabaseWithHooks(d.SQL(), path, backupHooks{
		afterStageReady: func(dir, database string) error {
			stageDir = dir
			file, openErr := os.OpenFile(database, os.O_WRONLY, 0)
			if openErr != nil {
				return openErr
			}
			_, writeErr := file.WriteAt([]byte("not a sqlite db!"), 0)
			closeErr := file.Close()
			if writeErr != nil {
				return writeErr
			}
			return closeErr
		},
	})
	require.ErrorContains(t, err, "validate backup")
	assert.NoDirExists(t, stageDir)
	assert.Empty(t, backupFiles(t, path))
}

func TestBackupDatabasePublishCollisionPreservesExistingTarget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "campaign.db")
	d, err := Open(path)
	require.NoError(t, err)
	defer d.Close()

	var stageDir, collidedPath string
	_, err = backupDatabaseWithHooks(d.SQL(), path, backupHooks{
		afterStageReady: func(dir, _ string) error {
			stageDir = dir
			return nil
		},
		beforePublish: func(path string) error {
			collidedPath = path
			return os.WriteFile(path, []byte("existing backup sentinel"), 0600)
		},
	})
	require.ErrorContains(t, err, "publish validated backup")
	contents, readErr := os.ReadFile(collidedPath)
	require.NoError(t, readErr)
	assert.Equal(t, "existing backup sentinel", string(contents))
	assert.NoDirExists(t, stageDir)
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
