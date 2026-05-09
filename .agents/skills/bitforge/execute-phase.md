# bitforge — Execute Phase

**Goal:** pick up the next unstarted (or in-progress) phase from `TASKS.md` and drive it to *complete* — every task ticked, every acceptance gate green, the Handoff block filled, the file committed.

**Inputs:** `TASKS.md`, `PLAN.md`, `DESIGN.md` (if applicable).

**Output:** modified code + tests + docs, updated `TASKS.md`, optionally a PR.

**Invoked by:** `$source-command-bitforge-next` (start / resume) and `$source-command-bitforge-close-phase` (closure-only path).

**Reading order:** read this file, then [`standards.md`](./standards.md), then the relevant phase in `TASKS.md`.

---

## Operating rules — these are binding

1. **You MUST treat TASKS.md / PLAN.md / DESIGN.md as the source of truth.** A conflict between code and any of them means the doc wins, unless the user explicitly approves a doc change first.
2. **You MUST keep TASKS.md current.** Mark tasks in-progress when you start them and complete when you finish them. Update the phase Status line.
3. **You MUST mirror the phase task list with OpenCode's todo tool.** Both lists end the phase ticked. The internal list is a copy of the file list, not a different list.
4. **You MUST run every Acceptance gate before marking the phase complete.** A green CI run is necessary but not sufficient — every Acceptance criterion is binding.
5. **You MUST load the `agent-ci` skill before running the CI gate.** Then run the command the skill prescribes for `agent-ci run --quiet --all`. Do not roll your own.
6. **For any phase that ships UI, you MUST load the `agent-browser` skill before browser validation.** Then use `agent-browser` against the running app and record what you saw in the Handoff block.
7. **You MUST fill the Handoff block before closing the phase.** Free-form prose is fine; emptiness is not.
8. **You MUST NOT skip phases or tasks.** If a task is impossible, surface why and add a new task or new phase.
9. **You MUST NOT silently change architecture or visual tokens to make a task easier.** Raise the conflict; update PLAN/DESIGN explicitly; then update tasks.
10. **You MUST stop on user-impacting risk** (destructive migration, secret rotation, prod data) and ask before proceeding.

---

## Process

### Step 1 — Select the phase

1. Read `TASKS.md` top-to-bottom.
2. Find the first phase whose Status is `in progress`. If none, find the first whose Status is `not started` and whose Dependencies are all `complete`.
3. If no phase is ready, tell the user which phase is blocked and why. Stop.

### Step 2 — Pre-flight

Before any code change:

- Verify the working tree is clean **and** on the right branch. Standard branch hygiene applies: if you're on a stale feature branch, switch to `main` (or the repo's default), pull, and create a fresh branch unless the user explicitly says to continue on the current one. If the workspace or repo has its own branch-hygiene rules in `AGENTS.md` (or legacy `CLAUDE.md`), those win.
- Read the relevant slice of PLAN.md and (if visual) DESIGN.md. The phase Tasks should make sense against them.
- Read the prior phase's Handoff block if it exists. It will tell you what's already done and what was deferred.
- Load the `agent-ci` skill, then run the skill-prescribed `agent-ci run --quiet --all` command to confirm the codebase is green *before* you change anything. CI was green when the prior phase closed. If it isn't now, fix that first — it's not your phase's problem yet, but it will be.

### Step 3 — Open the phase

In `TASKS.md`:
- Set the phase Status from `not started` → `in progress`.
- Stage no other change.

In your internal session:
- Mirror the entire phase Tasks list with OpenCode's todo tool. Same wording where possible.
- Set the first task's status to `in_progress`.

### Step 4 — Execute tasks one at a time

For each task, in order:

1. Set the task in your internal todo to `in_progress`.
2. Mark the corresponding line in `TASKS.md` from `- [ ]` to `- [~]` (in-progress marker).
3. Do the work. Keep edits scoped to the task; surface scope drift as a new task rather than silently expanding this one.
4. Write tests as you go (unit always; integration where boundaries cross). Tests live next to the code they cover.
5. Run the relevant local check after each meaningful edit (typecheck, the targeted test file). Fix what breaks before moving on.
6. Mark the task `- [x]` in `TASKS.md` and `completed` internally. Do not batch.

### Step 5 — Hygiene pass before closure

Before running the Acceptance gates:

