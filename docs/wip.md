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

### Dead code on /registergoal
- Used to be a bigger page
- Has sections like debt overview
- This page should only have goal registration now, nothing else
- Needs cleanup

### Locked achievements CSS bug
- Achievements with the pad lock icon on /achievements have a rounded border around the icon, like a margin between the icon and the rounded color around the achievement
- SVG get's cut off on the corners because of this

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

### MFA enrollment on /account

## Problems

### Site loads
But sometimes not? Server asleep?