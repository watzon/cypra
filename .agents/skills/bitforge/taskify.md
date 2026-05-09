# bitforge — Taskify Phase

**Goal:** produce `TASKS.md` — a fully concrete, ordered, multi-phase execution plan derived from PLAN.md (and DESIGN.md if applicable). Every `{{PLACEHOLDER}}` in the template gets replaced. Every phase has a CI gate, hygiene gate, and (where visual) browser-validation gate baked into Acceptance.

**Inputs:** `PLAN.md` (sealed). `DESIGN.md` (sealed, if applicable). `bitforge/templates/TASKS.md`.

**Output artifact:** `TASKS.md` at the project root.

**Reading order:** read this file, then [`standards.md`](./standards.md), then [`templates/TASKS.md`](./templates/TASKS.md).

---

## Operating rules

1. **You MUST start from `bitforge/templates/TASKS.md`.** Copy it to project-root `TASKS.md`. Do not author from a blank file.
2. **You MUST replace every `{{PLACEHOLDER}}` and every `<!-- AGENT: ... -->` comment.** Unreplaced placeholders are bugs.
3. **You MUST tailor phases to the project, not the template.** The template's example phases are a *shape*, not a contract. Add, remove, reorder, and rename phases so the project remains runnable at every phase boundary.
4. **You MUST honor the standards** in [`standards.md`](./standards.md) — every phase Acceptance includes the CI gate, the hygiene gate, and (for any phase that ships UI) the browser-validation gate.
5. **You MUST size tasks honestly.** 6–14 tasks per phase. Fewer means tasks are too coarse; more means too fine. Surface long tasks as their own phase.
6. **You MUST mark internal parallelism explicitly.** Independent integrations / modules within a phase get their own sub-track headings.
7. **You SHOULD spawn an adversarial review sub-agent.**
8. **You MUST NOT delete the operating-rules block at the top of TASKS.md.** It is binding for every agent that picks up `$source-command-bitforge-next`.

---

## Process

### Step 1 — Verify inputs

- `PLAN.md` exists and is sealed.
- `DESIGN.md` either exists and is sealed, or PLAN.md states the project is non-visual.
- `bitforge/templates/TASKS.md` is reachable in the skill bundle.

If any input is missing or in draft, stop and tell the user.

### Step 2 — Copy the template

Copy `bitforge/templates/TASKS.md` to `TASKS.md` at project root.

### Step 3 — Fill the front matter

- `{{PROJECT_NAME}}` → from PLAN §1.
- `{{COMPANION_DOCS}}` → list every canonical doc (`PLAN.md`, `DESIGN.md` when applicable, `BRAINSTORM.md` for archive reference).
- `{{ONE_SENTENCE_PROJECT_OUTCOME}}` → derived from PLAN's MVP definition. Make it concrete enough that a stranger can read it and know whether v1 shipped.
- `{{COMPANION_DOC_LIST}}` → "PLAN.md and DESIGN.md", "PLAN.md", or your equivalent.

### Step 4 — Plan the phase progression

Walk PLAN.md and DESIGN.md and lay out the phases. Use the template's example progression as a starting checklist. Add, drop, or reorder. The constraint that wins every argument: **the codebase must remain runnable at every phase boundary.**

A typical web-app progression:

| # | Phase | Owns |
|---|-------|------|
| 0 | Foundation | Repo bootable, CI green, license, contributor docs, env example |
| 1 | Schema & Types | Data model from PLAN §8, validators, fixtures, migration strategy |
| 2 | Backend Core | API surface from PLAN §9, auth, CRUD, health endpoints |
| 3 | Frontend Foundation | Tokens from DESIGN, primitives, routing, auth screens |
| 4 | Domain UI | First major feature surface |
| 5 | Async / Workers | Queues, jobs, scheduled work (if applicable) |
| 6 | Realtime | Streaming, WebSocket, live UI (if applicable) |
| 7 | Secrets / Connections | Secret store, OAuth, third-party auth (if applicable) |
| 8 | Integrations | Internally parallelizable: one sub-track per integration |
| 9 | AI / Agents | Model layer, agent runtime, tool surfaces (if applicable) |
| 10 | Approvals / Audit | Policy, gating, audit log (if applicable) |
| 11 | Advanced features | Domain-specific later work |
| 12 | Onboarding / Demo | The canonical end-to-end story from PLAN runs end-to-end |
| 13 | Polish | Light-mode parity, responsive, a11y, all states, performance |
| 14 | Ship | Deploy, observability, docs, release |

Trim aggressively. A non-visual CLI does not need Phases 3, 4, 6, 13 — and that is fine.

