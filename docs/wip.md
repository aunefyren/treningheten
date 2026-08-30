# Work in progress
This doc should contains plans, ideas, problems, bugs and so on. Finished stuff should not live here, it should be moved to /docs in relevant files. If none fit, it should probably be made. If features are in between start and done implementation, one can still create a doc which get's updated over time.

## Security audit — 2026-08-29

Full read-through of auth, middleware, every `controllers/` handler, the data-access layer,
the media/integration clients, config handling, the frontend render paths and the dependency
tree. Findings are ranked by severity and numbered stably — the numbering is kept even as
items ship, so earlier notes and commit messages keep resolving. Remove each item as it
lands and, where the fix changes documented behaviour, move the explanation into the
relevant `docs/*.md`.

**Shipped 2026-08-30 (fourth pass):** S7, S8 and S9. The unauthenticated surface is now
rate limited per client IP (`middlewares/ratelimit.go`) — the OAuth token endpoint, the
`/api/open` group and dynamic client registration — with a per-account lockout on repeated
password-grant failures so an attacker rotating IPs still can't grind one account. Because
bcrypt at cost 14 is a deliberate ~1s of a core, every hash and compare also passes through
a semaphore sized at half the machine's cores, shedding with 503 rather than queueing; the
limiters would have been defeatable by a spoofed `X-Forwarded-For`, so `main.go` now sets
trusted proxies to loopback + the private ranges. Both password-change paths revoke every
refresh token for the user (PATs deliberately survive); in `UpdateUser` the revoke runs
before the handler's existing token re-issue, so the caller keeps their own session. And
`config/config.json` is written `0600` in a `0700` directory, with an explicit `chmod` on
every save so installs that predate the change don't keep their world-readable secrets.
Design and trade-offs in [`security.md`](security.md); tests in
`middlewares/ratelimit_test.go`, `database/oauth_revoke_test.go` and
`files/config_permissions_test.go`.

**Residual risk, deliberately accepted (S7):** the limiter state is in-memory and
per-process, so a restart forgives every counter, and a reverse proxy on a *public*
address is not in `trustedProxyCIDRs` — such a deployment attributes every request to the
proxy's IP and shares one budget until its range is added. Limits are code constants, not
config, on the reasoning that an operator wanting different numbers has a reverse proxy to
do it in.

**Shipped 2026-08-29 (third pass):** S4, S5 and S6, the media-integration findings. Plex
PINs are now bound to the user who created them (`plexPinOwners`, in-memory with a 15-minute
TTL), so one user can no longer poll another's in-flight PIN and capture their Plex token.
Outbound requests to user-supplied provider URLs go through a destination policy enforced in
the dialer's `Control` hook (`controllers/media_dial.go`), which sees the resolved IP and so
covers DNS rebinding and redirects; link-local (cloud metadata) is always refused, and
loopback/private ranges follow the new `media.allow_private_targets` flag. Plex TLS
verification is now skipped only for hosts that cannot present a verifiable certificate
(bare IPs, loopback, `plex.direct`) instead of unconditionally. Design and the trade-offs are
in [`media.md`](media.md#outbound-request-safety); tests in `controllers/media_dial_test.go`
and `controllers/plex_security_test.go`.

**Residual risk, deliberately accepted:** `media.allow_private_targets` defaults to **true**,
because a self-hosted Plex on the LAN or on `127.0.0.1` is the normal deployment and
defaulting to `false` would break existing installs on upgrade. With it on, an authenticated
user can still probe the host's own network (the ABS connect error messages remain a coarse
port oracle, kept because they are genuinely useful when a URL is wrong). An instance with
untrusted users should set the flag to `false`. Worth revisiting if the app ever ships a
guided setup that could ask.

