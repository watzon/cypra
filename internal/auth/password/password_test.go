package password_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth/password"
	"github.com/watzon/cypra/internal/crypto"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestPasswordSetVerifyAndReset(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := seedUser(t, harness, tenantID)
	service := password.Service{DB: harness.SQL}
	ctx := context.Background()

	if err := service.SetPassword(ctx, tenantID, userID, "correct horse battery staple"); err != nil {
		t.Fatalf("set password: %v", err)
	}
	if err := service.Verify(ctx, userID, "correct horse battery staple"); err != nil {
		t.Fatalf("verify password: %v", err)
	}
	if err := service.Verify(ctx, userID, "wrong"); !errors.Is(err, crypto.ErrPasswordMismatch) {
		t.Fatalf("wrong password error = %v", err)
	}

	token, err := service.IssueResetToken(ctx, tenantID, userID, time.Hour)
	if err != nil {
		t.Fatalf("issue reset: %v", err)
	}
	if _, err := service.ResetPassword(ctx, token, "new correct horse battery staple"); err != nil {
		t.Fatalf("reset password: %v", err)
	}
	if err := service.Verify(ctx, userID, "new correct horse battery staple"); err != nil {
		t.Fatalf("verify reset password: %v", err)
	}
	if _, err := service.ResetPassword(ctx, token, "another password"); !errors.Is(err, password.ErrInvalidResetToken) {
		t.Fatalf("reuse reset error = %v", err)
	}
}

func TestPasswordRejectsCommonPassword(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := seedUser(t, harness, tenantID)
	service := password.Service{DB: harness.SQL}
	if err := service.SetPassword(context.Background(), tenantID, userID, "password"); !errors.Is(err, crypto.ErrPasswordRejected) {
		t.Fatalf("set common password error = %v", err)
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
