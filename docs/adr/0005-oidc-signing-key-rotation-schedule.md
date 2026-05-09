# ADR-0005: OIDC Signing Key Rotation Schedule

Status: accepted

Cypra will rotate tenant OIDC signing keys on a 90-day cadence with a 30-day overlap and cache-aware JWKS behavior.

## Context

Each tenant has an OIDC issuer and signing keys. Downstream consumers cache JWKS responses, so key rotation must avoid breaking token verification during cache and session overlap windows.

## Decision

Each tenant has at most one `active` and one `overlap` signing key. `kid` is `<tenant_id>:<seq>` so cross-tenant lookup mistakes are structurally visible. Rotation moves the current active key to `overlap` for 30 days and creates a new `active` key. Due keys rotate on a 90-day cadence, and overlap keys retire after their `retires_at` timestamp.

Tenant deletion moves active/overlap keys to `sunsetting` for 30 days. JWKS-serving code in later phases must include `active`, `overlap`, and `sunsetting` keys, then prune sunsetting keys after the sunset window.

## Consequences

- Consumers can verify tokens minted before rotation while caches refresh.
- Token minting uses only `active` keys.
- Tests exercise forced rotation and sunsetting before OIDC HTTP endpoints land.
