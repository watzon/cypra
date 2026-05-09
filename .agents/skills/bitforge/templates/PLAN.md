<!--
  PLAN.md template — bitforge

  HOW TO USE
  ----------
  This is the canonical structure for a bitforge project's PLAN.md (PRD + SDD).
  The `$source-command-bitforge-plan` flow copies this file to `PLAN.md` at the project root,
  then walks the user through filling every {{PLACEHOLDER}} and replacing every
  <!-- AGENT: ... --> comment.

  The skill at `bitforge/plan.md` owns the process (interview, resolve open
  questions, adversarial review, internal-consistency pass, seal). This file
  owns the *structure*. Do not invent new sections silently — if a project
  needs one, add it and document why.

  AUTHORING NOTES
  ---------------
  - Drop sections that genuinely don't apply. An internal CLI does not need
    a §16 Distribution section. A library does not need a §15 Operations
    section. Drop, do not pad.
  - Every load-bearing decision needs a one-or-two-sentence rationale.
  - "We'll figure it out later" belongs in §17 (Risks and Open Questions),
    never elsewhere. Surface deferrals; do not hide them.
  - Pseudocode types in §9 are fine. Full implementations are not.
-->

# {{PROJECT_NAME}} — Plan (PRD + SDD)

**Companion to:** {{COMPANION_DOCS}}
<!-- AGENT: e.g. `[`BRAINSTORM.md`](./BRAINSTORM.md)` (history of the discovery phase), `[`DESIGN.md`](./DESIGN.md)` (visual system). List every doc that this file defers to as source of truth. -->

**Status:** draft
<!-- AGENT: Flip to `sealed` only after the adversarial review and internal-consistency pass have run and the user has approved. -->

**Version:** 0.1
**Owner:** {{OWNER_NAME_OR_HANDLE}}

---

## 1. Executive Summary

{{ONE_PARAGRAPH_PITCH}}
<!-- AGENT: The hallway-pitch version. What this is, who it's for, the problem it solves. One paragraph, not three. -->

---

## 2. Product Vision

{{THREE_TO_FIVE_SENTENCE_VISION}}
<!-- AGENT: What the product is when v1 exists. The feeling a user gets after a successful first session. -->

---

## 3. Non-Goals

A bulleted list of things this project is explicitly NOT for v1. Each item should be a feature or behavior that has been considered and rejected.

- {{NON_GOAL_1}}
- {{NON_GOAL_2}}
- {{NON_GOAL_3}}

<!-- AGENT: Aim for 5–10 non-goals. A non-goal you forget is a non-goal that gets scope-creeped in. -->

---

## 4. Guiding Principles

Numbered principles that govern decisions. 5–7 is the sweet spot. One sentence per principle, one sentence of rationale underneath.

1. **{{PRINCIPLE_1_NAME}}** — {{PRINCIPLE_1_STATEMENT}}
   _Why:_ {{PRINCIPLE_1_RATIONALE}}
2. **{{PRINCIPLE_2_NAME}}** — {{PRINCIPLE_2_STATEMENT}}
   _Why:_ {{PRINCIPLE_2_RATIONALE}}
3. **{{PRINCIPLE_3_NAME}}** — {{PRINCIPLE_3_STATEMENT}}
   _Why:_ {{PRINCIPLE_3_RATIONALE}}

<!-- AGENT: A principle that no decision could ever violate is not a principle ("we will write good code" — useless). If a principle can't lose, replace it. -->

---

## 5. Personas and Use Cases

### Primary persona

**{{PERSONA_NAME}}** — {{PERSONA_ROLE}}, {{PERSONA_CONTEXT}}.
{{PERSONA_TRIGGER_AND_NEED}}

### Secondary personas

- **{{SECONDARY_PERSONA_1}}** — {{ROLE_AND_RELATIONSHIP_TO_PRIMARY}}
- **{{SECONDARY_PERSONA_2}}** — {{ROLE_AND_RELATIONSHIP_TO_PRIMARY}}

### Anti-personas

Who this is NOT for, and what tradeoffs are accepted because of that.

- {{ANTI_PERSONA_1}}
- {{ANTI_PERSONA_2}}

### Canonical end-to-end story

{{CANONICAL_STORY}}
<!-- AGENT: One paragraph. "A user signs up, then ___, then ___, and at the end ___." This is the demo target. The story should survive against the architecture in §7 — if the architecture cannot deliver it, one of the two is wrong. -->

