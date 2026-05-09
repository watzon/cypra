# Cypra — Design Specification

**Companion to:** [`PLAN.md`](./PLAN.md)
**Status:** sealed
**Version:** 1.0
**Direction:** Polished, technical, slightly editorial — Linear's calm density and Stripe's trustworthiness, with Vercel's confident type-as-system, expressed through an all-Monaspace stack. Dark and light modes shipping at parity. Three surfaces (operator dashboard, hosted login, setup wizard) share one design system; hosted login accepts per-tenant theme overrides.

---

## 1. Identity and Principles

### What Cypra is

Cypra is a security product that has to feel like one — but not a *grim* one. The product surface lives mostly between an operator who picks it because Keycloak is too much and an end-user who is signing in to someone else's app for thirty seconds. Both audiences should walk away with the same impression: *somebody competent built this and knew what they were doing.* The way Cypra earns that impression visually is by being precise without being austere — sharp tokens, tabular numerics, calm color, and one bold typographic decision (all-Monaspace) that signals "this is a tool, not a marketing site." The dashboard is dense the way Linear is dense: every pixel is doing work, but the page never feels crowded. The hosted login pages are calmer, generous-spaced, theme-able down to the accent color so the end-user feels like they're signing into the *tenant's* product, not Cypra's.

### Principles

1. **Identifiers are first-class.** Every `client_id`, `kid`, `tenant_id`, and slug renders in Krypton (mechanical mono) inside a copy-affordant pill, never as inline body text.
   _Why:_ This product is half about administering identifiers. Treating them as styled body text loses the affordance and the "you can copy this" cue.
2. **Calm density on the dashboard.** Information surfaces compact to the smallest readable size that still meets contrast and tap-target rules. No padding for padding's sake.
   _Why:_ Operators and tenant admins live in this UI. Dense layouts respect their time; spacious layouts patronize them.
3. **Generous space on the hosted login.** The end-user sees this surface for 20 seconds. Calm wins over efficient. White space, larger touch targets, no clever density.
   _Why:_ A stranger to Cypra needs zero cognitive overhead between "click the button" and "I'm signed in."
4. **Color carries meaning, not decoration.** Status colors are reserved for state. Brand teal is reserved for primary action. No decorative gradients, no accent-color-as-divider, no rainbow tagging.
   _Why:_ A security product abuses its trust budget when color is purely aesthetic — users learn to ignore the alarm color.
5. **Motion confirms, never decorates.** Tasteful micro-animations on state change (press, focus, modal entry); no springs, no parallax, no decorative loops, no shimmer. Reduced-motion users get the same UI minus the motion, never a degraded one.
   _Why:_ Motion in a functional UI is a contract: when something moves, something happened. Decorative motion breaks the contract.
6. **Theme-able where it matters; opinionated where it doesn't.** Tenants override accent, logo, and display name on hosted-login surfaces only. They cannot override the dashboard, layout, hierarchy, component anatomy, or the focus-ring contract.
   _Why:_ Tenants need their brand to land; nobody benefits from tenants reinventing input fields, and nobody benefits from a tenant accidentally hiding the focus ring.
7. **One typographic family, two voices.** Monaspace Neon for everything readable; Monaspace Krypton exclusively for identifiers and code. No third family, no marketing display face.
   _Why:_ A single mono superfamily is distinctive, fits the product, and gives us tabular numerics for free. Mixing families dilutes both.

### Anti-patterns

- **Enterprise-grim.** No slate-and-steel "trustworthy" gradient, no stock illustration of a shield-with-a-checkmark, no "security made simple" hero copy.
- **AI-sparkles.** No purple-pink gradients, no "✨" in UI chrome, no emoji-decorated headings, no glow effects that imply machine intelligence.
- **Skeuomorphism.** No leather-textured "vault" panels, no ribbon "secure" badges, no fake bevels, no faux dimensionality outside the elevation system in §5.
- **Cartoon empty states.** No mascots, no "oops!" friendly copy, no oversized illustrations. Empty states are one icon + one sentence + one action.
- **Decorative dividers and pills.** Color is for state, not chrome. A neutral border carries every layout boundary.
- **Bouncy springs on functional UI.** Easing curves are tasteful and short. No overshoot, no bounce, no rubber-band on dashboard interactions.
- **Decorative loading motion.** No shimmering skeletons, no pulse loops on loading state. Skeletons are static blocks; a moving skeleton implies progress that isn't happening.
- **Mystery icons.** Every icon is paired with text or has an accessible-name + tooltip. Conventional icon-only controls (close, copy, expand) are exempt; the collapsed sidebar is exempt because every item has an accessible-name + tooltip and the *expanded* sidebar is the default for new sessions.
- **Color-only status.** A green dot is never the only signal that a key is active; the dot is paired with a sigil shape and a label.
- **Apologetic copy.** No "oops", no "this shouldn't happen", no exclamation marks, no "we're sorry". State the failure and the next action.

---

## 2. Modes

### Modes shipping in v1

