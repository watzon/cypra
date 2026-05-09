# Phase 1-13 Validation Gap Report

Date: 2026-05-06

Scope: Static/codebase validation of `TASKS.md` phases 1 through 13, using one read-only inspector per phase plus a local `./bin/agent-ci run --quiet --all` attempt. This report focuses on holes to patch. It does not validate Phase 14.

## Executive Summary

The repo contains substantial scaffolding for phases 1-13, but many checked boxes are over-marked. The highest-risk gaps cluster around tenant isolation bypasses, incomplete WebAuthn/OIDC/session semantics, export/import being a JSON snapshot rather than a restoreable backup, dashboard surfaces relying on demo data, and live/e2e gates that skip or are explicitly deferred.

Current local CI is red. `./bin/agent-ci run --quiet --all` fails in frontend lint with 17 errors across `dashboard/src/App.test.tsx`, `components.tsx`, `modals.tsx`, and `screens.tsx`.

## Cross-Phase Gate Failures

- P0: CI gate is red. `bun run lint` fails with 17 TypeScript ESLint errors, including class-instance spread in `dashboard/src/App.test.tsx`, unbound methods, `require-await`, unsafe assignment, `consistent-type-definitions`, and unnecessary conditions.
- P0: Tenant isolation is not exclusively enforced through `TenantScopedDB`. `internal/db/tenant_scoped_db.go` exposes `DB() *gorm.DB`; `internal/db/instance_admin.go` exposes unrestricted GORM access; several handlers use raw `database/sql` directly.
- P0: Fuzzer enrollment is mostly synthetic. Multiple phase-specific fuzzer tests mutate raw SQL queries rather than exercising the actual HTTP handlers introduced in those phases.
- P1: Visual validation gates are often documented as handoff text only and do not cover the stated full route/state/mode/breakpoint matrix.
- P1: Several live/deployment/e2e gates are skipped or explicitly deferred while the phase remains marked complete.

## Phase 1: Schema, Types, Tenant Isolation

Status: Partially complete with P0 isolation holes.

- P0: `TenantScopedDB` exposes raw bypasses. `internal/db/tenant_scoped_db.go:25-27` exposes `DB() *gorm.DB`; `internal/db/instance_admin.go:22-24` exposes `AsInstanceAdmin(ctx) *gorm.DB`.
- P0: Production boot does not appear to install the GORM tenant plugin. `cmd/cypra/main.go:97-102` wraps GORM with `db.NewTenantScopedDB(gormDB)` without `gormDB.Use(...)`.
- P0: RLS setter middleware is later implemented as a no-op in `internal/httpserver/server.go:303-305`, weakening the Phase 1 invariant in actual HTTP paths.
- P1: Connection hook requirement is only partially satisfied. `internal/db/tenant_plugin.go:19-60` uses callbacks, not checkout/return hooks, and is not visibly wired in production boot.
- P1: RLS verification only checks `projects`, not every tenant-scoped table. See `internal/dbtest/migration_test.go:42-51`.
- P2: Column/type matching is weakly tested. Table existence is tested in `internal/dbtest/migration_test.go:12-27`, but column types are not comprehensively asserted.

Appears present: Phase 1 migrations exist, PLAN section 8 tables mostly exist, role grants exist, RLS policies are broadly present, and ADRs 0001/0009/0013 are accepted.

## Phase 2: Crypto, Secrets, Sessions, Bootstrap

Status: Core primitives exist, but bootstrap and master-key rotation have correctness holes.

- P0: First instance-admin creation can bypass setup-token redemption. `internal/bootstrap/bootstrap.go:91-103` accepts any non-nil setup token UUID; tests only reject `uuid.Nil`.
- P0: Master-key rotation target model does not match real encrypted columns. `internal/crypto/master_key_rotation.go:164-170` treats a column as an encrypted DEK, while signing keys store a JSON envelope in `internal/crypto/signing_keys.go:50-55` and `135-161`. CLI targets `private_key_encrypted` as the DEK column in `cmd/cypra/main.go:570-578`.
- P0: Master-key rotation progress is not row-crash-safe. Rewrap and `rows_done` updates are separate statements in `internal/crypto/master_key_rotation.go:86-92` and `164-170`.
- P1: Signing-key rotation exposes `RotateDue`/`ForceRotate`, but no automatic ticker/scheduler was found. See `internal/oidc/rotation.go:67-94`.
- P1: Repeated `ForceRotate` can violate the one-overlap unique index because it blindly moves active to overlap. See `internal/oidc/rotation.go:50-61` and `db/migrations/0001_init.up.sql:293-294`.
- P1: Refresh-token reuse metric is only a callback; `/metrics` later hardcodes `cypra_oidc_refresh_reuse_detected_total` as zero.
- P2: Refresh-token property test is fixed-shape rather than property-based/random. See `internal/sessions/refresh_test.go:74-93`.

