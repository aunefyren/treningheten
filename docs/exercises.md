# The `/exercises` activity timeline

`/exercises` is a searchable, sortable **activity timeline**: it floats each activity's key
metrics inline so you can browse recent sessions or hunt a specific one ("my longest run",
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

Search targets ("longest run", "oldest run") live at the **Operation** grain, while browsing
wants **session** grouping. So the feed is **activity-grained** (one item per Operation) and
carries session/day grouping metadata, letting the client render either view from one payload.

## Two modes, one endpoint

The client picks the mode; the endpoint is the same:

- **Browse** — no metric sort or type filter active. Cards are grouped by day and session
  (adjacent same-session activities collapse under a day header). This is the default landing
  view, newest first.
- **Find** — a metric sort or an activity-type filter is active. The list goes flat and
  ranked, with the sorted metric shown prominently.

## Backend

`GET /auth/activities` → `controllers.APIGetActivityFeed`. Query parsing/validation lives in
`parseActivityFeedFilter` (a bad `action_id`, date, `sort`, or `order` is a `400`; `limit` is
clamped to `[1, 100]`, a bad `offset` falls back to `0`). Dates accept either a full RFC3339
timestamp or a bare `YYYY-MM-DD` (read as midnight).

Filters: `action_id`, `start`/`end` (date range), `q` (case-insensitive substring over the
operation / session / day notes **and** the action name), `has_distance`. Sort:
`date | distance | duration | weight | reps` with `order` = `asc | desc`. Pagination:
`limit` / `offset`. Response: `{ activities: [...], total, has_more, message }`.

The aggregation is **query-time** (`database.GetActivityFeedForUser`): it walks the enabled
chain `operations → exercises → exercise_days → users` (same joins as `GetOperationsByUserID`),
LEFT JOINs `operation_sets`, and per operation returns `SUM(distance)`, `SUM(time)` (seconds —
repo convention), `SUM(repetitions)`, `MAX(weight)` as top weight, `COUNT(sets)`, and a
`has_strava` flag. It also carries the session's `counts_toward_goal` (a session-level flag, so
every activity of the session shares it) so the feed can flag a logged-but-excluded session. A
companion grouped query fills `session_activity_count` (the true number of activities in each
returned session, independent of the current filter) so a browse card can honestly say
"2 activities".

Each item is a slim `models.ActivityFeedItem` — **no `strava_streams`** (too heavy for a list).
The item does carry a handful of **precomputed stream scalars** (`avg_heartrate`, `max_heartrate`,
`avg_cadence`, `temp_c`, `elevation_gain_m`) read directly from rollup columns on the `Operation`
(`models.ComputeStreamRollup`, written on Strava sync and backfilled once by
`backfillOperationStreamRollups`) plus summed `moving_seconds` — so the list can show those
numbers without ever loading or parsing a stream blob. The full per-second HR/GPS detail is still
deferred to the builder/detail view. This is the "precomputed rollup columns without changing the
JSON" the shape was designed for; the MCP `list_activities` search consumes the same fields.

## Frontend

`web/js/exercises.js` is a filter/search bar + infinite-scroll timeline against
`api_url + "auth/activities"`. It groups adjacent same-session activities under day headers in
browse mode and shows a flat ranked list in find mode. Each card links to `/exercises/:dayID`
(the builder) and shows a muted "Doesn't count" badge when `counts_toward_goal` is false. Styling
follows the shared light module/inset system (see [styleguide.md](styleguide.md)).

## Activity detail (`/exercises/:id`)

The day/session builder (`web/js/exercise.js`) also renders a rich **read view** per activity.
For **cardio** (GPS/sensor) activities it surfaces the processed stream summary: per-distance
**splits** (each with a relative-pace bar and hover-to-highlight on the route map), a
**heart-rate chart** and an **elevation profile chart**, metric tiles (distance, pace, elevation
gain/descent, cadence, power, temperature), a **route map + overview**, and a **heart-rate zone**
bar — plus a **negative-split** badge when the second half was faster.

All of it consumes one server-computed `models.StreamSummary` attached to each `OperationObject`
(`stream_summary` — the same shape the MCP `get_activity_streams` tool returns), rather than
re-deriving stats in JS. The summary math (segments, route, elevation, HR zones), the HR-zone
anchoring precedence, and the stability-over-time guarantees live in [mcp.md](mcp.md). HR-zone
anchoring is driven by optional **max / resting heart rate** settings on `/account` plus an
auto-maintained **observed max HR**.

## Related

- **MCP parity — done.** The MCP `list_activities` tool is now backed by the same query-time
  aggregation (`GetActivityFeedForUser`): it exposes action-name, free-text, date-range,
  `has_distance`, metric sort and pagination, returning slim summaries (with `counts_toward_goal`)
  and deferring per-set detail to `get_activity`. See [mcp.md](mcp.md).

## Related, not yet done

- **Builder rework (`/exercises/:id`)** — the session builder still exposes gear at the session
  level only and doesn't cleanly organise a multi-activity-type session. The per-activity
  aggregate shape built here is exactly what a better session-summary header should consume.
  Tracked in [wip.md](wip.md).