### Other use cases

- {{USE_CASE_1}}
- {{USE_CASE_2}}
- {{USE_CASE_3}}

---

## 6. MVP Definition

### MVP theme

{{ONE_PARAGRAPH_MVP_THEME}}
<!-- AGENT: The smallest version that proves the thesis. Restrictive enough to actually ship in the planned timeline. -->

### MVP must-haves

- {{MUST_HAVE_1}}
- {{MUST_HAVE_2}}
- {{MUST_HAVE_3}}

<!-- AGENT: Each must-have should map to at least one component in §7 and at least one phase in TASKS.md. -->

### Out of scope for MVP

- {{OUT_OF_SCOPE_1}}
- {{OUT_OF_SCOPE_2}}

<!-- AGENT: Subset of §3 Non-Goals that are deferred (not rejected). Items here may move to a future version; items in §3 will not. -->

---

## 7. System Architecture

### Components

A list of every meaningful component with a one-line role.

- **{{COMPONENT_1}}** — {{ROLE_1}}
- **{{COMPONENT_2}}** — {{ROLE_2}}
- **{{COMPONENT_3}}** — {{ROLE_3}}

### Data flow

{{REQUEST_LIFECYCLE_PARAGRAPH_OR_DIAGRAM}}
<!-- AGENT: Trace a happy path through the components. A new contributor should be able to follow this end-to-end. ASCII diagrams welcome. -->

### Trust boundaries

{{WHERE_AUTH_AND_ENCRYPTION_LIVE}}
<!-- AGENT: Which components enforce authn / authz / encryption. What each layer can be trusted with. -->

---

## 8. Data Model

### Entities

For each entity: name, purpose, fields with types, primary key, relationships, indexes that matter.

#### {{ENTITY_1}}
- **Purpose:** {{ENTITY_1_PURPOSE}}
- **Fields:**
  - `{{FIELD_1}}: {{TYPE_1}}` — {{NOTE_1}}
  - `{{FIELD_2}}: {{TYPE_2}}` — {{NOTE_2}}
- **Primary key:** {{PRIMARY_KEY}}
- **Relationships:** {{RELATIONSHIPS}}
- **Indexes:** {{INDEXES}}

#### {{ENTITY_2}}
- **Purpose:** {{ENTITY_2_PURPOSE}}
- ...

### Invariants

Rules the data layer enforces. These become DB constraints + tests.

- {{INVARIANT_1}}
- {{INVARIANT_2}}

### Lifecycle and retention

{{SOFT_VS_HARD_DELETE_AND_RETENTION_WINDOWS}}

---

## 9. Interface Contracts

For every external surface (HTTP API, CLI, library export, MCP tool):

### {{SURFACE_NAME}}

- **Shape:** {{ROUTE_OR_COMMAND_OR_SIGNATURE}}
- **Inputs:** {{INPUT_TYPES}}
- **Outputs:** {{OUTPUT_TYPES_INCLUDING_ERRORS}}
- **Error model:** {{HOW_ERRORS_ARE_REPRESENTED}}
- **Versioning policy:** {{HOW_BREAKING_CHANGES_ARE_HANDLED}}

<!-- AGENT: Repeat per surface. Pseudocode types are fine; do not write implementations. Every error case here MUST be reachable in §14 Failure Modes. -->

---

## 10. Technology Choices

| Area | Choice | Rationale |
|------|--------|-----------|
| Language / runtime | {{LANGUAGE}} | {{RATIONALE}} |
| Web framework | {{FRAMEWORK}} | {{RATIONALE}} |
| Database | {{DB}} | {{RATIONALE}} |
| Queue / async | {{QUEUE}} | {{RATIONALE}} |
| Test runner | {{TEST_RUNNER}} | {{RATIONALE}} |
| Lint / format | {{TOOLING}} | {{RATIONALE}} |
| Package manager | {{PKG_MGR}} | {{RATIONALE}} |
| Deployment target | {{DEPLOYMENT}} | {{RATIONALE}} |

<!-- AGENT: Every row needs a rationale. "Standard for this stack" is not a rationale; explain why this stack is right for *this* project. -->

---

## 11. Security and Privacy

### Auth model

{{AUTH_MODEL}}

### Secret handling

{{HOW_SECRETS_ARE_STORED_AND_ACCESSED}}

### Threat model

