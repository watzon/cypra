# Cypra — Plan (PRD + SDD)

**Companion to:** [`BRAINSTORM.md`](./BRAINSTORM.md) (sealed), [`DESIGN.md`](./DESIGN.md) (visual system, authored next).

**Status:** sealed
**Version:** 1.0
**Owner:** @watzon

---

## 1. Executive Summary

Cypra is an open-source, self-hosted, multi-tenant authentication platform written in Go and shipped as a single multi-arch container. It targets solo developers and small dev shops who want passkeys, magic links, social login, and a working OIDC provider without paying for Auth0/Clerk/WorkOS or wrestling with Keycloak/Authentik. The wedge is operational simplicity: `docker compose up` plus a one-shot setup token, and a developer's app is signing users in against Cypra OIDC the same afternoon. v1 ships hosted login only; an embeddable widget, custom CNAME domains, SAML, and a TypeScript SDK are deliberately deferred to v1.1+.

---

## 2. Product Vision

When v1 exists, a solo developer can spin up Cypra on a small VPS in an afternoon, redeem a setup token printed to its logs, enroll a passkey for the dashboard, create a tenant for themselves, and wire a Next.js app to Cypra via off-the-shelf OIDC client code (Auth.js / NextAuth) — all through a clean dashboard that exposes no env-var soup. End-users of that app sign up with a passkey on their first visit and never see a password again, while the operator owns the database, the binary, and every byte of user data. The product feels like Clerk if Clerk were a single binary you ran yourself, with none of the multi-region complexity and none of the per-MAU billing.

---

## 3. Non-Goals

- **SAML 2.0 in v1.** Deferred to a later release.
- **Project-level custom domains, ever.** Tenant-level CNAME (v1.1) is the permanent answer.
- **Hash-method-compatible migration** from Auth0/Clerk/WorkOS. Operators migrate via CSV import and a forced first-login password reset.
- **External KMS dependency** for envelope encryption. Master key is a local file or env var.
- **Embeddable JS widget in v1.** Hosted login pages only at v1; widget lands in v1.1.
- **TypeScript SDK in v1.** Go SDK only; TS lands in v1.1.
- **Dashboard bulk CSV import UI in v1.** Replaced by `cypra import` CLI at v1.
- **Built-in ACME / TLS termination.** Cypra serves HTTP behind a reverse proxy (Caddy or equivalent).
- **Cross-tenant passkey roaming.** A passkey registered for tenant A is unusable for tenant B by design. Users with multiple tenant accounts register one passkey per tenant.
- **OIDC dynamic client registration**, **token introspection**, **implicit/hybrid grants**.
- **Helm chart, Kubernetes operator, multi-region writes, multi-Cypra-instances-per-Postgres.**
- **Compliance certifications** (SOC 2, HIPAA, FedRAMP, ISO 27001, PCI). Cypra ships GDPR-enabling features only.
- **i18n / non-English UI in v1.** Strings are gettext-ready so it isn't a rewrite later.
- **Profile-picture content scanning** (AV/CSAM). Type/size/content-type sniffing only.
- **Built-in bot-mitigation providers in v1.** A pluggable verifier hook is shipped; Turnstile/hCaptcha implementations land in v1.1.
- **Hosted Cypra SaaS.**

---

## 4. Guiding Principles

1. **Self-host first; hosted never blocks features.** Every feature must work for a single-VPS operator with no managed dependencies beyond Postgres and (optionally) S3-compatible storage.
   _Why:_ Cypra's wedge is "actually self-hostable." A managed-service dependency dissolves the wedge.
2. **Configure in the dashboard, not in env.** Bootstrap-only env vars are the irreducible set (see §11). Everything else lives in the DB and is editable by an admin.
   _Why:_ ENV-hell is the most-cited reason competitors are painful, and multi-tenant config in env is impossible to evolve without redeployment.
3. **Tenant isolation is a single enforced layer with no carve-outs.** Every data access — including raw SQL on perf-critical paths — passes through `TenantScopedDB`, which requires an explicit `tenantID` argument. Postgres RLS is the unbreakable second belt.
   _Why:_ Cross-tenant leaks are the #1 multi-tenant bug class. Carve-outs become CVEs.
4. **Security-relevant defaults are correct out of the box.** Argon2id, exact redirect-URI matching, refresh-token rotation with reuse-detection, passkey-first dashboard auth, envelope-encrypted secrets, uniform error messages — none are opt-in.
   _Why:_ The persona is a solo dev who does not have time to harden auth. Sharp defaults will draw blood.
5. **Cypra-the-project is not a GDPR processor.** Cypra ships GDPR-enabling features and an operator playbook. The operator is the controller.
   _Why:_ Treating self-hosted vendors as processors is a category error that creates legal exposure with no compliance value.
6. **No surprises in upgrade paths.** Migrations run on boot by default but can be opted out (`--skip-migrate`) so change-control shops aren't ambushed.
   _Why:_ Solo devs need lean defaults; agencies need the escape hatch. Both upgrade safely.
7. **MIT, forever.** No future BSL/SSPL relicense. Moat is operational excellence and brand.
   _Why:_ Open-source-first credibility is a one-shot resource.

---

## 5. Personas and Use Cases

### Primary persona

**The solo-or-small-team self-hoster** — a developer who ships small-to-medium apps, owns their infra (VPS or Railway/Fly project), and wants auth that isn't Auth0-billing or Keycloak-yaml.
Triggered by: starting a new app, hitting a paid auth tier, or bouncing off Keycloak's setup story. Need: passkeys, magic links, Google sign-in, OIDC provider — running today, not next sprint.

### Secondary personas

- **Small SaaS founder** — wants B2B-multi-tenant auth, has 1–50 customer orgs, doesn't want WorkOS pricing.
- **Agency** — bundles auth into client deliverables; one Cypra deployment per engagement, or one shared with a tenant per client.

### Anti-personas

- **Enterprises** needing SOC 2 Type II / HIPAA / FedRAMP attestation.
- **Apps with millions of users on day one** that need horizontal scale or sub-100ms global p99.
- **Teams that want "auth as a service" with no infra responsibility.** They should buy Clerk or WorkOS.

### Canonical end-to-end story

Operator runs `docker compose up` (with-tls profile) on a small VPS. On first boot, Cypra prints a setup token to logs (with a `redacted-on-export` log field tag so log shippers can strip it). Operator opens `https://auth.example.com`, redeems the token, completes WebAuthn passkey enrollment (RP ID = `auth.example.com` for the install dashboard), exports backup codes, becomes the first **instance admin**. They create a tenant `acme` (reachable at `https://acme.auth.example.com`), add a project under it, configure the Resend email provider (mandatory before mail-dependent flows), configure Google OAuth as upstream IdP, copy the per-tenant OIDC config — **issuer URL `https://acme.auth.example.com`**, `client_id`, `client_secret` — into their Next.js app's NextAuth config. An end-user visits the Next.js app, clicks "Sign in", is redirected to Cypra's hosted login on `acme.auth.example.com`, signs up with a passkey (RP ID = `acme.auth.example.com`, scoped to that tenant), is bounced back with an ID token, and the Next.js app shows a logged-in session. Target time: ≤ 30 minutes.

### Other use cases

- **"Personal projects" tenant** — solo dev creates one tenant, registers each side project as a separate OIDC client under it, reuses one passkey for dashboard admin (one passkey per tenant they admin).
- **Migrate off Auth0/Clerk** — operator runs `cypra import users.csv --tenant=acme` and configures forced first-login password reset.
- **Add SSO to an existing app** — operator points the existing app at the per-tenant issuer URL, enables Google as upstream, end-users sign in with their existing Google account.

---

## 6. MVP Definition

### MVP theme

**One tenant, one project, one Next.js app, end-to-end, self-hosted, in an afternoon.** Features that don't show up in that narrative are explicit v1.1+ deferrals.

### MVP must-haves

- **Multi-tenant identity model** with tenants, projects, per-tenant users, three intra-tenant roles (owner/admin/member), and a separate `instance_admin` role. → §7 _Tenancy core_, §7 _Authorization_.
- **End-user auth methods**: email + password (Argon2id), magic link, passkeys (WebAuthn, per-tenant RP ID), Google OAuth upstream. → §7 _Auth methods_.
- **OIDC provider (lean)**: per-tenant issuer URL `https://<tenant>.<install-domain>`; auth code + PKCE only; refresh tokens with rotation + family-tree reuse detection; per-tenant discovery doc, JWKS, consent screen, revocation endpoint. **No** DCR, **no** introspection, **no** implicit/hybrid grants. → §7 _OIDC provider_.
- **Two-factor**: TOTP and WebAuthn second-factor. → §7 _Auth methods_.
- **Hosted login pages** (server-rendered Go templates with HTMX) on `<tenant>.<install-domain>`. → §7 _Hosted login_.
- **Dashboard** (Vite + React + Tailwind v4 + ShadCN via Watermelon UI), embedded into the Go binary via `embed.FS`. → §7 _Dashboard_.
- **Email providers**: terminal (dev), SMTP, Resend. Configuring at least one is mandatory before any tenant can issue mail-dependent flows. → §7 _Email abstraction_.
- **Email worker**: out-of-band send queue with exponential backoff, isolating auth handlers from provider latency. → §7 _Email worker_.
- **Storage abstraction**: `local-disk` (with HMAC-signed URLs proxied through Cypra) and `s3-compatible` (with native presigned URLs). → §7 _Storage_.
- **Bootstrap setup token**: single-use, printed on first boot under a redaction-tagged log field, displayed in-band on `/setup/<token>`, mints first instance admin, forces passkey enrollment + backup codes. → §7 _Bootstrap_.
- **Break-glass CLI** (`cypra admin …`) for tenant-admin lockout, instance-admin recovery, key rotation, bootstrap reset. → §7 _CLI_.
- **App-level envelope encryption** (per-row DEK, KEK from env/file, never persisted). → §7 _Crypto_.
- **JWT access tokens (15 min) + opaque refresh tokens** with rotation and family-tree reuse-detection. → §7 _Sessions_.
- **GDPR-enabling features**: per-user data export, account deletion with PII scrub + audit-log redaction (not deletion), per-tenant deletion cascade with signing-key sunset. → §7 _GDPR services_.
- **Observability**: Prometheus `/metrics`, OpenTelemetry traces, structured JSON logs, `/healthz`, `/readyz`, reference `deploy/alerts.yml`. → §7 _Observability_.
- **Auto-migrations on boot** via `golang-migrate` (with `--skip-migrate` opt-out) and `cypra migrate` CLI. → §7 _CLI_, §15.
- **Backup tooling**: `cypra export` / `cypra import` with mandatory passphrase. → §7 _CLI_.
- **Audit log** with append-only semantics enforced via Postgres role split, redaction-on-DSR, NDJSON export, `state_before`/`state_after` JSONB snapshots for attack reconstruction. → §7 _Audit_.
- **Rate limiter** with per-IP / per-account / per-tenant counters; hard-coded sane defaults at v1 (real anti-stuffing defense is the bot-mitigation hook). → §7 _Rate limiter_.
- **Bot-mitigation hook** (interface only at v1; provider impls in v1.1). → §7 _Bot mitigation_.
- **Go SDK** (server-side OIDC + admin REST API). → §7 _SDKs_.
- **Distribution**: multi-arch Docker image, reference `docker-compose.yml` with `default` and `with-tls` (Caddy) profiles, Railway one-click template. → §16.

