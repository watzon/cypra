---
description: Pick up and execute the next phase from TASKS.md
---

Read `.agents/skills/bitforge/execute-phase.md` and follow it end-to-end. Also read `.agents/skills/bitforge/standards.md` for the cross-cutting non-negotiables.

Pre-flight:
- Verify `TASKS.md` exists at the project root. If not, route to `/bitforge:taskify`.
- Verify the working tree is clean and on the right branch. Standard branch hygiene applies: if on a stale feature branch, switch to `main` (or the repo's default), pull, and create a fresh branch unless the user explicitly says to continue. Workspace or repo rules in `AGENTS.md` (or legacy `CLAUDE.md`), if present, take precedence.
- Load the `agent-ci` skill, then run `agent-ci run --quiet --all` against the existing codebase to confirm green-before-edits. If red, surface to the user before starting.

Then execute Steps 1–8 of execute-phase: select the next ready phase, open it, mirror its task list with OpenCode's todo tool, work tasks one at a time (ticking both the internal todo and TASKS.md in lockstep), run the hygiene pass, run every Acceptance gate (CI / hygiene / test / visual validation / domain-specific), fill the Handoff block, mark the phase complete, and commit.

Stop and surface to the user on:
- destructive migrations
- secret rotations or production data touches
- any Acceptance gate that fails after the work looks done
- any conflict between PLAN.md / DESIGN.md and the work as written

User's input (if any): $ARGUMENTS
