# Style guide

The visual system for Treningheten's server-rendered frontend. This doc is
**authoritative** — when building or restyling UI, consume these tokens and components
rather than hardcoding values or writing inline `style="…"`. It is a living reference;
the phased rollout that produced it is tracked in the Style guide entry of [`wip.md`](wip.md).

**The short version:** load values from `--…` tokens (never hardcode a hex/px); reach for
an existing component (`.btn`, `.stat-card`, `.tag-chip`, the `.trm` modal) before inventing
one; use `.u-*` utilities for one-off spacing/alignment; keep it **light and calm** (see
Design language). New rules go in the right layer (see Where things live).

## Where things live

```
web/css/tokens.css       the single source of truth: palette, scales, semantic aliases
web/css/base.css         reset, body, typography, tables, button primitive, forms, top-nav
web/css/components.css    legacy component layer: fireworks, cards, notifications, dropdowns, builder inputs
web/css/instrument.css    data-dense components (profile, heatmap, stat/streak cards, tabs, chips,
                          activity statistics, soundtrack, gear, exercises timeline) — being
                          converted from the old dark "instrument" skin to the light system
web/css/utilities.css     atomic helpers (spacing / alignment / sizing) — loaded last so they win
web/css/modal.css        the shared light modal (TRModal, .trm-*) — app-wide dialogs
web/css/workout.css      the workout summary view (.workout-view, .wv-*)
web/css/admin.css        admin page
```

Every page links, **in this order**: `tokens.css` → `base.css` → `components.css` →
`instrument.css` → `utilities.css`, then any page-specific sheet (`modal.css` /
`workout.css` / `admin.css`).
`base`/`components`/`instrument` are the sequential split of the old `main.css` — the order
above reproduces its cascade exactly. `tokens.css` is first so every stylesheet — and any
inline `style="…"` string emitted from JS — resolves its `var(--…)` values from it. There is
no shared HTML head partial: each page in `web/html/` links its own stylesheets.

## Design language

**Light and simple first — modern bones, calm skin.** The target is *not* to convert the
app to the dark, over-the-top gear/workout-modal look. It's the **middle ground**: keep the
discipline of the modern work — one token system, consistent spacing, shared components,
navy + a green accent — but on **mostly light, calm surfaces**, in **sentence case**, with
**subtle** depth (thin borders, soft shadows; no glows or decorative brackets).

