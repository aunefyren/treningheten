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

### Gear tracker
- Allow to track gear (like shoes) either on workout builder or on account (or somewhere else)
- Sync over Strava gear, keep them in sync

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

### Frontpage performance & cleanup
Analysis of `web/html/frontpage.html` + `web/js/frontpage.js` and the API calls they drive.
Tackle in the priority order below, one item at a time. Mark items done by removing them.

A logged-in load fires ~`9 + N + M` requests (N = current-week users, M = activities),
re-fetching the same user images 2–4× and recomputing the whole season server-side.

**Determined plan (priority order):**

_All determined items done — see below. Remaining work is in the open topics._

**Done:**

- ~~Client-side image dedupe/cache.~~ `getProfileImageCached` in `frontpage.js` now fetches each
  user's thumbnail once (shared in-flight requests + `{userID → base64}` cache), so the same user
  appearing across current-week / activities is no longer re-fetched. Cuts a load from
  `9 + N + M` requests toward `9 + uniqueUsers`.
- ~~Frontpage bug fixes & cleanup.~~ Fixed in `frontpage.js`/`functions.js`: `place_week` now gets
  `user_id` after a save; `userDisplayName` guards `userList` misses (former members no longer
  crash the board); leaderboard sorts by completion (not random UUID); `activateCountdown` finish
  handler targets `countdown_number_<seasonID>`; `get_image` `=!`→`!=` typo; removed dead
  `GetProfileImageForUserOnLeaderboard`/`Place…` pair and the commented-out "no seasons" block;
  `place_current_week`/`place_leaderboard` build their HTML once instead of `+=` in a loop.
- ~~Characterization tests for week-result logic.~~ `controllers/season_test.go` pins
  `RetrieveWeekResultsFromSeasonWithinTimeframe` behaviour (streak accumulation, week completion,
  full-week participation, sick-leave preservation) via a `controllers`-package in-memory SQLite
  harness (`controllers/setup_test.go`).
- ~~Batch the `/weeks` computation.~~ `seasonWeekData` (`controllers/season.go`) pre-fetches a
  season's exercises/debts/sick-leave in ~4 bulk queries and buckets them by (user/goal, ISO week),
  replacing ~3·W·U per-week queries (≈780 → ~4 for a 26-week/10-user season). New bulk accessors in
  `database/` (covered by `database/weekdata_test.go`); behaviour stays pinned by
  `controllers/season_test.go`. The B1 cache was skipped — batching alone collapsed the cost, and
  the now-dominant remaining cost on the endpoint is `ConvertSeasonToSeasonObject` (see the
  lighter-endpoints open topic).
- ~~Serve images as raw bytes + server-side caching.~~ Profile/achievement image endpoints now
  return raw `image/jpeg` (cookie-or-header auth via `middlewares.AuthImageReadOnly`), consumed
  directly via `<img src>` across all pages (frontpage/seasons/exercises/user/account/wheel/
  achievements) using `profileImageURL`/`achievementImageURL` helpers — the XHR→base64 plumbing
  is gone. Caching: HTTP `Cache-Control`/`ETag` + an in-memory resized-image cache
  (`loadResizedImageCached`) that self-invalidates on file modtime. Docs: `docs/image-serving.md`.
  This subsumed both the old client-dedupe and the planned server thumbnail-cache step.

**Open topic (needs design decision before implementing):**

- **Lighter list endpoints / fewer season scans.** `get-on-going`, `?potential=true`, and
  `?countdown=true` each scan all enabled seasons and run full `ConvertSeasonToSeasonObject`
  (resolving every goal's user/sickleave/prize) — three full conversions per load. Countdown/
  potential lists only need id/name/start/end/join_anytime + membership. Consider lighter DTOs
  or a shared per-request season context. Also consolidate `APIGetSeasonWeeks` vs.
  `APIGetCurrentSeasonLeaderboard` (near-duplicate logic). Note: now that the `/weeks` walk is
  batched, `ConvertSeasonToSeasonObject` (~2·U queries) is the dominant remaining cost on that
  endpoint too.

## Problems

### Site loads
But sometimes not? Server asleep?

### Modal on exercise building page must be updated to match exercise builder

### If you have a long name (perhaps emoji as well) you can break the wheel look
Name gets placed outside inside of inside the slice

### Activity heatmap issues
- Spot based coloring, probably the GPS cords given by the device. Could this be smoothed out / replaced with a line?
- Is coloring based on frequency? A single run in a single place should not be red if there are a 100 other runs elsewhere