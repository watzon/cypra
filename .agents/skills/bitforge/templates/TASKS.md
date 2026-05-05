<!--
  TASKS.tmpl.md — bitforge phased execution template.

  HOW TO USE
  ----------
  This template is consumed by the bitforge skill's `$source-command-bitforge-taskify` flow.
  When taskifying a project, the agent copies this file to `TASKS.md` at the
  project root and replaces every `{{PLACEHOLDER}}` and `<!-- AGENT: ... -->`
  comment with project-specific content.

  Do not author TASKS.md from scratch. Always copy this file. The operating
  rules and standards baked in here are binding for every agent that picks up
  `$source-command-bitforge-next`.

  AUTHORING NOTES
  ---------------
  - Phases are ordered so the codebase remains runnable at every boundary.
    When designing phases, favor keeping the system bootable and testable
    after each phase, even if that means a phase ships less surface area.
  - Each phase has a fixed shape: Status / Dependencies / Deliverable / Tasks
    / Acceptance / Handoff. Do not invent new sections inside a phase — add a
    new phase or a "Cross-Phase Concerns" entry instead.
  - Acceptance criteria are testable. "Looks good" is not acceptance.
  - The Handoff block is free-form prose written at phase completion. It is
    the single most valuable artifact for the next agent — do not skip it.
  - The Standards section below is mandatory and applies to EVERY phase.
-->

# {{PROJECT_NAME}} — Implementation Tasks

**Companion to:** {{COMPANION_DOCS}}
<!-- AGENT: e.g. `[`PLAN.md`](./PLAN.md)` and `[`DESIGN.md`](./DESIGN.md)`. List every doc that this file defers to as source of truth. -->

This document is the canonical execution plan. It breaks the project into ordered phases that, when followed end-to-end, produce {{ONE_SENTENCE_PROJECT_OUTCOME}}.

<!-- AGENT: Replace ONE_SENTENCE_PROJECT_OUTCOME with a concrete description of
     what shipping this plan results in. e.g. "a working CLI that authenticates,
     reads from the configured backend, and renders a TUI dashboard." -->

---

## Operating rules for agents working from this file

The following rules use [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) terminology. They are not advisory.

> **You MUST keep this file current.** Whenever you start a task, you MUST mark its checkbox in-progress (`- [~]` or note it in the `Status` line of the phase). Whenever you complete a task, you MUST tick its checkbox (`- [x]`). Whenever you finish a phase, you MUST fill the **Handoff** block for that phase before opening the next one. Whenever you discover a gap that this file does not capture, you MUST add a task to the appropriate phase rather than work around it. **You MUST NOT mark a phase complete unless every task and every acceptance criterion in it has been verified.** **You MUST NOT skip phases or merge phases without an explicit instruction from the project owner.** **You SHOULD NOT add scope to a phase that is not on its acceptance list; surface it as a new task or new phase instead.**

> **You MUST treat {{COMPANION_DOC_LIST}} as the source of truth.** Any conflict between this file and any of them is a bug in this file; resolve by aligning to the canonical doc and updating tasks here. **You MUST NOT silently change architecture, data shapes, or visual tokens to make a task easier; raise the conflict and update the canonical doc explicitly.**
<!-- AGENT: COMPANION_DOC_LIST is the same set of docs from the top "Companion to" line, e.g. "PLAN.md and DESIGN.md". If there are no companion docs, replace the whole sentence with "the project's documented requirements". -->

> **Verification gates:** Each phase has acceptance criteria. **You MUST run them locally and confirm passing before declaring the phase complete.** A green CI run is necessary but not sufficient — the acceptance criteria are the binding test.

> **Internal todo discipline:** When picking up a phase, you MUST mirror its Tasks list with OpenCode's todo tool and tick the internal list in lockstep with this file. Both lists end the phase ticked. Do not batch.

---

## Standards (apply to every phase)

These gates are appended to the Acceptance block of every phase. They are **not optional** — a phase is not complete until each applicable gate is green.

- **CI gate** — load the `agent-ci` skill, then run `agent-ci run --quiet --all`; it is green for the changes this phase introduced. Do not roll your own command.
- **Hygiene gate** — lint clean, format clean, typecheck clean, no introduced dead code, no orphaned `console.log` / `print` / `dbg!`, no zombie `TODO` for finished work.
- **Test gate** — new behavior covered by unit tests; new boundary crossings (API↔DB, frontend↔API, worker↔queue, integration↔third-party) covered by integration tests.
- **Visual validation gate** *(applies to every phase that ships UI)* — load the `agent-browser` skill, then load every new or changed surface in `agent-browser` against the running dev server. The agent captures an accessibility-tree snapshot for each surface in each documented state (default / loading / empty / error). Observations are recorded in this phase's Handoff. axe-core (or equivalent) reports zero new violations on touched routes.
- **Phase boundary invariant** — at the end of this phase, a fresh clone, install, and test command succeeds.

