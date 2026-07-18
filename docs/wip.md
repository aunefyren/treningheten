# Work in progress
This doc should contains plans, ideas, problems, bugs and so on. Finished stuff should not live here, it should be moved to /docs in relevant files. If none fit, it should probably be made. If features are in between start and done implementation, one can still create a doc which get's updated over time.

## Plans & Ideas

### More workout tags
Ideas for tags:
- Easy
- Splits
Must respect Strava sync

### Sick leave is per season/goal
- makes sense that different seasons have different sick leave
- makes little sense that you can join multiple seasons at once, but only use sick leave on one goal

### Remove walk filter from Strava sync
- Instead, add a boolean to exercises, like "count toward goal" or similar
- Let users set on Strava settings whether any activity type count toward goal
  - Maybe not a Strava setting, maybe a global setting?
- Only on initial sync, if you edit any workout, simply change the bool if you want to count it
- Must be incorporated into every logic/if where the program counts amount of valid exercises
- Good opportunity to create helper functions? ExerciseCountTowardGoal() or/and IsSickLeave()?

### Flexible workouts
Work out more one week, have the extra effort carry over.
Must be season specific setting
Option to allow how many workouts carry over, how long they can exist before they decay
Must be user friendly and understandable in the UI

### Front page activities, add partner
- Allow a person to tag their partner/group on their exercise session within builder
- Show as activity together on season activity post if both have joined season
- Sync over Strava partner tag?
- Auto-detect partner of enough data?

### Music / audio integration
Connect music/audio providers on your account, then overlay your listening history onto
time-based activities. Full design + decisions now live in [`docs/media.md`](media.md).

**Still to build:**
- **Cross-activity stats — "fastest songs":** avg speed over
  `[StartedAt, EndedAt]` from a session activity's `OperationSet.StravaStreams`, which
  needs the stream-windowing done per track. (Per-track avg-HR on the card already shipped.)
- **Plex artwork:** Plex thumbs need the server token to fetch — store via a proxy
  rather than embedding the credential in a stored URL. (Spotify artwork already works.)
- **Audiobook/podcast classification:** Plex audiobooks surface as `track` → currently
  read as songs; classify from the Plex library section/agent instead.
- **Per-(session, provider) pull guard:** the single `Exercise.MediaRetrievedAt` spans
  both providers — the auto-trigger pulls all providers once together, which is fine for
  MVP, but connecting a provider *after* a session was already pulled needs the re-pull
  button. Generalize the guard when this becomes annoying.
- **Edge case (note, not solving now):** editing a session's **time** changes the
  window — the 🎧 re-pull re-matches on demand, but there's no automatic re-match on a
  time edit. (A skipped date-only session will match once a real time is set + re-pulled.)

**Open questions:**
- Privacy: listening data is sensitive even self-hosted — any per-activity visibility controls?
- Cross-provider de-dupe if a user has overlapping sources (e.g. casting Spotify through Plex)?
  (Per-provider rows side-step it for now; only matters once 2+ providers are connected.)

### Leave season button is not working
- Which season? all?
- Not broken, never built function, only button is present

### Delete account button is not working
- What gets left behind? Do seasons you joined still show you? Show 'Deleted user'?
- Not broken, never built function, only button is present

### Best effort system
- Manual programming per activity?
- "fastest 5K"...
- Must be calculated at save or during runtime?
- Notification integration for PR?
- PRs for reps and weight on strength exercises?
- Time based best efforts? During this season? During this year?

### AI Ollama feedback on exercises?
Per-exercise feedback in its own dedicated space (not the front-page greeting).
- How to avoid spamming the LMM
- Little model, can the feedback be decent?

### Style guide
- There is old (meh), simplistic UI (/seasons, cards on front page) thats needs fixing
- There is new, over the top UI elements (gear modal)
- Page needs a coherent style guide, somewhere in between
- Every button, card, page should be gone through to ensure coherent style
- Good opportunity to rework CSS, split into multiple shared files, less globals, more variables