**Light everywhere — no dark exception.** The whole app is going light. The deep-navy
"instrument" surfaces (translucent-white tiles on navy panels, `.workout-view`, the
exercises/gear/stats readouts, and the `.trm` modal) are **legacy being converted**, *not* a
look we keep for data-dense widgets. When you touch a dark instrument surface, convert it to the
light system — a **module panel** with **light inset tiles** and **display-font navy numerals**.
Data density is expressed with layout, tokens and the display font, **never** with a dark skin.
If a dense widget needs something the guide doesn't have yet, design a **light** component that
fits these rules and add it here. (Historically the guide reserved dark instrument panels "used
sparingly" — that framing is retired; the migration finishes them off.)

**Typography:** body font (`--font-body`) for everything by default, sentence case. The
condensed **display font (`--font-display`) is reserved for numerals / stat readouts** — big
metrics, scoreboards — not general buttons, labels, or headings.

**Blue theme; other colours only to signal.** The app runs a **blue** theme — the accent
(`--module-accent`, a friendly `--blue` #4a9fe0) carries identity (module left bars, progress
fills, navy headings), applied as a thin *label*, never a coat of paint. Reach for a
**different** hue only when it **communicates something specific**: green = success/complete,
amber = warning, red = error/destructive (e.g. the debt notice keeps a red left bar). A rainbow
of decorative per-module colours is clutter — don't.

**Inside a module, keep it flat.** Inner blocks share one convention — eggshell fill, hairline
edge, no ad-hoc heavy borders or divider lines under titles. Let the panel frame do the
separating; don't add a second border system inside it.

## Tokens

**Every colour in the app comes from a token here — never a hardcoded hex/rgb.** This is the
whole palette; if a value you need isn't here, it's a new decision: add the token, document it,
then use it.

### The colour palette

One blue-themed light system. Four groups:

| Group | Tokens | Role |
|---|---|---|
| **Neutrals** | `--white`, `--eggshell` (page/panel canvas), `--grey` (borders/dividers), `--darkblue`/`--black` (text) | surfaces & text — the calm base |
| **Blue (brand/theme)** | `--blue` `#4a9fe0` (accent), `--blue-050` `#e3eefa` (palest tint: tracks/insets), `--mediumblue` `#3a6ea5` (solid fills: nav, buttons), `--lightblue` `#7fa8cf` (secondary), `--darkblue` `#0d1b2a` (deep) | identity; carried by `--module-accent` |
| **Signals** | `--success` `#2fa866`, `--warning` `#eaa72c`, `--error` `#e5555b` | state only — tuned to sit with the blue |

> **One sky hue.** All theme blues sit on a single hue (~206°) so nav, buttons, avatar rings,
> progress fills and the accent read as **one family**. `--mediumblue`/`--lightblue` were
> previously a slate ~213° that clashed with the brighter `--blue` accent; they were warmed onto
> the accent's hue. Change one, keep them on-hue.
| **Exceptions** | `--strava` `#fc4c02` (external brand); fireworks celebration hues; the **streak-heat scale** `--flame-1/2/3` (amber→orange→red, for the 🔥 by streak length); the dark instrument sub-palette (navy shades, `--ink-*`) | intentional, documented; not for general UI |

**Rules:** blue is the theme; signal hues appear *only* to communicate state (success/warning/
error); external-brand colours only for that brand; decorative/instrument palettes stay inside
their component (fireworks, `.trm`/`.workout-view`). No decorative multi-colour elsewhere.

> **Migration status:** the light-UI palette is tokenised. A batch of legacy hardcoded hexes
> still lives in `components.css`/`instrument.css` (mostly the dark instrument shades and
> fireworks); these are being folded into named tokens file-by-file. Goal: zero hardcoded
> colours outside `tokens.css`.

### The three token tiers

Components should consume the **semantic** and **scale** tiers, not raw primitives.

### 1. Primitives (raw values — avoid referencing directly)

| Token | Value | Note |
|---|---|---|
| `--navy-900` | `#0b121c` | deepest panel base |
| `--navy-850` | `#0d1b2a` | == legacy `--darkblue` |
| `--navy-800` | `#101d2c` | solid panel fill behind nodes/icons |
| `--navy-750` | `#141f2e` | panel top / lighter surface |
| `--ink-050` | `#f3f6fb` | near-white text |
| `--ink-300` | `#9fb0c7` | muted labels |
| `--emerald` | `#00d27a` | accent |
| `--orange` | `#fc4c02` | Strava brand |
| `--amber` | `#f0b429` | audio layer |

Plus the **legacy palette** (`--red`, `--red-error(-border)`, `--yellow`,
`--yellow-light`, `--green`, `--green-success(-border)`, `--lightgreen`,
`--eggshell`, `--lightblue`, `--trans-lightblue`, `--mediumblue`, `--darkblue`,
`--black`, `--white`, `--grey`) — moved verbatim from `main.css`, unchanged, still
used by legacy components until they're migrated.

### 2. Semantic aliases (consume these)

| Token | Role |
|---|---|
| `--text` | primary text on dark |
| `--text-dim` | muted / secondary text |
| `--line` / `--line-soft` | hairline borders / dividers |
| `--surface` / `--surface-hi` | translucent raised fills (default / hover) |
| `--panel-top` / `--panel-bottom` / `--panel-solid` | solid navy panel surfaces |
| `--accent` / `--accent-soft` | positive/active accent + its tint (rings, glows) |
| `--focus-ring` | the **one** blue focus halo for every focusable control (form fields, `.btn`, modal) — a translucent `--blue`. Not the green `--accent`. |
| `--success` / `--warning` / `--error` | state signals (green / amber / red), tuned to the blue |
| `--danger` | destructive (points at `--error`) |
| `--strava` | Strava brand orange |
| `--audio` | soundtrack / listening layer amber |
| `--module-accent` | the single blue theme accent for module panels (left bar + progress). One knob recolours the whole set. |
| `--inset-bg` / `--inset-border` | the one convention for small boxes nested inside a module panel (number box, debt notices, joinable-season rows, progress track, push prompt): palest-sky fill + hairline grey edge. |

### 3. Scales

- **Type:** `--font-display` (Saira Semi Condensed → Roboto), `--font-body`
  (Hanken Grotesk → Roboto). Both webfonts are loaded on every page (Google Fonts,
  after the Roboto link). They don't render on the offline PWA page — self-hosting is a
  known follow-up. Use `--font-display` for numerals/headings/buttons, `--font-body` for prose.
- **Spacing:** `--space-0`…`--space-8` (`0`, `.25`, `.5`, `.75`, `1`, `1.5`, `2`,
  `2.5`, `3` rem).
- **Radius:** `--radius-xs` `.25rem`, `--radius-sm` `.5rem`, `--radius-md` `1rem`,
  `--radius-lg` `1.25rem`, `--radius-pill` `999px`, `--radius-circle` `50%`.
- **Elevation:** `--shadow-sm`, `--shadow-md`, `--shadow-lg`.

## Components

### Buttons — the full system

**One button system for the whole app.** Every clickable action — `<button>`, or an `<a>`/`div`
that behaves like a button — uses `.btn` plus modifiers. No page invents its own button. Calm,
light-first look: solid fill, **sentence case**, body font, `--radius-sm` corners (no uppercase,
no glow). `.btn` is fully self-defining, so it overrides the legacy global `button` element rule
wherever applied.

**Anatomy:** always `class="btn"` + at most one **colour** variant + at most one **size** +
optional **layout**. A leading `<img>` is auto-sized to 1rem.

```
<button class="btn btn--primary">Start new workout</button>
<button class="btn"><img src="/assets/plus.svg">Add gear</button>
<button class="btn btn--danger">Delete account</button>
<a class="btn btn--ghost" href="/gear">Manage gear</a>
<button class="btn btn--icon" aria-label="Increase"><img src="/assets/plus.svg"></button>
<div class="btn-group"> … buttons … </div>
```

**Colour (emphasis) — pick one:**

| Class | Fill | Use for |
|---|---|---|
| `.btn` | solid `--mediumblue`, white text/icon | the default / neutral action |
| `.btn--primary` | **same** solid `--mediumblue`, white text/icon | the **one** main action of a view (positive/go) |
| `.btn--danger` | `--danger` (soft `--error` red) | destructive (delete, disconnect, revoke) |
| `.btn--ghost` | transparent + `--mediumblue` border | low-emphasis / secondary |

Rule of emphasis: solid fill = important (default `.btn` and the one `.btn--primary`), **`--ghost`
for the quiet secondary ones**, `--danger` only for destructive. At most one `--primary` per view.

**One button blue.** Every solid button is `--mediumblue` with white text/icons — `.btn--primary`
is *visually identical* to `.btn`; it marks the main action semantically, and stands out by
placement (e.g. alone in the hero), not a unique colour. Why not a brighter blue for the CTA: a
fill light enough to look "bright" (the `--blue` accent) can't carry legible white text — white on
`--blue` is ~2.7:1 — so the bright accent is reserved for **rings and progress bars** (no text on
them) and never used as a button fill. **Green is signals-only**: complete/success (100%, ✅, 🔥
streaks), never a button.

**Size & layout (optional, combine with a colour):**

| Class | Effect |
|---|---|
| `.btn--sm` | compact |
| `.btn--block` | full-width |
| `.btn--icon` | square icon-only button (pair with a colour, or bare for neutral) |

**Groups:** wrap a row of buttons in `.btn-group` (flex, consistent gap) instead of ad-hoc
spacing.

**States** (automatic): `:hover` lifts + brightens, `:active` presses, `:focus-visible` shows an
`--accent-soft` ring, `:disabled` dims to 0.55. Don't restyle these per-button.

**Writing:** label the action, active voice, sentence case — "Save changes", not "Submit"; keep
the same verb through the flow (a "Publish" button → a "Published" toast).

**Migration map** — legacy button-likes → the system (translate on touch):

| Legacy | Replace with |
|---|---|
| `.regular-button` | `.btn` |
| `.danger-button` | `.btn btn--danger` |
| `.small-button-icon` / `.button-icon` (icon on `<img>`/`<a>`) | `<button class="btn btn--icon">` |
| `.btn_logo` (provider logo in a login button) | leading `<img>` inside the `.btn` |
| `.button-collection` | `.btn-group` |
| a bare clickable `<div>` acting as a button | a `.btn` (make it a real `<button>` where possible) |

**Genuinely separate** (own components, not buttons — leave): `.wheel-clear`/`.wheel-swatch`
(colour picker cells), `.we-add-*` (builder rows), `.trm-close` (modal ✕), `.user-stat-tab`
(tabs). The legacy global `button` rule still stands for anything not yet migrated.

### Cards

All cards are light — the base **`.card`** shell is a calm eggshell surface (see `base.css`); the
old `.card--light` modifier is **retired** (the whole app is light, so it lives in the base). Ordinary
content uses the **module panel**; data-dense metric readouts use **light metric tiles** (inset fill,
hairline, display-font navy numerals — see `.user-stat-card`).

**Module panel**: the shared style for content blocks that sit on the `.card` — `.season`,
`.current-week`, `.debt-module`, `.prize-module`, `.leaderboard`, `.activities`, `.week_days`,
`.ai-message-card`, … They render as **one identical panel**: white surface, hairline grey border, a
single blue `--module-accent` **left bar**, soft shadow, `--radius-md`, `overflow: hidden`,
`box-sizing: border-box` (so the padding stays inside the width — some modules default to content-box
and would otherwise render wider), and **one consistent inner padding (`--space-4`)** — inner blocks
(headers, progress bars) sit **inset, never flush** to the edge. Driven by **one shared rule** (scoped
under `.card` in `components.css`) — keep every module on that rule; don't give a module its own
colour, border, or padding.

**Dividers:** one treatment inside a light module — a single hairline in `--grey`. `<hr>` and
every internal border (activity rows, previous-weeks) use it; never the old faint-eggshell `<hr>`
or the bluish `--trans-lightblue` borders.

| Class | Use | Status |
|---|---|---|
| `.stat-card` | metric tile (statistics); `data-family="movement\|effort\|strength\|audio\|time\|neutral"` sets a per-family accent stripe (`--stat-accent`, tokenised). Sub-elements `.stat-card-label` / `.stat-card-value` / `.stat-card-unit`. | **light** |
| `.user-stat-card`, `.user-streak-card` | profile stat / streak tiles — light **metric tiles**: `--inset-bg` fill + `--inset-border` hairline, value in `--font-display` navy, muted `--lightblue` label. | light |
| `.ai-message-card` | the AI greeting widget | light |
| `.panel-card` | centred white content-card wrapper (extracted Phase 4) | transitional |
| `.card` / `.card-header` / `.card-body` | the shared page-shell wrapper (all pages) — **light** eggshell surface, navy text (`base.css`) | **light (base)** |

**Inset blocks** (inside a module panel): every small box nested in a panel — the number box,
progress-bar track, debt notices, joinable/countdown-season rows, the push prompt — shares **one**
convention: `--inset-bg` (palest sky) fill, `--inset-border` hairline, `--radius-sm`. Don't give a
nested box its own fill or border. The only exceptions are *signals* — the progress **fill** uses
`--module-accent`, and the debt notice keeps a red left bar (`--error`).

### Avatars

One circle pattern for every profile photo (`.current-week-user-photo`, `.activity-user-photo`,
`.leaderboard-week-member-image`, `.user-active-profile-photo`). The **wrapper** is the circle: it
carries the `--lightblue` ring, is `box-sizing: border-box`, and **clips its content
(`overflow: hidden`)** so the image can't poke past the ring. The **image** fills it
(`width/height: 100%`, `object-fit: cover`, `display: block`). Never put the ring on the image or
omit the clip.

### Day row

The front-page week calendar **stacks each day vertically** (`.card .form-group`): the day
label, then the number box + its `−`/`+`/`✎` `.btn--icon` buttons on one line, then the notes field
full-width beneath. One consistent layout whether or not the action buttons show (they only appear
up to today), so notes never get squeezed into a narrow, wrapping column. (Replaced a fixed 6rem
column, then a 2-col grid whose notes column varied per row.)

### Hero — the weekly-goal ring

The front page opens on its core loop: a **conic-gradient progress ring** (`.hero-ring`) showing
`workouts / goal` for the week in the **display font**, over a palest-sky track. A JS-set `--pct`
(0–100) fills the blue arc — animated via `@property --pct` where supported, snapping otherwise —
and it flips to `--success` green at 100% with a brief pulse. The ring lives in a clean centred
white **hero panel** (`.card .hero`, capped `max-width` and centred) with *no* left accent
bar (it's the thesis, not a module). Hidden until a season populates it. No app-name title or
subtitle sits in the hero — the nav already names the app, and the ring + CTA carry it. This is the
page's signature element — keep the rest of the page quiet so it stays the one memorable thing.
The **debt-spin** state ("you must spin the wheel") reuses `.hero` so it reads as a panel, not
bare text.

### Auth panel

`.auth-panel` — the centred white card that holds the **login / register / password-reset /
change-password / verify-code / OAuth-consent** form on the light auth pages. Same calm centred-panel treatment as the hero
(capped `max-width`, centred, soft shadow, **no left accent bar**). Keep it **flat inside** per the
module rule: no internal `<hr>` dividers or empty label spacers — the panel frame separates. One
primary action per form: a full-width `.btn.btn--primary.btn--block`. The alternate action below
(e.g. "I forgot my password") is a quiet `.u-fs-sm` link in `.auth-alt`. A consent checkbox + label
goes in `.auth-consent` (checkbox left, label wraps, aligned to the label's first line). The OAuth
authorize screen lists the requested scopes in a left-aligned inset `.auth-scopes`, with **Approve**
as the `.btn--primary` and **Deny** as a `.btn--ghost` (both `.btn--block`).

### Settings accordion

The `/account` settings live in a collapsible accordion inside **one white module panel**
(`.account-section-wrapper`). Each `.account-section` is a row separated from the next by the
single **`--grey` hairline** (last row none); its `.account-section-tab` is the clickable header
(title + a chevron `<img>` that swaps `chevron-right`↔`chevron-down` on toggle — rendered **dark**
on the light surface, `filter: none`, not `color-invert`). Bodies (`…-wrapper`) toggle `display`.
Rows of buttons use `.btn-group`; each section's own submit is a `.btn` (its primary action may be
`.btn--primary`); destructive actions (`Leave season`, `Delete account`) are `.btn--danger`.

### Chips & tags

Container `.tag-list` (read-only) or `.tag-selector` (editable); items are `.tag-chip`.
In a `.tag-selector`, chips render dimmed until `.tag-chip-selected` (solid green, bold).
`.tag-chip-readonly` disables the pointer. This is the target chip; use it for any tag UI.

### Segmented tabs

`.user-stat-tab` in a `.user-stat-tabs` bar — a segmented control for switching a panel (e.g. the
profile's month / year / all-time stats). Each segment is a light
`--inset-bg` pill; the active one (`.user-stat-tab-active`) is solid `--mediumblue`/white — the same
"active = solid mediumblue" rule as buttons. Use this for in-panel period/section switches; it is
*not* the top nav and not `.btn`.

### Meta tags

`.meta-tag` — a small hairline label (an achievement's category, a "Stackable" flag). Outline in
`currentColor`, `--radius-sm`, tiny text. Compose margins with `.u-*` utilities. Use it for a short
static descriptor on a tile; for interactive tag selection use `.tag-chip` instead.

### Skeletons

Loading placeholders are `.skeleton-block` (the shimmer; sweeps dark on the light card). **Size
them with classes, never inline** — the profile page uses `.skel-label`, `.skel-streak`,
`.skel-tile`, `.skel-bar` (+ `.skel-row` to lay a row out, `.skel-mt` for section spacing). Add a
new named size class here rather than an inline `width/height` when a new skeleton shape appears.

### Form controls

**One field for the whole app.** Text-like inputs (`text`/`email`/`date`/`number`/`file`/
`search`/…), `<select>` and `<textarea>` share a single light skin defined on the
zero-specificity `:where(…)` baseline in `base.css`. Because it carries **no specificity**, it
styles *any* plain field with **no class needed** — and a component overrides it with a single
class. Buttons and toggle inputs (`submit`/`button`/`reset`/`checkbox`/`radio`/`range`/`color`)
are excluded from the skin (they carry their own look). The modal's `.trm-*` fields resolve to
the **same values** (via the token remap on the `.trm` root), so a field is **identical on a page
and in a dialog** — don't build a third variant.

| State | Treatment |
|---|---|
| **Rest** | `--white` fill · `1px solid var(--grey)` hairline · `--radius-sm` · `--font-body` · `1rem` (16px — stops iOS zoom-on-focus) · `--darkblue` text · `box-sizing: border-box` |
| **Placeholder** | `--lightblue` |
| **Hover** | border → `--lightblue` |
| **Focus** | border → `--mediumblue` · fill → `--inset-bg` · `0 0 0 3px var(--focus-ring)` halo (the **one** blue focus ring — see below) |
| **Disabled / `readonly`** | `--inset-bg` fill · `--lightblue` text · `not-allowed` cursor (reads as read-only, e.g. Strava-synced gear) |

- **Textarea** adds `resize: vertical`, `min-height: 4.5rem`, `line-height: 1.45`. The front-page
  calendar notes (`.day-note-area`) and the admin season description are the **same** textarea —
  both inherit this skin; only `.day-note-area`'s day-row *layout* is scoped in `components.css`.
- **Checkbox / radio** stay native, tinted with `accent-color: var(--mediumblue)` so ticks read in
  the theme blue everywhere (accessible, no custom markup). The `.day-check` day toggles and the
  `.auth-consent` row are layout wrappers around native checkboxes, not a separate control.
- **In a modal**, prefer the explicit `.trm-field` / `.trm-label` / `.trm-input` / `.trm-select`
  wrappers for structure; the underlying control is this same field.
- **The one blue focus ring — `--focus-ring`.** Every focusable control — form fields **and**
  `.btn` **and** the modal — uses `0 0 0 (3px|.2rem) var(--focus-ring)` (a translucent `--blue`).
  **Green is signals-only and is never a focus ring** (the global `--accent`/`--accent-soft` are
  green — do not use them for focus). Change the ring in one place: the `--focus-ring` token.

### Fields & labels

**One label-over-control group for page forms: `.field`** — the non-modal sibling of the modal's
`.trm-field`. Wrap every labelled control in a `.field`; don't fall back to the legacy
`<label>…</label><br><input>` (which the global `label { text-align:center }` rule centred and left
ragged). One field per control keeps label, control and hint together with consistent rhythm.

```
<div class="field">
  <label for="season-name" class="field-label">Name</label>
  <input type="text" id="season-name" …>
  <span class="field-hint">Shown on the season card.</span>   <!-- optional -->
</div>

<div class="field-row">            <!-- two fields on one line; wraps when narrow -->
  <div class="field"> … start date … </div>
  <div class="field"> … end date … </div>
</div>

<div class="field-check">          <!-- checkbox + label on one line -->
  <input type="checkbox" id="join_anytime" …>
  <label for="join_anytime">Let users join the season at any point.</label>
</div>
```

| Class | Role |
|---|---|
| `.field` | the group: stacks `.field-label` → control (→ `.field-hint`), left-aligned, one `--space-4` bottom margin. Zeroes the control's own baseline margin so the group gap sets the rhythm. |
| `.field-label` | the label — **sentence-case, body font**, `--darkblue`, semibold. *Not* the modal's uppercase display eyebrow (`--font-display` is numerals-only per Design language). |
| `.field-hint` | small muted (`--lightblue`) helper/secondary text — under a control, or inline in a label (e.g. "User (optional)"). |
| `.field-req` | wrap a required-marker `*` (renders `--error`). |
| `.field-row` | lay two `.field`s side by side; wraps to stacked once the row is too narrow. |
| `.field-check` | checkbox/radio + label on one line, aligned to the label's first line. Generalises `.auth-consent`. |

- **Every control still uses the one field skin** (see Form controls) — `.field` only handles the
  *label + layout*, not the control's look.
- **In a modal, use the `.trm-*` siblings** (`.trm-field` / `.trm-label` / `.trm-row`), which carry
  the modal's denser eyebrow label — not `.field`.
- **Labels are written in sentence case, no trailing colon** ("Birth date", not "Birth date:").
- **Placeholder-only inputs are fine** where a visible label would be noise (the `.auth-panel` login /
  register fields lean on placeholders by design) — don't force a `.field-label` on those.

### Modal — `.trm` (TRModal)

The one shared modal (**light**: white panel, thin blue top keyline, neutral dimmed backdrop),
driven by `web/js/modal.js` (`TRModal`); markup is generated, not hand-written. Structure: `.trm`
(overlay/root) → `.trm-overlay` (backdrop) → `.trm-panel` → `.trm-head` (`.trm-title`,
`.trm-close`) + `.trm-body`. Inside the body use `.trm-section-label`, `.trm-row` (side-by-side),
`.trm-divider`, `.trm-field`/`.trm-label`/`.trm-input`/`.trm-select`, `.trm-badge`, `.trm-eyebrow`.
Buttons inside `.trm-body` default to the primary look (mediumblue/white, sentence case); add an
explicit `.btn` class to opt into the button system. The `.trm` root **overrides the semantic
tokens** to light values, so the whole panel relights from one place. **Never hand-render a
modal** — extend TRModal.

**This is the preferred surface for forms, config, and per-action feedback** — reach for it before
the top alert bar (see Alerts for why), especially in complex multi-section spaces like `/account`.
A modal keeps the task and its result in view without scrolling the page.

### Alerts

`.alert-success` / `.alert-info` / `.alert-danger` (in `base.css`) — legacy light-theme
status banners, still on the legacy semantic palette. Fine to use; will be re-skinned toward
the instrument look in the page sweep.

**Prefer a modal over the top alert bar.** These banners render at the **top of the page**, so on
a long/scrolled page the user has to **scroll back up** to see the feedback — the alert is easy to
miss and the required scroll jump is jarring. So lean on them **less**, and reach for the `.trm`
modal **more**:

- **Feedback / confirmation** (success, error, "are you sure?") on a long page → surface it in a
  modal (or a modal-anchored message), not a banner the user has scrolled away from. A modal appears
  in view, wherever the user is.
- **Complex, multi-section spaces** — e.g. `/account` settings with several accordion sub-sections —
  should **do their forms and config actions in a modal** rather than editing inline and reporting the
  result via a top banner. Opening a focused modal keeps the action and its feedback together, in
  view, without moving the page. Treat the accordion as navigation *into* a modal-driven task, not a
  page full of inline forms each reporting up to the shared top banner.
- Keep the alert bar for genuinely **page-level, persistent** notices (e.g. a global state banner) —
  not per-action feedback.

### Utilities — `.u-*`

Atomic helpers in `utilities.css`, loaded last so a single utility class wins over the
component layers. Introduced (Phase 4) to move presentational values out of inline
`style="…"` strings. **Values are preserved verbatim from the migrated inline styles
(em, not tokenised)** so the migration is visually identical; tokenising to `--space-*`
is a later pass. Compose them: `class="u-w-full u-text-center"`.

| Group | Classes |
|---|---|
| margin | `.u-m-0` `.u-m-1` (.25em) `.u-m-2` (.5em) `.u-my-1` (1em 0) |
| margin-top | `.u-mt-sm` (.5em) `.u-mt-1` `.u-mt-2` `.u-mt-3` (1/2/3em) |
| margin-bottom | `.u-mb-1` (1em) `.u-mb-2` (.5em) |
| padding | `.u-p-1` (.4em) |
| text / display | `.u-text-center` `.u-dim` (opacity .7) `.u-pointer` `.u-noselect` `.u-fs-sm` (.75em) |
| sizing | `.u-w-full` `.u-fill` (w+h 100%) `.u-w-{5,8,10,12,16,18,20}` (em) |

Component classes extracted from repeated inline styles (in `components.css`):
`.integration-btn` (12em connect/disconnect buttons), `.panel-card` (centred white card
wrapper), `.panel-wide` (1000px panel).

**Scope note:** visibility toggling (`display:none` + `.style.display` in JS) is *state*,
not theme — left inline on purpose. Dynamic (`${…}`) values stay inline too.

## Adding new UI — the rules

1. **Never hardcode a colour/size.** Reference a semantic token (`var(--text)`,
   `var(--surface)`, `var(--accent)`, `var(--radius-sm)`, `var(--space-4)`). If the value
   you need has no token, prefer the nearest one; add a token only if it's genuinely new.
2. **Reuse a component before inventing one.** Buttons → `.btn` (+ modifier). Metric tiles
   → `.stat-card`. Tags → `.tag-chip`. Modals → extend `TRModal`. Don't hand-roll parallels.
3. **One-off spacing/alignment → a `.u-*` utility**, composed on the element. Reach for a
   utility over an inline `style="…"`. If you need a value the utilities don't have, add the
   utility (keep the set small and predictable) rather than an inline style.
4. **A recurring widget → a semantic component class** (in `components.css` or
   `instrument.css`), not a pile of utilities. `.integration-btn` / `.panel-card` are examples.
5. **Put the rule in the right layer** (see *Where things live*): global element/base →
   `base.css`; shared component → `components.css`; the new dark design system →
   `instrument.css`; atomic helper → `utilities.css`. Tokens only in `tokens.css`.
6. **Inline `style="…"` is for genuinely dynamic values only** (computed `${…}`, JS-driven
   visibility). Static presentational styling belongs in a class.
7. **When you touch legacy UI, migrate it** toward the instrument look and these
   conventions — don't extend the legacy pattern.

## Decisions log

- **Mobile scaling fixes (iPhone-width sweep).** Three narrow-viewport overflow bugs, each a
  flex sizing issue where a child couldn't shrink below its content: (1) `/exercises` — the
  `.feed-panel select.feed-input` type filter clipped past the panel's right border on `≤480px`
  because a flex item's default `min-width:auto` held it at its widest option's width; added
  `min-width: 0` so it honours the `flex: 1 1 100%` mobile basis and the native select truncates.
  (2) `/statistics` weight modal — each `.weight-value` row was `nowrap` with three fixed
  `u-w-8`/`u-w-5`/`u-w-8` columns (21em) wider than the modal body, and since `.trm-body` is
  `overflow-y:auto` the x-axis computed to `auto` too → a horizontal scrollbar; replaced the fixed
  columns with flexible ones (`.weight-date` `flex:1 1 auto;min-width:0` truncates, `.weight-amount`
  / `.weight-actions` stay content-size, tokened gap). Retired the now-dead `.u-flex-end` utility.
  (3) `/account` — `.notification-options` (the wrapper for every Strava/Hevy/Plex/… button row) had
  no `gap`, so the 12em `.integration-btn`s butted together; added `gap: var(--space-2)`. General
  rule: any flex row of fixed/near-fixed children that must fit a narrow container needs a shrinkable
  child (`min-width: 0`) and a `gap`.
- **Exercise-builder set inputs — already migrated; removed the dead legacy CSS.** Investigating the
  reps/weight/time/distance/duration inputs (the intended target of a "migrate to the unified field"
  pass) showed they were **already** on the light workout editor's `.we-set-input`/`.we-input`
  (`workout.css`, unified field + `--focus-ring`) from the `/exercise` sweep. The old
  `.operation-set-*-input` / `.exercise-time-input` / `.operation-set-input*` / `.operation-set-title`
  / `.operation-type-input-lifting` rules in `components.css` (an `!important` block with `--eggshell`
  fills, `1rem`/`0.1rem`/`0.2rem` conflicting borders, duplicate `.exercise-time-input` definitions)
  were **dead**: those class names are only used as element `id`s (read via `getElementById`), and the
  single `class="exercise-time-input"` sat on a `type="hidden"` input. Deleted the block and the dead
  class. No visual change.
- **Field/label system (`.field`) for page forms.** Non-modal forms used a legacy
  `<label class="clickable">…</label><br><input>` pattern that the global `label { text-align:center }`
  rule centred and left ragged (and some inputs had no label at all — placeholder-only, or an empty
  `<label>`). Added a `.field` group (label-over-control) with `.field-label` (sentence-case **body**
  font — deliberately not the modal's uppercase display eyebrow, per the numerals-only display-font
  rule), `.field-hint`, `.field-req`, `.field-row` (side-by-side, wraps), and `.field-check` (checkbox
  row, generalising `.auth-consent`). Migrated **admin** (all field types — the canonical form; also
  gave the previously label-less season name/description real labels), **news**, **registergoal**
  (compete checkbox), **account** (settings text fields + inline share/password checkboxes), and the
  **weight-log modal** (to the `.trm-*` siblings, since it's modal content). **Deliberately left** (not
  a label-over-field, a bespoke component layout): account's notification/strava/hevy **toggle cards**
  (`.notification-option` 10rem centred cards, `.strava-option`) — a redesign for a dedicated account
  pass, not this system. See Fields & labels.
- **Form controls unified into one field (+ one blue focus ring).** There were ~four disagreeing
  input systems: the `:where()` baseline (chunky `0.2rem` border, hardcoded `Roboto`, **no focus
  state**, and it didn't even cover `<textarea>`), the bespoke `.day-note-area`, the admin season
  description (literally `class=""` → raw browser default), and the good modal `.trm-*` fields.
  Promoted the modal's quality to the app-wide baseline: hairline `--grey` border, `--radius-sm`,
  `--font-body`, `1rem`, and real `:hover`/`:focus`/`::placeholder`/`:disabled` states — now
  including `<select>` and `<textarea>`, so the two example textareas (calendar notes + season
  description) are one shared control. Checkboxes/radios stay native with `accent-color:
  var(--mediumblue)`. Folded `.day-note-area`/`.form-control(-small)` onto the baseline (dropped the
  `!important` legacy overrides + the hardcoded `#e9ecef` disabled fill); cleaned the admin
  textarea's invalid `type="text"`/`value=""`. Added **`--focus-ring`** (translucent `--blue`) as
  the single focus halo for fields **and** `.btn` **and** the modal — fixing that `.btn:focus-visible`
  and the modal ring were previously **green** (`--accent-soft`), off the blue theme. See Form controls.
- **Prefer modals over the top alert bar for feedback + complex forms.** The `.alert-*` banners
  render at the top of the page, so on a long/scrolled page the user must scroll up to see them —
  noisy and easy to miss. New per-action feedback (success/error/confirm) should surface in a modal
  in view instead, and complex multi-section spaces (e.g. `/account` settings) should run their
  forms/config actions in a `.trm` modal rather than inline forms reporting up to the shared top
  banner. The alert bar is reserved for page-level persistent notices. (See Alerts + Modal.)
- **Base `.card` flipped light; `.card--light` modifier retired (endgame).** With every page swept,
  the light shell + `.card-header`/`.card-body` moved into the base `.card` rule (`base.css`, was
  mediumblue/white), and all `.card--light .X` scoped selectors became `.card .X` (in `components.css`;
  the compound `.card.card--light` override was deleted). Removed `card--light` from all 19 HTML shells
  (now `class="card"`). Net effect: the app is light from the base with no opt-in modifier — one fewer
  class to remember. Historical decisions-log entries below still say "`.card--light`" (accurate for
  when they were written).
- **`/countdown` removed (dead page).** The full-page countdown was never hit in production — the
  front-page inline "seasons you are waiting for" list (`getCountdownSeasons` etc.) is the live
  feature. Deleted `countdown.html`/`countdown.js` + the `countdownRedirect()` flow; the front page
  now renders normally for an upcoming primary season instead of redirecting. **With this, the
  light sweep is complete — every page is light.**
- **`/statistics` converted to light (last content page).** `.stat-panel` / `.soundtrack-panel` →
  white module panels (soundtrack keeps an amber `--audio` left bar for the audio layer); `.stat-card`
  → light inset tiles with **tokenised family accents** (movement `--blue`, effort `--error`, strength
  `--warning`, audio `--audio`, time `--lightblue`, neutral `--grey`), display-font navy value;
  section titles + soundtrack text → navy/`--lightblue`; amber album-art/mark tints kept. The heatmap
  is a Leaflet map (unchanged) and the Chart.js charts use library defaults (dark-grey text, already
  fine on light — no per-chart config here, unlike the exercise HR chart). Added a `.u-flex-end`
  utility.
- **Gear form de-duplicated into one shared component.** The `/gear` page and the exercise-page
  "Manage gear" modal were two full copies (two renderers, `createGear`/`updateGearField`/`deleteGear`
  duplicated in both files, `.gear-*` vs `.trm-gear-*` classes, two CSS blocks) — which is how the
  disabled-field styling drifted. Consolidated: new **`web/js/gear-shared.js`** (`gearList`,
  `gearItemHTML`, `gearListHTML`, `gearAddFormHTML`, and one copy of the CRUD) loaded on both pages;
  each page sets an `onGearChanged` hook to re-render its own view. One **`.gear-*`** class set +
  one CSS block (base rules made light, so page and modal match; the `.card--light .gear-*` overrides
  and the dark `.trm-gear-*`/`.gear-btn` were deleted). Delete now confirms in both places.
- **`.trm` modal converted to light (app-wide).** Same token-override technique on the `.trm` root
  relit the whole modal; hardcoded bits fixed directly: the green-glow/noise backdrop → a **calm
  neutral scrim** (`rgba(13,27,42,.45)` + light blur); panel → white with a soft shadow and a thin
  blue top keyline (dropped the accent **bloom/glow** and the **corner register brackets** — both
  off-style now, `.trm-corner { display:none }`); inputs → white/grey with blue focus; the default
  modal button → the `.btn` look (mediumblue/white, **sentence case, body font**, no glow); gear-del
  un-inverted; Strava badge → orange outline. This turns **every page's dialogs** light (weight log,
  confirms, add-action, manage-gear). The exercise gear modal uses `.trm-gear-*`, so it came along
  for free — no `.gear-*` fold-back was needed (that class is `/gear`-only).
- **`/exercise` converted to light (builder + summary; modal is the next pass).** The whole
  `workout.css` (`.workout-view` summary + `.we-*` editor) went dark→light. **Technique worth
  reusing:** the view consumes semantic tokens (`--text`, `--surface`, `--line`, `--accent`, …), so
  overriding those tokens **on the `.workout-view` root** relit the entire subtree in one block;
  only rules with hardcoded values (navy gradient + noise `::before`, `rgba()` fills, `filter:
  invert`, `#ffb020`) were fixed directly. Session panels → module panels; activity/exercise tiles →
  inset with type-accent left bars (cardio blue, strength `--warning`, time `--strava`); editor
  inputs → white/grey with blue focus; icons un-inverted; the salmon "combine" and legacy Restore
  buttons → `.btn`; `.addExerciseWrapper` → dashed inset add-tile. Amber soundtrack node-glows kept
  (audio layer). The activity-title action icon dropped `color-invert` (was white), and the **HR
  Chart.js chart** now reads its axis-tick / gridline colours from the tokens at runtime
  (`getComputedStyle(...).getPropertyValue('--lightblue' / '--grey')`) instead of hardcoded white —
  reuse that pattern for the statistics charts/heatmap. **Deferred to the modal pass:** the `.trm` modal + the gear modal (`.trm-gear-*`)
  it opens, and two `.trm-*` inline styles in `exercise.js` — so the exercise page's dialogs are
  still dark for now.
- **`/gear` converted to light.** The `.gear-*` classes are **shared with the exercise-page gear
  modal** (still dark), so the light versions are **scoped under `.card--light`** (dark base kept for
  the modal until that page is swept — fold back then). Panel → light module panel; gear items →
  inset tiles; inputs/toggles/glyph/distance relit to tokens; trash icon un-inverted. **Dropped the
  green "gear accent"** — active gear now uses the blue `--module-accent` left bar (green is
  signals-only), retired stays grey. "Add gear" moved off the bespoke green `.gear-btn` → `.btn--primary`.
- **Dark-instrument look retired; `/exercises` converted to light.** Recalibrated the direction:
  the deep-navy instrument skin is **legacy being converted**, not a data-dense exception (Design
  language + Cards + Known-gaps updated to match). Applied it to the `/exercises` timeline — the
  whole `.feed-*` block went from `--darkblue`/translucent-white to a **light module panel** (white
  + hairline + blue left bar), **inset activity rows**, navy text, `--lightblue` muted labels,
  display-font only for numerals (rank, session time), tokenised fonts/colours (no hardcoded hex or
  `'Saira'`/`'Hanken'` strings). Dropped `color-invert` on the action logo (was invisible on light)
  and moved `.button-collection` → `.btn-group`.
- **`/wheel` swept light.** Turned out light-friendly already (white winner card, text info, the
  spinner is a JS-drawn `<canvas>`), so it was a light touch: the two canvas inline styles →
  `.wheel-canvas-wrap`, and the Reset/Replay `<a>` actions → `.btn.btn--ghost` (Spin was already
  `.btn--primary`). The bespoke `.wheel-*` swatch/clear picker controls (on `/account`) remain
  intentionally separate. Only visibility `display:none` inline remains.
- **`/news` swept light.** Posts (`.news-post`) became module panels (white + hairline + blue left
  bar + shadow); the admin "Create post" submit → `.btn--primary.btn--block`, the delete trash icon
  → `.btn.btn--icon` (off `.btn_logo` + inline size). Stripped the per-post duplicate ids
  (`news-title/body/date/delete` were repeated on every post; the container `#news-title` that JS
  reads is kept). Only visibility `display:none` inline remains.
- **`/achievements` swept light.** Reused the `/users` achievement patterns wholesale: `.meta-tag`
  badges, the `--cat-color` image border, and the `.card--light .achievement-*` inset relighting.
  Added `.achievement-img-logo` (padded square icon, was inline) and relit the progress bar
  (`.progress-bar-wrapper`/`.progress-bar` → inset track + blue fill). **Fixed a latent bug from the
  `/users` sweep:** the achievements panel rule used a class selector `.achievements-module`, but the
  element is `id="achievements-module"` — switched to `#achievements-module`, so the panel now applies
  on both pages.
- **`/registergoal` swept light.** The goal form is a `.season` module panel (inherits the panel
  rule) with a scoped `max-height` lift (was an inline `!important`); "Join season" → `.btn--primary
  .btn--block`, the goal counter `−`/`+` and debt-notice icons → `.btn.btn--icon` (off
  `.small-button-icon`), and the debt-spin state reuses `.hero` like the front page. Only
  visibility `display:none` inline remains.
- **`/admin` swept light.** Its cards (`.server-info`, `.invites`, `.debt-module`, `.prize-module`,
  `.add-season-module`, `.correlate-module`) were **already in the module-panel rule**, so they
  panelled automatically — the work was the buttons: all 5 form/action buttons moved off the legacy
  bare `<button>` (with `<p2>` + `.btn_logo` + `color-invert` icons) onto `.btn` (form submits
  `.btn--block`), and the delete-invite `.icon-img` → `.btn.btn--icon`. Aligned `admin.css`'s
  `.info-section` to the inset tokens (`--inset-bg`/`--inset-border`/`--radius-sm`). Zero inline
  styles to begin with; no new pattern.
- **`/seasons` swept light.** Each season is now a **module panel** (white + hairline + blue left
  bar + soft shadow, `overflow: hidden`) instead of the old thick-lightblue-border card; the
  expandable per-season leaderboard reuses the front-page `.leaderboard-*` with the grey hairline for
  internal dividers. The "Expand" toggle moved off the legacy bare `<button>` (with `<p2>` +
  duplicate `id="goal_amount_button"` + inline styles) onto `.btn.btn--sm` with a `color-invert`
  chevron. Also removed an invalid `cursor:hover` inline and fixed a member tooltip that showed the
  UUID twice. No new pattern — reuse of module-panel + leaderboard.
- **Small auth pages swept: `/verify`, `/authorize`, `/offline`.** `verify` (code entry) and
  `authorize` (OAuth consent) reuse `.auth-panel` — flat inside, one `.btn--primary.btn--block`
  submit; `authorize` adds `.auth-scopes` (inset scope list) + a `.btn--ghost` Deny. `offline`'s
  Reload button moved off the legacy bare `<button>` onto `.btn--primary` (dropped the `<p2>` tag +
  inline style). `/oauth` is a transient "Authorizing…" redirect (nav hidden) — nothing to sweep.
- **`/account` swept light.** The collapsible settings accordion now sits in one white module
  panel (`.account-section-wrapper`), section rows separated by the grey hairline, chevrons dark
  (`filter: none`). Reused: the avatar + `.user-name`, `.btn-group` (from the retired
  `.button-collection`), `.integration-btn`, `.btn--primary`/`--danger`. Moved every static inline
  style to classes/utilities (the notification button row → `.btn-group`; file-input height, form
  margin, label spacing, plex hint, PAT spacings → utilities/scoped CSS); stripped the duplicate
  `id="form-input-icon"`. Only visibility (`display:none`) + the dynamic `.wheel-swatch` colour stay
  inline. New settings-accordion pattern documented.
- **`/register` swept light.** Mirrors `/login`: form in a centred `.auth-panel`, flat inside
  (removed `<hr>`s + empty `#form-input-icon` labels), submit → `.btn.btn--primary.btn--block`,
  consent checkbox+label moved into a left-aligned `.auth-consent` row. Dropped the dead empty
  `#news_feed` module.
- **`/login` swept light.** Already had `.card--light` + `.btn`; finished it: the form now sits in a
  centred **`.auth-panel`** (hero-style centred card, no left bar), went **flat inside** (removed the
  internal `<hr>` dividers and the empty duplicate-id `#form-input-icon` label spacers), and each
  submit became a full-width `.btn.btn--primary.btn--block`. New `.auth-panel` pattern documented.
- **`/users/:id` swept light (first page after the front page).** Shell → `.card--light`; the
  profile+stats row and achievements each became a **module panel**; the metric/streak tiles were
  relit as light **inset tiles** with display-font navy numerals; the activity switcher became the
  documented **segmented tabs**; category/"Stackable" labels → **`.meta-tag`**; internal dividers →
  the grey hairline. All overrides are **scoped under `.card--light`** so `/account` and
  `/achievements` (which share `.user-stat-card` / `.achievement-*`) stay on the dark card until
  their own sweep. Most inline styles moved to CSS: **skeleton sizes** → `.skel-*` classes, **flame
  colours** → `.flame-*` (new `--flame-1/2/3` streak-heat tokens), brand links → `.user-link`; only
  visibility (`display:none`) and the data-driven category colour (`--cat-color`) stay inline.
- **One sky-hue blue ramp (front-page sweep).** The theme blues were two families — a slate
  `--mediumblue`/`--lightblue` (~213°, low saturation) against the brighter `--blue` accent
  (~206°) — so the sky-blue progress bars and left-accent bars clashed with the slate nav and
  buttons. Warmed the slates onto the accent's hue: `--mediumblue` `#415a77`→`#3a6ea5`,
  `--lightblue` `#778da9`→`#7fa8cf` (and `--trans-lightblue` to match); added `--blue-050`
  `#e3eefa` (palest tint) for progress tracks / inset fills, and `--inset-bg`/`--inset-border`
  for the single nested-box convention. App-wide token change (nav/buttons/rings warm slightly
  everywhere) — intended coherence, verified on the front page first.
- **trm/wv unification (Phase 1).** The instrument-panel language previously
  existed twice — `--trm-*` (in `modal.css`) and `--wv-*` (in `workout.css`),
  near-identical. Both prefixes were removed and their usages repointed at the
  semantic aliases above. Canonical values for the near-duplicates: `--text`
  = `#f3f6fb`, `--text-dim` = `#9fb0c7`, `--surface-hi` = `rgba(255,255,255,.08)`
  (the workout/wheelview values — the most recent surface; the differences from
  the modal values were sub-perceptual).
- **Button system (Phase 2).** BEM `.btn` + modifiers; migrated *every* call site off the
  legacy `regular-button`/`danger-button`/`trm-btn`/`btn-primary` and deleted those selectors,
  rather than aliasing. `.btn` is self-defining so it overrides the still-standing global
  `button` rule (kept because bespoke buttons depend on it).
- **Sequential CSS split (Phase 3).** `main.css` was cut on section boundaries into
  `base`/`components`/`instrument` so concatenation reproduced it byte-for-byte — a
  provably zero-cascade-change refactor. Per-page extraction was rejected: `.stat-card`
  (statistics+user) and `.soundtrack` (exercise+statistics) are shared.
- **Hybrid utilities, verbatim values (Phase 4).** Utilities carry the migrated inline
  values *verbatim* (em, un-tokenised) to keep the migration visually identical. Visibility
  (`display:none` + `.style.display`) was classified as *state* and left inline.
- **One button blue (front-page refinement 2).** Walked back the bright-accent primary: a fill
  bright enough to read as the "go" button can't carry legible white text (white on `--blue` is
  ~2.7:1), and the resulting navy-text-on-bright button clashed with the white-on-`--mediumblue`
  neutral buttons. Resolution: **all solid buttons are `--mediumblue`/white**; `.btn--primary` is
  now visually identical to `.btn` (emphasis via placement + `--ghost` for secondary), and the
  bright accent is reserved for rings/progress only. Same pass: trimmed the redundant hero
  title/subtitle; fixed module width (added `box-sizing: border-box` — `.leaderboard`/`.activities`
  were content-box, so the panel padding pushed them to 22rem); unified all light-module dividers
  to one `--grey` hairline (were eggshell `<hr>` + bluish `--trans-lightblue` borders); swept the
  debt-spin state onto `.hero`.
- **Primary → blue, avatars/day-row/insets (front-page sweep).** `.btn--primary` moved off green
  onto the bright `--blue` accent with navy text (contrast: white fails on `--blue`, navy passes);
  green is now signals-only. Unified three front-page patterns: **avatars** (wrapper clips the
  image so it can't escape the ring), the **day row** (a 2-column grid replacing a ragged fixed
  column), and **inset blocks** (one `--inset-bg`/`--inset-border` convention for every nested box,
  incl. the push prompt, whose bespoke buttons were retired onto `.btn`/`.btn--ghost`).
- **Light/simple-first recalibration.** The early direction over-committed to the dark
  instrument look (buttons went uppercase/emerald app-wide, and the guide named
  instrument-panel "the target"). Corrected to the intended middle ground: light/calm default,
  instrument panels reserved for data-dense widgets, sentence-case buttons on the body font
  with the display font kept for numerals. `.btn` was re-skinned (solid `--mediumblue` /
  `--lightgreen` fills, no uppercase, no glow). Token/split/inline-migration work was
  aesthetically neutral and unaffected.

## Known gaps

Small residuals (safe to use as-is; migrate on touch). Phase status lives in [`wip.md`](wip.md).

- **The whole app is light** — every page + the shared modal were swept and the base `.card` was
  flipped light (the `.card--light` modifier is gone). No dark instrument surfaces remain.
- **Form controls are unified** (one `:where()` field skin — text inputs, `select`, `textarea` — that
  the modal fields match; one `--focus-ring`). **Alerts** are the remaining un-unified control: `.alert-*`
  still sits on the legacy semantic palette (and see the modal-over-alert-bar direction in Alerts). Unify on touch.
- **Fields & labels are systematised** (`.field` / `.field-label` / `.field-row` / `.field-check`);
  admin, news, registergoal, account (text fields) and the weight modal are migrated. **One known
  remainder:** account's **toggle-card** components (`.notification-option`, `.strava-option`) —
  bespoke centred-card layouts, a dedicated account redesign, not the `.field` system. (The
  exercise-builder set inputs are already on the unified field via `.we-set-input`/`.we-input`.)
- **Bespoke buttons** (`.wheel-*`, `.we-add-*`, `.user-stat-tab`, `.wv-media-repull`, `.trm-close`)
  and the **legacy global `button`** rule still stand (all light now; `.gear-btn` was retired).
- **Inline styles** — a few static one-offs remain (e.g. a couple of weight-log layout widths). The
  **skeleton size-class** pattern (`.skel-*`) is the preferred alternative.
- The **`.u-*` em values** could be tokenised to `--space-*` (a visual decision, deferred).
