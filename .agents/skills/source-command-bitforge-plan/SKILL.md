---
name: "source-command-bitforge-plan"
description: "Produce or refine PLAN.md (PRD + SDD) from BRAINSTORM.md"
---

# source-command-bitforge-plan

Use this skill when the user asks to run `$source-command-bitforge-plan` or the legacy `/bitforge:plan` command.

## Command Template


Read `.agents/skills/bitforge/plan.md` and follow it end-to-end.

Pre-flight:
- Verify `BRAINSTORM.md` exists at the project root and is sealed. If it is draft, stop and tell the user to seal it (or run `/bitforge:brainstorm` to finish).
- If `PLAN.md` already exists and is sealed, ask whether this is a revision (append a Revisions section) or a re-derivation (overwrite with explicit user approval).

Then execute Steps 1–6 of the plan skill: read BRAINSTORM, resolve open questions, author PLAN.md against the required structure, run the adversarial review, do the internal-consistency pass, and seal.

User's input (if any): $ARGUMENTS
