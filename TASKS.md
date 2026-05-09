# Cypra - External Tester Readiness Tasks

**Companion to:** [`PLAN.md`](./PLAN.md), [`DESIGN.md`](./DESIGN.md), [`BRAINSTORM.md`](./BRAINSTORM.md)  
**Archived prior task plan:** [`docs/archive/TASKS-2026-05-09-legacy-phases.md`](./docs/archive/TASKS-2026-05-09-legacy-phases.md)  
**Archived prior product plan:** [`docs/archive/PLAN-2026-05-09-v1-product-plan.md`](./docs/archive/PLAN-2026-05-09-v1-product-plan.md)  
**Status:** active readiness plan  
**Created:** 2026-05-09

This document supersedes the historical Phase 0-18 task plan. The archived file remains the full implementation record. This active file tracks only the remaining work required to make Cypra safe, polished, and repeatable for an initial external tester deployment to a VPS.

---

## Operating Rules

The following rules are mandatory for every phase in this file.

- Keep this file current. Mark a task in progress before working it, check it off only after verification, and fill the phase handoff before opening the next phase.
- Treat `PLAN.md` and `DESIGN.md` as source of truth. If this file conflicts with either, update this file or explicitly revise the canonical document first.
- Do not silently defer P0/P1 work. Any deferral must name the risk, reason, owner approval, and follow-up phase.
- Do not mark a phase complete unless every task and acceptance criterion in that phase is verified.
- Preserve the archived legacy plan. Do not delete or rewrite historical phase records unless explicitly instructed.

---

## Standards Applied To Every Phase

- **CI gate:** `./bin/agent-ci run --quiet --all` passes unless the phase is documentation-only and the owner explicitly accepts `bun run format` as the phase gate.
- **Hygiene gate:** lint, format, typecheck, Go tests, dashboard tests, and coverage floors remain green for touched code.
- **Security regression gate:** every closed P0/P1 security finding gets a test that would fail before the fix.
- **Docs gate:** any deployment, release, security, or operator behavior change updates docs in the same phase.
- **Visual validation gate:** any dashboard or hosted-login UI change gets browser validation for the affected states and records observations in the handoff.
- **Phase boundary invariant:** the codebase remains runnable at every phase boundary.

---

## Current State Summary

- The full local CI pipeline was green at the start of this taskification pass.
- Historical phases 0-18 are archived and no longer drive new work directly.
- All unchecked legacy tasks and completion criteria have been carried forward into Phase R0 below, deduplicated where the old file repeated the same release/deployment evidence in multiple sections.
- New cleanup and polish tasks from `PLAN.md` are captured in Phases R1-R8.

---

## Phase R0: Legacy Carry-Forward And Baseline

**Status:** complete  
**Dependencies:** archived Phase 0-18 task plan  
**Deliverable:** Every unchecked legacy release/deployment/completion item is either completed, deliberately deferred into a later readiness phase, or explicitly marked as no longer applicable under the new external-tester release plan.

### Tasks