### Out of scope for MVP

- Embeddable widget (v1.1).
- TypeScript SDK (v1.1).
- Tenant-level CNAME domains (v1.1).
- SAML 2.0 (v1.2+).
- SMS / Twilio (v1.2+).
- Dashboard CSV import UI (post-v1).
- Built-in Turnstile / hCaptcha (v1.1).
- Additional upstream OAuth providers beyond Google (v1.1+).

---

## 7. System Architecture

All components live in **one Go binary** plus an embedded SPA (`dashboard/dist`). Component boundaries below are Go package boundaries, not separate services.

### Components

- **HTTP entrypoint** (`cmd/cypra serve`) — single `chi`-routed HTTP server. Subrouters: `/dashboard`, `/login`, `/oidc`, `/api/v1`, `/.well-known`, `/setup`, `/healthz`, `/readyz`, `/metrics`, `/storage` (HMAC-signed local-disk URLs). Trusts forwarded headers per `TRUSTED_PROXY_HEADERS` env.
- **Tenant resolver middleware** — resolves tenant from `Host` header. Match logic: strip the `<install-domain>` suffix; the remaining label is the tenant slug. Unknown hosts → 404. The install-level dashboard (`<install-domain>` with no subdomain) routes to instance-admin surfaces only. Source of truth for tenant scoping.
- **Tenancy core** — owns tenants, projects, memberships, instance admins. Enforces tenant slug rules: `^[a-z][a-z0-9-]{2,63}$`, denied list: `www, api, admin, dashboard, oidc, login, signup, setup, health, healthz, readyz, metrics, well-known, docs, assets, auth, id, me, root, public, static, _`.
- **Authorization** — role/permission resolver. Three intra-tenant roles (owner/admin/member); separate instance-admin role. Permissions include `user.password.reset`, `user.mfa.disable`, `user.factor.enroll`, `tenant.settings.write`, `audit.read`, `oidc.client.write`, `signing_key.rotate`, etc.
- **Auth methods** — pluggable verifier interface. v1 implementations: password (Argon2id), magic-link, passkey (WebAuthn, **per-tenant RP ID**), upstream Google OAuth (with mandatory `state` + `nonce`), TOTP, WebAuthn-as-second-factor.
- **Sessions** — JWT access tokens (15 min) + opaque refresh tokens (server-side row, 30-day default), refresh rotation, **family-tree reuse detection** (revoke entire family on detected reuse). Dashboard sessions live in `sessions`; instance-admin sessions live in `instance_admin_sessions` (separate table — cleaner under RLS).
- **OIDC provider** — **per-tenant issuer URL** `https://<tenant>.<install-domain>`. Per-tenant discovery (`/.well-known/openid-configuration`), JWKS (`/.well-known/jwks.json`), authorize (`/oidc/authorize`), token (`/oidc/token`), userinfo (`/oidc/userinfo`), revoke (`/oidc/revoke`), consent (`/oidc/consent`). Same handlers; tenant-scoped responses. `kid` is unique per tenant per key (`<tenant_id>:<kid_seq>`); cross-tenant verification is impossible by `kid` shape.
- **Signing keys** — per-tenant rotation. 90-day rotation, 30-day overlap; both keys in JWKS during overlap; only `active` mints; `overlap` verifies. JWKS responses carry `Cache-Control: public, max-age=300, must-revalidate`. Operators get a `cypra admin rotate-key --tenant=…` for force rotation. Documented operator playbook entry: downstream OIDC clients must refetch JWKS on `kid` miss.
- **Crypto** — Argon2id password hashing; AES-GCM envelope encryption (per-row DEK, KEK loaded from env/file at boot); WebAuthn via `go-webauthn/webauthn`; OIDC signing via `go-jose/go-jose/v3`.
- **Hosted login** — `html/template` + HTMX. Pages: `/login`, `/signup`, `/verify`, `/reset`, `/2fa`, `/consent`, `/error`. Themed per tenant (logo, accent color). No React on the security-critical surface.
- **Dashboard** — Vite + React + Tailwind v4 + ShadCN/Watermelon UI. SPA built into `dashboard/dist`, embedded via `embed.FS`. Asset paths versioned (`dashboard/dist/assets/index.<hash>.js`); SPA polls `GET /api/v1/version` and prompts reload on mismatch.
- **REST API** — versioned at `/api/v1`. JSON. Used by dashboard and Go SDK. Tenant-scoped via tenant resolver. Authenticated by dashboard session cookies or Personal Access Tokens (PATs).
- **Email abstraction + Email worker** — `Sender` interface with backends: `terminal`, `smtp`, `resend`. **Email worker** is an in-process goroutine pool that pulls from an `email_outbox` table; auth handlers `INSERT` rows and return; worker `SELECT FOR UPDATE SKIP LOCKED` and dispatches with exponential backoff. Mail send never blocks the auth path.
- **Storage abstraction** — `Storage` interface with `local-disk` and `s3-compatible` backends. **`Storage.SignedURL(key, ttl)`** returns:
  - `local-disk`: `https://<host>/storage/<HMAC-signed-token>` proxied by Cypra.
  - `s3-compatible`: native presigned URL.
  - Default TTL 5 minutes; max 1 hour; hard-coded.
- **Audit log** — append-only `audit_entries`. Schema includes `state_before`, `state_after` JSONB. Cypra runtime DB role has `INSERT, SELECT` and a narrow `UPDATE` (for redaction columns only). Redaction sets `redacted_at` and replaces PII fields with sentinels; never deletes rows.
- **Rate limiter** — token-bucket counters in `rate_limit_buckets` (`(scope, key, tenant_id NULL)`). Single-process v1; documented limit: under stuffing, hot bucket keys serialize on Postgres row locks — this is intentional (slow attacker = rate-limited attacker), but legitimate users behind shared NAT may see collateral 429s. **Real anti-stuffing defense is the bot-mitigation hook (v1.1 providers).**
- **Bot-mitigation hook** — `Verifier` interface. v1 ships a `noop` implementation; v1.1 adds Turnstile and hCaptcha.
- **Bootstrap** — first-boot detection (no `instance_admins` rows AND no un-consumed `bootstrap_tokens`). Atomic flow: `BEGIN; INSERT bootstrap_tokens (hash, expires); COMMIT; LOG token_plain (with redacted-on-export tag)`. If process crashes between commit and log, `cypra admin reset-bootstrap` re-issues. Token redeemed at `/setup/<token>`; setup page also displays the token in-band as confirmation. Auto-expires after 24 hours.
- **GDPR services** — `GET /api/v1/users/{id}/export`, `DELETE /api/v1/users/{id}`, `DELETE /api/v1/tenants/{id}`. Each writes audit entries; user-delete redacts user-PII fields on prior audit rows.
- **Observability** — Prometheus exporter at `/metrics`; OTLP-HTTP tracing when `OTEL_EXPORTER_OTLP_ENDPOINT` is set; `slog` JSON to stdout; `/healthz` (process up); `/readyz` (migrations applied + KEK loaded + DB reachable + storage backend reachable).
- **CLI** — `cypra serve`, `cypra migrate`, `cypra admin {promote, reset-passkey, reset-passwords, rotate-key, rotate-master-key, revoke-tenant-tokens, reset-bootstrap, list-tenants}`, `cypra export`, `cypra import`, `cypra version`.
- **SDKs** (`sdk/go/`) — Go module published separately from the server image. `oidc.Client` (consumes Cypra OIDC) + `admin.Client` (calls REST API with PAT auth). Released alongside server tags; semver-pinned. Lives in the same monorepo; consumed via `go get`. TS SDK is v1.1 work.

### Data flow

**Happy-path end-user sign-in via OIDC:**

```
                 ┌─────────────────────────────────────────────────────────────┐
                 │  HTTPS https://acme.auth.example.com  (TLS terminated by    │
                 │           Caddy reverse proxy, X-Forwarded-* set)           │
                 └────────────────────────────┬────────────────────────────────┘
                                              ▼
   ┌─────────┐  /oidc/authorize     ┌─────────────────────────────────────┐
   │ Browser │ ────────────────────►│ HTTP entrypoint (chi)               │
   │ (NextJS │                      │  └─ tenant_resolver_mw  → "acme"    │
   │  app    │                      │      └─ rls_setter_mw  (cypra.tid)  │
   │  client)│                      │          └─ /oidc/authorize handler │
   └─────────┘                      │              (uses TenantScopedDB)  │
        ▲                           └────────────────────┬────────────────┘
        │                                                ▼
        │                              ┌────────────────────────────────┐
        │  302 → /login?continue=…     │ OIDC: validates client_id,     │
        │                              │ scope, redirect_uri (exact),   │
        │                              │ PKCE challenge                 │
        │                              └────────────────┬───────────────┘
        │                                               ▼
        │                              ┌────────────────────────────────┐
        │  302 → /login                │ Hosted login (Go templates)    │
        │                              │ user picks "passkey" → JS calls│
        │                              │ navigator.credentials.get()    │
        │                              │ RP ID = acme.auth.example.com  │
        │                              └────────────────┬───────────────┘
        │                                               ▼
        │                              ┌────────────────────────────────┐
        │  POST /login/passkey/verify  │ Auth methods: passkey verifier │
        │                              │  – validates assertion         │
        │                              │  – sets session cookie scoped  │
        │                              │    to acme.auth.example.com    │
        │                              └────────────────┬───────────────┘
        │                                               ▼
        │  302 → /oidc/consent (if first-time)         consent recorded
        │  302 → /oidc/authorize/continue              ▼
        │                              ┌────────────────────────────────┐
        │  302 → app callback ?code=…  │ OIDC: mints authorization code │
        │                              │ signed with tenant's active key│
        └──────────────────────────────┴────────────────────────────────┘
                                              ▼
   ┌─────────┐  POST /oidc/token       ┌────────────────────────────────┐
   │ NextJS  │ ───────────────────────►│ OIDC token endpoint            │
   │ server  │  client_id+secret+code  │  – verifies PKCE               │
   └─────────┘                         │  – mints id_token (sub =       │
                                       │    "<tenant_id>:<user_id>")    │
                                       │  – mints access + refresh      │
                                       │    (with family_id)            │
                                       │  – writes refresh row + audit  │
                                       └────────────────────────────────┘
```

