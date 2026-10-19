# Duplicate activities

With more than one import source connected, the same workout often lands twice — most
commonly a Hevy workout plus Hevy's auto-push of it to Strava. Duplicate detection keeps one
copy: the one from the source the user ranks highest. The other copies are hidden, not
deleted, so changing the ranking later can bring them back.

Code: `controllers/activity_dedup.go` (logic), `database/activity_dedup.go` (queries).

## The setting

`User.ActivitySourcePriority` is a comma-separated, highest-first list of integration
providers (`models.IntegrationProvider*`), e.g. `"hevy,strava"`.

| Value | Meaning |
|---|---|
| `NULL` | Never set. Connecting Hevy sets it to the default `"hevy,strava"` (Hevy carries the richer set/rep data). |
| `""` (or one source) | Detection off. An explicit off is kept on later connects. |
| `"hevy,strava"` / `"strava,hevy"` | Detection on; the first source wins. |

Set with `PATCH /api/auth/users/:id` `{"activity_source_priority": "strava,hevy"}`.
`normaliseActivitySourcePriority` trims and lowercases the value, and rejects unknown or
repeated sources with a 400. Known sources are listed in `activitySources`; a new import
source (Garmin, Apple Health) is added there plus a case in `classifyImportedExercises`, with
no schema change.

On `/account` it is the **Duplicate activities** section under *Exercise connections*,
shown only when the server has at least two import sources enabled: a toggle and a ranked
list with "Move up" buttons. Each change saves immediately.

It replaces the Hevy-only `User.StravaSkipHevyDuplicates` flag (deprecated, no longer read).
`migrateStravaSkipHevyToSourcePriority` in `Migrate()` gives every user who had the flag on
`"hevy,strava"` (unless they already have a ranking) and clears the flag.

## What counts as a duplicate

Two sessions are duplicates when all of these hold:

- they come from **different** sources — two Strava activities close in time are never
  duplicates of each other;
- they **start within `duplicateActivityWindow` (±15 min)** of each other (by start time,
  not overlapping duration);
- both are enabled, switched on (`IsOn`) and timed, and both sources are in the ranking.

A session's source is decided from our own data, since "came from Hevy" can't be read
reliably from the Strava API (`external_id`/`device_name` are only heuristics):

| Source | Rule |
|---|---|
| Hevy | `Exercise.HevyWorkoutID` is set |
| Strava | exactly **one** distinct Strava id across the session's enabled sets |
| — | anything else: **manual sessions** and **combined Strava sessions** (several activities merged with `APIStravaCombine`) never hide anything and are never hidden. |

## Hiding and restoring

Hiding sets `Exercise.Enabled = false` and `Exercise.SupersededByID` to the winning session.
**Only the session row changes.** Its operations and sets are left as they are, because every
read path already filters on `exercises.enabled`, so a restored session comes back exactly as
it was, including any operation the user had removed. `SupersededByID` is what tells a hidden
duplicate (restorable) apart from a deleted session (not restorable).

When a session is hidden, anything that was hidden behind *it* moves to the new winner
(`RepointSupersededExercises`), so chains stay one level deep.

### On import (`resolveImportedDuplicates`)

Both syncs call it after storing a session (Strava: once the operation exists, since that is
what marks the session as Strava-sourced; Hevy: at the end of the import):

- If another source's session in the window ranks higher, the incoming session is hidden
  behind the best-ranked one. A newly imported Strava activity that ends up hidden skips the
  soundtrack pull.
- Otherwise every lower-ranked session in the window is hidden behind the incoming one.

Imports are **always stored first** and then possibly hidden; nothing is skipped before it is
saved. This costs the streams + detail calls for a Strava duplicate's first import, and in
return a ranking change can be undone.

Re-syncs leave hidden copies alone. `GetExerciseForUserWithStravaID` /
`GetExerciseForUserWithHevyWorkoutID` only see enabled sessions, so each sync first checks
`GetSupersededExerciseFor…` and returns early for a hidden copy. Without that check the
hourly sync would import the activity again every run. It also means the hidden copy is not
refreshed: an edit made upstream to a hidden workout shows up only after it is restored and
next synced.

### When the winner goes away

- **Hevy workout deleted** (`hevyDeleteWorkoutForUser`): everything it was hiding is restored
  (`restoreDuplicatesOf`), and each restored session is resolved again in case another
  duplicate still outranks it. If the deleted workout was itself a hidden copy, it only loses
  its `SupersededByID` mark (`hevyForgetHiddenWorkoutForUser`), so a later reconcile can't
  bring back a workout that no longer exists in Hevy.
- **Strava activities combined** (`APIStravaCombine`): a combined session takes no part in
  detection, so whatever its parts were hiding is restored. Splitting it again
  (`APIStravaDivide`) needs no hook, because the next sync resolves each single activity.
- **Winner switched off** in the builder (`IsOn = false`): the hidden copy **stays hidden**.
  Switching a session off is a reversible builder action, not a deletion.

### When the ranking changes (`reconcileDuplicateActivitiesForUser`)

A changed ranking (compared after normalising) starts a background reconcile of the user's
whole history (`goSafely`). It is idempotent:

1. restore every hidden duplicate of the user;
2. if detection is on, classify all imported sessions, sort them by time, and go through them
   highest-ranked first, hiding every lower-ranked other-source session within the window.

So turning detection on applies to past activities, flipping the ranking swaps which copy
shows, and turning it off brings everything back.

**Weeks already processed are not recalculated.** Debt, streaks and achievements from
`ProcessLastWeek` stay as they were. Which copy shows can change whether a week counts (e.g.
when only one of the two sources' activity types counts toward the goal), and that only
affects the current week and later ones. The account page says so.

## Related

- [hevy.md](hevy.md), [strava.md](strava.md) — the two import sources.
- [data-model.md](data-model.md) — `Exercise.Enabled` vs `IsOn` vs `SupersededByID`.