**Direction (confirmed):** *light/simple-first — modern bones, calm skin.* NOT converting the
app to the dark gear/workout-modal look. Keep the modern discipline (one token system,
consistent spacing, shared components, navy + green accent) but on **mostly light, calm
surfaces**, **sentence case**, subtle depth. Deep-navy instrument panels (`.trm`,
`.workout-view`, richest stat readouts) are the **sparingly-used exception** for data-dense
widgets only. Display font = numerals/readouts only; green accent used sparingly. (This
corrected an early over-commit to the instrument look; `.btn` was re-skinned to match — see
`styleguide.md` decisions log.)

**Current state (audit):** `web/css/main.css` is one ~3990-line monolith (of ~5200 total
across main/workout/modal/admin). It's visibly two eras: a legacy Bootstrap-derived top
half (three competing button systems — `.btn`, `.regular-button`, `.danger-button`; a
stray `rgba(0,123,255)` blue focus ring not in the palette) and a newer "instrument-panel"
bottom half (from ~line 3012, sectioned with `── … ──` comments — stat/streak cards,
activity tabs/tags) which is the target aesthetic. Root problems:
- **No token layer.** 18 colour vars but two palettes mixed (navy instrument set vs legacy
  semantic red/yellow/green) + ~80 hardcoded hexes in main.css. Zero spacing/radius/shadow/
  typography tokens — every value is a magic number, so there's nothing to be consistent to.
- **Styling lives in JS.** ~374 inline `style=` strings + heavy `.style.` use (frontpage,
  statistics, account, user worst). A CSS-only style guide can't be enforced while half the
  visual decisions happen at render time in JS.
- **Arbitrary split.** `modal.css` and `main.css` both define "Modal Content" (dup). No
  `docs/styleguide.md` exists; the style is undocumented.

**Phased plan (foundations first, each phase reviewable on its own):**
1. **Token layer — DONE.** Deduped palette + spacing/radius/shadow/type scales +
   *semantic* aliases now live in `web/css/tokens.css`, documented in
   [`styleguide.md`](styleguide.md). `--trm-*`/`--wv-*` prefixes removed;
   `main.css`/`modal.css`/`workout.css` (+ one inline style in `exercise.js`) repointed
   at the semantic names; `tokens.css` linked first in all 20 HTML heads. Values-preserving
   refactor — no intended visual change (maintainer to confirm visually).
   **Approach as executed:** the instrument-panel language already existed as tokens twice
   (`--trm-*` in modal.css, `--wv-*` in workout.css — near-identical). Phase 1:
   - New `web/css/tokens.css` = the single authoritative `:root`. `<link>`ed before
     `main.css` in all 20 HTML heads (no shared head partial exists).
   - Move the legacy `main.css` palette (`--red/--yellow/--green/--lightblue/…`) into it
     unchanged, and delete `main.css`'s `:root` (values identical → no visual change).
   - Migrate `modal.css` + `workout.css` off the `--trm-*/--wv-*` prefixes onto the new
     semantic names; delete both prefix blocks. Canonical values for the near-dupes:
     `--text: #f3f6fb`, `--text-dim: #9fb0c7`, `--surface-hi: rgba(255,255,255,.08)`
     (the workout/wheelview values — most recent surface; diffs are imperceptible).
   - Net-new scales (biggest gap today): `--space-*`, `--radius-*` (standardize on
     `.5rem`/`1rem`/pill), `--shadow-*`, `--font-display`/`--font-body`.
   - Saira/Hanken referenced but never loaded (only Roboto is) — keep the token, defer
     actually loading the webfonts to the component phase. Flagged, not fixed.
   - Verify by grep (zero remaining `--trm-*/--wv-*`, values preserved); maintainer does
     the visual pass. Ships with the Phase 5 `docs/styleguide.md` skeleton.
2. **Component primitives** — one button, one card, one form-control, one chip. Adopt the
   instrument-panel look as target; retrofit legacy pages to it.
   - **Button (`.btn`) — DONE.** BEM system (`.btn` + `--primary/--danger/--ghost/--sm/
     --block`) in main.css, documented in [`styleguide.md`](styleguide.md). All 68 `<button>`
     call sites migrated off `regular-button`/`danger-button`/`trm-btn`/`btn-primary`/
     classless (legacy selectors deleted); `.btn` is self-defining so it overrides the
     still-standing global `button` rule. Bespoke buttons (wheel/builder/modal/tabs) left
     for the Phase 6 sweep. Saira/Hanken webfonts now loaded on all 20 pages.
   - Remaining Phase 2: card, form-control, chip primitives.
