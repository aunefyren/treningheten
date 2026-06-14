# Treningheten docs

Feature and decision documentation. See the repo root `CLAUDE.md` for the
architecture overview.

## Domain

- [seasons-and-goals.md](seasons-and-goals.md) — seasons (time-boxed competitions),
  goals (how a user participates), weekly completion, sick leave, and the weekly
  processing loop.
- [streaks.md](streaks.md) — the **two** streak systems (personal activity streaks vs
  within-season goal streaks) and how each is computed.
- [data-conventions.md](data-conventions.md) — cross-cutting gotchas: the
  `Convert*Object` read layer, durations stored as seconds, per-operation units, soft
  deletes.
- [wheel-customization.md](wheel-customization.md) — per-user wheel appearance (color,
  border, emoji): storage, account-page picker, validation, and the
  distinct/stable color assignment.
- [heatmap.md](heatmap.md) — private per-user GPS activity heatmap on `/statistics`
  (Leaflet + Leaflet.heat over stored Strava `latlng` streams).
- [admin-stats.md](admin-stats.md) — aggregate usage statistics on the admin panel
  (users in seasons / with notifications / with Strava, achievement completion).

## Auth & integrations

- [oauth.md](oauth.md) — Treningheten as an OAuth 2.0 authorization server.
- [pat.md](pat.md) — Personal Access Tokens.
- [mcp.md](mcp.md) — Model Context Protocol server (read-only, personal tools for LLM
  clients).
- [strava.md](strava.md) — Strava integration: OAuth connect, the token-lifecycle
  scheme, hourly sync, rate limiting, and activity-to-exercise conversion.
- [hevy.md](hevy.md) — Hevy integration: per-user API-key auth (no OAuth), account
  setup/validation, and the planned workout sync + exercise mapping (WIP).
