# The `/exercises` workout timeline

`/exercises` is a searchable, sortable **workout timeline**: it floats each session's key
metrics inline so you can browse recent workouts or hunt a specific one ("my longest run",
"that padel match") without clicking into a day. It replaced the old year → week → day
accordion.

## Data grain

Recall the domain spine (see [data-model.md](data-model.md)):

```
ExerciseDay (calendar day, the /exercises/:id builder)
  └─ Exercise (session; several per day)
       └─ Operation (one activity type; has an Action)
            └─ OperationSet (reps / weight / distance / time / HR streams)
```

The feed's unit is the **session** (one item per `Exercise`), with its activities nested
inside it.

It was originally **activity**-grained (one item per `Operation`), which had a blunt
consequence: a session with no operations never appeared at all. That is not an edge case —
the front page's "+" button (`addExercise` → `POST /auth/exercises` → `APICreateExercise`) and
the week calendar both create an `Exercise` with **zero** `Operation` rows, so the most common
way to log a workout was invisible in the timeline while still counting toward the goal and
showing up on the front page and in the shared feed. Rooting the feed in the session fixes
that, and makes the counts honest: `total`, `limit`/`offset` and the "showing 12 of 340" line
are all measured in workouts.

The activity grain still exists as the nested `activities` array — every activity keeps its own
id and metrics, and remains what `get_activity` and the builder address. The MCP search runs the
same session query (see [mcp.md](mcp.md)); the standalone operation-rooted feed query was deleted
with the last caller.

## Two modes, one endpoint

The client picks the mode; the endpoint is the same:

- **Browse** — no metric sort or type filter active. Session cards are grouped under day
  headers. This is the default landing view, newest first.
- **Find** — a metric sort or an activity-type filter is active. The list goes flat and
  ranked, with the sorted metric shown prominently.

The activity-type filter can be set from the URL (`/exercises?action=<id>`) and is written
back to it, so `/statistics` can link straight to "all your runs" and a filtered list survives
a reload. The activity-type filter lists only the types the user has actually logged
(`GET /api/auth/actions?experienced=true`, the same source as `/statistics`), sorted by name.
The full catalogue holds hundreds of Hevy exercises the user has never done, and choosing one of
those could only return nothing.

Both modes rank and page **sessions**. A metric sort therefore reads as "my longest workout"
rather than "my longest single activity" — for `weight` the two are identical (a session's max
set *is* its heaviest activity), and for distance/duration/reps the session total is the more
useful answer. The card lists its activities, so which part of the workout produced the number
is still visible without opening it.

## Backend

`GET /auth/activities` → `controllers.APIGetActivityFeed`. Query parsing/validation lives in
`parseActivityFeedFilter` (a bad `action_id`, date, `sort`, or `order` is a `400`; `limit` is
clamped to `[1, 100]`, a bad `offset` falls back to `0`). Dates accept either a full RFC3339
timestamp or a bare `YYYY-MM-DD` (read as midnight).

Filters: `action_id`, `start`/`end` (date range), `q` (case-insensitive substring over the
operation / session / day notes **and** the action name), `has_distance`. Sort:
`date | distance | duration | weight | reps` with `order` = `asc | desc`. Pagination:
`limit` / `offset`. Response: `{ sessions: [...], total, has_more, message }`.

The aggregation is **query-time** (`database.GetSessionFeedForUser`), in two queries:

1. **The session page** — root table `exercises`, joined up to `exercise_days` for the user and
   date, then **LEFT** JOINed down to `operations` and `operation_sets`. The left joins are the
   whole point: an inner join drops sessions that have no activities. Per session it returns
   `SUM(distance)`, `SUM(repetitions)`, `MAX(weight)` as top weight, `COUNT(sets)`, a
   `has_strava` flag, `activity_count`, and a duration that **cascades** the session's own
   `Exercise.Duration` over `SUM(set.time)` (seconds — repo convention; same cascade as
   `resolveSessionWindow`), so a session logged with nothing but a duration still reads as one.
   It also carries the session-level `counts_toward_goal` and `private` flags, so the card can
   mark a logged-but-excluded session and one hidden from everyone else — these used to be
   repeated onto every activity row of a session, since they never varied within it.
2. **The activities** — one fetch of every enabled activity belonging to the returned sessions,
   attached in logging order. It is deliberately **unfiltered**: the filters decide which
   *sessions* match, never which activities are shown.

