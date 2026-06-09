# Strava integration

Treningheten can import a user's Strava activities and turn them into logged
exercises, so workouts recorded on Strava count toward their goals and streaks
without manual entry. The whole integration is gated by the `StravaEnabled` config
flag; when it is off, every Strava route and the sync job are disabled.

Code: `controllers/strava.go`, `models/strava.go`. Activities map onto the normal
`ExerciseDay → Exercise → Operation → OperationSet` model — see
[seasons-and-goals.md](seasons-and-goals.md) and
[data-conventions.md](data-conventions.md) for that model and its conventions.

## Configuration

Server-wide settings (in `config/config.json`, `models/config.go`):

| Field | Purpose |
|---|---|
| `StravaEnabled` | Master switch for the whole integration |
| `StravaClientID` | The Strava API application's client id |
| `StravaClientSecret` | The Strava API application's client secret |
| `StravaRedirectURI` | Where Strava sends the user back after they authorize |
| `StravaTokenKey` | Base64 AES-256 key used to encrypt stored refresh tokens; auto-generated on first run |

`StravaClientID`, `StravaRedirectURI` and `StravaEnabled` are surfaced to the frontend
(login/config payload, injected via the `stravaRedirectURI` template variable) so the
browser can build the authorize URL.

Per-user state lives on the `User` model (`models/user.go`):