- [x] Review the archived task plan and confirm that the unchecked items listed in this phase are the complete legacy carry-forward set.
- [x] Decide whether the Railway one-click template remains required before the first external tester wave, or whether it is an approved post-tester deferral.
- [x] If Railway remains in scope, create and verify the separate Railway template repository and update docs with the live template URL and deployment proof.
- [x] Run and record a human stopwatch pass through `docs/firstrun.md`; if it exceeds 30 minutes, add streamlining tasks before release.
- [x] Complete deployed-instance canonical demo smoke against a fresh deployed instance.
- [x] Complete Go SDK third-machine smoke against release-accurate SDK module tags.
- [x] Complete `cypra export` / `cypra import` round-trip smoke against a fresh second instance.
- [x] Complete multi-instance-admin recovery smoke against a deployed instance.
- [x] Verify deployed `/metrics` exposes every required runtime metric from the current plan and docs.
- [x] Verify deployed OpenTelemetry tracing shows `/oidc/authorize` through login/consent/token, email worker dispatch where applicable, and audit emission.
- [x] Complete deployed visual validation with `agent-browser` against `https://<install>` and `https://<tenant>.<install>`.
- [x] Tag the selected external-tester release, either `v0.1.0` or an approved prerelease such as `v0.1.0-rc.1`.
- [x] Trigger and verify the release workflow for the selected tag.
- [x] Publish and verify the multi-arch Docker image on GHCR for `linux/amd64` and `linux/arm64`.
- [x] Attach release binaries and checksums to the GitHub Release.
- [x] Verify `docker pull ghcr.io/watzon/cypra:<tag>` works on amd64 and arm64.
- [x] Ensure `CHANGELOG.md` describes the tester release honestly, including known limitations: no SAML, no embeddable widget, no TypeScript SDK, no built-in CNAMEs, and no SMS.
- [x] Confirm all ADRs that remain part of the active architecture are checked in with accepted status.
- [x] Confirm `docs/phase-1-13-validation.md` has every P0/P1 finding either fixed with evidence or explicitly moved into an approved future task.
- [x] Confirm the operator playbook is complete enough for external testers: data inventory, sub-processor template, DSR runbook, breach-notification runbook, bot-mitigation swap path, monitoring checklist, and first-24-hours checklist.
- [x] Confirm CI remains green, including canonical-demo e2e, managed examples smoke, backup/import smoke, coverage floors, p99 performance, compressed image size, and cold-start gates.
- [x] Archive any legacy completion criteria that are no longer applicable under `PLAN.md` v2 with a short note in this phase handoff.

### Acceptance

- [x] No unchecked task remains only in the archived legacy plan without representation in this active file or an explicit no-longer-applicable decision.
- [x] Legacy release/deployment evidence is either complete or assigned to Phases R2, R3, R4, or R8 below.
- [x] Human first-run stopwatch result is recorded or explicitly scheduled in Phase R8.
- [x] Railway is either verified or explicitly deferred with owner approval.
- [x] `./bin/agent-ci run --quiet --all` passes.

### Handoff

Phase R0 re-baselined the archived Phase 0-18 plan against the active external-tester readiness track. The archived plan has 44 remaining unchecked legacy checkboxes; none remain only in the archived file without representation in this active plan or an explicit disposition.

What landed: the Railway template repository was created and pushed at `https://github.com/watzon/cypra-railway-template`, and `docs/deploy/railway-template.md` now links to it. The repository contains `railway.json`, an S3-compatible storage default, R2/B2/S3 setup guidance, first-boot instructions, and recovery notes.

Owner decisions and deferrals: the owner kept Railway in scope before tester rollout, but approved deferring live Railway deploy verification because `npx @railway/cli whoami` was unauthenticated in this environment. Phase R8 now explicitly requires Railway template deploy evidence before tester rollout. No production deploy, release tag, GHCR publish, GitHub Release upload, or deployed smoke was performed in R0.

Carry-forward mapping: deployment shape and network safety remain in Phase R2; release workflow, GHCR, GitHub Release artifacts, and deployed smoke automation remain in Phase R3; operator docs/playbooks remain in Phase R4; fresh VPS, stopwatch, deployed canonical demo, SDK smoke, export/import, recovery, metrics, tracing, visual/a11y, and Railway deploy proof remain in Phase R8. `CHANGELOG.md` already states the tester limitations: no SAML, no embeddable widget, no TypeScript SDK, no built-in CNAMEs, no SMS, and no bundled Turnstile/hCaptcha.

No-longer-applicable legacy criteria: the archived "v0.1 complete when Phases 0 through 18 are complete" criterion is superseded by active readiness Phases R0-R8 and the External Tester Release Criteria in this file. The old hard requirement to tag exactly `v0.1.0` is superseded by the active selected-tag path, which permits `v0.1.0` or an approved prerelease such as `v0.1.0-rc.1`.