---

## How to read this file

- Each phase has a fixed shape:
  - **Status** — `not started` / `in progress` / `complete`. Update it.
  - **Dependencies** — phases that MUST be complete first.
  - **Deliverable** — what concretely exists at the end.
  - **Tasks** — GFM-checkbox list. Each task is sized to be a focused unit of work.
  - **Acceptance** — concrete, testable criteria for "done." Always ends with the Standards gates above.
  - **Handoff** — free-form notes added at phase completion. Anything the next agent should know: what changed, what was added beyond the task list, what was deferred, what surprised you, what the visual-validation pass observed.
- Phases are deliberately ordered so that the codebase remains in a runnable state at every checkpoint. Skipping ahead breaks this property.
- Some phases are internally parallelizable (e.g., independent integrations or modules). Those are called out explicitly.

---

<!-- ============================================================
     EXAMPLE PHASES BEGIN
     ============================================================
     The phases below are EXAMPLES that demonstrate the template's
     shape against a hypothetical web app with a backend, a database,
     a worker, integrations, and a frontend. Adapt them to your
     project — keep the structure, replace the content.

     Recommended phase progression for a typical app:
       0  Foundation         — repo bootable, CI green, licenses
       1  Schema & Types     — data model, validators, migrations
       2  Backend Core       — API, auth, CRUD
       3  Frontend Core      — design system, routing, auth UI
       4  Domain UI          — first major feature surface
       5  Background work    — workers, queues, jobs
       6  Realtime           — streaming, WS, live updates
       7  Secrets / config   — secret store, OAuth, env handling
       8  Integrations       — parallelizable third-party adapters
       9  AI / Agent layer   — only if applicable
      10  Approvals / audit  — policy, gating, compliance
      11  Advanced features  — domain-specific later work
      12  Onboarding / demo  — canonical end-to-end story
      13  Polish             — a11y, responsive, perf, states
      14  Ship               — deploy, observe, document, release

     Trim, expand, or reorder to fit. Most projects need fewer phases.
     ============================================================ -->

## Phase 0: Foundation

**Status:** not started
**Dependencies:** none
**Deliverable:** A bootable repo. Cloning the repo, running one command, and getting a green test run is possible. License files and contribution docs are in place. CI runs on every push.

<!-- AGENT: Phase 0 should always exist. The bar is "fresh clone → tests pass". -->

### Tasks

- [ ] Initialize root `package.json` (or equivalent for the chosen runtime) with workspaces if applicable.
- [ ] Pin the runtime version ({{RUNTIME_AND_VERSION}}) via the appropriate mechanism ({{VERSION_PIN_FILE}}).
- [ ] Add license file(s). {{LICENSE_NOTE}}
- [ ] Add `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`.
- [ ] Configure strict-mode {{LANGUAGE}} settings shared by all packages.
- [ ] Add root scripts: `test`, `lint`, `typecheck`, `dev`, `build`.
- [ ] Configure linter and formatter; document choice in `CONTRIBUTING.md`.
- [ ] Add `docker-compose.yml` for local dev dependencies ({{LOCAL_DEPS}}). Document in `README.md`.
- [ ] Create `.env.example` at root with every env var the system reads, grouped by service. Each var documented inline.
- [ ] Set up CI ({{CI_PLATFORM}}): install runtime, run lint, typecheck, tests. Cache the appropriate dependency directories.
- [ ] Load the `agent-ci` skill and verify it runs the same pipeline locally and produces green output.
- [ ] Write `README.md`: project intent (one paragraph), pointers to companion docs, prerequisites, local dev quickstart, CI badges.
- [ ] Author the initial ADRs for foundational decisions (runtime choice, license split, any unusual stack picks).
- [ ] Set `.gitignore`, `.editorconfig`, and any other root-level dotfiles needed.

### Acceptance

- [ ] `git clone && {{INSTALL_CMD}} && {{TEST_CMD}}` succeeds on a clean machine.
- [ ] `docker compose up -d` brings local deps up; the documented connection works.
- [ ] CI passes on the initial commit.
- [ ] Every directory that needs its own license has one.
- [ ] `README.md` contains a working "from-zero-to-tests-pass" instruction.
- [ ] **CI gate:** load the `agent-ci` skill, then run `agent-ci run --quiet --all`; it is green.
- [ ] **Hygiene gate:** lint / format / typecheck clean.
- [ ] **Phase boundary invariant:** clean clone → install → test succeeds.

### Handoff

