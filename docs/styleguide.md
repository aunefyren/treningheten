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
web/css/instrument.css    the new instrument-panel design system: profile, heatmap, stat/streak cards,
                          tabs, chips, activity statistics, soundtrack, gear, exercises timeline
web/css/utilities.css     atomic helpers (spacing / alignment / sizing) — loaded last so they win
web/css/modal.css        the shared "telemetry panel" modal (TRModal, .trm-*)
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

Two eras exist in the codebase and both are kept light-leaning:
- **Ordinary pages** (seasons, account, forms, frontpage, news): light surfaces, the
  mediumblue/lightblue palette, simple cards. This is the default.
- **Deep-navy instrument panels** (`.trm` modal, `.workout-view`, and the richest stat/gear
  readouts) are the **exception, used sparingly** — reserved for genuinely data-dense widgets
  where a scoreboard feel earns its keep. Don't spread them across ordinary UI.

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

Use light content cards for ordinary UI; reserve the dark **instrument** stat cards for
data-dense metric readouts (statistics/gear/profile), per the light-first direction.

**Module panel** (light pages): the shared style for content blocks that sit on a
`.card--light` page — `.season`, `.current-week`, `.debt-module`, `.prize-module`,
`.leaderboard`, `.activities`, `.week_days`, `.ai-message-card`, … They render as **one identical panel**:
white surface, hairline grey border, a single blue `--module-accent` **left bar**, soft
shadow, `--radius-md`, `overflow: hidden`, `box-sizing: border-box` (so the padding stays
inside the width — some modules default to content-box and would otherwise render wider), and
**one consistent inner padding (`--space-4`)** — inner blocks (headers, progress bars) sit
**inset, never flush** to the edge. Driven by **one shared rule** (scoped under `.card--light`
in `components.css`) — keep every module on that rule; don't give a module its own colour,
border, or padding.

**Dividers:** one treatment inside a light module — a single hairline in `--grey`. `<hr>` and
every internal border (activity rows, previous-weeks) use it; never the old faint-eggshell `<hr>`
or the bluish `--trans-lightblue` borders.

| Class | Use | Status |
|---|---|---|
| `.stat-card` | metric tile; `data-family="movement\|effort\|strength\|audio\|time\|neutral"` sets a per-family accent stripe (`--stat-accent`). Sub-elements `.stat-card-label` / `.stat-card-value` / `.stat-card-unit`. | **target** |
| `.user-stat-card`, `.user-streak-card` | profile stat / streak tiles. **On a `.card--light` page** these become light **metric tiles**: `--inset-bg` fill + `--inset-border` hairline, the value in `--font-display` navy, muted `--lightblue` label. The dark-card look is kept for un-swept pages (`/account`). | target |
| `.ai-message-card` | the AI greeting widget | target |
| `.panel-card` | centred white content-card wrapper (extracted Phase 4) | transitional |
| `.card` / `.card-header` / `.card-body` | the shared page-shell wrapper (all 20 pages); mediumblue with white text | **legacy — being flipped light** |
| `.card--light` | opt-in modifier on `.card` → light surface + dark text (page sweep). Add it per page as swept; inner widgets built for the dark card get light-friendly overrides scoped under it. Once every page carries it, flip the base `.card` and delete the modifier. | **sweep-in-progress** |

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

The front-page week calendar **stacks each day vertically** (`.card--light .form-group`): the day
label, then the number box + its `−`/`+`/`✎` `.btn--icon` buttons on one line, then the notes field
full-width beneath. One consistent layout whether or not the action buttons show (they only appear
up to today), so notes never get squeezed into a narrow, wrapping column. (Replaced a fixed 6rem
column, then a 2-col grid whose notes column varied per row.)

### Hero — the weekly-goal ring