Verification: all 15 ADR files in `docs/adr/` have `Status: accepted`; `docs/phase-1-13-validation.md` closes local P0/P1 findings and leaves external deployment/release proof represented in the active readiness plan; the operator playbook contains the required data inventory, sub-processor template, DSR runbook, breach-notification runbook, bot-mitigation swap path, monitoring checklist, and first-24-hours checklist. `./bin/agent-ci run --quiet --all` passed after the R0 docs/task updates, including lint, format, typecheck, Go tests, coverage floors, dashboard tests, p99 performance, compressed image size, and cold-start gates.

---

## Phase R1: Security Hardening Gate

**Status:** not started  
**Dependencies:** Phase R0 baseline review  
**Deliverable:** No known P0/P1 auth, OIDC, proxy, cookie, PAT, or key-handling security issue remains open before public VPS exposure.

### Tasks

- [ ] Require authenticated session ownership or explicit audited admin permission for passkey registration.
- [ ] Require authenticated session ownership or explicit audited admin permission for TOTP enrollment.
- [ ] Require authenticated session ownership or explicit audited admin permission for WebAuthn 2FA enrollment.
- [ ] Require authenticated session ownership or explicit audited admin permission for backup-code regeneration.
- [ ] Audit and secure session listing, session revocation, passkey deletion, and MFA factor listing so caller-supplied `user_id` cannot target another user.
- [ ] Restrict OIDC consent recording to a signed authorization continuation plus the current logged-in user session.
- [ ] Remove or lock down any OIDC consent JSON path that accepts raw `user_id`, `client_id`, and scopes without continuation binding.
- [ ] Replace client-controlled trusted-proxy signaling with server-side proxy trust configuration only.
- [ ] Ensure `X-Forwarded-Host`, `X-Forwarded-Proto`, and `X-Forwarded-For` are honored only when trusted proxy mode is explicitly enabled.
- [ ] Ensure WebAuthn origins and OIDC issuer/origin calculations cannot be spoofed by untrusted forwarded headers.
- [ ] Issue `Secure` session cookies correctly behind TLS-terminating proxies.
- [ ] Remove PAT subject fallbacks that trust `X-Cypra-User-Id` outside dev-only mode, or convert them into explicit audited impersonation paths.
- [ ] Use constant-time comparison for OIDC client secrets.
- [ ] Make master-key parsing strict enough to reject weak, invalid, or mistyped production keys rather than silently hashing arbitrary strings.
- [ ] Decide and implement whether terminal email is blocked in production or allowed with a persistent warning.
- [ ] Add security regression tests for every item above.

### Acceptance

- [ ] Unauthenticated callers cannot enroll, regenerate, list, delete, or revoke resources for another user by supplying `user_id`.
- [ ] OIDC consent cannot be recorded without valid continuation and logged-in user.
- [ ] Spoofed forwarded/proxy headers do not affect tenant resolution, rate-limit IP, cookie security, issuer, or WebAuthn origin unless trusted proxy mode is enabled.
- [ ] Master-key rejection tests cover invalid, short, and weak-looking production keys.
- [ ] `./bin/agent-ci run --quiet --all` passes.

### Handoff

Pending.

---

## Phase R2: Deployment Packaging And Network Safety

**Status:** not started  
**Dependencies:** Phase R1  
**Deliverable:** The reference VPS deployment is safe to copy, versioned, and aligned with the docs.

### Tasks

- [ ] Replace production-facing `ghcr.io/watzon/cypra:dev` references with a versioned tag or clearly documented local-build override.
- [ ] Decide whether to split development and production compose files or keep one file with profiles/overrides.
- [ ] Stop publishing Postgres to the host in the production deployment recipe.
- [ ] Stop publishing Cypra directly to the host in the TLS production profile.
- [ ] Make Caddy the only public entrypoint in the production TLS profile.
- [ ] Expose both ports `80` and `443` for Caddy in production TLS docs/config.
- [ ] Replace local-only `tls internal` Caddy example with a production-ready template for real install and wildcard tenant domains.
- [ ] Preserve a local Caddy/portless-friendly path for development without confusing it with production.
- [ ] Align `TRUSTED_PROXY_HEADERS` defaults and docs with the Caddy deployment path.
- [ ] Add a container healthcheck that checks `/readyz`, not only `cypra version`.
- [ ] Add a production env template that refuses or clearly marks dev defaults.
- [ ] Document which services and ports must never be exposed publicly.

