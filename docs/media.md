# Media / audio integration

Overlay your listening history onto time-based activities: a media **timeline**
matched against the activity's time window (and, where available, the Strava
speed/elevation/HR streams), saved on the activity. This unlocks cross-activity
stats like most-listened media and "fastest songs".

> **Status: WIP.** Done: the data model, config flags, database layer, read-path
> enrichment, the **Plex account connection** (plex.tv PIN flow + encrypted token
> storage + account-page connect/disconnect UI), and the **Plex history pull**
> (server history fetch + timestamp matching, the Strava-sync creation trigger, the
> manual per-activity re-pull endpoint + button, and the activity-card timeline).
> Next increments: cross-activity stats ("most listened", "fastest songs"), artwork,
> and audiobook classification.

## Feature flags

Two levels of gating, mirroring how Strava/Hevy are gated (config-JSON only — no
flags/env wiring):

- **`media.enabled`** — the tenant-wide feature flag for the *whole* integration.
  When off, the read path skips the playback overlay entirely (no extra query per
  operation) and the API surface is inert.
- **`media.plex.enabled`** (and a sibling per future provider) — the per-provider
  gate. A provider is only usable when **both** `media.enabled` **and** that
  provider's own `enabled` flag are true. Each provider is an **independent build**
  because their auth schemes differ.

`media.token_key` is a 32-byte AES-256-GCM key (auto-generated on first run, like
`strava_token_key`) used to encrypt stored provider credentials at rest, via the
shared `utilities.EncryptString`/`DecryptString` helpers (the same scheme Strava
uses). `media.plex.client_identifier` is the stable per-install Plex client id
(`X-Plex-Client-Identifier`), auto-generated on first run — Plex ties authorized
devices to it, so it must stay constant across the PIN create/poll flow and the
install's lifetime.

Spotify uses the OAuth authorization-code flow against a **registered app**, so
unlike Plex it needs credentials in config: `client_id` / `client_secret` from
developer.spotify.com and a `redirect_uri` that is whitelisted in that app and points
at this install's `/oauth` page (e.g. `https://your-host/oauth`). Spotify is only
considered enabled when all three are present (plus the flags).

```jsonc
"media": {
  "enabled": true,
  "token_key": "<generated>",
  "plex": { "enabled": true, "client_identifier": "<generated>" },
  "spotify": {
    "enabled": true,
    "client_id": "<from developer.spotify.com>",
    "client_secret": "<from developer.spotify.com>",
    "redirect_uri": "https://your-host/oauth"
  }
}
```

## Providers

Three providers are in scope, built one at a time: **Plex** and **Spotify** are
implemented; **Audiobookshelf** is still to come.

**Plex** — one Plex connection covers all three media types (music, podcasts,
audiobooks all live in Plex audio libraries), and Plex history is durable (no
Spotify ~24h-window risk). Auth = plex.tv PIN flow → `X-Plex-Token`; history from the
PMS `/status/sessions/history/all?accountID=…`.

