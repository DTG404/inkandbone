package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForeignKeysEnabledOnEveryPoolConnection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pool.db")
	pool, err := sql.Open("sqlite", sqliteDSN(path))
	require.NoError(t, err)
	defer pool.Close()

	// Production remains single-writer. This test widens only its private pool
	// so holding several Conn values proves the DSN initializes each physical
	// connection rather than repeatedly checking one pooled connection.
	pool.SetMaxOpenConns(4)
	ctx := context.Background()
	connections := make([]*sql.Conn, 0, 4)
	for range 4 {
		conn, err := pool.Conn(ctx)
		require.NoError(t, err)
		connections = append(connections, conn)
	}
	t.Cleanup(func() {
		for _, conn := range connections {
			_ = conn.Close()
		}
	})
	require.Equal(t, 4, pool.Stats().OpenConnections)

	for i, conn := range connections {
		var foreignKeys, busyTimeout int
		require.NoError(t, conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys), "connection %d", i)
		require.NoError(t, conn.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout), "connection %d", i)
		assert.Equal(t, 1, foreignKeys, "connection %d", i)
		assert.Equal(t, 5000, busyTimeout, "connection %d", i)
	}

	_, err = connections[0].ExecContext(ctx, `
		CREATE TABLE parents (id INTEGER PRIMARY KEY);
		CREATE TABLE children (id INTEGER PRIMARY KEY, parent_id INTEGER NOT NULL REFERENCES parents(id));
	`)
	require.NoError(t, err)
	for i, conn := range connections {
		_, err := conn.ExecContext(ctx, "INSERT INTO children (id, parent_id) VALUES (?, 999)", i+1)
		require.ErrorContains(t, err, "FOREIGN KEY constraint failed", "connection %d", i)
	}
}

func TestOpenRetainsSingleWriterPool(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "single-writer.db"))
	require.NoError(t, err)
	defer d.Close()

	assert.Equal(t, 1, d.SQL().Stats().MaxOpenConnections)
}

func TestIntegrityCheckRefusesCorruptDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrupt.db")
	d, err := Open(path)
	require.NoError(t, err)
	require.NoError(t, d.Close())

	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	require.NoError(t, err)
	_, err = f.WriteAt([]byte("not a sqlite db!"), 0)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	_, err = Open(path)
	require.ErrorContains(t, err, "integrity")
}

func TestBackupDatabaseIsConsistentPrivateAndIndependentlyValid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source's database.db")
	d, err := Open(path)
	require.NoError(t, err)
	defer d.Close()
	require.NoError(t, d.SetSetting("backup_sentinel", "alpha;beta"))

	backupPath, err := backupDatabase(d.SQL(), path)
	require.NoError(t, err)
	assert.FileExists(t, backupPath)
	info, err := os.Stat(backupPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	backup, err := sql.Open("sqlite", sqliteDSN(backupPath))
	require.NoError(t, err)
	defer backup.Close()
	require.NoError(t, integrityCheck(backup))
	var value string
	require.NoError(t, backup.QueryRow("SELECT value FROM settings WHERE key = 'backup_sentinel'").Scan(&value))
	assert.Equal(t, "alpha;beta", value)
}

func TestForeignKeyCheckReportsViolations(t *testing.T) {
	db, err := sql.Open("sqlite", sqliteDSN(":memory:"))
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`
		CREATE TABLE parent (id INTEGER PRIMARY KEY);
		CREATE TABLE child (id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES parent(id));
		PRAGMA foreign_keys = OFF;
		INSERT INTO child (id, parent_id) VALUES (1, 999);
		PRAGMA foreign_keys = ON;
	`)
	require.NoError(t, err)

	err = foreignKeyCheck(db)
	require.ErrorContains(t, err, "child")
}