{{ONE_PARAGRAPH_THREAT_MODEL}}
<!-- AGENT: Who is attacking, what they want, what we are defending. -->

### PII handling

{{PII_CLASSES_AND_TREATMENT}}

### Compliance constraints

{{HIPAA_GDPR_SOC2_OR_NONE}}

---

## 12. Observability

### Logging

{{STRUCTURED_LOGGING_STRATEGY}}
<!-- AGENT: Levels, fields, correlation IDs. -->

### Metrics

{{OPERATIONALLY_IMPORTANT_METRICS}}

### Tracing

{{HOW_REQUESTS_ARE_TRACED_ACROSS_COMPONENTS}}

### Error tracking

{{TOOL_AND_REDACTION_RULES}}

### Audit events

{{WHAT_GETS_RECORDED_AND_WHO_CAN_READ_IT}}

---

## 13. Performance and Scaling

### Targets

- {{TARGET_1}}
- {{TARGET_2}}

### Expected v1 load

{{LOAD_ESTIMATE}}

### First scaling lever

{{WHAT_WE_REACH_FOR_IF_TARGETS_ARE_EXCEEDED}}

---

## 14. Failure Modes

For each top failure mode in BRAINSTORM, the response.

- **{{FAILURE_1}}** — Detection: {{DETECT_1}}. Recovery: {{RECOVER_1}}. User-visible: {{UX_1}}.
- **{{FAILURE_2}}** — Detection: {{DETECT_2}}. Recovery: {{RECOVER_2}}. User-visible: {{UX_2}}.
- **{{FAILURE_3}}** — Detection: {{DETECT_3}}. Recovery: {{RECOVER_3}}. User-visible: {{UX_3}}.

<!-- AGENT: Every error case named in §9 should appear here. -->

---

## 15. Deployment and Operations

### Environments

{{DEV_STAGING_PROD_OR_OTHER}}

### Deployment mechanism

{{CI_CD_PIPELINE_AND_TARGET}}

### Backup and restore

{{BACKUP_STORY}}

### Migrations

{{DB_AND_API_MIGRATION_STRATEGY}}

### On-call

{{WHO_IS_ON_CALL_AND_WHEN_THIS_BREAKS}}

---

## 16. Distribution and Licensing

### License

{{LICENSE_OR_LICENSE_SPLIT}}
<!-- AGENT: For OSS, name the license per directory. For internal, name the consumers. For commercial, name the channel and pricing model. -->

### Distribution

{{HOW_USERS_GET_THIS}}

---

## 17. Risks and Open Questions

Numbered list. For each: the risk, the impact if realized, the chosen mitigation (or "deferred to Phase X").

1. **{{RISK_1}}** — Impact: {{IMPACT_1}}. Mitigation: {{MITIGATION_1}}.
2. **{{RISK_2}}** — Impact: {{IMPACT_2}}. Mitigation: {{MITIGATION_2}}.
3. **{{RISK_3}}** — Impact: {{IMPACT_3}}. Mitigation: {{MITIGATION_3}}.

<!-- AGENT: Every "we'll figure that out later" from BRAINSTORM lands here, not silently. -->

---

## 18. Glossary

Every domain term used elsewhere in this document, with a one-line definition.

- **{{TERM_1}}** — {{DEFINITION_1}}
- **{{TERM_2}}** — {{DEFINITION_2}}
- **{{TERM_3}}** — {{DEFINITION_3}}

<!-- AGENT: New contributors read this first. -->

---

## Appendix A — ADRs to author during Phase 0

- `docs/adr/0001-{{TOPIC}}.md` — captures rationale for {{DECISION_FROM_§10_OR_§7}}.
- `docs/adr/0002-{{TOPIC}}.md` — captures rationale for {{DECISION}}.
- `docs/adr/0003-{{TOPIC}}.md` — captures rationale for {{DECISION}}.

<!-- AGENT: Every load-bearing decision in §7 (architecture) and §10 (tech choices) gets an ADR. They get authored as Phase 0 tasks in TASKS.md. -->

---

## Revisions

<!-- AGENT: After PLAN.md is sealed, edits land here as a dated entry. Do not silently mutate sealed sections — edit, then add a Revisions entry stating what changed and why. Example:
### 2026-05-15
- §10 changed Postgres → SQLite for v1; rationale: removing the Docker dependency for self-host users. Updated §15 backup story accordingly.
-->