Appears present: Argon2id, AES-GCM envelope primitives, signing-key generation, JWT claim shape/leeway, refresh-token family rotation basics, bootstrap mint/redeem/revoke basics, seed helpers, and ADRs 0004/0005/0006.

## Phase 3: HTTP Server, Tenant Resolver, REST Skeleton

Status: Runnable server skeleton exists, but major middleware and API requirements are incomplete.

- P0: Rate limiter is not wired into HTTP/auth routes. `internal/ratelimit/ratelimit.go:15-32` exposes `Allow`, but no `internal/httpserver` route wiring was found.
- P0: Audit export appears insufficiently permission-guarded. `/api/v1/audit/export` is registered in `internal/httpserver/server.go:112-120`; handler in `internal/audit/audit.go:47-64` performs no `audit.read` check.
- P0: Structural audit wrapper is missing. Direct `audit.Write` calls exist in `internal/httpserver/handlers.go`, conflicting with the required registry middleware and direct-call lint rule.
- P0: RLS setter middleware is a no-op in `internal/httpserver/server.go:303-305`.
- P1: Projects and users are not full CRUD. Projects only list/create, users only list/create plus profile/DSR routes. See `internal/httpserver/server.go:169-181`.
- P1: Tenant resolver does not use `TRUSTED_PROXY_HEADERS` as specified. It trusts `X-Forwarded-Host` based on `X-Cypra-Trusted-Proxy` or loopback in `internal/httpserver/server.go:273-300`.
- P1: Email worker is `RunOnce`, not a started goroutine pool, and uses fixed one-minute retry rather than exponential backoff. See `internal/email/worker.go:16-83`.
- P2: OpenAPI doc is incomplete and only lists a subset of routes. See `internal/httpserver/server.go:239-240`.
- P2: Request logging lacks required `request_id`, `tenant_id`, and `actor_id` fields. See `internal/httpserver/server.go:265-270`.

Appears present: `serve`, `migrate`, `/healthz`, `/readyz`, tenant resolver basics, tenant CRUD, terminal sender, basic auth/PAT scaffolding, and admin SDK skeleton.

## Phase 4: Auth Methods, Email Providers, Storage

Status: Service stubs exist, but WebAuthn/OAuth/session semantics are not production-grade.

- P0: Passkey/WebAuthn is not a real WebAuthn implementation. `internal/auth/webauthn/webauthn.go:19-38` stores and compares credential/RP IDs; no `go-webauthn` dependency, challenge verification, origin checks, signature checks, or sign-counter logic were found.
- P0: Invite redemption skips the specified continuation state machine. `internal/auth/invite/invite.go:74-111` consumes the invite and creates the user/admin directly; no `GET /invite?token=...`, session-bound continuation, optional password step, or mandatory passkey enrollment was found.
- P0: Password sign-in and magic-link verify do not establish sessions. `internal/httpserver/auth_handlers.go:85-105` and `142-153` return user data without minting cookies/JWT/session records.
- P0: Google OAuth is not a real upstream round trip. Handlers use hardcoded client config in `internal/httpserver/auth_handlers.go:344-380`; token signature/issuer/audience validation is absent in `internal/auth/upstream/google/google.go:89-105`.
- P0: Local-disk signed `/storage/*` proxy route appears absent from `internal/httpserver/server.go:81-190`.
- P1: `email_provider_configs` has no enabled flag; resolver treats row existence as enabled.
- P1: Mail-dependent provider gating is partial. API magic-link/invite check provider config, but hosted magic-link/reset flows do not.
- P1: Bot mitigation is not wired into hosted login/signup/magic-link surfaces. API uses `decodeWithBot`, hosted handlers bypass it.
- P1: Phase 4 fuzzer enrollment uses raw SQL, not actual Phase 4 handlers.

Appears present: Password hashing/reset service, magic-link token issuance/consume, invite issue/idempotency, TOTP core, backup codes, Google state/nonce helpers, terminal/SMTP/Resend senders, local/S3 storage interfaces, and bot verifier interface.