**Spotify** — OAuth2 authorization-code flow, scope `user-read-recently-played`.
Cleanest API and gives **public album-art URLs** (so the rail can show artwork, which
Plex can't without a token proxy) and **no privacy-scoping** is needed (the history
is inherently the authenticated user's). The catch: history is limited to the last
~50 tracks / ~24h with no arbitrary past-window query, so the pull must run
**promptly** after the activity — the async creation trigger (fired off the hourly
Strava sync) covers the normal case, but a re-pull on an activity older than ~24h
finds nothing and backfill is impossible.

**Audiobookshelf** (not yet built) — self-hosted, API token + server URL;
listening-sessions endpoint with timestamps + durations; durable history.

## Connection API (Plex)

Routes live under `/api/auth/media` (`controllers/media.go`, `controllers/plex.go`).
The Plex endpoints return **404 when Plex is disabled** (either flag off), so a
disabled provider is indistinguishable from "no such route" — matching the MCP
server's behaviour when off.

- `GET /media/connections` — the user's connections as safe `MediaConnectionObject`s
  (no credentials; a `connected` boolean instead).
- `DELETE /media/:provider` — disconnect a provider. Soft-deletes the
  `MediaConnection`; already-overlaid `MediaPlayback` rows are **left intact**
  (historical facts about past activities, not live credentials).
- `POST /media/plex/pin` — starts the plex.tv PIN flow. Calls
  `POST plex.tv/api/v2/pins?strong=true` and returns `{ pin_id, code, auth_url }`;
  the frontend opens `auth_url` (the `app.plex.tv/auth` page) in a new tab.
- `POST /media/plex/pin/:pin_id/check` — polls the PIN. While unapproved returns
  `{ authorized: false }`; once approved it resolves the account token, fetches the
  account id (`plex.tv/api/v2/user`) and a **reachable** server URL
  (`plex.tv/api/v2/resources`), encrypts the token, and upserts the
  `MediaConnection`. A missing server is **not fatal**: the connection is stored
  without a `ServerURL` and the user can reconnect to retry discovery.

  **Server selection is probe-based** (`selectReachablePlexServer`): there is no
  single correct PMS address — Treningheten may run on the same LAN as Plex (the
  *local* URI is fastest / the only one reachable) or remotely (needs the public or
  relay URI). So it ranks every connection (`rankPlexServerConnections`: non-relay
  first, then **local**, then https, then owned) and **probes each `…/identity`**
  with a short timeout, keeping the first that responds. plex.direct hostnames serve
  a self-signed cert, so TLS verification is skipped (as the official clients do).
  The history pull uses the same TLS handling. *(If a connection was stored before
  probing existed and is unreachable, reconnect to re-pick.)*
- `PUT /media/plex/server` (`APISetPlexServerURL`) — **manual server-URL override**.
  Auto-discovery can't see a reverse-proxy front (e.g. Plex behind **Cloudflare on
  :443** — Plex only advertises the unreachable `:32400` addresses), so the user can
  set the URL they actually reach Plex on. The URL is validated, probed for
  reachability (the result is reported back), and saved regardless so a transient
  probe failure doesn't block them. The account-page Plex section exposes this as an
  editable "server URL" field.

The account page (`web/js/account.js`) renders a Plex section (gated on
`plex_enabled` from the login/config blob) with Connect / Reconnect / Disconnect,
opening the Plex window on connect and polling `…/check` every 3s until approved.

## Connection API (Spotify)

Spotify (`controllers/spotify.go`) mirrors the **Strava** OAuth pattern: the account
page sends the user to `accounts.spotify.com/authorize` (scope
`user-read-recently-played`, `state=spotify`); Spotify redirects back to the shared
`/oauth` page, which reads the `state` and relays the `code` to:

- `POST /media/spotify/callback` (`APISpotifyCallback`) — exchanges the code at
  `accounts.spotify.com/api/token` (HTTP Basic with the app credentials) for access +
  refresh tokens and stores them encrypted on the `MediaConnection`
  (`RefreshToken`/`TokenExpiresAt` are populated; `ServerURL`/`AccountID` stay null).

`spotifyEnsureToken` transparently refreshes the ~1h access token before each pull
(persisting the new token + expiry, and the rotated refresh token when Spotify sends
one). `spotifyFetchRecentlyPlayed` hits `GET /v1/me/player/recently-played?limit=50`;
`buildSpotifyPlaybackForWindow` maps the items (joining multiple artists, taking the
smallest album image as artwork) and matches them with the same shared
`playbackForWindow` Plex uses. No server discovery, no privacy scoping — the history
is the authenticated user's own. Disconnect is the generic `DELETE /media/spotify`.

## Data model

A **queryable table**, not a JSON blob on the operation — cross-activity stats
("most listened", "fastest songs") are trivial over rows and painful over JSON.
Models live in `models/media.go`, data access in `database/media.go`, both
registered in `database.Migrate()`.

### `MediaConnection` — per-(user, provider) account link

