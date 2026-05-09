# bitforge — Brainstorm Phase

**Goal:** turn a one-sentence idea into a comprehensive `BRAINSTORM.md` that has shaken every loose thread out of the user's head before any planning happens.

**Output artifact:** `BRAINSTORM.md` at the project root.

**Reading order:** read this file fully before asking the user the first question.

---

## Operating rules

1. **You MUST NOT skip to planning.** This phase is about discovery, not architecture. Defer technology choices. Defer schema design. Defer file layouts. Capture *what* and *why* before *how*.
2. **You MUST ask adversarial questions.** It is your job to find every gap, not the user's. Pretend you are about to inherit this project from someone who is leaving the company tomorrow.
3. **You MUST capture answers verbatim where the user is decisive, and as open questions where they hedge.** "I think maybe Postgres" is an open question, not a decision.
4. **You MUST surface contradictions.** If the user says "MVP, fast" but lists 14 must-haves, name it.
5. **You SHOULD spawn the `general-purpose` Agent for an adversarial second pass once the document is in draft.** Frame it as a devil's-advocate review and incorporate its findings.

---

## Process

### Step 1 — Frame the conversation

Tell the user, in one sentence: "I'm going to ask a lot of questions. Skip any that don't apply, but answer the ones that do — what we capture here becomes the source of truth for PLAN.md."

### Step 2 — Walk the question grid

Ask **only the questions that apply to this project shape**. Skip whole sections fearlessly (an internal CLI does not need a monetization section). Cluster questions; do not interrogate one at a time.

#### A. Core idea

- One-sentence description of what this thing *is*.
- Who it is for, specifically. (Not "developers" — *which* developers.)
- The single most important problem it solves.
- The competitive / prior-art landscape — what already exists and why this is different.
- The user's working title and code name (if different).

#### B. Users and use cases

- Primary persona: who, what context, what triggers them to use this.
- Secondary personas (if any).
- The single canonical end-to-end story — "a user signs up, then ___, then ___, and at the end ___".
- Three distinct use cases beyond the canonical one.
- Anti-personas: who is this NOT for, and what tradeoffs are we accepting because of that.

#### C. Scope

- MVP — the smallest thing that proves the thesis. List must-haves.
- Explicit non-goals for v1 — features this will NOT have, said out loud.
- Future versions — features deferred but real.
- Hard constraints (deadlines, platforms, budget, team size).

#### D. Success criteria

- How the user knows v1 is "good enough to ship".
- How the user knows it's working in the wild (metrics, qualitative signals, both).
- What "failure" looks like — the version where this exists but is not used.

#### E. Surface area

- Platforms / form factors (web, iOS, macOS, CLI, server, library, mobile-web, etc.).
- Languages and frameworks the user has a strong preference for or aversion to (and why).
- Existing code or services this must integrate with.
- Hosting / distribution (self-hosted, hosted SaaS, App Store, npm, Homebrew, internal).

#### F. Data and integrations

- Sources of truth: which data this owns, which data it borrows, which data it observes.
- Third-party services this must talk to. For each: read-only? write? auth model?
- PII / sensitive data classes.
- Offline behavior expectations.
- Data lifecycle: retention, deletion, export.

#### G. Failure modes and edge cases

- The three worst things that could go wrong in production.
- What happens when an integration is down.
- What happens when a user does the wrong thing (paste, drag, double-click, browser back).
- Rate limits, abuse, and adversarial input considerations.
- Recovery story — backups, restore, audit trail.

#### H. Security and privacy

- Auth model (anonymous / login / multi-tenant / SSO).
- Where secrets live and who can see them.
- Threat model in one paragraph: who is attacking and what they want.
- Compliance constraints (HIPAA, SOC2, GDPR, COPPA).

#### I. Operations

- Observability expectations: logs, metrics, traces, error tracking.
- Deployment cadence (continuous / weekly / monthly / on-demand).
- Who is on-call when this breaks.
- Cost ceiling for v1 (infra + paid services).

#### J. Open questions

Anything the user said "I'm not sure" or "we'll figure that out" about — capture it as a question with no answer. These will block taskification later.

### Step 3 — Draft `BRAINSTORM.md`

Use this skeleton. Drop sections that are empty rather than padding them.

```markdown
# {{Project}} — Brainstorm

**Status:** draft / sealed
**Date:** {{YYYY-MM-DD}}
**Working title:** {{name}}
**Owner:** {{user}}

## 1. The idea
One paragraph. What it is, who it's for, the problem it solves.

## 2. Personas and use cases
### Primary persona
### Secondary personas
### Canonical story
### Other use cases
### Anti-personas

## 3. Scope
### v1 must-haves
### Explicit non-goals for v1
### Deferred to later
### Hard constraints

## 4. Success
### Ship criteria
### In-the-wild signals
### Failure mode

## 5. Surface area
### Platforms
### Stack preferences and aversions
### Integrations with existing code
### Hosting and distribution

## 6. Data and integrations
### Owned data
### Borrowed data
### Third-party services
### PII / sensitive classes
### Offline behavior
### Lifecycle

## 7. Failure and edge cases
### Top failure modes
### Integration outages
### Adversarial input
### Recovery

## 8. Security and privacy
### Auth
### Secrets
### Threat model
### Compliance

## 9. Operations
### Observability
### Deployment cadence
### On-call
### Cost ceiling

## 10. Open questions
- [ ] question 1
- [ ] question 2
```

### Step 4 — Adversarial second pass

Once the draft exists, spawn an `Agent({subagent_type: "general-purpose"})` with this prompt:

> You are the devil's advocate for this brainstorm. Read `BRAINSTORM.md` at the project root. Your job is to find every place where the document is vague, contradictory, missing an obvious edge case, or assumes something it should have stated. Do not propose architecture. Do not solve problems. Just make a numbered list of every gap. Be ruthless. Aim for 15+ items.

Take the agent's findings and either (a) ask the user to resolve each one, or (b) add unresolved items to the **Open questions** section.

### Step 5 — Seal

When the user has answered every adversarial finding (or explicitly deferred it), change `**Status:** draft` to `**Status:** sealed` and tell the user:

> BRAINSTORM.md is sealed. Run `$source-command-bitforge-plan` to produce PLAN.md.

A sealed brainstorm is the input contract for `plan.md`. If reality changes later, do not silently edit it — append a "Revisions" section dated.

---

## Anti-patterns

- **"What stack do you want to use?"** — premature; this belongs in PLAN.md.
- **"Here's the schema I'd suggest…"** — premature; this belongs in PLAN.md.
- **Skipping H, I, or J** because the user "doesn't care yet" — they will care, just not now. Capture the gap, do not delete the section.
- **Asking 50 questions in one message.** Cluster by section; let the user answer in waves.
- **Padding empty sections with "TBD".** Drop the section instead.
