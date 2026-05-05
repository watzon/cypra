package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/db"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestTenantScopedDBRawRejectsZeroTenantID(t *testing.T) {
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "host=127.0.0.1 user=invalid dbname=invalid sslmode=disable",
		PreferSimpleProtocol: true,
	}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("open inert gorm db: %v", err)
	}

	tenantDB := db.NewTenantScopedDB(gormDB)
	result := tenantDB.Raw(context.Background(), "select 1", uuid.Nil)
	if !errors.Is(result.Error, db.ErrTenantContextMissing) {
		t.Fatalf("Raw error = %v, want %v", result.Error, db.ErrTenantContextMissing)
	}
}

func TestTenantFromContextRejectsMissingTenant(t *testing.T) {
	if _, ok := db.TenantFromContext(context.Background()); ok {
		t.Fatal("empty context unexpectedly had tenant")
	}
	if _, ok := db.TenantFromContext(db.ContextWithTenant(context.Background(), uuid.Nil)); ok {
		t.Fatal("nil tenant unexpectedly accepted")
	}
}
