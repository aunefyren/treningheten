# Docker test harness

A reusable live-test rig: Treningheten **built from this checkout** (uncommitted
changes included, not pulled from a registry) alongside the self-hostable services
it integrates with, on one Docker network. It exists for the checks the Go test
suite can't do — a real browser session, real mail, a real media provider, and
optionally a real MySQL-family schema migration.

Everything under `data/` is gitignored: Treningheten's config directory (config,
SQLite database, log), uploaded profile images, and each companion's state. What a
specific test run found belongs in a GitHub issue or the relevant `docs/*.md`, not here.

## What's in it

| Service | URL | Why |
| --- | --- | --- |
| `treningheten` | http://localhost:8080 \* | the app, `generateinvite` on, SMTP pointed at Mailpit |
| `mailpit` | http://localhost:8025 \* | catches every mail (verification, password reset, Sunday reminders) |
| `audiobookshelf` | http://localhost:13378 \* | the one media provider that runs fully offline (URL + token, no OAuth app) |
| `mariadb` *(profile `mysql`)* | — | production runs MySQL; collation / FK (errno 150) problems only show up there |
| `postgres` *(profile `postgres`)* | — | the third supported backend |
| `ollama` *(profile `ollama`)* | — | front-page AI messages; `ollama-pull` fetches `llama3.2:1b` once |

\* The default host and ports, used by the first checkout that runs `setup.sh`. Other
worktrees get their own — see [Several worktrees at once](#several-worktrees-at-once);
`seed.sh` prints the real URLs.

Not included, because they can't be set up without a human: **Strava** and **Spotify**
(need a registered app and a reachable redirect URI), **Plex** (needs a plex.tv claim
token), **Hevy** (needs a Pro API key). They can still be switched on by hand in
`data/treningheten/config/config.json`.

## Run it

Needs Docker with Compose v2.

```
./setup.sh                    # first time only: writes .env, creates ./data, seeds config.json (SQLite)
docker compose up -d --build
./seed.sh                     # registers the admin user, verifies it, connects Audiobookshelf
```

`seed.sh` prints the logins when it's done:

- Treningheten: `admin@test.local` / `Password123` (the first user, so admin)
- Audiobookshelf: `root` / `root`

It runs inside a throwaway alpine container on the harness network (so the host needs
neither `curl` nor `jq`), and every step checks first and skips what's already done,
so re-running it is harmless. What it does:

1. initialises Audiobookshelf's root user and mints an ABS API key;
2. registers the first Treningheten user with the invite code the app wrote to its
   log at startup, then reads the verification code out of the mail in Mailpit and
   verifies the account — the real registration flow, not a DB shortcut;
3. connects that user's Audiobookshelf provider (`POST /api/auth/media/audiobookshelf/connect`).

Override the defaults with `TT_EMAIL`, `TT_PASSWORD`, `ABS_USER`, `ABS_PASSWORD`.

Tear down, keeping `./data` so the next run picks up where this one left off:

```
docker compose down
```

`./clean.sh` stops everything and deletes `./data` to start completely clean. Use it
rather than `rm -rf data`: the containers write some of `data/` as root.

## Several worktrees at once

Agents each work in their own git worktree, and all of them share one Docker daemon. So
`setup.sh` writes a gitignored `.env` the first time it runs, which `docker compose` picks
up from this directory:

| Variable | Value |
| --- | --- |
| `COMPOSE_PROJECT_NAME` | `tt-<worktree directory>` — its own containers, network and image |
| `TT_SLOT` | the lowest slot no other worktree of this repo has claimed in its `.env` |
| `TT_PORT`, `MAILPIT_PORT`, `ABS_PORT` | slot 0: 8080 / 8025 / 13378; slot *n*: `18000 + 10n`, +1, +2 |
| `PUBLIC_HOST` | where published ports are reachable: `localhost`, or the daemon's host when `DOCKER_HOST=tcp://<host>:…` (e.g. `dind`) |

Inside the harness network nothing changes — services still reach each other as
`treningheten:8080`, `mailpit:8025`, `audiobookshelf:80`. Only the host-side URLs differ,
and `seed.sh` prints them. Edit `.env` to pick other ports; it survives `clean.sh`.

A harness started before this existed ran as Compose project `docker-test` with fixed
`tt-*` container names; stop it once with `docker compose -p docker-test down`.

### Against MariaDB instead of SQLite

On a clean `data/`:

```
./setup.sh mysql
docker compose --profile mysql up -d --build
./seed.sh
```

Treningheten may restart a couple of times while MariaDB initialises; `restart:
unless-stopped` covers it. Handy for checking that a new model migrates cleanly on
MySQL before it reaches production.

### Against Postgres instead of SQLite

On a clean `data/`:

```
./setup.sh postgres
docker compose --profile postgres up -d --build
./seed.sh
```

As with MariaDB, Treningheten may restart while Postgres initialises.

### With Ollama

```
docker compose --profile ollama up -d
```

then set `"ollama": {"enabled": true, …}` in `data/treningheten/config/config.json`
(the URL and model are already filled in) and `docker compose restart treningheten`.

## Testing the soundtrack overlay

ABS only has listening history once something is played. Add a library in ABS
(**Settings → Libraries → Add**, folder `/audiobooks`, which is mounted from
`data/abs/audiobooks`), drop an audio file in there, play it in the ABS web player
while a manual session with a start time is open in Treningheten, then use the
session's 🎧 button to pull it. See `docs/media.md`.

## Watching it work

```
docker compose logs -f treningheten
```

The log level is `debug` in the shipped config. The same log is also written to
`data/treningheten/config/treningheten.log`.
