# Integration health

Users connect external services (Plex, Spotify, Audiobookshelf, Strava, Hevy) once and then
forget about them. When a connection breaks, for example because a token is revoked or a
server moves, the sync jobs used to log the error and carry on. The user found out weeks
later that their history had gaps. That is how this started: a revoked Plex token dropped
the soundtrack from new sessions, and after reconnecting, each session had to be re-pulled
by hand.

Integration health does three things:

1. **Notice** that a connection is failing, and tell a rejected credential apart from an
   outage.
2. **Tell the user once**: a push notification when the connection breaks, and a notice
   on the account page while it stays broken.
3. **Catch up**: when the connection works again, re-pull everything missed since it started
   failing.

> **Status:** the shared framework and **Plex** are built. Spotify, Audiobookshelf, Strava
> and Hevy are follow-ups (see [wip.md](wip.md)).

## Model

`models.IntegrationStatus`, one row per (user, provider), and **only while something is
wrong**. No row means the connection is healthy. Rows are hard-deleted on recovery and on
disconnect, which keeps the unique `(user_id, provider)` index free for the next breakage.

| Field | Meaning |
|---|---|
| `Status` | `ok`, `auth_failed` or `unavailable` (constants in `models/integration.go`) |
| `FailingSince` | start of the current run of failures, which is the start of the gap re-pulled on recovery |
| `NotifiedAt` | when the user was told, so they are told once per breakage |

It is a table of its own, not columns on `MediaConnection` or `User`, because the
connections live in different places: Strava and Hevy are on `User`, the media providers on
`MediaConnection`. The health logic is the same for all of them.

## Classifying failures

A provider client wraps the errors that say something about the connection
(`controllers/integration_health.go`):

- `integrationAuthError(msg)` → `errors.Is(err, ErrIntegrationAuth)`: the provider
  **rejected the credential** (401/403). Only the user can fix this, by reconnecting.
- `integrationUnavailableError(msg)` → `errors.Is(err, ErrIntegrationUnavailable)`: the
  provider **didn't answer properly**: network error, timeout, any other non-200, or an
  unparseable reply. It usually fixes itself.

Both keep the provider's own message (`err.Error()` is unchanged), so the warnings shown on
the manual re-pull read the same as before. Any other error (a database write, a decrypt
failure) is **not** a health signal, and the tracker ignores it: it's our problem, not the
connection's.

For Plex, `plexServerStatusError` does the mapping for every PMS reply.

## State machine

`recordIntegrationFailure` / `recordIntegrationSuccess` are called around every
authenticated provider call that matters (for Plex: the history fetch in
`PlexSyncExerciseForUser` and the daily health check).

- **Auth failure** → `auth_failed` at once, and the user is notified.
- **Unavailable** → the start is noted (`FailingSince`), but the status stays `ok` for
  `integrationUnavailableGrace` (24h). Self-hosted servers go down for updates and power
  cuts, and a short outage shouldn't alarm anyone. After the grace period: `unavailable`,
  and the user is notified.
- `auth_failed` beats `unavailable`: a server that also goes quiet doesn't hide that the
  user has to reconnect.
- **Notify once per breakage.** Repeated failures in the same state neither rewrite the row
  nor notify again. Getting worse (`unavailable` → `auth_failed`) notifies again, because
  there is now something the user has to do.
- **Success** after any failure (including a short outage still inside the grace period)
  deletes the row and starts the **recovery backfill** from `FailingSince`.

## Recovery backfill

`recoverIntegration` re-pulls the provider's data from the start of the failure run, capped
at `integrationBackfillLimit` (90 days). For Plex, `plexBackfillSince` re-pulls the
soundtrack of every session **created** since then, with a day of slack. It **ignores the
session-level pull guards** on purpose: `Exercise.MediaRetrievedAt` / `MediaSettled` are
per session, not per provider. So another provider succeeding (or the reconcile cron
retiring a session whose window had closed) used to mark a session done even though Plex
had failed on it, and nothing would ever retry. The backfill is what fixes that.

If the provider fails again part-way, the backfill stops and the row keeps the **original**
gap start (`recordIntegrationFailureSince` only ever moves `FailingSince` earlier). The next
recovery then picks up where this one stopped.

The backfill runs through `startIntegrationRecovery`. That is `goSafely` in production: it
can re-pull months of sessions and must not hold up a request. It is a var only so tests
can run it inline.

Recovery starts from three places:

- **Any successful call** (`recordIntegrationSuccess`), e.g. the next sync or the daily
  check.
- **Reconnect** (`resumeIntegrationAfterReconnect`): completing the Plex PIN flow, or
  saving a server URL that answers the probe.
- **Disconnect** only clears the row (`clearIntegrationStatus`). There's nothing to
  re-pull.

## Daily health check

`IntegrationHealthCheckForAllUsers` runs at 07:15 (`main.go`, gated on `media.enabled` while
Plex is the only tracked provider). It makes one cheap authenticated call per connection, so
a broken connection is noticed even when the user hasn't logged a workout that would have
tripped over it. For Plex that is `GET {server}/library/sections` with the token. A
connection that isn't usable yet (no server or account resolved) is skipped: that's a setup
problem the account page already shows.

## Notifications

`PushNotificationsForAccountAlert` sends with category `account`. The service worker opens
`/account` on click. The text comes from `integrationAlertBody`.

It goes to subscriptions with **`Subscription.AccountAlert`**, a general "account updates"
opt-in for anything the user has to act on about their own account, not just integrations.
It **defaults to on**: the column is `default: true`, so existing rows get it on migration,
and a request without the field (an older cached client) counts as on. Creating a
subscription with it off needs the explicit post-insert write (see
[data-conventions.md](data-conventions.md#gorm-drops-a-false-on-insert-when-the-field-has-default-true)).
The update endpoint treats a missing `account_alert` as "leave unchanged".

## API & UI

`MediaConnectionObject` (`GET /api/auth/media/connections`, and the connect/server
responses) carries `status` (`ok` / `auth_failed` / `unavailable`) and `failing_since`. A
failure still inside the grace period reads as `ok`. The account page shows a notice above
the provider's controls (`integrationAlertHTML` in `web/js/account.js`, styled with
`.integration-alert`): red for `auth_failed`, amber for `unavailable`.

## Adding a provider

1. Wrap the provider client's errors with `integrationAuthError` /
   `integrationUnavailableError` (401/403 and refused refresh grants vs everything else).
2. Call `recordIntegrationFailure` / `recordIntegrationSuccess` around its sync's provider
   call, and its own cheap check from `IntegrationHealthCheckForAllUsers`.
3. Add a backfill to `recoverIntegration` and a display name to `integrationDisplayName`.
4. Call `resumeIntegrationAfterReconnect` from its connect handler and
   `clearIntegrationStatus` from its disconnect handler.
5. Expose `status` / `failing_since` to its account-page section.

## Tests

`controllers/integration_health_test.go` covers error classification, the state machine,
the end-to-end incident (revoked token → one push → account page shows it → reconnect →
gap re-pulled, even after another provider stamped the session as pulled), reconnect via
the PIN flow and the server override, a backfill that fails part-way, and the check's
edge cases. The fault sweeps have `integration health check`, `plex broken connection` and
`plex recovery` scenarios.
