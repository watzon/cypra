---
description: Verify acceptance, fill the Handoff, and mark the current phase complete
---

Read `.agents/skills/bitforge/execute-phase.md` (closure-only mode at the bottom) and `.agents/skills/bitforge/standards.md`.

Use this command when work for the current phase happened outside the agent (or across multiple sessions) and you need to verify, write the Handoff, and close the phase.

## Process

1. Read TASKS.md and identify the phase whose Status is `in progress`. If none, stop and tell the user.
2. Re-run every Acceptance gate yourself, even if a human says they passed:
   - **CI gate** — load the `agent-ci` skill, then run `agent-ci run --quiet --all`. Quote the result.
   - **Hygiene gate** — lint / format / typecheck across the touched packages.
   - **Test gate** — run the project's test suite; new tests should be visible.
   - **Visual validation gate** (UI phases) — load the `agent-browser` skill, then run `agent-browser` against every screen the phase touched, capture observations against DESIGN.md, run axe-core, record findings.
   - **Domain-specific gates** — every other line in this phase's Acceptance block. Verify each literally; do not assume.
3. Confirm every Tasks checkbox in this phase is `- [x]`. If any are still `- [ ]` or `- [~]`, refuse to close and route the user back to `/bitforge:next`.
4. Fill the **Handoff** block. Free-form prose, but include: what landed, deviations, surprises, deferrals, visual-validation observations, CI / test / hygiene status line.
5. Set the phase Status from `in progress` → `complete`.
6. Stage TASKS.md plus any code changes; commit per the workspace / repo conventions (rules in `AGENTS.md`, legacy `CLAUDE.md`, or `CONTRIBUTING.md`). Subject: `Phase N: <title> — complete`.
7. If the user asks for a PR, open it using whatever PR template the repo prescribes, including any issue-tracker reference line (Linear, Jira, GitHub Issues, etc.) if a ticket applies.
8. Tell the user which phase is next.

## Refuse to close if

- Any Acceptance gate fails or is unverified.
- Any Tasks checkbox is unticked.
- The Handoff block is empty.
- The working tree contains changes the user did not stage or that conflict with the phase's intended scope.

User's input (if any): $ARGUMENTS
