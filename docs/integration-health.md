# Integration health

Users connect external services (Plex, Spotify, Audiobookshelf, Strava, Hevy) once and then
forget about them. When a connection breaks, for example because a token is revoked or a
server moves, the sync jobs used to log the error and carry on. The user found out weeks
later that their history had gaps. That is how this started: a revoked Plex token dropped
the soundtrack from new sessions, and after reconnecting, each session had to be re-pulled
by hand.

Integration health does three things, for every integration:

1. **Notice** that a connection is failing, and tell a rejected credential apart from an
   outage.
2. **Tell the user once**: a push notification when the connection breaks, and a notice
   on the account page while it stays broken.
3. **Catch up**: when the connection works again, re-pull everything missed since it started
   failing.

## Per provider

| Provider | Rejected credential (`auth_failed`) | Detected by | Recovery |
|---|---|---|---|
| Plex | PMS 401/403; or no server / account id resolved (reason `setup_incomplete`) | each pull + daily check | re-pull sessions since the gap; a moved server is found again automatically |
| Spotify | token endpoint 401 or 400 `invalid_grant`; history 401; history 403 (reason `not_allowlisted`) | each pull + daily check | re-pull sessions since the gap, capped at Spotify's ~24h history |
| Audiobookshelf | 401/403 | each pull + daily check | re-pull sessions since the gap |
| Strava | token exchange 400/401 (`ErrStravaSessionInvalid`) | hourly sync | re-sync every week since the gap |
| Hevy | 401/403 (key regenerated, or PRO lapsed) | hourly sync | none needed: the events baseline only moves on success |

Anything else that comes from the provider (network error, timeout, other non-200, an
unparseable reply) is an **outage** (`unavailable`). Strava's 429 is our own rate limit and
counts as neither.

## Model

`models.IntegrationStatus`, one row per (user, provider), and **only while something is
wrong**. No row means the connection is healthy. Rows are hard-deleted on recovery and on
disconnect, which keeps the unique `(user_id, provider)` index free for the next breakage.

| Field | Meaning |
|---|---|
| `Status` | `ok`, `auth_failed` or `unavailable` (constants in `models/integration.go`) |
| `Reason` | refines the status where the generic advice ("reconnect") would be wrong: `not_allowlisted` (Spotify Development Mode, so the admin must add the user) or `setup_incomplete` (Plex never resolved a server or account). Empty otherwise |
| `FailingSince` | start of the current run of failures, which is the start of the gap re-pulled on recovery |
| `NotifiedAt` | when the user was told, so they are told once per breakage |

It is a table of its own, not columns on `MediaConnection` or `User`, because the
connections live in different places: Strava and Hevy are on `User`, the media providers on
`MediaConnection`. The health logic is the same for all of them.

## Classifying failures

Provider clients wrap the errors that say something about the connection
(`controllers/integration_health.go`):

- `integrationAuthError(msg)`, or `integrationAuthErrorFor(sentinel, reason)` to keep a
  provider sentinel such as `ErrStravaSessionInvalid` / `ErrSpotifyForbidden` reachable
  through `errors.Is` → `ErrIntegrationAuth`.
- `integrationUnavailableError(msg)` → `ErrIntegrationUnavailable`.

The provider's own message is kept (`err.Error()` is unchanged), so the warnings on the
manual re-pull read the same as before. Callers that wrap further use `%w`. Any other error
(a database write, a decrypt failure, our own rate limit) is **not** a health signal, and
the tracker ignores it.

The status mappers are `plexServerStatusError`, `absStatusError`, `stravaStatusError` and
`spotifyTokenRejected`, plus the inline switch in `spotifyFetchRecentlyPlayed` and
`hevyAPIGet`.

## State machine

`recordIntegrationFailure` / `recordIntegrationSuccess` / `recordIntegrationOutcome` are
called around each sync's provider call and from the daily check.

- **Auth failure** → `auth_failed` at once, and the user is notified.
- **Unavailable** → the start is noted (`FailingSince`), but the status stays `ok` for
  `integrationUnavailableGrace` (24h). Self-hosted servers go down for updates and power
  cuts, and a short outage shouldn't alarm anyone. After the grace period: `unavailable`,
  and the user is notified.
- `auth_failed` beats `unavailable`: a server that also goes quiet doesn't hide that the
  user has to act.
- **Notify once per breakage.** Repeated failures in the same state neither rewrite the row
  nor notify again. A change in what the user has to do (status or reason) notifies again.
- **Success** after any failure (including a short outage still inside the grace period)
  deletes the row and starts the **recovery backfill** from `FailingSince`.

## Recovery

`recoverIntegration` dispatches per provider, from `FailingSince` capped at
`integrationBackfillLimit` (90 days). It runs through `startIntegrationRecovery`, which is
`goSafely` in production because a backfill can take a while. It is a var only so tests can
run it inline.

- **Media** (`mediaBackfillSince`, `controllers/media_health.go`) re-pulls every session
  **created** since the gap began, with a day of slack, and **ignores the session-level pull
  guards** on purpose. `Exercise.MediaRetrievedAt` / `MediaSettled` are per session, not per
  provider, so another provider succeeding used to mark a session done even though this one
  had failed, and nothing would retry it. Spotify is capped at `spotifyHistoryWindow` (24h):
  older sessions would only ever find nothing.
