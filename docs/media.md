# Media / audio integration

Overlay your listening history onto your workouts: a media **timeline** matched
against the **session's** time window (and, where available, the Strava
speed/elevation/HR streams), saved on the session. This unlocks cross-activity
stats like most-listened media and "fastest songs".

> **Why the session, not the operation?** A workout session (`Exercise`) can hold
> several activities (`Operation`s) — a manual "run + bench" log, or a Hevy workout
> with many exercises. But the match window is a **single session window** (one start
> time + one duration); an operation carries no absolute start of its own, so it can't
> be placed on a sub-timeline within the session. Attaching the soundtrack per
> operation therefore duplicated the *same* tracks onto every operation. The
> soundtrack is a session-level fact, so it lives on the `Exercise`.

> **Status: WIP.** Done: the data model (session-level), config flags, database
> layer, read-path enrichment, the **Plex**, **Spotify**, and **Audiobookshelf**
> connections, and the **history pull** (server history fetch + timestamp matching, the
> Strava-sync creation trigger, the manual per-session re-pull endpoint + button, and the
> session-level timeline), the first **cross-activity stats** — the `/statistics`
> **Soundtrack** block (most-listened track/artist by play count, songs, unique artists,
> music + spoken-audio time), **Plex audiobook/podcast classification** (by library
> agent/name), and **Plex artwork** (served through an authenticated proxy). The one
> remaining increment is "fastest songs" (stream-windowed avg speed per track).

## Feature flags

Two levels of gating, mirroring how Strava/Hevy are gated (config-JSON only — no
flags/env wiring):

- **`media.enabled`** — the tenant-wide feature flag for the *whole* integration.
  When off, the read path skips the playback overlay entirely (no extra query per
  session) and the API surface is inert.
- **`media.plex.enabled`** (and the siblings `media.spotify.enabled`,
  `media.audiobookshelf.enabled`) — the per-provider gate. A provider is only usable when
  **both** `media.enabled` **and** that provider's own `enabled` flag are true. Each
  provider is an **independent build** because their auth schemes differ.

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

> **Development Mode allowlist (important).** A new Spotify app starts in
> *Development Mode*: only the app owner and up to **25 users the developer adds by
> hand** (dashboard → User Management, by Spotify name + email) may call the Web API.
> Any other user can *log in* fine but their data requests return **403** — the
> symptom is login works, retrieval doesn't. Fix: add the user in the dashboard, or
> request **Extended Quota Mode** (review process) to allow everyone. The code maps
> this 403 to `ErrSpotifyForbidden` with a clear message, and a re-pull surfaces it
> as a warning without discarding a successful Plex sync.

