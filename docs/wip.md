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
time-based activities — a media timeline matched to the existing speed/elevation/HR streams,
saved on the activity, enabling stats like most-listened media and "fastest songs".

**Determined direction:**
- Mirror the Strava integration pattern: per-provider account connection, each gated by its
  own config boolean, each an **independent build** (auth schemes differ per provider).
- Providers in scope: **Spotify**, **Plex**, **Audiobookshelf**.
- Pull listening history *after* the activity and build a media **timeline** (segments:
  media item + started/ended) aligned to the activity's time window. Save it on the activity
  (parallels how `OperationSet.StravaStreams` stores sensor streams).
- Matching is **timestamp overlap** — media `played_at`/session times that fall inside the
  activity window. Segment-level granularity (a track = a time window) is sufficient; no
  per-second needed.
- **Graceful degradation:** activities with Strava streams get the full overlay + stream-based
  stats ("fastest song" = avg pace over the song's window); timed activities without streams
  can still show a plain "what you listened to" list.
- Stats payoff: most-listened media, fastest songs, etc.
- **Attach point (decided):** store the timeline on the **Operation** (where one Strava activity
  maps and whose `OperationSet` holds the speed stream needed for stats). UI presents it merged
  at the "run"/session level — a UI concern, not a data one.
- **Sync trigger (decided): exercise-creation trigger** as the MVP. The common paths are
  creations (Strava sync creates the Operation within ~1h of the workout — inside Spotify's ~24h
  window; manual logging is fresh too), and the time window (start + duration) is already known
  at creation.
  - Fire the pull **async** (background goroutine after the Operation saves), never in the create
    request path — external APIs are slow/flaky; the timeline appears on next load like streams.
  - A creation trigger fires **once**, so pair it with a **manual per-activity "re-pull" button**
    (reuse the Strava re-sync button pattern on the exercise editor) to cover: connecting a
    provider after the activity exists, transient first-pull failures, and Plex/ABS durable-history
    backfill. (Spotify backfill stays bounded by its 24h window.)
  - Edge case (note, not solving now): editing an activity's **time** changes the window — a
    re-pull would need to re-match.

**Provider notes (auth + history source):**
- **Spotify** — OAuth2, scope `user-read-recently-played`. Cleanest API BUT history is limited
  to the last ~50 tracks / ~24h, cursor-based, no arbitrary past-window query. ⇒ sync must run
  **promptly** after the activity (tie to / cadence of the Strava cron) or history is lost.
- **Plex** — `X-Plex-Token` via plex.tv PIN flow + server URL; durable history at
  `/status/sessions/history/all`.
- **Audiobookshelf** — self-hosted, API token + server URL; listening-sessions endpoint with
  timestamps + durations. Lowest friction, most on-brand; durable history.

**Suggested build order:** Audiobookshelf or Spotify first. ABS = simplest auth + durable
history (low-risk, locks the data model); Spotify = headline "fastest songs" payoff but carries
the history-window risk. Leaning ABS first to settle the schema, then Spotify.

**First provider (decided): Plex.** One Plex connection covers all three media types (music,
podcasts, audiobooks all live in Plex audio libraries) so it exercises the full data model in a
single build, and Plex history is durable (no Spotify-window risk). Auth = plex.tv PIN flow →
X-Plex-Token; history from the PMS `/status/sessions/history/all?accountID=…`. Media-type
classification (song/podcast/audiobook) from the Plex library section type/agent is a
Plex-specific build detail, not a data-model concern.

**Data model (decided).** A queryable table, NOT a JSON blob on the operation — cross-activity
stats ("most listened", "fastest songs") are trivial over rows, painful over JSON.

- **`MediaConnection`** — per-(user, provider) account link. A dedicated table rather than
  piling provider columns onto `User` (3 providers × several credential fields each), keeps
  `User` clean and makes "build each provider independently" literal.
  Fields: `Enabled`, `UserID`(idx)/`User`, `Provider` ("plex"|"spotify"|"audiobookshelf"),
  `ServerURL`*(Plex PMS / ABS base; null for Spotify), `AccessToken`*(`json:"-"`, sensitive —
  Plex token / Spotify access / ABS token), `RefreshToken`*(`json:"-"`, Spotify),
  `TokenExpiresAt`*(Spotify), `AccountID`*(Plex account id / ABS user id — filters history to
  this user), `LastSyncedAt`*. Unique index (user_id, provider).
- **`MediaPlayback`** — one row per played item (the timeline + the stats source).
  Fields: `OperationID`(idx)/`Operation` (the activity anchor), `Provider` (provenance),
  `MediaType` ("song"|"podcast"|"audiobook"), `Title`, `Artist`*(artist/show/author),
  `Album`*(album/series), `ProviderItemID`*(Plex ratingKey — de-dupe/artwork/deep-link),
  `ArtworkURL`*, `StartedAt` (absolute; within activity window), `EndedAt`*(StartedAt+length,
  clamped to activity end), `TrackLength`*(full length in **seconds** per the repo convention,
  display-only). Index (operation_id, provider). `StartedAt`/`EndedAt` absolute timestamps are
  the source of truth for matching; "fastest song" = avg speed over `[StartedAt, EndedAt]` from
  the operation's `OperationSet.StravaStreams`. `Artist`/`Album` are generic across the three
  media types.
- **`Operation.MediaRetrievedAt`** *`*time.Time`* — per-activity pull guard mirroring
  `OperationSet.StravaDataRetrievedAt`; distinguishes "pulled, found nothing" from "never
  pulled" (drives re-pull button state). Single column is fine for the Plex MVP; **generalize
  to per-(operation, provider) when provider #2 lands** (small status table, or scope onto
  `MediaConnection`).
- **Idempotency (decided): delete-and-replace per (operation, provider)** on every pull — wipe
  that operation's rows for the provider and reinsert. Avoids a fragile compound de-dupe key
  (a song can legitimately repeat in one session); re-pull just works, same spirit as Strava
  sync.
- **Read path:** `OperationObject` gains `MediaPlayback []MediaPlayback`, enriched in
  `ConvertOperationToOperationObject` (Object-layer convention). New models go in `models/media.go`
  + `database/media.go`; register tables in `Migrate()`.

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

## AI Ollama feedback on exercises?
Per-exercise feedback in its own dedicated space (not the front-page greeting).
- How to avoid spamming the LMM
- Little model, can the feedback be decent?

(Done, separately: the front-page greeting can now occasionally comment on the most
recent workout via the optional `latest_workout` payload block — see `docs/ollama.md`.)

### Make first day of the week changeable
- Default monday, but choose
- Big changes to logic

## Problems

### Site loads
But sometimes not? Server asleep?
