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

## Domain

- [seasons-and-goals.md](seasons-and-goals.md) — seasons (time-boxed competitions),
  goals (how a user participates), weekly completion, sick leave, and the weekly
  processing loop.
- [streaks.md](streaks.md) — the **two** streak systems (personal activity streaks vs
  within-season goal streaks) and how each is computed.
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
- [media.md](media.md) — media/audio integration: overlaying listening history onto
  activities, the per-provider connection model, the playback timeline, and the
  tenant + per-provider feature flags (Plex first; WIP).
