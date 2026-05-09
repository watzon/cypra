# bitforge — Standards

These are the non-negotiables that apply across every phase of every bitforge project. They are encoded as Acceptance gates in [`TASKS.tmpl.md`](./TASKS.tmpl.md) and enforced by [`execute-phase.md`](./execute-phase.md).

---

## 1. CI is the gate

- **Tool:** load the `agent-ci` skill first, then run the command it prescribes for `agent-ci run --quiet --all`. Use `--pause-on-failure` when iterating locally.
- **Rule:** every phase's Acceptance includes a CI-green check. A red CI is a stop, not a footnote.
- **Rationale:** CI was green at the start of every phase (the previous phase's closure proved it). Any new red is caused by changes in this phase. Fix it before moving on.
- **Anti-pattern:** rolling your own one-off "I'll just run the tests" command. Use `agent-ci`. It runs the same pipeline GitHub Actions runs, so a green local result mirrors a green push.

## 2. Manual e2e testing

- **Tool:** load the `agent-browser` skill first, then use the command workflow it prescribes.
- **Rule:** any phase that ships UI MUST be browser-validated before the phase closes. Validation includes:
  - The new screens / surfaces load in a running dev server.
  - The agent captures an accessibility-tree snapshot (preferred) or screenshot of each surface in each documented state (default / loading / empty / error).
  - The agent compares observations against DESIGN.md and records discrepancies in the phase Handoff.
  - axe-core (or equivalent) reports zero new violations on touched routes.
- **Rationale:** unit tests do not catch token leaks, bad focus rings, missing empty states, copy regressions, or accessibility drift. A real browser does.
- **Anti-pattern:** "the screenshot looks right" without an accessibility-tree snapshot. Visual parity is necessary; semantic structure is also necessary.

## 3. Test pyramid

- **Unit (always):** every new function / module / component gets unit tests covering happy path and edge cases. Tests live next to the code (`Foo.ts` + `Foo.test.ts`, or the project's documented convention).
- **Integration (where boundaries cross):** API ↔ DB, worker ↔ queue, frontend ↔ API, integration adapter ↔ third-party (with mocks). One test per meaningful boundary.
- **End-to-end (UI / Ship phases):** the canonical user story from PLAN runs end-to-end against either a running dev stack or a deployed staging instance. Onboarding / Demo phase always has one. Ship phase always has one.
- **Rationale:** unit catches regressions cheaply; integration catches contract drift; e2e proves the demo still demos.
- **Anti-pattern:** writing only e2e and skipping unit because "the e2e covers it". E2es are slow and brittle; unit tests live forever.

## 4. Hygiene

Every phase ships a clean working tree. Before closing the phase:

- Lint clean (project linter — Biome, ESLint, ruff, golangci-lint, etc.).
- Format clean (project formatter).
- Typecheck clean (the language's strict-mode checker).
- No introduced dead code. Helpers earn their existence; speculative abstractions don't.
- DRY where it makes sense, not preemptively. Three similar lines is fine; introduce abstractions on the third use, not the first.
- No `console.log` / `print` / `dbg!` debug breadcrumbs.
- No `// TODO` for things you actually finished. Real TODOs reference an issue or a follow-up task.
- No commented-out code. If you wanted to keep it, git already did.

## 5. Internal todo discipline

Inside `$source-command-bitforge-next`:

- Mirror the phase's Tasks list with OpenCode's todo tool at the start.
- Set each task to `in_progress` before working on it.
- Mark each task `completed` immediately on finish — both in your internal list and in the `TASKS.md` file. Don't batch.
- Both lists end the phase ticked. Diverging lists mean something was missed.

## 6. Source-of-truth respect

- PLAN.md and DESIGN.md outrank TASKS.md.
- Conflicts surface to the user. Update the canonical doc explicitly with rationale. Then update tasks.
- Never silently rename schema fields, change visual tokens, or alter API shapes to make a task easier.

## 7. Phase boundary invariant

- The codebase is **runnable** at every phase boundary. A clean clone + install + test command succeeds at the end of every phase.
- This is non-negotiable. If a phase would break the boundary invariant, split it.

## 8. Documentation as you go

- Every package / module / app has a README that answers: what is this, who uses it, how do I run it locally, where are the tests.
- ADRs accompany every load-bearing decision in PLAN §10 (tech choices) and §7 (architecture). One ADR per decision.
- API docs (OpenAPI / typed SDK) are generated, not hand-maintained.
- DESIGN.md is updated when visual decisions change — never only in code.

## 9. Security defaults

- Secrets never live in the model context. Pass references / displayNames to agents; never raw values.
- Secrets never live in repo. `.env.example` is committed; `.env` is `.gitignore`d.
- API keys and tokens go through whatever secret-storage abstraction PLAN §11 specifies.
- Logs and audit events redact secret-shaped fields by default.
- Agents do not get write access to production data without explicit approval flows (PLAN §10 / Approvals phase, when applicable).

---

## Quick reference — every phase Acceptance ends with these

```markdown
- [ ] **CI gate:** the `agent-ci` skill was loaded, then `agent-ci run --quiet --all` was run and is green for changes introduced in this phase.
- [ ] **Hygiene gate:** lint clean, format clean, typecheck clean, no introduced dead code.
- [ ] **Test gate:** new behavior covered by unit tests; new boundary crossings covered by integration tests.
- [ ] **Visual validation (UI phases only):** the `agent-browser` skill was loaded, then every new / changed surface was loaded in `agent-browser`; observations recorded in Handoff; axe-core reports no new violations.
- [ ] **Phase boundary invariant:** clean clone → install → test succeeds.
```
