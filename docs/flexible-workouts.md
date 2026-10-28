# Saved workouts (flexible workouts)

A season can let participants **save extra workouts** from one week and use them later to
make up a short week. The UI calls them **saved workouts** everywhere (front page, admin form,
joining a season, `/seasons`, the Sunday reminder, the AI greeting and MCP); the code says
*carry-over* (`CarryOverMax`, `nextCarryBank`, `CarryOverUse`). Never put "carry-over",
"bank" or "credit" in front of a user. Issue: #24.

## The rule

**Season settings** (set when the season is created; seasons are create-only, so the rule
never changes during a season):

| Field | Meaning |
|---|---|
| `Season.CarryOverMax` | the most saved workouts a participant can hold at once. `0` = off: the default, and what every season created before the feature has |
| `Season.CarryOverWeeks` | how many weeks a saved workout lasts. `1` = only usable the week after it was earned. Stored as `0` while the feature is off |

`APIRegisterSeason` accepts `carry_over_max` 0–10 and, when it is above 0,
`carry_over_weeks` 1–12; with the count at 0 the lifetime is stored as 0 whatever was sent.

Per week, per goal, in this order (`nextCarryBank` in `controllers/carryover.go`):

1. **Expiry.** Saved workouts whose last usable week has passed are dropped.
2. **Sick leave comes first.** A sick-leave week neither uses nor earns saved workouts, but
   the ones held keep ageing.
3. **Using.** A short week uses saved workouts only when the participant chose **Use saved
   workouts** for it, and only when they cover the **whole** shortfall: then exactly the
   shortfall, oldest first. Topping up a partly done week is fine (2 done + 1 saved meets a
   goal of 3); spending some and still missing is not. A week that met its goal on workouts
   never spends any, opted in or not.
4. **Earning.** In a week beyond its goal, the extra workouts (`workouts − ExerciseInterval`)
   are saved, up to `CarryOverMax` held in total. Each lasts `CarryOverWeeks` weeks after the
   week it was earned in.

Each season keeps its **own** saved workouts. Workouts are shared across seasons, but each
season judges them against its own goal, so a week can save one in a season with a goal of 2
and nothing in a season with a goal of 3.

Weeks before the participant joined are judged like any other week, exactly as the season
streak does (it also counts them): the expiry and the cap keep what can come in that way small.

### A week made up with saved workouts counts as met

`UserWeekResults.GoalMet()` is `WeekCompletion ≥ 1 || CarryUsed > 0` (the walk only ever
spends saved workouts when they cover the whole shortfall, so any use means the goal was met).
Everything reads the week through the `models/week_result.go` methods, so a made-up week:

- grows the season streak (`nextSeasonStreak` takes `GoalMet()`), and the streak
  achievements follow;
- is a `WonWeek()` for a competing member: a place on the wheel with the usual
  `WheelTickets()`;
- is never `OwesSpin()`: no debt.

`WeekCompletion` stays the raw `workouts / goal`, so the "more than 100%" achievement still
needs real workouts. `SavedByCarryOver()` says whether the week only met its goal thanks to
saved workouts (the 💾 on the leaderboards).

## Computed, not stored

There is no stored balance. The saved workouts are worked out in the same week-by-week season
walk that builds streaks (`RetrieveWeekResultsFromSeasonWithinTimeframe` →
`GetWeekResultForGoal`), carried per goal next to the running streak (`seasonWalk`,
`goalWalkState`). A saved workout is held as the index of the last season week it can be used
in (`carryBank`), oldest first. So anything that changes an old week (a late Strava import, an
edited workout) changes what follows, the same way it changes streaks.

New fields on `models.UserWeekResults`, set by the walk:

| Field (JSON) | Meaning |
|---|---|
| `carried_in` | saved workouts available entering the week |
| `carry_expiries` | the use-by day (the Sunday of its last usable week) of each of them, oldest first |
| `carry_opt_in` | the participant chose "Use saved workouts" for the week |
| `carry_used` | saved workouts the week used (0, or the whole shortfall) |
| `carry_earned` | extra workouts the week saved |

For the running week these say what happens **if the week ended now**: opted in at 2/3 with
one saved workout, the week already reports `carry_used: 1` and counts as met; one more workout
and it reports 0 used again.

`/weeks-personal` (statistics) adds `carry_used` and `goal_met`; the day header
(`GET /api/auth/exercise-days/:id/week`) adds `goal_met` and `carry_used` per season. The day
header used to judge its week alone, with no streaks; it now walks the season up to that week
like the leaderboard, so the two always agree.

## Using them: an opt-in per week

`models.CarryOverUse` (`GoalID`, week `Date`, `Enabled`), modelled on `Sickleave`. One enabled
row means the goal chose "Use saved workouts" for the week holding `Date`. Registered in
`Migrate()`; data access in `database/carryover.go`.

