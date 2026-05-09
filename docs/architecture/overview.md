# Architecture Overview

Cypra is a single-process, self-hosted authentication platform. The production artifact is one Go binary with the React dashboard embedded from `dashboard/dist`.

## Runtime Components

- `cmd/cypra serve` starts the HTTP server and routes `/dashboard`, `/login`, `/oidc`, `/api/v1`, `/.well-known`, `/setup`, `/healthz`, `/readyz`, `/metrics`, and `/storage`.
- The tenant resolver derives tenant context from `Host`: the bare install domain is instance-admin scope, while `<tenant>.<install-domain>` is tenant scope.
- `internal/db.TenantScopedDB` is the application DB boundary for tenant-owned data. It requires a tenant id and sets `cypra.tenant_id` for Postgres RLS.
- Hosted login is Go templates and HTMX, not React. This keeps the security-critical flow small and per-tenant.
- The dashboard is a Vite/React/Tailwind SPA for instance and tenant administration.
- The OIDC provider exposes per-tenant discovery, JWKS, authorize, token, userinfo, revoke, and consent surfaces.
- The email worker runs in-process and dispatches rows from `email_outbox` through terminal, SMTP, or Resend backends.
- The storage abstraction supports local disk and S3-compatible backends. Local disk serves HMAC-signed `/storage` URLs; S3-compatible storage returns native presigned URLs.

## Data Flow

An OIDC sign-in starts at `/oidc/authorize` on the tenant host. Cypra validates client id, redirect URI, scopes, and PKCE, then sends the browser through hosted login. After authentication and optional consent, Cypra issues an authorization code. The downstream server exchanges that code at `/oidc/token`, where Cypra validates PKCE and client auth, mints an ID token, access token, and refresh token, and stores refresh-token family state for reuse detection.

## Isolation Model

Tenant isolation is defense in depth:

- The HTTP tenant resolver decides tenant context from the host before handlers run.
- Application code uses `TenantScopedDB`, which rejects missing tenant ids.
- Postgres RLS policies enforce `tenant_id = current_setting('cypra.tenant_id')::uuid` for tenant-owned tables.

Instance-admin operations are intentionally outside tenant scope and must use the documented `db.AsInstanceAdmin(ctx)` escape hatch. Those surfaces are limited to instance administration, diagnostics, recovery, and GDPR/operator workflows.

## Operational Boundaries

Cypra depends on Postgres and a master key. Operators may add Caddy, Prometheus, an OTLP collector, S3-compatible storage, and stdout log shipping. Cypra deliberately does not bundle a queue, Redis, or an error tracker in v0.1.