_Filled at phase completion. Free-form notes for whoever picks up Phase 1: what landed, deviations from the task list, surprises, gotchas, what is intentionally NOT here yet._

---

## Phase 1: {{PHASE_1_TITLE}}
<!-- AGENT: Common Phase 1 titles: "Schema and Types", "Core Domain Model",
     "Protocol Definition", "API Contract". This is the phase where the *shape*
     of the system is decided in code, before anything that uses it is built. -->

**Status:** not started
**Dependencies:** Phase 0
**Deliverable:** {{PHASE_1_DELIVERABLE}}
<!-- AGENT: One paragraph describing the concrete state of the codebase at end of phase. -->

### Tasks

- [ ] {{TASK_DEFINE_SCHEMAS_OR_TYPES}}
- [ ] {{TASK_GENERATE_VALIDATORS}}
- [ ] {{TASK_AUTHOR_FIXTURES}}
- [ ] {{TASK_AUTHOR_TESTS}}
- [ ] {{TASK_AUTHOR_MIGRATION_OR_VERSIONING_STRATEGY}}
- [ ] Document the package(s) in their respective `README.md`.

<!-- AGENT: Replace each TASK_* with concrete sized tasks. Aim for 6–14 tasks
     per phase — fewer means tasks are too large, more means they're too small. -->

### Acceptance

- [ ] {{ACCEPTANCE_TESTS_PASS}}
- [ ] {{ACCEPTANCE_FIXTURE_COVERAGE}}
- [ ] {{ACCEPTANCE_NO_REQUIRED_TYPE_MISSING}}
- [ ] **CI gate:** load the `agent-ci` skill, then run `agent-ci run --quiet --all`; it is green.
- [ ] **Hygiene gate:** lint / format / typecheck clean; no introduced dead code.
- [ ] **Test gate:** unit tests cover schema invariants; fixture coverage matches the schema surface.
- [ ] **Phase boundary invariant:** clean clone → install → test succeeds.

### Handoff

_Filled at phase completion._

---

## Phase 2: {{PHASE_2_TITLE}}
<!-- AGENT: Typical: "Backend Core" / "API surface" / "Service skeleton". This is
     where the schema becomes a running service with routes/CRUD/auth. -->

**Status:** not started
**Dependencies:** Phase 1
**Deliverable:** {{PHASE_2_DELIVERABLE}}

### Tasks

- [ ] Scaffold the {{SERVICE_NAME}} entry point.
- [ ] Wire the database layer; run migrations on startup with a flag to skip in tests.
- [ ] Integrate authentication ({{AUTH_PROVIDER}}). Configure session handling.
- [ ] Implement an `auth`/`requireAuth` mechanism: protected routes resolve user/session; unauthenticated requests get 401.
- [ ] Build CRUD modules for: {{CORE_RESOURCE_LIST}}.
- [ ] Build a health-check module: `/healthz` (alive), `/readyz` (deps reachable).
- [ ] Add API documentation generation (OpenAPI / equivalent), surface in dev only.
- [ ] Export the typed client/SDK so other packages consume the API with full type safety.
- [ ] Wire transactional email (if applicable). In dev, emails go to stdout.
- [ ] Add a seed script.
- [ ] Add basic rate limiting with documented defaults.

### Acceptance

- [ ] `{{DEV_CMD}}` starts the server, runs migrations, and listens on the documented port.
- [ ] `curl /healthz` returns 200; `curl /readyz` returns 200 only after deps are reachable.
- [ ] {{ACCEPTANCE_END_TO_END_HAPPY_PATH}}
- [ ] {{ACCEPTANCE_NO_SECRET_LEAKS}} <!-- e.g. "Creating a Connection via API never returns secret material in any response." -->
- [ ] OpenAPI doc lists every route with parameter and response schemas.
- [ ] **CI gate:** load the `agent-ci` skill, then run `agent-ci run --quiet --all`; it is green.
- [ ] **Hygiene gate:** lint / format / typecheck clean.
- [ ] **Test gate:** unit + integration tests cover every CRUD route and the auth boundary.
- [ ] **Phase boundary invariant:** clean clone → install → test succeeds.

### Handoff

_Filled at phase completion._

---

## Phase 3: {{PHASE_3_TITLE}}
<!-- AGENT: Typical: "Frontend Foundation and Design System". The phase where the
     UI shell, design tokens, primitives, routing, and auth screens land. SHIPS UI:
     visual validation gate is mandatory. -->

**Status:** not started
**Dependencies:** Phase 2
**Deliverable:** {{PHASE_3_DELIVERABLE}}

### Tasks

