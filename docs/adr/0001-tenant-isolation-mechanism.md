# ADR-0001: Tenant Isolation Mechanism

Status: proposed

Cypra will enforce tenant isolation with `TenantScopedDB`, Postgres RLS, a connection checkout reset hook, and a CI fuzzer that mutates tenant context on registered handlers.