### Trust boundaries

- **Outside ↔ Cypra:** TLS terminates at the operator's reverse proxy. Cypra trusts forwarded headers only when `TRUSTED_PROXY_HEADERS` env is non-`none`. When `none`, the bare `Host` header drives the tenant resolver — operators without a proxy should bind Cypra to localhost and run the proxy on the same host.
- **Cypra ↔ Postgres:** two logical roles. `cypra_runtime` has DML on app tables; `INSERT, SELECT` and narrow `UPDATE` on `audit_entries`; never `DELETE` on `audit_entries`. `cypra_migrate` has DDL. Driven by separate env vars `DATABASE_URL` (runtime) and `MIGRATE_DATABASE_URL` (migrate; falls back to runtime URL if unset, with a warning).
- **Cypra ↔ master key:** loaded at boot from `MASTER_KEY` or `MASTER_KEY_FILE`; held in memory only; never persisted.
- **Cypra ↔ S3-compatible storage:** credentials envelope-encrypted in `email_provider_configs`-style row; HTTPS only; presigned URLs minted Cypra-side.
- **Cypra ↔ upstream OAuth (Google):** standard OAuth client; `state` and `nonce` mandatory and validated on callback; client secret envelope-encrypted; tokens minimally retained.
- **Tenant ↔ Tenant:** zero shared state above the tenant level except: instance admins, the tenancy core itself, and the audit log (rows tenant-scoped). Enforcement: tenant resolver + `TenantScopedDB` + Postgres RLS, three independent layers.

---

## 8. Data Model

Every tenant-scoped table has `tenant_id UUID NOT NULL` with `FK to tenants(id) ON DELETE CASCADE` (except where lifecycle requires sunset, see signing keys), and `(tenant_id, …)` indexes. The tenant-scoped query layer (ADR-0001) refuses queries lacking a `tenant_id` predicate.

### Entities

#### `tenants`

- **Purpose:** the customer organization unit; isolation, branding, OIDC issuer.
- **Fields:** `id UUID PK`, `slug TEXT`, `name TEXT`, `branding JSONB`, `settings JSONB`, `created_at`, `updated_at`, `deleted_at NULL`.
- **Indexes:** unique `(slug)` partial `WHERE deleted_at IS NULL`; CHECK constraint enforcing slug regex.

#### `projects`

- **Purpose:** an OIDC-client-bearing app under a tenant. **One project = one OIDC client at v1** (cardinality lifted in v1.1+).
- **Fields:** `id UUID PK`, `tenant_id UUID FK`, `slug TEXT`, `name TEXT`, `created_at`, `updated_at`, `deleted_at NULL`.
- **Indexes:** unique `(tenant_id, slug)` partial `WHERE deleted_at IS NULL`.

#### `instance_admins`

- **Purpose:** humans who run the install. Outside the tenancy model.
- **Fields:** `id UUID PK`, `email CITEXT UNIQUE`, `display_name TEXT`, `created_at`, `last_login_at`, `disabled_at NULL`.

#### `tenant_memberships`

- **Purpose:** a user's role within a tenant. Owner/admin/member.
- **Fields:** `id UUID PK`, `tenant_id UUID FK`, `user_id UUID FK`, `role ENUM('owner','admin','member')`, `created_at`.
- **Indexes:** unique `(tenant_id, user_id)`.

#### `users`

- **Purpose:** end-user accounts in a tenant. Tenant-scoped (same email in two tenants = two users).
- **Fields:** `id UUID PK`, `tenant_id UUID FK`, `email CITEXT`, `email_verified_at NULL`, `metadata JSONB`, `profile_picture_object_id UUID NULL`, `created_at`, `updated_at`, `deleted_at NULL`.
- **Indexes:** unique `(tenant_id, email)` partial `WHERE deleted_at IS NULL`.

#### `password_credentials`

- **Purpose:** Argon2id hashes for password-auth users.
- **Fields:** `id UUID PK`, `user_id UUID FK`, `argon2id_hash BYTEA`, `must_reset BOOL`, `created_at`, `updated_at`.
- **Indexes:** unique `(user_id)`.

#### `passkey_credentials`

- **Purpose:** WebAuthn credentials per user. **RP ID is per-tenant** (`<tenant>.<install-domain>`) — enforced at registration time.
- **Fields:** `id UUID PK`, `user_id UUID FK`, `credential_id BYTEA UNIQUE`, `public_key BYTEA`, `sign_count BIGINT`, `transports TEXT[]`, `aaguid UUID NULL`, `rp_id TEXT NOT NULL`, `nickname TEXT`, `created_at`, `last_used_at`.

#### `totp_credentials`

- **Purpose:** TOTP second-factor secrets, envelope-encrypted.
- **Fields:** `id UUID PK`, `user_id UUID FK`, `secret_encrypted BYTEA`, `algorithm TEXT`, `digits INT`, `period_seconds INT`, `confirmed_at NULL`.

#### `user_backup_codes` / `instance_admin_backup_codes`

- **Purpose:** one-time recovery codes (Argon2id-hashed). Split into two tables for FK integrity (no polymorphic `id`).
- **Fields (each):** `id UUID PK`, `<user|instance_admin>_id UUID FK`, `code_hash BYTEA`, `used_at NULL`, `created_at`.

#### `magic_link_tokens`, `password_reset_tokens`, `email_verification_tokens`, `admin_invites`

- **Purpose:** short-lived, single-use tokens (and admin invitations).
- **Fields:** `id UUID PK`, `user_id_or_email …`, `token_hash BYTEA UNIQUE`, `expires_at`, `consumed_at NULL`, `revoked_at NULL`. Reset tokens carry `revoked_at` and are revoked when a competing credential change succeeds (closes the email-vs-revoke TOCTOU).
- **Indexes:** `(user_id, expires_at)`.

#### `sessions`

- **Purpose:** dashboard / hosted-login sessions for **end-users and tenant admins**.
- **Fields:** `id UUID PK`, `subject_id UUID`, `subject_kind ENUM('user','tenant_admin')`, `tenant_id UUID NOT NULL`, `created_at`, `last_seen_at`, `expires_at`, `revoked_at NULL`, `ip INET`, `user_agent TEXT`.

#### `instance_admin_sessions`

- **Purpose:** dashboard sessions for instance admins. Separate table so RLS doesn't have to handle null `tenant_id`.
- **Fields:** `id UUID PK`, `instance_admin_id UUID FK`, `created_at`, `last_seen_at`, `expires_at`, `revoked_at NULL`, `ip INET`, `user_agent TEXT`.

#### `oidc_clients`

- **Purpose:** registered OIDC clients (one per project at v1).
- **Fields:** `id UUID PK`, `tenant_id UUID FK`, `project_id UUID FK`, `client_id TEXT UNIQUE` (globally unique; needed for token-endpoint lookup), `client_secret_encrypted BYTEA NULL` (null = public/PKCE-only), `redirect_uris TEXT[]`, `allowed_scopes TEXT[]`, `token_endpoint_auth_method ENUM('client_secret_basic','client_secret_post','none')`, `created_at`, `updated_at`, `deleted_at NULL`.
- **Notes:** internal references from `oidc_authorization_codes`, `oidc_refresh_tokens`, `oidc_consents` use `oidc_client_uuid UUID FK → oidc_clients(id)`, **not** the text `client_id`. External lookups (token endpoint) resolve `client_id TEXT → oidc_clients.id` once.

#### `oidc_authorization_codes`

- **Purpose:** short-lived authorization codes (60 s).
- **Fields:** `id UUID PK`, `code_hash BYTEA UNIQUE`, `tenant_id UUID FK`, `oidc_client_uuid UUID FK`, `user_id UUID FK`, `redirect_uri TEXT`, `scope TEXT[]`, `pkce_challenge TEXT`, `pkce_method ENUM('S256')`, `nonce TEXT NULL`, `expires_at`, `consumed_at NULL`.
- **Cleanup:** background job hard-deletes rows with `expires_at + 7d < now()`. Same job handles all `*_tokens` cleanup.

#### `oidc_refresh_tokens`

- **Purpose:** opaque refresh tokens with family-tree reuse-detection.
- **Fields:** `id UUID PK`, `family_id UUID NOT NULL`, `parent_id UUID NULL` (FK to same table), `token_hash BYTEA UNIQUE`, `tenant_id UUID FK`, `oidc_client_uuid UUID FK`, `user_id UUID FK`, `scope TEXT[]`, `expires_at`, `consumed_at NULL`, `revoked_at NULL`, `revoke_reason TEXT NULL`.
- **Indexes:** `(family_id)`, `(token_hash)`.

#### `oidc_consents`

- **Purpose:** per-user-per-client recorded consent.
- **Fields:** `id UUID PK`, `tenant_id UUID FK`, `user_id UUID FK`, `oidc_client_uuid UUID FK`, `scopes TEXT[]`, `granted_at`, `revoked_at NULL`.
- **Indexes:** unique `(user_id, oidc_client_uuid)`.

#### `oidc_signing_keys`

- **Purpose:** per-tenant OIDC signing keypairs with rotation states.
- **Fields:** `id UUID PK`, `tenant_id UUID FK`, `kid TEXT UNIQUE` (shape: `<tenant_id>:<seq>` to make cross-tenant verification structurally impossible), `algorithm TEXT` (`RS256` default; `ES256` selectable), `public_key_jwk JSONB`, `private_key_encrypted BYTEA`, `state ENUM('active','overlap','retired','sunsetting')`, `activated_at`, `retires_at`, `sunset_until NULL`.
- **Notes:** **does NOT use `ON DELETE CASCADE`**. Tenant deletion sets all keys to `state='sunsetting'`, `sunset_until = now() + 30 days`; rows are hard-deleted after sunset by a job. JWKS continues serving sunsetting keys for the window.

