# ADR-0002: OIDC Provider Surface

Status: accepted

Cypra will implement a lean per-tenant OIDC provider surface with discovery, JWKS, authorize, token, userinfo, revoke, consent, and conformance-suite coverage.

## Context

Cypra's primary integration contract is OIDC. Each tenant needs an issuer scoped to its subdomain so downstream applications can isolate clients, signing keys, users, and consent records by tenant.

The v1 provider surface should stay intentionally small: auth-code + PKCE for browser apps, refresh-token rotation, per-tenant discovery/JWKS, userinfo, revocation, and consent recording. Higher-risk grant types are excluded.

## Decision

Cypra exposes a per-tenant issuer at `https://<tenant>.<install-domain>`. Discovery advertises only `response_type=code`, `grant_type=authorization_code|refresh_token`, `code_challenge_method=S256`, and token endpoint auth methods `client_secret_basic`, `client_secret_post`, and `none`.

JWKS returns active, overlap, and still-valid sunsetting keys with `Cache-Control: public, max-age=300, must-revalidate`. Discovery uses `Cache-Control: public, max-age=600, must-revalidate`.

Redirect URI matching is strict: scheme, port, path, query, and trailing slash are exact; host is case-folded; fragments are rejected.

## Consequences

- Cross-tenant verification is structurally avoided because issuers, clients, and signing keys are tenant-scoped.
- Public clients must use PKCE S256.
- Downstream apps get stable discovery/JWKS cache behavior.
- Human-visible login and consent UI can evolve later without changing the OIDC protocol surface.
