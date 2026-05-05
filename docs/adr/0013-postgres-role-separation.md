# ADR-0013: Postgres Role Separation

Status: accepted

Cypra will separate runtime and migration database roles, including narrow audit-log grants and CI checks that runtime cannot delete audit entries.

## Context

The audit log is an append-only security control. If the runtime database role can freely update or delete audit rows, application compromise can erase evidence of abuse.

Operators also need a migration role that can run DDL without granting the application process those privileges during normal serving.

## Decision

Phase 1 creates two logical Postgres roles:

- `cypra_migrate` owns migration-time privileges and is marked `BYPASSRLS` so schema/data migrations can operate across tenants intentionally.
- `cypra_runtime` has normal DML on application tables, but only `SELECT`, `INSERT`, and a column-limited `UPDATE` on `audit_entries`.

`cypra_runtime` cannot `DELETE` from `audit_entries`, and cannot update immutable columns such as `action`, `resource_kind`, `resource_id`, `actor_kind`, or `actor_id`. It can update only redaction fields required by DSR flows: `redacted_at`, `state_before`, `state_after`, `metadata`, `ip`, and `user_agent`.

Integration tests create an ephemeral login role that inherits `cypra_runtime`, insert an audit row, then assert immutable-field updates and deletes fail with insufficient privilege.

## Consequences

- `DATABASE_URL` and `MIGRATE_DATABASE_URL` must remain separate in operator configuration.
- Boot code in a later phase must warn if migration falls back to the runtime URL.
- Audit redaction is possible without granting evidence-destroying powers to the application.
- Any future audit column must be reviewed against the grant list before migration acceptance.
