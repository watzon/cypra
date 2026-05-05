# Cypra — Brainstorm

**Status:** sealed
**Date:** 2026-05-04
**Working title:** Cypra (shipping name)
**Owner:** @watzon

---

## 1. The idea

Cypra is an open-source, self-hosted, multi-tenant authentication platform aimed at solo developers and small dev shops who want passkeys, magic links, social logins, and an OIDC provider — without the pricing of WorkOS/Clerk/Auth0 or the operational complexity of Keycloak, Ory, or Authentik. It is **OSS-first, not SaaS-first**: the source is the product. The wedge against existing self-hostable competitors is **a setup story that actually fits in an afternoon** and a small-team self-host focus rather than an enterprise-on-prem one.

"Dead-simple setup" is operationalized as: **single `docker compose up`, then ≤10 dashboard steps from first boot to first working end-user sign-in on a downstream app.**

## 2. Personas and use cases

### Primary persona
A solo developer or small dev-shop owner who wants auth to *not* be a headache. They have one to a handful of apps, want passkeys / magic-link / Google sign-in / OIDC working in an afternoon, prefer to own the data and host the binary themselves. Triggered by: starting a new app, hitting Auth0/Clerk/WorkOS pricing, getting frustrated trying to stand up Keycloak.

### Secondary personas
- **Small SaaS founders** giving *their* end-customers tenant-style auth (B2B-multi-tenant) without paying WorkOS prices.
- **Agencies** bundling auth into client deliverables — one Cypra deployment per client engagement, or one shared deployment with a tenant per client.
- **Self-host enthusiasts** running their own infra and wanting a drop-in identity layer.

### Canonical story
Operator runs Cypra (Docker Compose locally, or Railway one-click in the cloud). On first boot, Cypra prints a one-shot **setup token** to its logs. Operator opens the install URL, redeems the token, completes passkey enrollment + mandatory backup-code export, becomes the first **instance admin**. They create a tenant for themselves, add one or more projects, configure an OIDC provider connection (Google) and an email provider (terminal in dev, Resend or SMTP in prod), and plug the resulting OIDC client config into their app via an off-the-shelf auth library. End-users of *that* app sign in via Cypra-hosted login pages on `<tenant-slug>.<install-domain>`.

### Other use cases
1. **Migrating off Auth0 / Clerk / WorkOS** — bulk import via a `cypra import` CLI (post-v1) plus a "users re-set passwords on first login" UX. Hash-method compat is explicitly out of scope.
2. **Adding SSO to an existing app** by pointing it at Cypra as an OIDC provider and configuring upstream IdPs (Google in v1; more deferred; SAML deferred).
3. **Running personal-project auth at low cost** — solo dev runs one tenant with several projects under it.

### Anti-personas
- **Large enterprises** requiring FedRAMP, SOC 2 Type II, HIPAA-grade controls.
- **Apps needing horizontal scale on day one** — millions of users, multi-region writes.
- **Teams that want a fully-managed/hosted-only experience.** They should buy Clerk or WorkOS.

## 3. Scope

