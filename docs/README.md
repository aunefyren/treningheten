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

## Auth & integrations

- [oauth.md](oauth.md) — Treningheten as an OAuth 2.0 authorization server.
- [pat.md](pat.md) — Personal Access Tokens.
- [mcp.md](mcp.md) — Model Context Protocol server (read-only, personal tools for LLM
  clients).
- [strava.md](strava.md) — Strava integration: OAuth connect, the token-lifecycle
  scheme, hourly sync, rate limiting, and activity-to-exercise conversion.
