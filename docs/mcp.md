# MCP Server

Treningheten exposes a [Model Context Protocol](https://modelcontextprotocol.io)
server so LLM clients (Claude Desktop, etc.) can read a user's training data and,
for example, give feedback on their latest run. This is Phase 3 of the auth work
(Phase 1: OAuth 2.0 — see [oauth.md](oauth.md); Phase 2: PATs — see [pat.md](pat.md)).

## Design decisions

| Decision | Choice | Rationale |
|---|---|---|
| SDK | Official `github.com/modelcontextprotocol/go-sdk` (v1.6.x) | Spec-correct protocol handling, schema inference, Streamable HTTP handler. |
| Transport | Streamable HTTP at `POST/GET /mcp` | Remote transport that reuses the OAuth bearer auth already built. |
| Auth | OAuth access token **or** Personal Access Token (bearer) | A user can do the browser OAuth flow or paste a `trh_pat_…` token into their client. |
| v1 tools | Read-only | Serves the "summarise / give feedback" use case; a read-only token suffices. |
| Per-tool scope | Checked in the MCP layer, not by HTTP method | Every MCP call is a `POST`, so the method-based read/write rule can't apply; the endpoint requires the read scope and bypasses the generic GET-only check. |
| User scoping | Tools act only on the authenticated user's data | The principal is resolved from the token per request. |

## Endpoint & auth flow

- **Endpoint:** `/mcp` (handled by `controllers.MCPHandler`).
- An unauthenticated request gets `401` with
  `WWW-Authenticate: Bearer … resource_metadata="…/.well-known/oauth-protected-resource"`,
  which lets an MCP client **discover** the authorization server and run the
  authorization-code + PKCE flow (with dynamic client registration) built in Phase 1.
- The gin wrapper authenticates the bearer (token or PAT), requires the `api`/`api:read`
  scope, and passes the principal to the per-request MCP server via the request context.

## Tools (v1, read-only)

| Tool | Arguments | Returns |
|---|---|---|
| `whoami` | — | Profile: name, email, admin, member-since |
| `list_weights` | `limit?` | Body-weight entries, newest first |
| `get_latest_weight` | — | Most recent weight entry |
| `list_exercises` | `action?`, `limit?` | Logged activities (id, date, action, type, equipment, duration, `has_streams`, per-set distance/time/moving-time/reps/weight), newest first; `action` filters by exercise type (e.g. `Run`) |
| `get_workout` | `activity_id` | The flat detail of one activity by id — for drilling in after `list_exercises` |
| `get_workout_streams` | `activity_id`, `from_seconds?`, `to_seconds?`, `resolution?`, `max_points?` | Processed Strava sensor data for one activity (summary header + downsampled time-series). See below |
| `get_statistics` | — | Per-window totals (activity count, km distance, seconds time) over three **rolling** windows: trailing ~1 month, trailing 12 months, all-time. Counts span **all** exercise types; distance/time only count activities that record them. Plus **personal** day/week activity streaks (current + best) |
| `list_seasons` | `active_only?` | The seasons the user has joined, newest first, with the user's weekly goal / competing / sickleave-left; `active_only` limits to ongoing seasons |
| `get_season` | `season_id` | One season + the user's personal goal data (`joined=false` with no goal fields if not joined) |
| `list_achievements` | — | The achievement catalog with the user's earned state (earned, times, last); hidden achievements read `"Hidden"` until earned |
| `get_achievement` | `achievement_id` | One achievement with the user's earned state |
| `list_achievement_delegations` | `limit?`, `offset?`, `achievement_id?` | The user's achievement awards (newest first), paginated, optionally filtered to one achievement; response includes `total` |
| `get_achievement_delegation` | `delegation_id` | One of the user's awards (ownership-checked) |

Each activity carries a stable `id` (the operation id) used to address `get_workout` / `get_workout_streams`, and a `has_streams` flag so the model knows whether stream detail is available before asking for it.

Durations are exposed in **seconds** for LLM friendliness. Note that the underlying
`*time.Duration` fields store a raw seconds count, not real nanosecond durations, so
the flattener casts the value directly (`int64(*d)`) — calling `.Seconds()` would
divide by 1e9 and wrongly yield `0`.

### Data assembly

`mcp_data.go` flattens the **enriched `*Object` layer** (via
`ConvertExerciseDaysToExerciseDayObjects`), not raw GORM models. This reuses the
canonical action resolution, exercise-time fallback and Strava roll-up the rest of
the app relies on, rather than re-walking and re-joining the tree by hand. Distance
totals in `get_statistics` are normalized to kilometres per the operation's
`distance_unit` (unknown units are treated as km, the model default).

### Workout streams (`get_workout_streams`)

Strava-imported activities carry second-by-second sensor streams (`time`, `latlng`,
`altitude`, `heartrate`, `cadence`, `watts`, `temp`, `velocity_smooth`), stored
locally on `OperationSet.StravaStreams` — so no Strava API call happens at read time.
These exist **only** for GPS/sensor activities; strength and manual logs have none.

Raw streams are too large and too low-level to hand an LLM directly (a long ride is
thousands of points across ~8 channels, easily tens of thousands of tokens), so the
tool processes them into:

1. **A whole-workout summary header** — `heartrate_bpm`/`cadence_rpm`/`temperature_c`
   (avg/min/max), `speed` (avg & max km/h + avg pace min/km), `elevation` (gain /
   min / max m), `power` (avg/max W + integrated work kJ), plus `available` and
   `has_gps`. The summary always describes the entire workout.
2. **A `series`** — raw samples restricted to `[from_seconds, to_seconds]` and
   downsampled to fit `max_points`. Arrays are time-aligned to `t_seconds`; only
   recorded channels are populated.

**Default (no knobs):** the whole workout, auto-downsampled to ~2000 points, so a
single call is always context-safe. **Zoom:** to read a moment at full fidelity,
re-call with a narrow `from_seconds`/`to_seconds` window and `resolution=1`. The
response reports `sampled_every_seconds`, `returned_points` and
`total_points_in_window` so the model knows how much detail it actually got.
`max_points` is capped at 5000 to protect the context window.

Energy/elevation are integrated using the time deltas, so totals stay correct even
when the series is not 1 Hz. Durations follow the seconds-as-int convention (see
above).

### Example use cases

> "Give me feedback on my newest run."

The client calls `list_exercises(action:"Run", limit:1)`, receives the activity (with
its `id` and `has_streams`), then — if `has_streams` is true — `get_workout_streams(id)`
for HR/pace/elevation trends, and composes feedback.

> "Did I fade on the final climb?"

After the overview call, the model spots the climb in the coarse series, then
re-calls `get_workout_streams(id, from_seconds=…, to_seconds=…, resolution=1)` to read
that segment's heart rate and speed at full resolution.

### Seasons, achievements & delegations (personal)

All of these are scoped to the authenticated user:

> Background on these mechanics lives in [seasons-and-goals.md](seasons-and-goals.md)
> and [streaks.md](streaks.md).

- **Seasons** are global, but the user participates through a **Goal**. `list_seasons`
  returns only the seasons the user has joined (their goals), each carrying the user's
  weekly target, competing flag and remaining sick-leave; `get_season` works for any
  season id but only fills the goal fields when the user has joined it (`joined`
  reflects this). v1 is deliberately lean — no within-season streak/standing (that
  needs the week-result computation in `season.go` and is a follow-up).
- **Achievements** combine the catalog with the user's earned state. Hidden
  achievements keep their description as `"Hidden"` until the user earns them (mirrors
  the web's `ConvertAchievement…UserObject` behaviour). See
  [data-conventions.md](data-conventions.md) for the `Convert*Object` read layer.
- **Achievement delegations** are the individual awards. Because a user can accrue
  many, `list_achievement_delegations` is paginated (`limit`/`offset`) and filterable
  by `achievement_id`, and reports `total`. `get_achievement_delegation` is
  ownership-checked (`GetAchievementDelegationByIDAndUserID`) — another user's award id
  returns "not found".

## Connecting a client

Point an MCP client at `https://<your-host>/mcp`. Either:
- let it run the OAuth flow (it will discover the auth server from the 401), or
- configure it with a Personal Access Token (`Authorization: Bearer trh_pat_…`),
  created from the **Developer access tokens** section on the account page. A
  read-only PAT is enough.

## Key code locations

- `controllers/mcp.go` — server construction, tool registration, gin handler + auth.
- `controllers/mcp_data.go` — assembly of the operation-centric model into flat DTOs.
- `controllers/mcp_streams.go` — processing of stored Strava streams (summary + downsampled series).
- `controllers/mcp_engagement.go` — assembly of seasons, achievements and achievement delegations (all user-scoped).
- `models/mcp.go` — MCP DTOs (`MCPProfile`, `MCPWeight`, `MCPActivity`, `MCPStatistics`, `MCPWorkoutStreams`, `MCPSeason`, `MCPAchievement`, `MCPAchievementDelegation`).
- `middlewares/auth.go` — `Authenticate` (shared token/PAT validation) + `BearerChallenge`.
- `main.go` — `router.Any("/mcp", controllers.MCPHandler())`.

## Known limitations / next steps

- Read-only only; write tools (`log_exercise`, `add_weight`) are a follow-up and
  would require the `api:write` scope and per-tool write checks.
- `get_statistics` distance is normalized to a single km total. If users start
  logging in mixed units in earnest, a per-unit breakdown may read better than a
  lossy normalization.
- `get_statistics` reports counts, distance/time totals and **personal** day/week
  streaks. It deliberately does **not** include *season* (within-season, weekly-goal)
  streaks, nor the per-action tops/averages from the web statistics endpoint
  (`APIGetUserStatistics`). The personal-streak logic is shared: both this tool and
  the web endpoint call `computePersonalStreaks` (`controllers/streaks.go`), so they
  cannot drift. Season streaks (computed in `season.go` via `GetWeekResultForGoal`)
  are a natural follow-up if/when a seasons tool lands.
- Seasons, achievements and achievement-delegation tools are read-only and personal.
  Within-season **standing/streak** and the **leaderboard** are not exposed yet (they
  need `season.go`'s week-result computation); season write/join actions are out of
  scope too.
- Granular per-resource scopes are still not implemented (coarse read/write/admin).
