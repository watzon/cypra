package pat_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/pat"
)

func TestPATCreateAuthenticateListAndRevoke(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	service := pat.Service{DB: harness.SQL}
	created, err := service.Create(context.Background(), tenantID, userID, "dev", []string{"projects.read"})
	if err != nil {
		t.Fatalf("create pat: %v", err)
	}
	if created.Plaintext == "" {
		t.Fatal("plaintext token not returned once")
	}
	authenticated, err := service.Authenticate(context.Background(), created.Plaintext)
	if err != nil || authenticated.ID != created.ID {
		t.Fatalf("authenticate got=%+v err=%v", authenticated, err)
	}
	tokens, err := service.List(context.Background(), tenantID, userID)
	if err != nil || len(tokens) != 1 || tokens[0].RevokedAt != nil || tokens[0].Last4 == "" {
		t.Fatalf("list tokens=%+v err=%v", tokens, err)
	}
	if err := service.Revoke(context.Background(), tenantID, userID, created.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := service.Authenticate(context.Background(), created.Plaintext); !errors.Is(err, pat.ErrInvalidToken) {
		t.Fatalf("authenticate revoked error = %v", err)
	}
}

func TestPATExpiredTokenIsInvalid(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	expiresAt := time.Now().Add(-time.Minute)
	created, err := (pat.Service{DB: harness.SQL}).CreateWithExpiry(context.Background(), tenantID, userID, "expired", []string{"projects.read"}, &expiresAt)
	if err != nil {
		t.Fatalf("create pat: %v", err)
	}
	if _, err := (pat.Service{DB: harness.SQL}).Authenticate(context.Background(), created.Plaintext); !errors.Is(err, pat.ErrInvalidToken) {
		t.Fatalf("authenticate expired error = %v", err)
	}
}

func TestPATsRevokedByMembershipDowngradeAndDelete(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO tenant_memberships (tenant_id, user_id, role) VALUES ($1, $2, 'admin')`, tenantID, userID); err != nil {
		t.Fatalf("seed membership: %v", err)
	}
	service := pat.Service{DB: harness.SQL}
	downgraded, err := service.Create(context.Background(), tenantID, userID, "downgrade", []string{"projects.read"})
	if err != nil {
		t.Fatalf("create downgrade pat: %v", err)
	}
	if _, err := harness.SQL.Exec(`UPDATE tenant_memberships SET role = 'member' WHERE tenant_id = $1 AND user_id = $2`, tenantID, userID); err != nil {
		t.Fatalf("downgrade membership: %v", err)
	}
	if _, err := service.Authenticate(context.Background(), downgraded.Plaintext); !errors.Is(err, pat.ErrInvalidToken) {
		t.Fatalf("authenticate downgraded pat error = %v", err)
	}
	if _, err := harness.SQL.Exec(`UPDATE tenant_memberships SET role = 'admin' WHERE tenant_id = $1 AND user_id = $2`, tenantID, userID); err != nil {
		t.Fatalf("restore membership: %v", err)
	}
	deleted, err := service.Create(context.Background(), tenantID, userID, "delete", []string{"projects.read"})
	if err != nil {
		t.Fatalf("create delete pat: %v", err)
	}
	if _, err := harness.SQL.Exec(`DELETE FROM tenant_memberships WHERE tenant_id = $1 AND user_id = $2`, tenantID, userID); err != nil {
		t.Fatalf("delete membership: %v", err)
	}
	if _, err := service.Authenticate(context.Background(), deleted.Plaintext); !errors.Is(err, pat.ErrInvalidToken) {
		t.Fatalf("authenticate deleted membership pat error = %v", err)
	}
}