## Phase 5: OIDC Provider

Status: Basic OIDC endpoints exist, but tenant scoping, consent, conformance, and error semantics are incomplete.

- P0: OpenID conformance suite is placeholder-only. `.github/workflows/build.yml:82-95` pulls the image and runs `--help`, not a seeded tenant conformance run.
- P0: Token endpoint client lookup is not tenant-scoped. `internal/oidc/provider.go:121-128`, `157-164`, and `260-263` load clients by global `client_id` only.
- P0: OIDC fuzzer enrollment is synthetic. `internal/httpserver/oidc_tenant_isolation_test.go:14-65` fuzzes raw SQL count queries, not OIDC handlers.
- P0: Consent flow is incomplete. `internal/httpserver/oidc_handlers.go:147-180` records JSON consent but does not resume auth; hosted form handler does not record consent.
- P1: `/oidc/authorize` uses a `user_id` query stub rather than redirecting to `/login` with continuation token. See `internal/httpserver/oidc_handlers.go:56-90`.
- P1: OIDC error model is incomplete and may emit invalid error strings like `invalid_grant: pkce mismatch`. Required errors are missing; `invalid_pkce_method` is not in the documented list.
- P1: UserInfo does not validate `aud` because `claimsAudience` returns empty, and response lacks `name`/`picture` signed URL.
- P1: Cleanup ticker exists but is not wired into server startup.

Appears present: Discovery/JWKS endpoints and cache headers, auth-code + PKCE exchange basics, refresh-token grant basics, revoke behavior, cleanup function, ADRs 0002/0003.

## Phase 6: CLI, Backup Tooling, PATs

Status: Command names exist, but backup/import and escape-hatch guarantees are not implemented as specified.

- P0: `cypra export`/`import` are not real backups. `cmd/cypra/main.go:372-449` writes/validates a JSON count snapshot, not `pg_dump`, storage manifest/content, sealed secrets, `.tar.zst`, restore, or rewrap.
- P0: Backup round-trip tests do not verify restore semantics. `cmd/cypra/main_test.go:40-79` checks JSON and non-empty import refusal, not tenant/project/user/OIDC/signing-key/refresh/passkey restore.
- P0: `db.AsInstanceAdmin` is not implemented as specified. `internal/db/instance_admin.go:13-24` returns `*gorm.DB`, has no audit logging, no allowlist enforcement, no build tags, and no custom linter.
- P0: CLI admin commands use direct `*sql.DB` and do not emit required cross-tenant audit entries. See `cmd/cypra/main.go:196-225` and `314-369`.
- P1: PAT expiry is unsupported. Migration `db/migrations/0004_personal_access_tokens.up.sql:3-14` has no `expires_at`; auth only checks `revoked_at IS NULL`.
- P1: PAT scopes are not enforced. `internal/auth/auth.go:51-65` maps any valid PAT to tenant admin behavior.
- P1: PAT role-downgrade revocation exists as a service helper but no real downgrade hook/trigger was found.
- P1: `cypra admin invite <email>` lacks the same-instance sanity check and tests do not redeem the CLI-issued token through hosted login.
- P1: CLI integration coverage is far below every subcommand plus documented errors.

Appears present: CLI dispatch and command names, passphrase-required export exit code, non-empty import refusal, reset-bootstrap delegation, basic PAT create/list/revoke/bearer auth, ADRs 0011/0014.

## Phase 7: Dashboard Foundation

Status: SPA foundation exists, but the setup/account/gallery requirements are partially stubbed.

- P0: Setup Wizard is client-stubbed, not end-to-end. `dashboard/src/screens.tsx:178-254` advances local state and does not verify/redeem token, perform WebAuthn, mint first admin, or redirect through React state.
- P1: Not every DESIGN section 8 primitive is implemented. Missing or incomplete examples include Combobox, Toolbar, Dropdown/Menu; `Popover` is a basic `details` wrapper.
- P1: Gallery does not render every primitive in every state in both modes. `dashboard/src/components.tsx:1361-1510` shows a subset.
- P1: `BackupCodeGrid` has `beforeunload` but lacks browser-back interception/route blocking.
- P1: Account/Profile is hardcoded and not data-backed for real passkeys/2FA/sessions.
- P1: Theme persistence is frontend-hardcoded to instance actor in `dashboard/src/theme.tsx:35-64`.
- P1: Dashboard Overview uses hardcoded metrics/audit rows rather than tile-level React Query/error/permission states.
- P2: API-version mismatch boot version is hardcoded to `dev`.
- P2: Vitest coverage is not every primitive/state/variant.