3. **CSS restructure — DONE.** `main.css` (3983 lines) split **sequentially** into
   `base.css` (reset/nav/forms/buttons) + `components.css` (legacy components) +
   `instrument.css` (new design system) — cuts on section boundaries so
   `cat base components instrument` reproduced `main.css` byte-for-byte (verified) →
   zero cascade change. All linked in order after `tokens.css` on all 20 pages; `main.css`
   removed. Dead image-lightbox modal removed (`.modal`/`.modal-content`/`#caption`/`.close`,
   0 usages — `#myModal` was long ago replaced by TRModal). Per-page extraction deliberately
   deferred: `.stat-card` (statistics+user) and `.soundtrack` (exercise+statistics) are
   shared, so page-splitting them would break pages; all three sheets stay global for now.
4. **Migrate JS inline styles → classes** — page by page, worst offenders first.
   **Status: 374 → 128 inline styles, all 20 JS files done.** Approach = hybrid:
   `utilities.css` (`.u-*` atomic helpers, values preserved verbatim, loaded last) + component
   classes (`.integration-btn`, `.panel-card`, `.panel-wide` in components.css). Removed 95
   empty `style=""` globally; migrated the 4 worst (account/frontpage/user/statistics) then
   all 16 remaining files. Visibility (`display:none` + `.style.display`, 97 JS toggles) left
   inline on purpose — it's state, not theme; dynamic `${…}` and `!important` overrides too.
   **Guardrail:** converting a standalone `style="…"`→`class="…"` collides when the element
   already has a non-adjacent `class=` (duplicate attr → second ignored, style lost). Fix =
   grep `class="…"[^>]*class="` and merge the two attrs (regex:
   `s#class="([^"]*)"([^>]*) class="([^"]*)"#class="\1 \3"\2#`). Verified zero duplicates.
   **Residual (128): 56 legitimately inline** (49 visibility, 5 dynamic, 2 `!important`);
   **72 bespoke one-offs** — mostly per-context **skeleton-loader / preview-box sizes**
   (`width×height×border-radius`), font-size combos, ad-hoc flex. Refinement left: dedicated
   **skeleton size classes** for those; optionally tokenise the verbatim `em` utility values.
5. **`docs/styleguide.md` — DONE.** Authoritative frontend visual reference: CSS layer/file
   layout + load order, the three token tiers, all shared components (`.btn`, cards,
   chips/tags, form controls, the `.trm` TRModal, alerts, `.u-*` utilities), an "Adding new
   UI — the rules" section, a decisions log (Phases 1–4), and a "Known gaps" list. Indexed
   in `docs/README.md` under Conventions.
