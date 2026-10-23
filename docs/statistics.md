# Statistics (`/statistics`)

The page for a user's own training, in five tabs: **Overview**, **Records**, **Activity
types**, **Seasons** and **Body**. Every number is computed server-side; `web/js/statistics.js`
only lays them out. The tab and the chosen activity type live in the URL (`?tab=types&action=<id>`),
so a view can be reloaded or linked; each tab loads its data the first time it opens.

## What counts

Statistics and records read **every logged workout**: an enabled day, an enabled and
switched-on workout, enabled activities and sets. Two things follow from that:

- **Counting toward the goal doesn't matter here.** That setting is about the weekly goal; an
  imported walk you excluded from the goal is still a walk you did. The one exception is the
  **personal streaks**, which keep their own rule (only goal-counting workouts, see
  [streaks.md](streaks.md)).
- **Hidden duplicates and deleted workouts are left out**, because both have
  `exercises.enabled = false` (see [activity-dedup.md](activity-dedup.md)).
- **Private workouts are included.** The page is only ever the owner's.

Distances and weights are reported **per unit** (`[]UnitAmount`) and never summed across units
(see [data-conventions.md](data-conventions.md)). Where a best has to be picked across units it
is compared in metres or kilograms (`distanceMeters`, `weightKg`) and shown in the unit it was
logged in.

The queries are in `database/statistics.go` (`GetStatisticsSessionRows`,
`GetStatisticsActivityRows`, `GetStatisticsSetRows`, `GetActionTracksForUser`), one flat row per
workout, activity or set, so the controllers never walk GORM trees or parse streams.

## Overview

`GET /api/auth/statistics/overview?period=4w|12w|year|all` → `APIGetStatisticsOverview`
(`controllers/statistics.go`, `buildStatisticsOverview`).

| Period | Range | Compared with | Buckets |
|---|---|---|---|
| `4w` (default) | this week and the 3 before | the 4 weeks before that | weeks |
| `12w` | this week and the 11 before | the 12 weeks before that | weeks |
| `year` | 1 January to today | the same span last year | weeks |
| `all` | the month of the first workout to today | nothing | months |

Weeks run Monday–Sunday in the server zone. The response holds the period's **totals**
(workouts, active time, active days, distance per unit) and the previous period's, a bucket per
week or month (empty ones included, so the chart has no gaps), the **activity mix** (activities
and time per type, most time first), the all-time **personal streaks** and up to six **records
set in the period**. A workout with no activities (the front page's "+") counts as a workout.

## Personal records

`GET /api/auth/records[?action_id=]` → `APIGetPersonalRecords`, and
`GET /api/auth/records/history?action_id=&key=` → `APIGetPersonalRecordHistory`
(`controllers/records.go`).

Records are computed on read. Each one is a series of candidates (one per activity or set);
`recordProgression` sorts them by date and keeps each one that beat everything before it. The
last entry is the record, the one before it is what it beat (`previous`), and the length is how
often it improved. A record set in the last 30 days is `fresh`. A tie doesn't count as an
improvement. Groups are ordered by how often the type was logged.

| Key | Applies to | Value | Compared as |
|---|---|---|---|
| `effort_<metres>` | foot sports and rides with a stream | fastest time over the distance | seconds, lower is better |
| `longest_distance` | `moving` activities | the activity's distance | metres |
| `longest_duration` | `moving` and `timing` activities | summed set times, else the activity's duration | seconds |
| `most_elevation` | activities with an elevation rollup | elevation gain | metres |
| `heaviest_set` | `lifting` sets with a weight | the set's weight (with its reps) | kilograms, more reps break a tie |
| `estimated_one_rep_max` | `lifting` sets of 1–10 reps | Epley: weight × (1 + reps/30), a single rep is the weight itself | kilograms |
| `most_reps` | `lifting` sets | reps in one set | reps |
| `most_volume` | `lifting` activities | weight × reps over every set | kilograms |

Epley drifts badly above 10 reps, so those sets don't feed the one-rep max. Each record carries
the activity type's Strava sport (`sport`) so the page can show a best effort as pace (foot
sports) or speed.

### Best efforts

A best effort is the fastest stretch of an activity over a fixed distance — the 5 km inside a
12 km run — found by a sliding window over the stream (`models.ComputeBestEfforts`,
`models/best_effort.go`). Distances depend on the sport (`BestEffortTargets`):