- [ ] Scaffold `{{WEB_APP_PATH}}` with {{FRONTEND_STACK}}.
- [ ] Configure {{STYLING_SYSTEM}}; author the token system from {{DESIGN_DOC_REF}}; both dark and light modes.
- [ ] Install typography ({{FONTS}}) and icons ({{ICON_LIBRARY}}).
- [ ] Implement design system primitives: {{PRIMITIVE_LIST}}.
- [ ] Set up the router with the route tree for the primary screens. Most routes render placeholders; full content arrives in later phases.
- [ ] Set up data fetching ({{QUERY_LIBRARY}}) and configure global defaults (stale time, retry, error toast).
- [ ] Wire the auth client; implement sign-in / sign-up / sign-out / accept-invite screens.
- [ ] Implement theme switching (system / dark / light) persisted per user.
- [ ] Implement a primary navigation shell with the items from {{DESIGN_DOC_REF}}.
- [ ] Implement keyboard-shortcut surfaces (command palette, cheatsheet) if applicable.
- [ ] Add a component gallery route that shows every primitive in every state, both modes.
- [ ] Add accessibility plumbing: focus-visible, `prefers-reduced-motion`, axe-core in dev.

### Acceptance

- [ ] `{{DEV_CMD}}` starts the frontend; visiting the dev URL shows the sign-in screen.
- [ ] After signing in, the user lands on `/` rendered inside the shell.
- [ ] Toggling theme flips every token reference; no hardcoded color values appear in the rendered DOM.
- [ ] All primitives are reproducible in code with matching spacing, typography, and color (verified against the design doc).
- [ ] Axe-core reports no violations on the component gallery.
- [ ] The user can complete sign-up, sign-in, sign-out end-to-end.
- [ ] **CI gate:** load the `agent-ci` skill, then run `agent-ci run --quiet --all`; it is green.
- [ ] **Hygiene gate:** lint / format / typecheck clean; no token leaks (no hardcoded hex outside the token file).
- [ ] **Test gate:** unit tests for primitives in every state + variant; integration tests for the auth flow.
- [ ] **Visual validation gate:** load the `agent-browser` skill, then observe every primitive in every state and both modes in `agent-browser`. Sign-in / sign-up / sign-out flows walked end-to-end. axe-core run on every authenticated route. Observations recorded in Handoff.
- [ ] **Phase boundary invariant:** clean clone → install → test succeeds.

### Handoff

_Filled at phase completion. Include visual-validation observations: which surfaces were loaded, which states were exercised, any token / state / a11y discrepancies and where they were tracked._

---

## Phase 4: {{PHASE_4_TITLE}}
<!-- AGENT: First domain feature surface. e.g. "Flow CRUD UI and Canvas Editor",
     "Document Editor", "Inventory Management UI". SHIPS UI. -->

**Status:** not started
**Dependencies:** Phase 3
**Deliverable:** {{PHASE_4_DELIVERABLE}}

### Tasks

- [ ] {{DOMAIN_TASK_1}}
- [ ] {{DOMAIN_TASK_2}}
- [ ] {{DOMAIN_TASK_3}}
- [ ] Implement create / edit / save / validate / publish flows for the primary resource.
- [ ] Implement empty states, loading states, and error states per the design doc.
- [ ] Implement keyboard navigation per the accessibility spec.
- [ ] Wire the command palette to context-relevant commands.

### Acceptance

- [ ] User can create, edit, save, and {{PRIMARY_VERB}} the primary resource end-to-end.
- [ ] All Phase 1 validator errors render visibly on the affected UI elements.
- [ ] Saved resources round-trip correctly: written via UI, fetched by API, contents match.
- [ ] Validation runs client-side on every edit and server-side on save; outputs match.
- [ ] Keyboard-only users can complete the primary flow.
- [ ] Visual parity with the design mocks at the target breakpoint.
- [ ] **CI gate:** load the `agent-ci` skill, then run `agent-ci run --quiet --all`; it is green.
- [ ] **Hygiene gate:** lint / format / typecheck clean.
- [ ] **Test gate:** unit tests for new behavior; integration tests for client↔API; e2e test for the primary flow.
- [ ] **Visual validation gate:** load the `agent-browser` skill, then load the primary screen in `agent-browser` in every documented state. Observations recorded in Handoff.
- [ ] **Phase boundary invariant:** clean clone → install → test succeeds.

### Handoff

_Filled at phase completion._

---

## Phase 5: {{PHASE_5_TITLE}}
<!-- AGENT: Background work / async / queues. e.g. "Runtime Worker", "Job Processor",
     "Indexer". Skip this phase if your app has no async work. -->

**Status:** not started
**Dependencies:** Phase 2
**Deliverable:** {{PHASE_5_DELIVERABLE}}

### Tasks

