# Streaks

Treningheten has **two different streak systems**. They answer different questions and
are computed in completely different places, so it is easy to confuse them — this
document is the map.

| | Personal streak | Season streak |
|---|---|---|
| **Question** | "How long have I kept exercising?" | "How long have I kept hitting my weekly goal in this season?" |
| **Depends on a season/goal?** | No | Yes (one per goal) |
| **Counts a period when…** | the day/week has any goal-counting session | the week's `WeekCompletion ≥ 1` (goal met) |
| **Granularity** | day **and** ISO-week | week |
| **Sick leave aware?** | No | Yes (freezes the streak) |
| **Stored?** | No (computed on read) | No (recomputed each time) |
| **Drives** | personal stats display | leaderboard, debts, wheel tickets |
| **Code** | `controllers/streaks.go` | `controllers/season.go` (`GetWeekResultForGoal`) |

## Personal streaks

Season- and goal-independent (no *weekly-goal* threshold), but a period only **counts**
when it contains at least one session that is enabled, "on" **and** flagged to count
toward the goal (`exerciseCountsTowardGoal` — see [data-model.md](data-model.md)). So a
logged-but-excluded activity (e.g. an imported walk a user opted out of) shows in the
history without keeping the personal streak alive.

For both day and week we track:

- **best** — the longest run of consecutive periods, ever.
- **current** — the run ending at the most recent period, but **only while still
  alive**. "Alive" means:
  - day streak: activity **today or yesterday** (otherwise current = 0),
  - week streak: activity **this ISO week or last** (otherwise current = 0).

So a streak you let lapse reports `current = 0` while `best` preserves the record.

Computed by `computePersonalStreaks` (`controllers/streaks.go`). This is a **single
shared builder** used by both:

- the web statistics endpoint `GET /api/auth/users/:user_id/statistics`
  (`APIGetUserStatistics`), and
- the MCP `get_statistics` tool.

so the two cannot drift.

> **History:** the web endpoint previously computed these inline and had an off-by-one
> bug — the first run in a user's history was undercounted by one (a lone active
> week/day read `0`). Extracting the shared builder fixed it; web streak numbers
> changed accordingly.

## Season streaks

Per **goal** (so per user per season). The unit is the **week**, and the rule is the
weekly goal, not mere activity. Computed in `GetWeekResultForGoal`
(`controllers/season.go`), which is called week-by-week with the user's running streak
threaded through (see [the weekly loop](seasons-and-goals.md#the-weekly-processing-loop)).

Per evaluated week, given `WeekCompletion = exercises_done / ExerciseInterval`, checked
**in this order** (`nextSeasonStreak`):

| Situation | Effect on the streak |
|---|---|
| Week covered by **used sick leave** | **frozen** — kept, never grown, **even if the goal was met** that week (week result flagged `SickLeave`) |
| `WeekCompletion ≥ 1` (goal met) | **+1** |
| Anything else (goal missed on a full week, or joined mid-week without meeting it) | **reset to 0** |

`nextSeasonStreak` is shared by the week results and the achievement pass
(`GenerateAchievementsForWeek`), so the 3/10/15/20-week streak achievements follow the same
rule: they need the streak to have *grown* that week, so a sick-leave week never awards one.
(Before this was shared, the achievement pass grew its own copy on a goal-met sick-leave week
and could award a streak achievement the real streak never reached. Awards made that way were
left in place.) Other achievements — e.g. beating the weekly goal — can still be earned in a
sick-leave week.

The streak value reported for a week (`UserWeekResults.CurrentStreak`) is the running
streak as it stood entering that week's update.

Because it is **recomputed** from week results every time (never persisted), reading a
user's current season streak means running the week loop for their goal — which is why
the MCP season tools deliberately omit standing/streak in v1 (see
[mcp.md](mcp.md)).

### Worked example

Weekly target `ExerciseInterval = 3`, one sick-leave week available:

| Week | Workouts | Sick leave | Streak after |
|---|---|---|---|
| 1 | 3 | — | 1 |
| 2 | 4 | — | 2 |
| 3 | 1 | used | 2 (frozen) |
| 4 | 3 | — | 3 |
| 5 | 2 | — | 0 (missed, no cover) |
| 6 | 5 | — | 1 |

## Where consequences flow

Season streaks feed the leaderboard and the wheel-of-fortune: ticket count is
`CurrentStreak + 1` (`UserWeekResults.WheelTickets`, see
[seasons-and-goals.md](seasons-and-goals.md#the-weekly-outcome-rules)). Personal streaks are
display-only (profile/statistics).

## Related

- [seasons-and-goals.md](seasons-and-goals.md) — goals, weekly completion, sick leave, the weekly loop
- [mcp.md](mcp.md) — `get_statistics` exposes personal streaks; season streaks are deferred
