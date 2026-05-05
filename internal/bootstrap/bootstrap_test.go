package bootstrap_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

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
