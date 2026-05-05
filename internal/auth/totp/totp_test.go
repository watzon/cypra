package totp_test

import (
	"bytes"
	"context"
	"encoding/base32"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth/totp"
	"github.com/watzon/cypra/internal/crypto"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestTOTPEnrollAndVerifyWindow(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	service := totp.Service{DB: harness.SQL, KEK: bytes.Repeat([]byte{1}, crypto.MasterKeyBytes)}
	secretText, uri, err := service.Enroll(context.Background(), tenantID, userID, "Cypra", "user@example.com")
	if err != nil {
		t.Fatalf("enroll totp: %v", err)
	}
	if !strings.HasPrefix(uri, "otpauth://totp/") {
		t.Fatalf("uri = %q", uri)
	}
	secret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secretText)
	if err != nil {
		t.Fatalf("decode secret: %v", err)
	}
	now := time.Unix(1_700_000_000, 0).UTC()
	ok, err := service.Verify(context.Background(), userID, totp.GenerateCode(secret, now.Add(30*time.Second)), now)
	if err != nil || !ok {
		t.Fatalf("verify totp ok=%v err=%v", ok, err)
	}
	ok, err = service.Verify(context.Background(), userID, "000000", now)
	if err != nil || ok {
		t.Fatalf("bad totp ok=%v err=%v", ok, err)
	}
}