`media.allow_private_targets` controls whether a user-entered provider server URL may
point at a loopback or private-network address — see
[Outbound request safety](#outbound-request-safety) below. It defaults to `true`
(missing-field installs included), because a self-hosted Plex/Audiobookshelf on the LAN is
the normal deployment.

```jsonc
"media": {
  "enabled": true,
  "allow_private_targets": true,
  "token_key": "<generated>",
  "plex": { "enabled": true, "client_identifier": "<generated>" },
  "spotify": {
    "enabled": true,
    "client_id": "<from developer.spotify.com>",
    "client_secret": "<from developer.spotify.com>",
    "redirect_uri": "https://your-host/oauth"
  },
  "audiobookshelf": { "enabled": true }
}
```

Audiobookshelf needs **no app-level credentials** in config — just the on/off flag. The
connection is a per-user **server URL + API token** entered on the account page (the token
comes from the user's ABS account settings), so `audiobookshelf.enabled` is the whole
provider gate.

## Outbound request safety

Plex and Audiobookshelf are the only integrations whose destination is chosen by the
**user** (a server URL they type in) rather than fixed in code. That makes them the app's
SSRF surface, and the artwork proxy makes it a *readable* one — it streams the response
body back to the caller. Three rules apply, all in `controllers/media_dial.go` unless
noted.

**Destinations are checked at dial time, not on the string.** `mediaHTTPClient` builds
clients whose `net.Dialer.Control` hook inspects the IP actually being connected to. A
pre-flight check on the URL would miss DNS rebinding (a name that resolves public when
validated and private when fetched) and would have to be repeated for every redirect hop;
the dial hook covers both for free. `validateMediaServerURL` still runs at connect time,
but only as a courtesy so the user gets a clear message instead of a generic failure —
enforcement is the hook.

One limitation to know: the clients still honour `HTTP_PROXY`/`HTTPS_PROXY`
(`http.ProxyFromEnvironment`), matching what these requests did before. When an egress proxy
is configured the dial goes to the *proxy's* address, so the destination check happens at the
proxy rather than here. That is an operator's deliberate configuration, not something a user
can trigger, but a deployment relying on this policy should not route media traffic through a
proxy.

**Link-local is always refused**, whatever the config says: `169.254.169.254` is the cloud
metadata endpoint and no media server lives there. Same for unspecified and multicast
addresses.

**Loopback and private ranges follow `media.allow_private_targets`**, which defaults to
**true**. This is a deliberate trade: a Plex server on the LAN — or on `127.0.0.1` next to
Treningheten — is the normal self-hosted case, and defaulting to `false` would break it on
upgrade. An instance with users who shouldn't be able to reach its network should set the
flag to `false`. Carrier-grade NAT (`100.64.0.0/10`) counts as private, since that is where
a Tailscale-reachable server sits. Residual risk with the flag on: an authenticated user can
still probe the host's own network. That is recorded as an open item in
[wip.md](wip.md).

**TLS is verified except where it cannot be.** `plexServerClient`
(`controllers/plex.go`) skips verification only for a bare IP, a loopback/mDNS name, or a
`plex.direct` host — addresses no public CA issues for. Every other hostname, which means
anything reached across the internet, is verified normally. It used to be skipped
unconditionally, which carried the user's decrypted `X-Plex-Token` over connections anyone
on the path could impersonate.

**A Plex PIN belongs to the user who created it.** plex.tv scopes a PIN to the
`X-Plex-Client-Identifier` that created it, but that identifier is install-wide here —
shared by every user — and PIN ids are sequential. So `APICreatePlexPin` records the
creator and `APICheckPlexPin` refuses anyone else, reporting a foreign PIN exactly like an
unknown one. The record lives in memory with a 15-minute TTL (`plexPinOwners`): a PIN is
valid for minutes, so persisting it would be schema churn, and the worst case on a restart
mid-flow is that the user clicks Connect again.

## Providers

Three providers are in scope, built one at a time — **Plex**, **Spotify**, and
**Audiobookshelf** are all implemented.

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

**Audiobookshelf** — self-hosted, **server URL + per-user API token** (from the ABS
account settings; no PIN/OAuth). Validated at connect by `GET /api/me` (which also yields
the ABS user id). History from `GET /api/me/listening-sessions` — the `/api/me` endpoint
is inherently scoped to the token's user, so **no fail-closed privacy scoping** is needed
(unlike Plex). History is **durable** (like Plex). Its listening sessions are coarser than
a scrobble (one continuous listen ≈ one book/episode), which is ideal for a spoken-audio
workout. Crucially, **ABS is the first provider that natively distinguishes audiobook vs
podcast** (the session `mediaType` is `book`/`podcast`), so it activates the typed rail
nodes + "minutes listened" metric already built in the frontend.

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

## Connection API (Audiobookshelf)

Audiobookshelf (`controllers/audiobookshelf.go`) is the simplest connect of the three —
no PIN, no OAuth redirect. The account page posts the server URL + API token directly:

- `POST /media/audiobookshelf/connect` (`APIAudiobookshelfConnect`) — validates the URL
  and token by calling `GET /api/me`; on success it encrypts the token and upserts the
  `MediaConnection` (`ServerURL` set, `AccountID` = the ABS user id for reference, no
  `RefreshToken`/`TokenExpiresAt` — ABS tokens are long-lived). Returns 404 when the
  provider is disabled (either flag off), like Plex.

Disconnect is the generic `DELETE /media/audiobookshelf`. The pull
(`AudiobookshelfSyncExerciseForUser`) resolves the session window, fetches
`/api/me/listening-sessions?itemsPerPage=100&page=0` (durable history, most-recent first),
maps each session via `buildAudiobookshelfPlaybackForWindow` (overlap match of
`[startedAt, updatedAt]` through the shared `playbackForWindow`; `mediaType`
`book`→`audiobook`, `podcast`→`podcast`; `TimeListening` as the listened time; the episode
id, else `LibraryItemID`, as the provider item id), and writes via
the idempotent delete-and-replace primitive. Because the history endpoint is `/api/me`
(inherently the token user's), there's **no privacy fail-closed** step. TLS uses default
verification (ABS sits behind the user's own normal certs, unlike Plex's plex.direct).

> **Listened time is clipped to the workout.** ABS listening sessions are coarse — one
> continuous listen, not per-track scrobbles — and `TimeListening` covers the *whole*
> listen, including any before the workout began or after it ended. Stored raw, a podcast
> started before a 60-minute run and another finished after it read as 40 + 49 minutes.
> `listenedWithinWindow` (`controllers/media_match.go`) stores only the in-window share:
> `TimeListening × overlap ÷ wall-clock span`, assuming pauses are spread evenly, capped at
> the overlap (merged duplicate sessions can sum to more listened time than their span). A
> long pause just before the run makes it slightly under-count. The timeline minutes, the `/statistics` spoken-time totals and
> the MCP `track_length_seconds` all read this value.

`spotifyEnsureToken` transparently refreshes the ~1h access token before each pull
(persisting the new token + expiry, and the rotated refresh token when Spotify sends
one). `spotifyFetchRecentlyPlayed` hits `GET /v1/me/player/recently-played?limit=50`;
`buildSpotifyPlaybackForWindow` maps the items (joining multiple artists, taking the
smallest album image as artwork) and matches them with the same shared
`playbackForWindow` Plex uses. `played_at` is logged when the track **stopped**, so it is
treated as the play's end: `scrobbleSpan` places the start one track length earlier (see
the Plex matching bullet). Spotify gives no listened duration — the span is always the
catalogue length, even for a play cut short — and a track paused earlier and only
finalised later (another app taking audio focus) is logged at that later moment, so it
can surface in a workout it was never heard in. No server discovery, no privacy scoping — the history
is the authenticated user's own. Disconnect is the generic `DELETE /media/spotify`.

## Data model

A **queryable table**, not a JSON blob on the session — cross-activity stats
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
boolean instead, plus the connection's health as `status` / `failing_since`. See
[integration-health.md](integration-health.md)).

### `MediaPlayback` — one row per played item

The timeline plus the stats source. `StartedAt`/`EndedAt` are absolute timestamps
and are the **source of truth** for matching against the session window and for
stream-based stats ("fastest song" = avg speed over `[StartedAt, EndedAt]` from a
session activity's `OperationSet.StravaStreams`).

| Field | Notes |
| --- | --- |
| `ExerciseID`/`Exercise` | the session anchor |
| `Provider` | provenance |
| `MediaType` | `song` / `podcast` / `audiobook` |
| `Title`, `Artist`*, `Album`* | generic across the three media types (artist/show/author, album/series) |
| `ProviderItemID`* | the **most specific playable thing** — a podcast *episode*, not its show. The key [overlap-merging](#duplicate-plays--overlap-merging) groups on |
| `ProviderParentID`* | the container: show / artist / album / library item |
| `ProviderGUID`* | the provider's stable cross-install id (Plex agent GUID, Spotify URI) |
| `ProviderSessionID`* | the provider's own record of *this play*, where it keeps one (an ABS listening session) |
| `ArtworkURL`* | |
| `StartedAt`, `EndedAt`* | absolute; `EndedAt` clamped to activity end |
| `StartedBefore` | the item was already playing when the session began, so `StartedAt` is the clamped session start rather than the real start |
| `TrackLength`* | **seconds** (repo convention), display-only: the item's full length for a scrobbled track (Plex/Spotify); the time listened *within the session window* for a listened-time item (ABS) |

**Every provider id is captured, even where nothing reads it yet.** They are one nullable
column each and are effectively impossible to backfill — the provider history endpoints
only reach back hours (Spotify) to days, so an id not stored at pull time is gone. What each
provider supplies:

| Provider | `ProviderItemID` | `ProviderParentID` | `ProviderGUID` | `ProviderSessionID` |
| --- | --- | --- | --- | --- |
| Plex | `ratingKey` | `grandparentRatingKey` (artist/show) else `parentRatingKey` (album) | `guid` (agent id, survives a library rebuild) | — |
| Spotify | `track.id` | `track.album.id` | `track.uri` | — |
| Audiobookshelf | `episodeId` else `libraryItemId` | `libraryItemId` (the show) | — | session `id` |

> **None of these is a unique key, and none should become one.** The same item legitimately
> plays twice in one session, and rows are disposable anyway (delete-and-replace per
> (session, provider)). A DB unique constraint would turn a messy pull into a *failed* pull
> while leaving the matcher producing the wrong rows. Identity is used for **merging**, in
> the matcher — not for rejecting writes, in the schema.

### `Exercise.MediaRetrievedAt` (`*time.Time`) — pull guard

A non-null value distinguishes "pulled, found nothing" from "never pulled", which
drives the re-pull button state. It lives on the **session** (`Exercise`) because the
soundtrack is matched and stored per session, not per operation. It always reflects the
**newest** pull (first, settle, or manual) — it's the "last synced at" the UI reads. A
companion `MediaSettled bool` marks that the single delayed re-pull has run — see
[Delayed settle re-pull & media reconcile cron](#delayed-settle-re-pull--media-reconcile-cron).

## Attach point & read path

The timeline attaches to the **Exercise** (session): the match window is a single
session window, so all of a session's operations share one soundtrack rather than
duplicating it per operation.

`ExerciseObject` gains `MediaPlayback []MediaPlaybackObject` and `MediaRetrievedAt`,
enriched in `ConvertExerciseToExerciseObject` **only when `media.enabled`** (see
`ConvertMediaPlaybackToObjects`). New read paths flatten this Object layer rather
than re-walking raw models.

## Idempotency

**Delete-and-replace per (session, provider)** on every pull
(`ReplaceMediaPlaybackForExerciseProvider`): wipe that session's rows for the
provider and reinsert. This side-steps a fragile compound de-dupe key (a song can
legitimately repeat within one session) and makes re-pull "just work" — the same
spirit as Strava sync. Other providers' rows on the same session are untouched.

**Non-destructive empty guard:** an **empty** pull won't delete existing rows for that
provider (so a stale/out-of-window Spotify pull can't erase the first pull's tracks) — see
[Delayed settle re-pull & media reconcile cron](#delayed-settle-re-pull--media-reconcile-cron).

## Duplicate plays & overlap merging

Providers routinely report **one continuous listen as several records**. Audiobookshelf opens
a fresh listening session on a device switch, an app restart, or a resume after an unclean
close, and does not always close the old one — so a single podcast episode can arrive as three
overlapping sessions. Rendered raw those become three timeline rows for one thing, and any of
them that began before the workout all stack on `00:00` (the display start is clamped to the
session start). Both halves of that were reported as a soundtrack bug.

`coalesceOverlappingEvents` (`controllers/media_match.go`) runs inside `playbackForWindow`,
**before** the window test, so an item split across records is matched and displayed as the
single span it actually was:

- **Identity** is `providerItemID` → `providerGUID` → normalised `title|artist`. The name
  fallback keeps the merge working for a provider that names nothing, at the cost of grouping
  two same-titled items — acceptable, since they still only merge when their spans overlap.
- **Overlap is required, not merely a shared identity.** Replaying an item later in a workout
  is a real second listen and stays its own row. Touching spans join; a real gap does not, so
  a pause and resume still reads as two listens.
- **Merging** keeps the earliest start and latest end, and fills any metadata the first record
  was missing. `trackLengthSec` combines according to `trackLengthIsListened`: time actually
  *listened* (ABS `timeListening`) **adds up** (and is then clipped to the window, see the
  Audiobookshelf section), whereas an item's own length (a Plex/Spotify
  track) takes the **longest** — two overlapping records of one track don't make it twice as
  long.

This is also why `ProviderItemID` must be the **episode**, not the library item: an ABS
library item is the *show*, so keying on it would collapse unrelated episodes of one series
into each other.

Because the merge is in the shared matcher, it covers Plex and Spotify too — Plex can log
overlapping scrobbles for one track across clients.

**Not retroactive on its own.** Rows regenerate from the provider on each pull, so already-
viewed sessions need a re-pull (the 🎧 button, or the admin bulk media re-sync) to pick the
merge up.

## Window resolution

`resolveSessionWindow(exercise, fallbackSeconds)` (pure, unit-tested) computes the
absolute `[start, end]` window **and** an `ok` flag:

- **Start** is `Exercise.Time`. When it's absent, or a **date-only midnight** stamp
  (manual past-day logs write midnight — not a real start of day), `ok = false`: we
  can't place a soundtrack within a day we have no clock time for, and a wrong
  soundtrack is worse than none, so the caller stamps the guard and stores nothing.
  Strava (`StartDate`), Hevy (`startTime`), and same-day manual logs (`now`) all carry
  real clock times, so `ok = true`.
- **Duration** cascade (first non-zero): the explicit `Exercise.Duration`, then the
  sum of the operations' own `Duration`s (`sessionFallbackSeconds`), then the sum of
  every set's logged `Time`, then a **2h** last-resort default.

## Sync flow (implemented)

`controllers/plex_sync.go` holds the pull. `PlexSyncExerciseForUser` resolves the
window with `resolveSessionWindow` (see above), fetches the server history for that
window (`plexFetchHistory` → `/status/sessions/history/all`), matches it
(`buildPlexPlaybackForWindow`), and writes via the idempotent delete-and-replace
primitive. It **always stamps `MediaRetrievedAt`** — even on "no server" / "no
trustworthy time" / "found nothing" — so the UI can tell pulled-empty from
never-pulled.

- **Trigger:** `TriggerMediaSyncForExercise` fires the pull **async** (background
  goroutine) and is wired into the **Strava sync** (`StravaSyncActivityForUser`),
  where the activity time window is fully known (each Strava activity is its own
  session, so it fires once per exercise). It is a no-op when media is off or the user
  has no connected provider. (Manual sessions rely on the re-pull button instead of a
  create-time trigger.)
- **Re-pull button:** `POST /api/auth/exercises/:exercise_id/media-sync`
  (`APISyncMediaForExercise`, synchronous — the user waits on the button) re-pulls
  every connected provider and returns the refreshed session. The session's
  🎧 button (`web/js/exercise.js`) calls it. Covers late connects, transient
  failures, and durable-history backfill.
- **Admin bulk re-sync:** `POST /api/admin/media/sync-for-users`
  (`APISyncMediaForUsers`, mirrors the Strava `sync-activities-for-users` endpoint).
  Body is two optional filters — `user_ids` (empty = every user with a media
  connection) and `exercise_ids` (empty = all of each user's sessions). It runs in the
  **background** (returns `202 Accepted`) and force re-pulls every enabled provider for
  each matched session via `syncMediaProvidersForExercise`, bypassing the
  `MediaRetrievedAt` guard — so it re-matches over a user's whole history, not just
  sessions still owing work. To re-sync only your own history: `{"user_ids":
  ["<your-user-id>"]}`. Safe to re-run (the delete-and-replace primitive is idempotent
  and no-ops on an empty pull). Durable providers (Plex, Audiobookshelf) backfill fully;
  Spotify only returns rows for sessions inside its ~24h window.
- **Matching** (`buildPlexPlaybackForWindow`, pure + unit-tested): `viewedAt` is a
  scrobble — logged when the track **finished** — so `scrobbleSpan` turns it into the play
  span `[viewedAt − TrackLength, viewedAt]` (with no known length, `viewedAt` stands in as
  the start). The item matches when that span overlaps `[start, end]` ± a 5-minute grace,
  so a track that started before the workout and finished in it matches as "already
  playing" (`StartedBefore`, start clamped to the session start). `EndedAt` = `viewedAt`,
  clamped to the activity end when the track started inside the activity. Spotify shares
  these semantics.
- **Audio only:** `isPlexAudioListen` keeps only `track` items (music, audiobooks,
  audio podcasts — all `track` in Plex). Video plays (`episode`/`movie`/`clip`, e.g. a
  TV show) are *watching*, not listening, and are dropped.
- **Type classification (Plex):** Plex stores audiobooks and podcasts as plain music
  `track`s, so the only signal is the **library** the item lives in. Each pull fetches
  `/library/sections` (`plexFetchLibrarySections`, best-effort — a failed lookup leaves
  everything a song rather than sinking the sync) and `classifyPlexSection` (pure,
  table-tested) maps the section's **metadata agent** (audnexus/lazyaudio/audible →
  audiobook, a podcast agent → podcast), falling back to a **keyword in the library name**
  ("Audiobooks"/"Podcasts"), else `song`. The heuristic is setup-dependent by nature —
  agents vary between installs — and defaults to `song` when unsure, since a wrong label is
  worse than a plain dot. (Audiobookshelf classifies natively; Spotify is music-only.)
- **Privacy — history is scoped to the user.** The PMS history `accountID` is a
  **server-local** id (the owner is usually `1`), *not* the plex.tv global id. At
  connect (and when the server URL is set manually), `resolvePlexServerAccountID`
  maps the plex.tv username/email to the server-local id via `{server}/accounts` and
  stores it as `MediaConnection.AccountID`. History is **always** fetched scoped to
  that id; if it can't be resolved, the sync **fails closed** (stamps the guard,
  stores nothing) rather than pulling every user's plays. Reconnect / re-save the
  server URL to (re)resolve it.
- **Artwork (Plex):** the pull stores each row's PMS-relative thumb path (e.g.
  `/library/metadata/…/thumb/…`) in `ArtworkURL`. That path needs the server token to
  fetch, which must never be exposed, so the **read layer** (`resolveMediaArtworkURL` in
  `ConvertMediaPlaybackToObjects`) rewrites a Plex `/library/…` value to an authenticated
  proxy URL — `GET /api/auth/media/plex/artwork?path=…` (`APIGetPlexArtwork`, in the
  image-auth group so an `<img>`/CSS background loads it via the session cookie). The proxy
  validates the path (`plexArtworkPathAllowed`: must match `^/library/[A-Za-z0-9/._-]+$`
  and contain no `..` — Go's HTTP client does not clean dot-segments, and a real thumb path
  never needs cleaning, so anything `path.Clean` would rewrite is rejected rather than
  normalised), then streams the thumb from the
  **requesting user's** Plex server with their decrypted token. Spotify already carries a
  public artwork URL (passed through unchanged); both surface in the `/statistics`
  Soundtrack block. (The exercise rail is a hairline time-spine and shows no thumbnails.)
- **Graceful degradation:** activities with Strava streams will get stream-based
  stats; timed activities without streams already show the listening overlay rendered
  on the card.

## Delayed settle re-pull & media reconcile cron

A single import-time pull isn't enough: only the **Strava** sync triggered it (so Hevy
and manual sessions got no soundtrack automatically), and providers whose history lags —
notably **Spotify**, whose `recently-played` isn't fully populated right after a
workout — aren't captured by one pull fired at import. So there's a **two-phase pull**
plus a dedicated reconcile cron.

- **Two-phase guard via `Exercise.MediaSettled bool`** (alongside `MediaRetrievedAt`),
  read as a small state machine (`reconcileMediaForExercise` in `plex_sync.go`):
  - `MediaRetrievedAt == nil` → **first pull**; stamp `MediaRetrievedAt`.
  - `MediaRetrievedAt != nil && !MediaSettled && windowSettledBy(now)` → **settle pull**;
    flip `MediaSettled = true`.
  - Backfill (session window already closed ≥ `mediaSettleWindow` at first pull) → flip
    `MediaSettled = true` on that first pull; no pointless second pass on history.
  - `resolveSessionWindow` returning `ok = false` (date-only / no clock time) makes
    `windowSettledBy` true, so such a session is settled immediately and retired from the
    reconcile scan.
  - A **boolean, not a timestamp**: the *when* of settling is read by nothing (unlike
    `MediaRetrievedAt`, which the UI shows). `AutoMigrate` backfills existing rows to
    `false`, so each already-pulled session gets exactly one harmless settle pass.
- **`MediaRetrievedAt` is the newest-pull time** — first, settle, and manual pulls all
  bump it; it's the "last synced at" the UI reads.
- **Media reconcile cron — `MediaReconcileForAllUsers`, hourly** (`main.go`, scheduled at
  `:45` when `media.enabled`). A **dedicated** media job, not an orchestrator sequencing
  Strava/Hevy. It lists users with a connection (`GetUserIDsWithMediaConnections`), pulls
  each one's recent sessions still owing work (`GetExercisesForMediaReconcile` —
  never-pulled or unsettled, bounded to `mediaReconcileLookback` = 14 days by
  `created_at`), and runs the state machine: first-pull-if-missing (this is what finally
  covers **manual + Hevy** sessions) plus the settle pass. It operates on already-imported
  exercises, so it has **no ordering dependency** on the Strava/Hevy crons — media is no
  longer a side-effect owned by the Strava sync.
