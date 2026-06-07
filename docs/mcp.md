# MCP Server

Treningheten exposes a [Model Context Protocol](https://modelcontextprotocol.io)
server so LLM clients (Claude Desktop, etc.) can read a user's training data and,
for example, give feedback on their latest run. This is Phase 3 of the auth work
(Phase 1: OAuth 2.0 — see [oauth.md](oauth.md); Phase 2: PATs — see [pat.md](pat.md)).

## Design decisions

| Decision | Choice | Rationale |
|---|---|---|
| SDK | Official `github.com/modelcontextprotocol/go-sdk` (v1.6.x) | Spec-correct protocol handling, schema inference, Streamable HTTP handler. |
| Transport | Streamable HTTP at `POST/GET /mcp` | Remote transport that reuses the OAuth bearer auth already built. |
| Auth | OAuth access token **or** Personal Access Token (bearer) | A user can do the browser OAuth flow or paste a `trh_pat_…` token into their client. |
| v1 tools | Read-only | Serves the "summarise / give feedback" use case; a read-only token suffices. |
| Per-tool scope | Checked in the MCP layer, not by HTTP method | Every MCP call is a `POST`, so the method-based read/write rule can't apply; the endpoint requires the read scope and bypasses the generic GET-only check. |
| User scoping | Tools act only on the authenticated user's data | The principal is resolved from the token per request. |

## Endpoint & auth flow

- **Endpoint:** `/mcp` (handled by `controllers.MCPHandler`).
- An unauthenticated request gets `401` with
  `WWW-Authenticate: Bearer … resource_metadata="…/.well-known/oauth-protected-resource"`,
  which lets an MCP client **discover** the authorization server and run the
  authorization-code + PKCE flow (with dynamic client registration) built in Phase 1.
- The gin wrapper authenticates the bearer (token or PAT), requires the `api`/`api:read`
  scope, and passes the principal to the per-request MCP server via the request context.

## Tools (v1, read-only)

| Tool | Arguments | Returns |
|---|---|---|
| `whoami` | — | Profile: name, email, admin, member-since |
| `list_weights` | `limit?` | Body-weight entries, newest first |
| `get_latest_weight` | — | Most recent weight entry |
| `list_exercises` | `action?`, `limit?` | Logged activities (date, action, type, duration, per-set distance/time/reps/weight), newest first; `action` filters by exercise type (e.g. `Run`) |
| `get_statistics` | — | Activity counts (month / year / all-time) + total distance and time |

Durations are exposed in **seconds** (not nanoseconds) for LLM friendliness.

### Example use case

> "Give me feedback on my newest run."

The client calls `list_exercises(action:"Run", limit:1)`, receives the date,
duration and per-set distance/time, and composes feedback.

## Connecting a client

Point an MCP client at `https://<your-host>/mcp`. Either:
- let it run the OAuth flow (it will discover the auth server from the 401), or
- configure it with a Personal Access Token (`Authorization: Bearer trh_pat_…`),
  created from the **Developer access tokens** section on the account page. A
  read-only PAT is enough.

## Key code locations

- `controllers/mcp.go` — server construction, tool registration, gin handler + auth.
- `controllers/mcp_data.go` — assembly of the operation-centric model into flat DTOs.
- `models/mcp.go` — MCP DTOs (`MCPProfile`, `MCPWeight`, `MCPActivity`, `MCPStatistics`).
- `middlewares/auth.go` — `Authenticate` (shared token/PAT validation) + `BearerChallenge`.
- `main.go` — `router.Any("/mcp", controllers.MCPHandler())`.

## Known limitations / next steps

- Read-only only; write tools (`log_exercise`, `add_weight`) are a follow-up and
  would require the `api:write` scope and per-tool write checks.
- `get_statistics` is a focused subset (counts + distance/time totals). The richer
  web statistics (streaks, per-action tops/averages) live in a 285-line controller
  and would need to be refactored into a reusable function to expose here.
- Seasons / leaderboard / goals tools were deferred from v1.
- Granular per-resource scopes are still not implemented (coarse read/write/admin).