#### `upstream_providers`

- **Purpose:** per-tenant OAuth upstream IdP config (Google v1).
- **Fields:** `id UUID PK`, `tenant_id UUID FK`, `kind ENUM('google')`, `client_id_encrypted BYTEA`, `client_secret_encrypted BYTEA`, `enabled BOOL`, `created_at`.

#### `email_provider_configs`

- **Purpose:** per-tenant email-provider config. **Mandatory**: tenants without a configured provider cannot issue magic-links / verifications / password-resets; the dashboard surfaces a hard block until configured.
- **Fields:** `id UUID PK`, `tenant_id UUID FK`, `kind ENUM('terminal','smtp','resend')`, `config_encrypted BYTEA`, `from_address TEXT`, `from_name TEXT`.

#### `email_outbox`

- **Purpose:** queued outbound emails for the email worker. Decouples mail send from auth handlers.
- **Fields:** `id UUID PK`, `tenant_id UUID NULL` (null for instance-level mail), `to_address TEXT`, `template TEXT`, `payload JSONB`, `attempts INT`, `next_attempt_at`, `sent_at NULL`, `failed_at NULL`, `last_error TEXT NULL`.
- **Indexes:** `(next_attempt_at) WHERE sent_at IS NULL AND failed_at IS NULL`.

#### `storage_objects`

- **Purpose:** metadata for binary blobs (profile pictures at v1).
- **Fields:** `id UUID PK`, `tenant_id UUID FK`, `backend ENUM('local-disk','s3-compatible')`, `key TEXT`, `content_type TEXT`, `byte_size BIGINT`, `created_at`, `deleted_at NULL`.

#### `audit_entries`

- **Purpose:** append-only log of admin and security-relevant actions.
- **Fields:** `id UUID PK`, `occurred_at TIMESTAMPTZ`, `tenant_id UUID NULL`, `actor_kind ENUM('user','tenant_admin','instance_admin','system')`, `actor_id UUID NULL`, `action TEXT`, `resource_kind TEXT`, `resource_id UUID NULL`, `state_before JSONB NULL`, `state_after JSONB NULL`, `ip INET NULL`, `user_agent TEXT NULL`, `metadata JSONB`, `redacted_at NULL`.
- **Indexes:** `(tenant_id, occurred_at DESC)`, `(action, occurred_at DESC)`.
- **Postgres role enforcement:** `cypra_runtime` has `INSERT, SELECT`, and `UPDATE (redacted_at, state_before, state_after, metadata, ip, user_agent)` only — enforced via column-level `GRANT`. No `DELETE`.

#### `bootstrap_tokens`

- **Purpose:** the first-boot setup token (≤ 1 un-consumed row at a time).
- **Fields:** `id UUID PK`, `token_hash BYTEA UNIQUE`, `created_at`, `expires_at`, `consumed_at NULL`, `revoked_at NULL`.
- **Constraint:** partial unique index `(true) WHERE consumed_at IS NULL AND revoked_at IS NULL` ensures at most one live token.

#### `master_key_rotations`

- **Purpose:** tracks in-progress master-key rotation for resumability.
- **Fields:** `id UUID PK`, `started_at`, `completed_at NULL`, `phase ENUM('rewrap','cutover','done')`, `rows_total BIGINT`, `rows_done BIGINT`, `error TEXT NULL`.

#### `rate_limit_buckets`

- **Purpose:** token-bucket counters keyed by `(scope, key)`.
- **Fields:** `scope TEXT`, `key TEXT`, `tenant_id UUID NULL` (NULL for IP-only buckets), `tokens DOUBLE`, `last_refill_at TIMESTAMPTZ`. PK: `(scope, key, tenant_id NULLS NOT DISTINCT)`.
- **Notes:** RLS policy permits NULL `tenant_id` rows (IP-scoped) for any tenant context.

### Invariants

1. **Tenant isolation (no carve-out).** Every tenant-scoped table has `tenant_id NOT NULL`. Every query — including raw SQL — passes through `TenantScopedDB`. Postgres RLS is the second belt with a connection-checkout reset hook (`RESET cypra.tenant_id` on every checkout from the pool). Test plan: a fuzzer that mutates `tenant_id` on every handler request context and asserts cross-tenant reads/writes return zero rows / fail authz.
2. **Per-tenant email uniqueness:** `(tenant_id, email)` unique among non-deleted users.
3. **Globally unique OIDC `client_id`** (text); internal references use `oidc_client_uuid`.
4. **Refresh-token family integrity.** Family is bounded by: created at first token-endpoint mint for a given `(client, user, scope)`; continues across rotation; ends at logout, full-family revocation, or reuse detection. Reuse cascade-revokes every row with the same `family_id`.
5. **OIDC signing-key rotation states:** at most one `active` per tenant; at most one `overlap` per tenant; tokens minted only by `active`; verifiers accept `active ∪ overlap ∪ sunsetting`. `kid` is `<tenant_id>:<seq>` so cross-tenant key confusion is impossible at lookup.
6. **Per-tenant WebAuthn RP ID.** A passkey registered under tenant A's RP ID cannot be presented to tenant B; enforced by browser per WebAuthn spec.
7. **Audit log append-only.** Enforced via column-level `GRANT` on Cypra runtime role; CHECK constraint on redaction-shape (`redacted_at NULL XOR PII fields are sentinels`).
8. **Master-key dependency.** If KEK fails to load, `/readyz` returns 503; process exits non-zero.
9. **Migrations dependency.** Pending migrations + no `--skip-migrate` → exit at boot with clear message.
10. **Single-use tokens.** All `*_tokens` rows are consumed atomically with `UPDATE … SET consumed_at = NOW() WHERE consumed_at IS NULL RETURNING …`; double-consume returns zero rows and fails closed.
11. **Bootstrap exclusivity.** Partial unique index on `bootstrap_tokens` ensures at most one live token; first-boot detector refuses if any `instance_admins` row exists.
12. **Issuer URL is per-tenant** and equals `https://<tenant>.<install-domain>`. The `iss` claim and discovery doc carry that string verbatim.
13. **Email provider mandatory.** Tenant cannot issue magic-links, password-resets, or verifications until `email_provider_configs` has an enabled row.
14. **DSR scrub completeness.** User delete updates `users` row, redacts user-PII fields on related `audit_entries` (in-place column update), purges profile picture from storage. Idempotent.

### Lifecycle and retention

- **Soft-delete** for `tenants`, `projects`, `users`, `oidc_clients`. Soft-deleted rows invisible to read paths; hard-deleted by a periodic job after a 30-day grace.
- **Hard-delete** for short-lived rows: `*_tokens` after `consumed_at` or `expires_at + 7d`; rate-limit buckets after 24h inactivity; expired `sessions` after 30d; expired `email_outbox` after `sent_at + 30d`.
- **Audit log retention:** default 90 days, tenant-overridable up to 730 days. Redacted rows count toward retention.
- **GDPR DSR — user delete:** see Invariant 14.
- **GDPR DSR — tenant delete:** revokes all sessions + refresh tokens; soft-deletes all OIDC clients; **signing keys → `state='sunsetting'`, `sunset_until = now() + 30d`** (resolves the cascade-vs-sunset contradiction); all tenant-scoped rows soft-deleted then hard-deleted after 30 days; profile pictures purged from storage backend.
- **`cypra import` after DSR delete:** importer checks for `gdpr_deletions` ledger entries (a small companion table tracking per-id deletions); if the bundle would resurrect a deleted user, import refuses unless `--allow-resurrect` is explicitly passed and a fresh audit entry is written. Documented operator obligation.

---

## 9. Interface Contracts

### REST API — `/api/v1/*`

- **Shape:** JSON over HTTPS. Resource-oriented routes (`/api/v1/tenants/{tenant_id}/users`).
- **Inputs:** `application/json`; `Authorization: Bearer <session_token | pat>`. `Idempotency-Key` optional on unsafe methods, required on `POST /imports`.
- **Outputs:** JSON `{data, meta}` on 2xx; JSON `{error: {code, message, details}}` on 4xx/5xx.
- **Error model:** structured codes — `tenant.not_found`, `tenant.slug_reserved`, `tenant.slug_invalid`, `tenant.email_provider_required`, `auth.invalid_credentials`, `auth.rate_limited`, `auth.mfa_required`, `auth.bot_mitigation_failed`, `auth.upstream_unavailable`, `auth.upstream_state_mismatch`, `oidc.invalid_client`, `oidc.invalid_grant`, `oidc.invalid_redirect_uri`, `oidc.kid_mismatch`, `validation.failed`, `gdpr.user_deletion_in_progress`, `gdpr.import_resurrect_blocked`, `internal.server_error`. Every code is reachable in §14.
- **Versioning:** `v1` is forever stable. Breaking changes go to `v2`; both run for one major release.

### OIDC surface — per-tenant `/.well-known/*` and `/oidc/*`

- **Issuer URL:** `https://<tenant>.<install-domain>` exactly. `iss` claim and discovery `issuer` field both equal this string.
- **Endpoints:**
  - `GET /.well-known/openid-configuration` — `Cache-Control: public, max-age=600, must-revalidate`.
  - `GET /.well-known/jwks.json` — `Cache-Control: public, max-age=300, must-revalidate`. Returns `active ∪ overlap ∪ sunsetting` keys with `kid = <tenant_id>:<seq>`.
  - `GET /oidc/authorize` — `code_challenge_method=S256` mandatory; `state` recommended (warning logged when absent).
  - `POST /oidc/token` — supports `authorization_code` and `refresh_token` grants only.
  - `GET /oidc/userinfo` — `Cache-Control: no-store`. Claims: `sub` (= `<tenant_id>:<user_id>`), `email`, `email_verified` (from `users.email_verified_at`), `name`, `picture` (signed URL, 5 min TTL), `updated_at`.
  - `POST /oidc/revoke` — RFC 7009. **Refresh tokens only at v1**; access tokens accepted but no-op (200 per spec). Revoking a refresh token revokes its entire family. `unsupported_token_type` returned for unknown token shapes.
  - `GET /oidc/consent` — HTML.
