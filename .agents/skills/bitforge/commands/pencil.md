---
description: Emit a Pencil handoff prompt the user can paste into a Pencil session to wireframe the project
---

Read `.agents/skills/bitforge/pencil-handoff.md` and follow it end-to-end.

Pre-flight:
- Verify `DESIGN.md` exists at the project root and is sealed. If not, stop and route to `/bitforge:design`.

Then execute Steps 1–3: read DESIGN.md, customize the handoff prompt with the project name and screen-specific primitive list from §8 / §10, and print the prompt block as a single fenced markdown code-block the user can copy.

Do NOT call any pencil MCP tools from this session. The handoff is for a separate Pencil-aware session.

User's input (if any): $ARGUMENTS
