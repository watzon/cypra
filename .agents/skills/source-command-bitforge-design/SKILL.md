---
name: "source-command-bitforge-design"
description: "Produce or refine DESIGN.md (visual design specification) from PLAN.md"
---

# source-command-bitforge-design

Use this skill when the user asks to run `$source-command-bitforge-design` or the legacy `/bitforge:design` command.

## Command Template


Read `.agents/skills/bitforge/design.md` and follow it end-to-end.

Pre-flight:
- Verify `PLAN.md` exists at the project root and is sealed. If not, stop and route the user to `/bitforge:plan`.
- If `PLAN.md` indicates this project is non-visual (CLI, library, MCP server, infrastructure), tell the user "PLAN.md indicates this project has no UI surface. Skipping DESIGN.md and going straight to `/bitforge:taskify`." and stop.
- If `DESIGN.md` already exists and is sealed, ask whether this is a revision or a re-derivation.

Then execute Steps 1–5 of the design skill: read PLAN, interview on aesthetic direction, author DESIGN.md against the required structure, run the adversarial review, and seal.

User's input (if any): $ARGUMENTS