- **redirect_uri matching:** **exact-match per OIDC Core / RFC 6749 §3.1.2.** Host comparison is case-insensitive (RFC 3986 §3.2.2); path / query / port comparison is exact. Trailing slash is significant. Fragments are forbidden in registered URIs and rejected at registration.
- **OIDC error codes** (token/authorize): `invalid_request`, `invalid_client`, `invalid_grant`, `unauthorized_client`, `unsupported_grant_type`, `invalid_scope`, `access_denied`, `interaction_required`, `login_required`, `consent_required`, `account_selection_required`, `server_error`, `temporarily_unavailable`. Revoke endpoint additionally: `unsupported_token_type`.
- **Clock skew:** ID-token `iat`/`exp`/`nbf` validated with **±60 s** leeway. TOTP windows: ±1 step (~±30 s). Hard-coded at v1.
- **Versioning:** OIDC is a stable external contract; spec-additive changes only. Algorithm removal requires a one-minor-release deprecation window + audit-log entry.

### Hosted login — `/login`, `/signup`, `/verify`, `/reset`, `/2fa`, `/setup/{token}`

- **Shape:** server-rendered HTML; HTMX-enhanced. POST endpoints accept `application/x-www-form-urlencoded`. CSRF-tokened.
- **Error model:** uniform "incorrect email or password" on auth failures (no enumeration). Internal errors → generic error page with `request_id`.
- **Versioning:** internal surface; no compatibility guarantees beyond "the dashboard and SDK still work."

### CLI — `cypra <subcommand>`

- **Subcommands:**
  - `serve [--skip-migrate]`
  - `migrate [--to <version>]`
  - `admin promote --tenant=<slug-or-id> --email=<email> [--role=<owner|admin|member>]` (omitting `--tenant` is rejected)
  - `admin reset-passkey --tenant=… --email=…`
  - `admin reset-passwords --tenant=…` (sets `must_reset=true` on all password credentials)
  - `admin rotate-key --tenant=…`
  - `admin rotate-master-key --old-from-env --new-from-env [--resume]`
  - `admin revoke-tenant-tokens --tenant=…`
  - `admin reset-bootstrap` (revokes any live bootstrap token; mints a fresh one only if no `instance_admins` exist)
  - `admin list-tenants [--json]`
  - `export --out=<path> --passphrase-from-stdin | --passphrase-file=<path>`
  - `import <path> --passphrase-from-stdin | --passphrase-file=<path> [--allow-resurrect]`
  - `version`
- **Inputs:** flags + positional args; `DATABASE_URL` and `MASTER_KEY` env required.
- **Error model:** stderr `error: <code>: <message>`; exit codes: `0` ok, `1` generic, `2` flag misuse, `3` missing env, `4` migration pending, `5` integrity check failed, `6` resource not found, `7` passphrase missing in non-interactive context.
- **Versioning:** flag removals require a one-minor-release deprecation window.

### Go SDK — `github.com/watzon/cypra/sdk/go`

- **Two clients:** `oidc.Client` (consumes Cypra-as-OIDC) and `admin.Client` (admin REST API, PAT-authenticated).
- **Inputs:** `oidc.Client`: per-tenant issuer URL, client ID, client secret, redirect URI. `admin.Client`: base URL, PAT.
- **Outputs:** typed structs mirroring REST API; OIDC types via `coreos/go-oidc`.
- **Error model:** typed — `cypra.ErrNotFound`, `cypra.ErrUnauthorized`, `cypra.ErrConflict`, `cypra.ErrRateLimited`, `cypra.ErrValidation`. `errors.Is`/`errors.As` compatible.
- **Versioning:** semver. Tagged `sdk/go/vX.Y.Z` independently from server.

### Storage backend interface (internal)

- **Shape:** `Storage interface { Put(ctx, key, content_type, r) error; Get(ctx, key) (io.ReadCloser, http.Header, error); Delete(ctx, key) error; SignedURL(ctx, key, ttl time.Duration) (string, error) }`.
- **TTL bounds:** default 5 min, max 1 hour, hard-coded.
- **Local-disk SignedURL implementation:** `https://<host>/storage/<base64url(payload)>.<HMAC-SHA256(payload, key=KEK-derived-subkey)>` where payload encodes `{key, exp}`. Verified at `/storage/*` handler.
- **Error model:** `storage.ErrNotFound`, `storage.ErrConflict`, `storage.ErrUnavailable`. `ErrUnavailable` flips `/readyz` to 503.
- **Versioning:** internal Go interface at v1; public extension point in v1.1+.

### Bot-mitigation hook interface (internal)

- **Shape:** `Verifier interface { Verify(ctx, evidence map[string]string) (Result, error) }`.
- **Versioning:** v1 ships interface + `noop` impl; v1.1 adds Turnstile/hCaptcha. Public stability deferred until v1.1.

---

## 10. Technology Choices

| Area                      | Choice                                                                                               | Rationale                                                                                                                                                                               |
| ------------------------- | ---------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Language / runtime        | **Go 1.23+**                                                                                         | Operator preference; single static binary matches distribution model; mature crypto stdlib; ~30 MB image.                                                                               |
| HTTP router               | **`go-chi/chi` v5**                                                                                  | Idiomatic `net/http`-compatible middleware; per-surface subrouters; zero magic.                                                                                                         |
| Query layer               | **GORM v2 (queries only) + `golang-migrate/migrate` (DDL only)**                                     | GORM for application reads/writes via `TenantScopedDB`; migrations are explicit SQL files, not GORM auto-migrate. Resolves the GORM-vs-explicit-migrations contradiction at the source. |
| Database                  | **Postgres 16+**                                                                                     | Operator preference; CITEXT, JSONB, partial indexes, RLS, CHECK constraints, advisory locks. Single-instance is fine for v1 scale targets.                                              |
| Migration tool            | **`golang-migrate/migrate` (SQL files in `db/migrations/`)**                                         | Explicit DDL is debuggable; auth-product schema correctness is security-critical; up/down required for every change.                                                                    |
| Crypto                    | **stdlib `crypto/*` + `go-webauthn/webauthn` + `golang.org/x/crypto/argon2` + `go-jose/go-jose/v3`** | Stdlib first; vetted libraries for the hard pieces; no CGO.                                                                                                                             |
| OIDC server               | **Hand-rolled on `go-jose/go-jose/v3`**                                                              | Existing libs are heavy (Hydra) or stale; lean v1 surface fits in ~2 kLOC. Trade-off: ongoing maintenance — accepted as a core competency.                                              |
| Web frontend              | **Vite + React 19 + Tailwind v4 + ShadCN/Watermelon UI**                                             | Operator preference; Watermelon UI extends ShadCN for admin surfaces.                                                                                                                   |
| Frontend bundling         | **Vite (build) + `embed.FS` (serve)**                                                                | Single binary; production = static assets compiled into the binary.                                                                                                                     |
| Frontend reproducibility  | **Pinned Bun version + committed `bun.lockb` + frozen-lockfile CI install**                          | Single-binary distribution implies reproducible SPA build; explicit pin closes the gap.                                                                                                 |
| Hosted-login UI           | **`html/template` + HTMX**                                                                           | Login pages must work without JS; HTMX adds passkey UX without a SPA on a security-critical surface.                                                                                    |
| Local dev                 | **`portless` (HTTPS + `*.cypra.localhost` wildcard)**                                                | Required for WebAuthn (no plain HTTP) and tenant-subdomain routing in dev.                                                                                                              |
| Reverse proxy (reference) | **Caddy** in `docker-compose.yml` profile `with-tls`                                                 | Auto-issues Let's Encrypt certs; v1.1 handles per-tenant CNAMEs via `on_demand_tls`. Cypra is unaware of TLS.                                                                           |
| Frontend pkg manager      | **Bun workspaces** (fallback: pnpm — ADR-0010)                                                       | Operator preference; SDK consumers receive published npm packages so internal Bun usage doesn't leak.                                                                                   |
| Build orchestration       | **`make` + `Taskfile.yml`**                                                                          | Cross-cuts Go + Bun without adding a JS-only build tool to a Go-primary repo.                                                                                                           |
| Object storage SDK        | **`aws-sdk-go-v2`**                                                                                  | S3/R2/B2/MinIO compatible; presigned URLs out of the box.                                                                                                                               |
| Email — Resend            | **Resend Go SDK**                                                                                    | Operator preference; no translation layer.                                                                                                                                              |
| Email — SMTP              | **`net/smtp` (stdlib)**                                                                              | Sufficient for transactional volumes.                                                                                                                                                   |
| Logging                   | **`log/slog` (stdlib)** in JSON mode                                                                 | Stdlib structured logging since 1.21. Custom handler wrapper enforces PII redaction.                                                                                                    |
| Metrics                   | **`prometheus/client_golang`**                                                                       | Standard.                                                                                                                                                                               |
| Tracing                   | **OpenTelemetry Go + OTLP-HTTP exporter**                                                            | Activated only when `OTEL_EXPORTER_OTLP_ENDPOINT` is set.                                                                                                                               |
| Test runner               | **Go `testing` + `testify/require` + `testcontainers-go` (Postgres)**                                | Tests run against real Postgres, not SQLite-shaped lies.                                                                                                                                |
| Lint                      | **`golangci-lint` (strict) + ESLint + Prettier**                                                     | Standard for both stacks.                                                                                                                                                               |
| CI                        | **GitHub Actions**                                                                                   | Default; reusable workflows for build/test/docker/release.                                                                                                                              |
| Container distribution    | **`ghcr.io/watzon/cypra`**                                                                           | Free, integrated with the repo.                                                                                                                                                         |
| Deployment targets (v1)   | **Bare Docker / Docker Compose / Railway template**                                                  | Matches §16.                                                                                                                                                                            |
| License                   | **MIT**                                                                                              | Per Principle 7.                                                                                                                                                                        |

---

## 11. Security and Privacy

### Auth model

- **Instance admins:** WebAuthn passkey-first against the install dashboard (RP ID = `<install-domain>`). Backup codes mandatory. Magic-link recovery via `cypra admin promote` + email.
- **Tenant admins/members:** WebAuthn passkey-first against their tenant subdomain (RP ID = `<tenant>.<install-domain>`). Backup codes mandatory. Magic-link recovery requires another tenant admin or instance-admin break-glass.
- **End-users:** password (Argon2id), magic-link, passkey (per-tenant RP ID), Google upstream. TOTP and WebAuthn second-factor available; not mandated by Cypra (per-tenant policy decision post-v1).
- **OIDC consumers (downstream apps):** authorization code + PKCE only. Token endpoint accepts `client_secret_basic`, `client_secret_post`, `none` (PKCE-public).
- **Upstream OAuth (Google):** mandatory `state` (random, signed) and `nonce` (echoed in `id_token`). Both validated on callback; mismatch → `auth.upstream_state_mismatch`.
- **SDK / admin API:** Personal Access Tokens (PATs) issued from the dashboard, scoped to the tenant of the issuing admin and to a permission set. PATs revoked when the issuing admin's role is downgraded or membership revoked. ADR-0011 covers token format, expiry, audit-log integration.

