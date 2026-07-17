package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// integrityCheck verifies the physical and structural consistency of a SQLite
// database. It intentionally does not perform a foreign-key check: a backup
// taken immediately before a repair migration must preserve logical orphans.
func integrityCheck(db sqlQueryer) error {
	rows, err := db.QueryContext(context.Background(), "PRAGMA integrity_check")
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
func foreignKeyCheck(db sqlQueryer) error {
	rows, err := db.QueryContext(context.Background(), "PRAGMA foreign_key_check")
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

type sqlExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type sqlQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type backupHooks struct {
	afterStageReady func(stageDirectory, stagedDatabase string) error
	beforePublish   func(backupPath string) error
}

// backupDatabase creates a consistent, independently validated SQLite copy
// next to databasePath.
func backupDatabase(db sqlExecer, databasePath string) (string, error) {
	return backupDatabaseWithHooks(db, databasePath, backupHooks{})
}

// backupDatabaseWithHooks stages the SQLite output in a private adjacent
// directory. Only a protected, validated file is atomically hard-linked to its
// final name, so a raced destination is never replaced.
func backupDatabaseWithHooks(db sqlExecer, databasePath string, hooks backupHooks) (backupPath string, err error) {
	if databasePath == ":memory:" {
		return "", nil
	}

	stagePrefix := "." + filepath.Base(databasePath) + ".backup-stage-"
	stageDirectory, err := os.MkdirTemp(filepath.Dir(databasePath), stagePrefix+"*")
	if err != nil {
		return "", fmt.Errorf("create private backup stage: %w", err)
	}
	if err := os.Chmod(stageDirectory, 0700); err != nil {
		_ = os.RemoveAll(stageDirectory)
		return "", fmt.Errorf("protect backup stage: %w", err)
	}
	defer func() {
		if cleanupErr := os.RemoveAll(stageDirectory); cleanupErr != nil {
			err = errors.Join(err, fmt.Errorf("clean backup stage: %w", cleanupErr))
		}
	}()
	stagedDatabase := filepath.Join(stageDirectory, "database.sqlite")
	suffix := strings.TrimPrefix(filepath.Base(stageDirectory), stagePrefix)
	backupPath = databasePath + ".backup-" + suffix

	if _, err = db.ExecContext(context.Background(), "VACUUM INTO ?", stagedDatabase); err != nil {
		return "", fmt.Errorf("vacuum backup: %w", err)
	}
	if err = os.Chmod(stagedDatabase, 0600); err != nil {
		return "", fmt.Errorf("protect backup: %w", err)
	}
	if hooks.afterStageReady != nil {
		if err = hooks.afterStageReady(stageDirectory, stagedDatabase); err != nil {
			return "", fmt.Errorf("inspect backup stage: %w", err)
		}
	}

	backup, openErr := sql.Open("sqlite", sqliteDSN(stagedDatabase))
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

	if hooks.beforePublish != nil {
		if err = hooks.beforePublish(backupPath); err != nil {
			return "", fmt.Errorf("before publishing backup: %w", err)
		}
	}
	if err = os.Link(stagedDatabase, backupPath); err != nil {
		return "", fmt.Errorf("publish validated backup: %w", err)
	}
	return backupPath, nil
}
