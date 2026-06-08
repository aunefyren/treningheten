# Data conventions & gotchas

Cross-cutting conventions in the data model that are non-obvious and have bitten real
code. Read this before touching read paths, durations, or distances.

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

## Durations are stored as **seconds**, not nanoseconds

Several fields are typed `*time.Duration` but **hold a plain integer count of
seconds**, not Go's native nanoseconds:

- `Operation.Duration`
- `OperationSet.Time`
- `OperationSet.MovingTime`

Strava import casts seconds straight in (`time.Duration(activity.ElapsedTime)` in
`controllers/strava.go`), and stats read them back as raw integers
(`int(*set.Time)`). Nothing in the app calls `.Seconds()`/`.Minutes()` on them.

**Rule:** treat these as a seconds count — convert with `int64(*d)`, **never**
`d.Seconds()`.

> This caused a real bug: the MCP layer was the first consumer to interpret the field
> as a true duration (`int64(d.Seconds())`), which divided a seconds-count by 1e9 and
> returned `0` for every Strava time. Fixed by switching to `int64(*d)`.

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
