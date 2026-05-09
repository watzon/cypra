package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/db"
	"github.com/watzon/cypra/internal/dbtest"
	"gorm.io/gorm"
)

type projectRow struct {
	ID       uuid.UUID `gorm:"column:id;primaryKey"`
	TenantID uuid.UUID `gorm:"column:tenant_id"`
	Slug     string    `gorm:"column:slug"`
	Name     string    `gorm:"column:name"`
}

func (projectRow) TableName() string { return "projects" }

func TestTenantScopedDBOperations(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "ops")
	tenantDB := harness.TenantDB
	ctx := context.Background()

	if _, err := tenantDB.AsInstanceAdmin(ctx, "test"); !errors.Is(err, db.ErrInstanceAdminContextRequired) {
		t.Fatalf("AsInstanceAdmin without context error = %v", err)
	}
	adminDB, err := tenantDB.AsInstanceAdmin(db.ContextAsInstanceAdmin(ctx), "integration smoke")
	if err != nil {
		t.Fatalf("AsInstanceAdmin error = %v", err)
	}
	if adminDB == nil {
		t.Fatal("AsInstanceAdmin returned nil access layer")
	}
	adminCtx := db.ContextAsInstanceAdmin(ctx)
	if err := adminDB.Exec(adminCtx, "cli.admin", `UPDATE tenants SET name = name WHERE id = ?`, tenantID); err != nil {
		t.Fatalf("instance admin exec: %v", err)
	}
	var tenantCount int
	if err := adminDB.RawScan(adminCtx, "instance_admin.read", `SELECT count(*) FROM tenants`, &tenantCount); err != nil {
		t.Fatalf("instance admin raw scan: %v", err)
	}
	if tenantCount == 0 {
		t.Fatal("instance admin raw scan saw no tenants")
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM audit_entries WHERE tenant_id IS NULL AND metadata->>'cross_tenant' = 'true'`, 2)
	if err := adminDB.Exec(adminCtx, "cli.admin", `UPDATE missing_table SET id = id`); err == nil {
		t.Fatal("instance admin exec unexpectedly succeeded for invalid statement")
	}

	plugin := db.TenantPlugin{}
	if plugin.Name() != "cypra_tenant_rls" {
		t.Fatalf("Name = %q", plugin.Name())
	}

	project := projectRow{ID: uuid.New(), TenantID: tenantID, Slug: "console", Name: "Console"}
	if err := tenantDB.Create(ctx, tenantID, &project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	var found []projectRow
	if err := tenantDB.FindWithContext(db.ContextWithTenant(ctx, tenantID), &found, "slug = ?", "console"); err != nil {
		t.Fatalf("find project from context: %v", err)
	}
	if len(found) != 1 || found[0].Name != "Console" {
		t.Fatalf("found = %#v", found)
	}

	if err := tenantDB.Update(ctx, tenantID, &project, map[string]any{"name": "Console App"}); err != nil {
		t.Fatalf("update project: %v", err)
	}

	var name string
	if err := tenantDB.RawScan(ctx, "SELECT name FROM projects WHERE slug = ?", tenantID, &name, "console"); err != nil {
		t.Fatalf("raw scan project: %v", err)
	}
	if name != "Console App" {
		t.Fatalf("name = %q", name)
	}
	var pluginCount int64
	if err := harness.Gorm.WithContext(db.ContextWithTenant(ctx, tenantID)).Raw("SELECT count(*) FROM projects WHERE slug = ?", "console").Scan(&pluginCount).Error; err != nil {
		t.Fatalf("plugin raw count: %v", err)
	}
	if pluginCount != 1 {
		t.Fatalf("plugin raw count = %d", pluginCount)
	}

	if err := tenantDB.Transaction(ctx, tenantID, func(tx *gorm.DB) error {
		return tx.Create(&projectRow{ID: uuid.New(), TenantID: tenantID, Slug: "admin", Name: "Admin"}).Error
	}); err != nil {
		t.Fatalf("transaction create: %v", err)
	}

	if err := tenantDB.Delete(ctx, tenantID, &projectRow{}, "slug = ?", "admin"); err != nil {
		t.Fatalf("delete project: %v", err)
	}
	var count int
	if err := tenantDB.RawScan(ctx, "SELECT count(*) FROM projects WHERE slug = ?", tenantID, &count, "admin"); err != nil {
		t.Fatalf("raw scan count: %v", err)
	}
	if count != 0 {
		t.Fatalf("deleted project count = %d", count)
	}
}

func TestTenantScopedDBRejectsMissingTenant(t *testing.T) {
	harness := dbtest.New(t)
	tenantDB := harness.TenantDB
	ctx := context.Background()

	if err := tenantDB.FindWithContext(ctx, &[]projectRow{}); !errors.Is(err, db.ErrTenantContextMissing) {
		t.Fatalf("FindWithContext error = %v", err)
	}
	if err := tenantDB.Create(ctx, uuid.Nil, &projectRow{}); !errors.Is(err, db.ErrTenantContextMissing) {
		t.Fatalf("Create error = %v", err)
	}
	if err := tenantDB.RawScan(ctx, "SELECT 1", uuid.Nil, new(int)); !errors.Is(err, db.ErrTenantContextMissing) {
		t.Fatalf("RawScan error = %v", err)
	}
	if err := tenantDB.Transaction(ctx, uuid.Nil, nil); !errors.Is(err, db.ErrTenantContextMissing) {
		t.Fatalf("Transaction error = %v", err)
	}
}
