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
- [x] Record the owner-approved Railway post-tester deferral; do not require the separate Railway template repository before the first external tester wave.
- [x] Confirm the human stopwatch pass through `docs/firstrun.md` is explicitly scheduled in Phase R8.
- [x] Confirm deployed-instance canonical demo smoke against a fresh deployed instance is assigned to Phase R8.
- [x] Confirm Go SDK third-machine smoke against release-accurate SDK module tags is assigned to Phases R3 and R8.
- [x] Confirm `cypra export` / `cypra import` round-trip smoke against a fresh second instance is assigned to Phases R3 and R8.
- [x] Confirm multi-instance-admin recovery smoke against a deployed instance is assigned to Phases R3 and R8.
- [x] Confirm deployed `/metrics` runtime-metric verification is assigned to Phase R8.
- [x] Confirm deployed OpenTelemetry trace verification for `/oidc/authorize` through login/consent/token, email worker dispatch where applicable, and audit emission is assigned to Phase R8.
- [x] Confirm deployed visual validation with `agent-browser` against `https://<install>` and `https://<tenant>.<install>` is assigned to Phase R8.
- [x] Confirm selected external-tester release tagging, either `v0.1.0` or an approved prerelease such as `v0.1.0-rc.1`, is assigned to Phases R3 and R8.
- [x] Confirm release workflow trigger and verification is assigned to Phase R3.
- [x] Confirm multi-arch GHCR image publication for `linux/amd64` and `linux/arm64` is assigned to Phase R3.
- [x] Confirm GitHub Release binary and checksum attachment is assigned to Phase R3.
- [x] Confirm `docker pull ghcr.io/watzon/cypra:<tag>` verification on amd64 and arm64 is assigned to Phase R3.
- [x] Ensure `CHANGELOG.md` describes the tester release honestly, including known limitations: no SAML, no embeddable widget, no TypeScript SDK, no built-in CNAMEs, and no SMS.
- [x] Confirm all ADRs that remain part of the active architecture are checked in with accepted status.
- [x] Confirm `docs/phase-1-13-validation.md` has fixed P0/P1 findings documented with evidence and remaining external proof tracked in this active plan.
- [x] Confirm operator playbook requirements are represented by existing playbook sections and Phase R4 follow-up tasks: data inventory, sub-processor template, DSR runbook, breach-notification runbook, bot-mitigation swap path, monitoring checklist, and first-24-hours checklist.
- [x] Confirm CI remains green, including canonical-demo e2e, managed examples smoke, backup/import smoke, coverage floors, p99 performance, compressed image size, and cold-start gates.
- [x] Archive any legacy completion criteria that are no longer applicable under `PLAN.md` v2 with a short note in this phase handoff.

### Acceptance

- [x] No unchecked task remains only in the archived legacy plan without representation in this active file or an explicit no-longer-applicable decision.
- [x] Legacy release/deployment evidence is either complete or assigned to Phases R2, R3, R4, or R8 below.
- [x] Human first-run stopwatch result is recorded or explicitly scheduled in Phase R8.
- [x] Railway is either verified or explicitly deferred with owner approval.
- [x] `./bin/agent-ci run --quiet --all` passes.

### Handoff

Phase R0 reconciled the archived legacy task plan into the active readiness plan. The archived unchecked set was complete: Phase 14 release/deployed evidence, Phase 18 human stopwatch/Railway/deployed smoke/metrics+OTEL/release evidence, and project-completion criteria. No unchecked legacy task remains only in the archive without representation in this active file.

Railway one-click deployment is explicitly owner-approved as a post-tester deferral, so the separate Railway template repository is not required before the first external tester wave. Human first-run stopwatch timing is scheduled in Phase R8. Deployment, release, GHCR, GitHub Release, deployed smoke, metrics, OTEL, and deployed visual-validation evidence remain assigned to Phases R2, R3, R4, and R8 rather than being attempted in this baseline reconciliation phase.

Confirmed existing readiness evidence: `CHANGELOG.md` lists the tester-release limitations including no SAML, no embeddable widget, no TypeScript SDK, no built-in CNAME management, and no SMS/Twilio; 15 ADR files are checked in with `Status: accepted`; `docs/phase-1-13-validation.md` has a Phase 18 closure appendix for fixed P0/P1 findings and leaves remaining external deployment/release proof tracked in this active plan; the operator playbook sections exist and Phase R4 owns their expansion into complete day-one operations guidance.

