<!--
  DESIGN.md template — bitforge

  HOW TO USE
  ----------
  This is the canonical structure for a bitforge project's DESIGN.md. The
  `$source-command-bitforge-design` flow copies this file to `DESIGN.md` at the project
  root, then walks the user through filling every {{PLACEHOLDER}} and
  replacing every <!-- AGENT: ... --> comment.

  The skill at `bitforge/design.md` owns the process (aesthetic interview,
  adversarial review, sealing). This file owns the *structure*.

  Skip this template entirely for non-visual projects (CLI, library, MCP
  server, infrastructure). PLAN.md should already note "this project has
  no UI surface."

  AUTHORING NOTES
  ---------------
  - Establish identity (§1) before tokens (§3). A token system without a
    point of view becomes a Figma component library, not a product.
  - Every component listed in §8 has every state it could plausibly support
    (default / hover / focus-visible / active / disabled / loading / empty
    / error / success — those that apply).
  - Every screen in §10 has every state it could plausibly support
    (default / loading / empty / error / read-only / permission-denied —
    those that apply).
  - Light + dark mode is the baseline. If only one ships in v1, say so in §2.
  - DESIGN.md is platform-neutral. No CSS classes, no Tailwind utilities,
    no SwiftUI types — those mappings live in implementation tasks.
-->

# {{PROJECT_NAME}} — Design Specification

**Companion to:** [`PLAN.md`](./PLAN.md)
**Status:** draft
<!-- AGENT: Flip to `sealed` only after the adversarial review has run. -->
**Version:** 0.1
**Direction:** {{ONE_SENTENCE_DIRECTION}}
<!-- AGENT: e.g. "Developer-tool foundation with editorial type for content surfaces. Dark-mode-first with full light-mode parity." -->

---

## 1. Identity and Principles

### What {{PROJECT_NAME}} is

{{ONE_PARAGRAPH_DESIGN_POV}}
<!-- AGENT: The product's design point of view in plain language. Why this product earns the visual choices below. -->

### Principles

1. **{{PRINCIPLE_1_NAME}}** — {{PRINCIPLE_1_STATEMENT}}
   _Why:_ {{PRINCIPLE_1_RATIONALE}}
2. **{{PRINCIPLE_2_NAME}}** — {{PRINCIPLE_2_STATEMENT}}
   _Why:_ {{PRINCIPLE_2_RATIONALE}}
3. **{{PRINCIPLE_3_NAME}}** — {{PRINCIPLE_3_STATEMENT}}
   _Why:_ {{PRINCIPLE_3_RATIONALE}}

<!-- AGENT: 5–7 numbered principles. Each one sentence + one-sentence rationale. A principle that no decision could ever violate is not a principle. -->

### Anti-patterns

Visual / interaction patterns this design refuses.

- {{ANTI_PATTERN_1}}
- {{ANTI_PATTERN_2}}
- {{ANTI_PATTERN_3}}

<!-- AGENT: Common ones to consider rejecting: emoji in UI chrome, AI-sparkle iconography, purple-to-pink gradients, bouncy spring animations on functional UI, cartoon empty-state illustrations, skeuomorphic textures, rainbow palettes. -->

---

## 2. Modes

### Modes shipping in v1

{{DARK_LIGHT_OR_BOTH_AND_RATIONALE}}

### Mode priority

{{WHICH_MODE_IS_PRIMARY_AND_WHY}}

---

## 3. Color

### Dark mode

```text
# Surfaces
bg-canvas         {{HEX}}    # canvas background, app shell base
bg-surface        {{HEX}}    # panels, cards, sidebar
bg-elevated       {{HEX}}    # popovers, dropdowns, dialogs
bg-overlay        {{RGBA}}   # modal scrim

# Borders
border-subtle     {{HEX}}
border-default    {{HEX}}
border-emphasis   {{HEX}}
border-focus      {{HEX}}

# Text
text-primary      {{HEX}}
text-secondary    {{HEX}}
text-tertiary     {{HEX}}
text-disabled     {{HEX}}
text-on-accent    {{HEX}}

# Accents
accent-primary    {{HEX}}    # primary actions, brand surfaces
accent-primary-hi {{HEX}}    # hover
accent-primary-lo {{HEX}}    # active / pressed
accent-primary-mu {{HEX}}    # disabled / muted

# Status
status-success    {{HEX}}
status-warn       {{HEX}}
status-error      {{HEX}}
status-info       {{HEX}}
status-pending    {{HEX}}

# Project-specific semantics
{{SEMANTIC_TOKENS}}
```

