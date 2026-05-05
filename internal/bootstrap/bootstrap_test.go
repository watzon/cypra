package bootstrap_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/watzon/cypra/internal/bootstrap"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestBootstrapMintRedeemCreateAdminAndRefuseSecondMint(t *testing.T) {
	harness := dbtest.New(t)
	var log bytes.Buffer
	service := bootstrap.NewService(harness.SQL, slog.New(slog.NewTextHandler(&log, nil)))
	ctx := context.Background()

	token, err := service.MintSetupToken(ctx)
	if err != nil {
		t.Fatalf("mint setup token: %v", err)
	}
	if !strings.Contains(log.String(), "redacted-on-export=true") {
		t.Fatalf("setup token log missing redaction marker: %s", log.String())
	}
	setupTokenID, err := service.RedeemSetupToken(ctx, token)
	if err != nil {
		t.Fatalf("redeem setup token: %v", err)
	}
	if _, err := service.CreateFirstInstanceAdmin(ctx, setupTokenID, "admin@example.com", "Admin"); err != nil {
		t.Fatalf("create first admin: %v", err)
	}
	if _, err := service.MintSetupToken(ctx); !errors.Is(err, bootstrap.ErrBootstrapUnavailable) {
		t.Fatalf("second mint error = %v, want unavailable", err)
	}
}

func TestBootstrapResetRemintsAfterPlaintextLostCrash(t *testing.T) {
	harness := dbtest.New(t)
	service := bootstrap.NewService(harness.SQL, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	ctx := context.Background()

	lostToken, err := service.MintSetupToken(ctx)
	if err != nil {
		t.Fatalf("mint setup token: %v", err)
	}
	newToken, err := service.RevokeSetupToken(ctx)
	if err != nil {
		t.Fatalf("reset setup token: %v", err)
	}
	if newToken == lostToken {
		t.Fatal("reset returned the lost token")
	}
	if _, err := service.RedeemSetupToken(ctx, lostToken); !errors.Is(err, bootstrap.ErrInvalidSetupToken) {
		t.Fatalf("lost token redeem error = %v, want invalid", err)
	}
	if _, err := service.RedeemSetupToken(ctx, newToken); err != nil {
		t.Fatalf("redeem new token: %v", err)
	}
}

func TestBootstrapRevokeRefusesAfterAdminExists(t *testing.T) {
	harness := dbtest.New(t)
	service := bootstrap.NewService(harness.SQL, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	ctx := context.Background()
	token, err := service.MintSetupToken(ctx)
	if err != nil {
		t.Fatalf("mint setup token: %v", err)
	}
	setupTokenID, err := service.RedeemSetupToken(ctx, token)
	if err != nil {
		t.Fatalf("redeem setup token: %v", err)
	}
	if _, err := service.CreateFirstInstanceAdmin(ctx, setupTokenID, "admin@example.com", "Admin"); err != nil {
		t.Fatalf("create first admin: %v", err)
	}
	if _, err := service.RevokeSetupToken(ctx); !errors.Is(err, bootstrap.ErrBootstrapUnavailable) {
		t.Fatalf("revoke after admin error = %v, want unavailable", err)
	}
}

func TestBootstrapRejectsNilAndExpiredSetupTokens(t *testing.T) {
	harness := dbtest.New(t)
	service := bootstrap.NewService(harness.SQL, nil)
	ctx := context.Background()
	now := time.Unix(100, 0).UTC()
	service.SetNow(func() time.Time { return now })

	if _, err := service.CreateFirstInstanceAdmin(ctx, uuid.Nil, "admin@example.com", "Admin"); !errors.Is(err, bootstrap.ErrInvalidSetupToken) {
		t.Fatalf("nil setup token error = %v, want invalid", err)
	}

	token, err := service.MintSetupToken(ctx)
	if err != nil {
		t.Fatalf("mint setup token: %v", err)
	}
	service.SetNow(func() time.Time { return now.Add(bootstrap.SetupTokenTTL + time.Second) })
	if _, err := service.RedeemSetupToken(ctx, token); !errors.Is(err, bootstrap.ErrInvalidSetupToken) {
		t.Fatalf("expired setup token error = %v, want invalid", err)
	}
}

func TestBootstrapPropagatesDatabaseErrors(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://invalid")
	if err != nil {
		t.Fatalf("open db handle: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close db handle: %v", err)
	}
	service := bootstrap.NewService(db, nil)
	ctx := context.Background()

	if _, err := service.IsFirstBoot(ctx); err == nil {
		t.Fatal("IsFirstBoot unexpectedly succeeded on closed db")
	}
	if _, err := service.MintSetupToken(ctx); err == nil {
		t.Fatal("MintSetupToken unexpectedly succeeded on closed db")
	}
	if _, err := service.RedeemSetupToken(ctx, "token"); err == nil {
		t.Fatal("RedeemSetupToken unexpectedly succeeded on closed db")
	}
	if _, err := service.CreateFirstInstanceAdmin(ctx, uuid.New(), "admin@example.com", "Admin"); err == nil {
		t.Fatal("CreateFirstInstanceAdmin unexpectedly succeeded on closed db")
	}
	if _, err := service.RevokeSetupToken(ctx); err == nil {
		t.Fatal("RevokeSetupToken unexpectedly succeeded on closed db")
	}
}