- [ ] Scaffold the worker process.
- [ ] Define the queue topology and document the message shapes.
- [ ] Implement the executor / processor and persistence of results.
- [ ] Implement per-job retry policy with backoff.
- [ ] Implement timeout enforcement.
- [ ] Implement structured logging with correlation IDs.
- [ ] Add a CLI entry point for local testing.
- [ ] Write integration tests covering happy path, retry, and failure modes.

### Acceptance

- [ ] `{{WORKER_DEV_CMD}}` starts the worker; the CLI runs a fixture job end-to-end.
- [ ] A deliberately failing job retries per its config and ultimately fails with a clear error.
- [ ] Logs are structured, queryable, and contain no secrets or PII.
- [ ] CI integration test passes, exercising the full queue → worker → DB path.
- [ ] **CI gate:** load the `agent-ci` skill, then run `agent-ci run --quiet --all`; it is green.
- [ ] **Hygiene gate:** lint / format / typecheck clean.
- [ ] **Test gate:** integration tests for happy / retry / timeout / poison-message paths.
- [ ] **Phase boundary invariant:** clean clone → install → test succeeds.

### Handoff

_Filled at phase completion._

---

## Phase 6: {{PHASE_6_TITLE}}
<!-- AGENT: Realtime / streaming. e.g. "Realtime and Run Inspector",
     "Live collaboration", "Stream UI". SHIPS UI. -->

**Status:** not started
**Dependencies:** {{PHASE_6_DEPS}}
**Deliverable:** {{PHASE_6_DELIVERABLE}}

### Tasks

- [ ] Implement the realtime transport ({{TRANSPORT}}).
- [ ] Implement client-side subscription with auto-reconnect and backoff; surface connection state in the UI.
- [ ] Wire UI animations / live updates to the event stream.
- [ ] Build the live-inspection UI with whatever tabs / sections are needed.
- [ ] Implement empty states for the live surface.

### Acceptance

- [ ] Triggering an event shows live UI updates without manual refresh.
- [ ] Closing and reopening the page shows the same final state for a completed event.
- [ ] Disconnecting mid-stream shows a connection-lost indicator; reconnecting resumes streaming.
- [ ] Visual parity with the relevant design mocks.
- [ ] **CI gate:** load the `agent-ci` skill, then run `agent-ci run --quiet --all`; it is green.
- [ ] **Hygiene gate:** lint / format / typecheck clean.
- [ ] **Test gate:** unit tests for the subscription state machine; integration test for transport reconnect.
- [ ] **Visual validation gate:** load the `agent-browser` skill, then load the live surface in `agent-browser`; reconnect indicator and empty state observed; observations recorded in Handoff.
- [ ] **Phase boundary invariant:** clean clone → install → test succeeds.

### Handoff

_Filled at phase completion._

---

## Phase 7: {{PHASE_7_TITLE}}
<!-- AGENT: Secrets / external auth. e.g. "Connections and Secret Storage",
     "OAuth Integration". Skip if your app has no external auth or stored secrets. -->

**Status:** not started
**Dependencies:** Phase 2
**Deliverable:** {{PHASE_7_DELIVERABLE}}

### Tasks

- [ ] Implement the secret-storage abstraction with versioned keys and {{ENCRYPTION_ALGORITHM}}.
- [ ] Document the key-rotation strategy.
- [ ] Implement OAuth flow for at least one provider.
- [ ] Implement API-key style auth for providers that don't support OAuth.
- [ ] Build the connection list / detail / add / consent UI.
- [ ] Implement runtime resolution: workers / handlers acquire secrets via a typed accessor that throws on inactive / unknown connections.
- [ ] Wire connection health checks on an interval; surface status changes via audit events.
- [ ] Add audit events for every meaningful connection lifecycle action.

### Acceptance

- [ ] User can connect an external account via OAuth and see it appear with status `active`.
- [ ] No API endpoint, log line, or audit event exposes a raw secret.
- [ ] Revoking a connection causes downstream consumers to fail with a clear, typed error rather than silently using a stale token.
- [ ] **CI gate:** load the `agent-ci` skill, then run `agent-ci run --quiet --all`; it is green.
- [ ] **Hygiene gate:** lint / format / typecheck clean; secret-shaped fields redacted in every log path.
- [ ] **Test gate:** unit tests for the secret abstraction; integration test for OAuth round-trip with a mock provider.
- [ ] **Visual validation gate:** load the `agent-browser` skill, then load connection add / list / detail / consent screens in `agent-browser`; observations recorded in Handoff.
- [ ] **Phase boundary invariant:** clean clone → install → test succeeds.

### Handoff

_Filled at phase completion._

---

