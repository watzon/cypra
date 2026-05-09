package db

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
)

//revive:disable:exported

type instanceAdminKey struct{}

var (
	ErrInstanceAdminContextRequired = errors.New("instance admin context required")
	ErrInstanceAdminReasonRequired  = errors.New("instance admin reason required")
	ErrInstanceAdminActionDenied    = errors.New("instance admin action denied")
)

var allowedInstanceAdminActions = map[string]struct{}{
	"bootstrap.recovery":       {},
	"cli.admin":                {},
	"instance_admin.read":      {},
	"instance_admin.recovery":  {},
	"master_key.rotation":      {},
	"tenant_admin.recovery":    {},
	"tenant_deletion.recovery": {},
}

type InstanceAdminDB struct {
	db     *gorm.DB
	reason string
}

func ContextAsInstanceAdmin(ctx context.Context) context.Context {
	return context.WithValue(ctx, instanceAdminKey{}, true)
}

func IsInstanceAdminContext(ctx context.Context) bool {
	value, _ := ctx.Value(instanceAdminKey{}).(bool)
	return value
}

func (tdb *TenantScopedDB) AsInstanceAdmin(ctx context.Context, reason string) (*InstanceAdminDB, error) {
	if !IsInstanceAdminContext(ctx) {
		return nil, ErrInstanceAdminContextRequired
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, ErrInstanceAdminReasonRequired
	}
	return &InstanceAdminDB{db: tdb.db.WithContext(ctx), reason: reason}, nil
}

func (adb *InstanceAdminDB) Exec(ctx context.Context, action string, stmt string, args ...any) error {
	return adb.withAudit(ctx, action, func(tx *gorm.DB) error {
		return tx.Exec(stmt, args...).Error
	})
}

func (adb *InstanceAdminDB) RawScan(ctx context.Context, action string, stmt string, dest any, args ...any) error {
	return adb.withAudit(ctx, action, func(tx *gorm.DB) error {
		return tx.Raw(stmt, args...).Scan(dest).Error
	})
}

func (adb *InstanceAdminDB) withAudit(ctx context.Context, action string, fn func(tx *gorm.DB) error) error {
	action = strings.TrimSpace(action)
	if _, ok := allowedInstanceAdminActions[action]; !ok {
		return ErrInstanceAdminActionDenied
	}
	return adb.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			`INSERT INTO audit_entries (tenant_id, actor_kind, actor_id, action, resource_kind, metadata)
			 VALUES (NULL, 'instance_admin', NULL, ?::text, 'instance_admin_escape_hatch', jsonb_build_object('cross_tenant', true, 'reason', ?::text))`,
			action,
			adb.reason,
		).Error; err != nil {
			return err
		}
		return fn(tx)
	})
}
