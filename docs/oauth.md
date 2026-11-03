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
| `POST /api/oauth/revoke` | RFC 7009 | client | Revoke a refresh token (first-party: from the `treningheten_rt` cookie when no `token` is sent) |

The token endpoint accepts `application/x-www-form-urlencoded` and supports the
grants `authorization_code`, `refresh_token`, and `password`. Errors follow
RFC 6749 §5.2 (`{"error": "...", "error_description": "..."}`) with
`Cache-Control: no-store`.

## Token model

- **Access token** — HS256 JWT signed with `ConfigFile.PrivateKey`. Claims:
  `id` (user UUID), `admin`, `scope`, `client_id`, plus `iss`/`aud` (both the
  resource = external URL, when configured), `exp`/`nbf`/`iat`. TTL = 60 min
  (`auth.AccessTokenTTL`). Expiry is checked strictly; `nbf` (the minting second) is
  accepted up to a minute in the future (`notBeforeLeeway`), because a wall clock that
  steps back (VM time sync, NTP) would otherwise reject a token minted a moment ago.
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
- `admin` — admin endpoints; implies write. **Reserved** for the first-party client and
  Personal Access Tokens: dynamically registered clients can never hold it (see below).

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
-> { access_token, token_type: "Bearer", expires_in, scope }   + Set-Cookie: treningheten_rt
```

For the first-party client the refresh token never appears in the response body or in
script: the server sets it as the `treningheten_rt` cookie (`HttpOnly; SameSite=Strict;
Path=/api/oauth; Max-Age = auth.RefreshTokenTTL`, plus `Secure` when the external URL is
https, the same rule as HSTS via `middlewares.IsHTTPSURL`). The helpers are
`setRefreshCookie` / `clearRefreshCookie` in `controllers/oauth_cookie.go`; every
first-party token issuer uses them: the password, refresh and authorization-code grants
(via `writeTokenResponse`) and the two `auth.IssueTokenSet(..., FirstPartyClientID)` call
sites in `controllers/user.go` (verify and account update). Third-party clients (DCR/MCP)
are unchanged and still get `refresh_token` in the body.

- **Refresh grant:** for the first-party client with no `refresh_token` form field, the token
  is read from the cookie and rotated into a new cookie. A failed cookie-borne refresh
  (invalid, expired, reused) clears the cookie. The cookie is never honoured for any other
  `client_id`.
- **Revoke:** `POST /api/oauth/revoke` with `client_id=treningheten-web` and no `token`
  falls back to the cookie, revokes it and clears it. This is what `logout()` calls, since
  script cannot clear an `HttpOnly` cookie.
- **Access token:** stored by `store_tokens` in the JS-readable `treningheten` cookie
  (`path=/; samesite=strict`, plus `secure` over HTTPS) with a lifetime equal to
  `expires_in` (60 min). It has to stay readable because `AuthImageReadOnly` /
  `ImageRequestUserID` authenticate `<img>` requests from it. On a missing or rejected
  access token the frontend attempts one silent refresh (`credentials` included, no token
  in the body) and treats a 400/401 as logged out (see `web/js/functions.js`).
- **Session hint:** because script cannot see the refresh cookie, `store_tokens` sets a
  `treningheten_session` flag in `localStorage` (removed by `clear_tokens`, `logout()` and a
  failed refresh) and the silent refresh only runs when it is set. Without it every
  logged-out page load would hit `/api/oauth/token` and burn its per-IP rate limit. The hint
  is not a credential.
- **Legacy cookie:** the old JS-set `treningheten_refresh` cookie is ignored by the server
  (the new cookie has a different name so the two can never be confused) and deleted by
  `functions.js` on load. There is no migration: users who only had the old cookie log in
  again once, when their access token expires.
- **CSRF:** `SameSite=Strict`, the narrow `Path` and the absence of CORS mean a cross-site
  page can neither send the cookie nor read the response.

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
`models.AdvertisedScopes` (`api`, `api:read`, `api:write`; never `admin`).

Registration is only available while MCP is enabled (`mcp_enabled`): otherwise
`POST /api/oauth/register` returns `404` and the authorization-server metadata omits
`registration_endpoint`.

### The reserved admin scope

`admin` is only ever carried by tokens of the first-party client (`FirstParty`) and by
PATs. Enforced in four places: DCR (`narrowScopeToSupported`), the authorize/consent
path (`narrowScope` drops it for non-first-party clients, even if the stored client scope
has it), the refresh grant (`auth.RefreshTokenSet` strips it from the rotated set for
non-first-party clients, so an older chain loses it at its next refresh; access tokens
already issued expire on their own), and discovery (`models.AdvertisedScopes`).

### Pruning

A daily job (`controllers.PruneUnusedOAuthClients`, 03:30) deletes non-first-party
clients created more than 7 days ago that have no live (unrevoked, unexpired) refresh
token, together with their authorization codes and refresh-token rows.

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
