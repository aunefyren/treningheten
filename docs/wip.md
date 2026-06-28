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
  registered in `Migrate()`); `Operation.MediaRetrievedAt` pull guard.
- Read path: `OperationObject.MediaPlayback` / `MediaRetrievedAt`, enriched in
  `ConvertOperationToOperationObject` only when `media.enabled`.
- Idempotent pull primitive (delete-and-replace per operation+provider) + db tests.
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
  creation trigger (`TriggerMediaSyncForOperation`); manual re-pull endpoint
  (`POST /operations/:id/media-sync`) + activity-card 🎧 timeline & re-pull button +
  unit tests.
- **Soundtrack overlay UI (DONE):** activity-card time-rail with elapsed stamps,
  per-track avg-HR (peak-effort highlight) from the Strava streams; placed below the
  HR chart. See `docs/media.md`.
- **Spotify (DONE):** OAuth authorization-code connect (mirrors Strava; shared `/oauth`
  page routed by `state=spotify`), token refresh, `recently-played` fetch + map (public
  album art, multi-artist join), shared `playbackForWindow` matcher; account-page
  Spotify section + mapper/token tests. Needs a registered app (`client_id`/`secret`/
  `redirect_uri`) in config. ~24h history window ⇒ relies on the prompt creation trigger.

**Still to build:**
- **Cross-activity stats:** "most listened" media, "fastest songs" (avg speed over
  `[StartedAt, EndedAt]` from the operation's `OperationSet.StravaStreams`) on the
  statistics page. (Per-track avg-HR on the card already shipped.)
- **Plex artwork:** Plex thumbs need the server token to fetch — store via a proxy
  rather than embedding the credential in a stored URL. (Spotify artwork already works.)
- **Audiobook/podcast classification:** Plex audiobooks surface as `track` → currently
  read as songs; classify from the Plex library section/agent instead.
- **Per-(operation, provider) pull guard:** the single `Operation.MediaRetrievedAt` now
  spans both providers — the auto-trigger pulls all providers once together, which is
  fine for MVP, but connecting a provider *after* an activity was already pulled needs
  the re-pull button. Generalize the guard when this becomes annoying.
- **Edge case (note, not solving now):** editing an activity's **time** changes the
  window — a re-pull would need to re-match.
- **Audiobookshelf** (provider #3): API token + server URL; durable history.

**Open questions:**
- Privacy: listening data is sensitive even self-hosted — any per-activity visibility controls?
- Cross-provider de-dupe if a user has overlapping sources (e.g. casting Spotify through Plex)?
  (Per-provider rows side-step it for now; only matters once 2+ providers are connected.)
- When provider #2 lands: generalize the single `Operation.MediaRetrievedAt` guard to
  per-(operation, provider).

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