### v1 must-haves
- **Identity model:** tenants = customer organizations; projects scoped under tenants; **users scoped per-tenant** (same email in two tenants = two users).
- **Roles within a tenant:** owner / admin / member. Multi-admin is supported.
- **Instance admin** role outside the tenant model: the human(s) running the install. Can break-glass into any tenant via CLI but does not auto-log into tenant dashboards.
- Default tenant routing via `<tenant-slug>.<install-domain>`. **Custom CNAME domains deferred to v1.1.**
- Auth methods (end-users): **email + password (Argon2id)**, **magic link**, **passkeys**, **Google OAuth**, and **Cypra as OIDC provider** for downstream apps.
- **OIDC v1 surface (intentionally lean):** authorization code + PKCE only; refresh tokens with rotation + reuse detection; discovery doc, JWKS, consent screen, revocation endpoint. **No** dynamic client registration, **no** token introspection, **no** implicit/hybrid grants.
- Two-factor: **TOTP and WebAuthn**.
- Dashboard: Vite + React + Tailwind v4 + ShadCN via Watermelon UI; embedded into the Go binary; Vite dev proxy in development.
- **Hosted login pages** at v1. **Embeddable widget deferred to v1.1.**
- Email providers: **terminal (dev), SMTP, Resend**.
- User profile: profile picture + JSON metadata field.
- **Storage abstraction with `local-disk` and `s3-compatible` backends** (R2/B2/MinIO/AWS). Local disk is the Compose default; `s3-compatible` is the Railway-template default.
- Dashboard auth: **passkey-based with mandatory backup-code export** at enrollment.
- **Bootstrap:** first-boot setup token printed to logs; redeemed once, then burns; mints the first instance admin and forces passkey enrollment.
- **CLI break-glass** path (`cypra admin …`) for tenant superadmin lockout, instance-admin recovery, and other sysadmin tasks. Documented prerequisite: shell access to the host (acknowledged limitation on managed PaaS).
- **App-level envelope encryption** for sensitive columns (OIDC private keys, OAuth client secrets, SMTP/Resend creds, TOTP secrets). Master key supplied via env or mounted file; never persisted in DB.
- **JWT access tokens + opaque refresh tokens with server-side storage**, rotation, and reuse-detection-based family invalidation.
- **All non-secret config managed in the dashboard, not env.** Bootstrap-only env: `DATABASE_URL`, `MASTER_KEY` (or file path), `LISTEN_ADDR`, `PUBLIC_BASE_URL`, `TRUSTED_PROXY_HEADERS` mode, `STORAGE_BACKEND`. Enumerated minimal set.
- **GDPR-enabling features** (see §8 Compliance for the framing).
- Observability: Prometheus `/metrics`, OpenTelemetry traces, structured JSON logs to stdout, `/healthz` and `/readyz`.
- **Automatic DB migrations on container boot, with an opt-out flag** for change-control environments (operator can run `cypra migrate` manually).
- SDK: **Go only at v1.** TypeScript SDK deferred to v1.1.
- Distribution: single multi-arch Docker image, `docker-compose.yml` in repo, **Railway one-click template** as a sponsorship play.
- Monorepo: **Bun workspaces** preferred; pnpm fallback at PLAN time if Bun blocks anything. SDK consumers see published npm packages, so Bun-on-Windows is an internal-dev concern only.
- **Tenant isolation as a named invariant** (see §8): every data access is scoped by `tenant_id` at a single enforced layer; the enforcement mechanism is a PLAN-phase decision.

