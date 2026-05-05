package dbtest_test

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestMigrationsCreatePlanEntities(t *testing.T) {
	harness := dbtest.New(t)
	wantTables := []string{
		"tenants", "projects", "instance_admins", "tenant_memberships", "users",
		"password_credentials", "passkey_credentials", "totp_credentials", "user_backup_codes",
		"instance_admin_backup_codes", "magic_link_tokens", "password_reset_tokens",
		"email_verification_tokens", "pending_invitations", "sessions", "instance_admin_sessions",
		"oidc_clients", "oidc_authorization_codes", "oidc_refresh_tokens", "oidc_consents",
		"oidc_signing_keys", "upstream_providers", "email_provider_configs", "email_outbox",
		"storage_objects", "tenant_auth_methods", "audit_entries", "bootstrap_tokens",
		"master_key_rotations", "rate_limit_buckets", "gdpr_deletions",
	}
	for _, table := range wantTables {
		dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1`, 1, table)
	}
}

func TestMigrationUpDownRoundTrip(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	if _, err := harness.SQL.Exec(`INSERT INTO projects (tenant_id, slug, name) VALUES ($1, 'app', 'App')`, tenantID); err != nil {
		t.Fatalf("populate schema before down migration: %v", err)
	}

	dbtest.ApplyDown(t, harness.SQL)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'tenants'`, 0)
	dbtest.ApplyUp(t, harness.SQL)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'tenants'`, 1)
}

func TestRLSRuntimeRoleSeesNoRowsWithoutTenantSetting(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	if _, err := harness.SQL.Exec(`INSERT INTO projects (tenant_id, slug, name) VALUES ($1, 'app', 'App')`, tenantID); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	runtimeSQL := openRuntime(t, harness.SQL, harness.DSN)
	dbtest.RequireCount(t, runtimeSQL, `SELECT count(*) FROM projects`, 0)
}

func TestAuditLogRuntimeRoleCannotMutateImmutableFieldsOrDelete(t *testing.T) {
	harness := dbtest.New(t)
	runtimeSQL := openRuntime(t, harness.SQL, harness.DSN)
	var auditID string
	if err := runtimeSQL.QueryRow(`INSERT INTO audit_entries (actor_kind, action, resource_kind) VALUES ('system', 'created', 'tenant') RETURNING id`).Scan(&auditID); err != nil {
		t.Fatalf("runtime insert audit row: %v", err)
	}

	if _, err := runtimeSQL.Exec(`UPDATE audit_entries SET action = 'tampered' WHERE id = $1`, auditID); err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("audit action update error = %v, want insufficient privilege", err)
	}
	if _, err := runtimeSQL.Exec(`DELETE FROM audit_entries WHERE id = $1`, auditID); err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("audit delete error = %v, want insufficient privilege", err)
	}
}

func TestRateLimitBucketsAllowOneNullTenantKey(t *testing.T) {
	harness := dbtest.New(t)
	if _, err := harness.SQL.Exec(`INSERT INTO rate_limit_buckets (scope, key, tenant_id, tokens, last_refill_at) VALUES ('login', '127.0.0.1', NULL, 1, now())`); err != nil {
		t.Fatalf("insert null-tenant rate limit bucket: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO rate_limit_buckets (scope, key, tenant_id, tokens, last_refill_at) VALUES ('login', '127.0.0.1', NULL, 1, now())`); err == nil {
		t.Fatal("duplicate null-tenant rate limit bucket unexpectedly succeeded")
	}
}

func openRuntime(t *testing.T, sqlDB *sql.DB, superDSN string) *sql.DB {
	t.Helper()
	dsn := dbtest.RuntimeDSN(t, sqlDB, superDSN)
	runtimeSQL, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open runtime db: %v", err)
	}
	t.Cleanup(func() { _ = runtimeSQL.Close() })
	return runtimeSQL
}
