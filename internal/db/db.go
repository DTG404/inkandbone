package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps sql.DB with typed query methods.
type DB struct {
	db *sql.DB
}

// OpenOptions controls database startup behavior. A file-backed repair is
// refused when BackupBeforeRepair is false; the option exists so callers can
// make that policy explicit without permitting an unprotected repair.
type OpenOptions struct {
	BackupBeforeRepair bool
}

// repairMigrations is deliberately keyed by immutable embedded filename. A
// migration is never treated as a destructive repair merely because of its SQL
// contents or numeric position.
var repairMigrations = map[string]struct{}{
	"056_integrity_repair.sql": {},
}

// Open opens (or creates) the SQLite database at path and runs pending migrations.
// Creates parent directories if they do not exist.
func Open(path string) (*DB, error) {
	return OpenWithOptions(path, OpenOptions{BackupBeforeRepair: true})
}

// OpenWithOptions opens (or creates) the SQLite database at path, validates it,
// runs pending migrations, and rejects any remaining foreign-key violations.
func OpenWithOptions(path string, opts OpenOptions) (*DB, error) {
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}
	sqldb, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	sqldb.SetMaxOpenConns(1) // SQLite is single-writer
	_, err = runMigrations(sqldb, path, opts)
	if err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &DB{db: sqldb}, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error { return d.db.Close() }

// SQL returns the underlying *sql.DB for use in tests.
func (d *DB) SQL() *sql.DB { return d.db }

func sqliteDSN(path string) string {
	var dsn string
	if path == ":memory:" {
		dsn = "file::memory:"
	} else {
		dsn = sqliteFileURI(path)
	}
	query := url.Values{}
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "busy_timeout(5000)")
	return dsn + "?" + query.Encode()
}