### Light mode

```text
{{LIGHT_MODE_TOKEN_BLOCK}}
```

<!-- AGENT: Same token names. Light mode is not "just invert"; author it explicitly. -->

### Contrast obligations

| Pair | Ratio | WCAG |
|------|-------|------|
| `text-primary` on `bg-canvas` (dark) | {{RATIO}} | AA / AAA |
| `text-primary` on `bg-canvas` (light) | {{RATIO}} | AA / AAA |
| `text-on-accent` on `accent-primary` | {{RATIO}} | AA |
| {{ADD_ROWS_FOR_OTHER_TEXT_ON_SURFACE_PAIRS}} | | |

---

## 4. Typography

### Families

- **Display:** {{DISPLAY_FAMILY}} — weights {{WEIGHTS}}
- **Body:** {{BODY_FAMILY}} — weights {{WEIGHTS}}
- **Mono:** {{MONO_FAMILY}} — weights {{WEIGHTS}}

### Scale

| Token       | Size | Line height | Weight | Use |
|-------------|------|-------------|--------|-----|
| display-lg  | {{PX}} | {{PX}}    | {{W}}  | hero / page heading |
| display-md  | {{PX}} | {{PX}}    | {{W}}  | section heading |
| display-sm  | {{PX}} | {{PX}}    | {{W}}  | sub-section heading |
| body-md     | {{PX}} | {{PX}}    | {{W}}  | default body |
| body-sm     | {{PX}} | {{PX}}    | {{W}}  | secondary body |
| label-md    | {{PX}} | {{PX}}    | {{W}}  | form labels, table headers |
| caption     | {{PX}} | {{PX}}    | {{W}}  | helper, meta |
| mono-md     | {{PX}} | {{PX}}    | {{W}}  | code, IDs |

### Tracking and OpenType features

{{LETTER_SPACING_AND_FEATURES}}
<!-- AGENT: Tabular figures for numeric tables, ligature handling, stylistic sets used. -->

---

## 5. Spacing and layout

### Spacing scale

| Token | Value |
|-------|-------|
| space-0 | 0 |
| space-1 | {{PX}} |
| space-2 | {{PX}} |
| space-3 | {{PX}} |
| space-4 | {{PX}} |
| space-5 | {{PX}} |
| space-6 | {{PX}} |
| space-8 | {{PX}} |
| space-10 | {{PX}} |
| space-12 | {{PX}} |

### Radius

| Token | Value | Use |
|-------|-------|-----|
| radius-sm | {{PX}} | tight: tags, inputs |
| radius-md | {{PX}} | default: buttons, cards |
| radius-lg | {{PX}} | large: modals, drawers |
| radius-pill | {{PX}} | pills, full-circle avatars |

### Shadow / elevation

| Token | Value | Use |
|-------|-------|-----|
| shadow-sm | {{CSS_OR_EQUIV}} | hover lift |
| shadow-md | {{CSS_OR_EQUIV}} | popovers, menus |
| shadow-lg | {{CSS_OR_EQUIV}} | modals |

<!-- AGENT: Specify whether shadows exist in dark mode (often replaced by border-emphasis). -->

### Layout primitives

- **Grid:** {{COLUMN_COUNT}} columns, gutter {{PX}}, max content width {{PX}}
- **Density rules:** {{COMPACT_VS_COMFORTABLE_AND_WHEN}}

---

## 6. Iconography

- **Library:** {{ICON_LIBRARY}}
- **Stroke width:** {{PX}}
- **Sizes:** {{PX}} / {{PX}} / {{PX}}
- **Rule:** {{WHEN_AN_ICON_IS_ALLOWED}}

