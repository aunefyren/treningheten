# Treningheten docs

Feature and decision documentation. See the repo root `CLAUDE.md` for the
architecture overview.

## Conventions

- [conventions.md](conventions.md) — code conventions: naming (camelCase, `ID`
  uppercase), Go error handling, API response shapes, frontend JS patterns, tests, and
  migrations.
- [styleguide.md](styleguide.md) — the frontend **visual system**: the CSS layer/file
  layout, design tokens, and the shared components (`.btn`, cards, chips, the `.trm`
  modal, `.u-*` utilities). Read before building or restyling UI.
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
- [streaks.md](streaks.md) — the **two** streak systems (personal activity streaks vs
  within-season goal streaks) and how each is computed.
- [exercises.md](exercises.md) — the `/exercises` workout timeline: the searchable/sortable
  session feed (`GET /auth/activities`), its browse vs find modes, and the query-time aggregation.
- [activity-feed.md](activity-feed.md) — the front-page **Activities** module: who you see
  (everyone you have ever shared a season with — co-membership instead of friend requests),
  the `ShareActivities` consent, and the peer query.
- [wheel-customization.md](wheel-customization.md) — per-user wheel appearance (color,
  border, emoji): storage, account-page picker, validation, and the
  distinct/stable color assignment.
- [heatmap.md](heatmap.md) — private per-user GPS activity heatmap on `/statistics`
  (Leaflet + Leaflet.heat over stored Strava `latlng` streams).
- [admin-stats.md](admin-stats.md) — aggregate usage statistics on the admin panel
  (users in seasons / with notifications / with Strava, achievement completion).
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
- [hevy.md](hevy.md) — Hevy integration: per-user API-key auth (no OAuth), account
  setup/validation, and the planned workout sync + exercise mapping (WIP).
- [ollama.md](ollama.md) — AI-generated front-page greeting: the pre-computed payload
  (incl. the optional `latest_workout` block), caching, and scheduling.
- [integration-health.md](integration-health.md) — noticing a broken integration
  connection (revoked token vs outage), notifying the user once, and re-pulling the
  gap on recovery. Plex first; the pattern for adding the other providers.
- [media.md](media.md) — media/audio integration: overlaying listening history onto
  activities, the per-provider connection model, the playback timeline, and the
  tenant + per-provider feature flags (Plex first; WIP).
