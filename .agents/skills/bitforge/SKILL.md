---
name: "bitforge"
description: "Use when the user wants to plan, design, or execute a multi-phase software project end-to-end. Covers the full Brainstorm -> Plan -> Design -> (Pencil wireframes) -> Taskify -> Execute pipeline that produces PLAN.md, DESIGN.md, and TASKS.md, then drives phased implementation with mandatory CI/test/visual validation gates. Triggers include 'let's plan X', 'I have an idea for...', 'design the system for...', 'break this into phases', 'pick up the next phase', 'what should we build next', or any of the $source-command-bitforge-* commands."
---

# bitforge - Phased Agentic Development

bitforge is a portable workflow for taking an idea from a sentence to a shipped product without leaving anything to the imagination. Drop this skill bundle into any workspace's `.agents/skills/` and run `bash .agents/skills/bitforge/install.sh` to wire up OpenCode source-command skills.

## The pipeline

```
   Brainstorm  ->  Plan  ->  Design  ->  <Pencil wireframes>  ->  Taskify  ->  Execute (phased)
       |           |         |              |                      |            |
   BRAINSTORM   PLAN.md   DESIGN.md     <project>.pen          TASKS.md     code + PRs
```

Four planning phases produce three durable docs. Then phased execution turns those docs into a working product, one phase at a time, with the codebase runnable at every phase boundary.

## When to invoke each phase

| Situation | Read | OpenCode command |
|-----------|------|------------------|
| User has an idea but no concrete spec | [`brainstorm.md`](./brainstorm.md) | `$source-command-bitforge-brainstorm` |
| BRAINSTORM.md exists; need a PRD+SDD | [`plan.md`](./plan.md) | `$source-command-bitforge-plan` |
| PLAN.md exists; project has a UI | [`design.md`](./design.md) | `$source-command-bitforge-design` |
| DESIGN.md exists; need wireframes | [`pencil-handoff.md`](./pencil-handoff.md) | `$source-command-bitforge-pencil` |
| PLAN.md (and DESIGN.md if applicable) exist; need execution plan | [`taskify.md`](./taskify.md) | `$source-command-bitforge-taskify` |
| TASKS.md exists; need to pick up work | [`execute-phase.md`](./execute-phase.md) | `$source-command-bitforge-next` |
| Need to know where the project stands | (this file) | `$source-command-bitforge-status` |
| A phase's tasks are all done; close it out | [`execute-phase.md`](./execute-phase.md) closure | `$source-command-bitforge-close-phase` |

Legacy `/bitforge:*` command names may appear in old project docs. In OpenCode, prefer the `$source-command-bitforge-*` names above.

## Cross-cutting standards (apply to every phase)

Read [`standards.md`](./standards.md) for the full list. The non-negotiables:

- **CI is the gate.** Every phase Acceptance includes `agent-ci run --quiet --all` green. Load the `agent-ci` skill first, then run the command it prescribes; do not roll your own.
- **Browser validation is mandatory for visual phases.** Any phase that ships UI loads the `agent-browser` skill first, then runs `agent-browser` against the running app and records what the agent saw.
- **Test pyramid baked in.** Unit (always), integration (where boundaries cross), e2e (UI/Ship phases). Tests live next to the code; separation of concerns is paramount.
- **Hygiene every phase.** Lint clean, format clean, typecheck clean, no introduced dead code, DRY where it makes sense (not preemptively).
- **Internal todo discipline.** Inside `$source-command-bitforge-next`, the agent MUST mirror the phase's task list with OpenCode's todo tool and tick it in lockstep with the file. Both lists end the phase ticked.
- **Source-of-truth respect.** PLAN.md and DESIGN.md outrank TASKS.md. Conflicts update the canonical doc, not the code.

## Routing into other skills

bitforge composes with other skills; it does not replace them.

- **iOS / Swift work** -> invoke the relevant router skill inside the relevant phase if available. bitforge plans the phases; the domain skill executes domain-specific tasks.
- **Issue-tracker / repo routing** -> if the workspace has its own routing skill (e.g. for mapping ticket prefixes to repos), invoke it to land in the right repo before invoking `$source-command-bitforge-next`. bitforge stays out of the routing decision.
- **Pencil wireframes** -> bitforge emits the handoff prompt; the user runs Pencil itself with that prompt. bitforge does not call Pencil tools directly.

## What this skill never does

- Skip phases. Phase ordering is a property of the plan, not a suggestion.
- Mark a phase complete without running its acceptance gates.
- Weaken acceptance to make a task pass.
- Author code before TASKS.md exists, except during Phase 0 of an explicit `$source-command-bitforge-next` run.
- Edit PLAN.md or DESIGN.md silently when reality diverges; surface the conflict, update the doc, then update tasks.

## Installing the OpenCode commands

The skill bundle is portable: drop the `bitforge/` directory into any workspace's `.agents/skills/` and run:

```bash
bash .agents/skills/bitforge/install.sh
```

This creates OpenCode source-command skills in `.agents/skills/source-command-bitforge-*`, giving you `$source-command-bitforge-brainstorm`, `$source-command-bitforge-plan`, `$source-command-bitforge-design`, `$source-command-bitforge-pencil`, `$source-command-bitforge-taskify`, `$source-command-bitforge-next`, `$source-command-bitforge-status`, and `$source-command-bitforge-close-phase` in that workspace. Re-run safely; it is idempotent.

## File map

```
bitforge/
├── SKILL.md              # this file - the router
├── brainstorm.md         # Phase 1: shake every idea loose
├── plan.md               # Phase 2: produce PLAN.md (PRD + SDD)
├── design.md             # Phase 3: produce DESIGN.md
├── pencil-handoff.md     # Phase 3.5: emit Pencil handoff prompt
├── taskify.md            # Phase 4: produce TASKS.md from template
├── execute-phase.md      # Execution: pick up + drive a phase
├── standards.md          # Cross-cutting non-negotiables
├── install.sh            # wire OpenCode source-command skills into the workspace
├── templates/            # canonical document skeletons
│   ├── PLAN.md           # PRD + SDD structure with placeholders
│   ├── DESIGN.md         # Visual design spec structure
│   └── TASKS.md          # Phased execution plan with standards baked in
└── commands/             # source-command prompt sources
    ├── brainstorm.md
    ├── plan.md
    ├── design.md
    ├── pencil.md
    ├── taskify.md
    ├── next.md
    ├── status.md
    └── close-phase.md
```

Skills own the process (interview, adversarial review, sealing). Templates own the structure (sections, placeholders, agent hints). To extend a document with project-specific sections, edit the template; to change how a document is produced, edit the skill.
