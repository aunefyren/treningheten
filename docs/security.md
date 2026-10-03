# Security hardening

Cross-cutting defences that don't belong to any one feature. Feature-local security
notes live with the feature: OAuth flow properties in [oauth.md](oauth.md), token
scopes in [pat.md](pat.md), outbound-request vetting in
[media.md](media.md#outbound-request-safety), the credential-serialization rule in
[conventions.md](conventions.md#never-serialize-a-credential).

The assumed threat model is a self-hosted instance with several invited users who do
**not** fully trust each other, plus the public internet reaching `/api/open`,
`/api/oauth` and `/mcp`.

## Rate limiting and the password-hash budget

`middlewares/ratelimit.go`. All state is in-memory and per-process: Treningheten is a
single binary, so there is no second instance to share counters with, and a limiter
that survives a restart isn't worth a Redis dependency. The goal is to make credential
stuffing slow and to keep a handful of clients from burning every core — not to be an
exact accountant.

Limits are **code constants, not config**. They are generous enough that a real user
never meets them, and an operator who needs different numbers is better served by the
reverse proxy in front.

| Surface | Limit | Keyed by |
| --- | --- | --- |
| `POST /api/oauth/token` | 20 / 5 min | client IP |
| `/api/open/*` (register, reset request, reset-code check, verification) | 15 / 5 min | client IP |
| `POST /api/oauth/register` (RFC 7591 DCR) | 5 / hour | client IP |
| password grant failures | 5 / 15 min, then locked | account (submitted e-mail) |

Blocked requests get **429** with a `Retry-After` header. The account lockout is
counted per e-mail address so an attacker rotating IPs still cannot grind one account,
and it is cleared on a successful login, so a user who eventually remembers their
password is not punished. The lock lifts as failures age out of the window rather than
on a separate timer — one clock to reason about.

### The password-hash budget

bcrypt runs at cost 14 (`models/user.go`), which is deliberately about a second of a
core. That is a good property against offline cracking and a liability when it is
reachable unauthenticated: without a cap, enough simultaneous logins saturate the box
*even while every individual caller stays inside its rate limit*.

So every bcrypt hash or compare goes through `middlewares.AcquirePasswordCostSlot()`,
a semaphore sized at half the machine's cores (minimum one). A caller that cannot get
a slot within 5 seconds is shed with **503** and a `Retry-After` — the alternative is
an unbounded queue of second-long operations. Call sites: the OAuth password grant,
registration, the password reset, and the account update.

**If you add a code path that hashes or compares a password, take a slot around it.**

### Client IP trust

The limiters key on `context.ClientIP()`, and Gin trusts every proxy by default — so
without configuration a caller could send their own `X-Forwarded-For` and get a fresh
budget per request, defeating the whole thing. `main.go` therefore calls
`SetTrustedProxies(trustedProxyCIDRs)`, limiting trust to loopback and the private
ranges (the normal reverse-proxy-on-the-same-host/Docker-network deployment).

**Consequence:** a reverse proxy on a *public* address is not trusted, so every request
through it is attributed to the proxy's IP and shares one budget. Such a deployment
needs its proxy's range added to `trustedProxyCIDRs`.

## Password changes end existing sessions

`database.RevokeAllRefreshTokensForUser` revokes every live refresh token for a user in
one statement. It is called from both password-change paths:

- `APIUpdateUser` (`POST /api/auth/users/:user_id`) — only when a new password was actually
  submitted, and **before** the handler issues its fresh token set. The caller's own
  session therefore survives (the frontend stores the pair the handler returns) while
  every other session dies.
- `APIChangePassword` (`POST /api/open/users/password`, the reset-code flow) — the
  user isn't logged in here, so nothing is re-issued and every session ends.

Without this, a refresh token an attacker obtained before the change keeps minting
access tokens for its full 30-day lifetime, and "I was compromised, so I changed my
password" achieves nothing.

**Personal Access Tokens deliberately survive a password change.** They are separate,
explicitly-created credentials that a user may have wired into scripts or an MCP
client; silently breaking them on a password change would be surprising. They are
listed and revocable in the "Developer access tokens" section of `/account` — see
[pat.md](pat.md).

## Config file permissions

`config/config.json` holds the JWT signing key, the database password, the SMTP
password, `Media.TokenKey` (which decrypts every stored provider credential) and the
VAPID private key. It is written `0600` inside a `0700` directory
(`files.SaveConfig`, `configFileMode` / `configDirMode`).

`os.WriteFile` only applies its mode when it *creates* the file, and `MkdirAll` only
when it creates the directory — so `SaveConfig` also `chmod`s both explicitly on every
save. Without that, an install that predates this change would keep its world-readable
`0644` secrets forever. The chmod failures are logged, not fatal: a bind mount or a
Windows host may not honour the change, and that shouldn't stop the app from starting.

## Confidential OAuth clients must present their secret

`OAuthClient.Public` is tagged `default:true`. GORM leaves a `false` zero value out of the
`INSERT`, so until 2026-09-29 every dynamically registered *confidential* client
(`token_endpoint_auth_method` other than `none`) was stored with `public = true` — and
`resolveClient` skips the secret check for public clients. Any caller knowing such a
client's id could use it with any secret or none. (PKCE was still required for the code
grant, and a valid code or refresh token was still needed, so the secret was a lost layer
rather than the only one.)

Two fixes: `database.CreateOAuthClient` writes `public` explicitly after the insert (see
[data-conventions.md](data-conventions.md)), and `resolveClient` now demands the secret
whenever a secret hash is stored, whatever `Public` says — which also covers rows written
before the fix, without a data migration. Tests: `database.TestCreateOAuthClientStoresConfidentialClientsAsConfidential`
and `controllers.TestOAuthConfidentialClientRegistration` (including a deliberately
mis-stored legacy row).

## Accepted risks

Deliberate trade-offs, recorded so they aren't rediscovered as findings:

- **Limiter state is in-memory.** A restart forgives every counter, and limits are code
  constants rather than config (see [Rate limiting](#rate-limiting-and-the-password-hash-budget)).
- **`media.allow_private_targets` defaults to `true`.** A LAN or loopback Plex is the
  normal deployment and `false` would break existing installs on upgrade. With it on, an
  authenticated user can still probe the host's own network — the Audiobookshelf connect
  errors stay a coarse port oracle, kept because they are useful when a URL is wrong. An
  instance with untrusted users should set it to `false`. Revisit if a guided setup ever
  ships that could ask. See [media.md](media.md#outbound-request-safety).
- **`/api/admin/exercise-days` returns every user's full day tree** — private sessions,
  notes, raw `latlng` streams and the listening timeline. An admin has direct DB access
  anyway, so withholding it at the API would be theatre. The remaining concern is size:
  the response is unbounded and inlines every stream blob, so it wants a date bound
  eventually on memory/latency grounds.
- **Achievements have no share gate** (`APIGetAchievements?user=<id>` ignores both share
  toggles). They read as public trophies.
- **Imported GPS has no privacy-zone concept.** Strava applies privacy zones to its own
  map rendering, not to the streams API, so a stored `latlng` track starts at the user's
  front door. Self-scoped everywhere today — **do not add any "share this route" feature
  without addressing this first.**

## Audited and found sound

From the 2026-08-29 security audit and the 2026-09-03 privacy audit, so a later pass
doesn't re-derive them:

- **SQL injection** — no string-built queries in `database/`; everything is GORM with bound
  parameters. The one `fmt.Sprintf` into SQL (`CREATE DATABASE` in `database/client.go`)
  takes its value from config, not a request.
- **Path traversal on images** — `safeImageFilePath` (`controllers/image.go`), with the
  filename built from the *parsed* UUID rather than the raw parameter.
- **OAuth authorization-code flow** — exact-match redirect URI, mandatory PKCE S256,
  constant-time verifier comparison, single-use codes consumed atomically, code bound to
  the issuing client, scope narrowed to the client's grant (`controllers/oauth_authorize.go`).
- **Refresh-token lifecycle** — rotation with reuse detection revoking the whole chain;
  admin status re-derived from the DB on each refresh (`auth/auth.go`).
- **Credential encryption at rest** — AES-256-GCM, random nonce per encryption, correct
  length checks (`utilities/crypto.go`).
- **Ownership scoping** — the `…ByIDAndUserID` pattern covers operations, operation sets,
  exercises, gear, weights, PATs and media sync. Pointer-returning getters must return
  `nil` on a miss (see [conventions.md](conventions.md#a-not-found-getter-must-return-nil-and-the-caller-must-check-it)).
- **Admin enforcement** — `Auth(true)` requires both the admin scope on the token and a
  live `admin` flag on the DB row; read-only scopes are blocked from write methods
  (`middlewares/auth.go`).
- **MCP** — authenticated, scope-checked, and every tool closes over the authenticated
  `userID` rather than taking one as an argument.
- **Password reset codes** — 16 random uppercase chars (~82 bits), 24h expiry, rotated on
  use; the request endpoint answers identically whether or not the account exists.
- **CSRF** — the API authenticates from the `Authorization` header, not the cookie. The one
  cookie-accepting group, `AuthImageReadOnly`, is GET-only and `SameSite=Strict`.
- **Social feeds** — all three go through `buildActivitiesFromExerciseDays`, which drops
  private sessions, and `share_activities = 1` is enforced **in SQL**
  (see [activity-feed.md](activity-feed.md)). Season activities also require the caller to
  hold a goal in the season.
- **`ShareStatistics`** is not nil-ed by `CensorUserObject`, so that gate genuinely works.
- **Ollama greeting** — generated and cached per user, served only to its subject, so the
  private-session data in its prompt never reaches anyone else.

## Still open

Tracked in [wip.md](wip.md#security--open): the wildcard CORS configuration (currently
inert by registration order), the JS-readable 30-day refresh cookie, whether the `admin`
scope should be grantable to dynamically registered clients, the absence of security
response headers (notably a CSP), secrets passed as argv by `entrypoint.sh`, and toolchain
patch drift.
