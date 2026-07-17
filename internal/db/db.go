package db

import (
	"database/sql"
	"embed"
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
	if err := integrityCheck(sqldb); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("database integrity: %w", err)
	}
	validatedBackupPath, err := runMigrations(sqldb, path, opts)
	if err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	if err := postMigrationForeignKeyCheck(sqldb, validatedBackupPath); err != nil {
		sqldb.Close()
		return nil, err
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
		absolutePath, err := filepath.Abs(path)
		if err != nil {
			absolutePath = path
		}
		dsn = (&url.URL{Scheme: "file", Path: filepath.ToSlash(absolutePath)}).String()
	}
	query := url.Values{}
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "busy_timeout(5000)")
	return dsn + "?" + query.Encode()
}

type migrationRunOptions struct {
	databasePath       string
	backupBeforeRepair bool
	repairMigrations   map[string]struct{}
}

func runMigrations(db *sql.DB, databasePath string, opts OpenOptions) (string, error) {
	return runMigrationsFromFS(db, migrationsFS, "migrations", migrationRunOptions{
		databasePath:       databasePath,
		backupBeforeRepair: opts.BackupBeforeRepair,
		repairMigrations:   repairMigrations,
	})
}

func runMigrationsFromFS(db *sql.DB, source fs.FS, directory string, opts migrationRunOptions) (string, error) {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return "", err
	}

	entries, err := fs.ReadDir(source, directory)
	if err != nil {
		return "", err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	validatedBackupPath := ""

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version := strings.TrimSuffix(entry.Name(), ".sql")
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count); err != nil {
			return validatedBackupPath, migrationError(version, validatedBackupPath, fmt.Errorf("check migration state: %w", err))
		}
		if count > 0 {
			continue
		}
		_, isRepair := opts.repairMigrations[entry.Name()]
		if isRepair && validatedBackupPath == "" && opts.databasePath != ":memory:" {
			if !opts.backupBeforeRepair {
				return "", fmt.Errorf("apply %s: repair migration requires a validated backup", version)
			}
			if err := integrityCheck(db); err != nil {
				return "", fmt.Errorf("pre-repair integrity for %s: %w", version, err)
			}
			validatedBackupPath, err = backupDatabase(db, opts.databasePath)
			if err != nil {
				return "", fmt.Errorf("backup before %s: %w", version, err)
			}
		}

		sqlBytes, err := fs.ReadFile(source, directory+"/"+entry.Name())
		if err != nil {
			return validatedBackupPath, migrationError(version, validatedBackupPath, err)
		}
		tx, err := db.Begin()
		if err != nil {
			return validatedBackupPath, migrationError(version, validatedBackupPath, fmt.Errorf("begin transaction: %w", err))
		}
		if _, err := tx.Exec(string(sqlBytes)); err != nil {
			_ = tx.Rollback()
			return validatedBackupPath, migrationError(version, validatedBackupPath, err)
		}
		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
			_ = tx.Rollback()
			return validatedBackupPath, migrationError(version, validatedBackupPath, fmt.Errorf("record migration: %w", err))
		}
		if err := tx.Commit(); err != nil {
			return validatedBackupPath, migrationError(version, validatedBackupPath, fmt.Errorf("commit: %w", err))
		}
	}
	return validatedBackupPath, nil
}

func migrationError(version, backupPath string, err error) error {
	if backupPath != "" {
		return fmt.Errorf("apply %s (validated backup: %s): %w", version, backupPath, err)
	}
	return fmt.Errorf("apply %s: %w", version, err)
}

func postMigrationForeignKeyCheck(db *sql.DB, validatedBackupPath string) error {
	if err := foreignKeyCheck(db); err != nil {
		if validatedBackupPath != "" {
			return fmt.Errorf("database foreign keys (validated backup: %s): %w", validatedBackupPath, err)
		}
		return fmt.Errorf("database foreign keys: %w", err)
	}
	return nil
}