Appears present: React/Vite/Bun scaffold, React Query config, token CSS, self-hosted fonts, hardcoded hex lint, Lucide import ban, embed.FS integration, dev axe hook, ADR-0010.

## Phase 8: Hosted Login and Tenant Theming

Status: Templates exist, but critical auth flows are not real end-to-end flows.

- P0: Hosted OIDC consent UI does not record consent. `internal/hostedlogin/templates/consent.html:4-19` posts only `decision`; `hostedConsentDecision` returns HTMX copy and never writes `oidc_consents`.
- P0: Passkey ceremony is not end-to-end. `internal/hostedlogin/static/passkey.js:1-17` creates random client-side challenges and never talks to server challenge/options endpoints; server WebAuthn is also stub-level.
- P1: Hosted login has no Google upstream entry point in `internal/hostedlogin/templates/login.html:7-29`.
- P1: 2FA page is a placeholder. `internal/httpserver/hostedlogin_handlers.go:121-128` always accepts a factor string/code.
- P1: Tenant accent validation checks white `text-on-accent`, but dark mode changes `--text-on-accent`; dark contrast is not validated.
- P1: Hosted-login integration tests do not cover every flow/state.
- P2: Email templates are Go string construction in `internal/email/templates.go`, not files under `internal/email/templates/`.
- P2: `/reset` POST is wired to `hostedMagicLink`, not a password reset flow.
- P2: Rate-limited countdown UI was not found.
- P2: Visual validation handoff covers only a small subset and uses HTTP rather than HTTPS/portless.

Appears present: Hosted-login template files, base layout branding, Powered by Cypra toggle, border-focus invariance, accent API rejection, uniform password failure copy, Caddy reference, ADR-0008.

## Phase 9: Dashboard Domain UI

Status: Many routes render, but large parts are demo/static or not wired to backend APIs.

- P0: Several Phase 9 surfaces silently fall back to demo data. `dashboard/src/screens.tsx:76-176` defines demo data; multiple screens use fallback data when API data is absent.
- P0: PAT tab likely fails end-to-end. Frontend calls `/api/v1/pats/` with role headers in `dashboard/src/api.ts:135-163`; backend requires `X-Cypra-User-Id` in `internal/httpserver/pat_handlers.go:63-74`.
- P0: Auth methods tab is UI-only. Cards/counts are hardcoded in `dashboard/src/screens.tsx:1113-1121`; switches do not persist to `tenant_auth_methods`; no backend endpoint was found.
- P0: Signing keys screen is static and lacks real list/rotate API wiring.
- P0: Audit log viewer renders hardcoded entries and does not poll real audit entries; backend only exposes export.
- P1: Project detail edit/rotate/delete are local UI only; backend projects only list/create.
- P1: User detail actions are confirmation-only and user detail tabs are placeholders.
- P1: User list has an invite modal that only closes instead of using the real invite helper.
- P1: Members & Roles is hardcoded and non-persistent; revoke pending invite only closes a modal.
- P1: Tests are mostly route render/axe checks, not dashboard-to-API integration for every sub-track.
- P2: Phase 14.5 explicitly acknowledges Phase 9 was marked complete based on route rendering without verifying button behavior.

Appears present: Tenant list/shell/overview/branding UI, project list/create modal, PAT reveal-once UI, basic backend tenant/project/user list/create/PAT routes.

## Phase 10: Provider Config, Instance Admins, GDPR, Observability

Status: Surfaces and routes exist, but secrets, observability, and destructive flows are incomplete.

- P0: Provider secrets are not encrypted at rest. `internal/httpserver/phase10_handlers.go:76` and `131` save raw config/client secrets. Email resolver expects encrypted config when KEK exists, so saved configs may fail to resolve.
- P0: Tenant Danger is mostly local UI state. `dashboard/src/screens.tsx:2248-2296` toggles React state; backend `DELETE /api/v1/tenants/{id}` immediately marks deleted and lacks suspend/resume/cancel/7-day countdown endpoints.
- P1: OpenTelemetry is incomplete. Only HTTP wrapping via `otelhttp` was found in `internal/httpserver/server.go:254-262`; no OTLP exporter init or DB/email/storage/OAuth child spans were found.
- P1: `/metrics` emits static zeros for most metrics in `internal/httpserver/phase10_handlers.go:327-351`; only DB pool stats and bot counters appear dynamic.
- P1: Instance Admins dashboard invite modal only closes; no API call.
- P1: Instance Diagnostics uses hardcoded migration state and email health.
- P1: Bot mitigation has counters but no audit-log wiring.
- P1: GDPR user delete scrubs user/audit rows and one profile-picture object, but not broader per-user storage ownership.
- P1: Tenant-delete cascade is partial and lacks the background hard-delete job.
- P2: `promtool` validation, docs preview, and full visual evidence are unverified.

