# OAuth 2.0 Authorization Server

Treningheten is its own OAuth 2.0 / 2.1 authorization server. This document covers
the endpoints, token model, flows, and the design decisions behind them. It is
Phase 1 of a larger effort (Phase 2: Personal Access Tokens, Phase 3: MCP server).

## Design decisions

| Decision | Choice | Rationale |
|---|---|---|
| AS scope | Full authorization server (authorization-code + PKCE, DCR, discovery) plus password & refresh-token grants | The end goal (an MCP server) requires a spec-compliant AS; building it now means MCP plugs straight in. |
| Access token | Stateless JWT (HS256), validated with no per-request DB hit | Cheap validation; keeps the hot path stateless. |
| State that must persist | Minimal tables: OAuth clients (DCR), single-use authorization codes, revocable/rotating refresh tokens | DCR, single-use codes, and revocation/rotation cannot be done statelessly. The refresh-token table is reused by Phase 2 PATs. |
| Refresh tokens | Opaque, hashed at rest, rotated on use, with reuse detection | Standard OAuth; enables revocation and detection of stolen tokens. |
| Web app login | First-party resource-owner password grant | OAuth 2.1 deprecates ROPC, but it is acceptable for the trusted first-party client and avoids a heavy frontend rework. Third-party/MCP clients must use authorization-code + PKCE. |
| Migration | Clean break: the old `POST /api/open/tokens/register` was removed and the web frontend updated in lockstep | Single self-hosted user base; no need to maintain a legacy shim. |

## Endpoints

| Method & path | Spec | Auth | Purpose |
|---|---|---|---|
| `GET /.well-known/oauth-authorization-server` | RFC 8414 | public | Authorization-server metadata |
| `GET /.well-known/oauth-protected-resource` | RFC 9728 | public | Protected-resource metadata (MCP discovery) |
| `GET /authorize` | — | browser | Consent/login HTML page (the advertised `authorization_endpoint`) |
| `GET /api/oauth/authorize` | RFC 6749 §4.1 | public | Validates an authorization request, returns client details for the consent screen |
| `POST /api/oauth/authorize/decision` | RFC 6749 §4.1 | Bearer | Records approval/denial; issues an authorization code on approval |
| `POST /api/oauth/token` | RFC 6749 §3.2 | client | Token endpoint (form-encoded) |
| `POST /api/oauth/register` | RFC 7591 | public | Dynamic Client Registration |
| `POST /api/oauth/revoke` | RFC 7009 | client | Revoke a refresh token |

The token endpoint accepts `application/x-www-form-urlencoded` and supports the
grants `authorization_code`, `refresh_token`, and `password`. Errors follow
RFC 6749 §5.2 (`{"error": "...", "error_description": "..."}`) with
`Cache-Control: no-store`.

## Token model

- **Access token** — HS256 JWT signed with `ConfigFile.PrivateKey`. Claims:
  `id` (user UUID), `admin`, `scope`, `client_id`, plus `iss`/`aud` (both the
  resource = external URL, when configured), `exp`/`nbf`/`iat`. TTL = 60 min
  (`auth.AccessTokenTTL`).
- **Refresh token** — opaque random string; only its SHA-256 hash is stored
  (`OAuthRefreshToken`). TTL = 30 days (`auth.RefreshTokenTTL`). Rotated on every
  use; the old token is revoked and linked to its replacement via `rotated_to`.
  Presenting an already-rotated token triggers reuse detection and revokes the
  whole chain.
- **Authorization code** — opaque, hashed, single-use (`consumed_at`),
  short-lived (10 min), bound to client + redirect URI + PKCE challenge.

### Audience binding

When `treningheten_external_url` is set, tokens carry `iss`/`aud` equal to it and
the middleware validates `aud` on every request. MCP requires this, so the
external URL **must** be configured for MCP use. When unset, audience binding is
skipped and discovery documents fall back to the request's scheme + host.

