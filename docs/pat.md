# Personal Access Tokens (PATs)

Personal Access Tokens are long-lived, user-created API credentials for scripts
and integrations. This is Phase 2 of the auth work (Phase 1: OAuth 2.0, see
[oauth.md](oauth.md); Phase 3: MCP server).

## Design decisions

| Decision | Choice | Rationale |
|---|---|---|
| Token type | Opaque, DB-backed, SHA-256 hashed at rest | PATs must be listable and revocable; stateless JWTs can't be revoked. Reuses the Phase 1 hashing approach. |
| Format | `trh_pat_<random>` | The prefix lets `middlewares.Auth` cheaply distinguish a PAT from an access-token JWT (prefix → DB lookup; otherwise → JWT parse). |
| Storage | Dedicated `PersonalAccessToken` table | PATs have a name, no rotation chain, and no client binding — different enough from `OAuthRefreshToken` to warrant their own table. |
| Permissions | `api:read` / `api:write`, plus optional `admin` | A read-only PAT may only call safe methods (`GET`/`HEAD`), enforced generically in the middleware. A PAT can never exceed the creating user's privileges. |
| Expiry | Required, 1–365 days (`models.PATMaxLifetimeDays`) | No perpetual tokens; bounds the blast radius of a leak. |
| Surface | Self-service API + a collapsible section on `/account` | Power-user feature, kept out of the way like the Strava integration block. |

## Scope model

PATs extend the Phase 1 scopes (see [oauth.md](oauth.md#scopes)):

| Scope | Meaning |
|---|---|
| `api:read` | Read-only — only `GET`/`HEAD` requests are allowed |
| `api:write` | Read + write |
| `admin` | Admin endpoints; implies write; only grantable to admin users |
| `api` | Legacy full read+write scope carried by OAuth access tokens (treated as write) |

Enforcement lives in `middlewares.Auth` via `auth.ScopeCanWrite` / `auth.ScopeHasAdmin`.
Matching is on whole space-delimited tokens, so `api:read` is never mistaken for
`api`. `Auth(true)` now requires the `admin` scope **in the token** *and* the user
to be an admin in the DB.

## Endpoints

All are under `/api/auth` (a logged-in session or any valid token can manage the
caller's own tokens). A token only ever acts on the calling user's PATs.

| Method & path | Purpose |
|---|---|
| `POST /api/auth/pats` | Create a PAT. Returns the plaintext token **once**. |
| `GET /api/auth/pats` | List the caller's active PATs (metadata only — never the token). |
| `DELETE /api/auth/pats/:pat_id` | Revoke one of the caller's PATs. |

### Create request

```json
{
  "name": "my laptop script",
  "scope": "api:read",       // "api:read" (default) or "api:write"
  "admin": false,            // include the admin scope (admins only)
  "expires_in_days": 90      // 1..365, required
}
```

### Create response

```json
{
  "message": "Token created.",
  "data": {
    "token": "trh_pat_…",     // shown once
    "pat": { "id": "…", "name": "…", "scope": "api:read", "expires_at": "…", "last_used_at": null }
  }
}
```

## Using a PAT

Send it as a bearer token, exactly like an access token:

```
Authorization: Bearer trh_pat_…
```

The middleware validates it (exists, not revoked, not expired), records
`last_used_at`, then applies the same enabled/verified checks and the
read/write + admin scope checks as for access tokens.

## Key code locations

- `models/pat.go` — `PersonalAccessToken`, request/response DTOs, `PATPrefix`, `PATMaxLifetimeDays`.
- `database/pat.go` — create / get-by-hash / list / revoke / touch.
- `auth/scope.go` — `ScopeCanRead` / `ScopeCanWrite` / `ScopeHasAdmin`.
- `auth/auth.go` — `GeneratePATToken`, `HashToken`.
- `middlewares/auth.go` — `authenticate` (JWT or PAT → `Principal`) + read/write + admin enforcement.
- `controllers/pat.go` — CRUD handlers.
- `web/js/account.js` (+ `web/css/main.css`) — the `/account` "Developer access tokens" section.

## Known limitations

- Scopes are coarse (`read`/`write`/`admin`); per-resource scopes come with MCP.
- A read-only PAT is enforced purely by HTTP method (`GET`/`HEAD` = read); any
  endpoint that mutates via `GET` (there are none currently) would not be caught.
