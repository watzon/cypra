package magiclink_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth/magiclink"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestMagicLinkIssueDispatchAndConsume(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	service := magiclink.Service{DB: harness.SQL}
	token, err := service.Issue(context.Background(), tenantID, &userID, "user@example.com", "https://app.example", time.Hour)
	if err != nil {
		t.Fatalf("issue magic link: %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM email_outbox WHERE template = 'magic-link'`, 1)
	gotUserID, email, err := service.Consume(context.Background(), token)
	if err != nil {
		t.Fatalf("consume magic link: %v", err)
	}
	if gotUserID != userID || email != "user@example.com" {
		t.Fatalf("consume result user=%s email=%s", gotUserID, email)
	}
	if _, _, err := service.Consume(context.Background(), token); !errors.Is(err, magiclink.ErrInvalidToken) {
		t.Fatalf("reuse error = %v", err)
	}
}
