# Cypra — External Tester Readiness Plan

**Archived prior plan:** [`docs/archive/PLAN-2026-05-09-v1-product-plan.md`](./docs/archive/PLAN-2026-05-09-v1-product-plan.md)  
**Companion docs:** [`DESIGN.md`](./DESIGN.md), [`BRAINSTORM.md`](./BRAINSTORM.md), [`TASKS.md`](./TASKS.md), [`docs/phase-1-13-validation.md`](./docs/phase-1-13-validation.md)  
**Status:** draft  
**Version:** 2.0-readiness  
**Owner:** @watzon  
**Created:** 2026-05-09

---

## 1. Executive Summary

Cypra is locally strong: the full local CI pipeline is green, the dashboard has a coherent visual system, managed e2e smoke tests exist, and the core self-hosted auth product is substantial. It is not yet safe or polished enough for outside testers to install on a VPS.

This plan replaces the original broad v1 product plan with a focused remediation roadmap. The goal is to move Cypra from "excellent development build" to "credible initial external tester deployment" by fixing the security blockers, hardening the deployment path, removing development-mode UX tells, and proving the entire flow on a fresh VPS.

The target release is not general availability. It is a controlled external test release for technical users who understand they are using early software, but who should still receive a safe, honest, repeatable install experience.

---

## 2. Current Readiness Snapshot

### Verified strengths

- `./bin/agent-ci run --quiet --all` passes locally.
- Coverage floors pass for `internal/crypto`, `internal/oidc`, `internal/auth`, `internal/db`, `internal/sessions`, and `internal/bootstrap`.
- Managed e2e, accessibility, performance, image-size, and cold-start gates exist in CI.
- The Dockerfile is compact and production-oriented: multi-stage build, distroless nonroot runtime, migrations included.
- `DESIGN.md` defines a distinctive, coherent product direction: calm dense operator dashboard, generous hosted login, all-Monaspace typography, dark/light parity.
- The local first-run path and canonical demo have a documented shape.

### Release blockers

- Public auth factor-management endpoints can mutate factors for arbitrary `user_id` without sufficient session/ownership authorization.
- OIDC JSON consent can record consent without requiring a signed continuation plus current user session.
- The reference deployment uses a mutable `:dev` image and exposes services that should stay internal.
- The release workflow cannot publish images or GitHub Releases.
- The deployed smoke workflow still contains placeholders for critical operational checks.
- Several dashboard/hosted-login surfaces still show no-op behavior, placeholder copy, or prototype-grade interactions.

---

## 3. Target Outcome

At the end of this plan, a technically competent external tester can:

1. Provision a small VPS with Docker Compose and DNS for an install domain plus wildcard tenant subdomains.
2. Deploy a versioned Cypra image behind Caddy without exposing Postgres or Cypra directly.
3. Redeem the setup token, enroll a passkey, save backup codes, create a tenant, create a project, configure providers, and connect the Next.js example.
4. Understand exactly which features are ready, which are beta, and which are deliberately deferred.
5. Run documented smoke checks for login, OIDC, metrics, backup/export/import, and recovery.
6. Recover from common failures using operator docs rather than reading source code.

Security target: no known P0/P1 auth, tenant-isolation, setup, OIDC, or deployment exposure issues remain open without explicit owner-approved deferral.

UX target: no end-user or operator path should contain no-op buttons, placeholder copy, externally loaded critical UI dependencies, or avoidable keyboard/accessibility traps.

---

## 4. Non-Goals For This Readiness Push

- No SAML, embeddable widget, TypeScript SDK, SMS, or tenant CNAME support.
- No Kubernetes, Helm, multi-region, or managed SaaS deployment path.
- No full mobile dashboard rewrite. Mobile should not be broken, but the tester target remains technical operators on desktop.
- No compliance certification claims. Cypra may provide GDPR-enabling tools and playbooks only.
- No broad rebrand or redesign. Preserve `DESIGN.md`; fix rough edges and production polish.
- No large architecture rewrite unless required to close a blocker.

---

## 5. Severity Model

- **P0 Blocking:** unsafe for external testers or prevents deployment/release proof.
- **P1 Major:** significant correctness, security, operability, accessibility, or UX issue that should be fixed before the first tester wave.
- **P2 Minor:** visible rough edge or missing robustness that can ship to a small trusted tester group if documented.
- **P3 Polish:** nice-to-have refinement after external tester readiness.

Every phase below must end with local CI green. Deployment-facing phases must also produce recorded evidence from a fresh environment.

---

## 6. Phase Roadmap

### Phase 0: Re-baseline And Taskify

**Purpose:** Convert this plan into executable `TASKS.md` phases without losing the existing remediation history.