Appears present: Phase 10 route registration, provider screens, read-only storage diagnostic UI, instance-admin list/demote backend with last-admin guard, DSR export/delete endpoints, bot verifier counters, playbook docs, some tests.

## Phase 11: Go SDK and Examples

Status: SDK skeletons compile, but e2e and server alignment are incomplete.

- P0: Phase 11 e2e acceptance is not gated in CI. `tests/e2e/examples-smoke.spec.ts:13-17` skips unless `CYPRA_E2E_LIVE` is set; `.github/workflows/build.yml:73-77` does not set it or boot a stack.
- P0: Playwright harness does not boot Cypra/Postgres/MailHog/Google stub. `tests/e2e/harness.ts:12-67` mostly launches a browser and navigates.
- P0: Admin SDK calls endpoints not exposed by the server. `sdk/go/admin/client.go:174-187` calls `/api/v1/members/`, `/api/v1/oidc-clients/`, and `/api/v1/signing-keys/`; `internal/httpserver/server.go:112-188` does not register them.
- P1: Admin SDK CRUD coverage is incomplete for projects/users/OIDC clients/signing keys/members.
- P1: Instance-admin SDK helpers with PAT auth likely 403 because bearer auth overwrites instance-admin actor state.
- P1: Go server example lacks callback/exchange/session middleware; it verifies bearer tokens but is not a full consumer sign-in flow.
- P1: Typed errors are implemented/tested for admin HTTP responses, not OIDC wrapper errors.
- P1: Next.js real sign-in remains unverified because e2e skips without live stack.
- P2: Public Go proxy `go get` acceptance is unverified.

Appears present: `sdk/go/oidc` wrapper, SDK module/READMEs, basic admin tenant/project/user methods, Next.js example skeleton, Go server example compiling, SDK release workflow, local SDK tag.

## Phase 12: Onboarding and Canonical Demo

Status: Local docs and dry-run exist, but canonical end-to-end acceptance is not met.

- P0: Phase is marked complete while handoff explicitly defers Railway, human stopwatch, and live checks.
- P0: Canonical-demo e2e does not exercise the required story. `tests/e2e/canonical-demo/canonical-demo.spec.ts:13-35` uses helper navigation stubs and checks title/time only.
- P0: CI only validates dry-run/skip behavior. Live test skips unless `CYPRA_E2E_LIVE` is set; CI does not set it.
- P0: No required Compose stack with MailHog, Google stub, and Next.js app was found. `deploy/docker-compose.yml` contains only Cypra/Postgres/Caddy.
- P0: Railway template is explicitly future/deferred. `docs/deploy/railway-template.md:3-12` says the separate repo was not created or verified.
- P1: Timing budget has one total elapsed assertion, not per-step duration logging.
- P1: 30-minute human stopwatch run is unverified and not recorded.
- P1: First-run docs are sparse, lack screenshots, and do not include live Resend/Google setup evidence.
- P1: Cross-flow accent visual check is incomplete/unverified.

Appears present: Canonical-demo spec file, first-run doc, Next.js tutorial/example, Setup Wizard CTA/env stanza and test.

## Phase 13: Polish, A11y, Performance

Status: Good UI polish scaffolding exists, but key acceptance gates are unverified or not real.

- P0: Cold-start criterion is explicitly deferred in `TASKS.md:877` despite being checked at `TASKS.md:850` and `864`.
- P0: Image-size gate does not measure compressed Docker image or enforce `<= 80 MB`. `Makefile:133-134` prints the local binary size only and `ci-pipeline` does not run it.
- P0: PLAN section 13 p99 performance targets are not implemented as CI synthetic/profile-matched assertions. No p99 benchmark for `/oidc/token`, `/login/passkey/verify`, or `/api/v1/users` was found.
- P1: Lighthouse gate is synthetic Playwright + axe timing, not real Lighthouse/LHCI. See `scripts/check-lighthouse-baseline.mjs` and `tests/e2e/lighthouse-baseline.json`.
- P1: Visual validation scope appears much narrower than required; handoff mentions only `/setup/cypra_setup_test` and `/dashboard/tenants?state=demo`.
- P1: axe coverage is partial; no evidence of axe across every hosted-login route.
- P1: Tap target relaxation is not structurally enforced. Small icon buttons are 28px in `dashboard/src/components.tsx` and used outside list rows.
- P1: Reduced-motion global rule exists, but active transform scale remains on buttons.
- P2: Hosted login has a hardcoded `#DC2626` error color outside dashboard color lint coverage.
- P2: Keyboard-only canonical demo is unverified because live canonical test skips and helpers navigate directly.