- **Immediate trigger covers Strava + Hevy.** `TriggerMediaSyncForExercise` fires the
  first pull async on import (Strava in `StravaSyncActivityForUser`; Hevy in the
  incremental events sync `HevyEventsSyncForUser`, keyed off the synced workout). Bulk
  Hevy **backfill** intentionally does *not* trigger per workout (it would fan out a
  goroutine each) — it leans on the reconcile cron instead. The trigger also settles a
  backfilled (already-closed) window immediately, so it only owes the cron a settle pass
  for genuinely fresh sessions.
- **Non-destructive empty guard.** `ReplaceMediaPlaybackForExerciseProvider` is per
  (session, provider), so a stale pull only touches *its own* provider's rows — a mixed
  **Plex + Spotify** session keeps Plex intact. But a Spotify pull **outside its ~24h
  window** returns an empty match and would delete the good rows the first pull captured.
  So the primitive **skips the delete-and-replace when the new pull is empty** (returns
  early before deleting). These providers only ever *add* to history (they don't retract
  plays), so an empty result means "nothing new / outside my window," not "authoritatively
  zero." First-pull-empty still stores nothing; later-empty preserves what's there. This
  closes both the settle-too-late and manual-too-late holes with no window arithmetic. A
  genuine "clear this soundtrack" would need an explicit destructive path, not the default.
