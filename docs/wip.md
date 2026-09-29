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

**Shipped 2026-08-31 (fifth pass):** S13 — a broken-ownership-check class caused by a getter
contract. `GetExerciseByIDAndUserID` and `GetExerciseDayByID` returned GORM's freshly allocated
zero-value struct on a miss instead of `nil`, so the `exercise == nil` guard in all seven callers
was dead code, and `APICreateOperationForUser` — which checked only the error — let **any
authenticated user attach an activity to another user's session** (verified: HTTP 201 before the
fix, 404 after). Both getters now return `(nil, nil)` on a miss, the operation handler gained the
missing ownership branch, and the two call sites that would newly dereference nil
(`CorrelateExerciseWithExerciseDay`, `ConvertExerciseToExerciseObject`'s date fallback) gained nil
branches. The rule is now in
[`conventions.md`](conventions.md#a-not-found-getter-must-return-nil-and-the-caller-must-check-it);
regressions pinned by `database.TestExerciseGettersReturnNilOnMiss` and
`controllers/operation_authz_test.go`. Found while wiring the MCP workout search — an existing
test had been hedging around the contract (`if byID != nil && byID.ID != uuid.Nil`) rather than
asserting it, which is what hid it.

**Not swept:** only these two getters return a bare pointer this way (`GetExerciseDayByIDAndUserID`
already returned nil correctly). Other getters return values or slices, where the miss is an
explicit zero check, so this class does not reach them — but a new pointer-returning getter must
follow the convention.

### Determined fixes (confirmed defects, no design question left)

#### S10 — LOW: DB and SMTP passwords passed as command-line arguments
`entrypoint.sh` builds `CMD` from env vars, including `--dbpassword` and `--smtppassword`,
then `exec $CMD`. Two problems: the secrets land in `/proc/<pid>/cmdline`, readable by any
process in the container; and `$CMD` is **unquoted**, so a password containing whitespace
splits into multiple arguments and one containing something like `--generateinvite true`
injects a flag.

**Fix:** have the binary read these from the environment directly (it already receives them
as env vars) rather than round-tripping through argv, and quote/array-ify the invocation.

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

## Privacy audit — 2026-09-03

A privacy-focused pass over the cross-user read paths, the share gates, the `Private`
session flag and the outbound data flows. It reuses the `S<n>` numbering of the security
audit above, since the two overlap and commit messages resolve against one sequence.

**Checked and found sound (recording so a later pass doesn't re-derive):** the three social
feeds (`APIGetSharedActivities`, `APIGetCurrentSeasonActivities`, `APIGetUserActivities`) all
flow through `buildActivitiesFromExerciseDays`, which drops private sessions, and the
underlying queries enforce `users.share_activities = 1` **in SQL** rather than in Go
(`database/exerciseday.go:376`, `:398`). Season activities additionally require the caller to
hold a goal in the season. `APIGetExerciseDay`, `APIGetOperation`, `APIGetActionStatistics`
and the operation-set routes are all `…ByIDAndUserID`-scoped. `ShareStatistics` is **not**
nil-ed by `CensorUserObject`, so unlike `ShareActivities` (S11) that gate really works. The
Ollama front-page message is generated per user and served only to its subject
(`APIGetOllamaFrontPageMessageForUser` takes the user id from the token, and the cache is
keyed per user), so the private-session data in its payload never reaches anyone else. MCP is
self-scoped throughout.

**Considered and deliberately accepted:**
- **`/api/admin/exercise-days` returns every user's full day tree** — private sessions, notes,
  raw `latlng` streams and, when media is on, the listening timeline. An admin has direct DB
  access anyway, so withholding it at the API is theatre. What does survive is a
  non-privacy concern: the response is unbounded, unpaginated and inlines every stream blob,
  so it grows with the install. Worth a date bound eventually, on memory/latency grounds.
- **Achievements have no share gate** (`APIGetAchievements?user=<id>` ignores both toggles).
  Achievements read as public trophies; treating them that way is a coherent position.
- **Imported GPS has no privacy-zone concept.** Strava applies privacy zones to its own map
  rendering, not to the streams API, so a stored `latlng` track starts at the user's front
  door. Self-scoped everywhere today, but it is a constraint on any future "share this
  activity's route" feature — do not add one without addressing it.

**Shipped 2026-09-03:** S14, S15, S16 and S11.

**S15 — the user censor is now an allowlist type.** `CensorUserObject` returned a redacted
`models.User`, which is a blocklist: any field added to `models.User` later was serialized to
every caller until somebody remembered to censor it. That had already happened — `BirthDate`,
`MaxHeartrate`, `RestingHeartrate` and `ObservedMaxHeartrate` were being served to any
authenticated caller (a read-only PAT included) by `GET /api/auth/users` and
`GET /api/auth/users/:id`, with nothing rendering them. It is now
`CensorUserObject(models.User) models.PublicUser`, an explicit allowlist of the thirteen
fields the frontend actually reads off another user, so a new field on `models.User` reaches
nobody until it is added to `PublicUser` on purpose. The read DTOs (`GoalObject`,
`ExerciseDayObject`, `Activity`, `DebtObject`, `WheelviewObject`, `UserWithTickets`,
`InviteObject`) carry `PublicUser`; the raw GORM rows keep `models.User` and had their `User`
association marked `json:"-"`, which closes the `models.Week` → `[]Goal` → `User` hole that
was safe only because nothing calls `Preload("User")`. Server-side consumers that genuinely
need the row (the birthday achievement, MCP `whoami`, HR-zone anchoring, the media sync and
reconcile, the reset e-mail) moved to `GetAllUserInformation`. `AchievementDelegation.User`
was a raw row embedded in a cross-user response and is now `json:"-"` — no client read it,
they use `user_id`. Rules in [`conventions.md`](conventions.md#never-serialize-a-credential);
`assertOnlyPublicFields` in `database/user_censor_test.go` pins the exact key set, and fails
if anyone widens it without meaning to.

**S14 — private sessions no longer surface as itemised profile statistics.** The counts,
sums and streaks deliberately still include them (a private session already counts toward the
weekly goal and the leaderboard, so excluding it would make the profile disagree), but they
are excluded from the **Tops**, which name one session and its date.

A **window** holding fewer than `userStatisticsMinSampleSize` (3) sessions of the headline
activity is withheld whole (`publishWindow` returns nil), not merely stripped of its averages.
The first cut only floored the averages, which was not enough: a month with a single session
still published its distance, its longest session and its total time, and those are that one
workout said three ways — visibly so, since "distance" and "best" then show the same number.
The three windows are now `*UserStatisticsCompilation`; nil means "not enough data", never
zero. When even the **all-time** window is below the floor there is nothing to build a
breakdown from, so the whole block goes — reachable even with a headline pick, since three
activities of three different types clear the pick's floor while the most common of them has
only one session. The three
`*ExerciseDayID` fields on the Tops were removed outright — nothing read them, and the link
only ever resolved for the owner.

The **headline activity** the breakdown is built around now respects the same floor. It is
still picked from the past month — "what they have been doing lately" is the more interesting
thing to read — but at one or two activities that pick describes a single workout rather than
a person, so a thin recent window falls back to the **all-time** pick (`chooseHeadlineAction`).
Below the floor on both counts there is nothing honest to headline: the block is omitted and
the client renders an explicit empty state (`.user-stats-empty`, see
[`styleguide.md`](styleguide.md)) instead of leaving a gap, with the threshold interpolated
from the new `minimum_sample_size` field so the copy cannot drift from the constant. The same
note appears inside a single withheld window's tab — the tab is kept rather than dropped,
because a missing tab hides the fact silently, and the panel opens on the first window that
has data. Falling back rather than blanking is what makes the floor affordable here — the section carries the
month/year/all-time windows together, so suppressing the pick outright would have taken years
of statistics off the page to hide a quiet month.

That also fixed a long-standing quirk: a nil pick used to match operations with **no** action,
so a profile with nothing to headline reported a nameless section built from actionless
operations. It now reports nothing and says why.

Behaviour is documented in [`data-model.md`](data-model.md); tests in
`controllers/user_statistics_privacy_test.go`.

**S16 — the Plex artwork proxy no longer accepts dot-segments.** `plexArtworkPathAllowed` was
a bare `/library/` prefix test, and Go's HTTP client does not clean `..` out of a request
path, so a caller could reach any endpoint on their own PMS with their own token. It now
matches `^/library/[A-Za-z0-9/._-]+$` and rejects any `..` outright rather than normalising —
a real thumb path never needs cleaning, so anything `path.Clean` would rewrite is hostile.
Cases in `controllers/plex_test.go`.

**S11 — `GET /api/auth/users/:user_id/activities` fixed, both halves together.** The gate read
a censored user whose `ShareActivities` was nil-ed, so the endpoint returned 403 to everyone;
behind it, the filter kept days whose owner was *not* the requested user, i.e. it would have
returned everybody else's activities. Fixing either half alone would have turned a dead
endpoint into a leaking one, so the S15 type change (which forced the gate to read the real
row) was completed by replacing the hand-rolled filter with the SQL-scoped
`GetExerciseDaysForSharingUsersInListUsingDates` for the single requested user — which
enforces `share_activities` in the query, leaving the explicit gate as defence in depth.

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
`buildActivitiesFromExerciseDays`, and (as of the 2026-09-03 privacy audit, S14) kept out of
the profile statistics' Tops while still counting toward the totals and streaks. Design +
behaviour live in [`docs/data-model.md`](data-model.md) and
[`docs/strava.md`](strava.md#activity-privacy). Open:
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

### /exercises builder rework (`/exercises/:id`) — fast-follow
The searchable activity timeline shipped ([docs/exercises.md](exercises.md)), and the session
builder now edits **gear per operation** (each moving activity card has its own selector, with a
session-level "Set gear for all" convenience — see [docs/gear.md](gear.md)). Remaining builder
work:
- The **session aggregate shape** now returned by `/auth/activities` (`models.SessionFeedItem`:
  workout totals plus the nested per-activity metrics) is exactly what a better **session
  summary header** should consume — reuse it rather than re-deriving.
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

### Docker test harness — follow-ups (open)
The harness itself is done (`docker-test/README.md`). Ideas not yet decided:
- **Seed ABS listening history** so the soundtrack overlay can be tested with zero
  clicking: generate a silent track with the `ffmpeg` bundled in the ABS image, create a
  library + scan via the ABS API, post sessions with chosen timestamps through ABS's
  local-session sync endpoint, and create a matching manual workout in Treningheten.
- **A Postgres profile** — blocked on the item under Problems below.

## Problems

### Weight entries accept any value, including negative
`APICreateWeightForUser` stores `weight: -1` without complaint (found by the handler flow
tests, 2026-09-29). Probably wants a sane range check (> 0, and some upper bound); the test
in `controllers/account_api_test.go` deliberately doesn't assert it until this is decided.

### `db_type: postgres` is silently rewritten to `mysql`
`files/config.go` (the `DBType` default check) only accepts `mysql`/`sqlite`; anything
else — including `postgres` — is replaced with `mysql` and saved. `CLAUDE.md` says
three backends are interchangeable. Decide: fix the check, or drop Postgres from the docs.

### Site loads
But sometimes not? Server asleep?