- Lint and format clean (run the project's linter / formatter; document choice lives in CONTRIBUTING.md / repo guidance).
- Typecheck clean.
- No newly-introduced dead code. If you wrote a helper for one caller and that caller is the only one, the helper is fine — abstractions are earned by ≥3 callers, not by speculation.
- No `console.log` / `print` debug breadcrumbs left.
- No `// TODO` for things you actually finished.
- DRY where it makes sense, not preemptively. Three similar lines is better than a premature abstraction.

### Step 6 — Run the Acceptance gates

For each Acceptance line in the phase:

- **CI gate** — load the `agent-ci` skill, then run the skill-prescribed `agent-ci run --quiet --all` command. It returns green. Quote the exit status in the Handoff.
- **Hygiene gate** — record the lint/format/typecheck outputs (or "clean").
- **Test gate** — record the test run output. New tests are visible in it.
- **Visual validation gate (UI phases)** — load the `agent-browser` skill, then run `agent-browser`. For each screen the phase touched:
  - Load the URL in the running dev server.
  - Capture an accessibility-tree snapshot or screenshot.
  - Compare against DESIGN.md: token usage, states, copy, focus rings, motion. Note discrepancies.
  - Run axe-core (or equivalent) on the page; resolve violations or add them as a task on the next phase if they are out of scope.
- **Domain-specific gates** — every other Acceptance line. Check each one literally.

A failing gate is a stop, not a footnote. Either fix it inside this phase or create a new phase / task for the user to approve.

### Step 7 — Fill the Handoff block

The Handoff is the single most valuable artifact for the next agent. Free-form, but useful Handoffs include:

- What landed (1–2 sentences).
- Deviations from the task list (if any) and why.
- Surprises and gotchas — anything the next agent will hit head-first.
- Deferrals — things you intentionally did not do, with the reason and where they went (next phase, follow-up task, separate ticket).
- Visual-validation observations (for UI phases): "Loaded /dashboard in agent-browser; tokens match; loading state matches DESIGN §10.2; empty state copy currently differs from §10.2 — opened follow-up task in Phase 4."
- CI / test / hygiene status line ("agent-ci green; 47 unit + 12 integration tests; typecheck clean").

### Step 8 — Close the phase

1. Set the phase Status from `in progress` → `complete`.
2. Confirm every `- [ ]` and `- [~]` in the phase is now `- [x]` (Tasks and Acceptance).
3. Confirm the Handoff block is filled.
4. Stage `TASKS.md` plus all code changes.
5. Commit per the workspace / repo conventions (any rules captured in `AGENTS.md`, legacy `CLAUDE.md`, or repo `CONTRIBUTING.md` — issue-tracker sync, PR description format, etc.). The commit subject names the phase: `Phase N: <title> — complete`.
6. If the user asks for a PR, open it using whatever PR template / format the repo or workspace prescribes. If the workspace links commits/PRs to an issue tracker (Linear, Jira, GitHub Issues, etc.), include the appropriate reference line for any ticket that applies.
7. Tell the user:

> Phase N (`<title>`) is complete. Next ready phase is Phase N+1 (`<next title>`). Run `$source-command-bitforge-next` when ready.

### Closure-only mode (`$source-command-bitforge-close-phase`)

When invoked through `$source-command-bitforge-close-phase`, skip Steps 3–4 and start at Step 5 (Hygiene). This is the path for "I did the work outside the agent; verify and close it". Refuse to close if any Acceptance gate is unverified — re-run them yourself.

---

## Failure modes

- **A task turns out to be impossible as written.** Surface it. Add a new task in this phase (if scope-appropriate) or a new phase (if architectural). Do not silently work around it.
- **Acceptance gate fails after the work looks done.** That is the gate doing its job. Fix the underlying code; do not weaken the gate.
- **CI fails for unrelated reasons.** Stop. Tell the user. Either fix the unrelated breakage as a separate task in this phase or escalate.
- **PLAN.md / DESIGN.md are wrong.** Surface it. Update the canonical doc with the user's approval. Update tasks. Then continue.
- **You realize the phase is mis-scoped.** Stop. Pull the user in. Re-shape the phase before continuing.

---

## Anti-patterns

- **Batch-completing the task list at the end.** You will lose track of what shipped. Tick as you go.
- **Treating CI green as enough.** It isn't. Acceptance is the binding test.
- **Filling the Handoff with "All tasks complete."** That tells the next agent nothing they couldn't read from the checkboxes.
- **Editing PLAN.md or DESIGN.md silently to make a task pass.** Always surface and approve first.
- **Marking the phase complete while a UI page hasn't been loaded in a browser.** No.