No PLAN.md or DESIGN.md conflict was found. Local generated Vite cache was added to `.gitignore` so generated `dashboard/.vite/` files no longer break formatting checks. Verification: `./bin/agent-ci run --quiet --all` passed after loading the `agent-ci` skill, including dashboard build, Go build, `golangci-lint`, color lint, Prettier format check, `go vet`, dashboard typecheck, `go test -p 1 ./...`, coverage floors, dashboard Vitest, p99 performance, compressed Docker image-size, and cold-start (`cold-start readyz: 190ms`).

---

## Phase R1: Security Hardening Gate

**Status:** complete  
**Dependencies:** Phase R0 baseline review  
**Deliverable:** No known P0/P1 auth, OIDC, proxy, cookie, PAT, or key-handling security issue remains open before public VPS exposure.

### Tasks

- [x] Require authenticated session ownership or explicit audited admin permission for passkey registration.
- [x] Require authenticated session ownership or explicit audited admin permission for TOTP enrollment.
- [x] Require authenticated session ownership or explicit audited admin permission for WebAuthn 2FA enrollment.
- [x] Require authenticated session ownership or explicit audited admin permission for backup-code regeneration.
- [x] Audit and secure session listing, session revocation, passkey deletion, and MFA factor listing so caller-supplied `user_id` cannot target another user.
- [x] Restrict OIDC consent recording to a signed authorization continuation plus the current logged-in user session.
- [x] Remove or lock down any OIDC consent JSON path that accepts raw `user_id`, `client_id`, and scopes without continuation binding.
- [x] Replace client-controlled trusted-proxy signaling with server-side proxy trust configuration only.
- [x] Ensure `X-Forwarded-Host`, `X-Forwarded-Proto`, and `X-Forwarded-For` are honored only when trusted proxy mode is explicitly enabled.
- [x] Ensure WebAuthn origins and OIDC issuer/origin calculations cannot be spoofed by untrusted forwarded headers.
- [x] Issue `Secure` session cookies correctly behind TLS-terminating proxies.
- [x] Remove PAT subject fallbacks that trust `X-Cypra-User-Id` outside dev-only mode, or convert them into explicit audited impersonation paths.
- [x] Use constant-time comparison for OIDC client secrets.
- [x] Make master-key parsing strict enough to reject weak, invalid, or mistyped production keys rather than silently hashing arbitrary strings.
- [x] Decide and implement whether terminal email is blocked in production or allowed with a persistent warning.
- [x] Add security regression tests for every item above.

### Acceptance

- [x] Unauthenticated callers cannot enroll, regenerate, list, delete, or revoke resources for another user by supplying `user_id`.
- [x] OIDC consent cannot be recorded without valid continuation and logged-in user.
- [x] Spoofed forwarded/proxy headers do not affect tenant resolution, rate-limit IP, cookie security, issuer, or WebAuthn origin unless trusted proxy mode is enabled.
- [x] Master-key rejection tests cover invalid, short, and weak-looking production keys.
- [x] `./bin/agent-ci run --quiet --all` passes.

### Handoff

Completed the pre-public-exposure security hardening pass. Ownership checks now bind passkey, TOTP, WebAuthn 2FA, backup-code, factor listing/deletion, and session listing/revocation operations to the authenticated user unless an explicit server-side admin path is used. OIDC consent recording is bound to the signed authorization continuation and the current logged-in user session. Forwarded host/proto/for headers are ignored unless server-side trusted proxy mode is enabled, and trusted request origin handling is reused for issuer, WebAuthn origin, cookies, and rate-limit IP decisions. PAT creation no longer falls back to bare caller-controlled `X-Cypra-User-Id`, OIDC client-secret comparison uses constant-time digest comparison, production CLI master-key parsing rejects invalid/short/weak-looking keys, and terminal email is blocked for non-local public URLs.

