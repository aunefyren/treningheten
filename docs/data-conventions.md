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
means setting `Enabled = false`; the standard getters filter on `enabled = 1`. Don't
assume a row is gone just because it's "deleted."

## Related

- [seasons-and-goals.md](seasons-and-goals.md) — entities that use these conventions
- [streaks.md](streaks.md) — consumers of the duration/activity data
- [mcp.md](mcp.md) — the read surface where these conventions matter most
