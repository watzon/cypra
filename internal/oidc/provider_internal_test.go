package oidc

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestProviderPropagatesClosedDatabaseErrors(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://invalid")
	if err != nil {
		t.Fatalf("open db handle: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close db handle: %v", err)
	}
	provider := Provider{DB: db, InstallDomain: "cypra.localhost"}
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()
	clientID := uuid.New()

	if _, err := provider.Authorize(ctx, AuthorizeRequest{TenantID: tenantID, ClientID: "client"}); err == nil {
		t.Fatal("Authorize unexpectedly succeeded on closed db")
	}
	if _, err := provider.Token(ctx, TokenRequest{GrantType: "authorization_code", ClientID: "client"}); err == nil {
		t.Fatal("authorization_code token unexpectedly succeeded on closed db")
	}
	if _, err := provider.Token(ctx, TokenRequest{GrantType: "refresh_token", ClientID: "client"}); err == nil {
		t.Fatal("refresh_token unexpectedly succeeded on closed db")
	}
	if _, err := provider.UserInfo(ctx, "acme", tenantID, "bad.token.value"); err == nil {
		t.Fatal("UserInfo unexpectedly succeeded on closed db")
	}
	if err := provider.Revoke(ctx, "opaque"); err == nil {
		t.Fatal("Revoke unexpectedly succeeded on closed db")
	}
	if err := provider.RecordConsent(ctx, tenantID, userID, clientID, []string{"openid"}); err == nil {
		t.Fatal("RecordConsent unexpectedly succeeded on closed db")
	}
	if _, err := provider.LoadClient(ctx, tenantID, "client"); err == nil {
		t.Fatal("LoadClient unexpectedly succeeded on closed db")
	}
	if _, err := provider.SigningKeys(ctx, tenantID); err == nil {
		t.Fatal("SigningKeys unexpectedly succeeded on closed db")
	}
	if _, err := provider.activeSigningKey(ctx, tenantID); err == nil {
		t.Fatal("activeSigningKey unexpectedly succeeded on closed db")
	}
	if err := CleanupExpired(ctx, db, time.Unix(100, 0).UTC()); err == nil {
		t.Fatal("CleanupExpired unexpectedly succeeded on closed db")
	}

	rotation := NewRotationService(db, nil)
	if err := rotation.ForceRotate(ctx, tenantID); err == nil {
		t.Fatal("ForceRotate unexpectedly succeeded on closed db")
	}
	if err := rotation.RotateDue(ctx); err == nil {
		t.Fatal("RotateDue unexpectedly succeeded on closed db")
	}
	if err := rotation.Run(ctx, 0); err == nil {
		t.Fatal("Run unexpectedly succeeded on closed db")
	}
	if err := rotation.SunsetKeys(ctx, tenantID); err == nil {
		t.Fatal("SunsetKeys unexpectedly succeeded on closed db")
	}
	if err := rotation.PruneSunsetKeys(ctx); err == nil {
		t.Fatal("PruneSunsetKeys unexpectedly succeeded on closed db")
	}
}