func sqliteFileURI(path string) string {
	windowsDrive := isWindowsDrivePath(path)
	unc := isUNCPath(path)
	if !windowsDrive && !unc && !filepath.IsAbs(path) {
		if absolutePath, err := filepath.Abs(path); err == nil {
			path = absolutePath
		}
		windowsDrive = isWindowsDrivePath(path)
		unc = isUNCPath(path)
	}
	normalized := filepath.ToSlash(path)
	if windowsDrive || unc {
		normalized = strings.ReplaceAll(path, `\`, "/")
	}
	if unc {
		withoutPrefix := strings.TrimLeft(normalized, "/")
		host, sharePath, found := strings.Cut(withoutPrefix, "/")
		if found {
			return (&url.URL{Scheme: "file", Host: host, Path: "/" + sharePath}).String()
		}
	}
	if windowsDrive {
		normalized = "/" + normalized
	}
	return (&url.URL{Scheme: "file", Path: normalized}).String()
}

func isWindowsDrivePath(path string) bool {
	return len(path) >= 3 && ((path[0] >= 'a' && path[0] <= 'z') || (path[0] >= 'A' && path[0] <= 'Z')) && path[1] == ':' && (path[2] == '/' || path[2] == '\\')
}

func isUNCPath(path string) bool {
	return strings.HasPrefix(path, `\\`) || strings.HasPrefix(path, "//")
}

type migrationRunOptions struct {
	databasePath         string
	backupBeforeRepair   bool
	repairMigrations     map[string]struct{}
	afterValidatedBackup func(path string)
	afterMigration       func(filename string)
}

func runMigrations(db *sql.DB, databasePath string, opts OpenOptions) (string, error) {
	return runMigrationsFromFS(db, migrationsFS, "migrations", migrationRunOptions{
		databasePath:       databasePath,
		backupBeforeRepair: opts.BackupBeforeRepair,
		repairMigrations:   repairMigrations,
	})
}

func runMigrationsFromFS(db *sql.DB, source fs.FS, directory string, opts migrationRunOptions) (backupPath string, err error) {
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return "", fmt.Errorf("get migration connection: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			err = errors.Join(err, migrationError("close migration connection", backupPath, closeErr))
		}
	}()

	fileBacked := opts.databasePath != "" && opts.databasePath != ":memory:"
	exclusiveConfigured := false
	if fileBacked {
		exclusiveConfigured = true
		if err := acquireExclusiveMigrationLock(ctx, conn); err != nil {
			_ = releaseExclusiveMigrationLock(ctx, conn)
			return "", fmt.Errorf("database integrity/exclusive migration lock: %w", err)
		}
	}
	defer func() {
		if !exclusiveConfigured {
			return
		}
		if releaseErr := releaseExclusiveMigrationLock(ctx, conn); releaseErr != nil {
			err = errors.Join(err, migrationError("release exclusive migration lock", backupPath, releaseErr))
		}
	}()

	entries, err := fs.ReadDir(source, directory)
	if err != nil {
		return "", err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var pending []fs.DirEntry
	repairPending := false
	var schemaMigrationsExists int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations'").Scan(&schemaMigrationsExists); err != nil {
		return "", fmt.Errorf("inspect migration state: %w", err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version := strings.TrimSuffix(entry.Name(), ".sql")
		var count int
		if schemaMigrationsExists > 0 {
			if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count); err != nil {
				return backupPath, migrationError(version, backupPath, fmt.Errorf("check migration state: %w", err))
			}
		}
		if count > 0 {
			continue
		}
		pending = append(pending, entry)
		if _, isRepair := opts.repairMigrations[entry.Name()]; isRepair {
			repairPending = true
		}
	}

	if repairPending && fileBacked {
		if !opts.backupBeforeRepair {
			return "", errors.New("repair migration requires a validated backup")
		}
		backupPath, err = backupDatabase(conn, opts.databasePath)
		if err != nil {
			return backupPath, migrationError("backup before repair", backupPath, err)
		}
		if opts.afterValidatedBackup != nil {
			opts.afterValidatedBackup(backupPath)
		}
	}
	if err := integrityCheck(conn); err != nil {
		return backupPath, migrationError("source integrity", backupPath, err)
	}
	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return backupPath, migrationError("create migration ledger", backupPath, err)
	}

	for _, entry := range pending {
		version := strings.TrimSuffix(entry.Name(), ".sql")
		sqlBytes, err := fs.ReadFile(source, directory+"/"+entry.Name())
		if err != nil {
			return backupPath, migrationError(version, backupPath, err)
		}
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return backupPath, migrationError(version, backupPath, fmt.Errorf("begin transaction: %w", err))
		}
		if _, err := tx.ExecContext(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback()
			return backupPath, migrationError(version, backupPath, err)
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
			_ = tx.Rollback()
			return backupPath, migrationError(version, backupPath, fmt.Errorf("record migration: %w", err))
		}
		if err := tx.Commit(); err != nil {
			return backupPath, migrationError(version, backupPath, fmt.Errorf("commit: %w", err))
		}
		if opts.afterMigration != nil {
			opts.afterMigration(entry.Name())
		}
	}
	if err := postMigrationForeignKeyCheck(conn, backupPath); err != nil {
		return backupPath, err
	}
	return backupPath, nil
}

func acquireExclusiveMigrationLock(ctx context.Context, conn *sql.Conn) error {
	var mode string
	if err := conn.QueryRowContext(ctx, "PRAGMA locking_mode=EXCLUSIVE").Scan(&mode); err != nil {
		return fmt.Errorf("set locking_mode exclusive: %w", err)
	}
	if !strings.EqualFold(mode, "exclusive") {
		return fmt.Errorf("set locking_mode exclusive: SQLite returned %q", mode)
	}
	if _, err := conn.ExecContext(ctx, "BEGIN EXCLUSIVE"); err != nil {
		return fmt.Errorf("begin exclusive lock transaction: %w", err)
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		_, _ = conn.ExecContext(ctx, "ROLLBACK")
		return fmt.Errorf("commit exclusive lock transaction: %w", err)
	}
	return nil
}

func releaseExclusiveMigrationLock(ctx context.Context, conn *sql.Conn) error {
	var mode string
	if err := conn.QueryRowContext(ctx, "PRAGMA locking_mode=NORMAL").Scan(&mode); err != nil {
		return fmt.Errorf("set locking_mode normal: %w", err)
	}
	if !strings.EqualFold(mode, "normal") {
		return fmt.Errorf("set locking_mode normal: SQLite returned %q", mode)
	}
	var schemaVersion int
	if err := conn.QueryRowContext(ctx, "PRAGMA schema_version").Scan(&schemaVersion); err != nil {
		return fmt.Errorf("release exclusive database lock: %w", err)
	}
	return nil
}

func migrationError(version, backupPath string, err error) error {
	if backupPath != "" {
		return fmt.Errorf("apply %s (validated backup: %s): %w", version, backupPath, err)
	}
	return fmt.Errorf("apply %s: %w", version, err)
}

func postMigrationForeignKeyCheck(db sqlQueryer, validatedBackupPath string) error {
	if err := foreignKeyCheck(db); err != nil {
		if validatedBackupPath != "" {
			return fmt.Errorf("database foreign keys (validated backup: %s): %w", validatedBackupPath, err)
		}
		return fmt.Errorf("database foreign keys: %w", err)
	}
	return nil
}
