# Hevy integration

Imports workouts from [Hevy](https://www.hevyapp.com/) (strength-training tracker).
Gated by the `HevyEnabled` config flag. **Status: in progress** — account connect &
validation and the exercise-mapping building blocks are implemented; the actual workout
import/sync is not yet wired (see `docs/wip.md`).

## Authentication — per-user API key (no OAuth)

Unlike Strava, Hevy has **no app-based OAuth flow**. Each user generates a personal
API key (a UUID) in the Hevy app under Settings; this is **only available to Hevy PRO
subscribers**. The key is sent as the `api-key` request header on every call. There is
no refresh/redirect lifecycle.

- Stored as `User.HevyAPIKey` (`*string`), **encrypted at rest** with AES-256-GCM using
  `ConfigStruct.HevyTokenKey` (auto-generated into `config/config.json` on first run,
  mirroring `StravaTokenKey`). The field is `json:"-"` — the key is **never** serialized
  back to clients.
- On connect, the user's public profile URL from `/user/info` is stored as
  `User.HevyProfileURL` (cleared on disconnect) so a Hevy link can be shown on the profile,
  gated by the `User.HevyPublic` visibility toggle (default true; mirrors `StravaPublic`).
- `User.HevyConnected` (`bool`, `gorm:"-"`) is a derived flag set only on the user's own
  `GET /api/auth/users/:id` response so the frontend can show connect vs. disconnect
  state without ever seeing the key.
- Encryption helpers: `encryptHevyAPIKey` / `decryptHevyAPIKey` in `controllers/hevy.go`.

## Config

- `HevyEnabled bool` — feature flag (config file / env only, like `StravaEnabled`; no CLI
  flag). Exposed to the frontend as `hevy_enabled` on the `auth/tokens/validate` response
  and as `hevyEnabled` in the HTML/JS template data, and reported by the admin server-info
  endpoint (`ServerInfoHevy`).
- `HevyTokenKey string` — AES key for encrypting stored API keys; auto-generated.

## Endpoints (`controllers/hevy.go`)

- `POST /api/auth/users/:user_id/hevy` — `APISetHevyAPIKey`. Validates the supplied key
  against Hevy (`GET /v1/user/info`) before storing it encrypted. Rejects when
  `HevyEnabled` is false or the key is empty/invalid.
- `DELETE /api/auth/users/:user_id/hevy` — `APIDeleteHevyAPIKey`. Clears the stored key
  (disconnect).
- `POST /api/auth/users/:user_id/hevy-sync` — `APISyncHevyForUser`. On-demand sync for the
  **authenticated user only** (resolved from the token, not the URL param). Runs a backfill
  if no baseline exists yet, otherwise an incremental events sync; async, returns
  immediately.

Managed from the `/account` page "Hevy" section (`web/js/account.js`:
`renderHevySection`, `setHevy`, `disconnectHevy`, `syncHevy`, `refreshHevySection`).

## Hevy API reference

Base URL `https://api.hevyapp.com/v1`, OpenAPI at <https://api.hevyapp.com/docs/>.
Auth header `api-key: <uuid>`. Key endpoints relevant to the integration:

- `GET /user/info` → `{ data: { id, name, url } }` — used to validate a key.
- `GET /workouts?page&pageSize` (pageSize max 10) — full list (initial backfill, planned).
- `GET /workouts/events?since=<ISO>&page&pageSize` — incremental `updated`/`deleted`
  events; the planned primary sync mechanism.
- `GET /exercise_templates?page&pageSize` (pageSize max 100) — exercise catalog; each
  template has an `is_custom` flag distinguishing Hevy's official catalog from a user's
  private custom exercises.

## Exercise → Action mapping (building blocks, implemented)

The Hevy model is finer-grained than Strava: a workout has many exercises, each with its
own template. The mapping layer lives in `controllers/hevy.go`:

- `hevyFetchExerciseTemplates(apiKey)` — pages through `GET /exercise_templates`
  (pageSize 100) into a `template_id → HevyExerciseTemplate` map. The workout payload only
  carries the template id + title, so this map supplies `is_custom`, `type`, and
  `primary_muscle_group`.
- `models.HevyActionType(hevyType)` — maps a Hevy template type onto Treningheten's
  `lifting`/`timing`/`moving` vocabulary (`duration`→timing; `*distance*`→moving; the rest,
  being rep/weight based, →lifting). Lives in `models` (not `controllers`) so the live
  import and the catalog seeder produce identical Actions; the thin `hevyTypeToActionType`
  wrapper in `controllers/hevy.go` is unit-tested in `controllers/hevy_test.go`.
- `models.HevyExerciseTemplate.ToAction()` — the single template→`Action` builder
  (Name/NorwegianName = title, Type from the mapping, BodyPart = primary muscle group,
  `HevyTemplateID` set, `HasLogo=false`). Used by both the seeder and the live path.
- `getOrCreateHevyAction(template)` — for `is_custom=false`, resolves a global `Action` by
  `Action.HevyTemplateID` (`database.GetActionByHevyTemplateID`) and **auto-creates** one
  via `ToAction()` (random UUID) on miss. For `is_custom=true` it returns `nil` — custom
  exercises are private and must not enter the shared vocabulary. With the catalog seeded
  (below) this miss path is now mainly a safety net for templates Hevy adds after the
  embedded snapshot was exported.

Model keys: `Exercise.HevyWorkoutID` (idempotency for the workout→`Exercise` import) and
`Action.HevyTemplateID` (sole dedup authority for catalog actions). Both are `AutoMigrate`d.

## Built-in catalog seed (implemented)

The full official Hevy exercise catalog (~433 templates) is **built into the binary** —
the Strava-catalog equivalent for Hevy — rather than being discovered lazily as users
sync. `database/data/hevy_templates.json` (exported from the Hevy API; refresh source in
`scripts/hevyseed/`) is `//go:embed`ed, and `database.SeedHevyActions()` (in
`database/seed_hevy.go`) runs at startup when `HevyEnabled`, right after `SeedActions()`.

- **Dedup authority is `HevyTemplateID`**, identical to the live path, so the seed is
  idempotent and never duplicates an Action a sync already created.
- **Overlaps merge into curated Actions.** `hevyOverlapOverrides` maps a Hevy template id
  onto an existing curated Action UUID (Running→Run, Cycling→Bicycling, Swimming→Swim,
  Walking, Hiking→Hike, Rowing Machine→Rowing, Elliptical Trainer→Elliptical, Yoga,
  Pilates, Plank). For these the seeder sets `HevyTemplateID` on the curated Action (only
  if empty) instead of creating a new one — so no duplicate cardio/bodyweight Actions.
  Ambiguous cases (Hevy's single generic *Skiing* vs the curated Alpine/Cross-Country
  split; *Boxing*/*Climbing* with no curated equivalent) are intentionally left to seed
  standalone. Since `Action.HevyTemplateID` is a single value, each curated Action absorbs
  exactly **one** Hevy template; variant templates (e.g. a treadmill *Running*) seed as
  their own Action.
- **New Actions get deterministic UUIDs** (`uuid.NewSHA1(hevyActionNamespace, templateID)`)
  so fresh installs are stable and re-seeds never duplicate.

Data integrity (override ids exist in the catalog, deterministic ids are unique and don't
collide with override targets) is covered by `database/seed_hevy_test.go`.

## Workout import (implemented)

`HevySyncWorkoutForUser(user, workout, templates)` imports one workout into the exercise
tree, idempotent by `Exercise.HevyWorkoutID`:

- Workout → `Exercise` (on the exercise day for `start_time`, stamped midnight UTC like
  Strava). `Note` = workout title; `Duration` = end−start (seconds count). `IsOn=true` so
  it counts toward season goals.
- Each Hevy exercise → `Operation`. Official catalog → `ActionID` set + `Type` from the
  Action; the user's per-exercise note goes in `Operation.Note`. Custom exercise →
  `ActionID=nil`, title in `Operation.Note` (the frontend uses note as the title fallback
  and an emoji icon when there's no Action), note in `Description`.
- Each set → `OperationSet`: `Weight`=`weight_kg`, `Repetitions`=`reps`,
  `Distance`=`distance_meters`/1000 (km), `Time`=`duration_seconds` (seconds count).
- Re-import rebuilds children: existing operations/sets are soft-disabled
  (`hevyDisableExerciseChildren`) and recreated, since Hevy exercises/sets have no stable
  ids to diff against.

`HevyBackfillForUser(user)` fetches the template catalog once, then pages `GET /workouts`
(pageSize 10), importing each workout. It runs **asynchronously** after a successful
connect so the response stays fast; per-workout errors are logged and skipped. On success
it records `User.HevyLastSync` (stamped at backfill *start*, so anything changed mid-backfill
is re-caught by the next events poll), which hands off to the incremental sync.

Both the backfill and the incremental sync award the "Influencer" achievement
(`fb4f6c1f-…`) once the API key proves valid (after the template fetch), async and
best-effort. This is the shared connection achievement — the Strava sync grants the
same id, so connecting **either** integration unlocks it.

Timezone: Hevy `start_time` is UTC with no local-time field, so it is converted to the
app's configured timezone (`hevyLocation()`, falling back to UTC) before the calendar day
is bucketed — otherwise late-night workouts would land on the wrong exercise day.

## Incremental sync (implemented)

`HevyEventsSyncForUser(user)` polls `GET /workouts/events?since=<HevyLastSync>`:

- `updated` events upsert via `HevySyncWorkoutForUser` (rebuilds the workout's contents).
- `deleted` events soft-disable the imported exercise + its operations/sets
  (`hevyDeleteWorkoutForUser`); a no-op if the workout was never imported.
- It runs only once a backfill has set `HevyLastSync` (so it never races the initial
  import), and advances `HevyLastSync` to the run's start time on success.

`HevyEventsSyncForAllUsers()` is the cron entrypoint (`database.GetHevyUsers()` → each
user). Scheduled hourly at **:30** when `HevyEnabled` (offset from Strava's :00), in
`main.go` via `codnect.io/chrono`.

## Source indicator (UI)

Hevy-imported workouts are marked with the Hevy logo (`/assets/hevy.png`), mirroring how
Strava is shown — but **without an external link**, since the Hevy API exposes no public
per-workout URL (the workout `id` is a private UUID, not the share slug).

- Front-page activity feed (`web/js/frontpage.js`): a Hevy logo next to the Strava logos,
  driven by `Activity.HevyWorkoutID` (populated in `season.go`/`user.go` from the exercise).
- Exercise builder (`web/js/exercise.js`): `sourceBadge()` renders a non-link "Hevy" badge
  when the parent exercise has a `hevy_workout_id` (and no `strava_id`), driven by
  `ExerciseObject.HevyWorkoutID`. CSS: `.wv-source-hevy` / `.hevy-logo-img`.
- User profile `/users/{id}` (`web/js/user.js`): a clickable Hevy logo linking to the
  user's Hevy profile, shown when `hevy_profile_url` is set **and** `hevy_public` is on —
  mirroring the Strava athlete link gated by `strava_public`. The "Show my Hevy on my
  profile" toggle lives in the `/account` Hevy section.

## Strava de-duplication (implemented)

When both integrations are connected, a strength session often lands twice (Hevy plus its
auto-push to Strava). "From Hevy" is **not** reliably detectable from the Strava API
(`external_id`/`device_name` are heuristics only), so dedup is done by **start-time
overlap on our own data** instead, gated by `User.StravaSkipHevyDuplicates` (auto-enabled
on first Hevy connect; toggle in the Strava settings on `/account`, shown only when Hevy is
enabled). Window: `hevyStravaOverlapWindow` (±15 min).

Bidirectional, so import order doesn't matter:

- **Strava import** (`StravaSyncActivityForUser`): if an enabled Hevy exercise overlaps the
  activity's start (`GetHevyExerciseForUserNearTime`), the activity is skipped.
- **Hevy import** (`HevySyncWorkoutForUser`): any enabled **Strava-sourced** exercise that
  overlaps (`GetStravaExerciseForUserNearTime` — only matches exercises with a set carrying
  a `strava_id`, never manual entries) is soft-disabled, so Hevy supersedes an
  already-imported Strava duplicate.
