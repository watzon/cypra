# internal/db

`internal/db` is Cypra's tenant-isolation boundary. Application code that reads or writes tenant-owned rows must use `TenantScopedDB`, not raw `*gorm.DB`.

## Contract

- Every public `TenantScopedDB` method requires an explicit non-zero `uuid.UUID` tenant id or a typed context value from `ContextWithTenant`.
- Missing tenant context returns `ErrTenantContextMissing`.
- Each operation runs inside a transaction, sets `cypra.tenant_id` with `SET LOCAL`, executes the statement, then resets the GUC with `SELECT set_config('cypra.tenant_id', '', false)` before the connection returns to the pool.
- Raw SQL is allowed only through `Raw` / `RawScan`, and still requires a tenant id.

Postgres RLS is the second line of defense. It should make a tenant-missing or tenant-mismatched query return zero rows even if a caller accidentally bypasses the higher-level API.
