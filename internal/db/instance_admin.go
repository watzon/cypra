package db

import (
	"context"

	"gorm.io/gorm"
)

//revive:disable:exported

type instanceAdminKey struct{}

func ContextAsInstanceAdmin(ctx context.Context) context.Context {
	return context.WithValue(ctx, instanceAdminKey{}, true)
}

func IsInstanceAdminContext(ctx context.Context) bool {
	value, _ := ctx.Value(instanceAdminKey{}).(bool)
	return value
}

func (tdb *TenantScopedDB) AsInstanceAdmin(ctx context.Context) *gorm.DB {
	return tdb.db.WithContext(ContextAsInstanceAdmin(ctx))
}