**Scope:**
- Archive or clearly supersede stale task sections that imply v0.1 is release-ready.
- Add a new readiness track to `TASKS.md` that mirrors the phases in this plan.
- Link each task back to a specific finding, file, or acceptance criterion.
- Keep the current product PRD archived, not deleted.

**Acceptance:**
- `TASKS.md` has a new readiness section with unchecked phases for this plan.
- Old phase records remain historically accessible but no longer imply release readiness.
- `README.md` points readers to the archived product plan and the current readiness plan accurately.
- Local CI is green.

---

### Phase 1: Security Hardening Gate

**Purpose:** Close all known security blockers before any public VPS exposure.

**Must fix:**
- Require authenticated session ownership or explicit audited admin permission for passkey registration, TOTP enrollment, WebAuthn 2FA enrollment, backup-code regeneration, session listing/revocation, passkey deletion, and MFA factor listing.
- Restrict OIDC consent recording to signed continuation plus current user session. Remove or lock down the raw JSON path that accepts arbitrary `user_id`.
- Replace client-controlled trusted-proxy signaling with server-side proxy trust configuration only.
- Ensure forwarded host/proto/IP are honored only when the deployment has explicitly enabled trusted proxy mode.
- Issue `Secure` session cookies correctly behind TLS-terminating proxies.
- Remove PAT subject fallbacks that trust `X-Cypra-User-Id` outside dev-only mode, or make them explicit audited impersonation paths.
- Use constant-time comparison for OIDC client secrets.
- Make master-key parsing strict enough to reject weak or mistyped production keys rather than silently hashing arbitrary strings.
- Decide whether terminal email is allowed in non-dev production. If allowed, surface a warning; if not, block it.

**Primary files likely touched:**
- `internal/httpserver/auth_handlers.go`
- `internal/httpserver/oidc_handlers.go`
- `internal/httpserver/server.go`
- `internal/httpserver/session_helpers.go`
- `internal/httpserver/pat_handlers.go`
- `internal/oidc/provider.go`
- `cmd/cypra/main.go`
- `internal/email/`

**Acceptance:**
- New tests prove an unauthenticated caller cannot enroll, regenerate, list, delete, or revoke resources for another user by supplying `user_id`.
- New tests prove OIDC consent cannot be recorded without a valid continuation and logged-in user.
- New tests prove spoofed `X-Forwarded-*` and `X-Cypra-Trusted-Proxy` headers do not affect tenant resolution, rate-limit IP, cookie security, or issuer/origin unless trusted proxy mode is explicitly configured.
- New tests cover strict master-key rejection.
- `./bin/agent-ci run --quiet --all` passes.

---

### Phase 2: Deployment Packaging And Network Safety

**Purpose:** Make the reference VPS deployment safe to copy, versioned, and aligned with the docs.

**Must fix:**
- Replace `ghcr.io/watzon/cypra:dev` in production-facing deployment docs with a versioned tag or documented local-build override.
- Split development compose defaults from production compose defaults where needed.
- Stop publishing Postgres to the host in the production recipe.
- Stop publishing Cypra directly to the host in the TLS profile; Caddy should be the public entrypoint.
- Expose both `80` and `443` for Caddy in the production TLS profile.
- Replace `tls internal` local-only Caddy config with a production-ready template that uses real install and wildcard tenant domains.
- Align `TRUSTED_PROXY_HEADERS` defaults with the Caddy deployment path.
- Add a real container healthcheck that checks `/readyz`, not only `cypra version`.
- Provide a production env template that refuses or loudly marks dev defaults.

**Primary files likely touched:**
- `deploy/docker-compose.yml`
- `deploy/Caddyfile.example`
- `deploy/README.md`
- `docs/deploy/vps.md`
- `.env.example` or a new `.env.production.example`
- `Dockerfile`

**Acceptance:**
- A fresh VPS compose deployment exposes only Caddy publicly.
- Postgres is reachable only inside the compose network unless an explicit development override is used.
- Caddy obtains or is configured for real public TLS with documented DNS requirements.
- Container health reflects DB/migration/storage/master-key readiness.
- Docs clearly distinguish local development from production deployment.
- `./bin/agent-ci run --quiet --all` passes.

---

### Phase 3: Release Automation And Smoke Evidence

**Purpose:** Make `v0.1.0` publishable and prove published artifacts work outside the local checkout.

