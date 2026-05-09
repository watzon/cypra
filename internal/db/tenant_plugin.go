package db

//revive:disable:exported

import (
	"fmt"

	"github.com/watzon/cypra/internal/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

// TenantPlugin wires GORM callbacks so direct GORM use can still honor a typed
// tenant context. TenantScopedDB remains the required application boundary.
type TenantPlugin struct{}

func (TenantPlugin) Name() string {
	return "cypra_tenant_rls"
}

func (TenantPlugin) Initialize(db *gorm.DB) error {
	startSpan := func(tx *gorm.DB, operation string) {
		ctx, span := observability.StartSpan(tx.Statement.Context, "db."+operation, attribute.String("db.system", "postgresql"))
		tx.Statement.Context = ctx
		tx.InstanceSet("cypra:db_span", span)
	}
	finishSpan := func(tx *gorm.DB) {
		if spanValue, ok := tx.InstanceGet("cypra:db_span"); ok {
			if span, ok := spanValue.(trace.Span); ok {
				observability.RecordError(span, tx.Error)
				span.End()
			}
		}
	}
	before := func(operation string) func(tx *gorm.DB) {
		return func(tx *gorm.DB) {
			startSpan(tx, operation)
			if IsInstanceAdminContext(tx.Statement.Context) {
				return
			}
			tenantID, ok := TenantFromContext(tx.Statement.Context)
			if !ok {
				_ = tx.AddError(ErrTenantContextMissing)
				return
			}
			_ = tx.AddError(setTenant(tx, tenantID))
		}
	}
	after := func(tx *gorm.DB) {
		finishSpan(tx)
		if IsInstanceAdminContext(tx.Statement.Context) {
			return
		}
		_ = tx.AddError(setTenantConfig(tx, "", false))
	}

	registrations := []struct {
		name   string
		before func() error
		after  func() error
	}{
		{"query", func() error {
			return db.Callback().Query().Before("gorm:query").Register("cypra:set_tenant", before("query"))
		}, func() error {
			return db.Callback().Query().After("gorm:after_query").Register("cypra:reset_tenant", after)
		}},
		{"create", func() error {
			return db.Callback().Create().Before("gorm:create").Register("cypra:set_tenant", before("create"))
		}, func() error {
			return db.Callback().Create().After("gorm:after_create").Register("cypra:reset_tenant", after)
		}},
		{"update", func() error {
			return db.Callback().Update().Before("gorm:update").Register("cypra:set_tenant", before("update"))
		}, func() error {
			return db.Callback().Update().After("gorm:after_update").Register("cypra:reset_tenant", after)
		}},
		{"delete", func() error {
			return db.Callback().Delete().Before("gorm:delete").Register("cypra:set_tenant", before("delete"))
		}, func() error {
			return db.Callback().Delete().After("gorm:after_delete").Register("cypra:reset_tenant", after)
		}},
		{"raw", func() error {
			return db.Callback().Raw().Before("gorm:raw").Register("cypra:set_tenant", before("raw"))
		}, func() error { return db.Callback().Raw().After("gorm:raw").Register("cypra:reset_tenant", after) }},
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