Verification: added regression coverage across `cmd/cypra`, `internal/httpserver`, `internal/email`, `internal/oidc`, and `internal/crypto`; `./bin/agent-ci run --quiet --all` passed, including lint, format, typecheck, Go tests, coverage floors, dashboard tests, p99 performance, compressed image size, and cold-start (`cold-start readyz: 1190ms`).

---

## Phase R2: Deployment Packaging And Network Safety

**Status:** complete  
**Dependencies:** Phase R1  
**Deliverable:** The reference VPS deployment is safe to copy, versioned, and aligned with the docs.

### Tasks

- [x] Replace production-facing `ghcr.io/watzon/cypra:dev` references with a versioned tag or clearly documented local-build override.
- [x] Decide whether to split development and production compose files or keep one file with profiles/overrides.
- [x] Stop publishing Postgres to the host in the production deployment recipe.
- [x] Stop publishing Cypra directly to the host in the TLS production profile.
- [x] Make Caddy the only public entrypoint in the production TLS profile.
- [x] Expose both ports `80` and `443` for Caddy in production TLS docs/config.
- [x] Replace local-only `tls internal` Caddy example with a production-ready template for real install and wildcard tenant domains.
- [x] Preserve a local Caddy/portless-friendly path for development without confusing it with production.
- [x] Align `TRUSTED_PROXY_HEADERS` defaults and docs with the Caddy deployment path.
- [x] Add a container healthcheck that checks `/readyz`, not only `cypra version`.
- [x] Add a production env template that refuses or clearly marks dev defaults.
- [x] Document which services and ports must never be exposed publicly.

### Acceptance

- [x] A fresh VPS compose deployment exposes only Caddy publicly.
- [x] Postgres is reachable only inside the compose network unless an explicit development override is used.
- [x] Caddy is configured for real public TLS with documented DNS requirements.
- [x] Container health reflects DB, migration, storage, and master-key readiness.
- [x] Docs clearly distinguish local development from production deployment.
- [x] `./bin/agent-ci run --quiet --all` passes.

### Handoff

Phase R2 hardened the reference deployment shape. Production compose now defaults to a versioned release image, keeps Cypra and Postgres private on the compose network, publishes only Caddy on ports 80 and 443 in the TLS profile, and defaults trusted proxy handling to `x-forwarded` for that Caddy path. Development-only host port mappings and the local `:dev` image now live in `deploy/docker-compose.dev.yml`, with `deploy/Caddyfile.local` preserving the `.localhost` / `tls internal` path separately from production.

Added `cypra healthcheck`, wired Docker and compose healthchecks to `/readyz`, and covered success/failure behavior with `cmd/cypra` tests. Added `.env.production.example` with explicit `CHANGE_ME` production placeholders and allowed it through `.gitignore`; `.env.example` remains the local-development template. Updated deployment docs and README to distinguish production TLS deployment from local development, document the required DNS records, note wildcard-certificate DNS-01 requirements, and state that `postgres:5432` and `cypra:8080` must never be exposed publicly.

Verification: `docker compose --env-file .env.production.example -f deploy/docker-compose.yml --profile with-tls config` showed only Caddy publishes `80` and `443`; Cypra and Postgres have no production host ports. `docker compose --env-file .env.example -f deploy/docker-compose.yml -f deploy/docker-compose.dev.yml config` showed local-only host ports remain in the dev override. `go test ./cmd/cypra`, `bun run lint`, `bun run typecheck`, and `bun run format` passed. Final `./bin/agent-ci run --quiet --all` passed after loading the `agent-ci` skill, including dashboard build, Go build, `golangci-lint`, color lint, Prettier format check, `go vet`, dashboard typecheck, `go test -p 1 ./...`, coverage floors, dashboard Vitest, p99 performance, compressed Docker image-size, and cold-start (`cold-start readyz: 147ms`).

---

## Phase R3: Release Automation And Smoke Evidence

**Status:** complete
**Dependencies:** Phase R2  
**Deliverable:** Cypra can publish a release and prove the published artifacts work outside the local checkout.

### Tasks