- **Manual 🎧 button unchanged:** still bypasses the guard and re-pulls every provider
  synchronously (`syncMediaProvidersForExercise`, shared with the trigger + cron). It does
  **not** flip `MediaSettled` — clicking it before the settle window shouldn't consume the
  automatic settle pass (which might catch later data).
- **Settle window: 1h** (`mediaSettleWindow` constant) — inside Spotify's 24h, so the
  automatic settle pass never erases; the manual-days-later case is covered by the empty
  guard.
- **Manual (non-Strava/Hevy) sessions get no create-time trigger** — the hourly reconcile
  cron does their first pull (up to ~1h latency), since only Strava/Hevy imports carry a
  fully-known time window at import.

### "Soundtrack" overlay (frontend)

The overlay (`mediaTimelineHTML` in `web/js/exercise.js`, styles in
`web/css/workout.css`) renders the listening as a **time-rail** placed at the **bottom
of the session** (below all the activity sub-cards): each track sits on a hairline spine
with an amber node, stamped with how far **into the session** it played
(`started_at − exercise.time`, formatted with the same `secondsToDurationString` the
splits use; falls back to wall-clock time when the session start is unknown).

**"Already playing" stamps.** An item that began before the session has its start clamped
to the session start, so its stamp would otherwise read a flat `00:00` — and several such
items would appear to have started together. A row whose `started_before` is set gets a
dimmed stamp with a leading ↑ and a "Already playing when the session started" tooltip
(`.wv-track-time--carried`), so the clamp reads as the approximation it is rather than as
a claim about when the play began. It reuses
the workout card's scoreboard numerals (Saira Semi Condensed) so a track time reads as
just another split, and introduces one reserved accent — `--wv-audio` (warm amber),
distinct from the green performance accent and Strava orange — to mark the audio layer.
The section mark is a static CSS equalizer silhouette (not animated: these are past
sessions, nothing is "now playing"). The re-pull control is the 🎧 section's refresh
button.

