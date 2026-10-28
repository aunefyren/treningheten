# Treningheten docs

Feature and decision documentation. The repo root `CLAUDE.md` holds the agent workflow and
the rules every change follows; plans and open work live in GitHub issues, not here.

## Architecture & conventions

- [architecture.md](architecture.md) — packages, routing and auth groups, how the frontend
  is served through Go templates, persistence, scheduled jobs, configuration, versioning.
- [conventions.md](conventions.md) — code conventions: naming (camelCase, `ID`
  uppercase), Go error handling, API response shapes, frontend JS patterns, tests, and
  migrations.
- [styleguide.md](styleguide.md) — the frontend **visual system**: the CSS layer/file
  layout, design tokens, and the shared components (`.btn`, cards, chips, the `.trm`
  modal, `.u-*` utilities). Read before building or restyling UI.
- [vocabulary.md](vocabulary.md) — the words the UI uses (workout, activity, activity type,
  day, weekly goal, …), what each maps to in the code, and the rules for new copy.
- [data-model.md](data-model.md) — what the core entities are and how they relate: the
  three struct flavors (model / `Object` / DTO), the domain-spine ER diagram, and a
  per-entity reference. Read this before touching the data layer.
- [data-conventions.md](data-conventions.md) — data-model gotchas: the `Convert*Object`
  read layer, durations stored as seconds, per-operation units, soft deletes.
- [`docker-test/README.md`](../docker-test/README.md) — the live Docker test harness:
  the app built from the checkout plus Mailpit, Audiobookshelf and optional
  MariaDB/Ollama, with a seed script that registers, verifies and connects a user.

## Domain

- [seasons-and-goals.md](seasons-and-goals.md) — seasons (time-boxed competitions),
  goals (how a user participates), weekly completion, sick leave, and the weekly
  processing loop.
- [flexible-workouts.md](flexible-workouts.md) — **saved workouts**: a season can let
  participants save workouts beyond their goal and use them to make up a short week; the
  rule (`nextCarryBank`), the per-week opt-in, and where it shows in the UI.
- [streaks.md](streaks.md) — the **two** streak systems (personal activity streaks vs
  within-season goal streaks) and how each is computed.
- [exercises.md](exercises.md) — the `/exercises` workout timeline: the searchable/sortable
  session feed (`GET /auth/activities`), its browse vs find modes, and the query-time aggregation;
  the day page (`/exercises/:id`) with its week header; the workout read view.
- [activity-feed.md](activity-feed.md) — the front-page **Activity feed** module: who you see
  (everyone you have ever shared a season with — co-membership instead of friend requests),
  the `ShareActivities` consent, and the peer query.
- [wheel-customization.md](wheel-customization.md) — per-user wheel appearance (color,
  border, emoji): storage, account-page picker, validation, and the
  distinct/stable color assignment.
- [statistics.md](statistics.md) — the `/statistics` tabs: the period overview, personal
  records and best efforts (stored, with the versioned stream backfill), per-type statistics
  from rollups, seasons and body weight; what counts and how units are kept apart.
- [heatmap.md](heatmap.md) — private per-user GPS activity heatmap on `/statistics`
  (Leaflet + Leaflet.heat over stored Strava `latlng` streams).
- [admin-stats.md](admin-stats.md) — aggregate usage statistics on the admin panel
  (users in seasons / with notifications / with Strava, achievement completion).
- [admin-users.md](admin-users.md) — disabling and re-enabling users from the admin
  panel: access cut-off, leaving unfinished seasons, and keeping history readable.
- [invites.md](invites.md) — invite codes and `/register?invite=` links (admin "Copy link",
  the startup `--generateinvite` link).
- [push-notifications.md](push-notifications.md) — web push categories, what each opens, and
  how pages close the notifications they answer.
- [image-serving.md](image-serving.md) — how profile/achievement images are served (raw
  bytes via `<img src>`, cookie-or-header auth, HTTP + server-side resize caching).
- [gear.md](gear.md) — gear tracking (shoes/bikes): manual + Strava-imported equipment,
  per-operation linkage, computed distance, and the session-level builder selector.

## Auth & integrations

- [security.md](security.md) — cross-cutting hardening: rate limiting and the bcrypt
  cost budget, trusted-proxy/client-IP resolution, session invalidation on password
  change, config file permissions, accepted risks, and the audit's "checked and found
  sound" list.
- [oauth.md](oauth.md) — Treningheten as an OAuth 2.0 authorization server.
- [pat.md](pat.md) — Personal Access Tokens.
- [mcp.md](mcp.md) — Model Context Protocol server (read-only, personal tools for LLM
  clients).
- [strava.md](strava.md) — Strava integration: OAuth connect, the token-lifecycle
  scheme, hourly sync, rate limiting, and activity-to-exercise conversion.
- [activity-dedup.md](activity-dedup.md) — duplicate activities across import sources:
  the user's source ranking, what counts as a duplicate, hiding vs restoring, and the
  reconcile when the ranking changes.
- [hevy.md](hevy.md) — Hevy integration: per-user API-key auth (no OAuth), account
  setup/validation, and the planned workout sync + exercise mapping (WIP).
- [ollama.md](ollama.md) — AI-generated front-page greeting: the pre-computed payload
  (incl. the optional `latest_workout` block), caching, and scheduling.
- [integration-health.md](integration-health.md) — noticing a broken integration
  connection (revoked token vs outage), notifying the user once, and re-pulling the
  gap on recovery, for Plex, Spotify, Audiobookshelf, Strava and Hevy.
- [media.md](media.md) — media/audio integration: overlaying listening history onto
  activities, the per-provider connection model, the playback timeline, and the
  tenant + per-provider feature flags (Plex first; WIP).
