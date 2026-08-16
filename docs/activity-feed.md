# Front-page activity feed

The **Activities** module on the front page shows this week's sessions from the people
you train alongside. It is a glanceable weekly digest — the searchable archive of your
*own* activities is [`/exercises`](exercises.md), which is a different feature backed by
a different endpoint.

## Who you see: co-membership, not friendship

Treningheten has **no friend-request system**, and deliberately so — a self-hosted
instance is a group who already know each other, and an approval flow would be
machinery for a relation the data already encodes.

Instead, **sharing a season with someone is the relation.** If you and another user have
both held a `Goal` in the same `Season`, at any point, you are peers and see each
other's activities. Peers are resolved in one self-join over `goals`
(`database.GetSeasonPeerUserIDsForUser`), which also returns the user themselves so
their own sessions appear in their own feed.

Two consequences worth being explicit about:

- **Past seasons count.** The season need not be ongoing. Someone you played one season
  with years ago stays a peer, and the relation does not expire.
- **The feed outlives the season.** Because it is not season-scoped, the module keeps
  working between seasons and for a user who has not joined one yet (they will see their
  own activities). Previously the module simply vanished in the gap between seasons.

A **disabled** `Goal` (a withdrawn membership) creates no relation, in either direction.

## Consent

Appearing in anyone's feed still requires the session owner's **`User.ShareActivities`**
opt-in (default on) — being a peer is necessary but not sufficient. The `/account`
checkbox states the audience plainly: *"Visible to everyone you have shared a season
with, including past seasons."*

Strava links are governed separately by the owner's **`User.StravaPublic`** (default
on); when it is off, the activity still appears but carries no Strava ids.

Consent also works **per session**: a session flagged `Exercise.Private` is dropped from
the feed entirely, for every viewer including its owner. That is the account-wide
`ShareActivities` switch's fine-grained counterpart — one session hidden rather than all
of them — and it is what a private Strava activity imports as (see
[strava.md](strava.md#activity-privacy)). A private session still counts toward the goal,
the season streak and the leaderboard; it hides *what* you did, not *that* you trained.

This was a deliberate, discussed **widening** of an existing setting rather than a new
consent surface: a user who opted in when the feed meant "my current season" now has a
larger audience. The alternative considered — turning `ShareActivities` into a
three-way scope (off / season-mates / all peers) — was rejected as more setting than the
situation warrants. If that changes, the scope belongs on the existing field, not in a
friend graph.

## Shape

`GET /api/auth/activities/shared` (`controllers.APIGetSharedActivities`):

1. Resolve the current Mon–Sun week (`utilities.FindEarlierMonday` / `FindNextSunday`).
2. Resolve the caller's peer ids — one query.
3. Fetch enabled exercise days in the window for those users who have
   `share_activities = 1` (`GetExerciseDaysForSharingUsersInListUsingDates`) — one
   query. An empty peer list returns nothing, never everyone.
4. Flatten to `models.Activity` via `buildActivitiesFromExerciseDays`, newest first,
   capped at **50** (`sharedActivityFeedLimit`).

The flattening step is shared with the season-scoped
`GET /api/auth/seasons/:season_id/activities` (`APIGetCurrentSeasonActivities`), which
still exists for a season-specific view, and with the profile feed
`GET /api/auth/users/:user_id/activities` (`APIGetUserActivities`). It applies the
`IsOn`/`Enabled`/`Private` filter, the `StravaPublic` gate, and the
general-"Workout"-action fallback so an activity is never actionless. Every social
surface goes through this one builder on purpose — a visibility rule added in one place
must not be missing from another.

The previous implementation resolved membership with a `GetGoalFromUserWithinSeason`
call **per candidate user** (memoised in a slice); the peer join replaced that N+1 with
two queries, so the widened audience costs less than the narrow one did.

## Frontend

`web/js/frontpage.js` — `getActivities()` takes no season and is called once the
front page loads, outside the ongoing-season branch. In the no-season path the page
strips `#current-week` and `#leaderboard` but keeps `#activities`, so the module renders
either way; an empty response leaves its "No public activities yet this week..." state.

## Related

- [exercises.md](exercises.md) — the personal, searchable activity timeline.
- [seasons-and-goals.md](seasons-and-goals.md) — what a `Goal` is and how membership works.