**Per-track effort.** When the session has a stream-bearing activity, each track shows
the average **heart rate** over its play window (`streamWindowStats` averages the
`heartrate` channel over `[start, started_at]`, using the stored `time` channel as the
elapsed-time axis; falls back to average pace, then elevation gain). The window is
`[started_at − track_length, started_at]` (a track scrobbles when it finishes), or the
gap since the previous track when the length is unknown. The **hardest-effort** track
(highest average HR) gets a quiet amber highlight — the rail's payoff: which song you
pushed hardest to. The frontend picks the first operation in the session that carries
streams; for a multi-activity session the effort stats are therefore best-effort
(they reflect that one activity's streams). This is an early slice of the planned
cross-activity stats.

**Type & source — shown only where they vary.** A uniform dimension is a default, not
information, so the rail stays as clean as the all-songs case and spends visual weight
only on genuine heterogeneity:

- **Type on the node.** Songs keep the plain amber dot (`.wv-node--song`); podcasts and
  audiobooks get a typed amber glyph on a disc that cuts the spine (`.wv-node--icon`, a
  mic / open-book SVG from `mediaNodeHTML`) — all inside the one `--wv-audio` accent, no
  second colour. **Audiobookshelf** classifies natively (`book`→audiobook,
  `podcast`→podcast) and **Plex** now classifies by library agent/name
  (`classifyPlexSection`), so both render the typed nodes for real; Spotify is music-only,
  so its rows stay dots.
- **Metric adapts to type.** Beats-per-minute is meaningless over a 40-minute talk, so
  spoken rows show **minutes listened** (`mediaSpanStatHTML`, from `track_length` else the
  played span) instead of the stream effort, and never claim the hardest-effort peak.
- **Source tag only when cross-source.** Provider is plumbing, not content, so it's hidden
  in the common single-provider session. When a session spans 2+ providers, each row gets
  the quietest possible `Plex`/`Spotify` tag (`.wv-src`) on the artist line to
  disambiguate interleaved plays. (This can appear today for a user with both Plex and
  Spotify connected; it pairs with the cross-provider de-dupe open question below.) A
  static design exploration of the mixed rail lives in
  [`docs/sketches/soundtrack-mixed.html`](sketches/soundtrack-mixed.html).

## Activity-statistics soundtrack block

The `/statistics` "Activity statistics" section (filter by action + date range,
`APIGetActionStatistics` in `controllers/operation.go`) overlays an aggregated
**Soundtrack** block below the distance/time pills. Because the soundtrack is a
**session-level** fact but that endpoint is operation-centric, the handler collects the
**distinct parent sessions** of the matched operations (`matchedSessionIDs`, keyed on
`OperationObject.Exercise`) and aggregates each session's `MediaPlayback` **once** —
never per matching operation.

`computeActionMediaStatistics` (pure, unit-tested) produces `ActionStatistics.Media`
(`*models.ActionMediaStatistics`):

- **Songs** count, **unique artists**, **music listening time** (Σ `TrackLength`, or the
  `StartedAt`→`EndedAt` span when length is absent), and the **top track** / **top
  artist** by play count (ties broken by title).
- **Spoken audio** (`podcast`/`audiobook`) folds into a single **`SpokenTime`** minutes
  stat rather than the song tallies — a repeated song is interesting, a repeated podcast
  segment isn't.
- **Relevance gate:** the field is `nil` unless `media.enabled` **and** at least one
  matched session had playback, so the frontend (`placeActivityMediaStatistics` in
  `web/js/statistics.js`) renders the block only when there is a real soundtrack. Durations
  hold a **seconds** count (repo convention), read by `secondsToDurationString`. The
  section heading uses the amber audio accent (`.activity-soundtrack-heading`).

"Fastest songs" (stream-windowed avg speed per track) is the remaining cross-activity stat.

## Open questions

- **Privacy:** listening data is sensitive even self-hosted — per-activity
  visibility controls? (History is already scoped to the connecting user's
  server-local account so other users' plays never leak in.)
- **Cross-provider de-dupe** if a user has overlapping sources (e.g. casting Spotify
  through Plex). Per-provider rows side-step it for now; only matters once 2+
  providers are connected.
- **Edit-time re-match:** editing a session's *time* changes the window — the 🎧
  re-pull re-matches on demand, but there's no automatic re-match on a time edit
  (noted, not solved). Notably, a manual past-day session that had no clock time (so
  its soundtrack was skipped) will match once a real time is set and the user
  re-pulls.
- **Per-(session, provider) pull guard:** the single `Exercise.MediaRetrievedAt` spans
  all providers. Fine for the common case, but connecting a provider *after* a session
  was already pulled relies on the 🎧 re-pull button. A provider that *failed* while
  another succeeded is now covered for every media provider: when the connection recovers, the
  [integration-health](integration-health.md#recovery-backfill) backfill re-pulls the gap
  regardless of the guard. Generalise when it becomes annoying.
- **Cross-provider de-dupe detail:** the within-provider overlap merge groups by provider
  id, so it does *not* catch the same play through two providers (two unrelated ids). A
  cross-provider pass would have to match on title/artist + time overlap.
- **Per-operation placement:** the soundtrack is session-level because operations
  carry no absolute start. If per-operation start times ever land (e.g. Hevy exercise
  timestamps), the timeline could be subdivided per activity.
