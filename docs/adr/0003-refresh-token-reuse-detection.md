# ADR-0003: Refresh Token Reuse Detection

Status: accepted

Cypra will model refresh tokens as revocable families so presenting an already-consumed token cascades revocation across the family.

## Context

Refresh tokens are bearer credentials. If an already-consumed token is presented, Cypra must assume a replay or theft event and invalidate the entire token lineage rather than issuing another child token.

## Decision

Refresh tokens are opaque 32-byte random values. Only SHA-256 hashes are stored. Each token belongs to a `family_id`; rotations insert a child token with `parent_id` pointing at the consumed parent.

Consuming a live refresh token atomically marks it consumed and inserts a child token in the same family. Re-presenting a consumed, expired, or revoked token triggers family-wide revocation with `revoke_reason='reuse_detected'` and emits an audit row.

## Consequences

- A stolen refresh token cannot be used quietly after the legitimate client rotates it.
- Clients must persist the latest refresh token after every successful refresh response.
- Incident response can identify affected families and users from audit rows.
- Refresh tokens remain export/import-safe because only opaque hashes are persisted.
