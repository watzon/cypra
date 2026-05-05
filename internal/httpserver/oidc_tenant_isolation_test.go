package httpserver_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/watzon/cypra/internal/db"
	fuzzharness "github.com/watzon/cypra/internal/db/fuzz"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestPhase5OIDCHandlersEnrolledInTenantIsolationFuzzer(t *testing.T) {
	harness := dbtest.New(t)
	protectedTenant := dbtest.SeedTenant(t, harness.SQL, "acme")
	otherTenant := uuid.New()
	projectID := uuid.New()
	userID := uuid.New()
	clientID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO projects (id, tenant_id, slug, name) VALUES ($1, $2, 'app', 'App')`, projectID, protectedTenant); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, protectedTenant); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_clients (id, tenant_id, project_id, client_id, redirect_uris, allowed_scopes, token_endpoint_auth_method) VALUES ($1, $2, $3, 'client', $4, $5, 'none')`, clientID, protectedTenant, projectID, pq.Array([]string{"https://app.example.com/callback"}), pq.Array([]string{"openid"})); err != nil {
		t.Fatalf("seed client: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_authorization_codes (code_hash, tenant_id, oidc_client_uuid, user_id, redirect_uri, scope, pkce_challenge, pkce_method, expires_at) VALUES ('hash'::bytea, $1, $2, $3, 'https://app.example.com/callback', $4, 'challenge', 'S256', now() + interval '1 minute')`, protectedTenant, clientID, userID, pq.Array([]string{"openid"})); err != nil {
		t.Fatalf("seed auth code: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_consents (tenant_id, user_id, oidc_client_uuid, scopes) VALUES ($1, $2, $3, $4)`, protectedTenant, userID, clientID, pq.Array([]string{"openid"})); err != nil {
		t.Fatalf("seed consent: %v", err)
	}

	fuzzer := fuzzharness.Harness{ProtectedTenant: protectedTenant, OtherTenant: otherTenant}
	for _, tc := range []struct {
		name  string
		query string
	}{
		{name: "oidc clients", query: `SELECT count(*) FROM oidc_clients WHERE tenant_id = $1`},
		{name: "authorization codes", query: `SELECT count(*) FROM oidc_authorization_codes WHERE tenant_id = $1`},
		{name: "refresh tokens", query: `SELECT count(*) FROM oidc_refresh_tokens WHERE tenant_id = $1`},
		{name: "consents", query: `SELECT count(*) FROM oidc_consents WHERE tenant_id = $1`},
		{name: "signing keys", query: `SELECT count(*) FROM oidc_signing_keys WHERE tenant_id = $1`},
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