That second point is why the activity-level filters (`action_id`, `q`, `has_distance`) are
expressed as **`EXISTS` subqueries** (or a `HAVING` on the session total) rather than as
predicates on the join. A predicate on the join would narrow the aggregate too, so a session
matched on its run would report only the run's distance and list only the run. Matching is
"the workout contains something that matches"; the card then describes the whole workout.

Each session is a `models.SessionFeedItem` holding a slim `models.ActivityFeedItem` per
activity — **no `strava_streams`** (too heavy for a list). The activity does carry a handful of **precomputed stream scalars** (`avg_heartrate`, `max_heartrate`,
`avg_cadence`, `temp_c`, `elevation_gain_m`) read directly from rollup columns on the `Operation`
(`models.ComputeStreamRollup`, written on Strava sync and by the versioned stream backfill
`backfillStreamDerivedData`, see [statistics.md](statistics.md#stream-derived-data-and-its-version)) plus summed `moving_seconds` — so the list can show those
numbers without ever loading or parsing a stream blob. The full per-second HR/GPS detail is still
deferred to the builder/detail view. This is the "precomputed rollup columns without changing the
JSON" the shape was designed for; the MCP `list_workouts` search consumes the same fields.

## Frontend

`web/js/exercises.js` is a filter/search bar + infinite-scroll timeline against
`api_url + "auth/activities"`. It groups session cards under day headers in browse mode and
shows a flat ranked list in find mode. Each card links to `/exercises/:dayID` (the builder),
shows a muted "Doesn't count" badge when `counts_toward_goal` is false and a "Hidden" badge
when `private` is true, and lists the session's metrics — distance, duration, reps/top weight.

**A session with exactly one activity collapses** (`feedSessionCard`): the card takes that
activity's icon and name and draws no breakdown, so the common case — one imported run — reads
as a single line rather than as a box wrapped around one row. It also borrows that activity's
**avg HR** and **elevation gain** scalars, which are per-activity readings with no meaningful
session-wide sum. Two or more activities get the breakdown underneath, and the icon of the
session's dominant category (🏋 strength, 🏃 cardio, 🎾 sport; the generic 🏅 when mixed).

**Long sessions fold.** From five activities up (`feedCollapseFrom`), a card shows the first three
(`feedVisibleActivities`) and folds the rest behind a "Show N more" row that unfolds in place.
Rows matching the active exercise filter or search text are always among the visible ones, and
both parts keep the workout's order. The card's title is a real link to `/exercises/:dayID`, so
the feed works from the keyboard. A click elsewhere on the card opens it too, except on the fold
toggle.

### Session headline (title, feed phrase, logos)

Every surface that names a session — the session header on `/exercises/:id`, the `/exercises`
cards and the front-page feed — reads the server's **`headline`** (`models.SessionHeadline`,
built by `controllers/session_headline.go`) instead of deriving its own. It is attached to
`ExerciseObject`, `SessionFeedItem` and the front-page `Activity`. Rules, in order:

1. **Title.** A session imported from **Strava or Hevy keeps the source's own title** (its
   `Exercise.Note`: "Afternoon Run", "Push day") — the maintainer's choice, as it is more
   personal. A manual session's note is a free-text note, not a title, so its title is
   generated: the single activity's name when there is one kind; **"Strength training"** when
   strength covers two thirds or more of the activities; the dominant category's activity
   names otherwise ("Run + Bicycling", then "+ N more"); and for a genuinely mixed session up to
   two category runs ("Run + strength training"). An empty session is "Workout".
2. **Feed phrase** (`past_tense_verb`). The single action's own phrase ("played some padel"),
   else the dominant category's ("did some strength training"), else "did a range of
   workouts" — now only for genuinely mixed sessions.
3. **Logos.** Each activity's logo, de-duplicated; strength activities without a logo of their
   own (every Hevy lift) borrow the Weight Training logo, so a lifting session shows it once.

The rules hang off **`Action.Category`** — strength / cardio / sport / mobility / general
(`models/action_category.go`) — which says what kind of training an action is, independent of
how it is measured (`Action.Type`: Weight Training and Yoga are both `timing`, Plank is
`lifting`). Curated actions set it by hand in `database/seed.go`; Hevy catalog actions get it
from the template's muscle group (cardio → cardio, "other" → mobility, everything else →
strength); user-created actions pick it in the "Add exercise" form or default from their type
(lifting → strength, moving → cardio, timing → general). `backfillActionCategories` filled
existing rows once.

### Equipment