Appears present: Coverage floor script and CI wiring, responsive sidebar/table CSS, `MobileBlockedBanner`, static skeletons, shortcut overlay alternative reachability, `IdentifierPill` full aria label, border-focus system ownership.

## Recommended Patch Order

1. Restore the CI gate first by fixing current frontend lint errors. This gives a reliable baseline before functional patches.
2. Seal tenant isolation bypasses: remove or restrict raw `*gorm.DB` access, wire the tenant plugin/RLS setter in production, and replace direct SQL in tenant-scoped handlers.
3. Replace synthetic fuzzer enrollment with handler-level tests for every phase-introduced route.
4. Make auth/OIDC real before UI polish: session issuance, real WebAuthn, Google upstream validation, invite continuation, consent resume, tenant-scoped OIDC client lookup, and rate limiter wiring.
5. Rebuild export/import to be a restoreable backup with storage and encrypted-secret rewrap, or uncheck/defer the Phase 6 acceptance honestly.
6. Convert dashboard demo fallbacks into explicit demo states and wire real APIs for auth methods, signing keys, audit listing, members, user actions, PAT headers, and instance admin invite.
7. Replace skipped/dry-run e2e with a real dockerized stack or downgrade checked acceptance boxes to `unverified` until live dependencies run.
8. Make Phase 13 gates measurable in CI: compressed Docker image size, cold-start stopwatch, profile-matched p99 checks, real Lighthouse/LHCI, full axe/browser matrix.

## Phase 18 Closure Appendix

This appendix records Phase 18 remediation evidence for findings that have been closed so far. Items not listed here remain open in `TASKS.md` Phase 18.

