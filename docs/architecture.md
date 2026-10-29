# Architecture

How the codebase is put together: packages, routing, how the frontend is served,
persistence, scheduled jobs and versioning. Feature behaviour lives in the per-feature docs
(see the [index](README.md)); code style lives in [conventions.md](conventions.md).

Treningheten is a single Go binary — a Gin HTTP server on GORM — that serves both the JSON
API and the frontend from `web/`. Module: `github.com/aunefyren/treningheten`.

## Packages

- **`main.go`** — startup, flag parsing, `initRouter()` (every route) and `scheduleJobs()`
  (every cron job).
- **`models/`** — plain structs: GORM table models plus request/response DTOs
  (`UserCreationRequest`, …). One file per entity; `models/gorm.go` holds the shared
  embedded fields.
- **`database/`** — all data access, one file per entity mirroring `models/`.
  `database/client.go` owns the global `database.Instance *gorm.DB`, `Connect()` (dispatches
  on the DB type) and `Migrate()`, the canonical list of `AutoMigrate` calls — **new models
  are added there**. `database/seed.go` seeds the curated default `Action` rows on startup;
  `database/seed_hevy.go` seeds the Hevy exercise catalog (see [hevy.md](hevy.md)).
- **`controllers/`** — Gin handlers, one file per domain, named `APIXxx` (newer) or `Xxx`.
  They bind JSON, call `database`/`utilities`, and answer with `gin.H`. Also home to the
  scheduled-job entry points. Raw GORM rows are enriched into `XxxObject` structs by
  `Convert<X>To<X>Object` — the canonical read layer, see
  [data-conventions.md](data-conventions.md).
- **`middlewares/`** — `auth.go` (below) and `ratelimit.go`: sliding-window limiters on the
  unauthenticated surface, the per-account login lockout, and `AcquirePasswordCostSlot()`,
  the semaphore every password hash/compare must take. See [security.md](security.md).
- **`auth/`** — access-token JWTs, the refresh-token lifecycle, PAT generation/hashing and
  scope helpers. See [oauth.md](oauth.md) and [pat.md](pat.md).
- **`files/config.go`** — config loading. `files.ConfigFile` is the global
  `models.ConfigStruct`, persisted to `config/config.json` (created on first run; missing
  secrets such as the JWT key and VAPID keys are generated and saved). It holds every secret
  the app has, so it is written `0600` in a `0700` directory.
- **`utilities/`** — cross-cutting helpers: `migrate.go` (data migrations beyond schema),
  `smtp.go` (email), `utilities.go` (week/date math, misc), credential encryption.