- [x] Implement the server release workflow for semver and prerelease tags.
- [x] Grant release workflow permissions sufficient for GHCR publishing and GitHub Release creation while keeping them minimal.
- [x] Publish multi-arch GHCR images for `linux/amd64` and `linux/arm64`.
- [x] Attach release binaries to GitHub Releases.
- [x] Generate and attach checksums for release binaries/images where appropriate.
- [x] Align SDK smoke versions with the actual release/tag strategy.
- [x] Replace deployed-smoke placeholder jobs with real export/import round-trip execution.
- [x] Replace deployed-smoke placeholder jobs with real multi-instance-admin recovery execution.
- [x] Avoid single-use setup-token secrets in scheduled smoke design by using disposable environments, reset/reseed steps, or another repeatable strategy.
- [x] Ensure smoke failures open actionable issues or upload actionable artifacts.
- [x] Update release docs to match the implemented workflow.

### Acceptance

- [x] A dry-run or test tag builds expected release artifacts without manual steps.
- [x] GHCR image publication is verified for amd64 and arm64.
- [x] GitHub Release artifact upload is verified.
- [x] Deployed smoke runs canonical demo, SDK compile/use, export/import round-trip, and multi-instance-admin recovery against a fresh instance.
- [x] Smoke failures open an actionable issue or upload actionable artifacts.
- [x] `./bin/agent-ci run --quiet --all` passes.

### Handoff

Phase R3 closed the release automation and external smoke evidence path. The release workflow accepts semver/prerelease server tags, builds Linux/Darwin binaries for amd64/arm64, builds/pushes a linux/amd64+linux/arm64 GHCR image, records image digests, generates checksums, and creates GitHub Releases. SDK tags remain separate under `sdk/go/vX.Y.Z[-prerelease]`, with third-machine SDK smoke validating `github.com/watzon/cypra/sdk/go@<version>`.

External release evidence used prerelease `v0.1.0-rc.2`. Release workflow run `25616763365` passed, including server binary builds, GHCR image publication, checksum generation, and GitHub Release creation. The GitHub Release at `https://github.com/watzon/cypra/releases/tag/v0.1.0-rc.2` contains Linux and Darwin amd64/arm64 tarballs, `checksums.txt`, and `image-digests.txt`. GHCR image `ghcr.io/watzon/cypra:0.1.0-rc.2` was verified as manifest digest `sha256:c2984203c40f70784d1804fa8ebc5eee4ca90dbe5c6b987d8d7d10a21f63c70d` with platforms `linux/amd64` and `linux/arm64`.

Deployed smoke now uses disposable Docker Compose environments from the published image instead of a long-lived `CYPRA_SMOKE_SETUP_TOKEN` secret. `scripts/smoke-disposable-release.sh` mints a fresh bootstrap token inside the disposable source stack, runs the canonical demo, exports a backup through a writable mounted artifact path, imports it into a second fresh stack, verifies restored readiness/version/admin listing, issues and redeems a recovery invite, and uploads logs/artifacts through the workflow. The final deployed smoke workflow run `25618512749` passed both `disposable-release-smoke` and `sdk-third-machine`; it exercised canonical demo, SDK compile/use, export/import round-trip, and multi-instance-admin recovery against `ghcr.io/watzon/cypra:0.1.0-rc.2` / SDK `v0.1.0-rc.2`.

Smoke failure reporting was verified before the final green run: failed workflow runs uploaded actionable artifacts and opened GitHub issues, including `https://github.com/watzon/cypra/issues/11` and `https://github.com/watzon/cypra/issues/12`, each pointing back to the failing run and artifact evidence. Local verification during closeout: `bash -n scripts/smoke-disposable-release.sh`, `bun run format`, `bun run typecheck`, and full disposable release smoke against `ghcr.io/watzon/cypra:0.1.0-rc.2` passed. Final phase gate: `./bin/agent-ci run --quiet --all` passed after loading the `agent-ci` skill, including dashboard build, Go build, `golangci-lint`, color lint, Prettier format check, `go vet`, dashboard typecheck, `go test -p 1 ./...`, coverage floors, dashboard Vitest, p99 performance, compressed Docker image-size, and cold-start (`cold-start readyz: 149ms`). `actionlint` is not installed locally, so workflow semantic linting was not run outside GitHub Actions.

---

## Phase R4: Operator Documentation And Recovery Playbooks