<!-- AGENT: e.g. "Icons accompany only actions or status, never decorative." -->

---

## 7. Motion

### Tokens

| Token       | Duration | Easing | Use |
|-------------|----------|--------|-----|
| dur-instant | {{MS}}   | linear | hover-state color |
| dur-fast    | {{MS}}   | {{CB}} | menu open / close |
| dur-normal  | {{MS}}   | {{CB}} | view transition |
| dur-slow    | {{MS}}   | {{CB}} | route change |

### Reduced motion

For each motion treatment, the reduced-motion fallback.

| Treatment | Default | `prefers-reduced-motion` |
|-----------|---------|---------------------------|
| {{NAME_1}} | {{DEFAULT_1}} | {{REDUCED_1}} |
| {{NAME_2}} | {{DEFAULT_2}} | {{REDUCED_2}} |

---

## 8. Component primitives

For every primitive: anatomy, sizes, variants, states, tokens used.

### Button

- **Anatomy:** label · optional leading icon · optional trailing icon · optional kbd hint
- **Sizes:** sm · md · lg
- **Variants:** primary · secondary · ghost · destructive
- **States:** default · hover · focus-visible · active · disabled · loading
- **Tokens used:** `accent-primary`, `text-on-accent`, `radius-md`, `space-3`, `dur-fast`

### IconButton

- **Anatomy:** icon-only, with accessible name
- **Sizes:** sm · md · lg
- **Variants:** ghost · subtle · accent
- **States:** default · hover · focus-visible · active · disabled
- **Tokens used:** `radius-md`, `space-2`

### TextInput

- **Anatomy:** label · field · helper / error · optional prefix / suffix
- **Sizes:** sm · md
- **Variants:** default · borderless
- **States:** default · focus · filled · disabled · readonly · error · success
- **Tokens used:** `border-default`, `border-focus`, `status-error`

### Select / Combobox
{{ANATOMY_VARIANTS_STATES_TOKENS}}

### Checkbox / Radio / Switch
{{ANATOMY_VARIANTS_STATES_TOKENS}}

### Tag / Chip
{{ANATOMY_VARIANTS_STATES_TOKENS}}

### Tooltip
{{ANATOMY_VARIANTS_STATES_TOKENS}}

### Toast
{{ANATOMY_VARIANTS_STATES_TOKENS}}

### Modal / Dialog
{{ANATOMY_VARIANTS_STATES_TOKENS}}

### Popover / Dropdown / Menu
{{ANATOMY_VARIANTS_STATES_TOKENS}}

### Tabs
{{ANATOMY_VARIANTS_STATES_TOKENS}}

### Card
{{ANATOMY_VARIANTS_STATES_TOKENS}}

### Avatar
{{ANATOMY_VARIANTS_STATES_TOKENS}}

### Skeleton
{{ANATOMY_VARIANTS_STATES_TOKENS}}

### Toolbar / SegmentedControl
{{ANATOMY_VARIANTS_STATES_TOKENS}}

### {{PRODUCT_SPECIFIC_PRIMITIVE_1}}
{{ANATOMY_VARIANTS_STATES_TOKENS}}

<!-- AGENT: Add product-specific primitives. Drop primitives the project genuinely doesn't use. -->

---

## 9. Composite components

Reusable layouts above the primitive layer.

### Page Header
- {{ANATOMY_AND_STATES}}

### Sidebar Nav
- {{ANATOMY_AND_STATES}}

### Breadcrumb
- {{ANATOMY_AND_STATES}}

### Empty State
- {{ANATOMY_AND_STATES}}
- _Pattern:_ icon · headline · supporting sentence · primary action

### Error State
- {{ANATOMY_AND_STATES}}
- _Pattern:_ icon · headline · what / why / what to do · retry action

### Loading State
- {{ANATOMY_AND_STATES}}
- _Rule:_ skeletons for known shapes; spinners only for unknown durations

### Confirmation Dialog
- {{ANATOMY_AND_STATES}}

### Settings Row
- {{ANATOMY_AND_STATES}}

### List Row
- {{ANATOMY_AND_STATES}}

