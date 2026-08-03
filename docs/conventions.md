# Code conventions

How code in this repo is written: naming, error handling, API shapes, frontend
patterns, tests, and migrations. These are **de-facto standards** extracted from the
existing code — follow them in new code. Some older code predates them; see
[Legacy & refactoring](#legacy--refactoring).

For *data-model* conventions (the `Convert*Object` read layer, durations-as-seconds,
units, soft deletes) see [data-conventions.md](data-conventions.md). For architecture
and package layout see the root `CLAUDE.md`.

## Naming

**camelCase is the target everywhere it's legal.** Go exported identifiers must be
PascalCase (language rule); unexported Go identifiers and all JavaScript identifiers
should be camelCase.

**Write `ID` (and other initialisms) in uppercase within camelCase.** So `userID`,
`seasonID`, `goalID`, `exerciseDayID` — never `userId`/`Userid`. Struct fields are
`ID`, not `Id`. This matches Go's own initialisms convention and is applied to JS too.

**Go:**

- **HTTP handlers** are `APIXxx` (e.g. `APIGetOngoingSeasons`, `APIRegisterSeason`).
  Internal helpers that aren't handlers drop the prefix (`GetOngoingSeasonsFromDBForUserID`).
- **Database accessors** (in `database/`) are verb-first and say what they do:
  `Get…`, `Create…InDB`, `Update…`, `Delete…ByID`, `Verify…`. Examples:
  `GetSeasonByID`, `GetGoalFromUserWithinSeason`, `CreateGoalInDB`,
  `VerifyUserGoalInSeason`.
- **UUID parsing** uses a `…String` → `…Parsed` (or `…Int`) pair:
  ```go
  var seasonIDString = context.Param("season_id")
  seasonIDParsed, err := uuid.Parse(seasonIDString)
  ```
- **DTOs**: request bodies are `XxxCreationRequest` / `XxxUpdateRequest`; the enriched
  read structs are `XxxObject` (see the `Convert*Object` layer in data-conventions).
- **Struct fields** are PascalCase with **snake_case JSON tags**
  (`CreatedAt time.Time `json:"created_at"``). Shared GORM fields live in
  `models.GormModel` (`ID`, `CreatedAt`, `UpdatedAt`, `DeletedAt`).

**JavaScript:** new functions and variables are camelCase. A lot of existing frontend
code is `snake_case` (`get_season`, `place_week`) or `PascalCase`
(`GetProfileImageForActivity`); that's legacy — see below.

## Go error handling

The standard idiom inside an API handler is: log at info with the wrapped error,
return a JSON error with an appropriate status, abort, and return.

```go
season, err := database.GetSeasonByID(seasonIDParsed)
if err != nil {
    logger.Log.Info("Failed to get season from database. Error: " + err.Error())
    context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get season from database."})
    context.Abort()
    return
}
```

Rules:

- Log messages and `errors.New(...)` strings are **capitalised sentences ending in a
  period**, e.g. `"Failed to parse season ID."`. When wrapping, append
  `" Error: " + err.Error()`.
- The **client-facing** `gin.H{"error": ...}` string should not leak internals — keep
  it a short human sentence (often the same as the log message, minus the raw error).
- Always `context.Abort()` then `return` after writing an error response.
- Status codes: `http.StatusBadRequest` (400) for bad input / failed parse / not found
  from user-supplied IDs; `http.StatusInternalServerError` (500) for unexpected
  failures; `http.StatusUnauthorized` (401) / `StatusForbidden` (403) for auth (handled
  mostly in middleware). *(Current code is occasionally loose here — prefer the mapping
  above in new code.)*
- Helper/non-handler functions return `error` (usually `errors.New("Failed to …")`)
  and let the caller decide the HTTP response. Don't write to the `context` from deep
  helpers.

## API response shape

Success responses are a `gin.H` with the **resource under a named key** plus a
`message`:

```go
context.JSON(http.StatusOK, gin.H{"seasons": seasonObjects, "message": "Seasons retrieved."})
```

- The resource key is named for the payload and **pluralised for lists**
  (`seasons`, `leaderboard`, `activities`); singular for one item (`exercise`,
  `image`).
- Error responses are always `gin.H{"error": "<sentence>"}` (see above).
- The frontend relies on this: every response is checked for `result.error` first,
  then reads the named key.

## Frontend (vanilla JS) conventions

There is no build step or framework — `web/js/*.js` is served through Go templates
(so `{{ .appVersion }}`-style variables work). See `CLAUDE.md` → "Frontend serving".

- **API calls** use `XMLHttpRequest` against `api_url` (`window.location.origin +
  "/api/"`, defined in `functions.js`), with `Authorization: jwt` and
  `withCredentials = true`. The first thing the `readyState == 4` handler does is
  `JSON.parse` inside try/catch, then check `result.error`.
- **User feedback** goes through the shared helpers in `functions.js`: `error(msg)`,
  `info(msg)`, `success(msg)`, `clearResponse()` — don't roll your own alert markup.
- **Auth/token plumbing** (`get_login`, `refresh_access_token`, `store_tokens`,
  cookies) lives in `functions.js`; reuse it rather than re-implementing token refresh.
- **Avoid `innerHTML +=` inside loops** — accumulate a string and assign once (repeated
  `+=` reparses the DOM each iteration).
- **Images load via `<img src>`, not XHR.** Profile/achievement images are served as raw
  bytes from cookie-authenticated endpoints, so embed them directly:
  `<img src="${profileImageURL(userID, true)}" onerror="${IMAGE_FALLBACK_ONERROR}">`
  (helpers in `functions.js`). The browser caches and dedupes them for free — don't
  re-introduce XHR→base64→`set .src` plumbing. See [image-serving.md](image-serving.md).
- **Modals use the shared `TRModal`** (`web/js/modal.js` + `web/css/modal.css`, the dark
  "telemetry panel"). Don't hand-render `#myModal` markup. Open with
  `TRModal.open({ eyebrow, title, body, onClose })`, swap content with `TRModal.setBody(html)`,
  dismiss with `TRModal.close()`. The legacy `toggleModal(html?)` / `closeModal()` globals are
  shims over it. Body content can use the shared `.trm-field` / `.trm-label` / `.trm-input` /
  `.trm-select` / `.trm-textarea` / `.trm-btn` / `.trm-divider` classes; un-classed elements
  (`label`, `input`, `button`, `hr`, headings) are themed for the dark panel automatically.
  Include `modal.css` + `modal.js` on the page (`modal.js` **after** `functions.js`).
  Don't *also* tag a modal control with a legacy app class (e.g. an old in-page form
  class): `modal.css` themes controls via `#trm-root .trm-body` specificity, but any
  `!important` rule in `main.css` beats that regardless of specificity — which once
  forced an eggshell background under the dark panel's light text (white-on-white inputs
  in the "Add exercise" modal). Use the `.trm-*` classes only.

