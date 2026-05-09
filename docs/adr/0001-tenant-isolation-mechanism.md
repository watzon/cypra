# ADR-0001: Tenant Isolation Mechanism

Status: accepted

Cypra will enforce tenant isolation with `TenantScopedDB`, Postgres RLS, a connection checkout reset hook, and a CI fuzzer that mutates tenant context on registered handlers.

## Context

Cypra is a self-hosted multi-tenant auth service. A single install stores many tenants' users, credentials, clients, sessions, and audit data in one Postgres database. A tenant-isolation bug is therefore a product-severity security failure.

PLAN §8 requires two independent layers:

- Application code refuses tenant-owned queries without an explicit tenant id.
- Postgres RLS returns zero rows when `cypra.tenant_id` is missing or mismatched.

## Decision

All tenant-owned database access goes through `internal/db.TenantScopedDB`.

`TenantScopedDB` requires a non-zero `uuid.UUID` tenant id on public methods, or a typed context value inserted by `db.ContextWithTenant`. It never reads tenant ids from stringly-typed context keys. Missing tenant context returns `db.ErrTenantContextMissing`.

Each operation runs in a transaction, executes `SET LOCAL cypra.tenant_id = <tenant>`, performs the query, then runs `SELECT set_config('cypra.tenant_id', '', false)` before the connection returns to the pool. The reset uses `set_config` because bare `RESET cypra.tenant_id` can fail when the custom GUC was never set on that pooled connection.

Every tenant-owned table carries a direct `tenant_id` column and has an RLS policy named `tenant_isolation`. Tables with instance-level rows, such as `audit_entries`, `pending_invitations`, `email_outbox`, and `rate_limit_buckets`, allow `tenant_id IS NULL` where PLAN explicitly requires instance-scoped records.

The Phase 1 fuzzer harness mutates request tenant context and asserts registered handlers return zero rows, permission denied, or not found. Phase 3 and later phases must enroll each new HTTP handler when it is introduced.

## Consequences

- Tenant isolation is testable at the DB layer before HTTP exists.
- Raw SQL remains available for advanced cases, but only through `TenantScopedDB.Raw` / `RawScan` with an explicit tenant id.
- Schema summaries that imply tenant ownership through a user relationship still get a direct `tenant_id`; this keeps RLS uniform and avoids join-dependent policies on security-critical credential tables.
- Any direct `*gorm.DB` use in tenant-owned application code is a review blocker unless it is migration, test setup, or instance-admin-only code touching non-tenant tables.