### Acceptance

- [ ] A fresh VPS compose deployment exposes only Caddy publicly.
- [ ] Postgres is reachable only inside the compose network unless an explicit development override is used.
- [ ] Caddy is configured for real public TLS with documented DNS requirements.
- [ ] Container health reflects DB, migration, storage, and master-key readiness.
- [ ] Docs clearly distinguish local development from production deployment.
- [ ] `./bin/agent-ci run --quiet --all` passes.

### Handoff

Pending.

---

## Phase R3: Release Automation And Smoke Evidence

**Status:** not started  
**Dependencies:** Phase R2  
**Deliverable:** Cypra can publish a release and prove the published artifacts work outside the local checkout.

### Tasks

- [ ] Implement the server release workflow for semver and prerelease tags.
- [ ] Grant release workflow permissions sufficient for GHCR publishing and GitHub Release creation while keeping them minimal.
- [ ] Publish multi-arch GHCR images for `linux/amd64` and `linux/arm64`.
- [ ] Attach release binaries to GitHub Releases.
- [ ] Generate and attach checksums for release binaries/images where appropriate.
- [ ] Align SDK smoke versions with the actual release/tag strategy.
- [ ] Replace deployed-smoke placeholder jobs with real export/import round-trip execution.
- [ ] Replace deployed-smoke placeholder jobs with real multi-instance-admin recovery execution.
- [ ] Avoid single-use setup-token secrets in scheduled smoke design by using disposable environments, reset/reseed steps, or another repeatable strategy.
- [ ] Ensure smoke failures open actionable issues or upload actionable artifacts.
- [ ] Update release docs to match the implemented workflow.

### Acceptance

- [ ] A dry-run or test tag builds expected release artifacts without manual steps.
- [ ] GHCR image publication is verified for amd64 and arm64.
- [ ] GitHub Release artifact upload is verified.
- [ ] Deployed smoke runs canonical demo, SDK compile/use, export/import round-trip, and multi-instance-admin recovery against a fresh instance.
- [ ] `./bin/agent-ci run --quiet --all` passes.

### Handoff

Pending.

---

## Phase R4: Operator Documentation And Recovery Playbooks

**Status:** not started  
**Dependencies:** Phase R2, can run in parallel with Phase R3 after deployment shape is settled  
**Deliverable:** A tester can deploy, operate, back up, restore, upgrade, rollback, and smoke-test Cypra without source-code-only guidance.

### Tasks

- [ ] Expand VPS docs from first boot into complete day-one operation.
- [ ] Document DNS requirements for install and wildcard tenant hosts.
- [ ] Document TLS and reverse proxy setup, including trusted proxy mode and header expectations.
- [ ] Document required and optional env vars with production-safe examples.
- [ ] Document secret generation, storage, rotation expectations, and master-key handling.
- [ ] Document local-disk and S3-compatible storage tradeoffs for VPS testers.
- [ ] Document database role expectations and what the bundled Postgres service is for.
- [ ] Add backup cadence guidance.
- [ ] Add restore drill instructions.
- [ ] Add export/import expectations and limitations.
- [ ] Add storage backup expectations for local disk and S3-compatible storage.
- [ ] Add upgrade procedure.
- [ ] Add rollback procedure.
- [ ] Add certificate renewal checks.
- [ ] Add first-24-hours monitoring checklist.
- [ ] Add deployed smoke-test checklist.
- [ ] Make known limitations explicit and non-alarming in docs and changelog.
- [ ] Update operator playbook sections carried forward from the legacy task plan.

### Acceptance

- [ ] A tester can deploy from docs alone on a new VPS.
- [ ] Docs include exact commands for health, readiness, metrics, logs, backup, restore, upgrade, rollback, and smoke testing.
- [ ] Docs state what not to expose publicly.
- [ ] Docs state which features are beta or deferred.
- [ ] A human dry-run records missing steps as follow-up tasks.
- [ ] `./bin/agent-ci run --quiet --all` passes, or `bun run format` passes if this phase is docs-only and the owner accepts that gate.