- **`logger/`** — logrus wrapper exposing `logger.Log`.
- **`internal/testdb`, `internal/smtptest`** — test-only: per-test Postgres/MySQL databases
  when the DSN env vars are set, and an in-process SMTP server. See
  [conventions.md](conventions.md#tests).
- **`scripts/`** — standalone dev tools (`hevyseed`, `mediaprobe`).
- **`docker-test/`** — the live Docker test harness, see its
  [README](../docker-test/README.md).

## Routing and auth

`initRouter()` in `main.go` defines all API routes in groups under `/api`:

| Group | Guard | What |
|---|---|---|
| `/api/oauth/*` | rate limited | OAuth 2.0 token, authorize, register, revoke |
| `/api/open/*` | rate limited | no auth: register, password reset, email verification |
| `/api/auth/*` | `middlewares.Auth(false)` | any logged-in user |
| `/api/auth/*` (images) | `middlewares.AuthImageReadOnly()` | image routes, also accept the cookie — see [image-serving.md](image-serving.md) |
| `/api/admin/*` | `middlewares.Auth(true)` | admins |

OAuth discovery documents are at the root (`/.well-known/oauth-authorization-server`,
`/.well-known/oauth-protected-resource`) and the MCP server is at `/mcp`
([mcp.md](mcp.md)). Login is the OAuth token endpoint (`POST /api/oauth/token`), not a
bespoke route.

`Auth(admin bool)` resolves a `Bearer` credential into a `Principal` from **either** an
access-token JWT **or** a Personal Access Token (dispatched on the `trh_pat_` prefix). It then
checks the user is enabled and (when SMTP is enabled) email-verified, enforces the admin scope
plus DB admin for `Auth(true)`, and blocks write methods for read-only scopes. 401/403 answers
carry RFC 6750 `WWW-Authenticate` challenges. `GetAuthUsername()` resolves the user UUID from
either token type.

Client IPs (which the rate limiters key on) are only trustworthy because `main.go` calls
`SetTrustedProxies(trustedProxyCIDRs)` — loopback and private ranges.

**Adding an endpoint:** register the route in `initRouter` under the right group, put the
handler in the matching `controllers/` file, and add the route to the test harness in
`controllers/api_harness_test.go` so it is covered by the robustness sweep.

## Frontend serving

The frontend is **not** a separate build. `main.go` serves `web/` through Go's
`text/template`, injecting config values (`appName`, `appVersion`, `vapidPublicKey`,
`stravaRedirectURI`, `ollamaEnabled`, …) at request time:

- `web/html/*.html` → Gin HTML templates at clean URLs (`frontpage.html` → `/`).
- `web/js/*.js`, `web/json/*.json`, `web/txt/*.txt` → text templates under `/js`, `/json`,
  `/txt`, with special cases such as `/service-worker.js`, `/manifest.json`, `/robots.txt`.
- `web/assets`, `web/css` → plain static files.

So frontend JS can use `{{ .appVersion }}`-style variables. Special URL-to-file mappings
(`/users/:user_id` → `user.html`, …) live in the `filePathTableList` map in
`registerTemplatedStaticFilesForDirectory`. The visual system is in
[styleguide.md](styleguide.md).

## Persistence

GORM with three interchangeable backends selected by `DBType`: `sqlite` (file at
`DBLocation`, the default), `mysql` (creates the database if missing) and `postgres`.
Production runs MySQL. The schema is applied at startup by `AutoMigrate` — there are no
migration files; see [conventions.md](conventions.md#migrations) and
[conventions.md](conventions.md#portable-sql). The entities and how they relate are in
[data-model.md](data-model.md).

## Scheduled jobs

`scheduleJobs()` in `main.go` registers cron tasks via `codnect.io/chrono`. The table there is
the source of truth; at the time of writing:

| Job | When | Enabled by |
|---|---|---|
| `SendSundayReminders` | Sunday 18:00 | always |
| `ProcessLastWeek` | Monday 08:00 | always — see [seasons-and-goals.md](seasons-and-goals.md) |
| `OllamaPreCacheForAllUsers` | daily 06:00 (and at startup) | `Ollama.Enabled` |
| `StravaSyncWeekForAllUsers` | hourly, `:00` | `StravaEnabled` |
| `HevyEventsSyncForAllUsers` | hourly, `:30` | `HevyEnabled` |
| `MediaReconcileForAllUsers` | hourly, `:45` | `Media.Enabled` |
| `IntegrationHealthCheckForAllUsers` | daily 07:15 | `Media.Enabled` |

## Configuration

Config comes from `config/config.json`, flags, or env vars (mapped to flags in
`entrypoint.sh`). Some settings — the integration blocks such as `Media`, `Ollama` and the
per-provider flags — exist only in the JSON. Typical local run:

```bash
dbpassword=pass ./treningheten --port 8080 --dbip 127.0.0.1 --dbusername user --dbname treningheten --generateinvite true
```

The DB and SMTP passwords are read from the `dbpassword` / `smtppassword` environment
variables (precedence: `--dbpassword` / `--smtppassword` flag > env var > config file). Prefer
the env vars: flag values are visible in the process list.

`--generateinvite true` prints a one-time invite code at startup, needed to register the
first user. `--disablesmtp true` turns mail off, which also skips email verification.

## Integrations

Each has its own document: [strava.md](strava.md), [hevy.md](hevy.md), [gear.md](gear.md),
[media.md](media.md) (Plex, Spotify, Audiobookshelf), [ollama.md](ollama.md),
[push-notifications.md](push-notifications.md), and [integration-health.md](integration-health.md)
for noticing broken connections. Strava and Hevy are gated by config flags; media has a tenant
flag plus one per provider, and its endpoints answer 404 when off.

## Versioning

The version string is the literal placeholder `{{RELEASE_TAG}}` in `files/config.go`; the
release workflow replaces it at build time, so locally it stays the placeholder. Releases are
cut by publishing a GitHub release (`.github/workflows/release.yaml`).