| Endpoint | Does |
|---|---|
| `POST /api/auth/carryover/:season_id` | turns it on for the caller's **running** week |
| `DELETE /api/auth/carryover/:season_id` | undoes it for the running week (disables the row) |

Both answer 400 when the season isn't running now, doesn't save workouts, or the caller has
no goal in it. `POST` also answers 400 in a sick-leave week, when it is already on, or when the
caller holds no saved workouts entering the week; `DELETE` when it isn't on this week. There is
no way to name another week: past weeks are settled and their opt-ins can't be changed.

Turning it on only says the week **may** use saved workouts; whether it does is decided when
the week is judged (rule step 3). So it is safe to turn on early.

## Where it shows

**Front page** (`web/js/frontpage.js`, all phrasing in the browser from the week result):

- **The ring** (`placeWeekProgress`): saved workouts making up the week are a second segment
  in the paler `--saved` tone right after the workouts arc, and the caption under `2 / 3` reads
  `+ 1 saved workout`. The ring turns green when the two together reach the goal.
- **The chip** under the ring (`.saved-chip`): `1 saved workout · use by Sun 9 Nov`; when they
  run out on different days, `2 saved workouts · 1 to use by Sun 9 Nov`; with none held but
  some being earned, `1 extra workout will be saved this week`. Hidden when there is nothing to
  say and in seasons without the feature. It opens a modal that explains saved workouts in plain
  words, lists each one with its use-by day, and offers the same choice as the week actions.
- **The note** under the chip once it is on: `1 saved workout makes up the rest of this week.`,
  or `Saved workouts are on. Yours make up the week once you're at 2/3.`, with **Undo** beside
  it.
- **Week actions** under the calendar's Save, next to "Use sick leave": **Use saved workouts**
  shows only while the week is short, not sick leave, and what you hold covers the whole gap,
  with a line saying what it will do: `Uses 1 of your 2 saved workouts if you end the week at
  2/3.` When you hold some but not enough, there is no button, only `Your saved workout can make
  up the week once you're at 2/3.` Once on, the button reads **Stop using saved workouts**.
- **Season panel**: `Saved workouts: N` and the season's rule under it.
- **Current week board**: `67% 💾` for a member whose saved workouts make up the week.

**Leaderboards** (front page and `/seasons`): `weekStatusIcon` / `streakStatusIcon`
(`web/js/functions.js`) give a made-up week its own icon, 💾, titled "Goal met with 1 saved
workout"; every icon now has a title. Order: sick leave, met on workouts, made up with saved
workouts, joined mid-week, missed.

**Admin season form** (`web/js/admin.js`): *Saved workouts per person* (0 turns it off) and
*Weeks each one lasts* (disabled while off), with a live line under them saying the rule as
people will read it, or `Off. Extra workouts aren't saved in this season.`

**Joining a season** (`registergoal.js`) and **`/seasons`**: the rule in one sentence when the
season has it, nothing when it doesn't. The sentence comes from `savedWorkoutsRule`, written
the same in Go and JS: `Up to 2 extra workouts can be saved, each for 1 week.` /
`1 extra workout can be saved, for 2 weeks.`

**Day page** (`/exercises/:id`): a season row's bar gets the same `--saved` segment and its
figure reads `2 + 1 saved / 3`.

**Sunday reminder** (`SendSundayReminders`): in a season that saves workouts, a member who is
short but holds enough to make up the week is told so and linked to the season on the front
page (`sundayReminderSavedWorkoutsNote`); one whose saved workouts already make up the week is
told the goal counts as met. The push notification is unchanged.

**AI greeting** ([ollama.md](ollama.md)) and **MCP** ([mcp.md](mcp.md)) get the rule, the
count and use-by day, the opt-in, and how many saved workouts make up the week, so a made-up
week is never described as missed.

## Testing it

- Pure rule: `TestNextCarryBank` (`controllers/carryover_test.go`).
- Season walk: a made-up week grows the streak, wins the wheel and owes nothing; sick leave
  takes precedence; three seasons sharing one user's workouts keep their own saved workouts;
  the day header agrees with the leaderboard.
- API: opt-in and undo, refusals (nothing saved, sick leave, twice, past week, ended season,
  feature off), season creation validation; the fault sweep has both endpoints.
- Live: `docker-test/seed-saved-workouts.sh` seeds a ready-made scenario, see
  [`docker-test/README.md`](../docker-test/README.md#saved-workouts-scenario).

## Related

- [seasons-and-goals.md](seasons-and-goals.md) — seasons, goals, the weekly outcome rules
- [streaks.md](streaks.md) — the season streak a made-up week grows
- [styleguide.md](styleguide.md) — the `--saved` token and the saved-workouts chip