## Phase 8: {{PHASE_8_TITLE}}
<!-- AGENT: Integrations / adapters. This phase is INTERNALLY PARALLELIZABLE —
     each integration is an independent track. Skip if your app has no third-party
     integrations. -->

**Status:** not started
**Dependencies:** Phase 5, Phase 7
**Deliverable:** {{PHASE_8_DELIVERABLE}}

This phase is **internally parallelizable** — each integration is an independent track that can be picked up by a separate agent once the dependencies are complete.

### Tasks (each integration is its own sub-track)

#### 8a — {{INTEGRATION_A}}

- [ ] Scaffold the integration package with auth setup guide.
- [ ] Implement the action / trigger handlers.
- [ ] Capability tags applied per action.
- [ ] Test fixtures and mocks (no real external calls in CI).

#### 8b — {{INTEGRATION_B}}

- [ ] Scaffold the integration package.
- [ ] Implement the action / trigger handlers.
- [ ] Capability tags applied per action.
- [ ] Test fixtures and mocks.

<!-- AGENT: Add 8c, 8d, etc. as needed. -->

### Acceptance

- [ ] Each integration's actions / triggers are visible in the UI after the user adds the corresponding connection.
- [ ] A flow / job using each integration succeeds end-to-end against a live test account (gated by env var, never run in CI).
- [ ] CI tests for each integration use mocks and pass deterministically.
- [ ] Capability tags on every integration action match the actual side effects.
- [ ] **CI gate:** load the `agent-ci` skill, then run `agent-ci run --quiet --all`; it is green.
- [ ] **Hygiene gate:** lint / format / typecheck clean across every sub-track.
- [ ] **Test gate:** mocked integration tests for every action / trigger; live-account tests behind env-var gate.
- [ ] **Visual validation gate (where the integration adds UI surfaces):** load the `agent-browser` skill, then load every new surface in `agent-browser`; observations recorded in Handoff.
- [ ] **Phase boundary invariant:** clean clone → install → test succeeds.

### Handoff

_Filled at phase completion._

---

## Phase 9: {{PHASE_9_TITLE}}
<!-- AGENT: AI / agent layer / model integration. Only include if applicable. -->

**Status:** not started
**Dependencies:** {{PHASE_9_DEPS}}
**Deliverable:** {{PHASE_9_DELIVERABLE}}

### Tasks

- [ ] {{AGENT_TASK_LIST}}

### Acceptance

- [ ] {{AGENT_ACCEPTANCE_LIST}}
- [ ] **CI gate:** load the `agent-ci` skill, then run `agent-ci run --quiet --all`; it is green.
- [ ] **Hygiene gate:** lint / format / typecheck clean.
- [ ] **Test gate:** unit tests for tool implementations; integration test for the model-call boundary with a recorded fixture.
- [ ] **Phase boundary invariant:** clean clone → install → test succeeds.

### Handoff

_Filled at phase completion._

---

## Phase 10: {{PHASE_10_TITLE}}
<!-- AGENT: Policy / approvals / audit. e.g. "Approvals and Audit". -->

**Status:** not started
**Dependencies:** {{PHASE_10_DEPS}}
**Deliverable:** {{PHASE_10_DELIVERABLE}}

### Tasks

- [ ] Implement the policy engine.
- [ ] Persist policy-decision records with their lifecycle states.
- [ ] Wire approvals into the runtime so gated operations block until decided.
- [ ] Build the approval UI surfaces (inline, queue, modal escalation).
- [ ] Implement audit-event emission for every meaningful action.
- [ ] Build the Audit Log UI with filters and a detail drawer.
- [ ] Implement timeout fallback per policy.
- [ ] Add notification surfaces (in-app, optionally email/push).

### Acceptance

- [ ] {{ACCEPTANCE_GATED_ACTION}} triggers approval; user can approve in the UI; the action resumes.
- [ ] An action explicitly denied by policy is denied unless an explicit override is set.
- [ ] Audit log shows every approval decision with full context.
- [ ] Policy composition behaves as specified.
- [ ] **CI gate:** load the `agent-ci` skill, then run `agent-ci run --quiet --all`; it is green.
- [ ] **Hygiene gate:** lint / format / typecheck clean.
- [ ] **Test gate:** unit tests for the policy engine; integration tests for the approval lifecycle; e2e test for the gated-action flow.
- [ ] **Visual validation gate:** load the `agent-browser` skill, then load approval queue / inline approval / audit log in `agent-browser`; observations recorded in Handoff.
- [ ] **Phase boundary invariant:** clean clone → install → test succeeds.

### Handoff

_Filled at phase completion._

---

## Phase 11: {{PHASE_11_TITLE}}
<!-- AGENT: Advanced / domain-specific later work. Customize heavily. -->

