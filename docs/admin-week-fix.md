# Fixing a week

The admin panel (`/admin`) has a **Fix a week** module for correcting a finished week after
the fact. The usual case: someone forgot to log, the Monday job gave them a debt, and the
admin wants to put the week right. It replaced the old "Generate debt" button, which could
only add debts and achievements. Cleaning up meant editing the database by hand.

## The flow

1. **Pick a day in a finished week.** The panel loads `GET /api/admin/weeks?date=YYYY-MM-DD`.
   This is a dry run of the fix for every season running that day. For each participant
   it shows:
   - workouts against the goal, and sick leave left
   - the week's outcome: won, owes a spin, sick leave, not competing, or joined this week
   - the debt, and what applying the fix would change
2. **Correct the inputs** with the row's actions, which reload the overview:
   - **Add workout**: `POST /api/admin/users/:user_id/exercises` `{date, note}`. This adds
     one goal-counting session to that day and creates the day if needed. The note defaults
     to "Added by admin". It is refused for a future day, a day already holding three
     counting workouts, or a user with no goal in a season running that day.
   - **Grant sick leave**: `POST /api/admin/users/:user_id/sickleave` `{season_id, count}`.
     This adds 1–10 unused sick-leave weeks to the user's goal, for someone who has used up
     their allowance. The user registers sick leave themselves as usual. Admins can't apply
     or retract sick leave for a past week.
3. **Apply**: `POST /api/admin/weeks/reconcile` `{date, dry_run, notify}`. **Notify affected
   users** is off by default, so an admin fix sends no e-mails or pushes unless it is ticked.
   A partial failure answers 500 and still includes the reports.

Removing a user's workouts is not part of the tool. The user can do it themselves.

## Reconciliation

`ReconcileWeekOfSeason` (`controllers/week_fix.go`) is the only code that writes a week's
debts. The Monday job calls it for last week through `ProcessWeekOfSeason`, with
notifications on. The admin tool calls the same code. It refuses a week whose Sunday isn't
over yet (`ErrWeekNotFinished`).

It reconciles the **whole season-week**, every participant at once, because fixing one
person can change the right answer for others. It works from the week results (the
`OwesSpin` / `WonWeek` rules in [seasons-and-goals.md](seasons-and-goals.md#the-weekly-outcome-rules)).

**Desired debts:**
- If nobody won the week, nobody owes.
- Otherwise every participant who owes a spin gets one debt:
  - assigned to the winner, if there is exactly one;
  - left to the wheel (no winner), if there are several.

A stored debt is **settled** once it is paid or its wheel has been spun. A spin is
recognised by its `Wheelview` rows. A settled debt is never changed. When it disagrees with
the week, the report carries a warning for the admin to judge.

**Per participant**, comparing the enabled debts of the season-week with the desired one:

| Stored | Week calls for | Action |
|---|---|---|
| none | a debt | create it, and grant the single-winner achievements |
| unsettled | no debt | switch it off and revoke its single-winner achievements |
| unsettled, other winner | a debt | re-assign it, or hand it back to the wheel, moving the achievements with it |
| settled, disagrees | anything | leave it and add a warning |
| several | — | keep one (a settled one if any, else the oldest) and switch the rest off |
| matching | — | nothing |

A spun debt matches a "wheel" week as long as the wheel landed on one of the week's current
winners.

**Example.** Two people lost to the only winner W. Then one of them is fixed into a winner.
That person's debt goes, and the other loser's debt is handed back to the wheel. W loses
both "Lucky bastard"s and the remaining loser loses "What you deserve". The wheel spin will
grant them again.

**Debts of people without a goal in the season any more** (they left or were disabled) are
reported and left alone.

**Writes are conditional.** Switching a debt off or changing its winner only succeeds while
the debt still has the winner that was read and is unpaid
(`database.SetDebtWinnerIfUnchanged`, `DisableDebtIfUnchanged`). A spin or payment that lands
in between makes the write a no-op and adds a "run it again" warning. Runs are serialised by a
mutex, so the Monday job and an admin can't both create the same debt.

### Achievements

- **Single-winner debt achievements.** These are "Lucky bastard" for the winner and "What
  you deserve" for the loser, dated the week's Sunday. They are granted and revoked with the
  debt they belong to, one pair per debt. Revoking switches the delegation off
  (`AchievementDelegation.Enabled = false`) rather than deleting it.
- **Weekly achievements,** plus the season achievements in a season's last week. These are
  granted **once per user per achievement per day** (`achievementGrantOptions.OncePerDay`),
  so re-running a week never hands them out twice. They are never revoked. One-off grants
  (wheel spins, joining a season, an admin's hand-out) keep stacking as before.

### Idempotency

With no data change, a second run changes nothing: no rows, no mails, and `changed: false` in
the report. The tests check this after every scenario.

## Report

Each season gets a `WeekReconcileReport` (`models/week_fix.go`):
- the week (number and Monday–Sunday), and the counts of winners and losers;
- `changed`, and `dry_run`;
- season-level warnings;
- one `WeekReconcileUser` per participant, with:
  - the week result and sick leave left;
  - the debt before and after;
  - the changes, achievements granted and revoked, and warnings.

## Tests

- `controllers/week_fix_test.go`, using `newAPIHarness`. It covers:
  - the forgot-to-log flow;
  - the ripple in both directions, and swapping the only winner;
  - paid and spun debts;
  - duplicate debts;
  - every reason a debt can stop being owed: sick leave, not competing, joining mid-week,
    workouts added, saved workouts;
  - weeks with no winners;
  - dry runs and notifications;
  - re-runs that don't double achievements;
  - a season's last week, two seasons in one week, and leavers;
  - request validation, and both admin actions.
- `database/week_fix_test.go` covers the conditional updates and the new queries.
- The fault sweep (`controllers/fault_scenarios_test.go`) replays every endpoint with each
  database call failing.
