# Work in progress
Plans, ideas, problems and bugs. Finished work does not live here — move it into the
relevant `docs/*.md` (or a new one) when it lands, and delete the item. A feature that is
part-built can get its own doc and be updated over time.

Security items keep their `S<n>` numbers from the 2026-08-29 security / 2026-09-03 privacy
audits so commit messages keep resolving. Shipped fixes, accepted risks and the
"checked and found sound" list live in [`security.md`](security.md).

## Security — open

### S10 — LOW: DB and SMTP passwords passed as command-line arguments
`entrypoint.sh` builds `CMD` from env vars, including `--dbpassword` and `--smtppassword`,
then `exec $CMD`. The secrets land in `/proc/<pid>/cmdline`, readable by any process in the
container; and `$CMD` is **unquoted**, so a password containing whitespace splits into
several arguments and one containing e.g. `--generateinvite true` injects a flag.

**Fix:** have the binary read these from the environment directly, and quote/array-ify the
invocation.

### S12 — LOW: Go toolchain behind on security patches
At the 2026-08-29 audit `govulncheck` reported 16 reachable stdlib vulnerabilities
(html/template escaper bypasses via Gin rendering, h2c `ReadHeaderTimeout`, an HTTP/2
transport loop reached from the Strava client). The Dockerfile has since moved to
`golang:1.26.0-alpine`, but that is still not a current patch, the CI matrix still tests
`1.24.x`/`1.25.x`, and there is no `govulncheck` step.

**Fix:** bump the Dockerfile base and CI matrix to a current patch release, and add
`govulncheck ./...` to `.github/workflows/go.yml` so it can't drift again.

### CORS is wide open — and inert by accident
`main.go` (`router.Use(cors.New(…))`, ~line 414) sets `AllowOrigins: ["*"]` **with**
`AllowCredentials: true` and an `AllowOriginFunc` returning `true`. It only doesn't bite
because the `Use` runs *after* every `/api` group, `/mcp` and `.well-known` route is
registered, and Gin freezes a route's chain at registration — so the API sends no CORS
headers at all. Moving the call up, or adding a route below it, silently turns on a
wildcard-with-credentials policy.

**Decide:** same-origin only → delete the middleware; third-party browser clients wanted →
explicit origin allow-list, moved above the groups deliberately.

### Refresh token lives 30 days in a JS-readable cookie
`web/js/functions.js` stores the access (7d) and refresh (30d) tokens via `document.cookie`
with `samesite=strict` — no `Secure`, and necessarily no `HttpOnly`. Any XSS is a 30-day
account compromise (this is what turned S3 into a takeover).

**Options**, increasing work: add `Secure`; move the refresh token to a server-set
`HttpOnly; Secure; SameSite=Strict` cookie that `/api/oauth/token` reads for the
`refresh_token` grant; or accept the risk and rely on escaping plus a CSP. Decide before
adding more `innerHTML` render paths.

### `admin` scope is grantable to dynamically registered clients
DCR (`POST /api/oauth/register`) is now rate limited (5/hour/IP, see `security.md`), but
still unauthenticated, and `SupportedScopes` includes `admin` (`models/oauth.go`) — a
third-party client can request it and get it if an admin approves consent.

**Decide:** reserve `admin` for the first-party client and PATs? Gate DCR behind a config
flag (off unless MCP is in use)? Prune unused clients?

### No security response headers
No `Content-Security-Policy`, `X-Content-Type-Options`, `X-Frame-Options`/`frame-ancestors`
or `Strict-Transport-Security`. A CSP is the structural mitigation for the `innerHTML` bug
class, but inline `onclick=` handlers are used throughout (`web/js/account.js`,
`web/js/exercise.js`, …), so a strict policy needs those refactored first. Scope as its own
task rather than bolting on a weak policy. The cheap headers (`nosniff`, frame-ancestors)
could land independently.

## In progress

### Postgres support
Decided: make it work (it was configurable but never functional). `files/config.go`
rewrites `db_type: postgres` to `mysql`, and the data layer is MySQL-flavoured throughout:
~617 backtick-quoted identifiers, ~250 integer-for-boolean comparisons/updates
(`enabled = ?", 1`, `Update("used", 1)`), five `type:longtext` columns, and an invalid
`sslmode` value in the connect branch. Plan: drop identifier quoting (except the reserved
`seasons.end`), use `true`/`false`, drop the `longtext` tags, accept `postgres` in config
and fail on unknown types, run the database suite against Postgres in CI, and add the
Docker harness profile. The legacy SQL-dump importer in `utilities/migrate.go` stays
MySQL-only.