**Both dark and light at parity.** Authoring order is dark-first (the dashboard's primary register) with a deliberate light-mode pass that re-grounds tokens against the lighter canvas — not a mechanical inversion. Hosted-login pages default to **system-preference**; tenant branding settings (v1.1) may pin a tenant's preference.

### Mode priority

Dashboard primary surface is dark, but no information density or interaction is lost in light mode. Hosted login follows system preference by default. Setup wizard inherits from system preference and announces the chosen mode in the first-step banner so the operator knows the dashboard will start there.

---

## 3. Color

### Dark mode

```text
# Surfaces
bg-canvas          #0A0A0B    # app shell base
bg-surface         #111113    # panels, cards, sidebar background
bg-elevated        #18181B    # popovers, dropdowns, dialogs
bg-overlay         rgba(0,0,0,0.72)   # modal scrim (dark)
bg-code            #161618    # identifier pills, code blocks

# Borders
border-subtle      #1F1F23
border-default     #2A2A2F
border-emphasis    #3F3F46
border-focus       #2DD4BF    # system focus-ring color — invariant under tenant theming

# Text
text-primary       #F4F4F5
text-secondary     #A1A1AA
text-tertiary      #71717A
text-disabled      #52525B    # WCAG 1.4.3 exempts disabled UI components from contrast minima
text-on-accent     #0A0A0B
text-identifier    #E4E4E7    # all Krypton-rendered text uses this token

# Accents
accent-primary     #2DD4BF    # primary actions, active states (teal-400)
accent-primary-hi  #5EEAD4    # hover
accent-primary-lo  #14B8A6    # active / pressed
accent-primary-mu  rgba(45,212,191,0.14)   # muted background fills (semantically: selection + transient feedback)

# Status (paired with sigil shape, never color-only)
status-success     #22C55E
status-warn        #F59E0B
status-error       #EF4444
status-info        #3B82F6
status-pending     #A855F7    # used by Toast pending, StatusPip pending, OIDC token-mint-pending, PAT creation in flight

# Project-specific semantics
context-instance   #A78BFA    # "you are acting as the instance admin" cue
context-tenant     #2DD4BF    # = accent-primary by default; overridable on hosted login only
key-state-active     #22C55E
key-state-overlap    #F59E0B
key-state-sunsetting #71717A
secret-mask        #71717A    # the dots in client_secret_•••••, AND the redaction sentinel in audit log
```

### Light mode

```text
# Surfaces
bg-canvas          #FAFAFA
bg-surface         #FFFFFF
bg-elevated        #FFFFFF    # paired with shadow-md
bg-overlay         rgba(9,9,11,0.40)
bg-code            #F4F4F5

# Borders
border-subtle      #EDEDEF    # distinct from bg-code so a code-pill on a card retains its edge
border-default     #E4E4E7
border-emphasis    #D4D4D8
border-focus       #0D9488    # system focus-ring color — invariant under tenant theming

# Text
text-primary       #09090B
text-secondary     #52525B
text-tertiary      #71717A
text-disabled      #A1A1AA
text-on-accent     #FFFFFF
text-identifier    #18181B

# Accents (deeper than dark mode — works on white)
accent-primary     #0D9488    # teal-600
accent-primary-hi  #0F766E
accent-primary-lo  #115E59
accent-primary-mu  rgba(13,148,136,0.10)

# Status
status-success     #16A34A
status-warn        #D97706
status-error       #DC2626
status-info        #2563EB
status-pending     #9333EA

# Project-specific semantics
context-instance   #7C3AED
context-tenant     #0D9488
key-state-active     #16A34A
key-state-overlap    #D97706
key-state-sunsetting #71717A
secret-mask        #A1A1AA
```

### Tenant theming overrides (hosted login only)

A tenant's branding settings supply a single accent color, a logo (SVG or PNG), and an optional display name. The hosted-login layer maps the tenant accent to a **narrow** set of tokens:
- `accent-primary*` (with derived hover/pressed/muted variants computed via fixed lightness deltas),
- `context-tenant`,
- *selected-state colors* on Checkbox / Radio / Switch.

The tenant accent does **not** override:
- `border-focus` (always the system teal — focus-ring contrast is a system invariant),
- any `status-*` token,
- any neutral surface, border, or text token,
- anything on the dashboard surface (theming applies to hosted-login routes only).

**Tenant accent validation gate.** At save time the dashboard refuses any accent that fails either of:
- `text-on-accent` vs. tenant accent ≥ **4.5:1** (small text AA),
- tenant accent vs. `bg-canvas` ≥ **3:1** (graphical-element AA per WCAG 1.4.11).

The save-error message names the failing pair and the measured ratio.

### Contrast obligations

| Pair | Ratio | WCAG criterion |
|---|---|---|
| `text-primary` on `bg-canvas` (dark) | 16.9 : 1 | 1.4.6 AAA |
| `text-primary` on `bg-canvas` (light) | 16.7 : 1 | 1.4.6 AAA |
| `text-secondary` on `bg-surface` (dark) | 7.2 : 1 | 1.4.6 AAA |
| `text-secondary` on `bg-surface` (light) | 7.5 : 1 | 1.4.6 AAA |
| `text-tertiary` on `bg-surface` (dark) | 4.6 : 1 | 1.4.3 AA |
| `text-tertiary` on `bg-surface` (light) | 4.6 : 1 | 1.4.3 AA |
| `text-on-accent` on `accent-primary` (dark) | 9.8 : 1 | 1.4.6 AAA |
| `text-on-accent` on `accent-primary` (light) | 4.7 : 1 | 1.4.3 AA |
| `text-identifier` on `bg-code` (dark) | 14.2 : 1 | 1.4.6 AAA |
| `text-identifier` on `bg-code` (light) | 13.4 : 1 | 1.4.6 AAA |
| `status-error` on `bg-surface` (dark) | 5.1 : 1 | 1.4.3 AA |
| `status-error` on `bg-surface` (light) | 5.5 : 1 | 1.4.3 AA |
| `status-success` on `bg-surface` (dark) | 5.6 : 1 | 1.4.3 AA |
| `border-focus` on `bg-canvas` (dark) — graphical | 8.1 : 1 | 1.4.11 (3:1 minimum) |
| `border-focus` on `bg-canvas` (light) — graphical | 4.6 : 1 | 1.4.11 (3:1 minimum) |
| `text-disabled` on any surface | n/a | exempt under 1.4.3 (disabled UI) |

---

## 4. Typography

### Families

- **Body / UI / display:** **Monaspace Neon** (neo-grotesque mono), weights 400 / 500 / 600 / 700.
- **Identifiers / code / secrets:** **Monaspace Krypton** (mechanical mono), weights 400 / 500.

Both faces are part of GitHub's Monaspace superfamily; metrics align so the two mix cleanly inline. Texture-healing OpenType feature is enabled on Neon to even out word shapes.

### Tradeoffs we accept

The all-Monaspace choice is a deliberate point-of-view decision; the costs are real and named here so they aren't relitigated in code review.

- **~10–15% fewer characters per row** at the same point size compared to Inter/Geist. Compensated by the body sizes in the scale below; long table cells use `body-sm`.
- **Long error messages take more horizontal space.** Error copy in §14 is constrained to ≤ 14 words per sentence to keep messages on two lines or fewer at the dashboard's narrowest column.
- **CJK / RTL coverage is not native.** Monaspace ships Latin / Greek / Cyrillic. When i18n lands (post-v1), CJK and Arabic fall back to the system mono stack (`ui-monospace` then `Menlo` / `Consolas` / system) rather than introducing a third typographic family. The hosted-login layer announces the language for the screen reader; no further adjustment.
- **Tabular numerics are mandatory** because mixed-width numerals look worse in mono than in proportional faces. Lining numerals are the only kind we use.
- **Density per row in dashboard tables** is achieved via `body-sm` cell text and `40 px` row height — not by fighting the font.

### Scale

| Token | Size | Line height | Weight | Family | Use |
|---|---|---|---|---|---|
| display-lg | 32 px | 40 px | 600 | Neon | hero / page heading on the setup wizard |
| display-md | 24 px | 32 px | 600 | Neon | section heading in dashboard pages |
| display-sm | 20 px | 28 px | 600 | Neon | sub-section heading, modal title |
| body-md | 14 px | 22 px | 400 | Neon | default body, form fields, inputs |
| body-sm | 13 px | 20 px | 400 | Neon | secondary body, table cells, tooltips, helpers |
| label-md | 13 px | 20 px | 500 | Neon | form labels, table headers, key-value labels |
| label-sm | 11 px | 16 px | 600 | Neon | tags, eyebrow labels, badges (uppercase + tracking) |
| caption | 12 px | 18 px | 400 | Neon | metadata, timestamps, helper |
| identifier-md | 13 px | 20 px | 400 | Krypton | inline `client_id`, `kid`, slug, fingerprint |
| identifier-sm | 12 px | 18 px | 400 | Krypton | identifier inside table cells, AuditEntry resource_id |
| code-md | 13 px | 20 px | 400 | Krypton | code blocks (SDK snippets, env-var examples) |

### OpenType features

- **Texture-healing on Neon** at body-md and below.
- **Tabular numerics** enabled globally on Neon.
- **Stylistic set ss01 (Neon)** disabled — keeps the default `g`/`a` shapes (more familiar).
- **Krypton ligatures off in Code Block contexts only** — we want operators copy-pasting JS / Go / shell to see `=>` / `!=` / `<=` literally in code-md, not collapsed to glyphs. Identifiers don't contain those sequences, so the rule is effectively scoped to Code Block.
- **label-sm** uses uppercase + 0.06em letter-spacing for eyebrow micro-labels.
- **identifier-** sizes inherit Krypton's tighter advance width; no added letter-spacing.

---

## 5. Spacing and layout

### Spacing scale

| Token | Value |
|---|---|
| space-0 | 0 |
| space-1 | 4 px |
| space-2 | 8 px |
| space-3 | 12 px |
| space-4 | 16 px |
| space-5 | 20 px |
| space-6 | 24 px |
| space-8 | 32 px |
| space-10 | 40 px |
| space-12 | 48 px |
| space-16 | 64 px |

### Radius

| Token | Value | Use |
|---|---|---|
| radius-sm | 4 px | identifier pills, tags, inline badges |
| radius-md | 6 px | inputs, buttons, cards, table cells with backgrounds |
| radius-lg | 10 px | modals, drawers, large cards |
| radius-pill | 9999 px | full-circle avatars, status dots |

### Shadow / elevation

| Token | Light value | Dark value |
|---|---|---|
| shadow-sm | `0 1px 2px rgba(9,9,11,0.06)` | `inset 0 0 0 1px border-emphasis` |
| shadow-md | `0 4px 12px rgba(9,9,11,0.08), 0 1px 3px rgba(9,9,11,0.04)` | `inset 0 0 0 1px border-emphasis` on `bg-elevated` |
| shadow-lg | `0 16px 32px rgba(9,9,11,0.12), 0 4px 8px rgba(9,9,11,0.06)` | `bg-overlay` scrim + `inset 0 0 0 1px border-emphasis` |

Dark-mode "shadows" are `inset` borders + an elevated surface tier; no glow.

### Layout primitives

- **Dashboard grid:** 12-column, 24 px gutter, max content width **1280 px**, side padding `space-6` (≥ md) / `space-4` (sm).
- **Hosted-login content width:** max **440 px** centered, vertical rhythm `space-6`. Generous; never dense.
- **Setup wizard width:** max **560 px** centered, slightly taller line-height for first-impression polish.
- **Density rules:**
  - Dashboard table rows: `40 px` minimum (compact); `48 px` comfortable, used in content-heavy detail tables.
  - Dashboard form rows: `space-5` between rows; labels above inputs.
  - Hosted login: `space-6` between rows; one action per screen wherever possible.
- **Sidebar nav width:** `240 px` expanded (default), `56 px` collapsed.

---

## 6. Iconography

- **Library:** **Lucide** (matches ShadCN/Watermelon UI defaults; tree-shaken to icons we use).
- **Stroke width:** `1.5 px` at all sizes.
- **Sizes:** `14 / 16 / 20 / 24 px`. Default `16 px`. `14 px` only inline within body-sm. `24 px` for empty/error-state hero icons.
- **Color:** matches surrounding text token. Status icons use the matching `status-*` token.
- **Rule:** icons accompany only **actions or status**, never decorative. Every icon-only control has an accessible name + tooltip.
- **Lucide icons reserved for specific semantics:** `Lock` (permission-denied affordance in sidebar nav), `Copy` (every IdentifierPill), `Eye` / `EyeOff` (MaskedSecret reveal), `RotateCw` (rotation actions), `Trash2` (destructive actions), `AlertTriangle` (warn states), `XCircle` (error states), `CheckCircle2` (success states), `Clock` (pending states).
- **Cypra-specific glyphs** (custom, drawn to match Lucide stroke; visually distinct **shapes**, not just color):
  - **Tenant** — square-bracket shape `[ ]` enclosing a dot. Used in TenantSwitcher and ContextBadge. Distinct from Instance via outline shape.
  - **Instance** — 2×2 grid of solid dots. Used in instance-admin ContextBadge. Distinct from Tenant via fill pattern.
  - **Passkey** — fingerprint-and-key composite.
  - **OIDC** — three concentric arcs.

---

## 7. Motion

### Tokens

| Token | Duration | Easing | Use |
|---|---|---|---|
| dur-instant | 80 ms | `linear` | hover-state color change, focus-ring fade-in |
| dur-fast | 150 ms | `cubic-bezier(0.20, 0.00, 0.00, 1.00)` | menu / popover / tooltip open and close |
| dur-normal | 220 ms | `cubic-bezier(0.20, 0.00, 0.00, 1.00)` | modal entry, drawer slide-in |
| dur-slow | 350 ms | `cubic-bezier(0.20, 0.00, 0.00, 1.00)` | route changes, tab content swap |
| dur-shake | 220 ms | `cubic-bezier(0.36, 0.07, 0.19, 0.97)` | invalid-input shake on form-submit failure (one-off curve, deliberate) |

The fast/normal/slow trio shares one easing curve to produce a unified feel across micro and macro motion. `dur-instant` is `linear` because color transitions don't benefit from a curve, and `dur-shake` uses a one-off curve because the shake is a deliberate physical metaphor that doesn't read with the unified curve.

### Motion treatments

- **Press:** scale `1.00 → 0.985` over `dur-instant`, paired with color shift to `accent-primary-lo`.
- **Hover:** color shift only; no transform. Buttons get `bg-elevated` on ghost variants.
- **Focus:** focus ring fades in over `dur-instant`. No scale.
- **Modal entry:** scrim fades in over `dur-fast`; modal slides up `8 px` + fades in over `dur-normal`.
- **Toast entry:** slides up `12 px` + fades in over `dur-fast`. Auto-dismiss timer paused on hover.
- **Tab content swap:** outgoing fades to 0 over `dur-fast`; incoming fades to 1 over `dur-fast`. No slide.
- **Skeleton:** **static** — no shimmer, no pulse. A skeleton signals "structure, content pending"; movement would imply progress that isn't happening.
- **Identifier copy confirm:** the IdentifierPill flashes `accent-primary-mu` background for `dur-fast`, paired with a "Copied" tooltip and an `aria-live="polite"` announcement.
- **Invalid-input shake:** `dur-shake`, only on form-submit failure (not on per-field blur validation).
- **Route change:** outgoing fades to 0 over `dur-fast`, route renders, incoming fades in over `dur-fast`. No slide.

### Reduced motion

`prefers-reduced-motion: reduce` collapses every duration to **0 ms** and removes every transform. Color transitions remain — they aren't motion. Layout is identical between modes. Reduced-motion is a *speed* toggle, never a *content* toggle.

| Treatment | Default | `prefers-reduced-motion` |
|---|---|---|
| Press | scale + color shift | color shift only |
| Modal entry | fade + slide-up | instant render with scrim fade only |
| Toast entry | fade + slide-up | instant render |
| Tab swap | crossfade | instant swap |
| Skeleton | static | static (unchanged) |
| Identifier copy | background flash + tooltip + aria-live | tooltip + aria-live (announcement preserved) |
| Invalid-input shake | shake + color shift | color shift only |
| Route change | crossfade | instant swap |

---

## 8. Component primitives

### Button

- **Anatomy:** label · optional leading icon · optional trailing icon · optional kbd hint (right-aligned, `caption` size).
- **Sizes:** `sm` (28 px h, `space-3` x-padding), `md` (36 px h, `space-4` x-padding, default), `lg` (44 px h, `space-5` x-padding).
- **Variants:** primary · secondary · ghost · destructive.
- **States:** default · hover · focus-visible · active · disabled · loading.
  - **Loading:** label hidden behind a spinner that occupies its space; button width frozen at the pre-loading width to prevent layout jitter.
  - **Disabled:** `accent-primary-mu` for primary; opacity `0.5` for secondary/ghost; `cursor: not-allowed`; tooltip explains why when disabled by permission.
- **Tokens used:** `accent-primary*`, `text-on-accent`, `border-default`, `radius-md`, `space-3..5`, `dur-instant`, `dur-fast`.

### IconButton

- **Anatomy:** icon-only with a required accessible-name and tooltip.
- **Sizes:** `sm` (28 × 28), `md` (32 × 32, default), `lg` (40 × 40).
- **Variants:** ghost · subtle · accent · destructive.
- **States:** default · hover · focus-visible · active · disabled · loading.
- **Tokens used:** `radius-md`, `space-2`, `bg-surface`, `bg-elevated`.

### TextInput

- **Anatomy:** label (above) · field · helper / error (below) · optional leading prefix · optional trailing suffix.
- **Sizes:** `sm` (32 px h), `md` (40 px h, default).
- **Variants:** default · borderless (used inline in dense tables).
- **States:** default · hover · focus · filled · validating · disabled · readonly · error · success.
  - **Validating:** suffix shows a small `Clock` icon; `aria-busy="true"` on the wrapper; submit blocked until resolved.
  - **Error:** `border-default` → `status-error`; helper area renders error message + error icon, `aria-describedby` wired.
  - **Success:** `status-success` border briefly during async-validation success, fades back to `border-default` after `dur-normal`.
  - **Readonly:** `bg-code` background, `text-secondary` color, no border-focus on click; copy button as suffix.
- **Tokens used:** `border-default`, `border-focus`, `bg-surface`, `bg-code`, `status-error`, `status-success`, `dur-instant`.

### Select / Combobox

- **Anatomy:** trigger (looks like TextInput) · listbox panel (popover) · option rows · optional search filter · optional grouping headings.
- **Sizes:** `sm`, `md`.
- **Variants:** single-select · multi-select.
- **States:** default · hover · focus · open · filled · disabled · readonly · error · loading · empty (no matches after filter) · error-with-retry (options failed to load).
  - **Loading:** skeleton rows in the listbox.
  - **Empty (filtered):** "No matches for `<term>`" centered in the listbox.
  - **Error-with-retry:** in-listbox error message + retry button; trigger shows a subdued `status-error` border.
- **Tokens used:** `bg-elevated`, `border-default`, `radius-md`, `shadow-md`, `dur-fast`.

### Checkbox / Radio / Switch

- **Anatomy:** input · label (right) · optional helper.
- **Sizes:** `sm` (16 px), `md` (20 px, default).
- **States:** default · hover · focus · checked · indeterminate (checkbox only) · disabled · pending.
  - **Pending:** used by Switch tied to async settings; the knob shows a `Clock` glyph or progress-stroke ring; `aria-busy="true"`; toggling locked until resolved.
- **Tokens used:** `border-default`, `accent-primary*`, `radius-sm` (checkbox) / `radius-pill` (radio, switch knob).

### Tag / Chip

- **Anatomy:** sigil shape · optional leading icon · label · optional removable × (interactive variant).
- **Sizes:** `sm`, `md`.
- **Variants:** neutral · status-success · status-warn · status-error · status-info · status-pending · accent · context-instance · context-tenant.
- **States:** default · hover (interactive) · focus (interactive) · disabled.
- **Tokens used:** `radius-pill` or `radius-sm`, `bg-surface`, `bg-code`, status tokens, `space-1..2`.
- **Color independence:** every variant pairs a sigil shape with the color (status-error chips have a filled circle; status-success have a check sigil; status-warn have a triangle; status-info have an outlined circle; status-pending have a clock).

### IdentifierPill (Cypra-specific)

- **Anatomy:** Krypton-rendered identifier (middle-ellipsis truncation, fixed minimum `8 + … + 8` chars) · trailing copy IconButton · optional reveal toggle (for masked variants).
- **Sizes:** `sm` (inline, body-sm context), `md` (default), `lg` (page-header hero context).
- **Variants:** identifier · masked-secret · code-snippet (block-level — anatomy borrows from Code Block §9; the variant exists for in-table compactness only and falls back to Code Block for ≥ 240 px width).
- **States:** default · hover · focus · copied (transient, `dur-fast`) · disabled (read-locked) · ultra-narrow (container < 96 px wide → only the copy IconButton renders, with the identifier in an `aria-label` and a tooltip carrying the value).
- **Token usage:** all rendered text uses `text-identifier`. Background `bg-code`. Copy-flash uses `accent-primary-mu`. Radius `radius-sm`.

### MaskedSecret (Cypra-specific)

- **Anatomy:** prefix (e.g. `client_secret_`) · masked dots in `secret-mask` · reveal toggle · copy IconButton.
- **States:** default (masked) · revealed (visible for 30 s, auto-remask) · disabled · copied.
- **Copy semantics:** clicking copy while masked is **refused** with a tooltip "Reveal first to copy" — never silently copies dots, never silently copies the secret. Clicking copy while revealed copies the secret and announces "Copied <secret name>" via `aria-live="polite"`.
- **Tokens used:** `secret-mask`, `text-identifier`, `bg-code`, `radius-sm`.

### Tooltip

- **Anatomy:** triangle pointer · content (one line preferred, two max).
- **Sizes:** `body-sm` text. When content includes an identifier, the identifier portion uses `text-identifier` in Krypton inline.
- **Variants:** neutral · destructive · long-content (3+ lines, expands to 2-line wrap with content cap; beyond cap, content is truncated with the full value available via `aria-describedby` on the trigger).
- **States:** default · open. (Non-focusable; focus-visible on trigger reveals the same content via `aria-describedby`.)
- **Tokens used:** `bg-elevated`, `text-primary`, `border-emphasis`, `radius-sm`, `shadow-md`, `dur-instant`.

### Toast

- **Anatomy:** sigil icon · message · optional action button · optional dismiss IconButton.
- **Variants:** success · warn · error · info · pending (with progress hint).
- **Stacking:** maximum **3** simultaneous toasts. New toasts appear at the bottom of the stack. When a 4th arrives, the oldest dismisses immediately.
- **States:** entering · visible · paused (on hover) · dismissing.
- **Tokens used:** status tokens, `bg-elevated`, `border-emphasis`, `radius-md`, `shadow-lg`, `dur-fast`.
- **A11y:** `success`/`info` use `aria-live="polite"`; `warn`/`error` use `aria-live="assertive"`. **Form-validation errors take precedence**: when a form submit fails with field errors, the form's own polite live region announces the summary first, and any concurrent assertive toast is queued until the form announcement completes.

### Modal / Dialog

- **Anatomy:** scrim · panel (header · body · footer) · close IconButton.
- **Sizes:** `sm` (400 px), `md` (560 px, default), `lg` (720 px).
- **Variants:** standard · destructive · form (focus first input on open).
- **States:** entering · visible · loading-action (footer primary in loading state) · dismissing.
- **Tokens used:** `bg-elevated`, `bg-overlay`, `border-emphasis`, `radius-lg`, `shadow-lg`.
- **Focus management:** focus trap; focus restored on close; ESC closes (suppressed when destructive primary is in loading state).

### Popover / Dropdown / Menu

- **Anatomy:** anchor · panel · option rows · optional dividers · optional footer action.
- **States:** closed · opening · open · closing.
- **Tokens used:** `bg-elevated`, `border-default`, `radius-md`, `shadow-md`, `dur-fast`.
- **Keyboard:** arrow keys, Home/End, type-ahead, ESC closes, Tab moves focus out (does not cycle within the menu).

### Tabs

- **Anatomy:** tab bar · triggers (label + optional badge) · indicator (animated underline) · panel.
- **Variants:** standard underline · pill (used inside cards).
- **States:** default · hover · focus · active · disabled · overflow.
  - **Overflow:** when tab strip exceeds container width, strip becomes horizontally scrollable with `border-emphasis` fade-mask gradients on left/right; keyboard arrow keys scroll the active tab into view.
- **Tokens used:** `border-emphasis`, `accent-primary`, `dur-fast`.

### Card

- **Anatomy:** optional header (title · subtitle · actions) · body · optional footer.
- **Variants:** standard · interactive (hover lift; in dark mode the lift is a `border-emphasis` tier change instead of a shadow change) · highlighted (uses `accent-primary-mu` background) · disabled (interactive variant only — opacity `0.5`, `cursor: not-allowed`, tooltip on hover explaining why).
- **States:** default · hover (interactive) · focus (interactive) · disabled (interactive).
- **Tokens used:** `bg-surface`, `border-subtle`, `radius-md`, `shadow-sm`, `space-4..6`.

### Avatar

- **Anatomy:** image OR initials fallback OR sub-derived fallback.
- **Sizes:** `xs` (20), `sm` (24), `md` (32, default), `lg` (40), `xl` (64).
- **States:** default · loading (skeleton circle) · errored (initials fallback rendered) · no-name (initials fallback unavailable; renders the last 4 hex chars of the user's `sub` UUID in Krypton, monospace, on `bg-code`).
- **Tokens used:** `radius-pill`, `bg-code`, `text-secondary`.

### Skeleton

- **Anatomy:** shape primitives (line · circle · block) sized to the content they replace.
- **States:** static. (No shimmer, no pulse.)
- **Timing:** skeletons appear after `120 ms` of pending state to avoid flashing on fast responses.
- **Tokens used:** `bg-code` for the fill, `border-subtle` for outline.

### Toolbar / SegmentedControl

- **Anatomy:** segment buttons in a row, equal width or content-width.
- **States:** default · hover · focus · selected · disabled.
- **Tokens used:** `bg-code`, `accent-primary-mu`, `border-default`, `radius-md`, `dur-fast`.

### TenantSwitcher (Cypra-specific)

- **Anatomy:** trigger (Tenant glyph · tenant name · slug pill in `text-identifier` Krypton · chevron) · panel listing tenants the actor has access to · search input (when ≥ `10` tenants — threshold is a constant, not a token) · "Create tenant" footer action (instance admin only).
- **States:** default · hover · focus · open · loading · empty (instance admin post-bootstrap) · error (failed to load tenant list — panel renders error state with retry, trigger shows `status-error` left-border indicator).
- **Tokens used:** `bg-elevated`, `border-default`, `radius-md`, `shadow-md`, `text-identifier`.

### ContextBadge (Cypra-specific)

- **Anatomy:** sigil (Cypra glyph — see §6 for shape distinguishability) · context label (e.g. "Instance admin", "Tenant: acme").
- **Variants:** instance (uses `context-instance` token + Instance glyph) · tenant (uses `context-tenant` + Tenant glyph).
- **Color independence:** the Instance and Tenant glyphs are drawn as visually distinct **shapes** (a 2×2 dot grid vs. a square-bracket shape) — colorblind users distinguish via shape; the text label is the third reinforcement.
- **States:** static (informational only).
- **Tokens used:** `context-*`, `radius-pill`, `label-sm` (uppercase + tracking).
- **Placement:** top-bar of the dashboard, right of the logo.

### StatusPip (Cypra-specific)

- **Anatomy:** sigil shape (8 px) + trailing label.
- **Variants:** active (filled circle, `key-state-active`) · overlap (filled triangle, `key-state-overlap`) · sunsetting (outlined circle, `key-state-sunsetting`) · revoked (filled square, `text-tertiary`) · pending (clock, `status-pending`) · success (check, `status-success`) · warn (triangle, `status-warn`) · error (filled circle, `status-error`) · info (outlined circle, `status-info`).
- **Color independence:** every variant pairs a unique sigil shape with the color.

### KeyRotationTimeline (Cypra-specific)

- **Anatomy:** horizontal timeline · key segments (color-coded by state, sigil-shape-paired) · current-time indicator · segment labels (date, kid in Krypton).
- **Variants:** compact (in tenant overview) · full (in signing-keys page).
- **States:**
  - **Default (≥ 2 keys):** full timeline render.
  - **One-key:** single segment plus a "next rotation" forecast block to the right.
  - **Zero-key:** empty-state message "No active signing key — Cypra cannot mint OIDC tokens for this tenant" with a "Create signing key" CTA. Surfaced only via `cypra admin` recovery; should never appear in normal operation.
  - **Hover-on-segment:** tooltip with kid + activated_at + retires_at.
- **Tokens used:** `key-state-*`, `text-identifier`, `border-subtle`.

### AuditEntry (Cypra-specific)

- **Anatomy:** time (relative + absolute on hover, tabular-numeric) · actor (avatar + name [Neon, `text-primary`] + role badge) · action (verb chip) · resource (IdentifierPill at `identifier-sm`) · expand IconButton revealing `state_before` / `state_after` diff.
- **States:** default · hover · expanded · redacted (PII fields rendered as `*** redacted (gdpr-…) ***` in `secret-mask` color, matching MaskedSecret's redaction semantic).
- **Tokens used:** `bg-surface`, `border-subtle`, `bg-code`, `text-identifier`, `secret-mask`.

### BackupCodeGrid (Cypra-specific)

- **Anatomy:** 5 × 2 grid of one-time codes (Krypton, `identifier-md`) · per-code copy · "Copy all" action · "Download .txt" action · "I've saved these" confirmation gate.
- **Navigation safety:** the screen registers a `beforeunload` warning ("Your backup codes are shown only once. Continue without saving?") and intercepts the browser back button with the same confirmation modal. ESC and route-change links inside the screen are disabled until the confirmation gate is satisfied; the only escape paths are explicit confirm or accept-the-warning navigation.
- **States:** unconsumed · consumed (struck through, `text-disabled`) · all-saved-confirmed.
- **Regeneration semantics:** when a user regenerates backup codes, all previously-issued codes are invalidated **immediately** (no grace window). The dashboard surfaces this with a destructive confirmation dialog: "Regenerating will invalidate your existing backup codes immediately."
- **Tokens used:** `bg-code`, `text-identifier`, `radius-sm`, `space-3`.

### PermissionMatrix (Cypra-specific)

- **Anatomy:** roles (rows) × permissions (columns) · checkbox-style sigils · disabled cells for system-defined permissions · sticky bottom save bar (Card-style, `shadow-lg`, surfaces only when dirty, contains "Discard" + "Save" actions and a dirty-row count).
- **States:** default · dirty (save bar surfaces) · saving · saved · error (in save bar, with retry) · read-only (member role viewing — checkbox sigils render as read-only, save bar hidden, info banner at top "You can view this matrix but not edit it.") · permission-denied (non-tenant-admin lands on a 403 — see §10 Error pages).
- **Tokens used:** `bg-surface`, `border-default`, `accent-primary`, `text-tertiary`.

### SetupTokenBanner (Cypra-specific)

- **Anatomy:** large IdentifierPill rendering the bootstrap token · "redacted-on-export" advisory caption · "Copy" + "I have copied this — show next step" actions.
- **States:**
  - **Visible:** rendered on `/setup/<token>` first visit.
  - **Consumed:** route 404s; the component is unreachable by URL. (This is the route-level resolution; the component itself never renders in a "consumed" state.)
- **Tokens used:** `bg-code`, `border-emphasis`, `status-warn` (advisory caption color).

### ProviderConfigCard (Cypra-specific)

- **Anatomy:** provider sigil + name · current-config summary (kind, from-address, last-used) · status pip (configured / required / failing) · primary action (Configure / Test send / Replace).
- **Variants:** email-provider · upstream-provider · storage-provider. Variant differentiation is **content-only** (the sigil, the summary fields, the test action's behavior); chrome and tokens are identical across variants.
- **States:** unconfigured (with "required" `status-error` pip if mail-dependent and unset) · configured-healthy · configured-failing · testing (loading on test action) · testing-while-dirty (form has unsaved changes; test action is disabled with tooltip "Save changes before testing.").
- **Tokens used:** `bg-surface`, `border-default`, status tokens.

### MobileBlockedBanner (Cypra-specific)

- **Anatomy:** a subdued Card-style block at the top of the dashboard viewport when width < md. Sigil (`Monitor` Lucide icon, `text-tertiary`) · single-sentence body ("Cypra dashboard is optimized for desktop. Some features may be cramped.") · dismiss IconButton.
- **States:** visible (until dismissed in this session) · dismissed (suppressed for the rest of the session).
- **Tokens used:** `bg-surface`, `border-subtle`, `text-secondary`. Same anatomy as a quiet inline alert; not a Toast (does not auto-dismiss).

### CodeBlock

- **Anatomy:** language label (caption, `text-tertiary`) · Krypton-rendered code (code-md) · copy IconButton · optional "Copy as cURL" toggle.
- **"Copy as cURL" semantics:** toggle persists per-snippet within the page session; resets on route change. Default is the language-native form (Go / TS / shell).
- **Used in:** OIDC client config snippets, operator-playbook embeds in dashboard, in-product docs strips.

---

## 9. Composite components

### Page Header

- **Anatomy:** breadcrumb (when ≥ 2 levels deep) · title (display-md) · optional subtitle (body-md, `text-secondary`) · primary action (right-aligned) · optional secondary actions (overflow menu).
- **States:** default · loading (skeleton title and breadcrumb) · error (renders Error State in place of body, header retained).

### Sidebar Nav

- **Anatomy:** brand row (Cypra logo + ContextBadge) · TenantSwitcher · scoped-nav groups (Overview · Users · Projects · Auth methods · Members · Audit · Settings) · instance-admin-only group below a divider (Tenants · Storage · Audit (instance) · Diagnostics) · footer (account avatar + theme toggle + sign-out).
- **Variants:** expanded (240 px, default for new sessions) · collapsed (56 px, icon-only with required tooltips on hover/focus, accessible-name on every item).
- **States:** default · hover · focus · active (current route) · permission-denied (renders disabled with `Lock` Lucide sigil + tooltip explaining the missing permission).

### Breadcrumb

- **Anatomy:** root link · `/` separator · ancestor links · current segment (non-link).
- **States:** default · truncated (middle ancestors collapse to "…" with popover when 4+ levels).

### Empty State

- **Anatomy:** 24 px Lucide icon (`text-tertiary`) · headline (display-sm) · supporting sentence (body-md, `text-secondary`) · single primary action.
- **Pattern:** acknowledge the absence, explain in one sentence, offer one action. **Editorial copy is forbidden.**

### Error State

- **Anatomy:** 24 px error icon (`status-error`) · headline (display-sm) · what / why / what to do (body-md, max 2 sentences combined) · retry action · optional "View logs" link.
- **Pattern:** what + why + what to do. No apologies, no "oops", no exclamation marks, no "this shouldn't happen".

### Loading State

- **Anatomy:** static skeletons matching the populated-state shape.
- **Rule:** skeletons for known shapes (lists, tables, cards); spinners only for **unknown** durations (initial app boot, OIDC token mint pending, mail send pending). **Skeletons do not animate.**
- **Timing:** skeleton appears after `120 ms` of pending state.

### Confirmation Dialog

- **Anatomy:** sigil icon (`status-warn` or `status-error`) · headline · body · destructive primary + cancel secondary in footer · for high-stakes destructive ops (tenant delete, key force-rotate, master-key rotation), requires typing the resource identifier (slug or `kid`) — typed input lives in a Krypton-rendered TextInput with case-sensitive exact match including whitespace trimming on both ends.
- **States:** default · confirming (typed match validated) · in-flight (primary loading) · error (in-line error above footer with `Try again` action).

### Settings Row

- **Anatomy:** label · helper (body-sm, `text-secondary`) · control (right-aligned).
- **Variants:** input · switch · select · button-action · disclosure (expandable for nested settings) · branding-toggle (a Switch + a small `Eye` IconButton that opens a hosted-login preview in a Modal).

### List Row

- **Anatomy:** leading avatar/icon · primary text · meta (right) · trailing actions IconButton group.
- **Action visibility:** trailing actions are revealed on row hover (cursor surfaces). On touch surfaces, actions are persistently visible at `40 × 40 px` minimum tap target. The `32 × 32 px` IconButton size is used **only on cursor surfaces** when the row itself is the click target.
- **States:** default · hover · focus · selected · loading (skeleton in primary slot).

### Save Bar

- **Anatomy:** sticky-bottom Card that surfaces only when a form is dirty. Left: dirty-row count (e.g. "3 unsaved changes"). Right: "Discard" secondary + primary action (e.g. "Save").
- **Used in:** PermissionMatrix, branding settings, redirect-URI editor.
- **States:** hidden · visible-dirty · saving · saved (renders briefly with success sigil, then auto-hides over `dur-slow`) · error.

---

## 10. Screen inventory

Every PLAN §6 must-have surface gets a screen. Each section enumerates intent, hierarchy, and every state.

### Setup Wizard (`/setup/<token>`)

- **Intent:** turn a raw install into a usable Cypra instance.
- **Hierarchy:** primary action: *redeem token*; secondary: *cancel + revoke*; content: token confirmation, then passkey enrollment, then backup codes, then "create your first tenant" CTA.
- **States:**
  - **Default (token-valid):** SetupTokenBanner with the in-band token confirmation, "Continue" primary.
  - **Token-expired:** Error State explaining how to re-issue (`cypra admin reset-bootstrap`).
  - **Token-already-consumed:** Error State with the same recovery path.
  - **Loading:** skeleton for the banner during initial verification.
  - **Passkey-enrollment-failed:** in-line error in the WebAuthn step with retry.
  - **Backup-codes:** BackupCodeGrid; the screen cannot be left without confirming "I have saved these" (see component for `beforeunload` + back-button handling).
- **Sub-surfaces:** confirmation dialog if the operator tries to navigate away mid-bootstrap.

### Hosted Login (`/login`)

- **Intent:** authenticate an end-user against the tenant.
- **Hierarchy:** tenant logo + display name (top), then identifier input (email), then sign-in method picker (passkey / password / magic link / Google), then tenant-configurable footer.
- **Tab order:** tenant logo (skipped — non-interactive) → email field → primary method button (passkey if enrolled, else password) → secondary methods in fixed order (password → magic-link → Google) → "Forgot?" link → "Create account" link → footer "Powered by" link (when shown).
- **States:**
  - **Default:** identifier field focused, method buttons below.
  - **Method-specific:** password reveals a password field; passkey triggers WebAuthn; magic-link shows "Check your email."
  - **2FA-required:** TOTP / WebAuthn-2FA challenge (lands on `/2fa`).
  - **Rate-limited:** uniform "Too many attempts. Try again in N minutes." with countdown.
  - **Bot-mitigation challenge (v1.1):** placeholder slot reserved for verifier widget.
  - **Error (uniform):** "Sign in didn't work. Check your details and try again." — no enumeration.
  - **Loading:** spinner inside the active sign-in button (no full-screen blocker).
  - **Email-provider-not-configured (tenant misconfig):** end-user sees the uniform error; the operator-facing tenant-admin testing the login sees a developer-detail error if signed in as admin.

### Hosted Login Auxiliary Screens

Compact §10 entry covering the four auxiliary hosted-login routes — they share anatomy and state shape.

#### Sign Up (`/signup`)
- **Intent:** create a new end-user account in the tenant.
- **Hierarchy:** tenant logo + display name · email field · "Continue" primary · "I already have an account" link.
- **States:** default · loading · error (uniform) · rate-limited · email-already-exists (uniform with "Sign in instead" link).

#### Verify Email (`/verify`)
- **Intent:** consume a single-use verification token.
- **Hierarchy:** tenant logo · "Verifying…" spinner → outcome.
- **States:** verifying · verified (success copy + "Continue") · expired (Error State with "Send a new link") · already-consumed (Error State).

#### Reset Password (`/reset`)
- **Intent:** request or consume a password-reset token.
- **Hierarchy (request flow):** tenant logo · email field · "Send reset link" primary.
- **Hierarchy (consume flow):** tenant logo · new-password field · confirm-password field · "Set password" primary.
- **States:** default · loading · sent (success copy with "Check your email") · invalid-or-expired-token (Error State) · validation-failed (inline error explaining the deny-list rule that fired).

#### 2FA Challenge (`/2fa`)
- **Intent:** complete second-factor authentication after a primary factor succeeded.
- **Hierarchy:** tenant logo · factor picker (TOTP · WebAuthn-2FA · backup code) · factor-specific input · "Verify" primary.
- **States:** default · loading · error (uniform) · rate-limited · no-factors-enrolled (Error State with admin-recovery instruction).

### Hosted Consent (`/oidc/consent`)

- **Intent:** capture consent for an OIDC client.
- **Hierarchy:** tenant logo + name · "Sign in to <client_name>" · scope list (with plain-language descriptions) · "Allow" + "Deny" actions.
- **States:**
  - **Default (first-time):** scope list visible, both actions enabled.
  - **Returning (already-consented):** auto-redirect, no UI surfaced.
  - **Scope-upgraded:** consent previously granted but the client now requests additional scopes; the new scopes are highlighted with a `+` sigil and grouped under a "New permissions requested" subhead. User must re-consent.
  - **Loading:** skeleton scope list.
  - **Error:** "Couldn't reach the sign-in service. Try again."

### Error Pages (404 / 403 / 500 / 503)

Shared Error State anatomy with route-specific copy. All four pages render under the dashboard chrome when authenticated; under a minimal hosted-login chrome when on a tenant subdomain unauthenticated.

#### 404 Not Found (`*`)
- **Copy:** "We couldn't find that page." + body "The link may be wrong, or this resource was deleted." + actions "Go to dashboard" / "Go back".

#### 403 Forbidden (`*`)
- **Copy:** "You don't have access to this." + body "Ask the tenant owner for the `<missing-permission>` permission." + actions "Go to dashboard" / "Sign out".
- **Used by:** non-instance-admin landing on `/dashboard/instance/*`; tenant member landing on owner-only surfaces.

#### 500 Server Error (`*`)
- **Copy:** "Something on Cypra failed." + body "The error has been logged with request ID `<request_id>` (rendered in IdentifierPill, copy-affordant). Tell your operator." + action "Go back".

#### 503 Maintenance (`*`)
- **Copy:** "Cypra is briefly unavailable." + body "Migrations or master-key rotation may be in progress. Try again in a minute." + no action (page auto-retries `/readyz` every 30 s and unblocks when the service is ready).

### Dashboard Overview (`/dashboard`)

- **Intent:** the actor's landing page; surfaces what they have access to and current health.
- **Hierarchy:** ContextBadge (instance vs tenant), then tile grid: tenant count (instance-admin) / project count / user count / signing-key health / recent audit entries.
- **States:**
  - **Loading:** tile skeletons.
  - **Empty (post-bootstrap):** "Create your first tenant" prominent CTA; other tiles hidden.
  - **Populated:** stat tiles + last 10 audit entries.
  - **Error (some tiles):** failed tiles render Error State in-place; healthy tiles unaffected.
  - **Permission-denied tiles:** rendered disabled with tooltip explaining the missing permission.
  - **No-permissions:** if the actor has zero permissions on the current tenant, the page renders an Empty State: "You're a member of this tenant, but you don't have permissions to see anything yet. Ask the tenant owner."

### Tenant List (`/dashboard/tenants`, instance-admin)

- **Intent:** browse and manage all tenants on the install.
- **Hierarchy:** primary action *Create tenant*; content: searchable, sortable table.
- **States:** default · loading · empty (post-bootstrap; CTA = create tenant) · search-empty ("No tenants match `<query>`") · error · permission-denied (non-instance-admin → 403 page).

### Tenant Detail / Settings (`/dashboard/tenants/<slug>` and nested)

- **Intent:** configure a single tenant.
- **Hierarchy:** tenant header (display name, slug pill, member count, ContextBadge) · tab nav.
- **Tab structure (matched 1:1 to routes in §11):**

| Tab | Route |
|---|---|
| Overview | `/dashboard/tenants/<slug>` |
| Projects | `/dashboard/tenants/<slug>/projects` |
| Users | `/dashboard/tenants/<slug>/users` |
| Auth methods | `/dashboard/tenants/<slug>/auth-methods` |
| Signing keys | `/dashboard/tenants/<slug>/signing-keys` |
| Audit | `/dashboard/tenants/<slug>/audit` |
| Settings | redirects → `/dashboard/tenants/<slug>/settings/branding` |

  Settings sub-tabs (under `/settings/`): Branding · Email · Upstream · Members · API tokens · Danger.

- **States:** per-tab default / loading / empty / error / read-only (member role) / permission-denied (per permission).
- **Sub-surfaces:** confirmation dialog for tenant delete (typing slug to confirm).

### Branding tab (`/dashboard/tenants/<slug>/settings/branding`)

- **Intent:** configure the tenant accent, logo, display name, and "Powered by Cypra" toggle.
- **Hierarchy:** SettingsRow `branding-toggle` for "Show 'Powered by Cypra' on hosted login" (default ON for free OSS; preview opens hosted-login Modal preview); SettingsRow `input` for display name; SettingsRow `input` (file picker) for logo (validation: SVG sanitized at upload, PNG ≥ 256 × 256, accepted aspect ratios square / 16:9 / 21:9); SettingsRow `input` (color picker, hex) for accent — with the live tenant-accent validation gate from §3 ("Accent must be readable on both white and your accent's text color").
- **States:** default · dirty (Save Bar surfaces) · saving · saved · validation-failed (inline error on the failing field).

### Project List + Project Detail

- **Intent:** create and configure OIDC clients (one per project at v1).
- **Hierarchy:** project list = simple table. Detail = OIDC client config: issuer URL (IdentifierPill, copy), client_id (IdentifierPill), client_secret (MaskedSecret), redirect URIs editor, allowed scopes editor, token-endpoint auth method picker, CodeBlock snippets for Auth.js / NextAuth.
- **States:** default · loading · empty · error · read-only (member) · permission-denied · secret-rotation-in-progress (banner: "Rotation in progress — downstream apps using the previous secret will fail at the token endpoint after 5 minutes. Coordinate the cutover with your app owner.").
- **Sub-surfaces:** *Rotate client secret* modal (warns about downstream), *Delete project* modal (typing slug).

### User List (`/dashboard/tenants/<slug>/users`)

- **Intent:** find a user.
- **Hierarchy:** primary actions: *Invite user* · *Import via CLI* (opens guidance modal at v1: copy-paste-able `cypra import` command + a link to docs); content: searchable table.
- **States:** default · loading · empty (CTA: invite or use CLI import) · error · search-empty ("No users match `<query>`").

### User Detail (`/dashboard/tenants/<slug>/users/<id>`)

- **Intent:** view and manage one user.
- **Hierarchy:** identity card (avatar · email · `sub` IdentifierPill · enrolled methods chips) · tabs (Auth methods · Sessions · Consents · Audit · Metadata).
- **States:** per-tab default / loading / empty / error · `gdpr.user_deletion_in_progress` banner if a DSR delete is mid-flight.
- **Sub-surfaces:** confirmation dialogs for *Reset password* (`user.password.reset`), *Disable MFA* (`user.mfa.disable`), *Enroll factor on user's behalf* (`user.factor.enroll`), *Delete user (DSR)*, *Export user data*.

### Signing Keys (`/dashboard/tenants/<slug>/signing-keys`)

- **Intent:** view rotation state, force-rotate.
- **Hierarchy:** KeyRotationTimeline (full variant) · key list table (kid pill · state pip · activated_at · retires_at · sunset_until) · primary action *Rotate now*.
- **States:** default (≥ 1 active key) · one-key (no overlap visible — KeyRotationTimeline shows the next-rotation forecast block) · zero-key (empty state — should not occur in normal operation) · loading · error · post-rotation (success Toast + timeline updates) · permission-denied.
- **Sub-surfaces:** confirmation dialog requiring typed `kid` to force a rotation.

### Audit Log Viewer (per-tenant + instance-level)

- **Intent:** review what happened, when, by whom.
- **Hierarchy:** filter bar (date range · action · actor · resource_kind) · streaming list of AuditEntry rows · primary action *Export NDJSON*.
- **Streaming policy:** initial render is a paged fetch; live updates poll every 10 s while the page is visible.
- **States:** default · loading · empty (after filter, "No entries match these filters") · error · expanded-entry · redacted-entry · stream-disconnected (silent banner: "Live updates paused — last sync `<time>`. Reconnecting…" with a manual refresh action).

### Email / Upstream / Storage Provider Config

- **Intent:** configure a provider.
- **Hierarchy:** ProviderConfigCard at top · form below · diagnostic action ("Send test email" / "Try OAuth round-trip" / "Try storage write").
- **States:** unconfigured · configured-healthy · configured-failing · testing · testing-while-dirty (test disabled, tooltip "Save changes before testing.") · save-success (Toast) · save-error.

### Members & Roles (`/dashboard/tenants/<slug>/settings/members`)

- **Intent:** invite, change roles, remove members.
- **Hierarchy:** primary action *Invite member*; content: list of (avatar · email · role select · last seen · remove IconButton); PermissionMatrix below the member list.
- **States:** default · loading · empty (only the actor — copy: "You're the only admin. Invite someone.") · error · invitation-pending (chip on the row) · permission-denied · last-admin-self-action-blocked (the actor cannot remove themselves or downgrade their own role if they are the last owner; the action is disabled with tooltip "You're the last owner. Promote someone else first.").

### Account / Profile (`/dashboard/account`)

- **Intent:** the actor's own settings — passkeys, 2FA, sessions, PATs, theme.
- **Hierarchy:** account header · Passkeys section · 2FA section · Active sessions list (with sign-out-others) · PATs section.
- **States:** per-section default / loading / empty / error.
- **Last-passkey self-removal guard:** when the actor has only one passkey enrolled and no password fallback, the remove action is disabled with tooltip "This is your only sign-in method. Add another before removing this one."
- **Last-PAT-of-its-kind:** no special guard at v1; PATs revoke freely.
- **Sub-surfaces:** modal — *Add passkey*, modal — *Regenerate backup codes* (BackupCodeGrid, with the immediate-invalidation warning), modal — *Create PAT* (one-time-shown PAT in IdentifierPill with a `beforeunload` warning + "I have copied this token" confirmation gate matching BackupCodeGrid's pattern — the PAT shown only once must obey the same one-shot UX as the backup codes).

### Instance Admin Surfaces (`/dashboard/instance/...`, instance-admin only)

- **Pages:** Admins (list, invite, demote) · Storage (storage-provider config) · Audit (instance-level) · Diagnostics (health probes, `cypra version`, migration state, master-key-rotation phase).
- **States:** standard (default · loading · error · permission-denied for non-instance-admin who somehow lands on these routes — renders 403 page).

### No marketing surfaces at v1

No landing page, no docs site embedded, no pricing page. Docs live in the repo as Markdown.

---

## 11. Navigation

### Information architecture

**Persistent left sidebar nav** is the primary pattern on the dashboard. The sidebar is scoped to the current tenant context; switching tenants happens via TenantSwitcher at the top of the sidebar. Instance-admin nav lives in a divided section at the bottom, visible only to instance admins. **Tabs within a page** carry sub-navigation.

Hosted-login surfaces have **no nav**.

### Route tree

```
# Dashboard (authenticated)
/dashboard                                                   # overview
/dashboard/account                                           # actor's own settings
/dashboard/tenants                                           # instance admin only
/dashboard/tenants/<slug>                                    # tenant overview
/dashboard/tenants/<slug>/projects
/dashboard/tenants/<slug>/projects/<slug>
/dashboard/tenants/<slug>/users
/dashboard/tenants/<slug>/users/<id>
/dashboard/tenants/<slug>/auth-methods
/dashboard/tenants/<slug>/signing-keys
/dashboard/tenants/<slug>/audit
/dashboard/tenants/<slug>/settings                           # → redirects to /settings/branding
/dashboard/tenants/<slug>/settings/branding
/dashboard/tenants/<slug>/settings/email
/dashboard/tenants/<slug>/settings/upstream
/dashboard/tenants/<slug>/settings/members
/dashboard/tenants/<slug>/settings/api-tokens
/dashboard/tenants/<slug>/settings/danger
/dashboard/instance/admins                                   # instance admin only
/dashboard/instance/storage                                  # instance admin only
/dashboard/instance/audit                                    # instance admin only
/dashboard/instance/diagnostics                              # instance admin only

# Hosted login (unauthenticated, per-tenant subdomain)
/login
/signup
/verify
/reset
/2fa
/oidc/authorize         # OAuth/OIDC entry
/oidc/consent
/error                  # generic hosted-login error landing

# Bootstrap (one-shot, install-level)
/setup/<token>

# Error pages (any host)
/__cypra/404            # rendered for any unmatched route
/__cypra/403
/__cypra/500
/__cypra/503
```

### Breadcrumb policy

Breadcrumbs appear on every page **two or more levels deep** under the dashboard root. Single-level pages (`/dashboard`, `/dashboard/account`) omit them. Hosted-login pages never use breadcrumbs.

### Deep-linking

- Every dashboard page is deep-linkable; route state survives reload.
- Tab state in tabbed pages reflects in the URL (sub-routes for primary tabs; `?tab=…` only inside the Settings sub-area).
- Audit Log filter state reflects in the URL query for shareable filtered views.
- The Setup Wizard route is single-use; the route 404s once the token is consumed.
- Hosted-login routes preserve the `continue` parameter end-to-end.

---

## 12. Accessibility

- **Contrast:** every token combination meets WCAG AA per §3 contrast obligations. AAA where listed. **Tenant-supplied accents are validated for AA against `text-on-accent` AND ≥ 3:1 against `bg-canvas` at config time.** `border-focus` is a system invariant — not affected by tenant theming, ensuring focus-ring contrast is always met.
- **Focus:** focus-visible ring is `border-focus` color, **2 px** width, **2 px** offset, radius matches the focused element + **2 px**. Mouse-only focus suppressed; keyboard focus is always visible. IdentifierPill, MaskedSecret, IconButton, and the typed-confirm TextInput in Confirmation Dialog all participate in this contract.
- **Keyboard:** every primary flow is keyboard-completable. Documented shortcuts:

  | Shortcut | Action |
  |---|---|
  | `cmd/ctrl + k` | Open command palette |
  | `cmd/ctrl + /` | Open keyboard shortcut overlay |
  | `g` then `o` | Go to overview |
  | `g` then `t` | Go to tenants (instance admin) |
  | `g` then `u` | Go to users (current tenant) |
  | `g` then `p` | Go to projects (current tenant) |
  | `g` then `a` | Go to audit (current tenant) |
  | `g` then `s` | Go to settings (current tenant) |
  | `/` | Focus search in current page |
  | `c` | Primary "create" action on current page (where applicable) |
  | `escape` | Close any overlay (modal, popover, drawer) |
  | `shift + ?` | Open keyboard shortcut overlay |
  | `tab` / `shift + tab` | Move focus forward / backward |
  | `↑` / `↓` | Move focus within lists, menus, tables |
  | `enter` | Activate focused control |
  | `space` | Toggle focused checkbox / switch / disclosure |

  The shortcut overlay is also reachable from a `?` IconButton in the dashboard footer (alternative entry-point for keyboards where `shift + ?` and `cmd + /` are awkward — non-US, non-Latin, etc.).

  Hosted-login surfaces use only standard form keyboarding (no chord shortcuts).

- **Screen reader:**
  - **Toast pattern:** `aria-live="polite"` for success/info; `aria-live="assertive"` for warn/error. **Form-validation errors take precedence**: when a form fails to submit, its own polite live region announces the summary first; concurrent assertive toasts queue until the form's announcement completes.
  - **Validation pattern:** `aria-invalid="true"` + `aria-describedby` to the error message; live region announces "form has errors" once on submit.
  - **Async pattern:** pending button announces "loading"; on completion announces the result ("Saved" / "Failed: <reason>").
  - **Identifier pronunciation:** every IdentifierPill carries an `aria-label` of the form `"<identifier kind>: <full value>"` (e.g. `"Client ID: cypra_acme_p7q9a3"`). The screen reader reads the full string from the `aria-label`, not character-by-character from the visual Krypton text. Krypton's mechanical glyphs are visual-only.
  - **Copy confirmation:** "Copied <identifier name>" announced via `aria-live="polite"`.
  - **WebAuthn flows:** Cypra adds visible hint text matching the browser's native UI ("Touch your security key" / "Use your fingerprint").
  - **Route changes:** the SPA writes the new page title to a dedicated `<div role="status" aria-live="polite">` region inserted at the document root on app boot. Implementation guidance: announce the new `<h1>` value, fire after route's first paint.

- **Motion:** every motion treatment in §7 has a reduced-motion fallback.

- **Tap targets:**
  - Default minimum: **40 × 40 px** on touch surfaces.
  - **`32 × 32 px` IconButton relaxation applies only on cursor surfaces (mouse-primary)** when the IconButton is inside a List Row whose row itself is the click target. Touch surfaces always get `40 × 40 px`.

- **Color independence:** every status conveys meaning via *both* color and a sigil shape. ContextBadge variants (instance vs tenant) use shape-distinct glyphs (2×2 dot grid vs square-bracket shape), the text label, AND color — three reinforcements.

- **Form labels:** every input has a visible label; placeholder is never the label.

- **Live regions:** route changes announce the new page title via the polite live region above. Pending-state changes (form `validating`, button `loading`) update the same control's `aria-busy` attribute.

---

## 13. Responsive behavior

| Breakpoint | Min width | Behavior summary |
|---|---|---|
| sm | 0 px | Hosted login: full content; dashboard: degraded with MobileBlockedBanner. |
| md | 768 px | Dashboard: sidebar collapses to a top-bar drawer; tab-bars become scroll-overflow. |
| lg | 1024 px | Dashboard: sidebar expanded by default; tables show full column set. |
| xl | 1280 px | Dashboard: max content width caps at 1280 px; whitespace expands to canvas edge. |

Per major surface:

- **Dashboard sidebar** — `< md`: drawer; `md`: collapsed (icon-only with required tooltips); `≥ lg`: expanded. User toggle persists.
- **Dashboard tables** — `< md`: column-stack mode (each row collapses to a card with primary text + meta). `≥ md`: scroll-x with sticky first column. `≥ lg`: full columns visible.
- **Audit Log specifically** — uses **AuditEntry compact mode** at `< md` (actor + action + relative time only; tap to expand) instead of column-stack. The `< md` table column-stack rule is the default; Audit Log opts into compact mode because its row anatomy is already vertical-friendly. This is the single named exception to the default; no other table opts out.
- **Hosted login** — `< sm` (i.e. all widths < 0 px — never; hosted login is mobile-first by default at all widths): 100% width with `space-4` padding **up to 440 px**; centered above 440 px. Phrased correctly: **always** capped at 440 px content width; below 440 px viewport it's 100% width with `space-4` padding.
- **Setup wizard** — same pattern: capped at 560 px content width; below 560 px viewport, 100% width with `space-4` padding.
- **Tabs within pages** — `< md`: horizontal scroll with fade indicators; `≥ md`: visible inline.
- **MobileBlockedBanner** — visible on the dashboard only at `< md`, dismissible per session.

The dashboard is **explicitly desktop-primary** and v1 ships with the MobileBlockedBanner. Hosted login is mobile-first.

---

## 14. Content style

- **Voice and tone:** terse, technical, precise. Confidence without ceremony. **No exclamation marks. No apologies. No "oops". No "this shouldn't happen". No "we're sorry".** The product is calm.
- **Sentence length cap in errors:** max 14 words per sentence; total error copy ≤ 2 sentences combined. The all-mono typeface widens characters; this cap keeps errors on two lines or fewer at the dashboard's narrowest column.
- **Capitalization:** **sentence case** for buttons, labels, helpers, table headers ("Add member", not "Add Member"). **Title case** only for proper nouns and page-title-style headings ("OIDC Signing Keys" — title case because the page is named after a domain term). Identifiers always render in their literal case.
- **Empty-state copy:** "No <thing> yet." + one sentence on what the thing is + one action verb. *Example:* "No projects yet. A project lets an app authenticate against this tenant. Create project."
- **Error message copy patterns** — what + why + what to do. No apologies. Examples (each ≤ 2 sentences, ≤ 14 words/sentence):
  - "Sign in didn't work. Check your details and try again."
  - "Couldn't save the upstream provider. The Google client secret is empty. Paste it again from the Google Cloud Console."
  - "Couldn't rotate the signing key. The tenant has no active key. Create one before rotating."
  - "Couldn't reach the storage backend. Profile pictures may be temporarily unavailable. Try again or check the storage config."
- **Date / number / currency:**
  - Relative dates for events within 7 days ("2 minutes ago", "3 hours ago", "yesterday").
  - Absolute for older: ISO 8601 with second resolution and UTC — `2026-05-04 14:23:01 UTC`. Tooltip on relative dates shows the absolute.
  - Numbers: thousands grouping with thin space (US-locale formatting at v1; i18n deferred).
  - Bytes: binary IEC ("1.4 MiB") on technical surfaces; no decimal SI.
  - No currency at v1.
- **Identifier display:** middle-ellipsis truncation, fixed minimum `8 + … + 8`; full value via `aria-label` and tooltip; click copies.
- **Password strength UX:** no strength meter (Argon2id is the parameter authority — meters lie); minimum length and a deny-list of common passwords enforced server-side, with the failing rule surfaced as the inline error.

---

## 15. Asset inventory

- **Logos:**
  - Cypra wordmark (Neon weight 600, custom kerning) — dashboard sidebar, default hosted-login footer.
  - Cypra mark (a single Krypton glyph, stylized) — favicon size and collapsed sidebar.
  - Sizes: wordmark vector + raster at 32, 48, 64, 128 px; mark vector + raster at 16, 32, 48, 64, 128, 256, 512 px.
  - Lockups: wordmark-only · mark-only · stacked (used in setup wizard hero).
  - Clear-space: minimum padding equal to the wordmark's x-height.
- **Tenant logos:**
  - Accepted: SVG (preferred), PNG (≥ 256 × 256, transparent background recommended).
  - Aspect ratios: square, 16:9, or 21:9 (wider rejected).
  - Sanitization: SVGs scrubbed of `<script>`, foreign elements, external `href` references at upload time.
- **Favicons:** Cypra mark at 16 × 16 (ICO), 32 × 32 (PNG), 192 × 192, 512 × 512, plus an SVG mask-icon. Per-tenant favicon override deferred to v1.1 (open question §16).
- **Open Graph:** default 1200 × 630 PNG with the mark + wordmark + tagline. Per-tenant OG override deferred to v1.1.
- **App icons:** N/A at v1.
- **Loading / placeholder imagery:** none. Skeletons fill the role.
- **Email templates:** plain-text and minimal-HTML variants for magic-link, password-reset, email-verification, admin-invite, breach-notification (operator-driven). Templated in the same Krypton/Neon system; tenant logo + accent applied.

---

## 16. Open questions

Resolutions captured here; remaining questions land in TASKS.md as deferrals.

- ✅ **Density toggle:** closed as out of scope at v1. Densities are hardcoded per surface.
- ✅ **MaskedSecret reveal-pin:** closed as out of scope at v1. Secrets auto-remask after 30 s; no pin.
- ✅ **Backup-code regeneration:** closed — old codes invalidate **immediately** on regenerate; documented in BackupCodeGrid and the operator playbook.
- ✅ **Last-admin self-action guard:** captured in Members and Account screens.
- [ ] **Per-tenant favicon support.** Deferred to v1.1; needs an asset-pipeline story for arbitrary tenant images.
- [ ] **Tenant impersonation by instance admin.** Useful for support; security-sensitive (every action audited as impersonation, separately). Deferred to v1.1.
- [ ] **Per-tenant Open Graph image override.** Same shape as logo override; v1.1.
- [ ] **Default theme (light/dark) per tenant on hosted-login pages.** System-preference is the v1 default; tenants may want to pin. Add as a tenant branding setting in v1.1.
- [ ] **Command palette content.** v1 ships `cmd+k` open behavior; the palette content (jump-to-tenant, jump-to-user-by-email, recent-actions) is not yet enumerated.
- [ ] **In-dashboard CSV import UI.** Explicitly v1.1; v1 ships the CLI-guidance modal with `cypra import` instructions and a screencast embed.
- [ ] **OIDC consent screen scope-language.** Cypra-supplied default copy at v1; tenant override at v1.1.

---

## Revisions

<!-- After DESIGN.md is sealed, edits land here as a dated entry. Do not silently mutate sealed sections. -->