- **Strava** (`stravaBackfillSince`) re-syncs each week from the gap's start up to (not
  including) the current week, which the hourly sync covers anyway. The shared rate limiter
  sets the pace.
- **Hevy**: nothing. The events sync only moves `HevyLastSync` forward after a run that
  succeeded, so the first good run already covered the gap.

If the provider fails again part-way, the backfill stops and the row keeps the **original**
gap start (`recordIntegrationFailureSince` only ever moves `FailingSince` earlier). The next
recovery then picks up where this one stopped.

Recovery starts from:

- **Any successful call** (`recordIntegrationSuccess`): the next sync, or the daily check.
- **Reconnect** (`resumeIntegrationAfterReconnect`): the Plex PIN flow, a Plex server URL
  that answers, the Spotify callback, the Audiobookshelf connect. Strava's reconnect runs a
  sync straight away, so its success does the same. A new Hevy key clears the status and its
  backfill re-imports everything.
- **Disconnect** only clears the row (`clearIntegrationStatus`). There's nothing to re-pull.

## Plex self-repair

`plexCheckConnection` tries to fix the connection before reporting it:

- **Moved server.** When the stored URL stops answering, `plexRediscoverServer` asks plex.tv
  for the user's servers again, probes them, and switches to the first that answers (with
  its server-local account id). This also replaces a URL the user set by hand, but **only
  with one that answers**: a working address always beats a dead one. A reverse-proxy setup
  whose advertised addresses are all unreachable never finds a candidate, so it is left
  alone.
- **Unfinished setup.** A connection without a server or account id gets the same lookup.
  If that still finds nothing, it is `auth_failed` with reason `setup_incomplete`. The pull
  (`PlexSyncExerciseForUser`) used to skip such a connection without telling anyone; it now
  records the same status.

## Daily health check

`IntegrationHealthCheckForAllUsers` runs at 07:15 (`main.go`, gated on `media.enabled`). It
makes one cheap authenticated call per media connection (`MediaHealthCheckForProvider`), so
a broken one is noticed even when the user hasn't logged a workout that would have tripped
over it:

- Plex: `GET {server}/library/sections`, after the self-repair above.
- Spotify: refresh the token if it is due, then read recently-played.
- Audiobookshelf: `GET /api/me`.

Strava and Hevy aren't in it: their hourly syncs make the same call anyway, for every
connected user.

## Notifications

`PushNotificationsForAccountAlert` sends with category `account`. The service worker opens
`/account` on click. The text comes from `integrationAlertBody` (provider, status, reason).

It goes to subscriptions with **`Subscription.AccountAlert`**, a general "account updates"
opt-in for anything the user has to act on about their own account. It **defaults to on**:
the column is `default: true`, so existing rows get it on migration, and a request without
the field (an older cached client) counts as on. Creating a subscription with it off needs
the explicit post-insert write (see
[data-conventions.md](data-conventions.md#gorm-drops-a-false-on-insert-when-the-field-has-default-true)).
The update endpoint treats a missing `account_alert` as "leave unchanged".

## API & UI

- Media connections: `MediaConnectionObject` carries `status`, `status_reason` and
  `failing_since`.
- Strava / Hevy: the user's **own** `GET /api/auth/users/:id` carries
  `integration_health: {strava: {...}, hevy: {...}}`, the same three keys, for the
  connected and enabled ones only (`userIntegrationHealth`). It is a `gorm:"-"` field on
  `models.User` and is not on `PublicUser`.

A failure still inside the grace period reads as `ok`. The account page renders a notice
above each provider's controls (`integrationAlertHTML` in `web/js/account.js`, styled with
`.integration-alert`): red for `auth_failed`, amber for `unavailable`, with reason-specific
wording. A broken Strava connection also gets a **Reconnect Strava** button.

## Adding a provider

1. Tag the provider client's errors with `integrationAuthError` /
   `integrationUnavailableError` (or `integrationAuthErrorFor` to keep a sentinel / reason).
2. Call `recordIntegrationFailure` / `recordIntegrationSuccess` (or
   `recordIntegrationOutcome`) around its sync's provider call. For a media provider, add it
   to the switches in `media_health.go` (enabled, sync, check), which also puts it in the
   daily check.
3. Add its recovery to `recoverIntegration` and a display name to `integrationDisplayName`.
4. Call `resumeIntegrationAfterReconnect` from its connect handler and
   `clearIntegrationStatus` from its disconnect handler.
5. Expose its health to its account-page section and call `integrationAlertHTML`.

## Tests

- `controllers/integration_health_test.go`: classification, the state machine, and the
  original Plex incident end to end (revoked token → one push → account page → reconnect →
  gap re-pulled, even after another provider stamped the session as pulled), plus reconnect
  paths and a backfill that fails part-way.
- `controllers/integration_providers_test.go`: Plex finding a moved server and finishing an
  incomplete setup; Spotify classification, the allowlist case and the 24h backfill cap;
  Audiobookshelf; Hevy; Strava's week backfill.
- Existing Strava tests now assert "kept and marked broken" instead of "cleared".
- The fault sweeps have `integration health check`, `media health check`,
  `plex broken connection`, `plex moved server`, `plex recovery` and `strava recovery`
  scenarios.
