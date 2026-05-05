---
name: "source-command-bitforge-status"
description: "Show the current bitforge pipeline state - which artifacts exist, where the project is, what to do next"
---

# source-command-bitforge-status

Use this skill when the user asks to run `$source-command-bitforge-status` or the legacy `/bitforge:status` command.

## Command Template


Report the bitforge pipeline state for the project at the current working directory. Do not modify any files.

## Step 1 — Inventory artifacts

Check for and report on each:

- `BRAINSTORM.md` — exists? draft / sealed?
- `PLAN.md` — exists? draft / sealed?
- `DESIGN.md` — exists? draft / sealed? (or "n/a — non-visual project" if PLAN says so)
- `*.pen` — any Pencil files at the project root?
- `TASKS.md` — exists?

Render as a small table.

## Step 2 — If TASKS.md exists, summarize phase progress

For every phase in TASKS.md, report:
- Phase number and title
- Status (`not started` / `in progress` / `complete`)
- Task progress as `done / total` (count of `- [x]` over total checkboxes in the phase's Tasks block — exclude Acceptance and Handoff)
- For the in-progress phase only, list any tasks marked `- [~]`

## Step 3 — Recommend the next command

Based on state, suggest exactly one next step:

| State | Recommendation |
|-------|----------------|
| No BRAINSTORM.md | `/bitforge:brainstorm` |
| BRAINSTORM.md draft | `/bitforge:brainstorm` to continue |
| BRAINSTORM.md sealed, no PLAN.md | `/bitforge:plan` |
| PLAN.md draft | `/bitforge:plan` to continue |
| PLAN.md sealed, project is visual, no DESIGN.md | `/bitforge:design` |
| DESIGN.md sealed, no .pen file | `/bitforge:pencil` (optional) |
| All planning docs sealed, no TASKS.md | `/bitforge:taskify` |
| TASKS.md exists, a phase is in progress | `/bitforge:next` to continue, or `/bitforge:close-phase` if all tasks are ticked |
| TASKS.md exists, all phases complete | "v0.1 complete — verify the project-completion checklist at the bottom of TASKS.md" |

## Step 4 — Surface anomalies

If you detect any of these, name them:
- TASKS.md exists but PLAN.md does not (skipped phases).
- A phase marked complete with unticked Acceptance lines (incomplete close).
- Multiple phases marked `in progress` simultaneously (parallel violation).
- A `{{PLACEHOLDER}}` or `<!-- AGENT: ... -->` survives in TASKS.md (incomplete taskify).

User's input (if any): $ARGUMENTS