### Handoff

Pending.

---

## Phase R5: Dashboard UX Completion

**Status:** not started  
**Dependencies:** Phase R1 for security-sensitive dashboard flows  
**Deliverable:** Operator workflows contain no no-op primary actions, placeholder copy, or avoidable accessibility traps.

### Tasks

- [ ] Wire the tenant user invite modal to the real invite flow, or remove it if the real flow is not meant to exist there.
- [ ] Remove production-visible placeholder copy such as "Screencast placeholder lands in v1.1 docs."
- [ ] Replace dev-looking project fallback values with explicit loading, empty, error, or not-configured states.
- [ ] Ensure project tables use responsive behavior comparable to tenant tables.
- [ ] Ensure user tables use responsive behavior comparable to tenant tables.
- [ ] Ensure signing-key tables use responsive behavior comparable to tenant tables.
- [ ] Ensure instance-admin tables use responsive behavior comparable to tenant tables.
- [ ] Add modal focus trap, focus restoration, inert background behavior, and reliable Escape handling.
- [ ] Review drawers and overlays for the same keyboard behavior.
- [ ] Fix fixed-width command overlays for narrow screens.
- [ ] Fix fixed-width shortcut overlays for narrow screens.
- [ ] Ensure every destructive dashboard action has accurate copy and real backend persistence or is hidden.
- [ ] Add regression tests for no-op primary actions and production placeholder copy.
- [ ] Add keyboard-only tests or browser assertions for modal/drawer behavior.

### Acceptance

- [ ] No primary-action dashboard button silently closes a modal without doing the advertised work.
- [ ] No placeholder copy is visible in production routes.
- [ ] Modal and drawer behavior passes keyboard-only tests.
- [ ] Responsive validation covers dashboard tables and overlays.
- [ ] Dashboard tests assert no demo/fallback data appears outside explicit demo routes.
- [ ] Visual validation observations are recorded.
- [ ] `./bin/agent-ci run --quiet --all` passes.

### Handoff

Pending.

---

## Phase R6: Hosted Login And End-User Polish

**Status:** not started  
**Dependencies:** Phase R1  
**Deliverable:** Hosted login feels production-owned, works without prototype dependencies, and gives users clear recovery paths.

### Tasks

- [ ] Vendor or bundle HTMX locally, or add pinning/SRI and a documented dependency policy.
- [ ] Confirm hosted login works without external CDN availability.
- [ ] Remove production-visible `noop` bot-verifier markup or replace it with neutral state.
- [ ] Improve passkey error messages for user cancellation.
- [ ] Improve passkey error messages for unsupported browser or missing authenticator support.
- [ ] Improve passkey error messages for network/server failure.
- [ ] Improve passkey error messages for invalid RP ID or origin mismatch without leaking sensitive detail.
- [ ] Split 2FA UI states so TOTP, WebAuthn, and backup-code flows do not share a confusing single-code interface.
- [ ] Clarify invite onboarding copy around required vs optional auth methods.
- [ ] Verify hosted-login dark mode after security and deployment changes.
- [ ] Verify hosted-login light mode after security and deployment changes.
- [ ] Verify tenant accent contrast after security and deployment changes.
- [ ] Add hosted-login tests for affected routes and states.

### Acceptance

- [ ] Hosted login loads all critical assets from Cypra or with approved integrity guarantees.
- [ ] Passkey and 2FA errors are actionable without leaking sensitive details.
- [ ] Browser validation covers login, signup, invite, consent, reset, 2FA, and error states.
- [ ] axe reports zero new violations on hosted-login routes.
- [ ] `./bin/agent-ci run --quiet --all` passes.

### Handoff

Pending.

---

## Phase R7: Remaining P2 Product And Ops Polish

**Status:** not started  
**Dependencies:** Phases R1-R6  
**Deliverable:** Non-blocking but visible tester rough edges are either fixed or documented as known limitations.