Equipment belongs to the **exercise**, so it lives on **`Action.Equipment`** (barbell, dumbbell,
machine, cable, smith_machine, kettlebell, band, suspension, bodyweight, trap_bar, sandbag; empty
when unknown), not on each logged activity. Hevy catalog actions take it from the bracket in
their title ("Bench Press (Dumbbell)" → dumbbell, `models.HevyTitleEquipment`; qualifiers such as
"Weighted" are not equipment). The strength card leads its summary line with it ("Dumbbell · 3
sets · …"), and the "Add exercise" form offers it.

The old per-activity `Operation.Equipment` field is **legacy**: the builder no longer shows or
sends it, and the update endpoint leaves it untouched when omitted (it used to blank it), so data
from older sessions is kept and still reaches MCP.

The URL and the nav item stay **"Exercises"**; the page calls its items **workouts**, and only
the API payload and the internals talk about sessions (see [vocabulary.md](vocabulary.md)). Styling follows the shared light module/inset system
(see [styleguide.md](styleguide.md)): the **session card** is the panel's inset tile and the
activities inside it are unboxed, so the panel keeps one border system.

## Day page (`/exercises/:id`)

The page for one calendar day (`ExerciseDay`). Below the header sit the day's workouts, each
a read view with an editor behind it (see *Activity detail*), then the add row, Combine and
links to `/exercises`, `/gear` and, in the current week, the front page.

### Day header

One panel above the workouts, built in `web/js/exercise.js` (`dayPageHTML`, `placeDayWeek`):

- **Date**: weekday as the heading, the full date under it.
- **Week count**: the week's counted workouts, the same number as the front-page ring.
- **Week strip**: one tile per weekday with that day's counted workouts. The day on screen is
  outlined, today's weekday name is underlined. The strip never leaves the week.
- **Season rows**: one per season the user had a goal in that week (see below).
- **Day note**, saved on change with a quiet "Saved" beside the label instead of an alert.
- **Gear line**: the gear on the day's workouts and a **Manage gear** button that opens the
  shared gear modal. It used to be reachable only from the editor's "Set gear for all", which
  only exists for workouts with two or more gear-carrying activities.

The header reads `GET /api/auth/exercise-days/:exercise_day_id/week`
(`APIGetExerciseDayWeek`, `controllers/exercise_day_week.go`), scoped to the caller like every
read here (a miss is a 404):

```json
{ "week": {
    "workouts": 3, "workout_limit": 3, "loggable": true,
    "days": [{ "id": "…", "date": "…", "workouts": 1, "loggable": true }],
    "seasons": [{ "id": "…", "name": "Spring league", "goal": 4, "workouts": 3,
                  "completion": 0.75, "sick_leave": false }] },
  "message": "Week retrieved." }
```

**Several seasons.** A user can hold goals in more than one season at once. The workout count
is the same in all of them (it follows the user's counting settings, not the season); the goal,
completion and sick leave belong to each season. So the count is shown once and each season
gets its own row: a progress bar against its goal (green at 100%, like the ring), or the
resting wording on a sick-leave week. The numbers come from `GetWeekResultForGoal`, the
function the leaderboard and weekly processing use, so the page can't disagree with them. A
season is listed when it overlaps the week and the user's goal existed by the week's end; past
weeks show that week's result.

**Which days can be logged.** `dayLoggableError` (shared with `APICreateExercise`) is the one
rule: only days in the current ISO week, never a day after today. On top of that a day holds at
most `maxWorkoutsPerDay` (3) workouts that count. The strip and the add row follow it:

| Day | Strip tile | Below the workouts |
|---|---|---|
| Current week, up to today | opens the day | "Add workout", or the limit note at three counted workouts |
| Current week, future | dimmed, can't be opened | "You can log workouts on this day once it arrives." (the note still saves) |
| Past week | opens a day that has a row; others dimmed ("Nothing logged") | no add row; a note says the week is settled |

**Day rows are created on demand.** A day without an `ExerciseDay` row has `id: null`. In the
current week its tile goes through `GET /exercise-days/week?weekDay=n`, which gets or creates
the row (the front page's edit button does the same); merely viewing the week creates nothing.
Past weeks only have rows for days that were saved, and nothing can be added there, so a
missing row there means nothing was logged.

The week is re-read after anything that can move a count: adding, deleting or restoring a
workout, and changing whether it counts.

## Activity detail (`/exercises/:id`)

The day/session builder (`web/js/exercise.js`) also renders a rich **read view** per activity.
For **cardio** (GPS/sensor) activities it surfaces the processed stream summary, in this order:

- **Metric tiles** — duration (with moving time when pauses are long), distance, average
  **pace** for foot sports or average **speed** (with its max) otherwise, elevation (with the
  descent), avg HR, cadence (steps per minute for foot sports), avg power and normalised power
  (with max and work). Each tile appears only when its channel was recorded; the peak or related
  total sits under the headline as a quiet detail line.
- **Route** — the map and the per-distance **splits** (relative-pace bars, a **negative-split**
  badge) side by side in a wide card, or the map pinned above the splits in a narrow one. Each
  split is a button: hover/focus previews it on the map, click/tap pins it.
- An **elevation profile chart**.
- An **over-time chart** for heart rate, cadence and power: one chart, a segmented switch
  between the recorded channels, a readout of avg/max(/min, normalised) and a dashed average
  line.
  - **Every channel shares one time frame.** The x-axis is pinned to `0 … maxSecs` from the
    activity's timeline (`traceTimeline`), never to where a channel's readings start or stop,
    and the average line spans the full axis — so switching tabs never rescales time. A channel
    with no readings at the start or end (cadence before the first step) shows empty edges.
  - **Missing data is a break, not a line.** Zero HR/cadence readings are dropped
    (`skipZero`), and any gap longer than `traceGapSeconds` (30 s) between kept readings
    becomes a `null` point with `spanGaps: false`, so the line breaks instead of drawing a
    ramp across time that had no readings (a padel player standing between points). Strava's
    smart recording spaces samples up to ~20 s, which stays bridged.
  - Distance sports plot **moving time** (stopped samples are dropped and the rest spread over
    `moving_time`); time-based activities plot the recorded clock.
- A **heart-rate zone** bar.
- When the summary carries an `analysis` block, an **Effort analysis** section
  (`.wv-analysis`): insight tiles for **aerobic decoupling** (with a signal-coloured verdict
  chip), **pace consistency** (seconds per km, plus the share of average pace) and **stops**, and
  an **Effort by gradient** widget (per-terrain-band share bar + avg HR on the calm→hot zone ramp).

**Time-based** (`timing`) activities imported from Strava with sensor data — padel, yoga and
the like — get the same sections minus the distance views: duration plus the sensor tiles, the
route/elevation only when GPS was recorded, the over-time chart and the HR zones. Their trace
plots the recorded clock rather than moving samples only, since standing still is part of the
session. The server leaves speed, splits and the pace-based analysis out of their summary (see
[mcp.md](mcp.md)); the shared sections live in `renderStreamSectionsHTML`.

All of it consumes one server-computed `models.StreamSummary` attached to each `OperationObject`
(`stream_summary` — the same shape the MCP `get_activity_streams` tool returns), rather than
re-deriving stats in JS. The summary math (segments, route, elevation, HR zones), the HR-zone
anchoring precedence, and the stability-over-time guarantees live in [mcp.md](mcp.md). HR-zone
anchoring is driven by optional **max / resting heart rate** settings on `/account` (including a
max **per sport**) plus an auto-maintained **observed max HR**.

### Heart-rate zones

The user picks a **zone system** on `/account` (`User.HRZoneSystem`, saved with the training
profile). Systems are defined once, server-side, in `hrZoneSystems` (`controllers/hr_zones.go`)
and served to `/account` by `GET /api/auth/hr-zone-systems` for its live preview:

| Key | Zones | Edges |
|---|---|---|
| `percent_max` (default) | Z1 Recovery → Z5 Anaerobic | 60 / 70 / 80 / 90 % of max HR |
| `reserve` | same five | same fractions of heart-rate reserve (Karvonen); needs a resting HR |
| `olympiatoppen` | I-1 Easy → I-5 Very hard | 72 / 82 / 87 / 92 % of max HR ([Olympiatoppen I-scale](https://olt-skala.nif.no/)) |

- The bottom zone has no floor: time under Olympiatoppen's nominal 55 % counts as I-1.
  I-6 – I-8 (anaerobic, sprint, strength) can't be read from heart rate and are left out.
- **NULL means never chosen**, and keeps the pre-selection rule: `reserve` when a resting HR
  is set, `percent_max` otherwise. No backfill. Saving `reserve` without a resting HR is
  rejected; a reserve system that can't be applied at read time (rest not below the max)
  falls back to `percent_max`, and `hr_zone_system` reports what was actually applied.
- Adding a system is one entry in `hrZoneSystems` plus a key constant in `models/hr_zone.go`.
  The UI reads codes/names/bounds from the server; the colour ramp (`.wv-zone-1…5`,
  `components.css`) covers five zones, so a system with a different count needs more steps.

### Per-sport max heart rate

Max HR differs by sport — cycling typically sits 5–10 bpm under running — so a user can set an
explicit max per sport (`models.UserSportMaxHeartrate`, one row per user + `Action`). It is the
**first** step of the anchoring precedence in `resolveUserHR`, reported as
`hr_max_basis: "sport_max"`:

1. the sport's own max (`sport_max`)
2. the global max (`max`)
3. the global all-time observed max (`observed_max`)
4. the age-based estimate (`age`)
5. the activity's own peak (`observed`)

- **Explicit always beats inferred.** A sport max only wins over the global max because the
  user typed it; nothing is ever inferred per sport.
- **The observed max per sport is shown, never applied.** `/account` lists the highest heart
  rate seen in each sport as a hint with a one-click pin, but it does not anchor zones on its
  own: a sport with a handful of easy sessions would get an observed max far below the real
  one, and every later activity in it would show inflated zones. It is read at request time
  from the per-operation `max_heartrate` rollup (`GetObservedMaxHeartratesByActionForUserID`
  — already spike-capped, follows deletes, no backfill or sync hook).
- **Resting HR and the zone system stay global.** Resting HR doesn't depend on the sport; a
  sport max must still sit above it (validated on save).
- **Sports** are the curated activity-type actions (`GetActivityTypeActions` — those with a
  Strava sport name), the same list as goal counting. Only those are accepted.
- **Resolved per activity.** `attachStreamSummaries` resolves the anchor per operation (a day
  can mix a run and a ride); `loadActivityStreamContext` (MCP) uses the operation's action.
  Both take the user's maxes from `loadSportMaxHeartrates`. The zone maths
  (`computeHRZones`) is untouched — the sport max only changes `hrAnchor.MaxBpm`.
- **API.** Saved with the training profile: `PUT …/training-profile` takes
  `sport_max_heartrates: [{action_id, max_heartrate}]` and **replaces the whole set** (omitted
  or null clears every sport; 100–240 bpm, one per sport). `GET /api/auth/sport-heartrates`
  (`APIGetSportHeartrates`) lists every sport with `max_heartrate` and
  `observed_max_heartrate` (each null when absent) for `/account`.
- **UI.** Under "Max heart rate per sport" in the Training section: sports with a max are
  listed (input, observed hint + pin, remove); the add select offers the rest, sports with
  heart-rate data first, prefilling the observed max. Saved by the same button as the rest of
  the profile. The zone preview stays on the global anchor.

## Ownership

Every read and write in this area is scoped to the calling user at the query, not by a
post-hoc comparison: `GetExerciseDayByIDAndUserID`, `GetAllExerciseByIDAndUserID`,
`GetOperationByIDAndUserID`, `GetOperationSetByIDAndUserID`. A miss is indistinguishable
from "no such row", so the endpoints return **404** rather than confirming that another
user's day exists. Use the scoped getter even when the handler looks read-only — the day
note is written by `POST /api/auth/exercise-days/:exercise_day_id`, and that endpoint once
used the unscoped `GetExerciseDayByID`, which let any authenticated user overwrite anyone's
note. Regression tests: `controllers/exercise_day_authz_test.go`.

## Related

- **MCP parity — done.** The MCP `list_workouts` tool runs the same filter
  (`models.ActivityFeedFilter`) against the same `GetSessionFeedForUser`, so the two searches
  agree on what a match is and both count in workouts. It exposes action-name, free-text,
  date-range, `has_distance`, metric sort and pagination, returning slim session summaries with
  their activities nested, and deferring per-set detail to `get_activity`. `get_activity` and
  `get_activity_streams` still address an activity (operation) id; `get_activity_soundtrack`
  takes either that or the workout id, which is how a session with no activities is reachable.
  See [mcp.md](mcp.md).

## Related, not yet done

- **Builder rework (`/exercises/:id`)** — the day header around the builder is done (see *Day
  page*); the builder itself still doesn't cleanly organise a multi-activity-type workout.
  The session aggregate shape built here is exactly what a better session-summary header
  should consume. Tracked in [#29](https://github.com/aunefyren/treningheten/issues/29).
