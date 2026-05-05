---
name: "source-command-bitforge-brainstorm"
description: "Start the bitforge brainstorm phase - produce or refine BRAINSTORM.md"
---

# source-command-bitforge-brainstorm

Use this skill when the user asks to run `$source-command-bitforge-brainstorm` or the legacy `/bitforge:brainstorm` command.

## Command Template


Read `.agents/skills/bitforge/brainstorm.md` and follow it end-to-end.

If `BRAINSTORM.md` already exists at the project root:
- If it's marked `Status: sealed`, ask the user whether to open it for revision (a Revisions section gets appended) or move on to `/bitforge:plan`.
- If it's draft, continue from where it left off — re-read it, identify unanswered sections / open questions, and resume the interview there.

If no `BRAINSTORM.md` exists, start from Step 1 of the brainstorm skill: frame the conversation, then walk the question grid.

User's seed prompt (if any): $ARGUMENTS