**Status:** not started
**Dependencies:** {{PHASE_11_DEPS}}
**Deliverable:** {{PHASE_11_DELIVERABLE}}

### Tasks

- [ ] {{ADVANCED_TASKS}}

### Acceptance

- [ ] {{ADVANCED_ACCEPTANCE}}
- [ ] **CI gate:** load the `agent-ci` skill, then run `agent-ci run --quiet --all`; it is green.
- [ ] **Hygiene gate:** lint / format / typecheck clean.
- [ ] **Test gate:** test pyramid honored for new behavior.
- [ ] **Visual validation gate (if UI):** load the `agent-browser` skill, then observe new surfaces in `agent-browser`.
- [ ] **Phase boundary invariant:** clean clone → install → test succeeds.

### Handoff

_Filled at phase completion._

---

## Phase 12: {{PHASE_12_TITLE}}
<!-- AGENT: Onboarding & canonical demo. Stitch together everything built so far
     into the one story a new user follows on day one. SHIPS UI. -->

**Status:** not started
**Dependencies:** {{PHASE_12_DEPS}}
**Deliverable:** A new user can sign up, complete onboarding, and run the canonical demo: {{DEMO_DESCRIPTION}}.

### Tasks

- [ ] Build the templates / examples gallery (if applicable).
- [ ] Author the canonical demo content.
- [ ] Author a sample fixture and an end-to-end test asserting the demo with mocks.
- [ ] Build the onboarding flow: welcome → workspace → optional invite → optional connect-an-integration → first action.
- [ ] Wire the in-app "Use this" path that takes a user from gallery → instantiated, ready-to-run resource.

### Acceptance

- [ ] A new user signs up, lands in onboarding, picks the demo, and runs it end-to-end against a sandbox.
- [ ] Each step in the demo shows correctly in the relevant inspector / live UI.
- [ ] Visual parity with the onboarding mocks.
- [ ] **CI gate:** load the `agent-ci` skill, then run `agent-ci run --quiet --all`; it is green.
- [ ] **Hygiene gate:** lint / format / typecheck clean.
- [ ] **Test gate:** the canonical-demo e2e test runs in CI with mocks; documented manual recipe runs against live deps.
- [ ] **Visual validation gate:** load the `agent-browser` skill, then load every onboarding step in `agent-browser`; canonical-demo flow walked end-to-end; observations recorded in Handoff.
- [ ] **Phase boundary invariant:** clean clone → install → test succeeds.

### Handoff

_Filled at phase completion._

---

## Phase 13: Polish — Light Mode, Responsive, A11y, States

**Status:** not started
**Dependencies:** Phase 12
**Deliverable:** Every surface has full light-mode parity, responsive behavior across the documented breakpoints, WCAG AA accessibility baseline, and considered empty / loading / error states. Performance is measured and within targets.

### Tasks

- [ ] Audit every route in light mode; fix any token leaks (hardcoded colors, dark-only assumptions).
- [ ] Audit every route at the documented breakpoints; verify nav collapses, drawers open, tables collapse extra columns.
- [ ] Implement read-only views at the smallest breakpoints if the design calls for it.
- [ ] Run axe-core across every authenticated route; resolve every violation.
- [ ] Verify keyboard navigation across the primary flows.
- [ ] Verify screen-reader announcements match the design doc spec.
- [ ] Implement every empty state per the design doc: each has icon, headline, supporting sentence, primary action.
- [ ] Implement every loading state: skeletons for known shapes, spinners only for unknown durations.
- [ ] Implement every error state: inline for resolvable, toast for transient, page-level for unrecoverable.
- [ ] Performance pass: {{PERFORMANCE_TARGETS}}.
- [ ] Add Lighthouse / WebPageTest baselines and document targets.
- [ ] Pass `prefers-reduced-motion` audit: every motion treatment has a static fallback.
- [ ] Pass `prefers-reduced-transparency` audit (if applicable).

### Acceptance

- [ ] Every route renders correctly in light mode with no token leaks.
- [ ] Axe-core reports 0 violations on every route.
- [ ] Keyboard-only flow: a user can complete the canonical demo without touching a mouse.
- [ ] Documented performance targets are met locally and in CI synthetic tests.
- [ ] **CI gate:** load the `agent-ci` skill, then run `agent-ci run --quiet --all`; it is green.
- [ ] **Hygiene gate:** lint / format / typecheck clean.
- [ ] **Test gate:** added tests for new empty / loading / error states.
- [ ] **Visual validation gate:** load the `agent-browser` skill, then load every authenticated route in `agent-browser` in light mode AND dark mode AND every documented breakpoint. Observations recorded in Handoff.
- [ ] **Phase boundary invariant:** clean clone → install → test succeeds.