## Scopes

`models.SupportedScopes`:

- `api` — full read+write API access (the scope carried by OAuth access tokens).
- `api:read` — read-only access (`GET`/`HEAD` only). Used by read-only PATs.
- `api:write` — explicit read+write access.
- `admin` — admin endpoints; implies write.

Enforcement (in `middlewares.Auth`, helpers in `auth/scope.go`):
- a valid token is always required;
- `Auth(true)` additionally requires the `admin` scope **in the token** and the
  user to be an admin in the DB;
- write (non-`GET`/`HEAD`) requests require a write-capable scope, so read-only
  tokens are limited to safe methods.

Scope matching is on whole space-delimited tokens (so `api:read` is never
mistaken for `api`). The `scope` claim is carried on every token. Granular
per-endpoint / per-resource scopes are still deferred to the MCP phase. See
[pat.md](pat.md) for how Personal Access Tokens use these scopes.

## Flows

### First-party web login (password grant)

```
POST /api/oauth/token
  grant_type=password&client_id=treningheten-web&username=<email>&password=<pw>
-> { access_token, token_type: "Bearer", expires_in, refresh_token, scope }
```

The web frontend stores the access token in the `treningheten` cookie and the
refresh token in `treningheten_refresh`. On a rejected access token it silently
calls the refresh grant once (see `web/js/functions.js`), then retries.

### Authorization code + PKCE (third-party / MCP)

1. Client directs the browser to `GET {external}/authorize?response_type=code&client_id=…&redirect_uri=…&scope=…&state=…&code_challenge=…&code_challenge_method=S256`.
2. The consent page (`web/html/authorize.html` + `web/js/authorize.js`) validates
   the request via `GET /api/oauth/authorize`, ensuring the user is logged in
   (redirecting to `/login?return=…` if not).
3. On approval it `POST`s to `/api/oauth/authorize/decision` (Bearer token),
   which returns a redirect to `redirect_uri?code=…&state=…`.
4. The client exchanges the code at the token endpoint with its `code_verifier`:
   ```
   POST /api/oauth/token
     grant_type=authorization_code&client_id=…&code=…&redirect_uri=…&code_verifier=…
   ```

Only `S256` PKCE is supported; `redirect_uri` must exactly match a registered URI.

### Dynamic Client Registration

```
POST /api/oauth/register
  { "client_name", "redirect_uris", "grant_types"?, "response_types"?,
    "scope"?, "token_endpoint_auth_method"? }
```

`token_endpoint_auth_method: "none"` (default) creates a public client (PKCE, no
secret). Any other method creates a confidential client and returns a one-time
`client_secret` (stored bcrypt-hashed). Requested scope is narrowed to
`SupportedScopes`.

## Key code locations

- `models/oauth.go` — tables, DTOs, scope constants, `FirstPartyClientID`.
- `database/oauth.go` — client / auth-code / refresh-token persistence, rotation,
  reuse-chain revocation.
- `auth/auth.go` — JWT issuance/validation, refresh-token issue/rotate.
- `controllers/oauth.go` — token endpoint (password, refresh) + client auth.
- `controllers/oauth_authorize.go` — authorize info/decision + code grant + PKCE.
- `controllers/oauth_clients.go` — DCR + revocation.
- `controllers/oauth_discovery.go` — `.well-known` metadata.
- `middlewares/auth.go` — Bearer validation (RFC 6750) + admin gate.
- `database/seed.go` — `SeedOAuthClients` seeds the first-party `treningheten-web` client.
- `web/js/login.js`, `web/js/functions.js`, `web/html/authorize.html`,
  `web/js/authorize.js` — frontend.

## Known limitations (Phase 1)

- Granular per-endpoint scope enforcement is not yet wired (coarse auth/admin only).
- Silent refresh in the web app happens at page load / on rejected token, not on
  every individual in-flight request.
- The first-party web client uses the deprecated password grant by design.
