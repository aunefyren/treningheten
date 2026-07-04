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

### Music / audio integration
Connect music/audio providers on your account, then overlay your listening history onto
time-based activities. Full design + decisions now live in [`docs/media.md`](media.md).

**Foundation — DONE (the schema-locking increment):**
- Config flags: tenant `media.enabled` + per-provider `media.plex.enabled`, plus an
  auto-generated `media.token_key` (AES-256-GCM, for credential encryption at rest).
- Data model `MediaConnection` + `MediaPlayback` (`models/media.go`, `database/media.go`,
  registered in `Migrate()`); `Exercise.MediaRetrievedAt` pull guard.
- Read path: `ExerciseObject.MediaPlayback` / `MediaRetrievedAt`, enriched in
  `ConvertExerciseToExerciseObject` only when `media.enabled`.
- Idempotent pull primitive (delete-and-replace per session+provider) + db tests.
- **Session-level soundtrack (DONE):** `MediaPlayback` is keyed to the `Exercise`, not
  the `Operation` — the match window is a single session window, so a multi-operation
  session (manual "run + bench", Hevy) shares one soundtrack instead of duplicating the
  same tracks per operation. Window resolution is `resolveSessionWindow` (pure,
  unit-tested): start = `Exercise.Time`, **skipped** when absent or a date-only midnight
  stamp (a wrong soundtrack is worse than none); duration cascades
  `Exercise.Duration` → Σ operation durations → Σ set times → 2h default. A one-time
  `Migrate()` cleanup drops legacy per-operation rows (re-synced per session).
- **Plex account connection (DONE):** plex.tv PIN flow → encrypted `X-Plex-Token` on
  `MediaConnection`; probe-based reachable-server discovery (`selectReachablePlexServer`,
  TLS-skip for plex.direct); API under
  `/api/auth/media` (list / disconnect / pin create / pin check); account-page Plex
  section (connect / reconnect / disconnect) + helper tests.
- **Plex history pull + matching (DONE):** `controllers/plex_sync.go` —
  `/status/sessions/history/all` fetch, timestamp-overlap matching
  (`buildPlexPlaybackForWindow`, ±5min grace, EndedAt clamped to activity end);
  **audio-only** (`track`; TV/movie plays dropped); **privacy-scoped** to the user's
  **server-local** account id (`resolvePlexServerAccountID` via `{server}/accounts`,
  fails closed if unresolved — never pulls other users' history); async Strava-sync
  creation trigger (`TriggerMediaSyncForExercise`); manual re-pull endpoint
  (`POST /exercises/:exercise_id/media-sync`) + session 🎧 timeline & re-pull button +
  unit tests.
- **Soundtrack overlay UI (DONE):** session-level time-rail with elapsed stamps,
  per-track avg-HR (peak-effort highlight) from a session activity's Strava streams;
  placed below the activity sub-cards. See `docs/media.md`.
- **Spotify (DONE):** OAuth authorization-code connect (mirrors Strava; shared `/oauth`
  page routed by `state=spotify`), token refresh, `recently-played` fetch + map (public
  album art, multi-artist join), shared `playbackForWindow` matcher; account-page
  Spotify section + mapper/token tests. Needs a registered app (`client_id`/`secret`/
  `redirect_uri`) in config. ~24h history window ⇒ relies on the prompt creation trigger.
- **Delayed settle re-pull & reconcile cron (DONE):** two-phase pull via
  `Exercise.MediaSettled bool` — the import-time first pull + one **settle** re-pull once
  the session window has been closed ≥ `mediaSettleWindow` (1h), to catch Spotify's
  delayed `recently-played`. A dedicated hourly **`MediaReconcileForAllUsers`** cron
  (not a Strava side-effect) does first-pull-if-missing (finally covers **manual + Hevy**
  sessions) + the settle pass over a 14-day lookback; the immediate trigger now covers
  **Hevy** (incremental events) as well as Strava (bulk backfill leans on the cron).
  `ReplaceMediaPlaybackForExerciseProvider` now **no-ops on an empty pull** so a
  stale/out-of-window Spotify pull can't erase the first pull's tracks. Manual 🎧 button
  unchanged (bypasses guard, doesn't consume the settle pass). See `docs/media.md`.
- **Audiobookshelf (DONE — provider #3):** self-hosted **server URL + per-user API token**
  connect (no PIN/OAuth), validated via `GET /api/me`; durable history from
  `/api/me/listening-sessions` (inherently user-scoped → no privacy fail-closed); shared
  `playbackForWindow` matcher (`controllers/audiobookshelf.go`). **First provider to
  classify audiobook vs podcast natively** (session `mediaType`), so it lights up the
  typed rail nodes + "minutes listened" metric already in the frontend. Account-page
  section + mapper/classify tests. Config gate `media.audiobookshelf.enabled` (no app
  credentials). V1 caveat: coarse ABS listening sessions match on **start** time, so a
  listen begun before the workout is missed (overlap match is the later refinement).

**Still to build:**
- **Cross-activity stats:** "most listened" media, "fastest songs" (avg speed over
  `[StartedAt, EndedAt]` from a session activity's `OperationSet.StravaStreams`) on the
  statistics page. (Per-track avg-HR on the card already shipped.)
- **Plex artwork:** Plex thumbs need the server token to fetch — store via a proxy
  rather than embedding the credential in a stored URL. (Spotify artwork already works.)
- **Audiobook/podcast classification:** Plex audiobooks surface as `track` → currently
  read as songs; classify from the Plex library section/agent instead.
- **Per-(session, provider) pull guard:** the single `Exercise.MediaRetrievedAt` spans
  both providers — the auto-trigger pulls all providers once together, which is fine for
  MVP, but connecting a provider *after* a session was already pulled needs the re-pull
  button. Generalize the guard when this becomes annoying.
- **Edge case (note, not solving now):** editing a session's **time** changes the
  window — the 🎧 re-pull re-matches on demand, but there's no automatic re-match on a
  time edit. (A skipped date-only session will match once a real time is set + re-pulled.)

**Open questions:**
- Privacy: listening data is sensitive even self-hosted — any per-activity visibility controls?
- Cross-provider de-dupe if a user has overlapping sources (e.g. casting Spotify through Plex)?
  (Per-provider rows side-step it for now; only matters once 2+ providers are connected.)

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

### AI Ollama feedback on exercises?
Per-exercise feedback in its own dedicated space (not the front-page greeting).
- How to avoid spamming the LMM
- Little model, can the feedback be decent?

(Done, separately: the front-page greeting can now occasionally comment on the most
recent workout via the optional `latest_workout` payload block — see `docs/ollama.md`.)

### Better gear management

### Make first day of the week changeable
- Default monday, but choose
- Big changes to logic

## Problems

### Site loads
But sometimes not? Server asleep?