A dedicated table rather than piling provider columns onto `User` keeps `User` clean
and makes "build each provider independently" literal. Credential fields are
`json:"-"` (never serialised) and stored encrypted with `media.token_key`.

| Field | Notes |
| --- | --- |
| `Enabled`, `UserID`/`User`, `Provider` | identity; unique per (user, provider) |
| `ServerURL`* | Plex PMS / ABS base; null for Spotify |
| `AccessToken`* | Plex token / Spotify access / ABS token (sensitive) |
| `RefreshToken`*, `TokenExpiresAt`* | Spotify token lifecycle |
| `AccountID`* | Plex account id / ABS user id — filters history to this user |
| `LastSyncedAt`* | |

`MediaConnectionObject` is the safe read shape (no credentials; a `connected`
boolean instead).

### `MediaPlayback` — one row per played item

The timeline plus the stats source. `StartedAt`/`EndedAt` are absolute timestamps
and are the **source of truth** for matching against the activity window and for
stream-based stats ("fastest song" = avg speed over `[StartedAt, EndedAt]` from the
operation's `OperationSet.StravaStreams`).

| Field | Notes |
| --- | --- |
| `OperationID`/`Operation` | the activity anchor |
| `Provider` | provenance |
| `MediaType` | `song` / `podcast` / `audiobook` |
| `Title`, `Artist`*, `Album`* | generic across the three media types (artist/show/author, album/series) |
| `ProviderItemID`* | Plex ratingKey — de-dupe/artwork/deep-link |
| `ArtworkURL`* | |
| `StartedAt`, `EndedAt`* | absolute; `EndedAt` clamped to activity end |
| `TrackLength`* | full length in **seconds** (repo convention), display-only |

### `Operation.MediaRetrievedAt` (`*time.Time`) — pull guard

Mirrors `OperationSet.StravaDataRetrievedAt`; a non-null value distinguishes
"pulled, found nothing" from "never pulled", which drives the re-pull button state.
A single column suffices for the Plex MVP; **generalise to per-(operation, provider)
when provider #2 lands**.

## Attach point & read path

The timeline attaches to the **Operation** (where one Strava activity maps and whose
`OperationSet` holds the speed stream needed for stats); the UI presents it merged at
the session/"run" level — a UI concern, not a data one.

`OperationObject` gains `MediaPlayback []MediaPlaybackObject` and `MediaRetrievedAt`,
enriched in `ConvertOperationToOperationObject` **only when `media.enabled`** (see
`ConvertMediaPlaybackToObjects`). New read paths flatten this Object layer rather
than re-walking raw models.

## Idempotency

**Delete-and-replace per (operation, provider)** on every pull
(`ReplaceMediaPlaybackForOperationProvider`): wipe that operation's rows for the
provider and reinsert. This side-steps a fragile compound de-dupe key (a song can
legitimately repeat within one session) and makes re-pull "just work" — the same
spirit as Strava sync. Other providers' rows on the same operation are untouched.

## Sync flow (implemented)

`controllers/plex_sync.go` holds the pull. `PlexSyncOperationForUser` resolves the
activity window from the operation's exercise (`activityWindowForExercise` — Time +
Duration, with the same fallbacks as `ConvertExerciseToExerciseObject`; a 3h default
when no duration is recorded), fetches the server history for that window
(`plexFetchHistory` → `/status/sessions/history/all`), matches it
(`buildPlexPlaybackForWindow`), and writes via the idempotent delete-and-replace
primitive. It **always stamps `MediaRetrievedAt`** — even on "no server" / "found
nothing" — so the UI can tell pulled-empty from never-pulled.

- **Trigger:** `TriggerMediaSyncForOperation` fires the pull **async** (background
  goroutine) and is wired into the **Strava sync** (`StravaSyncActivityForUser`),
  where the activity time window is fully known. It is a no-op when media is off or
  the user has no connected provider. (Manual operations are created empty — sets
  added afterward — so they rely on the re-pull button instead of a create-time
  trigger.)
