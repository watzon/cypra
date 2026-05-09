package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/watzon/cypra/internal/db"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestContextAsInstanceAdmin(t *testing.T) {
	ctx := db.ContextAsInstanceAdmin(context.Background())
	if !db.IsInstanceAdminContext(ctx) {
		t.Fatal("context was not marked instance-admin")
	}
}

func TestAsInstanceAdminRequiresContextAndReason(t *testing.T) {
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "host=127.0.0.1 user=invalid dbname=invalid sslmode=disable",
		PreferSimpleProtocol: true,
	}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("open inert gorm db: %v", err)
	}

	tenantDB := db.NewTenantScopedDB(gormDB)
	if _, err := tenantDB.AsInstanceAdmin(context.Background(), "recovery"); !errors.Is(err, db.ErrInstanceAdminContextRequired) {
		t.Fatalf("missing context error = %v", err)
	}
	ctx := db.ContextAsInstanceAdmin(context.Background())
	if _, err := tenantDB.AsInstanceAdmin(ctx, "   "); !errors.Is(err, db.ErrInstanceAdminReasonRequired) {
		t.Fatalf("missing reason error = %v", err)
	}
	access, err := tenantDB.AsInstanceAdmin(ctx, "tenant admin recovery")
	if err != nil {
		t.Fatalf("as instance admin: %v", err)
	}
	if access == nil {
		t.Fatal("access layer is nil")
	}
	if err := access.Exec(ctx, "unknown", "SELECT 1"); !errors.Is(err, db.ErrInstanceAdminActionDenied) {
		t.Fatalf("unknown action error = %v", err)
	}
}