For each phase: title, dependencies, deliverable, 6–14 tasks, acceptance, empty Handoff block.

### Step 5 — Bake in the standards (every phase)

Every phase Acceptance MUST include — verbatim or in spirit — these gates. Read [`standards.md`](./standards.md) for the rationale.

```markdown
- [ ] **CI gate:** load the `agent-ci` skill, then run `agent-ci run --quiet --all`; it is green for the changes introduced in this phase.
- [ ] **Hygiene gate:** lint clean, format clean, typecheck clean. No introduced dead code. DRY where it makes sense; not preemptively.
- [ ] **Test gate:** unit tests cover new behavior; integration tests cover new boundary crossings; this phase's deliverable is exercised end-to-end where applicable.
```

For any phase that ships UI (touches a visual surface), add:

```markdown
- [ ] **Visual validation:** load the `agent-browser` skill, then load the relevant screen(s) in `agent-browser`; the agent records what it saw against DESIGN.md, and any token leaks / state gaps / a11y violations from axe-core are tracked.
```

For any phase that introduces user-visible flows (auth, onboarding, primary feature), the visual validation is **mandatory** and the recorded observations live in the Handoff block.

### Step 6 — Author the Cross-Phase Concerns

Replace the template's generic invariants with project-specific ones drawn from PLAN.md guiding principles (§4) and architecture (§7). Aim for 5–10. Examples that survive almost everywhere:

- The codebase remains runnable at every phase boundary.
- Schema changes land in the schema package first with a migration; downstream services consume migrated types.
- Every API endpoint that mutates state emits an audit event with a resolved actor.
- Every secret-handling code path goes through the secret abstraction. Direct DB reads of secret tables are forbidden.
- Every UI surface that renders user-supplied content validates against the relevant schema before render.
- Every long-running operation streams progress; nothing blocks a request thread.
- Every change to PLAN.md or DESIGN.md triggers a corresponding ADR or doc update; conflicting tasks are updated in the same PR.

### Step 7 — Author the project-completion checklist

Replace `{{LAST_PHASE_NUMBER}}` and `{{CANONICAL_DEMO_REF}}` with concrete values. The demo reference should point at a phase that proves PLAN's canonical user story end-to-end.

### Step 8 — Adversarial review

Spawn `Agent({subagent_type: "general-purpose"})` with:

> Review TASKS.md as a senior tech lead. Read TASKS.md, PLAN.md, and DESIGN.md (if it exists) at the project root. For each phase, list every place where:
> - a task is too vague to start tomorrow morning
> - a phase Acceptance fails to actually verify the Deliverable
> - a phase's Dependencies are wrong (missing or extra)
> - the codebase would not be runnable at this phase boundary
> - a feature listed in PLAN's MVP must-haves never appears in any phase
> - a screen in DESIGN.md never appears in any phase
> - a Cross-Phase Concern is violated by a task in some phase
> - the CI / Hygiene / Test / Visual-validation gates are missing or weakened
> - parallelism is claimed without independent sub-tracks
>
> Aim for 25+ findings. Do not propose alternatives — just name the gaps.

Update TASKS.md.

### Step 9 — Final sanity sweep

- Every must-have in PLAN §6 appears in at least one phase's Tasks.
- Every screen in DESIGN.md §10 appears in at least one phase's Tasks.
- Every integration in PLAN §7 appears in Phase 8 (or wherever integrations land) with its own sub-track.
- No `{{PLACEHOLDER}}` survives.
- No `<!-- AGENT: ... -->` comment survives.
- Phase numbering is contiguous.
- Every phase has Status / Dependencies / Deliverable / Tasks / Acceptance / Handoff. No bonus sections inside a phase.

### Step 10 — Commit and announce

Stage `TASKS.md`. Tell the user:

> TASKS.md is ready. The pipeline is now Brainstorm → Plan → Design → (Pencil) → Taskify, all complete. Run `$source-command-bitforge-next` to pick up Phase 0.

---

## Anti-patterns

- **Authoring TASKS.md from scratch.** Always copy `templates/TASKS.md` — it carries operating rules and standards baked in.
- **Generic tasks like "build the API".** That's a phase, not a task. The task is "implement the POST /flows route with Phase-1 schema validation, returning 201 with the created resource".
- **Skipping the visual validation gate** because "we'll catch it in Polish". You won't. Bake it in per phase.
- **Letting `{{PLACEHOLDER}}` survive.** Reviewers will find them; better that you do.
- **Cramming everything into one Phase 1.** Split until the codebase is runnable at every boundary.
