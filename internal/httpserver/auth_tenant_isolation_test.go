package httpserver_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/db"
	fuzzharness "github.com/watzon/cypra/internal/db/fuzz"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestPhase4AuthHandlersEnrolledInTenantIsolationFuzzer(t *testing.T) {
	harness := dbtest.New(t)
	protectedTenant := dbtest.SeedTenant(t, harness.SQL, "acme")
	otherTenant := dbtest.SeedSecondTenant(t, harness.SQL, "bravo")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, protectedTenant); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO passkey_credentials (tenant_id, user_id, credential_id, public_key, rp_id) VALUES ($1, $2, 'credential'::bytea, 'public'::bytea, 'acme.cypra.localhost')`, protectedTenant, userID); err != nil {
		t.Fatalf("seed passkey: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO totp_credentials (tenant_id, user_id, secret_encrypted) VALUES ($1, $2, 'secret'::bytea)`, protectedTenant, userID); err != nil {
		t.Fatalf("seed totp: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO user_backup_codes (tenant_id, user_id, code_hash) VALUES ($1, $2, 'hash'::bytea)`, protectedTenant, userID); err != nil {
		t.Fatalf("seed backup code: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO magic_link_tokens (tenant_id, user_id, email, token_hash, expires_at) VALUES ($1, $2, 'user@example.com', 'hash'::bytea, now() + interval '15 minutes')`, protectedTenant, userID); err != nil {
		t.Fatalf("seed magic link: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO pending_invitations (tenant_id, email, role, token_hash, created_by_kind, expires_at) VALUES ($1, 'invitee@example.com', 'member', 'hash'::bytea, 'system', now() + interval '7 days')`, protectedTenant); err != nil {
		t.Fatalf("seed invite: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO email_outbox (tenant_id, to_address, template, payload) VALUES ($1, 'user@example.com', 'magic-link', '{}'::jsonb)`, protectedTenant); err != nil {
		t.Fatalf("seed outbox: %v", err)
	}

	fuzzer := fuzzharness.Harness{ProtectedTenant: protectedTenant, OtherTenant: otherTenant}
	for _, tc := range []struct {
		name  string
		query string
	}{
		{name: "password signup/signin users", query: `SELECT count(*) FROM users WHERE tenant_id = $1`},
		{name: "passkey credentials", query: `SELECT count(*) FROM passkey_credentials WHERE tenant_id = $1`},
		{name: "totp credentials", query: `SELECT count(*) FROM totp_credentials WHERE tenant_id = $1`},
		{name: "backup codes", query: `SELECT count(*) FROM user_backup_codes WHERE tenant_id = $1`},
		{name: "magic link tokens", query: `SELECT count(*) FROM magic_link_tokens WHERE tenant_id = $1`},
		{name: "pending invitations", query: `SELECT count(*) FROM pending_invitations WHERE tenant_id = $1`},
		{name: "email outbox", query: `SELECT count(*) FROM email_outbox WHERE tenant_id = $1`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := fuzzer.AssertIsolated(context.Background(), func(ctx context.Context) fuzzharness.Outcome {
				tenantID, ok := db.TenantFromContext(ctx)
				if !ok {
					return fuzzharness.Outcome{Err: db.ErrTenantContextMissing}
				}
				var rows int
				if err := harness.SQL.QueryRowContext(ctx, tc.query, tenantID).Scan(&rows); err != nil {
					return fuzzharness.Outcome{Err: err}
				}
				return fuzzharness.Outcome{Rows: rows}
			})
			if err != nil {
				t.Fatalf("handler leaked cross-tenant rows: %v", err)
			}
		})
	}
}
