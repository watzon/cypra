# bitforge — Plan Phase

**Goal:** produce `PLAN.md` — a combined PRD + Software Design Document that is the canonical source of truth for the project's *what* and *how*.

**Inputs:** `BRAINSTORM.md` (sealed), `bitforge/templates/PLAN.md` (the canonical structure), the user's answers to follow-up questions.

**Output artifact:** `PLAN.md` at the project root.

**Reading order:** read this file, then `bitforge/templates/PLAN.md`, then BRAINSTORM.md.

---

## Operating rules

1. **You MUST treat sealed `BRAINSTORM.md` as input contract.** Every must-have, non-goal, persona, and integration listed there shows up here. If something feels missing from BRAINSTORM, ask the user — do not invent.
2. **You MUST resolve every Open Question in BRAINSTORM.md before sealing the plan.** Either inline the answer, defer it explicitly with a justification, or kick it back to the user.
3. **You MUST make architectural decisions concrete.** "Use a database" is not a decision; "Postgres 16, single instance, schema-per-tenant, migrations via {{tool}}" is.
4. **You MUST justify every load-bearing decision in one or two sentences.** A reader should be able to challenge any choice and find the rationale.
5. **You MUST list explicit non-goals.** A non-goal you forget will get scope-creeped in.
6. **You SHOULD spawn an adversarial review sub-agent once the draft exists.**
7. **You MUST NOT include code.** Pseudocode for an interface contract is fine; full implementations are not.

---

## Process

### Step 1 — Read BRAINSTORM.md fully

Re-read it. Note any inconsistencies between sections — they will surface again here.

### Step 2 — Resolve open questions

Walk the **Open questions** list with the user. Each one becomes:

- **Answered** — inline the answer in the relevant PLAN section.
- **Deferred** — moved to PLAN's Risks & Open Questions with a deferral rationale ("blocked on X", "not needed until Phase Y", "will discover during Phase 0 spike").
- **Closed as out of scope** — moved to Non-Goals.

### Step 3 — Copy and fill the template

Copy `bitforge/templates/PLAN.md` to `PLAN.md` at the project root. Walk it section by section and replace every `{{PLACEHOLDER}}` and `<!-- AGENT: ... -->` comment with project-specific content drawn from BRAINSTORM.md and the follow-up answers from Step 2.

Drop sections that genuinely don't apply (e.g., a CLI-only project may drop §15 Operations beyond a single line). Add new top-level sections for project-shape-specific concerns (e.g., "Plugin SDK" for a tool with a public extension API). When you add a section, document why in the section itself.

The template is the structural source of truth — its inline `<!-- AGENT: ... -->` hints carry the per-section guidance. This skill governs the *process*; the template governs the *shape*.

### Step 4 — Adversarial review

Spawn `Agent({subagent_type: "general-purpose"})` with:

> You are reviewing PLAN.md as a senior engineer who is about to inherit this project. Read PLAN.md at the project root. For each section, list every place where:
> - a decision sounds vague ("we'll use a database", "queues will handle async work")
> - a tradeoff is asserted without rationale
> - an invariant is stated without a mechanism to enforce it
> - the data model leaves a question unanswered (uniqueness, ordering, deletion semantics, foreign keys)
> - the interface contract has an unspecified error case
> - a risk is named without a mitigation
> - the canonical user story breaks given the architecture as written
>
> Be ruthless. Aim for 20+ findings. Do not propose alternatives — just name the gaps.

Walk the findings with the user. Update PLAN.md.

### Step 5 — Internal consistency pass

Cross-check:

- Every must-have in §6 has a component in §7 that owns it.
- Every component in §7 appears in §10 with chosen tech.
- Every data invariant in §8 has either a constraint or a test plan reference.
- Every error case in §9 is reachable in the failure modes in §14.
- Every term used in any section appears in §18.
- Non-goals in §3 are not contradicted elsewhere.

### Step 6 — Seal

Change `Status: draft` → `Status: sealed`. Tell the user:

> PLAN.md is sealed. If this project has a UI, run `$source-command-bitforge-design`. Otherwise, run `$source-command-bitforge-taskify`.

If reality changes later, edit PLAN.md openly with a Revisions section and a date — do not silently mutate sections that taskify already consumed.

---

## Anti-patterns

- **"We'll figure out the schema during Phase 1."** No — name the entities and key fields now. The phase is for *implementation*, not discovery.
- **Listing 12 must-haves and calling it MVP.** Push back. MVP means the smallest thing that proves the thesis.
- **A principle that no decision could ever violate.** ("We will write good code.") If a principle can't lose, it isn't a principle.
- **Citing a "future phase" without acknowledging it in §17.** Risks must list deferrals.
- **Tech choices without rationale.** Every row of §10 needs a one-sentence reason.
