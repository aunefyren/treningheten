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
`has_strava` flag. A companion grouped query fills `session_activity_count` (the true number of
activities in each returned session, independent of the current filter) so a browse card can
honestly say "2 activities".

Each item is a slim `models.ActivityFeedItem` — **no `strava_streams`** (too heavy for a list;
HR/GPS detail is deferred to the builder/detail view). The response is shaped so precomputed
Operation rollup columns could back it later **without changing the JSON**.

## Frontend

`web/js/exercises.js` is a filter/search bar + infinite-scroll timeline against
`api_url + "auth/activities"`. It groups adjacent same-session activities under day headers in
browse mode and shows a flat ranked list in find mode. Each card links to `/exercises/:dayID`
(the builder). Styling follows the dark instrument-panel system shared with stats/gear.

## Related, not yet done

- **MCP parity** — `list_exercises` currently filters only by action type + limit, not the
  richer feed search (date range, note text, metric sort). Tracked in [wip.md](wip.md).
- **Builder rework (`/exercises/:id`)** — the session builder still exposes gear at the session
  level only and doesn't cleanly organise a multi-activity-type session. The per-activity
  aggregate shape built here is exactly what a better session-summary header should consume.
  Tracked in [wip.md](wip.md).