The front page opens on its core loop: a **conic-gradient progress ring** (`.hero-ring`) showing
`workouts / goal` for the week in the **display font**, over a palest-sky track. A JS-set `--pct`
(0–100) fills the blue arc — animated via `@property --pct` where supported, snapping otherwise —
and it flips to `--success` green at 100% with a brief pulse. The ring lives in a clean centred
white **hero panel** (`.card--light .hero`, capped `max-width` and centred) with *no* left accent
bar (it's the thesis, not a module). Hidden until a season populates it. No app-name title or
subtitle sits in the hero — the nav already names the app, and the ring + CTA carry it. This is the
page's signature element — keep the rest of the page quiet so it stays the one memorable thing.
The **debt-spin** state ("you must spin the wheel") reuses `.hero` so it reads as a panel, not
bare text.

### Chips & tags

Container `.tag-list` (read-only) or `.tag-selector` (editable); items are `.tag-chip`.
In a `.tag-selector`, chips render dimmed until `.tag-chip-selected` (solid green, bold).
`.tag-chip-readonly` disables the pointer. This is the target chip; use it for any tag UI.

### Segmented tabs

`.user-stat-tab` in a `.user-stat-tabs` bar — a segmented control for switching a panel (e.g. the
profile's month / year / all-time stats). On a `.card--light` page each segment is a light
`--inset-bg` pill; the active one (`.user-stat-tab-active`) is solid `--mediumblue`/white — the same
"active = solid mediumblue" rule as buttons. Use this for in-panel period/section switches; it is
*not* the top nav and not `.btn`.

### Meta tags

`.meta-tag` — a small hairline label (an achievement's category, a "Stackable" flag). Outline in
`currentColor`, `--radius-sm`, tiny text. Compose margins with `.u-*` utilities. Use it for a short
static descriptor on a tile; for interactive tag selection use `.tag-chip` instead.

### Skeletons

Loading placeholders are `.skeleton-block` (the shimmer; auto-darkens on `.card--light`). **Size
them with classes, never inline** — the profile page uses `.skel-label`, `.skel-streak`,
`.skel-tile`, `.skel-bar` (+ `.skel-row` to lay a row out, `.skel-mt` for section spacing). Add a
new named size class here rather than an inline `width/height` when a new skeleton shape appears.

### Form controls

An unopinionated baseline styles bare inputs/selects via
`:where(input[type=text|password|email|date|file|number|time], select)` in `base.css`
(low specificity — easy to override). Inside the modal, use the modal's own
`.trm-field` / `.trm-label` / `.trm-input` / `.trm-select`. `workout.css` sets its own
editor input type (≥16px so iOS doesn't zoom on focus). No single unified control yet —
prefer the modal fields in modals, the baseline elsewhere.

### Modal — `.trm` (TRModal)

The one shared modal ("telemetry panel"), driven by `web/js/modal.js` (`TRModal`); markup
is generated, not hand-written. Structure: `.trm` (overlay/root) → `.trm-overlay` (backdrop)
→ `.trm-panel` → `.trm-head` (`.trm-title`, `.trm-close`) + `.trm-body`. Inside the body use
`.trm-section-label`, `.trm-row` (side-by-side), `.trm-divider`, `.trm-field`/`.trm-label`/
`.trm-input`/`.trm-select`, `.trm-badge`, `.trm-eyebrow`. Buttons inside `.trm-body` default
to the primary look; add an explicit `.btn` class to opt into the button system. **Never
hand-render a modal** — extend TRModal. (The old `#myModal` image lightbox was removed in
Phase 3.)

### Alerts

`.alert-success` / `.alert-info` / `.alert-danger` (in `base.css`) — legacy light-theme
status banners, still on the legacy semantic palette. Fine to use; will be re-skinned toward
the instrument look in the page sweep.

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

Not yet unified (safe to use as-is; migrate on touch). Phase status lives in [`wip.md`](wip.md).

- **Cards / form-controls / alerts** aren't unified — legacy (`.card`, `.alert-*`, the
  `:where()` input baseline) coexists with the darker instrument versions. Light-first: the
  light legacy forms are fine for ordinary UI; use instrument styling only for data-dense
  widgets. Unify toward a calm, light shared version when touched.
- **Bespoke buttons** (`.wheel-*`, `.we-add-*`, `.gear-btn`, `.user-stat-tab`,
  `.wv-media-repull`, `.trm-close`) and the **legacy global `button`** rule still stand.
  (The front-page push-prompt buttons were migrated onto `.btn`/`.btn--ghost`.)
- **Inline styles** still remain on un-swept pages (mostly skeleton-loader sizes and bespoke
  one-offs). The **skeleton size-class** pattern now exists (`.skel-*`, front page + profile);
  extend it as pages are swept rather than re-adding inline `width/height`.
- The **`.u-*` em values** could be tokenised to `--space-*` (a visual decision, deferred).
