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
(`models.ComputeStreamRollup`, written on Strava sync and backfilled once by
`backfillOperationStreamRollups`) plus summed `moving_seconds` — so the list can show those
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
session-wide sum. Two or more activities get the breakdown underneath and a title naming what
is in the workout ("Cycling + Bench press", then "+N more" past two); a session with none is
titled "Workout".

The URL, the nav item and the page's own language stay **"exercises"** — only the API payload
and the internals talk about sessions. Styling follows the shared light module/inset system
(see [styleguide.md](styleguide.md)): the **session card** is the panel's inset tile and the
activities inside it are unboxed, so the panel keeps one border system.

## Activity detail (`/exercises/:id`)

The day/session builder (`web/js/exercise.js`) also renders a rich **read view** per activity.
For **cardio** (GPS/sensor) activities it surfaces the processed stream summary: per-distance
**splits** (each with a relative-pace bar and hover-to-highlight on the route map), a
**heart-rate chart** and an **elevation profile chart**, metric tiles (distance, pace, elevation
gain/descent, cadence, power, temperature), a **route map + overview**, and a **heart-rate zone**
bar — plus a **negative-split** badge when the second half was faster. When the summary carries an
`analysis` block it adds an **Effort analysis** section (`.wv-analysis`): insight tiles for
**aerobic decoupling** (with a signal-coloured verdict chip), **pace consistency** and **stops**,
and an **Effort by gradient** widget (per-terrain-band share bar + avg HR on the calm→hot zone ramp).

All of it consumes one server-computed `models.StreamSummary` attached to each `OperationObject`
(`stream_summary` — the same shape the MCP `get_activity_streams` tool returns), rather than
re-deriving stats in JS. The summary math (segments, route, elevation, HR zones), the HR-zone
anchoring precedence, and the stability-over-time guarantees live in [mcp.md](mcp.md). HR-zone
anchoring is driven by optional **max / resting heart rate** settings on `/account` plus an
auto-maintained **observed max HR**.

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

- **Builder rework (`/exercises/:id`)** — the session builder still exposes gear at the session
  level only and doesn't cleanly organise a multi-activity-type session. The session aggregate
  shape built here is exactly what a better session-summary header should consume.
  Tracked in [wip.md](wip.md).
