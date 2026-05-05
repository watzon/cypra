// Package migrate applies Cypra SQL migrations.
package migrate

//revive:disable:exported

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const DefaultDir = "db/migrations"

func Apply(ctx context.Context, db *sql.DB, dir string) error {
	if dir == "" {
		dir = DefaultDir
	}
	if err := ensureVersionTable(ctx, db); err != nil {
		return err
	}
	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return err
	}
	files, err := migrationFiles(dir, ".up.sql")
	if err != nil {
		return err
	}
	for _, file := range files {
		version := migrationVersion(file)
		if applied[version] {
			continue
		}
		content, err := os.ReadFile(file) // #nosec G304 -- migration dir is operator/project provided.
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}
		if _, err := db.ExecContext(ctx, string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", filepath.Base(file), err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}
	}
	return nil
}

func Pending(ctx context.Context, db *sql.DB, dir string) (bool, error) {
	if dir == "" {
		dir = DefaultDir
	}
	if err := ensureVersionTable(ctx, db); err != nil {
		return false, err
	}
	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return false, err
	}
	files, err := migrationFiles(dir, ".up.sql")
	if err != nil {
		return false, err
	}
	for _, file := range files {
		if !applied[migrationVersion(file)] {
			return true, nil
		}
	}
	return false, nil
}

func ensureVersionTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`)
	if err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}
	return nil
}

func appliedVersions(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("list applied migrations: %w", err)
	}
	defer func() { _ = rows.Close() }()
	applied := map[string]bool{}
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scan applied migration: %w", err)
		}
		applied[version] = true
	}
	return applied, rows.Err()
}

func migrationFiles(dir, suffix string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*"+suffix))
	if err != nil {
		return nil, fmt.Errorf("glob migrations: %w", err)
	}
	sort.Strings(files)
	return files, nil
}

func migrationVersion(file string) string {
	base := filepath.Base(file)
	return strings.TrimSuffix(base, ".up.sql")
}