**Status:** complete
**Dependencies:** Phase R2, can run in parallel with Phase R3 after deployment shape is settled  
**Deliverable:** A tester can deploy, operate, back up, restore, upgrade, rollback, and smoke-test Cypra without source-code-only guidance.

### Tasks

- [x] Expand VPS docs from first boot into complete day-one operation.
- [x] Document DNS requirements for install and wildcard tenant hosts.
- [x] Document TLS and reverse proxy setup, including trusted proxy mode and header expectations.
- [x] Document required and optional env vars with production-safe examples.
- [x] Document secret generation, storage, rotation expectations, and master-key handling.
- [x] Document local-disk and S3-compatible storage tradeoffs for VPS testers.
- [x] Document database role expectations and what the bundled Postgres service is for.
- [x] Add backup cadence guidance.
- [x] Add restore drill instructions.
- [x] Add export/import expectations and limitations.
- [x] Add storage backup expectations for local disk and S3-compatible storage.
- [x] Add upgrade procedure.
- [x] Add rollback procedure.
- [x] Add certificate renewal checks.
- [x] Add first-24-hours monitoring checklist.
- [x] Add deployed smoke-test checklist.
- [x] Make known limitations explicit and non-alarming in docs and changelog.
- [x] Update operator playbook sections carried forward from the legacy task plan.

### Acceptance

- [x] A tester can deploy from docs alone on a new VPS.
- [x] Docs include exact commands for health, readiness, metrics, logs, backup, restore, upgrade, rollback, and smoke testing.
- [x] Docs state what not to expose publicly.
- [x] Docs state which features are beta or deferred.
- [x] A human dry-run records missing steps as follow-up tasks.
- [x] `./bin/agent-ci run --quiet --all` passes, or `bun run format` passes if this phase is docs-only and the owner accepts that gate.

### Handoff

Phase R4 expanded the operator documentation from first-boot notes into a day-one VPS operating guide. `docs/deploy/vps.md` now covers host prerequisites, DNS, TLS/Caddy, trusted proxy headers, production env setup, secret generation and rotation expectations, storage and database choices, health/readiness/log/metrics checks, first boot, backup cadence, restore drills, export/import limits, upgrade, rollback, certificate renewal, first-24-hours monitoring, deployed smoke testing, and beta limitations. Supporting docs now align: `deploy/README.md`, `.env.production.example`, `docs/deploy/observability.md`, `docs/playbook/*`, and `CHANGELOG.md`.

No PLAN.md or DESIGN.md conflict was found. This was a docs/operator phase, so no UI browser validation was required. The operator read-through dry-run found one missing restore-verification detail: the restore drill tried to use a restore HTTPS host without starting Caddy. The docs now verify the restored stack with the Cypra container healthcheck and `version --json`; no follow-up task was needed.

Pre-flight root cause: the external `agent-ci` skill currently prescribes `npx @redwoodjs/agent-ci`, which failed in its container before project checks because `make` was unavailable. This repo's source of truth is consistently `./bin/agent-ci run --quiet --all` via `PLAN.md`, `TASKS.md`, and `CONTRIBUTING.md`; Bitforge local instructions were updated to prefer the repository's canonical CI command after loading the `agent-ci` skill. Verification used the repo-local gate.

Verification: `bun run format` passed; `git diff --check` passed; `make lint` passed; `make typecheck` passed; final `./bin/agent-ci run --quiet --all` passed, including build, lint, format, typecheck, Go tests, coverage floors, dashboard tests, performance, image-size, and cold-start (`cold-start readyz: 145ms`).

---

## Phase R5: Dashboard UX Completion

**Status:** complete
**Dependencies:** Phase R1 for security-sensitive dashboard flows  
**Deliverable:** Operator workflows contain no no-op primary actions, placeholder copy, or avoidable accessibility traps.

### Tasks

