# Vocabulary

The words the interface uses for the domain, and what each one maps to in the code. Every
page, button, tooltip, confirm dialog and API error message a user can see follows this
list. The code and the API keep their historical names (`Exercise`, `Operation`, "session");
this page is the bridge between the two.

| In the UI | Means | Model | Internal / API name |
|---|---|---|---|
| **Workout** | One logged workout: a run, a gym visit, a padel match. What counts toward the weekly goal. | `Exercise` | "exercise" (routes, JSON), "session" (feed internals, docs) |
| **Activity** | One part of a workout: the run, the bench press, the plank. A workout holds one or more. | `Operation` | "operation", "activity" (feed, MCP) |
| **Activity type** | The catalogue entry an activity is: Running, Bench Press (Barbell). | `Action` | "action" |
| **Set** | One set or lap inside an activity. | `OperationSet` | "operation set" |
| **Day**, **day note** | A calendar day and its note. Its page is `/exercises/:id`. | `ExerciseDay` | "exercise day" |
| **Weekly goal** | How many workouts per week a user commits to in a season. | `Goal.ExerciseInterval` | "exercise interval" |
| **Season** | A time-boxed competition. | `Season` | — |
| **Sick leave** | A season week excused from the goal; the streak is kept. | `Sickleave` | — |
| **Saved workout** | A workout beyond the weekly goal, kept to make up a later short week ("Use saved workouts"). Never "carry-over", "bank" or "credit" in the UI. See [flexible-workouts.md](flexible-workouts.md). | computed; opt-in `CarryOverUse` | "carry-over" (`CarryOverMax`, `carry_used`, `/carryover`) |
| **Streak** | Weeks in a row at 100% of the weekly goal (see [streaks.md](streaks.md)). | — | — |
| **Gear** | Shoes, bikes and the like, tracked by distance or time. | `Gear` | — |
| **Record** | The best you have done for one measure of one activity type: longest run, heaviest bench press set. "Personal record" in full; never "PR" or "PB" in the UI. | computed (`PersonalRecord`) | "record" |
| **Best effort** | The fastest stretch of an activity over a fixed distance (the 5 km inside a 12 km run). Feeds the time records. | `BestEffort` | "best effort" |

## Names, not units

Two words name a place in the app rather than a thing in it. They stay as they are because
they are in URLs and in people's habits:

- **Exercises**: the nav section and its URLs (`/exercises`, `/exercises/:id`). The page
  lists workouts; links to it read "All exercises".
- **Activity feed**: the front-page module showing workouts from the people you share a
  season with (see [activity-feed.md](activity-feed.md)). Its items are workouts.

## Rules

- The UI says **workout**, never "session". "Session" is fine in code, comments and these
  docs, where it names the same `Exercise` row.
- **"Exercise" is never a unit of anything** in the UI: not a workout ("three exercises a
  day"), not an activity ("Add exercise"), not a count ("Exercise count"). As a verb it is
  fine ("How many days a week are you going to exercise?").
- An **activity** is always part of a workout. Something that applies to the whole workout
  (duplicates, privacy, counting toward the goal, the soundtrack) is said about the workout.
- Keep the same word through a flow: the "Add workout" button creates a workout, "Delete
  workout" asks "Delete this workout?".
- New copy follows the writing rules in [styleguide.md](styleguide.md#writing) as well:
  sentence case, active voice, plain verbs.