**Must fix:**
- Implement the server release workflow for semver tags.
- Publish multi-arch GHCR images for `linux/amd64` and `linux/arm64`.
- Attach release binaries and checksums to GitHub Releases.
- Ensure workflow permissions are sufficient and minimal.
- Align SDK smoke versions with the actual release/tag strategy.
- Replace deployed-smoke placeholders with real export/import and multi-instance-admin recovery smokes.
- Avoid single-use setup-token secrets in scheduled smoke design; use a reset/reseed strategy or disposable environment.

**Primary files likely touched:**
- `.github/workflows/release.yml`
- `.github/workflows/smoke.yml`
- `docs/contributing/release.md`
- `CHANGELOG.md`
- release scripts under `scripts/` if needed

**Acceptance:**
- A dry-run or test tag builds the expected release artifacts without manual steps.
- GHCR image publication is verified for amd64 and arm64.
- GitHub Release artifact upload is verified.
- Deployed smoke runs canonical demo, SDK compile/use, export/import round-trip, and multi-instance-admin recovery against a fresh instance.
- Smoke failures open an actionable issue or upload actionable artifacts.
- `./bin/agent-ci run --quiet --all` passes.

---

### Phase 4: Operator Documentation And Recovery Playbooks

**Purpose:** Make a tester successful without reading source code or asking for private guidance.

**Must fix:**
- Expand VPS docs from first boot into full day-one operation.
- Document DNS, TLS, reverse proxy, trusted proxy mode, env vars, secrets, storage, and database roles.
- Add backup cadence, restore drill, export/import expectations, storage backup expectations, and master-key handling.
- Add upgrade and rollback procedure.
- Add certificate renewal checks.
- Add first-24-hours monitoring checklist.
- Add smoke-test checklist for a deployed instance.
- Make known limitations explicit and non-alarming.

**Primary files likely touched:**
- `docs/deploy/vps.md`
- `deploy/README.md`
- `docs/deploy/observability.md`
- `docs/playbook/README.md`
- `docs/playbook/monitoring.md`
- `docs/firstrun.md`
- `CHANGELOG.md`

**Acceptance:**
- A tester can deploy from docs alone on a new VPS.
- Docs include exact commands for health, readiness, metrics, logs, backup, restore, upgrade, rollback, and smoke testing.
- Docs state what not to expose publicly.
- Docs state which features are beta/deferred.
- A human dry-run records any missing steps as follow-up tasks.
- `./bin/agent-ci run --quiet --all` passes.

---

### Phase 5: Dashboard UX Completion

**Purpose:** Remove in-product no-ops, placeholders, and accessibility traps from operator workflows.

**Must fix:**
- Wire or remove the no-op tenant user invite modal.
- Remove placeholder copy such as "Screencast placeholder lands in v1.1 docs."
- Replace dev-looking fallback project values with explicit not-configured or loading/error states.
- Ensure project, user, signing-key, and instance-admin tables use responsive behavior comparable to tenant tables.
- Add modal focus trap, focus restoration, inert background behavior, and reliable Escape handling.
- Review drawers and overlays for the same keyboard behavior.
- Fix fixed-width command/shortcut overlays for narrow screens.
- Ensure every destructive action has accurate copy and real backend persistence or is hidden.

**Primary files likely touched:**
- `dashboard/src/screens.tsx`
- `dashboard/src/components.tsx`
- `dashboard/src/modals.tsx`
- `dashboard/src/api.ts`
- `dashboard/src/App.test.tsx`

**Acceptance:**
- No primary-action dashboard button silently closes a modal without doing the advertised work.
- No placeholder copy is visible in production routes.
- Modal and drawer behavior passes keyboard-only tests.
- Responsive snapshots or browser validation cover dashboard tables and overlays.
- Dashboard tests assert no demo/fallback data appears outside explicit demo routes.
- `./bin/agent-ci run --quiet --all` passes.

---

### Phase 6: Hosted Login And End-User Polish

**Purpose:** Make the hosted login surface feel production-owned rather than prototype-owned.

**Must fix:**
- Vendor or bundle HTMX locally, or add pinning/SRI and a clear dependency policy.
- Remove production-visible `noop` bot-verifier markup or replace it with neutral state.
- Improve passkey error messages for cancellation, unsupported browser, network failure, invalid RP ID/origin, and server rejection.
- Split 2FA UI states so TOTP, WebAuthn, and backup-code flows do not share a confusing single-code interface.
- Clarify invite onboarding copy around required vs optional auth methods.
- Verify hosted-login dark/light mode and tenant accent contrast after the security and deployment changes.
- Confirm all hosted-login routes work without external CDN availability.

**Primary files likely touched:**
- `internal/hostedlogin/templates/*.html`
- `internal/hostedlogin/static/passkey.js`
- `internal/httpserver/hostedlogin_handlers.go`
- `internal/hostedlogin/*_test.go`
- `tests/e2e/accessibility.spec.ts`