### Tenant-isolation invariant (mechanism)

- **`TenantScopedDB`** wraps `*gorm.DB`. Public methods: `Find`, `Create`, `Update`, `Delete`, `Raw(stmt string, tenantID uuid.UUID, args ...any)`. **Raw SQL must take `tenantID` as a typed argument**; the wrapper rejects calls with a zero `tenantID`. Handlers receive `*TenantScopedDB`, never `*gorm.DB` directly.
- **Postgres RLS:** every tenant-scoped table has policy `USING (tenant_id = current_setting('cypra.tenant_id')::uuid)`. The connection-checkout hook (`gorm` callback `CheckedOut`) sets `cypra.tenant_id` from request context; the hook also sets a hard `RESET cypra.tenant_id` on connection return (`Returned`). Without a tenant context, RLS rejects all reads.
- **Defense in depth:** an attacker who finds a code-path bypass through `TenantScopedDB` must also have broken RLS, and vice versa.
- **Test plan:** integration-test fuzzer mutates `tenant_id` on every handler request context and asserts cross-tenant reads return zero rows. Runs on every CI build.

### Secret handling

- **Master KEK** loaded from `MASTER_KEY` or `MASTER_KEY_FILE`; held in memory only; never persisted; never logged.
- **Per-row DEKs** generated freshly per encrypted row, encrypted under KEK with AES-GCM, stored alongside ciphertext.
- **Master-key rotation (`cypra admin rotate-master-key`):** writes a `master_key_rotations` row, walks every encrypted column in batches, re-wraps each row's DEK under the new KEK in a single transaction, updates `rows_done`. **Resumable** via `--resume`; safe under crash. New writes during rotation use the new KEK; verifier code accepts both KEKs while `phase != 'done'`. After all rows re-wrapped, `phase='cutover'`; operator confirms via CLI (`rotate-master-key --confirm-cutover`) which removes the old KEK from memory and sets `phase='done'`.
- **OIDC signing-key rotation:** automatic, 90-day, 30-day overlap. Operator-triggered force-rotate via `cypra admin rotate-key --tenant=…`.
- **Loss of master key:** documented as irrecoverable. `cypra export` includes a passphrase-wrapped secrets bundle so a backup tarball is restorable on a fresh install.
- **Bootstrap-only env vars (irreducible set):**
  - `DATABASE_URL` (runtime role) — required.
  - `MIGRATE_DATABASE_URL` (migrate role) — falls back to `DATABASE_URL` with a warning at boot.
  - `MASTER_KEY` _or_ `MASTER_KEY_FILE` — required.
  - `LISTEN_ADDR` — default `:8080`.
  - `PUBLIC_BASE_URL` — required; shapes per-tenant issuer URLs and emailed links.
  - `TRUSTED_PROXY_HEADERS` — `none` | `x-forwarded` | `forwarded`. Default `none`.
  - `STORAGE_BACKEND` + `STORAGE_S3_*` (when applicable) — required.
  - `OTEL_EXPORTER_OTLP_ENDPOINT` (optional).
  - `LOG_LEVEL` (optional, default `info`).
- All other config is dashboard-managed.

### Threat model

**Who:** opportunistic credential-stuffers; targeted attackers going after specific tenant admins or end-users; **malicious tenant** on a multi-tenant install attempting to escalate or extract another tenant's data; malicious downstream client; future hostile insider on a multi-admin instance.
**What they want:** end-user accounts (ATO); stored secrets (signing keys → forged tokens, OAuth client secrets → impersonation, SMTP creds → phishing pivot); cross-tenant administrative access; subdomain shadowing.
**Defenses:** the tenant-isolation invariant (three layers: resolver, `TenantScopedDB`, RLS); per-tenant WebAuthn RP IDs (closes the cross-tenant passkey assertion); per-tenant signing keys with namespaced `kid`; encrypted secrets at rest with KEK off-DB; rate-limiter at IP/account/tenant; uniform error messages; exact redirect-URI matching; refresh-token rotation with reuse detection; passkey-first admin auth; setup-token bootstrap; column-level GRANTs on audit log; reserved-slug denylist; `state`+`nonce` on upstream OAuth; mandatory PKCE.
**Out of scope:** sustained attacker with `MASTER_KEY` in hand; operator-misconfigured Postgres (open binding, default password, unencrypted backups); operator-installed third-party plugins.

### PII handling

- **Identifiers:** email (per-tenant, CITEXT), display-name, profile picture, JSON metadata (operator-controlled).
- **Auth artifacts:** Argon2id password hashes, WebAuthn public keys, TOTP secrets (encrypted), Argon2id-hashed backup codes.
- **Network identifiers:** IP, user-agent on sessions and audit entries.
- **Treatment:** emails never logged at info level (handler-layer redaction); profile pictures purged on user-delete; metadata is operator-controlled (documented warning); audit-log PII redacted (not deleted) on DSR.

### Compliance constraints

- **GDPR-enabling features** (per Principle 5): DSR access (`GET /api/v1/users/{id}/export`), DSR delete (`DELETE /api/v1/users/{id}`), tenant-deletion cascade with documented sunset, append-only audit log, operator playbook (data inventory, sub-processor list template, DSR runbook, breach-notification runbook).
- **Cypra-the-project is NOT a processor.** Self-hosted means the operator is the controller.
- **Out of scope:** SOC 2, HIPAA, FedRAMP, ISO 27001, PCI.

---

## 12. Observability

### Logging

- **`log/slog`** in JSON mode to stdout. Default level `info`.
- **Required fields on every record:** `time`, `level`, `msg`, `request_id` (UUID v7), `tenant_id`, `actor_id`.
- **PII redaction enforced at handler-layer wrapper:** no email bodies, no token bodies, no plaintext passwords ever in logs.
- **Bootstrap token logging:** the `setup_token` field is emitted with a `redacted-on-export: true` log attribute so downstream log shippers can strip it before forwarding. Documented in operator playbook.

### Metrics

- **Prometheus** at `/metrics`.
- **Critical metrics at v1:**
  - `cypra_http_request_duration_seconds{route, method, status}` — histogram.
  - `cypra_auth_attempts_total{kind, outcome}` (kinds: `password,passkey,magic_link,google,totp,webauthn2fa`; outcomes: `success,fail,rate_limited,bot_blocked`).
  - `cypra_oidc_token_issued_total{grant}` and `cypra_oidc_refresh_reuse_detected_total`.
  - `cypra_signing_key_age_seconds{tenant, state}`.
  - `cypra_storage_operation_duration_seconds{backend, op}`.
  - `cypra_email_outbox_pending` (gauge), `cypra_email_send_total{outcome}`.
  - `cypra_db_pool_*`.
  - `cypra_rate_limit_exceeded_total{scope}`.
  - `cypra_migrations_pending` — 0 or 1.
  - `cypra_master_key_rotation_phase` — gauge encoding `none|rewrap|cutover|done`.
- **Reference alerts** at `deploy/alerts.yml`: high error rate, high auth-failure rate, signing-key near-expiry, refresh-token reuse-detected, master-key not set, migrations pending, storage write failures, db conn saturation, email outbox stuck (high pending + low send rate).

### Tracing

- **OpenTelemetry Go** + OTLP-HTTP exporter, activated when `OTEL_EXPORTER_OTLP_ENDPOINT` is set.
- One span per HTTP request; child spans for DB queries (GORM hook), upstream OAuth, email sends, storage ops.
- `traceparent` propagated.

### Error tracking

- **No bundled error-tracker dependency.** Errors at `error` level with `stack_trace`. Operators wire Sentry/Datadog via stdout shipping.

### Audit events

- **Subjects:** every action that changes auth state, tenant config, OIDC client config, signing keys, admin membership, DSR processing, master-key rotation phase changes, bootstrap-token consumption.
- **Visibility:** tenant admins see entries scoped to their tenant; instance admins see all entries; tenant _members_ see none; end-users see none.
- **Tamper-resistance:** column-level `GRANT` on `audit_entries`; `cypra_runtime` cannot `DELETE`; CHECK constraint enforces redaction shape.
- **Reconstruction:** `state_before`/`state_after` JSONB capture resource state across mutations.
- **Export:** NDJSON via `GET /api/v1/audit/export?since=…&until=…` (tenant-scoped).

---

## 13. Performance and Scaling

### Targets

- **p99 OIDC `/oidc/token`** ≤ 200 ms on a 2-vCPU / 4 GiB VPS, 10k users / tenant, < 50 RPS sustained.
- **p99 hosted-login `POST /login/passkey/verify`** ≤ 250 ms.
- **p99 dashboard `GET /api/v1/users` (paged 50)** ≤ 100 ms.
- **Cold start** ≤ 3 s from `docker run` to `/readyz=200` (assumes migrations applied).
- **Image size** ≤ 80 MB compressed.

### Expected v1 load

- 1–10 tenants, ≤ 100k users total, ≤ 50 RPS sustained, ≤ 500 RPS peak. Single Cypra + single Postgres on a $20/mo VPS.

### Rate limiter under attack — known limit

The Postgres rate limiter uses `INSERT … ON CONFLICT (scope, key, tenant_id) DO UPDATE` and serializes on row locks under hot-key contention. Under a credential-stuffing attack, this is **deliberate**: serialization delivers the rate limit as a side effect. Legitimate users behind the same NAT may experience 429s during an attack; this is documented as a known operational limit. **The real anti-stuffing defense is the bot-mitigation hook** (v1.1 Turnstile/hCaptcha providers); v1 ships the hook interface and the rate limiter only.

### First scaling lever

Move `rate_limit_buckets`, `oidc_refresh_tokens` reuse-detection state, and email-outbox dispatch coordination to **Redis** (post-v1), enabling multi-instance Cypra behind a load balancer. Until then, "scale up the VPS" is the documented answer.

---

## 14. Failure Modes

Every error code in §9 is reachable below.