## Decisions taken — no action

- **Goals stored as competing before the fix** stay as they are; nothing records what the
  member chose. Fix by hand if a member raises it.

## Plans & ideas

### Onboarding flow
- Small: an invite link that autofills the invite code.
- Bigger: the invite link starts an onboarding flow, no navbar:
  - user registration with the invite code from the URL
  - season registration, if the invite carries a season context
  - optional: choose which activities count as exercise (many don't count walks)
  - optional: connect Strava or Hevy
  - optional: enable notifications
- Breaking out should return you to where you were without re-entering anything.
- Can it reuse existing forms/pages, or does it need a new page?

### Leave season / delete account
Both are button stubs on `/account`, never built.
- **Leave season:** which season — one, or all?
- **Delete account:** what is left behind? Do joined seasons still show you, as "Deleted
  user"?

### "Generate debt" on /admin → "fix last week"
The button now recalculates more than debt (achievements too). Typical use: a user forgot
to log, the admin adds exercises and removes the debt and achievement delegations in the
DB, then clicks the button. Could become a proper "fix last week" tool: add/remove
exercises, reset achievements/debt for the week, recalculate — anything else?

### Sick leave is per season/goal
Different seasons having different sick leave makes sense; being in several seasons at once
but only able to use sick leave on one goal doesn't.

### Flexible workouts
Extra effort one week carries over. Season-specific setting; configurable how many carry
over and how long before they decay. Must be understandable in the UI.

### Make the first day of the week configurable
Default Monday. Big changes to the week logic, which (like seasons) works in the server's
zone — see `seasons-and-goals.md`.

### Best-effort system
- "Fastest 5K" etc. — per-activity programming? Computed at save or at runtime?
- Time-boxed: this season, this year? PR notifications? Rep/weight PRs for strength?
- First increment: **best split** (fastest 1 km/mile) — the per-distance segments and
  processed stream summary already exist (see [mcp.md](mcp.md)).
- Later: grade-adjusted pace (GAP) and VAM.

### Front page activities: tag a partner
- Tag a partner/group on a session in the builder.
- Show as a joint activity in the season feed if both are in the season.
- Sync Strava's partner tag? Auto-detect from data?

### More workout tags
E.g. Easy, Splits. Must respect Strava sync.

### Session builder (`/exercises/:id`) fast-follow
- A **session summary header** — consume `models.SessionFeedItem` (returned by
  `/auth/activities`) rather than re-deriving totals.
- A fuller per-`Operation` card layout (own metrics/sets), and clearer affordances for adding
  a *second activity type* to a session vs a *second session* to the day.
- Soundtrack is session-scoped, so builder changes to session time affect the media match
  window (see [media.md](media.md#open-questions)).

### Gear follow-ups
- **Auto-assign primary:** the selector suggests the primary gear but doesn't persist it
  until the user interacts. Could auto-assign on the first operation.
- **Primary per type:** one primary per user today; a primary shoe *and* bike might be
  better.
- **Gear management UI:** is `/gear` done? The modal covers the whole page — move content
  out of the modal, or remove it?

### Private sessions follow-ups
- **Hevy has no privacy source** — workouts import visible. Does the Hevy API expose a
  per-workout visibility field?
- **MCP doesn't expose `private`** — deliberate (MCP is self-scoped). Revisit if an LLM
  client needs it.
- **Bulk privacy** (hide a week, hide-by-default) — probably not wanted.

### Media / audio
Open items live in [`media.md`](media.md#open-questions). The one remaining planned
increment is **"fastest songs"**: avg speed over `[StartedAt, EndedAt]` from the session's
`OperationSet.StravaStreams`, which needs per-track stream windowing.

### Ollama per-exercise feedback
Feedback in its own space (not the front-page greeting). How to avoid spamming the model?
Is a small model's feedback decent?

### Integrations
- **Garmin Connect**
- **Apple Health**
- **Standardise data models first?** Strava data is stored as Strava streams. Do we need a
  universal format other services convert into?

### MFA enrollment on /account

### Docker test harness
- **Seed ABS listening history** for zero-click soundtrack testing: generate a silent track
  with the ABS image's `ffmpeg`, create a library + scan via the ABS API, post sessions with
  chosen timestamps through ABS's local-session sync, and create a matching manual workout.
- **Postgres profile** — part of the Postgres work under *In progress*.

## Unclear

### Site loads — but sometimes not?
Server asleep? Needs a reproduction or more detail.
