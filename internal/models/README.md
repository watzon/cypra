# internal/models

`internal/models` contains GORM structs that mirror the SQL schema in `db/migrations`.

## Contract

- Migrations are the source of truth for DDL, indexes, constraints, grants, and RLS.
- Model structs do not define `gorm:"index"` tags.
- Tenant-owned models carry a direct `TenantID` field so RLS policies can use a uniform `tenant_id = current_setting('cypra.tenant_id')` predicate.
- Models are intentionally thin data shapes. Business rules belong in service packages and database access belongs behind `internal/db.TenantScopedDB`.

## Schema/model coverage notes

The structs in `models.go` intentionally mirror PLAN §8 table names for GORM-backed reads/writes. The schema remains authoritative for columns, constraints, indexes, grants, and enum definitions.

Known intentional gaps:

- `personal_access_tokens` is omitted from `models.go` because PAT operations are implemented in `internal/pat` with explicit SQL and never use GORM model reads. This keeps token-hash handling out of generic model scans.
- `pending_invitations` is the model/schema name for PLAN's historical `admin_invites` token concept. The migration header documents the rename; no separate `AdminInvite` model should be added.
- Index-only, check-constraint, grant, and RLS details are intentionally absent from struct tags. The migration and integration tests own those invariants.
- JSON/encrypted byte columns use `datatypes.JSON` or `[]byte` at the model boundary. Encryption envelope shape validation belongs to `internal/crypto`, not the model layer.
