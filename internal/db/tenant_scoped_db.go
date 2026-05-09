package db

//revive:disable:exported

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantScopedDB is the only application-facing DB boundary for tenant-owned
// data. Every operation runs in a transaction with cypra.tenant_id set before
// the statement and reset before the connection returns to the pool.
type TenantScopedDB struct {
	db *gorm.DB
}

func NewTenantScopedDB(db *gorm.DB) *TenantScopedDB {
	return &TenantScopedDB{db: db}
}

func (tdb *TenantScopedDB) Find(ctx context.Context, tenantID uuid.UUID, dest any, conds ...any) error {
	return tdb.withTenant(ctx, tenantID, func(tx *gorm.DB) error {
		return tx.Find(dest, conds...).Error
	})
}

func (tdb *TenantScopedDB) FindWithContext(ctx context.Context, dest any, conds ...any) error {
	tenantID, ok := TenantFromContext(ctx)
	if !ok {
		return ErrTenantContextMissing
	}
	return tdb.Find(ctx, tenantID, dest, conds...)
}

func (tdb *TenantScopedDB) Create(ctx context.Context, tenantID uuid.UUID, value any) error {
	return tdb.withTenant(ctx, tenantID, func(tx *gorm.DB) error {
		return tx.Create(value).Error
	})
}

func (tdb *TenantScopedDB) Update(ctx context.Context, tenantID uuid.UUID, model any, updates any) error {
	return tdb.withTenant(ctx, tenantID, func(tx *gorm.DB) error {
		return tx.Model(model).Updates(updates).Error
	})
}

func (tdb *TenantScopedDB) Delete(ctx context.Context, tenantID uuid.UUID, value any, conds ...any) error {
	return tdb.withTenant(ctx, tenantID, func(tx *gorm.DB) error {
		return tx.Delete(value, conds...).Error
	})
}

func (tdb *TenantScopedDB) Raw(ctx context.Context, stmt string, tenantID uuid.UUID, args ...any) *gorm.DB {
	if tenantID == uuid.Nil {
		result := tdb.db.WithContext(ctx)
		_ = result.AddError(ErrTenantContextMissing)
		return result
	}
	if strings.TrimSpace(stmt) == "" {
		result := tdb.db.WithContext(ctx)
		_ = result.AddError(fmt.Errorf("raw statement empty"))
		return result
	}

	return tdb.db.WithContext(ContextWithTenant(ctx, tenantID)).Raw(stmt, args...)
}

func (tdb *TenantScopedDB) RawScan(ctx context.Context, stmt string, tenantID uuid.UUID, dest any, args ...any) error {
	if tenantID == uuid.Nil {
		return ErrTenantContextMissing
	}
	return tdb.withTenant(ctx, tenantID, func(tx *gorm.DB) error {
		return tx.Raw(stmt, args...).Scan(dest).Error
	})
}

func (tdb *TenantScopedDB) Transaction(ctx context.Context, tenantID uuid.UUID, fn func(tx *gorm.DB) error) error {
	return tdb.withTenant(ctx, tenantID, fn)
}

func (tdb *TenantScopedDB) withTenant(ctx context.Context, tenantID uuid.UUID, fn func(tx *gorm.DB) error) error {
	if tenantID == uuid.Nil {
		return ErrTenantContextMissing
	}
	return tdb.db.WithContext(ContextWithTenant(ctx, tenantID)).Transaction(func(tx *gorm.DB) error {
		if err := setTenant(tx, tenantID); err != nil {
			return err
		}
		return resetTenantAfter(tx, func() error {
			return fn(tx)
		})
	})
}

func setTenant(tx *gorm.DB, tenantID uuid.UUID) error {
	return setTenantConfig(tx, tenantID.String(), true)
}

func resetTenantAfter(tx *gorm.DB, fn func() error) error {
	err := fn()
	resetErr := setTenantConfig(tx, "", false)
	if err != nil {
		return err
	}
	return resetErr
}

func setTenantConfig(tx *gorm.DB, value string, local bool) error {
	ctx := tx.Statement.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if tx.Statement.ConnPool == nil {
		return tx.Exec("SELECT set_config('cypra.tenant_id', ?, ?)", value, local).Error
	}
	_, err := tx.Statement.ConnPool.ExecContext(ctx, "SELECT set_config('cypra.tenant_id', $1, $2)", value, local)
	return err
}
