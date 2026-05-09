# Threat Model

## Actors

- Opportunistic credential-stuffers attempting account takeover.
- Targeted attackers against tenant admins or end users.
- Malicious tenants attempting cross-tenant data access.
- Malicious downstream OIDC clients.
- Future hostile insiders on a multi-admin instance.

## Assets

- End-user accounts, sessions, refresh tokens, and passkeys.
- OIDC signing keys and client secrets.
- SMTP/Resend credentials and storage credentials.
- Tenant data, audit logs, profile pictures, and metadata.
- Instance-admin access and bootstrap/recovery tokens.

## Defenses

- Tenant isolation is enforced by tenant resolver, `TenantScopedDB`, and Postgres RLS.
- WebAuthn RP IDs are per tenant, so passkey assertions are not portable across tenants.
- OIDC issuers and signing-key `kid` values are per tenant.
- Redirect URI matching is exact and rejects fragments.
- PKCE is mandatory for authorization-code flows.
- Refresh tokens are opaque, hashed server-side, rotated on every use, and revoked by family on reuse detection.
- Stored secrets use AES-GCM envelope encryption through `internal/crypto` with the KEK supplied by `MASTER_KEY` or `MASTER_KEY_FILE`.
- Passwords use Argon2id and common-password rejection.
- Google upstream OAuth validates `state` and `nonce`.
- Audit logs are append-only for the runtime role; GDPR delete redacts audit PII instead of deleting rows.

## Out Of Scope

- An attacker who already has the live `MASTER_KEY`.
- Operator-misconfigured Postgres, backups, DNS, or reverse proxies.
- SOC 2, HIPAA, FedRAMP, ISO 27001, and PCI certification.
- Built-in bot provider integrations; v0.1 ships the verifier seam and rate limiter, with Turnstile/hCaptcha planned later.

## Secret Handling

Plaintext secrets are shown once where needed: bootstrap setup token, recovery magic links, PAT creation, OIDC client-secret rotation, and backup codes. Those code paths must tag logs with `redacted-on-export` and must never persist plaintext. Export/import re-wraps envelope-encrypted columns under the destination KEK.

## GDPR Posture

Cypra is self-hosted. The operator is the controller; Cypra-the-project is not a processor. v0.1 provides DSR-supporting primitives: user export, user delete, tenant delete, profile-picture purge, audit redaction, and a deletion ledger to prevent backup imports from silently resurrecting deleted users.
