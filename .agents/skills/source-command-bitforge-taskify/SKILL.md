---
name: "source-command-bitforge-taskify"
description: "Produce TASKS.md from PLAN.md (and DESIGN.md) using the bitforge template"
---

# source-command-bitforge-taskify

Use this skill when the user asks to run `$source-command-bitforge-taskify` or the legacy `/bitforge:taskify` command.

## Command Template


Read `.agents/skills/bitforge/taskify.md` and follow it end-to-end.

Pre-flight:
- Verify `PLAN.md` exists and is sealed. If not, stop and route to `/bitforge:plan`.
- Verify `DESIGN.md` exists and is sealed if PLAN says the project is visual. If draft, route to `/bitforge:design`.
- Verify `.agents/skills/bitforge/templates/TASKS.md` is reachable.

Then execute Steps 1–10: copy the template, fill front matter, plan phase progression, fill each phase, bake in standards (CI / hygiene / test / visual validation gates), author cross-phase concerns and the project-completion checklist, run the adversarial review, do the final sanity sweep, commit, and announce.

User's input (if any): $ARGUMENTS
