# Work in progress

## Plans & Ideas

### More workout tags
Ideas for tags:
- Easy
- Splits

Must respect Strava sync

### Remove walk filter from Strava sync
- Instead, add a boolean to exercises, like "count toward goal" or similar
- Let users set on Strava settings whether any activity type count toward goal
- Only on initial sync, if you edit any workout, simply change the bool if you want to count it
- Must be incorporated into every logic/if where the program counts amount of valid exercises

### Flexible workouts
Work out more one week, have the extra effort carry over.
Must be season specific setting
Option to allow how many workouts carry over, how long they can exist before they decay
Must be user friendly and understandable in the UI

### Front page activities, add partner
- Allow a person to tag their partner/group on their exercise session within builder
- Show as activity together on season activity post if both have joined season
- Sync over Strava partner tag?
- Auto-detect partner of enough data?

### Gear tracker — possible follow-ups
The gear feature shipped (manual + Strava gear, per-operation storage with a session-level
builder selector + "Manage gear" modal, locally computed km distance — see `docs/gear.md`).
Open refinements left for later:
- **Per-operation gear UI.** The schema stores gear on the operation, but the builder only
  exposes a session-level selector. A combined Strava session that genuinely mixes gear can't
  be edited per-activity yet.
- **Auto-assign primary.** The selector *suggests* the user's primary gear for a session with
  no gear, but it isn't persisted until the user interacts. Could auto-assign on the first
  operation instead.
- **Primary per type.** Only one primary per user today; a primary shoe *and* a primary bike
  might be more useful.

### leave season button is not working
- Which season? all?

### Delete account button is not working
- What gets left behind? Do seasons you joined still show you? Show 'Deleted user'?

### best effort system
- Manual programming per activity?
- "fastest 5K"...
- Must be calculated at save or during runtime?
- Notification integration for PR?
- PRs for reps and weight on strength exercises?
- Time based best efforts? During this season? During this year?

## AI Ollama feedback on exercises?
- How to avoid spamming the LMM
- Little model, can the feedback be decent?

### Make first day of the week changeable
- Default monday, but choose
- Big changes to logic

## Problems

### Site loads
But sometimes not? Server asleep?

### Modal on exercise building page must be updated to match exercise builder

### If you have a long name (perhaps emoji as well) you can break the wheel look
Name gets placed outside inside of inside the slice

### Activity heatmap issues
- Spot based coloring, probably the GPS cords given by the device. Could this be smoothed out / replaced with a line?
- Is coloring based on frequency? A single run in a single place should not be red if there are a 100 other runs elsewhere
