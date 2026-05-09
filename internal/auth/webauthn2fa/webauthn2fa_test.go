package webauthn2fa_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth/webauthn"
	"github.com/watzon/cypra/internal/auth/webauthn2fa"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestBeginEnrollStoresSecondFactorCeremony(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := seedUser(t, harness, tenantID)
	service := webauthn2fa.Service{Primary: webauthn.Service{DB: harness.SQL}}

	creation, ceremonyID, err := service.BeginEnroll(context.Background(), webauthn2fa.BeginEnrollRequest{
		TenantID: tenantID,
		UserID:   userID,
		RPID:     "acme.cypra.localhost",
	})
	if err != nil {
		t.Fatalf("begin enroll: %v", err)
	}
	if creation.Response.RelyingParty.ID != "acme.cypra.localhost" {
		t.Fatalf("rp id = %q", creation.Response.RelyingParty.ID)
	}
	var kind string
	var storedUser uuid.UUID
	if err := harness.SQL.QueryRow(`SELECT kind, user_id FROM webauthn_challenges WHERE id = $1`, ceremonyID).Scan(&kind, &storedUser); err != nil {
		t.Fatalf("load ceremony: %v", err)
	}
	if kind != webauthn.KindWebAuthn2FAEnrollment || storedUser != userID {
		t.Fatalf("kind/user = %q/%s", kind, storedUser)
	}
}

func TestBeginVerifyStoresSecondFactorAssertionCeremony(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	service := webauthn2fa.Service{Primary: webauthn.Service{DB: harness.SQL}}

	assertion, ceremonyID, err := service.BeginVerify(context.Background(), webauthn2fa.BeginVerifyRequest{
		TenantID: tenantID,
		RPID:     "acme.cypra.localhost",
	})
	if err != nil {
		t.Fatalf("begin verify: %v", err)
	}
	if assertion.Response.RelyingPartyID != "acme.cypra.localhost" {
		t.Fatalf("rp id = %q", assertion.Response.RelyingPartyID)
	}
	var kind string
	if err := harness.SQL.QueryRow(`SELECT kind FROM webauthn_challenges WHERE id = $1 AND user_id IS NULL`, ceremonyID).Scan(&kind); err != nil {
		t.Fatalf("load ceremony: %v", err)
	}
	if kind != webauthn.KindWebAuthn2FAAssertion {
		t.Fatalf("kind = %q", kind)
	}
}

func seedUser(t *testing.T, harness *dbtest.Harness, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'mfa-user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return userID
}