| Sports (`Action.StravaName`) | Distances |
|---|---|
| Run, TrailRun, VirtualRun, Walk, Hike | 1 km, 1 mile, 5 km, 10 km, half marathon, marathon |
| Ride, VirtualRide, EBikeRide, GravelRide, MountainBikeRide | 10, 20, 40 and 100 km |

- **Distance** is the same cumulative distance the splits use (`models.CumulativeDistanceMeters`:
  haversine over GPS, else integrated `velocity_smooth`), so a treadmill run with a speed stream
  gets best efforts too.
- **Time is the recorded clock.** A pause inside the window counts, as on a stopwatch.
- **The window covers exactly the distance**: its start is interpolated between samples.
- **GPS jumps are removed first.** A step between two samples faster than 1.5× the sport's cap
  adds no distance, and a whole window faster than the cap (30 km/h on foot, 100 km/h on a bike)
  is skipped. Without the first rule a 900 m jump could sit inside a longer window whose average
  still looked believable.
- We compute them ourselves rather than read Strava's `best_efforts`: Strava only offers them
  for runs, and only from the detailed-activity call.

Best efforts are **stored** (`models.BestEffort`, one row per activity and distance), because
the window search is far too heavy to run over every stream on every read. They are derived
data, always replaced as a set (`database.ReplaceBestEffortsForOperation`):

- **Strava sync** writes them with the stream (`StravaSyncOperationForActivity`).
- **Changing an activity's type** in the builder recomputes them from the stored stream
  (`refreshBestEfforts`), since the sport decides the distances and the cap.
- **The stream backfill** (`backfillStreamDerivedData` in `database/client.go`) computes them for
  existing activities, together with the rollups (below).

Reads join the rows through the usual enabled/on filters (`GetBestEffortRowsForUser`), so a
deleted or hidden workout drops out without touching them.

### Stream-derived data and its version

`Operation.StreamDerivedVersion` records which `models.StreamDerivedVersion` an activity's
rollup columns (average and max heart rate, cadence, temperature, elevation gain, average power)
and best efforts were computed with. At startup the backfill reprocesses every stream-bearing
activity below the current version, then stamps it, so it is self-limiting. **Bump the constant
when a derivation changes** and every stored stream is reprocessed once on the next boot. It
replaced the earlier all-NULL rollup backfill, which couldn't pick up a new column such as
`avg_watts`.

## Activity types

`GET /api/auth/actions/:action_id/statistics?start=&end=` → `APIGetActionStatistics`, and
`GET /api/auth/actions/:action_id/heatmap?start=&end=` → `APIGetActionHeatmap`. Both take whole
days (`YYYY-MM-DD`; an RFC 3339 timestamp is still accepted from older clients).

`buildActionStatistics` returns:

- **Totals**: activities, workouts, distance per unit, time and moving time, reps, and volume
  (weight × reps) per weight unit.
- **Averages per activity**, in the period's main unit.
- **Bests**: furthest, longest, heaviest set and most reps in a set, each with its day.
- **Sensors** from the rollup columns: heart rate, cadence and power are averaged weighted by
  each activity's moving time (else its duration), max heart rate is the highest, elevation gain
  is summed. Cadence is in the sport's unit (steps per minute for foot sports, see
  `cadenceUnitForAction`).
- **Trend**: one point per activity (distance, durations, average heart rate, top set and the
  best one-rep max estimate), which the page charts as pace or speed, distance, one-rep max, top
  set, duration or heart rate, with a rolling average over five activities.
- **Soundtrack** (`media`), aggregated over the distinct workouts, unchanged.

The page also shows that type's records and links to `/exercises?action=<id>`. The heatmap
endpoint returns only GPS tracks, so the statistics response never carries raw streams; see
[heatmap.md](heatmap.md).

## Seasons and Body

**Seasons** reads `GET /seasons/:id/weeks-personal` as before and opens on the season running
now (else the latest one that has started). The figures are tiles; the weekly chart draws the
goal as a dashed line and sick-leave weeks as grey points.

**Body** reads `GET /weights`: latest weight, change over 30 and 90 days (against the last
weight logged at least that long before), the chart, and the same log modal behind
"Log weight".

## MCP

`get_personal_records`, `get_activity_type_statistics` and `get_statistics` with `period` expose
the same computations to LLM clients, and `get_activity` takes `include: ["efforts"]`; see
[mcp.md](mcp.md#statistics-and-records-get_statistics-get_personal_records-get_activity_type_statistics).

## Not yet done

- Notifying a user when they set a record.
- Grade-adjusted pace and best efforts for other sports (swim, ski).