- [x] Wire the tenant user invite modal to the real invite flow, or remove it if the real flow is not meant to exist there.
- [x] Remove production-visible placeholder copy such as "Screencast placeholder lands in v1.1 docs."
- [x] Replace dev-looking project fallback values with explicit loading, empty, error, or not-configured states.
- [x] Ensure project tables use responsive behavior comparable to tenant tables.
- [x] Ensure user tables use responsive behavior comparable to tenant tables.
- [x] Ensure signing-key tables use responsive behavior comparable to tenant tables.
- [x] Ensure instance-admin tables use responsive behavior comparable to tenant tables.
- [x] Add modal focus trap, focus restoration, inert background behavior, and reliable Escape handling.
- [x] Review drawers and overlays for the same keyboard behavior.
- [x] Fix fixed-width command overlays for narrow screens.
- [x] Fix fixed-width shortcut overlays for narrow screens.
- [x] Ensure every destructive dashboard action has accurate copy and real backend persistence or is hidden.
- [x] Add regression tests for no-op primary actions and production placeholder copy.
- [x] Add keyboard-only tests or browser assertions for modal/drawer behavior.

### Acceptance

- [x] No primary-action dashboard button silently closes a modal without doing the advertised work.
- [x] No placeholder copy is visible in production routes.
- [x] Modal and drawer behavior passes keyboard-only tests.
- [x] Responsive validation covers dashboard tables and overlays.
- [x] Dashboard tests assert no demo/fallback data appears outside explicit demo routes.
- [x] Visual validation observations are recorded.
- [x] `./bin/agent-ci run --quiet --all` passes.

### Handoff

Phase R5 completed the dashboard UX hardening pass. The tenant user invite action now reuses the real invite modal and posts to `/api/v1/admin/invite` instead of silently closing. Production-visible placeholder copy was removed, project issuer/client/secret fallbacks now show explicit not-configured/not-available states, and app setup snippets are hidden until both issuer URL and client ID exist. Project, user, signing-key, and instance-admin tables now opt into the existing responsive stacked-table behavior.

Shared modal and drawer behavior now uses a portal-backed dialog behavior helper with Escape close, focus trap, focus restoration, `data-autofocus="true"`, and inert/`aria-hidden` background handling. The command palette and shortcut overlay now use the shared modal path, removing their fixed-width overlay shells for narrow screens. Destructive dashboard actions were reviewed against existing persistence paths and copy; no hidden no-op destructive primary action remained in scope.

Visual validation used `agent-browser` against the local Vite dashboard at `http://127.0.0.1:5174` with explicit `?state=demo` fixture routes because no live backend was running for the browser pass. Narrow viewport `390x844` observations covered `/dashboard/tenants/acme/users?state=demo`, `/dashboard/tenants/acme/projects?state=demo`, `/dashboard/tenants/acme/signing-keys?state=demo`, and `/dashboard/instance/admins?state=demo`; the affected tables remained reachable and usable with the mobile navigation/banner present. The command palette opened with `Control+k`, focused the search field, displayed in the shared modal shell, and closed with Escape. The shortcut overlay opened from the floating shortcuts button, displayed in the shared modal shell, and closed with Escape. Screenshots were captured under `/var/folders/41/0kyhddh92xnfbg8nqmytvb8r0000gn/T/opencode/` for users, projects, signing keys, and instance admins. Browser axe on the active route returned `[]` violations.

Regression coverage was added in `dashboard/src/App.test.tsx` for the real invite POST, absence of the removed placeholder copy, modal focus trap/inert/Escape/focus restoration behavior, drawer Escape behavior, production error/empty states without demo records, and no axe violations on the covered dashboard routes. Verification: `bun run format`, `make lint`, `make typecheck`, `go test -p 1 ./...`, and `bun run --filter dashboard test` passed; dashboard Vitest reported `44 passed` with the existing jsdom canvas/navigation warnings. Final phase gate: `./bin/agent-ci run --quiet --all` passed, including build, lint, format, typecheck, Go tests, coverage floors, dashboard tests, p99 performance, Docker image-size, and cold-start (`cold-start readyz: 215ms`).

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
- [ ] Add any release-candidate failure as a P0/P1 task before tester rollout.

### Acceptance

- [ ] A clean VPS install reaches working Next.js sign-in without source-code-only instructions.
- [ ] Human first-run time is recorded and is at or below 30 minutes, or streamlining tasks are added before release.
- [ ] Deployed smoke, SDK smoke, export/import, recovery, metrics, and tracing evidence are recorded.
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
