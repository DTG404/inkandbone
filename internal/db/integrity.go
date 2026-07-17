package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// integrityCheck verifies the physical and structural consistency of a SQLite
// database. It intentionally does not perform a foreign-key check: a backup
// taken immediately before a repair migration must preserve logical orphans.
func integrityCheck(db *sql.DB) error {
	rows, err := db.Query("PRAGMA integrity_check")
	if err != nil {
		return fmt.Errorf("run integrity_check: %w", err)
	}
	defer rows.Close()

	var problems []string
	for rows.Next() {
		var result string
		if err := rows.Scan(&result); err != nil {
			return fmt.Errorf("read integrity_check: %w", err)
		}
		if result != "ok" {
			problems = append(problems, result)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read integrity_check: %w", err)
	}
	if len(problems) > 0 {
		return fmt.Errorf("integrity_check failed: %s", strings.Join(problems, "; "))
	}
	return nil
}

// foreignKeyCheck verifies that no committed row violates a declared foreign
// key. It is run after migrations, so a registered repair gets an opportunity
// to quarantine or repair pre-existing orphans first.
func foreignKeyCheck(db *sql.DB) error {
	rows, err := db.Query("PRAGMA foreign_key_check")
	if err != nil {
		return fmt.Errorf("run foreign_key_check: %w", err)
	}
	defer rows.Close()

	var problems []string
	for rows.Next() {
		var table, parent string
		var rowID sql.NullInt64
		var foreignKeyID int64
		if err := rows.Scan(&table, &rowID, &parent, &foreignKeyID); err != nil {
			return fmt.Errorf("read foreign_key_check: %w", err)
		}
		row := "without rowid"
		if rowID.Valid {
			row = fmt.Sprintf("rowid %d", rowID.Int64)
		}
		problems = append(problems, fmt.Sprintf("%s %s references %s (foreign key %d)", table, row, parent, foreignKeyID))
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read foreign_key_check: %w", err)
	}
	if len(problems) > 0 {
		return fmt.Errorf("foreign_key_check failed: %s", strings.Join(problems, "; "))
	}
	return nil
}

// backupDatabase creates a consistent, independently validated SQLite copy
// next to databasePath. The temporary output is never published as a backup;
// only a chmod'd and validated file is atomically renamed into place.
func backupDatabase(db *sql.DB, databasePath string) (backupPath string, err error) {
	if databasePath == ":memory:" {
		return "", nil
	}

	temporary, err := os.CreateTemp(filepath.Dir(databasePath), filepath.Base(databasePath)+".backup-*.tmp")
	if err != nil {
		return "", fmt.Errorf("reserve backup path: %w", err)
	}
	temporaryPath := temporary.Name()
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return "", fmt.Errorf("close backup reservation: %w", err)
	}
	if err := os.Remove(temporaryPath); err != nil {
		return "", fmt.Errorf("prepare backup path: %w", err)
	}
	defer func() {
		if err != nil {
			_ = os.Remove(temporaryPath)
		}
	}()

	if _, err = db.Exec("VACUUM INTO ?", temporaryPath); err != nil {
		return "", fmt.Errorf("vacuum backup: %w", err)
	}
	if err = os.Chmod(temporaryPath, 0600); err != nil {
		return "", fmt.Errorf("protect backup: %w", err)
	}

	backup, openErr := sql.Open("sqlite", sqliteDSN(temporaryPath))
	if openErr != nil {
		return "", fmt.Errorf("open backup for validation: %w", openErr)
	}
	backup.SetMaxOpenConns(1)
	validationErr := integrityCheck(backup)
	closeErr := backup.Close()
	if validationErr != nil {
		return "", fmt.Errorf("validate backup: %w", validationErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("close validated backup: %w", closeErr)
	}

	backupPath = strings.TrimSuffix(temporaryPath, ".tmp")
	if err = os.Rename(temporaryPath, backupPath); err != nil {
		return "", fmt.Errorf("publish validated backup: %w", err)
	}
	return backupPath, nil
}