---

## 10. Screen inventory

For each screen mapped 1:1 to PLAN §6 must-haves.

### {{SCREEN_NAME_1}}

- **Intent:** {{ONE_SENTENCE_INTENT}}
- **Hierarchy:** primary action: {{PRIMARY}}; secondary: {{SECONDARY}}; content surface: {{CONTENT}}
- **States:**
  - **Loading:** {{DESCRIPTION}}
  - **Empty:** {{DESCRIPTION}}
  - **Populated:** {{DESCRIPTION}}
  - **Error:** {{DESCRIPTION}}
  - **Read-only:** {{DESCRIPTION_OR_NA}}
- **Sub-surfaces:** {{MODALS_DRAWERS_POPOVERS_REACHABLE}}
- **Notes:** {{REVIEWERS_HEADS_UP}}

### {{SCREEN_NAME_2}}
{{REPEAT_STRUCTURE}}

### {{SCREEN_NAME_3}}
{{REPEAT_STRUCTURE}}

<!-- AGENT: One section per screen. Cover every screen in PLAN's MVP. -->

---

## 11. Navigation

### Information architecture

{{TOP_NAV_VS_SIDE_NAV_VS_BOTH}}

### Route tree

```
{{ROUTE_TREE}}
```

### Breadcrumb policy

{{WHEN_BREADCRUMBS_APPEAR}}

### Deep-linking

{{DEEP_LINK_EXPECTATIONS}}

---

## 12. Accessibility

- **Contrast:** every token combination meets WCAG AA per §3 contrast obligations.
- **Focus:** focus-visible ring spec — {{COLOR}} · {{WIDTH}} · {{OFFSET}} · {{RADIUS}}. Mouse-only focus is invisible.
- **Keyboard:** every primary flow is keyboard-completable. Documented shortcuts: {{SHORTCUT_TABLE_REF}}.
- **Screen reader:** announcement patterns — {{TOAST}}, {{VALIDATION}}, {{ASYNC}}.
- **Motion:** every motion treatment in §7 has a reduced-motion fallback.
- **Tap targets:** minimum {{PX}}×{{PX}} on touch surfaces.
- **Color independence:** no information conveyed by color alone — icon + text or sigil + color.

---

## 13. Responsive behavior

| Breakpoint | Min width | Behavior summary |
|------------|-----------|------------------|
| sm | {{PX}} | {{NAV}}, {{LAYOUT_RULE}} |
| md | {{PX}} | {{NAV}}, {{LAYOUT_RULE}} |
| lg | {{PX}} | {{NAV}}, {{LAYOUT_RULE}} |
| xl | {{PX}} | {{NAV}}, {{LAYOUT_RULE}} |

For each major surface, the behavior at each breakpoint:

- **{{SURFACE_1}}** — {{BEHAVIOR_PER_BP}}
- **{{SURFACE_2}}** — {{BEHAVIOR_PER_BP}}

---

## 14. Content style

- **Voice and tone:** {{TERSE_OR_CONVERSATIONAL_OR_FORMAL}}.
- **Capitalization:** {{SENTENCE_OR_TITLE_CASE}}.
- **Empty-state copy:** {{PATTERN}} — acknowledge the absence, explain briefly, offer one action.
- **Error message copy:** {{PATTERN}} — what + why + what to do.
- **Date / number / currency:** {{FORMATS}}.

---

## 15. Asset inventory

- **Logos:** {{SIZES}}, {{LOCKUPS}}, clear-space {{RULE}}
- **Favicons:** {{SIZES}}
- **Open Graph:** {{SIZES}}
- **App icons:** {{PER_PLATFORM}}
- **Loading / placeholder imagery:** {{ASSET_OR_NA}}

---

## 16. Open questions

Unresolved design decisions. Each becomes a task or a deferral in TASKS.md.

- [ ] {{OPEN_QUESTION_1}}
- [ ] {{OPEN_QUESTION_2}}

---

## Revisions

<!-- AGENT: After DESIGN.md is sealed, edits land here as a dated entry. Do not silently mutate sealed sections. -->