### Explicit non-goals for v1
- SAML 2.0.
- SMS / Twilio.
- Custom CNAME domains (v1.1).
- Embeddable JS widget (v1.1).
- TypeScript SDK (v1.1).
- Bulk CSV import + bulk password reset UI (replaced by a `cypra import` CLI post-v1).
- Hash-method-compatible migration from Auth0/Clerk/WorkOS.
- Helm chart, Kubernetes operator, multi-region deployments, multi-Cypra-instance-per-Postgres.
- External KMS dependency.
- Cypra-shipped backup tooling (operators run `pg_dump`).
- A hosted Cypra SaaS.
- Compliance certifications (SOC 2, HIPAA, FedRAMP, ISO 27001, PCI).
- i18n / non-English UI (English-only at v1, gettext-ready strings so it's not a rewrite later).
- Cypra-side bot mitigation (Turnstile/hCaptcha integration is BYO via a hook).

### Deferred to later
- SAML 2.0.
- SMS-based MFA + magic links via Twilio.
- Custom CNAME domains (tenant-level first; project-level remains an open question).
- Embeddable JS widget — **iframe-isolated, hosted on the tenant's Cypra domain, postMessage protocol back to the host page** (Stripe Elements / Clerk-style boundary).
- TypeScript SDK; later: Python, Ruby, PHP, Rust.
- Additional upstream OIDC providers (GitHub, Apple, Microsoft, Discord, …).
- Walkthrough videos for each provider's credential setup.
- OIDC dynamic client registration, token introspection, additional grant types.
- Helm chart, Coolify/Render templates, install scripts.
- PII expansion: phone numbers, addresses, more profile fields.
- Tenant logo / email-template attachment storage (uses the same storage abstraction when added).
- Hosted Cypra offering.
- i18n.
- Migration *between* Cypra instances (org acquisition / split / demo-to-prod tooling).

### Hard constraints
- **Stack:** Go backend, Postgres, GORM (with raw SQL via GORM at named perf-critical paths — list to be pinned in PLAN), Vite + React + Tailwind v4 + ShadCN/Watermelon UI dashboard, single-binary distribution.
- **No forced external managed services.** Cypra runs end-to-end with just the Cypra container + Postgres + a storage backend (local disk by default).
- **Lean dev budget.** Free-tier-only during development.
- **License: MIT, forever.** No future BSL/dual-license rug-pull. The competitive moat is operational excellence + brand if a hosted offering ever happens, not a restrictive license.
- **No hard timeline.** Solo project, ships when it's right.

## 4. Success

### Ship criteria
"Cypra v1 is good enough to ship when I can self-host it in the cloud, redeem the bootstrap token, set up a tenant for myself, add a project, connect Google OAuth, and connect a Next.js web app via an off-the-shelf OIDC client (target: Auth.js / NextAuth) — end to end, in a single sitting, on `<tenant>.<install-domain>` (no CNAME required)."

### In-the-wild signals
- GitHub stars (primary public signal).
- Issue / PR activity from people who aren't the author.
- Reports of self-hosting Cypra in production for real apps.

### Failure mode
Cypra exists, the docs are nice, the tech is fine — but nobody self-hosts it because (a) the setup story is still painful, (b) "yet another auth project" fatigue, or (c) the Auth0/Clerk pricing wall isn't actually as painful as we assumed for the target audience.

## 5. Surface area

### Platforms
- Self-hosted server (Linux/amd64 + Linux/arm64 Docker images).
- Web dashboard (browser, served by the Cypra binary).
- Hosted login pages (browser, served by the Cypra binary on `<tenant>.<install-domain>`).
- Go SDK (server-side).

### Stack preferences and aversions
- **Preferences:** Go, Postgres, GORM, single-binary distribution, Docker Compose, Vite + React + Tailwind v4 + ShadCN via Watermelon UI, Bun workspaces, MIT license.
- **Aversions:** ENV-hell config, forced external managed services, restrictive licenses, heavy Java/JVM stacks, micro-services-first architectures.
- **Dev-vs-prod note:** Vite dev mode proxies the Go API. Cookie domain, CORS, and CSRF behavior must be designed so dev (cross-origin) and prod (single-origin) match in security model — flagged for PLAN.

### Integrations with existing code
- None at the project level (greenfield repo). Note: the *whole product* is integrations with downstream customer apps via OIDC; "greenfield" refers only to the Cypra repository.

### Hosting and distribution
- Multi-arch Docker image as primary distribution.
- `docker-compose.yml` reference setup in repo.
- Railway one-click template at v1 (sponsorship target).
- Single-binary release for direct VM/bare-metal install.
- Helm / Coolify / install-script — deferred.
- **TLS posture:** v1 expects an upstream reverse proxy (Caddy/Traefik/Cloudflare) for TLS termination; Cypra documents trusted-header config and provides example Caddy/Traefik configs. Built-in ACME is deferred and tied to v1.1 CNAME work.

## 6. Data and integrations

### Owned data
- Tenants (organizations) and tenant settings.
- Projects (under tenants).
- Tenant memberships and roles (owner / admin / member).
- Instance admins.
- Users (scoped per-tenant).
- Sessions and refresh tokens (with rotation/reuse-detection state).
- OIDC clients (Cypra-as-provider) and their secrets.
- Upstream IdP / OAuth provider configurations and their secrets.
- SMTP / Resend / email-provider configs and their secrets.
- Audit log entries.
- User profile pictures (via storage abstraction).
- User JSON metadata.
- Passkeys (WebAuthn credential records), TOTP secrets, backup codes (hashed).
- OIDC signing keys.

### Borrowed data
- Identity from upstream OAuth providers (Google in v1) — minimal stored beyond linkage.

### Third-party services
- **Google OAuth** — read user profile + email at sign-in.
- **Resend** — write (transactional email). API-key auth.
- **SMTP server (operator-supplied)** — write. Username/password auth.
- **S3-compatible object storage (operator-supplied, optional)** — read/write profile-picture blobs. Defaults to local disk if not configured.
- **Twilio (deferred)** — write (SMS).

### PII / sensitive classes
- Emails, hashed passwords (Argon2id), passkeys, TOTP secrets, backup codes (hashed), profile pictures, IP addresses (audit/rate-limit), user agent, JSON metadata (operator-controlled — could be anything).

### Offline behavior
- Not applicable for v1. Cypra is a server.

### Lifecycle
- **Retention:** audit log retention default + tenant-configurable cap (open question on default).
- **Deletion:** GDPR-mandated user/tenant delete hard-deletes user records and authoritatively scrubs PII; **audit log entries are redacted, not deleted**, with documented legal-basis justification.
- **Export:** per-user GDPR data-export endpoint; tenant-admin bulk export.
- **Tenant-deletion cascade:** revokes all sessions, invalidates all refresh tokens for that tenant, marks OIDC clients as deleted (downstream apps get definitive errors rather than silent failures); JWKS for that tenant remains queryable for a documented sunset window for downstream caches.

## 7. Failure and edge cases

### Top failure modes
1. **OIDC signing-key compromise.** Mitigation: app-level envelope encryption at rest, key-rotation with overlap window (default schedule TBD in PLAN), audit log of key access.
2. **DB exfiltration.** Argon2id slows offline cracking; envelope-encrypted secrets are useless without the master key, which is never in the DB.
3. **Mass account-takeover via stuffing.** Per-IP / per-account / per-tenant rate limiting; uniform error messages; breach-notification runbook (operator-driven — see §8).

### Integration outages
- **Email provider down.** Magic link and password reset fail with a clear error UX; retry/queue in PLAN.
- **Upstream OAuth provider down.** Affected sign-in method fails; password / passkey / magic link still work.
- **Postgres down.** Cypra is down. Postgres-HA is the operator's responsibility.
- **Storage backend down.** Profile-picture reads fail gracefully (placeholder); writes return a clean error.

### Adversarial input
- Login abuse (stuffing, enumeration). Rate-limit + uniform errors. Bot-mitigation hook is BYO.
- Sign-up abuse. Rate-limit + email-verification-required mode.
- Profile-picture upload abuse. Type/size validation, content-type enforcement, optional re-encode.
- OIDC client misuse (open redirector, wildcard redirect URIs). **Strict redirect URI matching** (no wildcards) at v1.
- CSV import abuse — **deferred with the import feature** (post-v1 CLI).

### Recovery
- **End-user lost passkey** with no other factor → admin-mediated reset (admin can re-enroll, e.g., switch them to magic-link flow). Documented stance: if no admin can help, "tough luck, re-register."
- **End-user has only passkey + lost device** → backup codes (mandatory export at enrollment).
- **Tenant superadmin lockout** → another tenant admin (multi-admin supported) or instance-admin CLI break-glass.
- **All admins of a tenant locked out** → instance-admin CLI break-glass (`cypra admin promote --tenant=… --email=…`).
- **Instance admin lockout** → host shell access + `cypra admin` CLI (acknowledged caveat: doesn't work cleanly on shell-less PaaS; documented as a known limitation).
- **Backup/restore:** operator's responsibility; Cypra docs include a recommended `pg_dump` workflow and a "what to also back up alongside Postgres" checklist (storage backend, master key file). No tooling at v1.

## 8. Security and privacy

### Auth (dashboard)
- **Passkey-first** for instance admins and tenant admins/members. Backup codes mandatory at enrollment.
- **Bootstrap setup token** (printed to logs on first boot, single-use, auto-expiring) is the only path to creating the very first instance admin. Subsequent admin invites are email-magic-link-based and require an existing admin to issue.
- Magic link / admin-mediated reset as recovery paths.

### Auth (end-users)
- Email + password (Argon2id), magic link, passkeys, Google OAuth at v1.
- TOTP and WebAuthn second factor.
- Cypra as OIDC provider for downstream apps; JWT access tokens + opaque server-stored refresh tokens with rotation and reuse detection.

### Tenant isolation (named invariant)
- **Every data access is scoped by `tenant_id` at a single enforced layer.** No ad-hoc raw-SQL path may bypass that layer. Enforcement mechanism (Postgres RLS vs. middleware-enforced query scopes vs. ORM scopes) is a PLAN-phase decision; the invariant is non-negotiable.

### Secrets
- App-level envelope encryption for sensitive columns (OIDC private keys, OAuth client secrets, SMTP/Resend creds, TOTP secrets at rest).
- Master key via env var or mounted file. Never in the DB.
- Master-key rotation: documented re-wrap procedure (PLAN to design); loss of master key = irrecoverable encrypted data, documented up front.
- No external KMS dependency at v1.
- Secrets never logged.
- Bootstrap-only env vars enumerated: `DATABASE_URL`, `MASTER_KEY` (or file), `LISTEN_ADDR`, `PUBLIC_BASE_URL`, `TRUSTED_PROXY_HEADERS`, `STORAGE_BACKEND` and its config.

### Threat model
**Who:** opportunistic credential-stuffers hitting public Cypra instances; targeted attackers going after a specific tenant; potentially malicious tenants attempting to escape isolation; operators of forks-gone-wrong.
**What they want:** end-user accounts, stored secrets (OIDC private keys → forged tokens, OAuth client secrets → impersonation), user PII, administrative access.
**Cypra's hard requirements:** the tenant-isolation invariant; encrypted secrets at rest with key off-DB; rate-limiting at IP / account / tenant; audit log of admin actions and key access; uniform error messages on auth endpoints; strict redirect-URI matching; refresh-token rotation with reuse detection; passkey-first admin auth; setup-token bootstrap.

### Compliance
**Framing:** Cypra is self-hosted, so the operator is the **GDPR controller** and Cypra-the-project is **not a processor.** Cypra ships **GDPR-enabling features**, not a compliance certification:
- Per-user data export endpoint (DSR access).
- Per-user account deletion with PII scrub; audit log redaction (not deletion) for legal-basis defensibility.
- Per-tenant deletion that cascades sessions/tokens/clients with documented behavior.
- Audit log of admin and security-relevant actions.
- A **GDPR operator playbook** doc covering: what data Cypra processes, how to fulfill DSARs using Cypra's endpoints, sub-processors the *operator* will be using (Google, Resend, the operator's storage backend), and a breach-notification runbook the operator can adapt.
- Cypra does not assume DPO, DPA, or processor responsibilities on behalf of the operator.

Out of scope: SOC 2, HIPAA, FedRAMP, ISO 27001, PCI.

## 9. Operations

### Observability
- Prometheus `/metrics`.
- OpenTelemetry tracing hooks.
- Structured JSON logs to stdout.
- `/healthz` and `/readyz`.

### Deployment cadence
- On-demand. Tagged Docker releases.

### On-call
- The operator. Cypra docs include a "things to monitor / alert on" checklist (open question on the exact list).

### Cost ceiling
- **Lean.** No forced external services. Production should be runnable on a small VPS + Postgres for the small-team-target audience.

### Upgrade path
- Automatic migrations on boot, with `--skip-migrate` flag and a `cypra migrate` command for change-control shops.
- OIDC signing-key rotation: overlap window (default schedule TBD in PLAN; automatic with operator override).
- Master-key rotation: documented re-wrap workflow (PLAN to design).
- Breaking config changes: surfaced in release notes; dashboard-config-not-env reduces blast radius.

## 10. Open questions

- [ ] Project-level custom domains: ever, or is tenant-level the permanent answer (post v1.1 CNAME)?
- [ ] Audit log retention default and tenant-overridable cap. What fields are immutable; what's the export format?
- [ ] Rate-limiting defaults (per-IP, per-account, per-tenant) and whether they live in dashboard config or are hard-coded.
- [ ] OIDC signing-key rotation: default schedule, automatic vs. operator-triggered, JWKS publication semantics for downstream caches.
- [ ] Email-provider default: do we ship a "use Resend free tier" recommendation in onboarding to spare solo devs from SMTP setup?
- [ ] Bot-mitigation hook shape — Turnstile/hCaptcha integration interface (Cypra-hosted? webhook? plugin?). v1 is BYO; what does "BYO" actually look like at the API surface?
- [ ] Embeddable widget (v1.1 work): final shape of postMessage protocol, host-page CSP guidance.
- [ ] DPA-template-and-friends: do we ship operator-facing GDPR doc templates (DPA, ROPA, sub-processor list) at v1, or just the playbook?
- [ ] Refresh-token reuse-detection model — *which* family invalidation strategy (full descendants? just suspicious branch?), and what does "logout everywhere" mean for the user.
- [ ] Cookie/session domain model when both hosted login and (later) embedded widget exist on different RP IDs vs. CNAMEs — passkey-roaming implications.
- [ ] Clock-skew tolerance defaults for OIDC `iat`/`exp`/`nbf` and TOTP windows.
- [ ] Storage backend: defaults documented, but do we want to ship `s3-compatible` as the *only* remote backend, or also a built-in Cloudflare R2 / Backblaze B2 quick-start preset in the dashboard?
- [ ] Admin-mediated reset semantics — does the admin reset a password, disable MFA, enroll a new factor on the user's behalf, or all three (with separate permissions)?
- [ ] Profile-picture content scanning — virus/CSAM scan hook at upload, or operator's problem entirely?
- [ ] "Things to monitor / alert on" checklist — final list of recommended Prometheus alerts for operators.
- [ ] Auto-migration `--skip-migrate` operator UX: does Cypra refuse to start if a migration is pending, or start in a degraded read-only mode?
- [ ] Multiple Cypra instances behind one Postgres: hard-no for v1 (lean), or supported via per-install schema/prefix? Lean toward hard-no; capture the decision.
- [ ] Backups: is "no Cypra-shipped tooling at v1" really fine, or do we want at least a `cypra export` CLI that bundles `pg_dump` + the storage-backend manifest + a sealed-secrets export?
- [ ] CNAME work in v1.1: built-in ACME issuance (per-tenant cert) vs. operator-supplied wildcard cert via reverse proxy. Tied to TLS posture decision.
