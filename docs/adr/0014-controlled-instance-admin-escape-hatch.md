# ADR-0014: Controlled Instance-Admin Escape Hatch

Status: accepted

Cypra will provide a narrow instance-admin database escape hatch for operator CLI and instance-level API paths that must intentionally cross tenant boundaries.

## Context

Most Cypra data access is tenant-scoped. Some operator actions, such as listing tenants, recovering instance admins, and revoking all tenant tokens, are intentionally cross-tenant. These paths need to be explicit so tenant isolation remains the default.

## Decision

`internal/db` exposes `ContextAsInstanceAdmin` and `TenantScopedDB.AsInstanceAdmin(ctx, reason)`. Call sites must opt into the marker and provide a reason before using cross-tenant access. The returned access layer is intentionally constrained: it exposes only allowlisted instance-admin actions and writes an audit entry with `cross_tenant=true` before running the operation.

Direct raw `TenantScopedDB.DB()` access is not available. Production HTTP handlers must use tenant-scoped helpers by default; cross-tenant access is reserved for documented instance-admin paths and operator recovery workflows.

## Consequences

- Cross-tenant intent is visible at call sites instead of hidden in generic query helpers.
- Cross-tenant use has a narrow API surface, a reason string, and an audit trail instead of unrestricted `*gorm.DB` access.
- RLS remains the database backstop; app-level helpers do not weaken runtime role grants.