| Original finding                                                                      | Closure evidence                                                                                                                                                                                                                                                                                                                               |
| ------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Cross-phase P0: CI gate is red.                                                       | `./bin/agent-ci run --quiet --all` is green. Latest successful run covered install, dashboard build, Go build, `golangci-lint`, color lint, Prettier check, `go vet`, dashboard typecheck, `go test -p 1 ./...`, coverage floors, dashboard Vitest, p99 performance, compressed image-size, and cold-start gates.                              |
| Cross-phase P0/P1: tenant isolation bypasses and synthetic fuzzer coverage.           | Phase 15 constrained audited instance-admin DB access, wired tenant-aware HTTP DB access, restored production tenant plugin/RLS enforcement, expanded RLS/model/fuzzer coverage, and added audit-write lint enforcement. Evidence: Phase 15 handoff in `TASKS.md`; CI fuzzer/tests pass in `./bin/agent-ci run --quiet --all`.                 |
| Phase 1 P0/P1: `TenantScopedDB`/RLS production enforcement holes.                     | Phase 15 closed the production GORM tenant plugin and HTTP tenant DB access gaps, constrained `db.AsInstanceAdmin`, and restored local CI. Evidence: `internal/db`, `internal/httpserver`, custom lint/allowlist checks, and passing `go test ./internal/db ./internal/db/fuzz ./internal/httpserver` via full CI.                             |
| Phase 2 P0/P1: setup-token, master-key rotation, signing-key worker, refresh metrics. | Setup token redemption now backs first-admin creation and setup UI; master-key rotation uses real encrypted shapes; signing/email workers start with server boot; refresh reuse metrics are exposed. Evidence: `internal/bootstrap` coverage is 86.2%, `cmd/cypra`/`internal/crypto` tests pass, and Phase 15 handoff records the remediation. |
| Phase 3 P0/P1: rate limiter, audit export auth, structural audit, server workers.     | Auth and mutating routes are rate-limited/audited, tenant-admin audit export is permission-guarded, direct audit writes are linted, and workers start from `serve`. Evidence: `internal/httpserver/server_test.go`, `internal/ratelimit`, audit lint target in `Makefile`, and full CI.                                                        |
| Phase 4 P0/P1: WebAuthn/session/Google/invite/storage/provider gating gaps.           | WebAuthn uses server-generated go-webauthn ceremonies; hosted password, magic-link, passkey, Google, invite, and 2FA flows establish sessions; local storage proxy and provider enabled-state gates are implemented. Evidence: `tests/e2e/examples-smoke.spec.ts` passed with `CYPRA_E2E_MANAGED=1`; relevant Go auth/storage tests pass.      |
| Phase 5 P0/P1: OIDC conformance placeholder, tenant-scoped clients, consent flow.     | OIDC client lookup is tenant-scoped, hosted/JSON consent records decisions and resumes authorization, cleanup runs from server startup, userinfo validates claims, and conformance is seeded rather than `--help`. Evidence: `internal/httpserver/oidc_handlers_test.go`, `tests/openidconformance`, and `.github/workflows/build.yml`.        |
| Phase 6 P0/P1: backup/import, audited CLI escape hatch, PAT expiry/scopes.            | `cypra export`/`import` use a passphrase-encrypted restoreable backup envelope with storage and secret rewrap; CLI admin commands use the audited instance-admin path; PAT expiry/scopes/revocation are enforced. Evidence: `cmd/cypra/backup.go`, `cmd/cypra/*_test.go`, `internal/pat`, and managed `tests/e2e/backup-import.spec.ts`.       |
| Phase 7 P0/P1: Setup Wizard, gallery coverage, backup-code route blocking.            | Setup Wizard verifies setup token, enrolls a real WebAuthn passkey, shows one-time backup codes, mints the first instance admin, establishes a session, and navigates to dashboard; gallery/state coverage and backup-code interception are implemented. Evidence: managed `redeemBootstrapToken` e2e path and `dashboard/src/App.test.tsx`.   |
| Phase 8 P0/P1: hosted consent/passkey/Google/2FA/reset/rate-limit/theming gaps.       | Hosted consent records and resumes OIDC, passkey JS uses server options, Google upstream appears when configured, 2FA verifies TOTP/WebAuthn/backup codes, reset uses password-reset flow, and dark-mode accent/link/error contrast is validated. Evidence: hosted Go tests, e2e examples smoke, and `dogfood-output/phase16-hosted-*.json`.   |
| Phase 9 P0/P1: dashboard demo/static data and missing real API persistence.           | Dashboard demo data is explicit demo/gallery state only; project, user, member, auth-method, PAT, signing-key, audit, invite, provider, diagnostics, and danger flows are wired to real APIs with loading/empty/error states. Evidence: `dashboard/src/App.test.tsx` demo-data guards/API call assertions and full Phase 17 visual matrix.     |
| Phase 10 P0/P1: provider encryption, Tenant Danger, OTEL/metrics, GDPR cascade.       | Provider/upstream secrets are envelope-encrypted and decrypted by diagnostics; Tenant Danger supports schedule/cancel/sunset cleanup; metrics expose runtime values; OTEL initializes and child spans are emitted; GDPR user/tenant delete handles audit/storage side effects. Evidence: `internal/httpserver/phase10_handlers_test.go`.       |
| Phase 11 P0: examples e2e skips in CI without a live stack.                           | `.github/workflows/build.yml` now runs `tests/e2e/examples-smoke.spec.ts` with `CYPRA_E2E_MANAGED=1`; local evidence: `CYPRA_E2E_MANAGED=1 bun run test:e2e -- tests/e2e/examples-smoke.spec.ts tests/e2e/canonical-demo/canonical-demo.spec.ts` passed. The managed harness now starts Postgres, Cypra, SMTP stub, Google stub, and Next.js.  |
| Phase 11 P1: Go server example lacks callback/exchange/session middleware.            | `examples/go-server/main.go` now implements login redirect, callback code exchange, ID-token verification, cookie sessions, logout, and reusable `requireAuth`; `go test ./...` passes in `examples/go-server`.                                                                                                                                |
| Phase 11 P1: Next.js real sign-in remains unverified.                                 | `examples/nextjs` maps Cypra claims into Auth.js session; `tests/e2e/examples-smoke.spec.ts` redeems a real invite, signs in through Auth.js/Cypra consent, and asserts `session.cypra.sub` and `session.cypra.email`.                                                                                                                         |
| Phase 6 P0: backup/import tests do not verify restored passkey semantics.             | `cmd/cypra/main_test.go` verifies encrypted backup/import restores tenants, projects, users, OIDC clients, signing keys, TOTP secrets, refresh-token family semantics, and local storage; `tests/e2e/backup-import.spec.ts` exports after browser WebAuthn registration, imports into a fresh managed stack, and asserts the restored passkey. |
| Phase 4/8 P0: WebAuthn and hosted passkey ceremonies were not real.                   | `internal/auth/webauthn` uses server-stored go-webauthn ceremonies with request-origin validation; `tests/e2e/examples-smoke.spec.ts` and the canonical demo now exercise browser-generated primary passkey registration/assertion plus WebAuthn second-factor enrollment/verification through the hosted helper.                              |
| Phase 12 P0/P1: canonical demo is dry-run/navigation-only and lacks per-step timings. | `tests/e2e/canonical-demo/canonical-demo.spec.ts` now performs bootstrap, tenant/project/provider config, downstream Next.js sign-in, Cypra consent, session-claim assertions, hosted passkey registration/assertion, Google stub callback/session verification, per-step timing logs, and an 8-minute budget assertion.                       |
| Phase 12 P0: CI only validates dry-run/skip behavior.                                 | `.github/workflows/build.yml` runs the canonical demo with `CYPRA_E2E_MANAGED=1`; local managed canonical run passed in ~14-18 seconds.                                                                                                                                                                                                        |
| Phase 12 P1: first-run docs are sparse and lack provider setup commands.              | `docs/firstrun.md` and `docs/tutorials/nextjs.md` now document local/deployed commands, Resend free-tier setup, Google Cloud Console setup, callback registration, and browser screenshot checkpoints.                                                                                                                                         |
| Phase 13 P0: image-size gate measures only the binary.                                | `make image-size` now builds the production Docker image, measures the compressed `docker image save` output, enforces `<= 80 MB`, and is included through `make cold-start` in `make ci-pipeline`; local evidence: `cypra:image-size compressed: 9474521 bytes`.                                                                              |
| Phase 13 P0: cold-start criterion was deferred.                                       | `scripts/check-cold-start.mjs` starts Compose Postgres, migrates with the production image, runs `docker run`, polls `/readyz`, and enforces `<= 3 s`; local evidence: `cold-start readyz: 154ms`.                                                                                                                                             |
| Phase 13 P1: Lighthouse gate was synthetic.                                           | `scripts/check-lighthouse-baseline.mjs` now uses real Lighthouse with Chrome against hosted setup and dashboard demo routes; local evidence: `Lighthouse baseline passed for 2 routes.`                                                                                                                                                        |
| Phase 13 P2: hosted-login hardcoded color is outside lint coverage.                   | `scripts/check-no-hardcoded-colors.mjs` now scans `internal/hostedlogin` HTML templates and only allows hex values in token source files.                                                                                                                                                                                                      |
| Phase 13 P0: p99 performance targets were not CI assertions.                          | `internal/httpserver/performance_test.go` contains Phase 18 p99 assertions for `/oidc/token`, `/login/passkey/verify`, and `/api/v1/users?limit=50&page=1`; `make performance` is wired into `make ci-pipeline`; local evidence: `CYPRA_PERF_GATE=1 go test -count=1 -run TestPhase18P99PerformanceAssertions ./internal/httpserver` passed.   |
| Phase 13 P1: axe coverage was partial.                                                | `tests/e2e/accessibility.spec.ts` runs browser axe across authenticated dashboard routes and hosted-login routes with `CYPRA_E2E_MANAGED=1`; `.github/workflows/build.yml` runs the accessibility matrix; local evidence: `CYPRA_E2E_MANAGED=1 bunx playwright test tests/e2e/accessibility.spec.ts` passed.                                   |
| Phase 13 P1: tap target relaxation was not structurally enforced.                     | `dashboard/src/tokens.css` enforces coarse-pointer `min-width`/`min-height` via tokens with an explicit `data-touch-target="cursor-only"` opt-out; `dashboard/src/App.test.tsx` asserts the policy.                                                                                                                                            |
| Phase 13 P1: reduced-motion transforms remained active.                               | `dashboard/src/index.css` disables transforms under `prefers-reduced-motion: reduce`; `dashboard/src/App.test.tsx` asserts animation duration, transition duration, and transform reset.                                                                                                                                                       |

The appendix is intentionally evidence-only. Remaining P0/P1 findings that still depend on external deployment/release proof or a human stopwatch run remain tracked in `TASKS.md` Phase 18 and are not marked closed here yet.