- **OIDC signing-key compromise.** Detection: out-of-band only at v1. Recovery: `cypra admin rotate-key --tenant=…` immediately; `cypra admin revoke-tenant-tokens --tenant=…` invalidates every refresh. User-visible: end-users with active sessions get logged out.
- **DB exfiltration.** Detection: operator's IDS (out of scope). Recovery: rotate KEK (re-wrap), rotate every signing key, rotate every OIDC client secret, `cypra admin reset-passwords --tenant=…` flips `must_reset=true`. User-visible: "password changed, please reset" on next login.
- **Mass account-takeover via stuffing.** Detection: `cypra_rate_limit_exceeded_total` + `cypra_auth_attempts_total{outcome="fail"}` spikes. Recovery: rate limits engage automatically (with collateral damage); enabling bot-mitigation provider (v1.1) is the real remedy; operator can hard-block IPs at the reverse proxy. User-visible: legitimate users in affected ranges see "too many attempts."
- **Email provider down.** Detection: `cypra_email_send_total{outcome="fail"}` spike; outbox pending climbs. Recovery: worker retries with exponential backoff; operator switches provider in dashboard. User-visible: recipients see "email is taking longer than usual."
- **Upstream OAuth provider down (Google).** Detection: handler returns `auth.upstream_unavailable`. User-visible: "Sign in with Google" inline error; user falls back to other methods.
- **Upstream OAuth `state`/`nonce` mismatch.** Detection: handler returns `auth.upstream_state_mismatch`. Recovery: user retries; cause logged with `request_id`. User-visible: "sign-in failed, please try again."
- **Postgres down.** Detection: `/readyz` 503. Recovery: operator's responsibility (HA out of scope). User-visible: 503; dashboard shows maintenance banner.
- **Storage backend down.** Detection: `/readyz` 503; profile-picture fetches 502. Recovery: operator switches backend in dashboard; `storage_objects` rows pointing at the old backend remain readable until the operator runs `cypra admin migrate-storage --from=… --to=…` (post-v1; v1 documented as "switching backends without migration leaves old objects unreadable"). User-visible: profile pictures show placeholder; uploads fail.
- **Master key fails to load.** Detection: `/readyz` 503; process exits non-zero. Recovery: fix env/file; restart.
- **Migration pending without `--skip-migrate`.** Detection: process exits at boot with `error: migration_pending`. Recovery: `cypra migrate`.
- **Refresh-token reuse detected.** Detection: token presented with `consumed_at IS NOT NULL`. Recovery: cascade revoke entire `family_id`; emit metric + audit. User-visible: end-user signed out across all devices in that family.
- **Refresh-token race (legit + attacker present same parent).** Detection: both calls succeed (single atomic consume each); on the _next_ call, second presenter triggers reuse-detection. Recovery: family revoked. User-visible: whichever party loses the race is signed out.
- **JWKS-cache stale at downstream.** Detection: not measurable from Cypra side. Mitigation: 5-min `Cache-Control` on JWKS + documented "refetch on `kid` miss" guidance in operator playbook + 30-day overlap window.
- **Setup-token leaked via stdout shipping.** Detection: `bootstrap_tokens.consumed_at` set when no operator action was taken. Recovery: token is single-use; if leaked pre-consume, operator runs `cypra admin reset-bootstrap`. Mitigation: `redacted-on-export` log attribute documented for log-shipper config. User-visible: legitimate operator sees "token already consumed" → `cypra admin reset-bootstrap`.
- **Tenant superadmin / all-admins lockout.** Detection: support request. Recovery: `cypra admin promote --tenant=… --email=…` mints a magic-link.
- **Instance-admin lockout.** Detection: nobody can log in. Recovery: shell + `cypra admin reset-bootstrap` re-issues a setup token. Acknowledged limitation: requires shell access; PaaS without shell is unsupported for this recovery.
- **Tenant slug collision with reserved name.** Detection: registration-time validation rejects with `tenant.slug_reserved` or `tenant.slug_invalid`. User-visible: form error.
- **Tenant has no configured email provider.** Detection: any handler that needs to send mail (magic-link / password-reset / verification / admin invite) returns `tenant.email_provider_required` and writes an audit entry. The dashboard surfaces a hard-block banner on the affected tenant until configured. User-visible: tenant admin sees "configure an email provider before issuing magic links."
- **Tenant deleted while OIDC clients in flight.** Detection: downstream apps see `oidc.invalid_client` after sunset window. Recovery: signing keys remain on JWKS for 30 days post-tenant-delete (`state='sunsetting'`); after sunset, all tokens issued by that tenant fail to verify. Documented timeline in playbook.
- **GDPR DSR — user delete in flight.** Detection: deletion job writes audit; subsequent reads return `gdpr.user_deletion_in_progress`. Recovery: idempotent. User-visible: requesting party receives completion email.
- **GDPR DSR — backup restore would resurrect deleted user.** Detection: `cypra import` checks `gdpr_deletions` ledger; refuses with `gdpr.import_resurrect_blocked` unless `--allow-resurrect`.
- **Bot-mitigation hook returns failure.** Detection: hook returns `verifier.failed`; auth handler returns `auth.bot_mitigation_failed`. User-visible: "verification failed."
- **Validation failures.** All input-validation errors → `validation.failed` with `details: [{field, code, message}]`. User-visible: inline form errors.
- **Master-key rotation crashes mid-rewrap.** Detection: process exits with rotation `phase='rewrap'`, `rows_done < rows_total`. Recovery: `cypra admin rotate-master-key --resume` continues from `rows_done`. Old + new KEK both required at restart until cutover.

---

## 15. Deployment and Operations

### Environments

Cypra ships only for self-hosters. Reference setups:

- **Local dev:** `make dev` runs Postgres in Docker + Cypra via `go run` + Vite dev server through `portless` for HTTPS at `https://cypra.localhost` and `https://*.cypra.localhost` (wildcard). Per-tenant RP IDs work natively in dev.
- **Self-hosted prod (reference):** `docker compose up` in `with-tls` profile = Cypra + Postgres + Caddy on a single host.
- **Railway one-click:** template wires Cypra + Postgres + an S3-compatible bucket recommendation in README. `STORAGE_BACKEND=s3-compatible` defaulted because Railway filesystem is ephemeral.

### Deployment mechanism

- **CI:** GitHub Actions — `build.yml`, `release.yml` (multi-arch image to GHCR; SDK Go module published).
- **Release cadence:** on-demand. Semver. Pre-1.0 may break between minors; post-1.0 strict semver on REST API and CLI.
- **Docker image:** multi-arch (`linux/amd64`, `linux/arm64`); multi-stage Dockerfile; distroless final stage; embeds dashboard SPA built with pinned Bun.

### Backup and restore

- **`cypra export --out=cypra-backup.tar.zst --passphrase-from-stdin`** (or `--passphrase-file=<path>`). Bundles `pg_dump` + storage manifest (or full content for `local-disk`) + envelope-encrypted secrets export (KEK re-wrapped under the passphrase). **Mandatory passphrase** — refuses non-interactive runs without `--passphrase-file`.
- **`cypra import cypra-backup.tar.zst --passphrase-from-stdin [--allow-resurrect]`** restores into an empty Cypra. Refuses to overwrite existing data; refuses to resurrect DSR-deleted users without `--allow-resurrect`.
- **Operator's responsibility:** scheduling, off-site copies, periodic test-restore. Playbook entry covers all three.

### Migrations

- **Auto-migrations on boot** by default via `golang-migrate` runner (NOT GORM auto-migrate). Refuses to start with pending migrations unless `--skip-migrate`.
- `cypra migrate` runs migrations explicitly; `cypra migrate --to <version>` for targeted moves (down migrations required for every change).
- **Schema drops** never within a single release. Two-release deprecation: release N marks deprecated, N+1 removes.

### On-call

- **The operator.** Cypra ships:
  - `deploy/alerts.yml` (Prometheus rules).
  - "What to monitor" doc covering every metric in §12.
  - "First 24 hours after install" checklist.
  - "GDPR operator playbook" (data inventory, DSR runbook, breach-notification runbook, sub-processor list template).

---

## 16. Distribution and Licensing

### License

**MIT, forever, repository-wide.** Per Principle 7. Dashboard SPA, CLI, Go SDK, OIDC provider — all MIT.

### Distribution

- **Primary:** Multi-arch Docker image at `ghcr.io/watzon/cypra:<version>` and `:latest`.
- **Reference compose:** `docker-compose.yml` with two profiles — `default` (Cypra + Postgres; operator brings their own TLS proxy) and `with-tls` (Cypra + Postgres + Caddy with auto-ACME).
- **Railway one-click:** `watzon/cypra-railway-template` repo referencing the GHCR image. Sponsorship target.
- **Single-binary release:** `cypra` static binary attached to GitHub Releases.
- **Go SDK:** `github.com/watzon/cypra/sdk/go`, tagged `sdk/go/vX.Y.Z`.
- **TypeScript SDK at v1.1:** `@cypra/sdk` on npm.
- **Docs:** `docs/` Markdown in the repo at v1; static site post-v1.
- **No Helm, Coolify, Homebrew, install scripts at v1.** Community-contribution-friendly post-v1.

---

## 17. Risks and Open Questions

