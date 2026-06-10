---
name: seasons-goals-workouts-relationship
description: Domain model — workouts are independent but contribute to each season's own goal; a user can be in 0/1/many seasons
metadata:
  type: project
---

A user's workouts form one independent weekly stream. Each season the user is in
applies its OWN weekly goal to that same workout count, so the same workouts can
satisfy one season's goal while falling short of another's. A user can be in zero,
one, or multiple active seasons simultaneously.

Mechanics (confirmed by user):
- Exactly ONE goal per user per season, created when joining the season.
- The `competing` flag is chosen at join time and determines wheel participation.
- Sick leave is granted per season and only applies to that season.
- A season streak = consecutive weeks within that season where you reached that
  season's goal. It is separate from the personal profile streak.
- Wheel mechanic (confirmed in controllers/debt.go ~line 519-538): missing your
  goal while competing incurs a debt and YOU spin the wheel. The wheel is filled
  with the people who SUCCEEDED that week (competing, goal met, no sick leave).
  Each successful person gets `Tickets = CurrentStreak + 1` entries. The spinner
  lands on one of them and THAT person wins the prize.
  - So tickets are earned by HITTING the goal, not missing it. A longer streak
    gives YOU more tickets, so when a rival fails and spins you are more likely to
    win the prize. Missing means you spin and hand a prize to a successful rival.
  - Flat-payload field name (locked): `wheel_tickets_if_you_hit_goal` =
    current_streak + 1, emitted only when competing.

**Why:** This is the core framing for the Ollama front-page feature
([[ollama-frontpage-flat-payload]] context). The flat payload must compute
goal-met / remaining / streak / spin-risk PER season (derived from the shared
`workouts_this_week` count), not once globally, or the AI conflates seasons.

**How to apply:** When flattening data for the model, put `workouts_this_week`
at the top level and emit one entry per active season under `seasons[]`. Reuse
`GetWeekResultForGoal` (controllers/season.go) which already computes week
completion, streak, sick leave, and debt per goal.
