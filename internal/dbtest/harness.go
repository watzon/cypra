// Package dbtest provides Postgres-backed integration-test helpers.
package dbtest

//revive:disable:exported

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // Registers the pgx database/sql driver for test connections.
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/watzon/cypra/internal/crypto"
	"github.com/watzon/cypra/internal/db"
	postgresdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Harness struct {
	DSN      string
	SQL      *sql.DB
	Gorm     *gorm.DB
	TenantDB *db.TenantScopedDB
}

func New(t *testing.T) *Harness {
	t.Helper()
	ctx := context.Background()
	container, err := tcpostgres.Run(
		ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("cypra"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Errorf("terminate postgres container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get postgres dsn: %v", err)
	}
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	ApplyUp(t, sqlDB)
	gormDB, err := gorm.Open(postgresdriver.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm: %v", err)
	}
	if err := gormDB.Use(db.TenantPlugin{}); err != nil {
		t.Fatalf("install tenant plugin: %v", err)
	}

	return &Harness{
		DSN:      dsn,
		SQL:      sqlDB,
		Gorm:     gormDB,
		TenantDB: db.NewTenantScopedDB(gormDB),
	}
}

func ApplyUp(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	applyMigrations(t, sqlDB, "*.up.sql", false)
}

func ApplyDown(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	applyMigrations(t, sqlDB, "*.down.sql", true)
}

func RuntimeDSN(t *testing.T, sqlDB *sql.DB, superDSN string) string {
	t.Helper()
	roleName := "runtime_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quotedRole := `"` + roleName + `"`
	if _, err := sqlDB.Exec(`CREATE ROLE ` + quotedRole + ` LOGIN PASSWORD 'runtime-password' IN ROLE cypra_runtime`); err != nil {
		t.Fatalf("create runtime role: %v", err)
	}
	t.Cleanup(func() { _, _ = sqlDB.Exec(`DROP ROLE IF EXISTS ` + quotedRole) })

	parsed, err := url.Parse(superDSN)
	if err != nil {
		t.Fatalf("parse super dsn: %v", err)
	}
	parsed.User = url.UserPassword(roleName, "runtime-password")
	return parsed.String()
}

func SeedTenant(t *testing.T, sqlDB *sql.DB, slug string) uuid.UUID {
	t.Helper()
	tenantID := uuid.New()
	_, err := sqlDB.Exec(
		`INSERT INTO tenants (id, slug, name, branding, settings) VALUES ($1, $2, $3, '{}'::jsonb, '{}'::jsonb)`,
		tenantID, slug, slug,
	)
	if err != nil {
		t.Fatalf("seed tenant %s: %v", slug, err)
	}
	SeedSigningKey(t, sqlDB, tenantID)
	return tenantID
}

func SeedSecondTenant(t *testing.T, sqlDB *sql.DB, slug string) uuid.UUID {
	t.Helper()
	return SeedTenant(t, sqlDB, slug)
}

func TestMasterKey() []byte {
	return bytes.Repeat([]byte{9}, crypto.MasterKeyBytes)
}

func SeedSigningKey(t *testing.T, sqlDB *sql.DB, tenantID uuid.UUID) {
	t.Helper()
	key, err := crypto.GenerateSigningKey(tenantID, 1, crypto.SigningAlgRS256, TestMasterKey(), time.Now().UTC())
	if err != nil {
		t.Fatalf("generate tenant signing key: %v", err)
	}
	_, err = sqlDB.Exec(`INSERT INTO oidc_signing_keys (tenant_id, kid, algorithm, public_key_jwk, private_key_encrypted, state, activated_at, retires_at) VALUES ($1, $2, $3, $4, $5, 'active', $6, $7)`, tenantID, key.KID, key.Algorithm, []byte(key.PublicJWK), key.PrivateKeyEncrypted, key.ActivatedAt, key.RetiresAt)
	if err != nil {
		t.Fatalf("seed tenant signing key: %v", err)
	}
}

func MigrationFiles(t *testing.T, pattern string, reverse bool) []string {
	t.Helper()
	root := repoRoot(t)
	files, err := filepath.Glob(filepath.Join(root, "db", "migrations", pattern))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no migration files matched %s", pattern)
	}
	sort.Strings(files)
	if reverse {
		for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
			files[i], files[j] = files[j], files[i]
		}
	}
	return files
}

func applyMigrations(t *testing.T, sqlDB *sql.DB, pattern string, reverse bool) {
	t.Helper()
	if !reverse {
		if _, err := sqlDB.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
			t.Fatalf("create schema_migrations: %v", err)
		}
	}
	for _, file := range MigrationFiles(t, pattern, reverse) {
		content, err := os.ReadFile(file) // #nosec G304 -- migration files are discovered from this repository's db/migrations directory.
		if err != nil {
			t.Fatalf("read migration %s: %v", file, err)
		}
		if _, err := sqlDB.Exec(string(content)); err != nil {
			t.Fatalf("apply migration %s: %v", filepath.Base(file), err)
		}
		version := strings.TrimSuffix(strings.TrimSuffix(filepath.Base(file), ".up.sql"), ".down.sql")
		if reverse {
			_, _ = sqlDB.Exec(`DELETE FROM schema_migrations WHERE version = $1`, version)
		} else if _, err := sqlDB.Exec(`INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING`, version); err != nil {
			t.Fatalf("record migration %s: %v", version, err)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo root not found from %s", dir)
		}
		dir = parent
	}
}

func RequireCount(t *testing.T, sqlDB *sql.DB, query string, want int, args ...any) {
	t.Helper()
	var got int
	if err := sqlDB.QueryRow(query, args...).Scan(&got); err != nil {
		t.Fatalf("scan count: %v", err)
	}
	if got != want {
		t.Fatalf("count mismatch: got %d, want %d for %s", got, want, query)
	}
}

func TenantProjectSQL(tenantID uuid.UUID, slug string) string {
	return fmt.Sprintf("INSERT INTO projects (tenant_id, slug, name) VALUES ('%s', '%s', '%s')", tenantID, slug, slug)
}
