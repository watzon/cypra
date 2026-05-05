# ADR-0014: Controlled Instance-Admin Escape Hatch

Status: accepted

Cypra will provide a narrow instance-admin database escape hatch for operator CLI and instance-level API paths that must intentionally cross tenant boundaries.

## Context

Most Cypra data access is tenant-scoped. Some operator actions, such as listing tenants, recovering instance admins, and revoking all tenant tokens, are intentionally cross-tenant. These paths need to be explicit so tenant isolation remains the default.

## Decision

`internal/db` exposes `ContextAsInstanceAdmin` and `TenantScopedDB.AsInstanceAdmin(ctx)`. Call sites must opt into the marker before using cross-tenant access. Phase 6 CLI admin paths use direct DB access for break-glass operations and log one-time secrets with `redacted-on-export=true`.

## Consequences

- Cross-tenant intent is visible at call sites instead of hidden in generic query helpers.
- Future lint rules can restrict `AsInstanceAdmin` imports to CLI admin and instance API packages.
- RLS remains the database backstop; app-level helpers do not weaken runtime role grants.
