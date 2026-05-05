# bitforge — Design Phase

**Goal:** produce `DESIGN.md` — a complete visual design specification: identity, principles, design tokens (color/type/space/motion/radius/shadow), iconography, components, screens, states, accessibility, and responsive behavior.

**Inputs:** `PLAN.md` (sealed). `bitforge/templates/DESIGN.md` (the canonical structure). The user's voice / aesthetic preferences.

**Output artifact:** `DESIGN.md` at the project root.

**Skip if:** the project is non-visual (CLI, library, MCP server, infrastructure). Tell the user: "PLAN.md indicates this project has no UI surface. Skipping DESIGN.md and going straight to `$source-command-bitforge-taskify`." Do not produce a token-only DESIGN.md for non-visual projects.

---

## Operating rules

1. **You MUST establish identity before tokens.** A design without a stated point of view becomes a Figma component library, not a product. Write principles before colors.
2. **You MUST define explicit anti-patterns.** "What this design refuses to be" is as load-bearing as "what it is".
3. **You MUST author full token systems for both color modes.** Dark + light is the baseline. If only one mode is in scope for v1, say so explicitly under Modes.
4. **You MUST define every state for every component.** Hover / focus / active / disabled / loading / empty / error / success — for every component that supports them. Missing states are missing requirements.
5. **You MUST define screen states.** Each screen has loading, empty, error, populated, and (where relevant) read-only states.
6. **You MUST specify accessibility numerically.** Color contrast ratios, minimum tap targets, focus ring specs, motion fallbacks.
7. **You MUST NOT include vendor-specific implementation.** No CSS classes, no Tailwind utilities, no SwiftUI types. DESIGN.md is platform-neutral; mappings live in implementation tasks.
8. **You SHOULD spawn an adversarial review sub-agent.**

---

## Process

### Step 1 — Read PLAN.md, especially §5 Personas, §6 MVP, §7 Architecture

The personas tell you the *register* (developer tool ≠ consumer social ≠ enterprise dashboard). The MVP tells you which screens are in scope. The architecture tells you what state surfaces matter (realtime? streaming? long-running jobs? approval gates?).

### Step 2 — Interview the user on aesthetic direction

Cluster the questions; do not interrogate.

- **Reference points** — products the user respects (Linear, Raycast, Sentry, Things, Figma, Stripe, etc.). Three is enough.
- **Anti-references** — products the user wants to avoid feeling like.
- **Density** — comfortable / dense / very dense. Linear-density vs Stripe-marketing-density vs spreadsheet.
- **Mood** — calm / energetic / serious / playful / industrial / soft. One or two words, then a sentence.
- **Brand colors** — fixed brand colors, or freedom to pick? Hex codes if known.
- **Type preference** — known typeface, or pick from a curated set (Inter, Geist, IBM Plex, JetBrains Mono, etc.).
- **Mode priorities** — dark-first / light-first / both equal.
- **Motion appetite** — minimal-functional / moderate / expressive. Consider `prefers-reduced-motion` from the start.

### Step 3 — Copy and fill the template

Copy `bitforge/templates/DESIGN.md` to `DESIGN.md` at the project root. Walk it section by section and replace every `{{PLACEHOLDER}}` and `<!-- AGENT: ... -->` comment with project-specific content drawn from PLAN.md and the aesthetic interview from Step 2.

Drop sections that genuinely don't apply (e.g., an iPad-only app may drop responsive breakpoints). Add product-specific primitives in §8 — every project has at least one component that doesn't appear in the template defaults.

The template is the structural source of truth — its inline `<!-- AGENT: ... -->` hints carry the per-section guidance. This skill governs the *process*; the template governs the *shape*.

### Step 4 — Adversarial review

Spawn `Agent({subagent_type: "general-purpose"})` with:

> Review DESIGN.md as a senior product designer. Read DESIGN.md at the project root. List every place where:
> - a token is defined but never used (orphan)
> - a token is referenced but never defined (dangling)
> - a component lacks a state it should plausibly have (a button with no disabled? a select with no loading?)
> - a screen lacks a state it should plausibly have (a list with no empty? a detail with no error?)
> - a contrast pair is unverified
> - the principles contradict the components
> - the anti-patterns contradict actual choices in the document
> - reduced-motion fallbacks are missing
> - keyboard shortcuts are claimed but not enumerated
>
> Aim for 20+ findings.

Update DESIGN.md.

### Step 5 — Seal

Change `Status: draft` → `Status: sealed`. Tell the user:

> DESIGN.md is sealed. Run `$source-command-bitforge-pencil` to get the handoff prompt for the Pencil design agent, or run `$source-command-bitforge-taskify` to skip wireframes and go straight to TASKS.md.

---

## Anti-patterns

- **Tokens copied from a starter.** Every token should be intentional in this product's context.
- **"Inter, default sizes".** Pick the actual scale; vagueness here cascades into spacing inconsistencies forever.
- **Skipping anti-patterns.** They are not optional. They keep the design from drifting.
- **Skipping screen states.** "Empty / loading / error" is the difference between a polished product and a demo.
- **Pretending light mode is "just invert".** It isn't. Author it explicitly even if the conversion looks mechanical.