- **Re-pull button:** `POST /api/auth/operations/:operation_id/media-sync`
  (`APISyncMediaForOperation`, synchronous — the user waits on the button) re-pulls
  every connected provider and returns the refreshed operation. The exercise-card
  🎧 button (`web/js/exercise.js`) calls it. Covers late connects, transient
  failures, and durable-history backfill.
- **Matching** (`buildPlexPlaybackForWindow`, pure + unit-tested): a history item
  matches when its scrobble time (`viewedAt`) falls in `[start, end]` ± a 5-minute
  grace. `StartedAt` = `viewedAt`; `EndedAt` = `StartedAt + TrackLength`, clamped to
  the activity end when the track started inside the activity.
- **Audio only:** `isPlexAudioListen` keeps only `track` items (music, audiobooks,
  audio podcasts — all `track` in Plex). Video plays (`episode`/`movie`/`clip`, e.g. a
  TV show) are *watching*, not listening, and are dropped. `classifyPlexMediaType`
  currently labels everything audio as `song`; distinguishing audiobooks/podcasts
  needs the item's library section/agent (a later refinement).
- **Privacy — history is scoped to the user.** The PMS history `accountID` is a
  **server-local** id (the owner is usually `1`), *not* the plex.tv global id. At
  connect (and when the server URL is set manually), `resolvePlexServerAccountID`
  maps the plex.tv username/email to the server-local id via `{server}/accounts` and
  stores it as `MediaConnection.AccountID`. History is **always** fetched scoped to
  that id; if it can't be resolved, the sync **fails closed** (stamps the guard,
  stores nothing) rather than pulling every user's plays. Reconnect / re-save the
  server URL to (re)resolve it.
- **Artwork** is not stored yet (the Plex thumb needs the server token to fetch;
  embedding it in a stored URL would leak the credential — deferred to a proxy).
- **Graceful degradation:** activities with Strava streams will get stream-based
  stats; timed activities without streams already show the listening overlay rendered
  on the card.

### "Soundtrack" overlay (frontend)

The card overlay (`mediaTimelineHTML` in `web/js/exercise.js`, styles in
`web/css/workout.css`) renders the listening as a **time-rail** placed at the **bottom
of the activity card** (below the HR chart / map): each track sits on a hairline spine
with an amber node, stamped with how far **into the session** it played
(`started_at − exercise.time`, formatted with the same `secondsToDurationString` the
splits use; falls back to wall-clock time when the session start is unknown). It reuses
the workout card's scoreboard numerals (Saira Semi Condensed) so a track time reads as
just another split, and introduces one reserved accent — `--wv-audio` (warm amber),
distinct from the green performance accent and Strava orange — to mark the audio layer.
The section mark is a static CSS equalizer silhouette (not animated: these are past
sessions, nothing is "now playing"). The re-pull control is the 🎧 section's refresh
button.

**Per-track effort.** When the Strava set carries streams, each track shows the average
**heart rate** over its play window (`streamWindowStats` averages the `heartrate`
channel over `[start, started_at]`, using the stored `time` channel as the elapsed-time
axis; falls back to average pace, then elevation gain). The window is
`[started_at − track_length, started_at]` (a track scrobbles when it finishes), or the
gap since the previous track when the length is unknown. The **hardest-effort** track
(highest average HR) gets a quiet amber highlight — the rail's payoff: which song you
pushed hardest to. This is an early slice of the planned cross-activity stats.

## Open questions

- **Privacy:** listening data is sensitive even self-hosted — per-activity
  visibility controls? (History is already scoped to the connecting user's
  server-local account so other users' plays never leak in.)
- **Cross-provider de-dupe** if a user has overlapping sources (e.g. casting Spotify
  through Plex). Per-provider rows side-step it for now; only matters once 2+
  providers are connected.
- **Edit-time re-match:** editing an activity's *time* changes the window — a re-pull
  would need to re-match (noted, not solved).
