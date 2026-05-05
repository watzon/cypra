package pat_test

import (
	"context"
	"errors"
	"testing"

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