1. **`TenantScopedDB` bypass.** Impact: cross-tenant data leak; worst possible Cypra bug. Mitigation: typed-`tenantID` argument on every method including `Raw`; Postgres RLS as second belt; integration-test fuzzer on every CI build. ADR-0001.
2. **OIDC implementation correctness.** Impact: spec misreads = CVEs. Mitigation: `openid-conformance-suite` in CI nightly; lean v1 surface; ADR-0002 enumerates every endpoint and its spec section.
3. **Master-key loss.** Impact: irrecoverable encrypted columns. Mitigation: `cypra export` includes passphrase-wrapped secrets bundle; CLI emits "have you backed up your master key?" warning on first successful tenant create.
4. **Refresh-token reuse-detection edge cases.** Impact: failure-open or excessive UX punishment. Mitigation: ADR-0003; property-based tests on family graph; explicit handling of legit+attacker race.
5. **Caddy + Cypra `Host`-header trust.** Impact: tenant resolver picks wrong tenant if `Host` is spoofed. Mitigation: tenant resolver matches `<install-domain>` suffix exactly; rejects unknown hosts; `X-Forwarded-Host` honored only when `TRUSTED_PROXY_HEADERS` is set.
6. **Bun workspaces in CI.** Impact: dashboard / SDK build break. Mitigation: pinned Bun action; pnpm fallback documented in ADR-0010; switchable in one PR.
7. **Dashboard SPA bundle bloat.** Impact: image size grows; cold start slows. Mitigation: Vite chunking + tree-shaking; `make image-size` CI check fails over 80 MB.
8. **First-admin bootstrap on a remote VPS without DNS.** Impact: passkey enrollment requires HTTPS + RP ID. Mitigation: `PUBLIC_BASE_URL` is required at boot; setup wizard accepts any RP ID consistent with `PUBLIC_BASE_URL`. Operator must point DNS before bootstrapping. Documented limitation.
9. **PaaS without shell access.** Impact: unrecoverable instance-admin lockout. Mitigation: documented limitation; Railway template includes `railway run` instructions for `cypra admin` commands.
10. **Hand-rolled OIDC provider goes stale.** Impact: missed spec updates / security drift. Mitigation: pinned crypto/OIDC libs; quarterly errata review; conformance suite nightly.
11. **Postgres-backed rate limiter under load.** Impact: hot-key contention serializes (acceptable as stuffing defense), but legitimate users behind shared NAT see collateral 429s. Mitigation: documented; bot-mitigation hook (v1.1 providers) is the real defense; Redis lever ready for v2.
12. **GDPR claim drift.** Impact: legal exposure if Cypra-the-project is misread as a processor. Mitigation: every external doc clarifies "operator is controller"; Principle 5; landing-page copy reviewed before v1 launch.
13. **Bot-mitigation hook becomes "BYO = nobody uses it".** Impact: stuffing succeeds. Mitigation: rate-limit defaults are effective without bot mitigation; v1.1 prioritizes Turnstile (free tier).
14. **Vite proxy in dev creates dev/prod cookie mismatch.** Impact: works in dev, broken in prod. Mitigation: `portless` makes dev single-origin HTTPS; we never use `vite proxy` for the API.
15. **Operator missing `portless`.** Impact: contributor friction. Mitigation: `make dev` checks for `portless`, prints install instructions; documented in `CONTRIBUTING.md`.
16. **Storage-backend swap orphans objects.** Impact: profile pictures unreachable after backend switch. Mitigation: documented as a known limit at v1; `cypra admin migrate-storage` lands post-v1.
17. **JWKS cache TTL > overlap window at downstream.** Impact: tokens signed with new key fail at clients caching JWKS for >30 days without `kid`-miss refetch. Mitigation: `Cache-Control: max-age=300` on JWKS; operator playbook documents the downstream-client expectation; 30-day overlap is the back-stop.
18. **Issuer URL pinning brittleness.** Impact: changing `PUBLIC_BASE_URL` shape between versions breaks every downstream OIDC client. Mitigation: `PUBLIC_BASE_URL` is part of the §11 stable bootstrap contract; documented as "do not change without re-issuing all clients."
19. **Cross-tenant subdomain shadowing.** Impact: tenant slug `www` shadows install-level routes. Mitigation: reserved-word denylist enforced at tenant creation; CHECK constraint on `tenants.slug` regex.
20. **Upstream OAuth state/nonce mishandling.** Impact: GitHub-style "OAuth state not validated" CVE. Mitigation: state/nonce mandatory at v1; signed-state to prevent client-side tampering; ADR-0012 covers.
21. **Postgres role separation drift.** Impact: a single overprivileged `DATABASE_URL` defeats audit-log immutability. Mitigation: `cypra serve` warns at boot if `MIGRATE_DATABASE_URL` falls back to `DATABASE_URL`; CI integration test runs against the runtime role and asserts `DELETE` on `audit_entries` fails.
22. **Single-process rate limiter at v2.** Tracked: when an operator credibly needs multi-instance Cypra, Redis is the lever.
23. **Embeddable widget RP-ID story (v1.1).** Tracked: per-tenant RP ID model at v1 means widget on third-party domains can't directly use the tenant's passkey credential. Resolution likely requires WebAuthn cross-origin extensions or a Cypra-hosted iframe.

---

## 18. Glossary

- **Active key** — an OIDC signing key currently used to sign new tokens. At most one per tenant.
- **Bootstrap token / Setup token** — single-use first-boot token, printed to logs (with `redacted-on-export` tag) and displayed in-band on `/setup/<token>`. Synonyms.
- **Break-glass CLI** — `cypra admin` subcommands that recover from total lockout; require host shell access.
- **Caddy reference compose** — the `with-tls` profile of `docker-compose.yml` that fronts Cypra with Caddy.
- **CNAME custom domain** — tenant-supplied DNS CNAME pointing at the install. v1.1.
- **Dashboard** — the React SPA served from `/dashboard/*`, embedded in the Go binary.
- **DEK** — data encryption key. Per-row symmetric key wrapped under the KEK.
- **DSR** — data subject request (GDPR access / deletion / portability).
- **Embeddable widget** — iframe-isolated login UX hosted on the tenant's Cypra domain. v1.1.
- **End-user** — a user of a _downstream_ app that delegates auth to Cypra; not a Cypra admin.
- **Envelope encryption** — per-row DEKs encrypted under a single KEK held in memory.
- **Family / `family_id`** — refresh-token lineage. A family is created at first token-endpoint mint for a `(client, user, scope)` tuple, continues across rotation, and ends at logout, full revocation, or detected reuse. Reuse cascade-revokes the entire family.
- **Family-tree reuse detection** — the strategy where presenting an already-consumed refresh token revokes every descendant of the original.
- **Hosted login** — server-rendered Go-template pages on `<tenant>.<install-domain>` that handle sign-in.
- **Install / Install domain** — one Cypra deployment (one binary + one Postgres). The install domain is the public DNS name (`auth.example.com`); tenants live at `<tenant>.<install-domain>`. "Install" and "instance" are synonyms in v1.
- **Instance admin** — a Cypra-platform-level admin (the operator). Lives outside the tenancy model.
- **Issuer URL** — the OIDC issuer string, equal to `https://<tenant>.<install-domain>` per tenant. Used as the `iss` claim, the `issuer` field in discovery, and the value downstream OIDC clients pin.
- **JWKS** — JSON Web Key Set (RFC 7517). Cypra publishes per-tenant JWKS at `/.well-known/jwks.json` listing `active`, `overlap`, and `sunsetting` signing keys. Downstream clients fetch and cache; should refetch on `kid` miss.
- **KEK** — key encryption key. The master key loaded at boot from env or file; never persisted.
- **Magic link** — single-use email link that signs a user in.
- **Master key** — synonym for KEK in operator-facing docs.
- **Operator** — the human running the install. Always a controller in GDPR terms.
- **OIDC client** — an app registered to authenticate via Cypra's OIDC provider. At v1, one per project.
- **Operator playbook** — Cypra-shipped Markdown doc covering ops, GDPR, backup, incident response, and downstream-client guidance.
- **Overlap window** — the 30-day period during OIDC signing-key rotation where both old and new keys are in JWKS.
- **PAT** — Personal Access Token. Issued from the dashboard; used by the Go SDK and admin REST API.
- **portless** — local dev tool that provides HTTPS + wildcard `.localhost` subdomains, mandatory for WebAuthn in dev.
- **Project** — an OIDC-client-bearing app under a tenant. At v1, one project = one OIDC client; the cardinality may lift in v1.1+.
- **PKCE** — proof key for code exchange (RFC 7636). Mandatory at v1.
- **RP ID** — relying party identifier. WebAuthn-scoped registrable domain. **Per-tenant in Cypra v1** (`<tenant>.<install-domain>`).
- **Storage abstraction** — internal Go interface with `local-disk` (HMAC-signed proxied URLs) and `s3-compatible` (native presigned URLs) backends.
- **Sunsetting key** — an OIDC signing key whose tenant has been deleted; remains on JWKS for 30 days post-tenant-delete, then hard-deleted.
- **Tenant** — a customer organization in the multi-tenant model. The unit of isolation, branding, OIDC issuer, and (later) custom domain.
- **TenantScopedDB** — the Go wrapper around `*gorm.DB` that enforces the tenant-isolation invariant in code, including for raw SQL.
- **Tenant-scoped query layer** — `TenantScopedDB` + Postgres RLS, taken together.
- **Upstream provider** — an external OAuth/OIDC IdP (Google at v1) configured per tenant.

---

## Appendix A — ADRs to author during Phase 0

- `docs/adr/0001-tenant-isolation-mechanism.md` — `TenantScopedDB` (with typed-`tenantID` raw-SQL escape hatch) + Postgres RLS + checkout reset hook; fuzzer test plan.
- `docs/adr/0002-oidc-provider-surface.md` — endpoints, supported algorithms, conformance plan; explicit non-support for DCR / introspection / implicit / hybrid.
- `docs/adr/0003-refresh-token-reuse-detection.md` — family-tree algorithm with sequence diagrams, race semantics, property-based tests.
- `docs/adr/0004-envelope-encryption-and-master-key-rotation.md` — KEK/DEK scheme, online resumable re-wrap procedure, `master_key_rotations` state, recovery story.
- `docs/adr/0005-oidc-signing-key-rotation-schedule.md` — 90-day rotation, 30-day overlap, JWKS Cache-Control, downstream-client expectation.
- `docs/adr/0006-bootstrap-setup-token.md` — single-use, log-printed (redacted-on-export tag) + in-band display, exclusive-by-construction, `cypra admin reset-bootstrap` revocation.
- `docs/adr/0007-storage-abstraction.md` — `Storage` interface, local-disk HMAC-signed URLs, S3-compatible presigned URLs, TTL bounds.
- `docs/adr/0008-tls-via-reverse-proxy.md` — no built-in ACME; Caddy reference compose; trusted-proxy header model; v1.1 CNAME story via Caddy `on_demand_tls`.
- `docs/adr/0009-gorm-with-explicit-sql-migrations.md` — GORM for queries via `TenantScopedDB`; `golang-migrate` SQL files for DDL; named perf-critical paths use raw SQL **through `TenantScopedDB.Raw`** with mandatory `tenantID`.
- `docs/adr/0010-bun-workspaces-with-pnpm-fallback.md` — Bun preferred; pinned in CI; pnpm fallback documented.
- `docs/adr/0011-personal-access-tokens.md` — PAT format, scoping (tenant + permissions), rotation, revocation on role change, audit-log integration.
- `docs/adr/0012-upstream-oauth-state-and-nonce.md` — signed `state`, `nonce` validation, callback error mapping, replay protection.
- `docs/adr/0013-postgres-role-separation.md` — `cypra_runtime` vs `cypra_migrate`; column-level `GRANT` on `audit_entries`; CI test asserting `DELETE` on audit log fails.

---

## Revisions

<!-- After PLAN.md is sealed, edits land here as a dated entry. Do not silently mutate sealed sections. -->