**Acceptance:**
- Hosted login loads all critical assets from Cypra or with approved integrity guarantees.
- Passkey and 2FA errors are actionable without leaking sensitive details.
- Browser validation covers login, signup, invite, consent, reset, 2FA, and error states.
- axe reports zero new violations on hosted-login routes.
- `./bin/agent-ci run --quiet --all` passes.

---

### Phase 7: Fresh VPS Release Candidate Trial

**Purpose:** Prove the system in the environment external testers will use.

**Must do:**
- Deploy the latest release candidate to a clean VPS using only public docs.
- Use real DNS and TLS for install and wildcard tenant hosts.
- Redeem setup, create first admin, create tenant, create project, configure providers, and connect the Next.js example.
- Run the deployed smoke workflow.
- Run export/import restore into a fresh second instance.
- Run multi-instance-admin recovery.
- Verify `/metrics` and alert rule scrape shape.
- Verify an OTEL trace captures OIDC authorize through login/consent/token and audit emission.
- Record human stopwatch timing for first run.
- Capture visual/a11y observations for dashboard and hosted-login routes.

**Acceptance:**
- A clean VPS install reaches working Next.js sign-in without source-code-only instructions.
- Human first-run time is recorded and is at or below the 30-minute target, or streamlining tasks are added before release.
- Deployed smoke, SDK smoke, export/import, recovery, metrics, and tracing evidence are recorded.
- Any failure becomes a P0/P1 task before tester rollout.
- `v0.1.0` or the selected tester tag exists and matches the deployed artifact.

---

## 7. Cross-Phase Gates

Every implementation phase must satisfy:

- **CI gate:** `./bin/agent-ci run --quiet --all` passes.
- **Hygiene gate:** lint, format, typecheck, Go tests, dashboard tests, and coverage floors remain green.
- **Security regression gate:** new tests cover every closed P0/P1 security finding.
- **Docs gate:** any deployment, release, or operator behavior change updates docs in the same phase.
- **Visual validation gate:** any dashboard or hosted-login change gets browser validation for the affected states.
- **No silent deferrals:** any P0/P1 not fixed must be explicitly listed with owner approval and a reason it is safe for external testers.

---

## 8. Initial Finding Backlog

### P0

- Public auth factor-management endpoints need ownership/admin authorization.
- OIDC consent recording must require signed continuation and current user session.
- Release workflow must publish real artifacts.
- Production compose must not use `:dev` image or expose internal services.

### P1

- Trusted proxy handling must not be controlled by request headers.
- Secure cookies must work behind TLS termination.
- Caddy and compose TLS profile must be production-ready.
- Deployed smoke placeholders must become real checks.
- SDK smoke versions must match the release strategy.
- VPS docs need backup, restore, upgrade, rollback, observability, and smoke steps.
- Dashboard no-op invite and placeholder copy must be removed.
- Modal focus trap and responsive overlay issues must be fixed.
- Hosted login must not depend on an unverified CDN script.
- Passkey and 2FA error/flow copy needs production polish.

### P2

- Production-visible `noop` bot verifier marker should be removed or neutralized.
- Terminal email should be blocked or loudly warned in production.
- OpenAPI dev gating should not be coupled only to `LOG_LEVEL=debug`.
- Mobile dashboard messaging should feel intentional, not apologetic.
- Observability docs should include Prometheus/OTEL examples.

---

## 9. Release Criteria For External Testers

Cypra is ready for the first external tester wave only when:

- All Phase 1 P0/P1 security issues are fixed with tests.
- A versioned release image exists and can be pulled on amd64 and arm64.
- The production VPS compose path exposes only intended public services.
- Public docs are enough to deploy, operate, back up, restore, upgrade, and smoke-test.
- The dashboard has no known no-op primary actions or placeholder copy in production routes.
- Hosted login works without prototype-grade dependencies or ambiguous passkey/2FA failure states.
- A fresh VPS release candidate trial is recorded as passing.
- Any remaining P2/P3 limitations are listed honestly in `CHANGELOG.md` and docs.

---

## 10. Open Decisions

- Should terminal email be fully disallowed when `PUBLIC_BASE_URL` is non-local, or allowed with a persistent warning?
- Should production and development compose files split into separate files, or should one file use profiles/overrides?
- What tester tag should precede `v0.1.0` if we want one more private release candidate, for example `v0.1.0-rc.1`?
- Should OpenAPI remain dev-only, admin-only, or available behind a dedicated explicit env flag?
- Is mobile dashboard support required for the first tester wave, or is a clear desktop-optimized contract acceptable?