## Tests

A suite is grown incrementally; run it with `go test ./...`. Coverage is concentrated
in the `database/` layer, the `models/` value types, `auth/` (scopes + token
handling), the outbound integration clients, and pure-logic helpers in
`controllers/`. The `APIXxx` HTTP handlers themselves are still largely untested.

- Test files are `*_test.go` next to the code, `package controllers` / `package
  database` etc. (white-box).
- Prefer **table-driven** tests (`cases := map[...]...` or `[]struct{...}`) with
  `t.Run(name, …)` subtests. See `controllers/hevy_test.go`,
  `controllers/strava_test.go`.
- **Pure functions** (image encode/resize, tag derivation, week/date math) are the
  easiest wins — test them directly with no DB. See `controllers/image_test.go`.
- **`database/` tests** use the in-memory SQLite harness `newTestDB(t)` (in
  `database/setup_test.go`): isolated `:memory:` DB, real `Migrate()`, discarding
  logger, cleanup on `t.Cleanup`. Reuse its helpers (`makeTestUser`, `boolPtr`,
  `strPtr`) instead of reinventing setup.
- **`controllers/` tests** that touch any logging need `logger.Log` set, or they
  nil-panic. The package's `TestMain` (in `controllers/setup_test.go`) stubs it with a
  discarding logger once for the whole package. That file also has
  `newControllerTestDB(t)` — the same in-memory harness pointed at
  `database.Instance`, for controller logic that reads or writes — plus
  `createTestUser` / `seedExerciseDayWithExercises`.
- **`auth/` tests** must install a **valid base64** signing key (see `withSigningKey`
  in `auth/auth_test.go`). `files.GetPrivateKey` regenerates *and persists* a key when
  it fails to decode, so a junk key makes a test write to the real config file.
- **`middlewares/` tests** use `gin.CreateTestContext` + `httptest.NewRecorder` (and
  `gin.SetMode(gin.TestMode)` in `TestMain`) to assert status codes, `WWW-Authenticate`
  challenges and abort behaviour without standing up the app.
- **Integration clients** (Strava, Hevy, Spotify) keep their endpoints in package-level
  **`var`s, not consts, purely so tests can point them at an `httptest` server** —
  nothing in the app reassigns them. Each provider has a stub helper that swaps the
  URLs (and any app credentials), captures the requests, and restores everything on
  `t.Cleanup`: `stubStrava`, `stubHevyAPI`, `stubSpotify`. Plex and Audiobookshelf
  need no such swap — their server URL is already a function argument. Test both what
  the client *sends* (auth header shape, paths, query params) and how it maps
  responses, especially the status codes that carry meaning: Strava's 400/401 →
  `ErrStravaSessionInvalid` (clears the connection) vs a transient 429/5xx, Hevy's
  401/403 → "key rejected", and Spotify's 403 → `ErrSpotifyForbidden`.
- **Password tests are slow**: bcrypt runs at cost 14, so each hash *and each
  comparison* costs about a second. Keep the case list short and put anything
  extravagant behind `testing.Short()`.
- Add tests when finishing or refactoring a feature; for risky refactors, write a
  **characterization test** that pins current behaviour first. When fixing a bug,
  first confirm the new test **fails** against the unfixed code — a persistence bug
  like the dropped `false` in
  [data-conventions.md](data-conventions.md#gorm-drops-a-false-on-insert-when-the-field-has-default-true)
  passes a naive test that never reloads the row.

## Migrations

- **Schema** changes are applied at startup by GORM `AutoMigrate` — there are **no
  migration files**. Register every new model in `database.Migrate()`
  (`database/client.go`); that's the canonical list.
- **Data** migrations (anything beyond schema: backfills, format changes) go in
  `utilities/migrate.go`.
- Three backends must keep working (`sqlite`, `mysql`, `postgres`); data-access SQL
  uses MySQL-style backtick-quoted identifiers, which SQLite accepts (so tests run the
  same queries).

## Legacy & refactoring

Older code predates some of these conventions — most visibly `snake_case`/`PascalCase`
JS function names and a few loose HTTP status choices. Don't do sweeping renames for
their own sake. **When you meaningfully touch a function, bring it up to standard**
(camelCase, the error/response idioms above), and fix call sites in the same change.
Leave untouched legacy alone unless it's in scope.