6. **Page sweep old→new** — light/simple-first, page by page, maintainer verifies each live.
   Per-page checklist: buttons on `.btn`; surfaces calm/light + token-based; sentence case,
   display font for numerals only; finish inline-style tail; retire page-specific legacy cruft.
   **Migration mechanism:** the shared `.card` shell (mediumblue + white text, all 20 pages)
   is flipped light via an opt-in `.card--light` modifier added per page — inner dark-card
   widgets get light-friendly overrides scoped under it. Once every page carries it, flip the
   base `.card` and delete the modifier.
   - **Front page — IN PROGRESS.** `.card--light` = **soft eggshell card** so the content
     "module" blocks (`.season`/`.current-week`/`.debt-module`/`.prize-module`/`.leaderboard`/
     `.week_days`/`.ai-message-card`, all already white + `0.5rem` radius) read as one consistent
     set of **white panels + thin border + soft shadow** — fixes the white-on-white the
     maintainer flagged. `.ai-message-card` (was dark-instrument translucent) folded into the
     same panel profile. Removed a redundant inline `cursor:pointer`.
     **Unified module panel + single accent** (via `/frontend-design`): multi-colour domain bars
     were jarring → reverted to **one** shared style for every module — white, hairline border,
     a single **blue** `--module-accent` left bar (blue theme; other hues reserved for
     success/warning/error signals), soft shadow, `--radius-md`, `overflow:hidden` so inner
     headers/progress bars clip to the rounded corners (fixes the top-overlap). Inner tones calmed
     (lightblue `week_info` → eggshell; progress track eggshell, blue fill). Headings/title navy;
     AI shimmer fixed (dark sweep). One shared rule drives all modules; documented as the
     **module-panel** pattern in `styleguide.md`.
     **Refinement pass (via `/frontend-design`, from the maintainer's 2026-07-15 screenshots) — DONE, awaiting visual review:**
     - **One sky-hue blue.** The theme had two blue families — a slate `--mediumblue`/`--lightblue`
       (~213°) clashing with the brighter `--blue` accent (~206°). Warmed the slates onto the
       accent's hue (`#415a77`→`#3a6ea5`, `#778da9`→`#7fa8cf`); added `--blue-050` tint +
       `--inset-bg`/`--inset-border`. App-wide token change (intended coherence).
     - **Primary CTA → blue.** `.btn--primary` off green onto the bright `--blue` accent with navy
       text (white fails contrast on `--blue`); green is now signals-only.
     - **Avatars** clip to their ring (`overflow:hidden` on the wrapper) — fixes photos escaping.
     - **Day row** is a 2-column grid (label + number/±/✎ cluster | notes) — fixes the ragged calendar.
     - **Inset blocks:** one `--inset-bg`/`--inset-border` convention for every nested box (progress
       track, debt notices, season rows, number box, push prompt). Push-prompt buttons → `.btn`/`.btn--ghost`.
     - **Signature shipped:** the weekly-goal **ring hero** (`.hero-ring`) — display-font `workouts/goal`
       + conic-gradient arc filling blue, green at 100%; replaces the calendar's week bar. All documented
       in `styleguide.md` (palette, Avatars, Day row, Hero, Inset blocks + decisions log).
     **Refinement pass 2 (from live review) — DONE, awaiting visual review:**
     - **One button blue.** Walked back the bright-accent primary (couldn't carry white text; navy-on-bright
       clashed with the neutral buttons) → every solid button is `--mediumblue`/white; `.btn--primary` is
       now visually identical to `.btn`. Bright accent reserved for rings/progress only.
     - **Hero trimmed:** dropped the redundant app-name title + subtitle (nav names the app; ring + CTA carry it).
     - **Module width fixed:** `.leaderboard`/`.activities` were content-box, so panel padding pushed them to
       22rem — added `box-sizing: border-box` to the module-panel rule.
     - **Dividers unified** to one `--grey` hairline (were eggshell `<hr>` + bluish `--trans-lightblue` lines).
     - **Debt-spin state swept** onto `.hero`.
     Inline-style tail (~7) still to tidy. (Verify state is the separate `/verify` page — out of this sweep.)
   - **`/users/:id` — DONE, awaiting visual review.** Shell → `.card--light`; profile+stats row and
     achievements are module panels; metric/streak tiles relit as light inset tiles with display-font
     navy numerals; activity switcher → segmented tabs; category/"Stackable" → `.meta-tag`; dividers →
     grey hairline. Inline styles moved to CSS (skeletons → `.skel-*`, flame → `.flame-*` + `--flame-*`
     tokens, brand links → `.user-link`); only `display:none` state + the data-driven `--cat-color` stay
     inline. All scoped under `.card--light` so shared classes on `/account` + `/achievements` are
     untouched until their sweep. New patterns documented in `styleguide.md` (Segmented tabs, Meta tags,
     Skeletons, light metric tile).
   - **`/login` — DONE, awaiting visual review.** Already had `.card--light` + `.btn`; finished it:
     form wrapped in a centred **`.auth-panel`** (hero-style card, no left bar), went flat inside
     (dropped internal `<hr>`s + empty duplicate-id `#form-input-icon` labels), submits →
     full-width `.btn.btn--primary.btn--block`. New `.auth-panel` pattern documented.

#### Phase 6 page-sweep — process & status (RESUME HERE)

**Important:** all 20 HTML shells **already carry `.card--light`** (added broadly in the "css
imprvoements" commit). So the remaining work per page is **not** the shell — it's migrating that
page's **JS-rendered content** to the light system, moving inline styles into CSS, and reusing the
shared patterns. Once every page's content is verified, flip the base `.card` to light and delete
the `.card--light` modifier (endgame).

**Per-page checklist (repeatable):**
1. Read the page's `web/html/<page>.html` (confirm `.card--light`) and `web/js/<page>.js`.
2. **Group content into module panels** (`.card--light`-scoped) — reuse the module-panel rule or
   `.hero`/`.auth-panel` for centred single-purpose pages.
3. **Buttons** → `.btn` (+ `--primary`/`--ghost`/`--block`); one primary per view. Kill bespoke buttons.
4. **Reuse components before inventing:** avatars (wrapper clips), inset blocks (`--inset-bg`/`--inset-border`),
   light metric tiles (`.user-stat-*`), segmented tabs, `.meta-tag`, chips, dividers = one `--grey` hairline,
   display font for numerals only.
5. **Move CSS out of JS:** static inline `style=` → classes (skeletons → `.skel-*`, colour tiers → token-backed
   classes, etc.). Leave inline ONLY: visibility (`display:none`) state and genuinely dynamic (`${…}`/CSS-var) values.
6. **Scope new overrides under `.card--light`** so shared classes on un-swept pages stay put.
7. **No hardcoded hex/px** — use tokens; if a value is new, add the token + document it.
8. **Document** every new pattern/decision in `styleguide.md` (component + decisions log) and tick this list.
9. `go build ./...`, `node --check` the JS; maintainer does the visual pass (don't self-screenshot).

**Status:**
- ✅ **Content swept:** front page, `/users/:id`, `/login`, `/register`, `/account`, `/verify`,
  `/authorize`, `/offline`, `/seasons`, `/admin`, `/registergoal`, `/achievements`, `/news`,
  `/wheel`, `/exercises`, `/gear`. (`/oauth` = transient.)
- ⬜ **Shell is light, content NOT yet swept:** countdown, exercise, statistics.
- **Coupling note:** `/gear`'s light `.gear-*` styles are scoped under `.card--light`; the **dark
  base `.gear-*` is kept for the exercise-page gear modal**. When `/exercise` is swept, convert the
  modal too and fold the `.card--light .gear-*` overrides back into the base (drop the dark values +
  the now-dead `.gear-btn` rule).
- **Direction clarified: NO dark exception — everything goes light.** The dark "instrument" skin is
  legacy to convert (Design-language section updated). `/exercises` was the first heavy conversion:
  the `.feed-*` timeline went dark→light (module panel + inset rows + navy text + tokens).
- **Remaining data-dense pages still on the dark instrument skin — convert to light:** `exercise`
  (builder / `.workout-view`), `gear`, `statistics` (stat cards + heatmap). Plus the `.trm` modal
  (shared, dark) will need converting eventually. `countdown` still pending keep/remove.
- The `.account-section`, `.auth-panel`, module-panel + `.hero` patterns are reusable.
- **Heaviest (data-dense, instrument panels legit):** `exercise` (builder), `exercises` (timeline),
  `gear`, `wheel`, `statistics`.
- **Countdown pending a decision** (see the "/countdown still in use?" note): keep+sweep or remove+rework front page.
- **Heaviest (data-dense, instrument panels legit):** `exercise` (builder), `statistics`, `gear`, `wheel`.
- **Heaviest:** `exercise` (builder), `statistics` (charts/heatmap), `gear`, `wheel` — data-dense; instrument
  panels are legit there, migrate carefully.
- **Endgame:** when all content is swept + verified, flip base `.card` → light and delete `.card--light`.

Recommended first PR: Phase 1 (tokens) + the skeleton of the Phase 5 doc together — small,
low-risk, and the anchor for all later work.

### /exercises rebuild — searchable activity timeline (IN PROGRESS)
Replacing the year → week → day accordion on `/exercises` with a searchable, sortable
**activity timeline** that floats key metrics inline so you don't have to click into a day.

**Model recap:** `ExerciseDay` (calendar day = the `/exercises/:id` builder) → `Exercise`
(session; several per day) → `Operation` (one activity type, has an `Action`) →
`OperationSet` (reps/weight/distance/time/HR streams). Search targets ("longest run", "a
padel match", "oldest run") live at the **Operation** grain; browsing wants **session**
grouping. So the feed is activity-grained with session/day grouping metadata.

**Decisions (agreed):**
- **Hybrid feed.** Browse mode = grouped by day + session; find mode = flat ranked
  activities when a metric sort or type filter is active. One endpoint, client picks mode.
- **Query-time aggregation** for v1 (no schema change). New endpoint aggregates
  `operation_sets` in SQL (`GROUP BY` operation → SUM distance/time/reps, MAX weight,
  COUNT sets). API shaped so we can swap in precomputed Operation rollup columns later
  **without changing the response**. HR avg/max deferred to the detail view for v1 (lives
  in the stream blobs; too heavy for a list).
- **Pure timeline** — drop the year accordion + goals-by-year from `/exercises` (goals live
  on Seasons/Statistics). Keep the "Manage gear" link.

**Backend (new, list-first):**
- `GET /auth/activities` — filters: `action_id`, `start`/`end` (date range), `q` (note /
  action name), `has_distance`; sort: `date|distance|duration|weight|reps` + `order`;
  pagination `limit`/`offset`. Returns slim per-activity items (operation id, `exercise_id`,
  `exercise_day_id`, day date, session time, action {name/type/logo}, session activity
  count, note, tags, aggregated metrics {distance+unit, duration, reps, top weight+unit,
  set count}, `has_strava`/`has_gps`) — **no `strava_streams`** in the list. Plus `total`
  and `has_more`. A small companion count map gives `session_activity_count` per returned
  `exercise_id`.
- Reuses the existing enabled-chain joins (operations→exercises→exercise_days→users) from
  `GetOperationsByUserID`.

**Frontend:** `web/js/exercises.js` rewritten to a filter/search bar + infinite-scroll
timeline; browse groups adjacent same-session activities under day headers, find mode shows
a flat ranked list with the sorted metric prominent. Each card links to `/exercises/:dayID`.
Dark instrument-panel styling consistent with the stats/gear redesign.

**Builder considerations (`/exercises/:id`, NOT in this pass — captured for the fast-follow):**
- The builder still exposes gear at the **session** level only, though the schema stores it
  per **operation** — a genuinely multi-type/mixed-gear session can't be edited per activity
  (see the gear follow-ups above). The multi-type representation problem is the same root:
  the builder doesn't cleanly show/organise a session that has several `Operation`s of
  different `Action`s.
- The per-activity **aggregate shape** built for `/auth/activities` is exactly what a better
  **session summary header** on the builder should consume (activity chips + per-activity
  metrics) — reuse it there rather than re-deriving.
- When the builder is redone: consider a per-`Operation` card layout (each activity type its
  own sub-card with its own gear/metrics/sets), a session header summarising the mix, and
  clearer affordances for adding a *second activity type* to an existing session vs a *second
  session* to the day.
- Watch the media/soundtrack coupling: soundtrack is session-scoped (`Exercise`), so builder
  changes to session time/duration affect the match window (already noted under media).

### /exercises has been refined, let the MCP server benefit
- Users can now more easily find exercises on /exercises
- The MCP server should also be able to find exercises
- Allow MCP to find relevant exercises without shifting through tons of data

### Gear tracker — possible follow-ups
The gear feature shipped (see [`docs/gear.md`](gear.md)). Open refinements left for later:
- **Per-operation gear UI.** The schema stores gear on the operation, but the builder only
  exposes a session-level selector. A combined Strava session that genuinely mixes gear can't
  be edited per-activity yet.
- **Auto-assign primary.** The selector *suggests* the user's primary gear for a session with
  no gear, but it isn't persisted until the user interacts. Could auto-assign on the first
  operation instead.
- **Primary per type.** Only one primary per user today; a primary shoe *and* a primary bike
  might be more useful.

### Better gear management
- Or maybe this is finished now that we have a /gear page?
- The modal covers the entire /gear page? Move stuff away from modal? Remove modal?

### Make first day of the week changeable
- Default monday, but choose
- Big changes to logic

## Problems

### Site loads
But sometimes not? Server asleep?