# ADR-0011: Personal Access Tokens

Status: accepted

Cypra will scope personal access tokens to tenant, permissions, rotation, and audit semantics so SDK and admin API access is revocable.

## Context

Operators and tenant admins need non-browser API access for scripts, SDKs, and automation. These credentials must be revocable, scoped to a tenant, and never stored in plaintext.

## Decision

PATs are opaque 32-byte random bearer tokens. Cypra stores only SHA-256 hashes in `personal_access_tokens`. Tokens are revealed once at creation, then list endpoints return metadata only.

Each PAT is scoped to `tenant_id`, issuing `user_id`, and a string permission set. Revocation sets `revoked_at`; authentication updates `last_used_at` and rejects revoked hashes.

## Consequences

- Lost plaintext PATs cannot be recovered, only replaced.
- Token lists are safe to render in the dashboard because the secret is never returned after creation.
- Future role-downgrade hooks can revoke affected tokens by tenant/user scope.
