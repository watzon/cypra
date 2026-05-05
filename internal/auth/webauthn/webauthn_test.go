package webauthn_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth/webauthn"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestPasskeyRPIDIsTenantBound(t *testing.T) {
	harness := dbtest.New(t)
	acme := dbtest.SeedTenant(t, harness.SQL, "acme")
	bravo := dbtest.SeedSecondTenant(t, harness.SQL, "bravo")
	userID := seedUser(t, harness, acme)
	service := webauthn.Service{DB: harness.SQL}
	credentialID := []byte("credential")
	if err := service.Register(context.Background(), acme, userID, "acme.cypra.localhost", credentialID, []byte("public")); err != nil {
		t.Fatalf("register passkey: %v", err)
	}
	if got, err := service.Assert(context.Background(), acme, "acme.cypra.localhost", credentialID); err != nil || got != userID {
		t.Fatalf("assert passkey got=%s err=%v", got, err)
	}
	if _, err := service.Assert(context.Background(), acme, "bravo.cypra.localhost", credentialID); !errors.Is(err, webauthn.ErrRPMismatch) {
		t.Fatalf("cross-rp error = %v", err)
	}
	if _, err := service.Assert(context.Background(), bravo, "bravo.cypra.localhost", credentialID); err == nil {
		t.Fatal("cross-tenant credential unexpectedly asserted")
	}
}

func seedUser(t *testing.T, harness *dbtest.Harness, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return userID
}
