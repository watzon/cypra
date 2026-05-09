# bitforge — Pencil Handoff

**Goal:** emit a handoff prompt the user pastes into Pencil so the Pencil design agent can build a complete `.pen` wireframe set from `DESIGN.md`.

**Inputs:** `DESIGN.md` (sealed).

**Output:** the handoff prompt printed to chat. The user pastes it into a Pencil session.

**Skip if:** no DESIGN.md exists, or the user explicitly says "no wireframes". Tell them: "Without DESIGN.md or a Pencil session, run `$source-command-bitforge-taskify` to proceed."

---

## Operating rules

1. **You MUST NOT call Pencil tools yourself.** This skill produces a handoff prompt. Pencil work happens in a separate Pencil-aware session driven by the user.
2. **You MUST verify DESIGN.md is sealed.** A draft DESIGN.md produces wireframes that have to be redone. Tell the user to seal it first.
3. **You MUST tailor the prompt to the project's screen inventory.** A generic Pencil prompt is half as useful as one that names the screens.
4. **You MUST encode the build order.** Variables → atomic components → molecules → organisms → screens → state variations. Skipping the order produces brittle Pencil files.

---

## Process

### Step 1 — Read DESIGN.md fully

Pay attention to:
- §3 Color tokens (both modes)
- §4 Typography scale
- §5 Spacing / radius / shadow
- §7 Motion tokens
- §8 Component primitives + their states
- §9 Composite components
- §10 Screen inventory + screen states
- §11 Navigation
- §13 Responsive breakpoints

### Step 2 — Build the handoff prompt

Emit it as a single fenced markdown block the user can copy. Structure:

````markdown
```
You are the Pencil design agent for {{Project}}. Read DESIGN.md at the project root before doing anything else; it is the source of truth for every token, component, state, and screen.

Build the .pen file in this order. Do not skip ahead. After each phase, verify against DESIGN.md before moving to the next.

## Phase 1 — Variables

Define every design token from DESIGN.md as Pencil variables. Use the same names as in the document. Sections to create:

- Colors (dark mode set + light mode set)
- Typography (families, weights, the full type scale with size + line-height + tracking)
- Spacing scale
- Radius tokens
- Shadow / elevation tokens
- Motion tokens (durations + easing)

Verification: every token in DESIGN.md §3, §4, §5, §7 has a matching Pencil variable.

## Phase 2 — Atomic components

Build every primitive listed in DESIGN.md §8. For each component create:

- All size variants
- All style variants (primary / secondary / ghost / destructive / etc.)
- Every state DESIGN.md specifies: default, hover, focus-visible, active, disabled, loading (where applicable), error / success (where applicable)
- Both color modes

Atomic components include (at minimum, plus product-specific):
{{INSERT screen-specific primitive list from DESIGN.md §8}}

Verification: every primitive in DESIGN.md §8 exists in the .pen with every state listed.

## Phase 3 — Molecule components

Compose atomic components into the molecules in DESIGN.md §9. Examples:
- Page header (breadcrumb + title + actions)
- Sidebar nav row (icon + label + active indicator + count badge)
- Form row (label + input + helper / error)
- List row (avatar + primary + secondary + meta + actions)
- Empty state (icon + headline + body + primary action)
- Error state
- Loading state
- Confirmation dialog
- Settings row

Every molecule has its own state set where applicable (selected vs default for nav rows, expanded vs collapsed for collapsible rows, etc.).

## Phase 4 — Layout components

Build the reusable layout shells in DESIGN.md:
- App shell (top bar + side nav + content)
- Page layouts (split, single-column, dashboard grid, canvas)
- Modal / drawer shells
- Toolbar / segmented control bars

Verify breakpoint behavior: each layout demonstrates its sm / md / lg form.

## Phase 5 — Screens

For every screen in DESIGN.md §10, build:
- The default populated state
- The empty state (where applicable)
- The loading state (skeleton or spinner per DESIGN.md §7)
- The error state (with retry where applicable)
- The read-only or permission-denied state (where applicable)

Use the components from earlier phases — never inline-style anything. If a screen reveals a missing component or state, surface it; do not paper over it.

## Phase 6 — Interaction states

For every interactive surface, demonstrate:
- Hover, focus-visible, active, disabled (where the component supports them)
- Open vs closed for any disclosure (dropdown, popover, accordion, drawer, modal)
- Selected vs unselected (lists, tabs, segmented controls)
- Validation states for inputs (default, focus, error, success)
- Async states for actions (idle, pending, success toast, error toast)

## Phase 7 — Validation pass

Walk DESIGN.md from §1 to §16 and confirm:
- Every token is referenced somewhere in the .pen
- Every component listed in §8 / §9 exists with every state
- Every screen in §10 exists in every state listed
- Both color modes work — switch the mode variable and every screen still reads
- The principles in §1 visibly hold (e.g. "quiet by default" → static surfaces are near-grayscale)
- The anti-patterns in §1 are absent

If any check fails, fix it inside the .pen and note the gap. Do not edit DESIGN.md from inside Pencil — surface the gap to the user.

## Notes

- Use the get_guidelines tool first to load Pencil best practices.
- Stay in two color modes. Avoid hardcoded hex anywhere outside the variables panel.
- Name layers and components consistently with DESIGN.md vocabulary so cross-referencing is trivial.
- When in doubt, build smaller components and compose, rather than larger ones with internal variants.
```
````

### Step 3 — Print and instruct

Print the prompt block above with `{{Project}}` and the screen-specific primitive list filled in from DESIGN.md. Then tell the user:

> Copy the block above into a Pencil session. After Pencil produces the .pen, return here and run `$source-command-bitforge-taskify`.

---

## Anti-patterns

- **Calling Pencil's `batch_design` from this session.** This session does not own the Pencil .pen. Hand off, do not drive.
- **Generic prompts with no screen list.** The agent enumerates §10 screens — that's why DESIGN.md exists.
- **Telling Pencil to "just follow DESIGN.md".** Specify the build order. Without it, Pencil will start with screens and pay for it later.
