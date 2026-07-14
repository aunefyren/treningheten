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

Three tiers. Components should consume the **semantic** and **scale** tiers, not
raw primitives.

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
| `--danger` | destructive/error |
| `--strava` | Strava brand orange |
| `--audio` | soundtrack / listening layer amber |
| `--module-accent` | the single blue theme accent for module panels (left bar + progress). One knob recolours the whole set. |

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

### Button — `.btn`

The one button primitive. Calm, light-first look: solid fill, **sentence case**, body font,
`--radius-sm` corners (no uppercase, no glow). `.btn` is **fully self-defining** (sets its own
width/margin) so it overrides the legacy global `button` element rule wherever applied.

```
<button class="btn">Neutral action</button>
<button class="btn btn--primary">Primary CTA</button>
<button class="btn btn--danger">Delete</button>
<button class="btn btn--ghost">Low-emphasis</button>
<button class="btn btn--primary btn--sm">Small</button>
<button class="btn btn--block">Full width</button>
```

| Class | Use |
|---|---|
| `.btn` | base — neutral action, solid `--mediumblue` fill, white text |
| `.btn--primary` | green CTA (`--lightgreen`) |
| `.btn--danger` | destructive (`--red`) |
| `.btn--ghost` | transparent + `--mediumblue` border, low emphasis |
| `.btn--sm` | compact size |
| `.btn--block` | full-width |

`<img>` inside a `.btn` is auto-sized to 1rem (leading icon). Focus shows an
`--accent-soft` ring; `:disabled` dims to 0.55.

**Not part of this primitive** (bespoke buttons with their own components, retired
later in the page sweep): `.wheel-clear`/`.wheel-swatch` (colour picker),
`.we-add-exercise`/`.we-add-set` (builder rows), `.wv-media-repull`, `.trm-close`
(modal X), `.push-prompt-*`, `.user-stat-tab`, `.gear-btn`. The legacy global
`button` element rule also still stands — it styles those bespoke and any not-yet-
migrated buttons until they're individually retired.

### Cards

Use light content cards for ordinary UI; reserve the dark **instrument** stat cards for
data-dense metric readouts (statistics/gear/profile), per the light-first direction.

**Module panel** (light pages): the shared style for content blocks that sit on a
`.card--light` page — `.season`, `.current-week`, `.debt-module`, `.prize-module`,
`.leaderboard`, `.activities`, `.week_days`, `.ai-message-card`, … They render as **one identical panel**:
white surface, hairline grey border, a single blue `--module-accent` **left bar**, soft
shadow, `--radius-md`, `overflow: hidden`, and **one consistent inner padding
(`--space-4`)** — inner blocks (headers, progress bars) sit **inset, never flush** to the
edge. Driven by **one shared rule** (scoped under `.card--light` in `components.css`) — keep
every module on that rule; don't give a module its own colour, border, or padding.

| Class | Use | Status |
|---|---|---|
| `.stat-card` | metric tile; `data-family="movement\|effort\|strength\|audio\|time\|neutral"` sets a per-family accent stripe (`--stat-accent`). Sub-elements `.stat-card-label` / `.stat-card-value` / `.stat-card-unit`. | **target** |
| `.user-stat-card`, `.user-streak-card` | profile stat / streak tiles | target |
| `.ai-message-card` | the AI greeting widget | target |
| `.panel-card` | centred white content-card wrapper (extracted Phase 4) | transitional |
| `.card` / `.card-header` / `.card-body` | the shared page-shell wrapper (all 20 pages); mediumblue with white text | **legacy — being flipped light** |
| `.card--light` | opt-in modifier on `.card` → light surface + dark text (page sweep). Add it per page as swept; inner widgets built for the dark card get light-friendly overrides scoped under it. Once every page carries it, flip the base `.card` and delete the modifier. | **sweep-in-progress** |

### Chips & tags

Container `.tag-list` (read-only) or `.tag-selector` (editable); items are `.tag-chip`.
In a `.tag-selector`, chips render dimmed until `.tag-chip-selected` (solid green, bold).
`.tag-chip-readonly` disables the pointer. This is the target chip; use it for any tag UI.

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
- **Bespoke buttons** (`.wheel-*`, `.we-add-*`, `.gear-btn`, `.user-stat-tab`, `.push-prompt-*`,
  `.wv-media-repull`, `.trm-close`) and the **legacy global `button`** rule still stand.
- **~128 inline styles remain** (56 legitimately dynamic/visibility; ~72 bespoke one-offs,
  mostly skeleton-loader sizes — want dedicated skeleton size classes).
- The **`.u-*` em values** could be tokenised to `--space-*` (a visual decision, deferred).
