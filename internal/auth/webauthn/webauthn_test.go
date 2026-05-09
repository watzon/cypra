package webauthn_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth/webauthn"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestBeginRegistrationStoresServerChallengeWithTenantRPID(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := seedUser(t, harness, tenantID)
	service := webauthn.Service{DB: harness.SQL}

	creation, ceremonyID, err := service.BeginRegistration(context.Background(), webauthn.BeginRegistrationRequest{
		TenantID: tenantID,
		UserID:   userID,
		RPID:     "acme.cypra.localhost",
	})
	if err != nil {
		t.Fatalf("begin registration: %v", err)
	}
	if ceremonyID == uuid.Nil {
		t.Fatal("ceremony id is nil")
	}
	if creation.Response.RelyingParty.ID != "acme.cypra.localhost" {
		t.Fatalf("rp id = %q", creation.Response.RelyingParty.ID)
	}
	if len(creation.Response.Challenge) < 16 {
		t.Fatalf("challenge length = %d, want server-generated challenge", len(creation.Response.Challenge))
	}

	var storedRPID, storedChallenge string
	if err := harness.SQL.QueryRow(`SELECT rp_id, session_data->>'challenge' FROM webauthn_challenges WHERE id = $1 AND tenant_id = $2 AND consumed_at IS NULL`, ceremonyID, tenantID).Scan(&storedRPID, &storedChallenge); err != nil {
		t.Fatalf("load stored ceremony: %v", err)
	}
	if storedRPID != "acme.cypra.localhost" || storedChallenge != creation.Response.Challenge.String() {
		t.Fatalf("stored rp/challenge = %q/%q", storedRPID, storedChallenge)
	}
}

func TestFinishRegistrationRejectsWrongRPIDAndConsumesChallenge(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := seedUser(t, harness, tenantID)
	service := webauthn.Service{DB: harness.SQL}

	_, ceremonyID, err := service.BeginRegistration(context.Background(), webauthn.BeginRegistrationRequest{
		TenantID: tenantID,
		UserID:   userID,
		RPID:     "acme.cypra.localhost",
	})
	if err != nil {
		t.Fatalf("begin registration: %v", err)
	}
	_, err = service.FinishRegistration(context.Background(), webauthn.FinishRegistrationRequest{
		TenantID:   tenantID,
		UserID:     userID,
		RPID:       "bravo.cypra.localhost",
		CeremonyID: ceremonyID,
	})
	if !errors.Is(err, webauthn.ErrRPMismatch) {
		t.Fatalf("finish error = %v, want rp mismatch", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM webauthn_challenges WHERE id = $1 AND consumed_at IS NOT NULL`, 1, ceremonyID)
}

func TestAssertionCeremonyIsTenantBound(t *testing.T) {
	harness := dbtest.New(t)
	acme := dbtest.SeedTenant(t, harness.SQL, "acme")
	bravo := dbtest.SeedSecondTenant(t, harness.SQL, "bravo")
	service := webauthn.Service{DB: harness.SQL}

	_, ceremonyID, err := service.BeginAssertion(context.Background(), webauthn.BeginAssertionRequest{
		TenantID: acme,
		RPID:     "acme.cypra.localhost",
	})
	if err != nil {
		t.Fatalf("begin assertion: %v", err)
	}
	_, _, err = service.FinishAssertion(context.Background(), webauthn.FinishAssertionRequest{
		TenantID:   bravo,
		RPID:       "bravo.cypra.localhost",
		CeremonyID: ceremonyID,
	})
	if !errors.Is(err, webauthn.ErrCeremonyInvalid) {
		t.Fatalf("finish error = %v, want invalid ceremony", err)
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