| Field | Purpose |
|---|---|
| `StravaCode` | The connection credential — see [token lifecycle](#token-lifecycle) for the `c:` / `r:` scheme |
| `StravaID` | The athlete's numeric Strava id (filled in on first sync) |
| `StravaIgnoreWalks` | When true, **skip** activities of sport type `walk` on import. (JSON/DB name stays `strava_walks`; the Go field is named for what it does to avoid the inverted-meaning trap.) |
| `StravaPublic` | Whether the user's Strava-derived activities are shown on their profile |

## Connecting an account (OAuth login)

Strava uses standard OAuth 2.0 authorization-code. The flow:

1. **Authorize** — the account page builds a Strava authorize URL
   (`web/js/account.js`):

   ```
   https://www.strava.com/oauth/authorize
     ?client_id=<StravaClientID>
     &response_type=code
     &redirect_uri=<StravaRedirectURI>
     &approval_prompt=force
     &scope=activity:read_all
   ```

   The user approves on Strava (`activity:read_all` = read all their activities).

2. **Callback** — Strava redirects to the redirect URI with `?code=…`. The callback
   page (`web/js/oauth.js`) reads that code and `POST`s it to
   `POST /api/auth/users/:user_id/strava` (`APISetStravaCode`).

3. **Store + first sync** — the handler stores the code as `StravaCode = "c:" + code`
   and immediately runs a sync for the current week (`StravaSyncWeekForUser`).

No Strava access token is ever persisted — only the refresh token is kept (after the
first exchange), and only ever inside the `StravaCode` field.

### Token lifecycle

`StravaCode` is a single string that encodes **what kind** of credential it holds via a
one-letter prefix, dispatched in `StravaGetAuthorizationForUser`:

- **`c:<authorization_code>`** — a fresh, one-time authorization code from the OAuth
  callback. On the next sync it is exchanged via `StravaAuthorize`
  (`grant_type=authorization_code`) for an access token + refresh token, and the field
  is **rewritten** to `r:<encrypted_refresh_token>`.
- **`r:<encrypted_refresh_token>`** — the long-lived refresh token. Each sync decrypts
  it, exchanges it via `StravaReauthorize` (`grant_type=refresh_token`) for a fresh
  access token, and re-stores the (possibly rotated) refresh token.

Both exchanges `POST` to `https://www.strava.com/oauth/token` (note: **not** under
`/api/v3`) with the client id/secret. The access token returned is used in-memory for
that sync only.

**Encryption at rest:** the refresh token is encrypted with AES-256-GCM
(`utilities.EncryptString`, key `StravaTokenKey`) before being stored, so the database
never holds a usable refresh token in plaintext. Values written before encryption was
introduced are plaintext; they fail to decrypt, are used as-is once, and are
re-encrypted on the next successful exchange — so no migration is needed.

**Failure handling:** the token exchanges return an error on any non-200 response, and
`StravaGetAuthorizationForUser` **never overwrites `StravaCode` on failure** and never
stores an empty token. A transient error or a revoked token therefore surfaces as a
failed sync rather than silently bricking the connection.

## Retrieval

All Strava reads go through helpers in `controllers/strava.go`:

| Helper | Endpoint | Use |
|---|---|---|
| `StravaGetActivities(token, before, after)` | `GET /athlete/activities` | A page (up to 30) of the athlete's activities in a UNIX-time window |
| `StravaGetActivity(token, id)` | `GET /activities/{id}` | Full detail for one activity |
| `StravaGetActivityStreams(token, id)` | `GET /activities/{id}/streams` | The per-second sensor streams (`time,latlng,altitude,heartrate,cadence,watts,temp,velocity_smooth`, `key_by_type=true`) |

### Rate limiting

Strava's read limit is windowed at **~100 requests per 15 minutes**. A flat
per-minute cap can still exceed that window under load, so `stravaWait()` enforces a
**sliding 15-minute window** (`stravaRateWindow` / `stravaRateLimit`, currently 90 per
15 min): it records request timestamps, drops ones that have aged out, and blocks
until a slot frees when the window is full. Each of the three read helpers calls it
before issuing its request; the token-exchange calls are not throttled. (The daily cap
is not separately tracked.)

## Scheduling

Three ways a sync runs (all no-ops unless `StravaEnabled`):

| Trigger | Entry point | Scope |
|---|---|---|
| **Hourly cron** (`0 0 * * * *`, registered in `main.go`) | `StravaSyncWeekForAllUsers` → `StravaSyncWeekForUser` | Every connected user, current week |
| **On connect** + **manual** (`POST /api/auth/users/:user_id/strava-sync`, `APISyncStravaForUser`, optional `pointInTime` query) | `StravaSyncWeekForUser` | The calling user, the chosen week |
| **Admin bulk refresh** (`POST /api/admin/strava/sync-activities-for-users`, `APISyncStravaActivitiesForUsers`) | `SyncStravaActivitiesForUsers` | Re-fetches full detail for already-imported activities |

`StravaSyncWeekForAllUsers` pulls connected users via `database.GetStravaUsers`
(enabled users whose `strava_code` is non-null).

The **bulk refresh** walks each user's existing Strava-linked operation sets and
re-pulls each activity's full detail, but **skips any set whose `StravaDataRetrievedAt`
is less than 7 days old** — so it backfills/refreshes older imports without hammering
the API for recent ones.

### A weekly sync (`StravaSyncWeekForUser`)

1. Resolve the week window with `utilities.FindEarlierMonday` / `FindNextSunday`.
2. Get an access token (authorize/reauthorize per the token lifecycle).
3. `StravaGetActivities(token, before=sunday, after=monday)` — the activities in that
   week (Strava's `before`/`after` are exclusive UNIX bounds).
4. Award the "connected Strava" achievement (`fb4f6c1f-…`), async and best-effort.
5. Import each activity via `StravaSyncActivityForUser`.
6. Kick off an async Ollama front-page cache refresh for the user.

## Data conversion

`StravaSyncActivityForUser` + `StravaSyncOperationForActivity` map one Strava activity
onto the platform model. The whole path is **idempotent**: it looks up existing rows by
Strava activity id and updates them rather than duplicating, so re-syncing a week is
safe.

**Filtering & identity**
- If `StravaWalks` is enabled and the activity's sport type is `walk`, it is skipped.
- The user's `StravaID` is backfilled from `activity.Athlete.ID` on first import.

**Activity → ExerciseDay → Exercise**
- Find the exercise already linked to this Strava id
  (`GetExerciseForUserWithStravaID`). If none, find-or-create the `ExerciseDay` for the
  activity's **local** start date (`StartDateLocal`), then create an `Exercise`. The
  day is stamped at **midnight UTC** (not the server's local zone) so it is
  deterministic and matches the date-string day lookups regardless of where the server
  runs.
- The exercise is set `Enabled`/`IsOn = true`, `Note = activity.Name`,
  `Time = activity.StartDate`, `Duration = ElapsedTime`.

**Activity → Operation → OperationSet**
- **Action mapping:** `activity.SportType` is matched to an `Action` via
  `GetActionByStravaName` (case-insensitive match on `Action.StravaName`); unknown
  sport types fall back to the generic **`Workout`** action. The operation's `Type`
  comes from the matched action.
- The **Operation** (found/created by Strava id + user + exercise) gets `ActionID`,
  `Duration = ElapsedTime`, `Note = activity.Name`, the derived `Tags`
  (see [tags](#activity-tags)), and — when detail was fetched this run —
  `Description`.
- The **OperationSet** (found/created by Strava id + user + operation) is where the
  Strava data lands: `StravaID`, `MovingTime`, `Time = ElapsedTime`,
  `StravaDataRetrievedAt = now`, `StravaDetailRetrievedAt` (when detail was fetched),
  the sensor `StravaStreams`, and `Distance = activity.Distance / 1000` (Strava
  reports metres; stored as km).

**Session aggregation**
- `SyncStravaOperationsToExerciseSession` then recomputes the parent exercise's
  `Duration` as the **sum** of its operations' durations and its `Note` as the
  operations' notes joined with `" + "`. This is what lets a single exercise session
  hold several Strava activities (see [combine/divide](#user-facing-management)).

> **Unit caveat:** `ElapsedTime`/`MovingTime` are Strava **seconds**, cast straight
> into `time.Duration(…)` — i.e. the stored value is a seconds count, not real
> nanoseconds. Always read these with `int64(*d)`, never `.Seconds()`. See
> [data-conventions.md](data-conventions.md#durations-are-stored-as-seconds-not-nanoseconds).

### Activity tags

Treningheten owns an 8-value tag vocabulary (`models/tag.go`): **Race, Long Run,
Workout, Commute, For a Cause, Recovery, With Pet, With Kid**. Tags live on the
`Operation` as a JSON `TagList` column and are surfaced per-activity in the UI (not
folded into the aggregated session note).

Strava's **public** API only exposes a subset of its in-app tags, so only the first
four are auto-derived (`controllers/strava.go` `stravaDerivedTags`):

| Tag | Strava source |
|---|---|
| Commute | `commute == true` |
| Race | `workout_type` 1 (run) / 11 (ride) |
| Long Run | `workout_type` 2 |
| Workout | `workout_type` 3 (run) / 12 (ride) |

Both `commute` and `workout_type` ride along in the **list** payload, so deriving
these tags costs **no extra API call**. The other four tags (For a Cause, Recovery,
With Pet, With Kid) exist only in Strava's *internal* app — the public v3 API returns
no field for them — so they can only ever be set **manually** in Treningheten
(operation edit view → tag chips → `PUT /api/auth/operations/:id`).

**Merge on re-sync** (`mergeStravaTags`): each sync recomputes the four
Strava-managed tags from the activity and **preserves** any user-managed tags already
present. Strava is the source of truth for its four, so if a user removes e.g.
"Commute" but Strava still reports it, the next sync re-adds it. User-only tags are
never touched by sync.

The manual-edit path (`APIUpdateOperation`) validates incoming tags against the
vocabulary, de-duplicates, and — because `OperationUpdateRequest.Tags` is a pointer —
leaves stored tags **untouched** when the field is omitted (so ordinary action/unit
edits don't wipe tags).

### Description

`activity.description` is **only** returned by the detailed endpoint
(`GET /activities/{id}`), not the list sync. To avoid an extra rate-limited call on
every hourly run, `StravaSyncActivityForUser` takes a `hasDetail` flag:

- **Bulk refresh** and **single-set sync** already fetch the detailed activity, so
  they pass `hasDetail = true` and use its description directly.
- The **weekly list sync** passes `hasDetail = false` and fetches detail **lazily**,
  guarded by `OperationSet.StravaDetailRetrievedAt`: only when it is missing or older
  than `stravaDetailRefreshInterval` (7 days). A list-only sync that doesn't fetch
  detail never overwrites an existing description.

The description is stored on `Operation.Description` (parallel to `Note`) and shown
per-activity in the summary card — it is deliberately **not** rolled into the
`Exercise.Note` aggregation (which joins operation notes with `" + "`).

### Sensor streams

The per-second streams fetched at import are stored on `OperationSet.StravaStreams`
(`StravaStreamsJSON`, a GORM JSON type). They are processed for the frontend charts and
for the MCP `get_workout_streams` tool — their structure and processing are documented
in [mcp.md](mcp.md#workout-streams-get_workout_streams).

## User-facing management

Beyond connect/sync, a few endpoints let users curate imported activities:

- `POST /api/auth/exercises/strava-combine` (`APIStravaCombine`) — merge several Strava
  activities into one exercise session.
- `POST /api/auth/exercises/:exercise_id/strava-divide` (`APIStravaDivide`) — split a
  combined session back apart.
- `POST /api/auth/operation-sets/:operation_set_id/strava-sync`
  (`APISyncStravaOperationSet`) — refresh a single set's data from Strava.

> The account page warns users to log a session to **either** Strava or Treningheten,
> not both — there is no cross-source de-duplication beyond the Strava-id idempotency,
> so a manually logged session plus its Strava import would double-count.

## Key code locations

- `controllers/strava.go` — auth/token exchange, retrieval helpers, rate limiter, sync
  orchestration, conversion, tag derivation/merge (`stravaDerivedTags`,
  `mergeStravaTags`), the description detail-fetch guard.
- `models/strava.go` — Strava API response shapes (incl. `WorkoutType`,
  `Description`), `StravaActivityStreams`, `StravaStreamsJSON`.
- `models/tag.go` — the 8-tag vocabulary, the Strava-managed subset, and the
  `TagList` JSON column type.
- `models/user.go` — `StravaCode`, `StravaID`, `StravaWalks`, `StravaPublic`.
- `models/config.go` / `files/config.go` — the `Strava*` config fields.
- `database/user.go` (`GetStravaUsers`), `database/operation.go`
  (`GetActionByStravaName`, the Strava-id lookups).
- `main.go` — the hourly cron registration and route wiring.

## Related

- [data-conventions.md](data-conventions.md) — durations-as-seconds, units, the `Convert*Object` read layer
- [mcp.md](mcp.md) — workout streams exposed to LLM clients
- [seasons-and-goals.md](seasons-and-goals.md) — how imported exercises feed goals and streaks
