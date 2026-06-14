# Admin panel statistics

The admin panel (`/admin`) shows a **Statistics** module with aggregate,
non-personal usage numbers next to the existing **Server info** module.

## Definition of "active user"

Every user metric is scoped to **enabled users** (`users.enabled = 1`). There is
no last-login / last-activity tracking in the schema, so "active" means
"enabled" — not "recently seen". This keeps the metrics free of new columns or
migrations. If a more behavioural definition is wanted later, it would require a
`last_seen` column (touched in `middlewares/auth.go`) or deriving activity from
the latest `Exercise.Time` per user.

## Metrics

| Metric | Definition |
|---|---|
| **Total users** | Count of enabled users. The denominator for the percentages below. |
| **In a season now** | Distinct enabled users with an enabled `Goal` in an enabled `Season` whose `start ≤ now ≤ end`. |
| **Notifications enabled** | Distinct enabled users with at least one row in `Subscription`. |
| **Strava connected** | Enabled users with a non-empty `StravaCode`. |
| **Total achievements** | Count of enabled achievements. |
| **Avg. completion** | Average fraction of all achievements unlocked across all active users: `distinct (user, achievement) delegations / (total users × total achievements)`. |

All percentages are rounded to one decimal and guard against a zero
denominator (returning `0` when there are no users / achievements).

## Implementation

- `models/stats.go` — `AdminStatsReply` (the API DTO, raw counts + percentages)
  and `AdminStatsCounts` (the raw counts gathered from the DB).
- `database/stats.go` — `GetAdminStatsCounts(now)` runs the count queries
  (scoped to enabled rows, soft-delete-aware via `Instance.Model`).
- `controllers/stats.go` — `APIGetAdminStats` computes percentages via
  `percentageOf` and responds with `{ "stats": ... }`.
- Route: `GET /api/admin/stats`, registered in `main.go` under the
  `middlewares.Auth(true)` admin group.
- Frontend: `web/js/admin.js` — `get_admin_stats()` fetches and
  `place_admin_stats()` renders the module, reusing the `server_info_row` /
  `info-badge` helpers from the Server info module.
