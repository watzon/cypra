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
	"strconv"
	"strings"
)

const DefaultDir = "db/migrations"

type Status struct {
	Current int
	Pending []string
}

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
	status, err := CurrentStatus(ctx, db, dir)
	if err != nil {
		return false, err
	}
	return len(status.Pending) > 0, nil
}

func CurrentStatus(ctx context.Context, db *sql.DB, dir string) (Status, error) {
	if dir == "" {
		dir = DefaultDir
	}
	if err := ensureVersionTable(ctx, db); err != nil {
		return Status{}, err
	}
	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return Status{}, err
	}
	files, err := migrationFiles(dir, ".up.sql")
	if err != nil {
		return Status{}, err
	}
	status := Status{Pending: []string{}}
	for _, file := range files {
		version := migrationVersion(file)
		if applied[version] {
			status.Current = max(status.Current, migrationNumber(version))
			continue
		}
		status.Pending = append(status.Pending, version)
	}
	return status, nil
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
	if len(files) == 0 && dir == DefaultDir {
		resolved, ok := findDefaultDir()
		if ok && resolved != dir {
			files, err = filepath.Glob(filepath.Join(resolved, "*"+suffix))
			if err != nil {
				return nil, fmt.Errorf("glob migrations: %w", err)
			}
		}
	}
	sort.Strings(files)
	return files, nil
}

func migrationVersion(file string) string {
	base := filepath.Base(file)
	return strings.TrimSuffix(base, ".up.sql")
}

func migrationNumber(version string) int {
	prefix, _, _ := strings.Cut(version, "_")
	number, err := strconv.Atoi(prefix)
	if err != nil {
		return 0
	}
	return number
}

func findDefaultDir() (string, bool) {
	wd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		candidate := filepath.Join(wd, DefaultDir)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, true
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return "", false
		}
		wd = parent
	}
}
