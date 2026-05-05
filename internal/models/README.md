# internal/models

`internal/models` contains GORM structs that mirror the SQL schema in `db/migrations`.

## Contract

- Migrations are the source of truth for DDL, indexes, constraints, grants, and RLS.
- Model structs do not define `gorm:"index"` tags.
- Tenant-owned models carry a direct `TenantID` field so RLS policies can use a uniform `tenant_id = current_setting('cypra.tenant_id')` predicate.
- Models are intentionally thin data shapes. Business rules belong in service packages and database access belongs behind `internal/db.TenantScopedDB`.
