package db

//revive:disable:exported

import (
	"fmt"

	"gorm.io/gorm"
)

// TenantPlugin wires GORM callbacks so direct GORM use can still honor a typed
// tenant context. TenantScopedDB remains the required application boundary.
type TenantPlugin struct{}

func (TenantPlugin) Name() string {
	return "cypra_tenant_rls"
}

func (TenantPlugin) Initialize(db *gorm.DB) error {
	before := func(tx *gorm.DB) {
		tenantID, ok := TenantFromContext(tx.Statement.Context)
		if !ok {
			_ = tx.AddError(ErrTenantContextMissing)
			return
		}
		_ = tx.AddError(setTenant(tx, tenantID))
	}
	after := func(tx *gorm.DB) {
		_ = tx.AddError(tx.Exec("SELECT set_config('cypra.tenant_id', '', false)").Error)
	}

	registrations := []struct {
		name   string
		before func() error
		after  func() error
	}{
		{"query", func() error { return db.Callback().Query().Before("gorm:query").Register("cypra:set_tenant", before) }, func() error {
			return db.Callback().Query().After("gorm:after_query").Register("cypra:reset_tenant", after)
		}},
		{"create", func() error { return db.Callback().Create().Before("gorm:create").Register("cypra:set_tenant", before) }, func() error {
			return db.Callback().Create().After("gorm:after_create").Register("cypra:reset_tenant", after)
		}},
		{"update", func() error { return db.Callback().Update().Before("gorm:update").Register("cypra:set_tenant", before) }, func() error {
			return db.Callback().Update().After("gorm:after_update").Register("cypra:reset_tenant", after)
		}},
		{"delete", func() error { return db.Callback().Delete().Before("gorm:delete").Register("cypra:set_tenant", before) }, func() error {
			return db.Callback().Delete().After("gorm:after_delete").Register("cypra:reset_tenant", after)
		}},
		{"raw", func() error { return db.Callback().Raw().Before("gorm:raw").Register("cypra:set_tenant", before) }, func() error { return db.Callback().Raw().After("gorm:raw").Register("cypra:reset_tenant", after) }},
	}

	for _, registration := range registrations {
		if err := registration.before(); err != nil {
			return fmt.Errorf("register %s set tenant callback: %w", registration.name, err)
		}
		if err := registration.after(); err != nil {
			return fmt.Errorf("register %s reset tenant callback: %w", registration.name, err)
		}
	}
	return nil
}
