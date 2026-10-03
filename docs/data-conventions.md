# Data conventions & gotchas

Cross-cutting conventions in the data model that are non-obvious and have bitten real
code. Read this before touching read paths, durations, or distances.

For **what the entities are and how they relate** (the entity catalog, the domain-spine
diagram, and the model / `Object` / DTO struct flavors), see
[data-model.md](data-model.md). This doc covers the *gotchas* that apply to those
entities.

## The `Convert*Object` read layer

Database access returns raw GORM models (`Exercise`, `Operation`, `OperationSet`,
`Goal`, `Season`, …). Throughout the app these are converted to an **`Object` form**
via `Convert<X>To<X>Object` functions in `controllers/` before use
(`Exercise → ExerciseObject`, `Operation → OperationObject`, `Goal → GoalObject`, …).

This grew from an early decision made before the maintainer was comfortable with
GORM's relation preloading, so the `Convert*` functions hand-resolve what GORM could
otherwise `Preload`. They are **not** trivial field copies — they do real work:

- resolve associations (e.g. `Operation.Action` from `ActionID`),
- roll up derived data (e.g. an exercise's Strava IDs),
- apply fallbacks (e.g. `ExerciseObject.Time` falls back to the exercise day's date),
- compute values (e.g. `GoalObject.SickleaveLeft`).

**Rule:** these `Object` functions are the canonical read layer. New read paths (API
handlers, the MCP data layer, jobs) should consume `…Object` structs rather than
re-walking raw GORM models — otherwise the enrichment/fallback logic gets duplicated
and drifts.

## Durations are a raw **seconds** count (`int64`), not nanoseconds

A handful of fields hold a workout/set duration as a plain integer count of
**seconds**:

- `Exercise.Duration`, `Operation.Duration`
- `OperationSet.Time`, `OperationSet.MovingTime`
- the derived statistics totals: `StatisticsSumCompilation.Time` /
  `StatisticsAverageCompilation.Time`, the `UserStatistics*Compilation.Time` fields,
  and the `ActionMediaStatistics` times (`ListeningTime`, `SpokenTime`, …)

They are typed **`*int64`** (nullable on persisted rows) or **`int64`** (the computed
stats) — *not* `time.Duration`. The DB column is a `BIGINT` holding the seconds value
directly, and the JSON is a plain integer of seconds (the frontend formats it with
`secondsToDurationString`). Strava/Hevy import store seconds straight in
(`int64(activity.ElapsedTime)`); reads and the MCP layer consume the integer as-is.

**Rule:** treat these as a seconds count. When you derive one from a real elapsed
time, convert explicitly — `int64(end.Sub(start).Seconds())` — and never assign a
`time.Duration` to these fields.

> These were historically typed `*time.Duration` while holding a seconds count — a lie
> the compiler couldn't catch. The MCP layer once read one with `.Seconds()`, dividing
> the seconds value by 1e9 and returning `0` for every Strava time. The fields were
> retyped to `int64`; **no data or schema migration was needed** — the stored values
> were already seconds and GORM maps both `time.Duration` and `int64` to `BIGINT`.

## Units: distance and weight are per-operation, free-form

`Operation` carries a `DistanceUnit` (default `km`) and `WeightUnit` (default `kg`).
These are **free-form strings set per operation**, so different activities can use
different units. Strava import stores distance already converted to km
(`meters / 1000`).

**Rule:** never sum a distance/weight column across operations without accounting for
units — a naive sum mixes km with miles and produces a confidently-wrong number. The
MCP `get_statistics` normalises distance to km server-side before summing (unknown
units are treated as km, the dominant case); follow that pattern, or report per-unit.

## Soft deletes

Most tables carry an `Enabled` flag (and GORM's `DeletedAt`). "Deleting" generally
means setting `Enabled = false`; the standard getters filter on `enabled = true`. Don't
assume a row is gone just because it's "deleted."

`Exercise` additionally has `IsOn` (a *reversible* builder soft-delete — off = removed
from counts but restorable in the builder) and `CountsTowardGoal` (excluded from the goal
while still visible). These are three distinct flags — see [data-model.md](data-model.md).

## GORM drops a `false` on insert when the field has `default: true`

A bool tagged `gorm:"default: true"` (`Exercise.Enabled`, `Exercise.IsOn`,
`Exercise.CountsTowardGoal`, `UserActivityGoalSetting.CountsTowardGoal`, …) is a **zero
value** when it's `false`, so GORM leaves it out of the `INSERT` entirely and the column
default — `true` — silently wins. `Instance.Save()` hits this too: on a row that doesn't
exist yet it falls back to an insert. An `UPDATE` on an existing row is unaffected.

**Rule:** when a *newly created* row must carry `false`, write it with an explicit column
update after the insert, never by setting the struct field. The established helpers are
`database.SetExerciseCountsTowardGoal` (used by both the Strava and Hevy importers to
persist a per-activity-type opt-out) and `database.UpsertActivityGoalSettingInDB`. This
trap is why an excluded Strava walk still counted toward the weekly goal in production.

**Capture the value before `Create`.** GORM also reads the column default *back into the
struct* after the insert, so `field` is already `true` by the time a follow-up update
reads it — take a copy first (`competing := goal.Competing`, then `Create`, then
`Update("competing", competing)`).

The same trap was behind three more bugs, all fixed with this pattern (2026-09-29): a
member who joined a season as **non-competing was stored as competing**
(`database.CreateGoalInDB`, and so could be handed debts), a session created "off" was
stored as on (`APICreateExercise` → `database.SetExerciseIsOn`), and every **confidential
OAuth client was stored as public** (`database.CreateOAuthClient`), so its secret was never
checked — see [security.md](security.md). When adding a `default: true` bool, grep for its
`Create` paths.

Test seeds hit the same wall — `disableRow` in `database/activity_test.go` exists purely
to flip a seeded row to `Enabled: false` after insert.

## Date-range bounds are strings — never add `.000` to the lower bound

The whole-day lookups (`database/exerciseday.go`, `debt.go`, `sickleave.go`) bound the
`date` column with strings: `>= "YYYY-MM-DD 00:00:00"` and `<= "YYYY-MM-DD 23:59:59"`.
SQLite stores times as text (`"2026-09-29 00:00:00 +0000 UTC"`) and compares as text, so a
lower bound of `"… 00:00:00.000"` sorts *above* a value stored at exactly midnight (`' '`
< `'.'`). Days are always stamped at midnight, so on SQLite the range's first day —
Monday, for a week — was invisible: week results missed Monday's workouts and single-date
lookups created a duplicate day for the same date. MySQL reads both forms as the same
DATETIME, which is why it only showed up on SQLite. Regression test:
`database.TestExerciseDayLookupsFindMidnightDays`.

(The upper bound has the mirror-image edge — a value stamped exactly `23:59:59` sorts
above `"… 23:59:59"` on SQLite — but nothing stamps that time today.)

On MySQL and Postgres the string bounds are read in the **server zone**: MySQL because
the driver connects with `loc=Local` and `main.go` sets `time.Local` from the configured
`timezone`, Postgres because `Connect` passes the same zone as the session `TimeZone`. So
a "day" is a calendar day in the configured zone. Manual days (local midnight) and
Strava/Hevy days (UTC midnight, a few hours later in Nordic zones) both land on the right
date; SQLite instead compares the stored wall-clock text.

The single-day lookups order by `created_at` and take one row, so if duplicate days for a
date ever exist they all resolve to the oldest — "find or create" callers keep reusing it
rather than adding another. Duplicates left over from before the fix are merged at startup
by `mergeDuplicateExerciseDays` (`database/exerciseday_merge.go`): sessions move to the
oldest day, notes are appended, a missing goal is adopted, and the rest are disabled.

## Batch inserts with mixed optional fields fail on SQLite

`tx.Create(&slice)` issues one multi-row `INSERT`. When the rows differ in which
`default:`-tagged fields are set, GORM fills the gaps with the SQL `DEFAULT` keyword, which
SQLite rejects (`near "DEFAULT": syntax error`). `ReplaceMediaPlaybackForExerciseProvider`
hit this — a soundtrack mixing a book (no episode id) and a podcast episode was never
stored on SQLite — and now inserts row by row inside its transaction. Prefer that for any
batch whose rows aren't uniformly shaped.

## Related

- [seasons-and-goals.md](seasons-and-goals.md) — entities that use these conventions
- [streaks.md](streaks.md) — consumers of the duration/activity data
- [mcp.md](mcp.md) — the read surface where these conventions matter most
