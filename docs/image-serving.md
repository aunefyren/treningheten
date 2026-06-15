# Image serving

How profile and achievement images are served and consumed. Read this before touching
the image endpoints or adding a page that shows user/achievement images.

## Endpoints

- `GET /api/auth/users/:user_id/image` — a user's profile photo.
- `GET /api/auth/achievements/:achievement_id/image` — an achievement badge.

Both accept an optional `?thumbnail=true` (250×250 box vs 1000×1000). Handlers live in
`controllers/image.go`.

They **return raw image bytes** (`image/jpeg`, or `image/svg+xml` for the profile
default placeholder) — not JSON. This lets the frontend embed them directly in an
`<img src>`, so the browser caches, dedupes and parallelises them natively. (They used
to return base64 inside JSON, fetched per-occurrence over XHR; that was ~33% larger,
uncached, and re-fetched the same user many times per page.)

## Auth: cookie or header

`<img>` tags can't send an `Authorization` header, so these routes use the
`middlewares.AuthImageReadOnly()` middleware (in `main.go` they're a separate
`/api/auth` group from the header-only `Auth(false)` group). It accepts the access token
from the `Authorization` header **or**, failing that, the `treningheten` cookie, then
runs the shared `Authenticate()` enabled/verified checks. It requires only a valid login
(no scope checks) because these images are already visible to any logged-in user, and
the cookie is `samesite=strict` so it isn't sent cross-site.

## Caching

Two layers:

1. **HTTP** — `serveImageBytes` sets `Cache-Control: private, max-age=300` and an
   `ETag` (content hash), and honours `If-None-Match` (304). Repeat loads within the
   window don't hit the server at all.
2. **Server-side resize cache** — `loadResizedImageCached` caches the resized bytes in
   memory keyed by source path + target size, so the expensive Lanczos3 resize runs once
   per `(image, size)`. It **self-invalidates on the source file's modification time**,
   so a re-uploaded profile photo (`UpdateUserProfileImage` rewrites the `.jpg`) is
   picked up with no explicit invalidation call.

Because of the HTTP `max-age`, a user who just changed their own photo could otherwise
see a stale cached copy; the account page (`account.js`) appends a `?v=<timestamp>`
cache-buster to its own avatar to always show the current one.

## Frontend

Embed images directly; never re-introduce XHR→base64 fetching. Use the helpers in
`functions.js`:

```js
<img src="${profileImageURL(userID, true)}"      onerror="${IMAGE_FALLBACK_ONERROR}">
<img src="${achievementImageURL(achievementID, true)}" onerror="${IMAGE_FALLBACK_ONERROR}">
```

`IMAGE_FALLBACK_ONERROR` swaps in the local `barbell.gif` placeholder if a request fails
(e.g. an expired session, or an achievement with no image — the achievement endpoint
404/400s rather than serving a default).

## Uploads

Uploads are unaffected: the account page still POSTs the new photo as base64 inside the
user-update payload, decoded by `UpdateUserProfileImage` (`Base64ToImageBytes`). Only the
read/display path changed to raw bytes.
