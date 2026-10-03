# Seasons & Goals

The competitive core of Treningheten. A **season** is a time-boxed competition; a
user takes part by registering a **goal**. This document covers those two entities,
how participation works, sick leave, and the weekly processing loop that ties them
together. Streaks have their own document — see [streaks.md](streaks.md).

## Seasons

A `Season` (`models/season.go`) is a **global**, time-boxed competition. It is not
owned by a user.

| Field | Meaning |
|---|---|
| `Name`, `Description` | Display text |
| `Start`, `End` | The competition window |
| `Prize` | What the winner receives (see prizes/wheel) |
| `Sickleave` | How many weeks of sick leave each participant is allowed (see below) |
| `JoinAnytime` | If false, users may only join before/at the start; if true, they can join mid-season |
| `Enabled` | Soft-delete flag |

**Status** is derived from `Start`/`End` relative to now (not stored):

- `now < Start` → **upcoming**
- `Start ≤ now ≤ End` → **ongoing** ("active")
- `now > End` → **ended**

Code: `models/season.go`, `controllers/season.go`, `database/season.go`. Seasons are
created by admins (`POST /api/admin/seasons`).

**Seasons live in the server's zone.** `APIRegisterSeason` keeps the calendar dates the admin
picked and builds `Start` (Monday 00:00) and `End` (Sunday 23:59:59) in `time.Local` — the
configured `timezone` — via `seasonBoundariesInServerZone`. The request's `timezone` field is
accepted but ignored. Weekly processing computes its Monday–Sunday windows in that same zone,
and a season stored in another one put its final Sunday outside the last window, so that
week was never processed. The admin form (`addSeason` in `web/js/admin.js`) validates the
weekday on the picked calendar date in the browser's zone (`parsePickedDate`) and sends it
as `YYYY-MM-DDT00:00:00Z`, so the date survives unchanged whatever zone the browser is in.

## Goals — how a user participates

A user **does not join a season directly**. They create a `Goal` (`models/goal.go`),
which is the join record. There is **one goal per user per season**.

| Field | Meaning |
|---|---|
| `SeasonID`, `UserID` | The participation link |
| `ExerciseInterval` | The user's **weekly target** — number of workouts per week |
| `Competing` | `true` = competing for the prize (and exposed to debts); `false` = participating casually |
| `Enabled` | Soft-delete flag (leaving a season; also set when an admin disables the user — see [admin-users.md](admin-users.md)) |

`GoalObject` (the enriched read form, via `ConvertGoalToGoalObject`) adds a computed
`SickleaveLeft`. Because of this indirection, **"my seasons" = "the seasons I have a
goal in"** (`database.GetGoalsForUserUsingUserID`). Joining is
`POST /api/auth/goals`; leaving is `DELETE /api/auth/goals/:goal_id`.

> See [data-conventions.md](data-conventions.md) for the `Convert*Object` read-layer
> pattern these types follow.

## Weekly completion

Each week, a goal is evaluated against its target:

```
WeekCompletion = exercises_done_that_week / ExerciseInterval
```

The week's **goal is met when `WeekCompletion ≥ 1`**. This per-week pass/fail is what
drives season streaks, debts, and the leaderboard. The computation lives in
`GetWeekResultForGoal` (`controllers/season.go`).

A subtlety: `FullWeekParticipation` is `false` when the goal was created in or after
the week being evaluated (i.e. you joined mid-week), which keeps a just-joined user
from being penalised for a week they couldn't fully take part in.

## Sick leave

Sick leave lets a participant **miss a week without breaking their season streak**.

- The season grants an allowance: `Season.Sickleave` weeks.
- A `Sickleave` row (`models/sickleave.go`) belongs to a **goal** and a week; it has a
  `Used` flag and a `Date`. `SickleaveLeft` on `GoalObject` is the count of the
  participant's unused sick-leave rows.
- When a week is covered by **used** sick leave, the season streak is **frozen**
  (preserved, not incremented and not reset) and the week result's `SickLeave` flag is
  set.

Registered via `POST /api/auth/sickleave/:season_id`. Code: `models/sickleave.go`,
`database/sickleave.go` (`GetUsedSickleaveForGoalWithinWeek`,
`GetUnusedSickleaveForGoalWithinWeek`).

## The weekly processing loop

Two scheduled jobs (cron via `codnect.io/chrono`, registered in `main.go`) run the
weekly rhythm:

| Job | Schedule | What it does |
|---|---|---|
| `SendSundayReminders` | 18:00 Sunday (`0 0 18 * * 7`) | Nudges users who haven't hit their goal yet |
| `ProcessLastWeek` | 08:00 Monday (`0 0 8 * * 1`) | Finalises the finished week for each ongoing season (`ProcessWeekOfSeason`): generates **debts** (`GenerateDebtForWeek`) and awards weekly/season **achievements**. Week results — including streaks — are computed as part of this; they are not stored on their own. |

A loser whose debt can't be checked or written doesn't stop the others: `GenerateDebtForWeek`
attempts every loser, then returns the collected failures wrapped in
`ErrDebtPartiallyGenerated`. `ProcessWeekOfSeason` still runs the achievements (the week's
results are sound) and then returns that error, so the cron logs it and the admin "generate
debt" action answers 500 instead of claiming success. Re-running is safe — losers who already
have a debt for the week are skipped.

For display, the leaderboard rebuilds week results on demand via
`RetrieveWeekResultsFromSeasonWithinTimeframe`, which walks the season's weeks calling
`GetWeekResultForGoal` and threading each user's running streak through. **Season
streaks are therefore recomputed, not stored** — see
[streaks.md](streaks.md#season-streaks).

## Consequences: debts, leaderboard, prizes

Missing a week while `Competing` can create a **debt**; the season leaderboard ranks
participants; the season `Prize` is awarded through the wheel-of-fortune, whose ticket
count is tied to the season streak (`debt.go` uses `CurrentStreak + 1`). These
mechanics are adjacent to this document; the key cross-cutting concept is the season
streak, documented next.

**Who can see a debt.** A wheel spin is a season-social event, not a private one: the week
table on the front page and `/seasons` links every participant's debt, so
`GET /api/auth/debts/:debt_id` is scoped to **season membership** — the caller must hold a
`Goal` in the debt's season (`database.VerifyUserGoalInSeason`), otherwise 403. Acting on a
debt is narrower still: only the loser may spin it (`APIChooseWinnerForDebt` checks
`debt.LoserID`). The winner is saved before anything announces it, with a
`winner_id IS NULL` condition (`database.UpdateDebtWinner`) so two concurrent spins can't both
land; a failed save answers 500 and creates no wheel views, so the loser can simply spin again. The candidate list returned alongside a debt is built from the **censored**
user getter — it is a response body, so it must never carry credentials (see
[conventions.md](conventions.md#never-serialize-a-credential)).

## Related

- [streaks.md](streaks.md) — personal vs season streaks, in detail
- [data-conventions.md](data-conventions.md) — the `Convert*Object` read layer, units, durations
- [mcp.md](mcp.md) — how seasons/goals are exposed (read-only, personal) to LLM clients