**Shipped 2026-08-29 (second pass):** S3, the stored-XSS-to-account-takeover chain.
`APIUpdateExerciseDay` loaded the day with the unscoped `GetExerciseDayByID`, so any
authenticated user could overwrite anyone's note; and `web/js/exercise.js` piped that note
into `innerHTML` unescaped in two places, plus raw into a `<textarea>` in a third. Since both
auth cookies are `document.cookie`-readable by design, a payload in someone else's note stole
their access **and** refresh token. Fixed by scoping the write to the caller (404 on a miss,
so it doesn't confirm another user's day exists) and routing all three sinks through the
`escapeHTML()` helper the file already had. The escaping rule is now in
[`conventions.md`](conventions.md#frontend-vanilla-js-conventions) and the ownership rule in
[`exercises.md`](exercises.md#ownership); `controllers/exercise_day_authz_test.go` pins the
authorization half. **Not addressed here:** the cookie storage model itself and the absent
CSP — both still open below, and both would have contained the blast radius.

**Shipped 2026-08-29:** S1 and S2, the two credential-leak findings. `CensorUserObject` was
being applied to a loop copy in `GetUsersInformation` and
`GetAllUsersWithSundayAlertsEnabled`, so `GET /api/auth/users` served every user's bcrypt
hash, e-mail, live reset code and Strava token to any authenticated caller — a read-only PAT
included. `GET /api/auth/debts/:debt_id` leaked the same set a second way, through its
candidate list, and had no authorization on the debt id at all. Fixed by index-assigning the
censor, splitting the getters into censored/uncensored families for the two server-side jobs
that legitimately need credentials, marking every credential and recovery field `json:"-"`
(with a derived `strava_connected` boolean replacing `strava_code` on the account page), and
scoping `APIGetDebt` to season membership. Conventions now live in
[`conventions.md`](conventions.md#never-serialize-a-credential) and
[`seasons-and-goals.md`](seasons-and-goals.md#consequences-debts-leaderboard-prizes);
regressions are pinned by `database/user_censor_test.go` and
`models/user_serialization_test.go`.

Scope note: the threat model assumed is a self-hosted instance with several invited users who
do **not** fully trust each other, plus the public internet reaching `/api/open`,
`/api/oauth` and `/mcp`.

### Determined fixes (confirmed defects, no design question left)

#### S10 — LOW: DB and SMTP passwords passed as command-line arguments
`entrypoint.sh` builds `CMD` from env vars, including `--dbpassword` and `--smtppassword`,
then `exec $CMD`. Two problems: the secrets land in `/proc/<pid>/cmdline`, readable by any
process in the container; and `$CMD` is **unquoted**, so a password containing whitespace
splits into multiple arguments and one containing something like `--generateinvite true`
injects a flag.

**Fix:** have the binary read these from the environment directly (it already receives them
as env vars) rather than round-tripping through argv, and quote/array-ify the invocation.

#### S11 — LOW: `GET /api/auth/users/:user_id/activities` is broken and, if fixed naively, leaks
`APIGetUserActivities` (`controllers/user.go:1045`) has two independent problems:

- It gates on `user.ShareActivities` (`:1063`), but `user` came from `GetUserInformation`,
  whose censor sets `ShareActivities = nil` (`database/user.go:469`). The gate therefore
  **always** returns 403 — the endpoint is dead.
- The filter behind it (`:1116`) keeps exercise days where `userID != *exerciseDay.UserID`,
  i.e. it returns every *other* sharing user's activities and excludes the requested user's.
  That's inverted; making the censor stop nil-ing the field without fixing this would turn a
  broken endpoint into a data-leaking one.

**Fix:** both together — decide whether `ShareActivities` should be visible on a censored
object (it's a visibility flag, not PII; probably yes) and invert the filter to `==`.

#### S12 — LOW: Go toolchain is behind on security patches
`govulncheck` reports **16 stdlib vulnerabilities reachable from this code** at the local
toolchain (go1.26.2). The Dockerfile pins `golang:1.25.0-alpine`, which is older still.
Reachable highlights: `GO-2026-6091` / `GO-2026-4982` / `GO-2026-4980` (html/template escaper
bypasses → XSS, reached through Gin's template rendering), `GO-2026-6089` (missing
`ReadHeaderTimeout` on the h2c check), `GO-2026-4918` (infinite loop in the HTTP/2 transport,
reached from the Strava client). Nothing in the direct module dependencies is reachable.

**Fix:** bump the Dockerfile base and the CI matrix to a current patch release, and add
`govulncheck ./...` to `.github/workflows/go.yml` so this doesn't drift again.

### Open topics (need a decision, not just a patch)

#### CORS is configured wide open — and is currently inert by accident
`main.go:403-411` sets `AllowOrigins: ["*"]` **with** `AllowCredentials: true` and an
`AllowOriginFunc` that returns `true` for everything. That combination normally means any
website can make credentialed cross-origin calls and read the responses.

It doesn't bite today only because of registration order: `router.Use(cors…)` runs at
`main.go:403`, *after* every `/api` group, `/mcp` and the `.well-known` routes are registered,
and Gin freezes a route's handler chain at registration time. So the CORS middleware applies
only to the static/templated routes declared below it, and the API sends no CORS headers at
all — which is why the browser SPA (same-origin) works and why nothing has broken.

That's a landmine: moving the `Use` call up, or adding a route below it, silently turns a
wildcard-with-credentials policy on. Decide what the API's actual cross-origin story is —
if it's "same-origin only", delete the middleware; if third-party browser clients are wanted,
set an explicit origin allow-list and move it above the groups deliberately.

#### Refresh token lives 30 days in a JS-readable cookie
`web/js/functions.js:6-13` stores the access token (7d) and refresh token (30d) via
`document.cookie` with `path=/; samesite=strict` — no `Secure`, and necessarily no `HttpOnly`
since the SPA reads them back to build the `Authorization` header. Any XSS is therefore a
30-day account compromise, not a 60-minute one — which is exactly what made the day-note sink
(S3, now fixed) a takeover rather than a defacement.

Options, roughly in increasing order of work: add `Secure`; move the refresh token to an
`HttpOnly; Secure; SameSite=Strict` cookie set by the server and have `/api/oauth/token`
read it from there for the `refresh_token` grant (the access token can stay in JS memory);
or accept the risk and rely on per-sink escaping plus a CSP. Worth deciding before adding
more `innerHTML` render paths.

#### Dynamic client registration is open to the internet
`POST /api/oauth/register` (`controllers/oauth_clients.go:29`) is unauthenticated and
unlimited: anyone can create `OAuthClient` rows forever. RFC 7591 allows this and MCP clients
expect it, so it isn't wrong — but on a single-household instance it's unbounded write access
for anonymous callers, and `SupportedScopes` includes `admin` (`models/oauth.go:20`), so a
registered third-party client *can* request the admin scope and get it if an admin approves the
consent screen.

Decide: gate DCR behind a config flag (default off unless MCP is in use), rate-limit it, prune
unused clients — and separately, whether `admin` should be grantable to a
dynamically-registered client at all, or reserved for the first-party client and PATs.

#### No security response headers
No `Content-Security-Policy`, `X-Content-Type-Options`, `X-Frame-Options`/`frame-ancestors`
or `Strict-Transport-Security` anywhere. A CSP in particular is the structural mitigation for
the whole `innerHTML` class of bug — escaping each sink, as S3's fix did, is per-site and
relies on the next author remembering. But a CSP is a real piece of work given how much markup
the JS builds inline, and inline `onclick=` handlers are used throughout
(`web/js/account.js:859`, `web/js/exercise.js:49`, …), so a strict policy needs those
refactored first. Worth scoping as its own task rather than bolting on a weak policy.

### Checked and found sound
Recording these so a future pass doesn't re-derive them:

- **SQL injection** — no string-built queries anywhere in `database/`; everything is GORM with
  bound parameters. The one `fmt.Sprintf` into SQL (`database/client.go:132`, `CREATE
  DATABASE`) takes its value from config, not from a request.
- **Path traversal on images** — `safeImageFilePath` (`controllers/image.go:78-86`) plus
  building the filename from the *parsed* UUID rather than the raw parameter.
- **OAuth authorization-code flow** — exact-match redirect URI, mandatory PKCE S256,
  constant-time verifier comparison, single-use codes consumed atomically, code bound to the
  issuing client, scope narrowed to the client's grant (`controllers/oauth_authorize.go`).
- **Refresh-token lifecycle** — rotation with reuse detection revoking the whole chain, admin
  status re-derived from the DB on each refresh (`auth/auth.go:207-262`).
- **Credential encryption at rest** — AES-256-GCM with a random nonce per encryption, correct
  length checks (`utilities/crypto.go`).
- **Ownership scoping** — the `…ByIDAndUserID` pattern is applied consistently across
  operations, operation sets, exercises, gear, weights, PATs and media sync. The two places it
  was missed (the day-note write, the debt read) are fixed; see the shipped notes above.
- **Admin enforcement** — `Auth(true)` requires *both* the admin scope on the token and a live
  `admin` flag on the DB row (`middlewares/auth.go:136-152`); read-only scopes are blocked from
  write methods at `:155`.
- **MCP** — authenticated, scope-checked, and every tool closes over the authenticated
  `userID` rather than taking one as an argument (`controllers/mcp.go:116-143`).
- **Password reset codes** — 16 random uppercase chars (~82 bits), 24h expiry, rotated on use,
  and the request endpoint returns an identical response whether or not the account exists
  (`controllers/user.go:547-607`, `database/user.go:427`).
- **CSRF** — the API authenticates from the `Authorization` header, not the cookie, so ordinary
  form/XHR CSRF doesn't apply. The one cookie-accepting group is `AuthImageReadOnly`, which is
  GET-only and `SameSite=Strict`.

## Plans & Ideas

### More workout tags
Ideas for tags:
- Easy
- Splits
Must respect Strava sync

### Onboarding flow
- A small improvement would be an invite link, which includes the invite and autofill the form
- A bigger improvement would be an invite link which acts as a onboarding flow
  - Initial page is user registration with invite code autofill from the URL
  - Next page is season registration IF the invite was sent with a season context (both possible)
  - Next page is an optional page to choose which activities count as exercise (many do not count walks)
  - Next page is an optional page to connect Strava or Hevy
  - Next page is an optional page to enable notifications
  - Anything else?
- This should be a smooth, user friendly flow, no navbar
- If you break out of the onboarding, you should be redirected back where you were, no need to re-enter anything
- Would this be able to reuse any forms, pages or code, or require a totally new page?

### Sick leave is per season/goal
- makes sense that different seasons have different sick leave
- makes little sense that you can join multiple seasons at once, but only use sick leave on one goal

### Implement Garmin connect

### Implement Apple Health connect

### Standardize data models
- We save some data in the form of Strava data streams
- If we implement other services, like Garmin, how should the data be saved?
- Do we need a universal data format all other services can be converted into?

### Flexible workouts
- Work out more one week, have the extra effort carry over.
- Must be season specific setting
- Option to allow how many workouts carry over, how long they can exist before they decay
- Must be user friendly and understandable in the UI

### Front page activities, add partner
- Allow a person to tag their partner/group on their exercise session within builder
- Show as activity together on season activity post if both have joined season
- Sync over Strava partner tag?
- Auto-detect partner of enough data?

### Music / audio integration
Overlay your listening history onto time-based activities. The **Plex**, **Spotify**, and
**Audiobookshelf** connections, the history pull, the session-level timeline, the
`/statistics` Soundtrack block, Plex audiobook/podcast classification, and Plex artwork
(via proxy) have all shipped. Full design + current status live in
[`docs/media.md`](media.md) — keep open items there, not duplicated here.

**Recently shipped (2026-08-29):** provider identity + duplicate-play merging.
`MediaPlayback` now stores every id each provider hands over (`ProviderItemID`,
`ProviderParentID`, `ProviderGUID`, `ProviderSessionID`) — deliberately including ones
nothing reads yet, since provider history only reaches back hours to days and an id not
captured at pull time cannot be backfilled. `coalesceOverlappingEvents` merges overlapping
records of one item in the shared matcher, fixing Audiobookshelf reporting a single podcast
episode as three sessions; and `StartedBefore` marks a listen that was already playing at
session start, so the clamped `00:00` stamp no longer reads as fact. Deliberately **not**
done: a DB unique constraint on provider ids — the same item legitimately plays twice in a
session, and rows are disposable delete-and-replace. Details in [`media.md`](media.md).

**Still to build:**
- **Cross-activity stats — "fastest songs":** avg speed over `[StartedAt, EndedAt]` from a
  session activity's `OperationSet.StravaStreams`, which needs the stream-windowing done
  per track. The one remaining planned increment. (Per-track avg-HR on the card already
  shipped.)
- **Per-(session, provider) pull guard:** the single `Exercise.MediaRetrievedAt` spans all
  providers — fine for the common case, but connecting a provider *after* a session was
  already pulled relies on the 🎧 re-pull button. Generalize the guard when it becomes
  annoying.
- **Edge case (note, not solving now):** editing a session's **time** changes the match
  window — the 🎧 re-pull re-matches on demand, but there's no automatic re-match on a time
  edit. (A skipped date-only session will match once a real time is set + re-pulled.)

**Open questions:**
- Privacy: listening data is sensitive even self-hosted — any per-activity visibility controls?
- Cross-provider de-dupe if a user has overlapping sources (e.g. casting Spotify through Plex)?
  (Per-provider rows side-step it for now; only matters once 2+ providers are connected.
  Note the new **within-provider** overlap merge does *not* address this — it groups by
  provider id, and the same play through two providers has two unrelated ids. A
  cross-provider pass would have to match on title/artist + time overlap.)

### Private sessions — follow-ups
The per-session privacy flag shipped: `Exercise.Private`, mirrored from Strava on every sync,
toggled in the builder for manual/Hevy sessions, filtered out of every feed by
`buildActivitiesFromExerciseDays`. Design + behaviour live in
[`docs/data-model.md`](data-model.md) and [`docs/strava.md`](strava.md#activity-privacy).
Open:
- **Hevy has no privacy source.** Hevy workouts import as visible; only the builder toggle
  hides them. Check whether the Hevy API exposes a per-workout visibility field.
- **MCP doesn't expose `private`.** Deliberate for now — MCP is self-scoped, so nothing leaks
  either way, and threading it through `operationObjectToActivity` / `resolveExerciseDate`
  costs more than it currently returns. Revisit if an LLM client ever needs to reason about it.
- **Bulk privacy.** No way to hide a whole week, or to say "hide everything by default and
  opt in per session". Probably not wanted; note it if it comes up.

### Strava sync overrides a builder-deleted session
Noticed while building session privacy, not fixed: `StravaSyncActivityForUser` sets
`Enabled`/`IsOn = true` on **every** sync, so turning an imported session off in the builder
is silently undone by the next hourly run. `CountsTowardGoal` protects itself (snapshot on
first import only) and `Private` is deliberately Strava-owned, but `IsOn` is a user decision
being overwritten by a background job. Options: only set `IsOn` on a new import, or treat a
manual off as a delete Strava may not resurrect.

### Leave season button is not implemented
- Which season? all?
- Not broken, never built function, only button stub is present on /account

### Delete account button is not implemented
- What gets left behind? Do seasons you joined still show you? Show 'Deleted user'?
- Not broken, never built function, only button stub is present on /account

### Generate debt button on /admin is misleading and could be improved
- This admin function used to be for recalculating debt for a given week, after some changes happened in the DB in the back end
- It now does this, but also more. It regenerates achievements for example
- A typical use case:
  - User forgets to log exercises
  - Manually fixed in DB by adding exercises, removing debt object, removing ahcivmenent delegations
  - Click generate debt button to "recalculate week"
- Module/function could be remade to a "fix last week button"
  - Add/remove exercises
  - reset achievements/deb for week
  - Recalculate week
  - Anything else?

### Best effort system
- Manual programming per activity?
- "fastest 5K"...
- Must be calculated at save or during runtime?
- Notification integration for PR?
- PRs for reps and weight on strength exercises?
- Time based best efforts? During this season? During this year?
- The per-distance **segments** and the processed stream summary now ship (see
  [mcp.md](mcp.md)), so the data exists to compute a **best split** (fastest 1 km/mile)
  across activities — a natural first increment.
- Grade-adjusted pace (GAP) and VAM (vertical ascent speed) — deferred as noisier/advanced
  elevation follow-ups.

### AI Ollama feedback on exercises?
Per-exercise feedback in its own dedicated space (not the front-page greeting).
- How to avoid spamming the LMM
- Little model, can the feedback be decent?

### Locked achievements CSS bug
- Achievements with the pad lock icon on /achievements have a rounded border around the icon, like a margin between the icon and the rounded color around the achievement
- SVG get's cut off on the corners because of this

### /exercises builder rework (`/exercises/:id`) — fast-follow
The searchable activity timeline shipped ([docs/exercises.md](exercises.md)), and the session
builder now edits **gear per operation** (each moving activity card has its own selector, with a
session-level "Set gear for all" convenience — see [docs/gear.md](gear.md)). Remaining builder
work:
- The per-activity **aggregate shape** built for `/auth/activities` is exactly what a better
  **session summary header** should consume (activity chips + per-activity metrics) — reuse it
  rather than re-deriving.
- Consider a fuller per-`Operation` card layout (each activity type its own sub-card with its
  own metrics/sets) and clearer affordances for adding a *second activity type* to an existing
  session vs a *second session* to the day. (Gear is already per-operation; this is the
  remaining organisation/metrics work.)
- Watch the media/soundtrack coupling: soundtrack is session-scoped (`Exercise`), so builder
  changes to session time/duration affect the match window (already noted under media).

### Gear tracker — possible follow-ups
The gear feature shipped (see [`docs/gear.md`](gear.md)), and per-operation gear editing now
ships too — each moving activity card in the builder has its own gear selector, with a
session-level "Set gear for all" convenience for combined sessions that mix 2+ moving
activities. Open refinements left for later:
- **Auto-assign primary.** The selector *suggests* the user's primary gear for a moving
  activity with no gear, but it isn't persisted until the user interacts. Could auto-assign on
  the first operation instead.
- **Primary per type.** Only one primary per user today; a primary shoe *and* a primary bike
  might be more useful.

### Better gear management
- Or maybe this is finished now that we have a /gear page?
- The modal covers the entire /gear page? Move stuff away from modal? Remove modal?

### Make first day of the week changeable
- Default monday, but choose
- Big changes to logic

### MFA enrollment on /account

## Problems

### Site loads
But sometimes not? Server asleep?