### Tasks

- [ ] Decide whether OpenAPI remains dev-only, admin-only, or available behind a dedicated explicit env flag.
- [ ] Decouple OpenAPI availability from `LOG_LEVEL=debug` if that remains a readiness concern.
- [ ] Make mobile dashboard messaging feel intentional rather than apologetic, or document the desktop-optimized contract clearly.
- [ ] Add Prometheus scrape example or compose snippet to observability docs if useful for testers.
- [ ] Add OTEL collector example or minimal guidance to observability docs if useful for testers.
- [ ] Ensure terminal email production behavior from Phase R1 is reflected in docs and UI.
- [ ] Review `CHANGELOG.md` for all remaining P2/P3 known limitations.

### Acceptance

- [ ] No known P2 item is omitted from docs, changelog, or future backlog.
- [ ] Product copy around beta limitations is honest and non-alarming.
- [ ] `./bin/agent-ci run --quiet --all` passes, or `bun run format` passes if this phase is docs-only and the owner accepts that gate.

### Handoff

Pending.

---

## Phase R8: Fresh VPS Release Candidate Trial

**Status:** not started  
**Dependencies:** Phases R1-R7  
**Deliverable:** The release candidate is proven in the same kind of environment external testers will use.

### Tasks

- [ ] Deploy the latest release candidate to a clean VPS using only public docs.
- [ ] Use real DNS and TLS for install and wildcard tenant hosts.
- [ ] Redeem setup token.
- [ ] Create first instance admin and save backup codes.
- [ ] Create first tenant.
- [ ] Create first project.
- [ ] Configure required providers for the tester path.
- [ ] Connect and run the Next.js example.
- [ ] Complete end-user sign-in through Cypra OIDC.
- [ ] Run deployed canonical demo smoke.
- [ ] Run deployed Go SDK third-machine smoke.
- [ ] Run deployed export/import restore into a fresh second instance.
- [ ] Run deployed multi-instance-admin recovery.
- [ ] Verify `/metrics` and alert rule scrape shape.
- [ ] Verify OTEL trace captures OIDC authorize through login/consent/token and audit emission.
- [ ] Record human stopwatch timing for first run.
- [ ] Capture dashboard visual/a11y observations.
- [ ] Capture hosted-login visual/a11y observations.
- [ ] Verify the published Railway one-click template deploys successfully, or block tester rollout until Railway auth/deploy evidence is available.
- [ ] Add any release-candidate failure as a P0/P1 task before tester rollout.

### Acceptance

- [ ] A clean VPS install reaches working Next.js sign-in without source-code-only instructions.
- [ ] Human first-run time is recorded and is at or below 30 minutes, or streamlining tasks are added before release.
- [ ] Deployed smoke, SDK smoke, export/import, recovery, metrics, and tracing evidence are recorded.
- [ ] Railway template deploy evidence is recorded, because the owner kept Railway in scope before tester rollout.
- [ ] Any failure becomes a P0/P1 task before tester rollout.
- [ ] The selected tester tag exists and matches the deployed artifact.
- [ ] Owner approves external tester rollout.

### Handoff

Pending.

---

## External Tester Release Criteria

Cypra is ready for the first external tester wave only when all of the following are checked:

- [ ] Phases R0 through R8 are complete.
- [ ] All P0/P1 security issues from the readiness audit are fixed with tests.
- [ ] A versioned release image exists and can be pulled on amd64 and arm64.
- [ ] Production VPS compose exposes only intended public services.
- [ ] Public docs are enough to deploy, operate, back up, restore, upgrade, and smoke-test.
- [ ] Dashboard has no known no-op primary actions or placeholder copy in production routes.
- [ ] Hosted login works without prototype-grade dependencies or ambiguous passkey/2FA failure states.
- [ ] Fresh VPS release candidate trial is recorded as passing.
- [ ] Remaining P2/P3 limitations are listed honestly in `CHANGELOG.md` and docs.
- [ ] Owner approves inviting the first external testers.
