// Package db contains Cypra's tenant-scoped database boundary.
package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ErrTenantContextMissing is returned when a tenant-owned operation lacks a non-zero tenant id.
var ErrTenantContextMissing = errors.New("tenant context missing")

type tenantContextKey struct{}

// ContextWithTenant returns a child context carrying the tenant identifier used
// by TenantFromContext-aware database calls.
func ContextWithTenant(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantContextKey{}, tenantID)
}

// TenantFromContext extracts the typed tenant identifier from ctx.
func TenantFromContext(ctx context.Context) (uuid.UUID, bool) {
	tenantID, ok := ctx.Value(tenantContextKey{}).(uuid.UUID)
	return tenantID, ok && tenantID != uuid.Nil
}
