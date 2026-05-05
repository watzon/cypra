# ADR-0009: GORM with Explicit SQL Migrations

Status: accepted

Cypra will use GORM for application queries through `TenantScopedDB` while keeping all DDL in explicit `golang-migrate` SQL files.

## Context

Cypra needs Postgres-specific features: RLS, enum types, partial unique indexes, column-level grants, CITEXT, JSONB, and custom role privileges. These are security boundaries, not incidental persistence details.

GORM is useful for query construction and mapping Go structs, but GORM automigration is the wrong source of truth for security-sensitive DDL.

## Decision

DDL lives in explicit SQL migrations under `db/migrations` using up/down pairs. GORM models in `internal/models` mirror those tables but do not carry index tags and do not own schema evolution.

Application queries use GORM only behind `internal/db.TenantScopedDB`, which sets the tenant RLS GUC before tenant-owned statements. Test setup and migration harness code may use raw `database/sql` as an infrastructure concern.

Migration tests apply every up migration, populate representative rows, apply every down migration in reverse order, and apply the up migrations again. This proves down files are not decorative and that a populated development database can round-trip cleanly.

## Consequences

- Reviews can reason about schema, grants, and policies from SQL alone.
- GORM remains a runtime convenience rather than a DDL authority.
- New tables require a migration, a model mirror when app code reads them, and tests for any tenant-isolation or privilege behavior.
- The project must keep migration order deterministic and down files maintained.