### Handoff

_Filled at phase completion._

---

## Phase 14: Ship — Deployment, Observability, Docs

**Status:** not started
**Dependencies:** Phase 13
**Deliverable:** The platform is deployable end-to-end. Production deployment is documented. Observability (logs, metrics, traces) is wired. The README, CONTRIBUTING, ADRs, and authoring docs are complete.

### Tasks

- [ ] Author production Dockerfile(s) for each deployable component. Multi-stage builds; non-root user; healthcheck instructions.
- [ ] Author `docker-compose.prod.yml` (or equivalent) for self-hosted deployments.
- [ ] Document a hosted deployment recipe ({{HOSTING_TARGET}} — pick one and write it).
- [ ] Wire tracing (OpenTelemetry / equivalent) and document the local collector setup.
- [ ] Add structured logging with request IDs threaded through to background jobs and audit events.
- [ ] Add error tracking; ensure no PII / secret leaks in error payloads.
- [ ] Add basic metrics for the operationally important counters / gauges.
- [ ] Complete `README.md`: architecture overview, screenshots, quickstart, contribution links.
- [ ] Author `docs/architecture/overview.md` synthesizing the canonical doc(s) for new contributors.
- [ ] Author authoring docs for whoever extends the system (e.g., new integrations).
- [ ] Author `docs/security/` covering threat model, secret handling, policy semantics.
- [ ] Final ADR sweep: every resolved decision in the canonical doc has a corresponding ADR file.
- [ ] Author `CHANGELOG.md` with the v0.1 release notes.

### Acceptance

- [ ] A new contributor following only `README.md` + `docs/` can deploy a working instance.
- [ ] Tracing shows the path from request → background job → side effect.
- [ ] Errors surface in the chosen tracker with redacted payloads.
- [ ] CHANGELOG describes v0.1 honestly.
- [ ] **CI gate:** load the `agent-ci` skill, then run `agent-ci run --quiet --all`; it is green.
- [ ] **Hygiene gate:** lint / format / typecheck clean.
- [ ] **Test gate:** smoke-test against a deployed instance from CI (or a documented manual recipe).
- [ ] **Visual validation gate:** load the `agent-browser` skill, then walk the deployed instance in `agent-browser` against the canonical demo; observations recorded in Handoff.
- [ ] **Phase boundary invariant:** clean clone → install → test succeeds.

### Handoff

_Filled at phase completion. This is the last handoff._

<!-- ============================================================
     EXAMPLE PHASES END
     ============================================================ -->

---

## Cross-Phase Concerns

These are properties the project MUST maintain across all phases. They are not phase tasks; they are invariants.

<!-- AGENT: Author 5–10 invariants specific to your project. The examples below
     are GENERIC patterns adapted from common project shapes; replace with what
     actually matters for your codebase. Drop any that don't apply. -->

- **The codebase MUST remain runnable at every phase boundary.** A clean `{{INSTALL_CMD}} && {{TEST_CMD}} && {{DEV_CMD}}` succeeds at the end of every phase.
- **Schema changes MUST land in {{SCHEMA_PACKAGE}} first** with a migration; downstream services consume the migrated types.
- **Every API endpoint that mutates state MUST emit an audit event** with a resolved actor.
- **Every secret-handling code path MUST go through the secret-storage abstraction.** Direct DB reads of secret tables are forbidden outside the store implementation.
- **Every UI surface that renders user-supplied content MUST be validated against the relevant schema** before render.
- **Every integration node / handler MUST declare every capability tag it consumes.** Undeclared side effects are bugs.
- **Every long-running operation MUST stream progress** via the established event mechanism, not block the request.
- **Every change to {{CANONICAL_DOC}} MUST trigger a corresponding ADR or doc update**, and any tasks here that conflict with the change MUST be updated in the same PR.
- **Every PR MUST load the `agent-ci` skill and pass `agent-ci run --quiet --all` locally before being opened.** A red CI is a stop, not a footnote.
- **Every UI change MUST load the `agent-browser` skill and be browser-validated via `agent-browser` before the phase closes.** Observations live in the phase Handoff.

---

## Project completion

The project is **v0.1 complete** when:

- [ ] Phases 0 through {{LAST_PHASE_NUMBER}} are all marked complete.
- [ ] {{CANONICAL_DEMO_REF}} runs end-to-end against real dependencies in a deployed instance.
- [ ] CI is green, including the e2e demo test.
- [ ] All ADRs corresponding to the canonical doc's resolved decisions are checked in.
- [ ] A v0.1 git tag exists.
- [ ] `CHANGELOG.md` documents v0.1.
