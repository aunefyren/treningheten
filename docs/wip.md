# Work in progress
This doc should contains plans, ideas, problems, bugs and so on. Finished stuff should not live here, it should be moved to /docs in relevant files. If none fit, it should probably be made. If features are in between start and done implementation, one can still create a doc which get's updated over time.

## Plans & Ideas

### More workout tags
Ideas for tags:
- Easy
- Splits
Must respect Strava sync

### Sick leave is per season/goal
- makes sense that different seasons have different sick leave
- makes little sense that you can join multiple seasons at once, but only use sick leave on one goal

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

### Music / audio integration
Overlay your listening history onto time-based activities. The **Plex**, **Spotify**, and
**Audiobookshelf** connections, the history pull, the session-level timeline, the
`/statistics` Soundtrack block, Plex audiobook/podcast classification, and Plex artwork
(via proxy) have all shipped. Full design + current status live in
[`docs/media.md`](media.md) — keep open items there, not duplicated here.

**Still to build:**
- **Cross-activity stats — "fastest songs":** avg speed over `[StartedAt, EndedAt]` from a
  session activity's `OperationSet.StravaStreams`, which needs the stream-windowing done
  per track. The one remaining planned increment. (Per-track avg-HR on the card already
  shipped.)
- **Per-(session, provider) pull guard:** the single `Exercise.MediaRetrievedAt` spans all
  providers — fine for the common case, but connecting a provider *after* a session was
  already pulled relies on the 🎧 re-pull button. Generalize the guard when it becomes
  annoying.
- **Edge case (note, not solving now):** editing a session's **time** changes the match
  window — the 🎧 re-pull re-matches on demand, but there's no automatic re-match on a time
  edit. (A skipped date-only session will match once a real time is set + re-pulled.)

**Open questions:**
- Privacy: listening data is sensitive even self-hosted — any per-activity visibility controls?
- Cross-provider de-dupe if a user has overlapping sources (e.g. casting Spotify through Plex)?
  (Per-provider rows side-step it for now; only matters once 2+ providers are connected.)

### Leave season button is not working
- Which season? all?
- Not broken, never built function, only button is present

### Delete account button is not working
- What gets left behind? Do seasons you joined still show you? Show 'Deleted user'?
- Not broken, never built function, only button is present

### Generate debt button on /admin is misleading and could be improved
- This admin function used to be for recalculating debt for a given week, after some changes happened in the DB in the back end
- It now does this, but also more. It regenerates achievements for example
- A typical use case:
  - User forgets to log exercises
  - Manually fixed in DB by adding exercises, removing debt object, removing ahcivmenent delegations
  - Click generate debt button to "recalculate week"
- Module/function could be remade to a "fix last week button"
  - Add/remove exercises
  - reset achievements/deb for week
  - Recaluclate week
  - Anything else?

### Best effort system
- Manual programming per activity?
- "fastest 5K"...
- Must be calculated at save or during runtime?
- Notification integration for PR?
- PRs for reps and weight on strength exercises?
- Time based best efforts? During this season? During this year?

### AI Ollama feedback on exercises?
Per-exercise feedback in its own dedicated space (not the front-page greeting).
- How to avoid spamming the LMM
- Little model, can the feedback be decent?

### Frontpage hero is wider than other modules on mobile
- Wider looks good on desktop, but should revert to standard width on mobile

### Locked achievements CSS bug
- Achievements with the pad lock icon on /achievements have a rounded border around the icon, like a margin between the icon and the rounded color around the achievement
- SVG get's cut off on the corners because of this

### /exercises builder rework (`/exercises/:id`) — fast-follow
The searchable activity timeline shipped ([docs/exercises.md](exercises.md)), and the session
builder now edits **gear per operation** (each moving activity card has its own selector, with a
session-level "Set gear for all" convenience — see [docs/gear.md](gear.md)). Remaining builder
work:
- The per-activity **aggregate shape** built for `/auth/activities` is exactly what a better
  **session summary header** should consume (activity chips + per-activity metrics) — reuse it
  rather than re-deriving.
- Consider a fuller per-`Operation` card layout (each activity type its own sub-card with its
  own metrics/sets) and clearer affordances for adding a *second activity type* to an existing
  session vs a *second session* to the day. (Gear is already per-operation; this is the
  remaining organisation/metrics work.)
- Watch the media/soundtrack coupling: soundtrack is session-scoped (`Exercise`), so builder
  changes to session time/duration affect the match window (already noted under media).

### /exercises has been refined, let the MCP server benefit
- Users can now more easily find exercises on /exercises
- The MCP server should also be able to find exercises
- Allow MCP to find relevant exercises without shifting through tons of data
- Surface the `CountsTowardGoal` flag (now on `Exercise`, see [docs/data-model.md](data-model.md))
  in MCP exercise/activity output so clients can tell goal-counting sessions from excluded ones

### Gear tracker — possible follow-ups
The gear feature shipped (see [`docs/gear.md`](gear.md)), and per-operation gear editing now
ships too — each moving activity card in the builder has its own gear selector, with a
session-level "Set gear for all" convenience for combined sessions that mix 2+ moving
activities. Open refinements left for later:
- **Auto-assign primary.** The selector *suggests* the user's primary gear for a moving
  activity with no gear, but it isn't persisted until the user interacts. Could auto-assign on
  the first operation instead.
- **Primary per type.** Only one primary per user today; a primary shoe *and* a primary bike
  might be more useful.

### Better gear management
- Or maybe this is finished now that we have a /gear page?
- The modal covers the entire /gear page? Move stuff away from modal? Remove modal?

### Make first day of the week changeable
- Default monday, but choose
- Big changes to logic

### MFA enrollment on /account

## Problems

### Site loads
But sometimes not? Server asleep?