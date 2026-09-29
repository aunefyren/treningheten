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

### Background work goes through `goSafely`, never a bare `go`

Handlers fire best-effort work into the background — achievement grants, Strava/Hevy syncs,
Ollama cache refreshes, media pulls — and deliberately ignore the outcome. A panic on such a
goroutine cannot be recovered by the handler that spawned it, so **one nil dereference in a
throwaway achievement grant takes down the whole server**, dropping every in-flight request. Use
`goSafely("what this is", func() { … })` (`controllers/background.go`), which recovers and logs
the panic with its stack:

```go
goSafely("achievement grant", func() {
    GiveUserAnAchievement(userID, uuid.MustParse("…"), time.Now(), 5)
})
```

This is not licence to swallow errors: `fn` still logs its own, and anything whose failure the
user must hear about does not belong on a background goroutine at all. CI found this the hard
way — a controller test finished, its `t.Cleanup` reset `database.Instance` to nil, and a
still-running achievement goroutine dereferenced it and killed the test binary. The same crash
was reachable in production from any panic in any of those tasks.

### A "not found" getter must return nil, and the caller must check it

Data-access getters that return a **pointer** (`GetExerciseByIDAndUserID`,
`GetExerciseDayByID`, `GetExerciseDayByIDAndUserID`) report "no such row" as `(nil, nil)` —
**not** an error, and **not** the zero-value struct. GORM's `Find` allocates the struct whether
or not a row matched, so `return exercise, nil` on a miss hands back a non-nil pointer with a
zero `ID`, and every `if x == nil` check downstream becomes dead code that never fires. Write
the miss branch explicitly:

```go
if record.Error != nil {
    return nil, record.Error
} else if record.RowsAffected != 1 {
    return nil, nil          // not `return exercise, nil`
}
```

And on the calling side, **both** branches are required — an `err != nil` check alone passes a
miss straight through:

```go
exercise, err := database.GetExerciseByIDAndUserID(exerciseID, userID)
if err != nil {
    // 500
} else if exercise == nil {
    // 404 — wrong id, or not this user's
}
```

This matters beyond tidiness: these getters are **user-scoped**, so the nil check *is* the
ownership check. `APICreateOperationForUser` tested only the error and discarded the result,
which let any authenticated user attach an activity to another user's session. Pinned by
`database.TestExerciseGettersReturnNilOnMiss` and
`controllers/operation_authz_test.go`.

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

### Never serialize a credential

`models.User` carries a bcrypt hash, a live password-reset code, a verification code and
the Strava/Hevy credentials. All of them are `json:"-"`, so no handler can leak them by
forgetting to censor. When a page needs to know that a connection *exists*, add a derived
`gorm:"-"` boolean and set it in the handler — `HevyConnected` / `StravaConnected` in
`GetUser` are the pattern:

```go
userObject.StravaConnected = userObject.StravaCode != nil && *userObject.StravaCode != ""
```

The data-access layer reinforces this with two families of getter, and the names are the
contract:

- **Censored (default):** `GetUserInformation`, `GetUsersInformation`, `GetUsersByIDs`,
  `GetAllUsersWithSundayAlertsEnabled` — run every row through `CensorUserObject` and so
  return `models.PublicUser`. Use these for anything that reaches a response body.
- **Uncensored (explicit `All`/`Uncensored` in the name):** `GetAllUserInformation`,
  `GetAllUsersUncensored`, `GetAllUsersWithSundayAlertsEnabledUncensored` — for server-side
  jobs that genuinely need the values (the Strava sync needs `StravaCode`; the Sunday
  reminder needs the e-mail address). Never let their result reach `context.JSON`.

(The index-assign rule that used to live here is obsolete: `CensorUserObject` returns a new
value of a different type, so a censored copy cannot be silently discarded any more.)

### What one user may see of another is an allowlist type

`json:"-"` handles fields that are secret from *everyone*. It cannot express "visible to its
owner, private from their peers" — a birth date and a resting heart rate are edited by their
owner on `/account` and are nobody else's business. For those, the boundary is the **type**:

- **`models.User`** is the full row. It is what you get back from the uncensored getters and
  what a user sees of themselves.
- **`models.PublicUser`** is what one user may see of another. `CensorUserObject` returns it,
  and so do all the censored getters.

`PublicUser` is an allowlist, and that is the entire point. Its predecessor redacted named
fields and returned a `User`, which is a blocklist — so `BirthDate`, `MaxHeartrate`,
`RestingHeartrate` and `ObservedMaxHeartrate`, all added long after the censor was written,
were served to every authenticated caller (a read-only PAT included) simply by existing.
**To expose something new about a user to their peers, add it to `models.PublicUser`
deliberately.** Adding a field to `models.User` exposes it to nobody.

Do not call the reduced type `UserObject`: in this codebase the `*Object` suffix means an
*enriched* read model (`ConvertExerciseToExerciseObject` resolves relations and does real
work). This is a reduction and should not borrow that word.

### Raw GORM rows are not response shapes

A GORM row's `User` association is `json:"-"` on every model that has one (`Goal`,
`ExerciseDay`, `Debt`, `Gear`, `MediaConnection`, `Notification`, `Wheelview`,
`AchievementDelegation`, `Invite`). The read DTO beside it — `GoalObject`,
`ExerciseDayObject`, `DebtObject`, `Activity`, … — carries `models.PublicUser` instead.

This is not theoretical tidiness. `models.Week` embeds `Goals []Goal` and `APIGetWeek`
returns it straight to the client; that shipped without leaking **only** because nothing
calls `Preload("User")`, so the association marshalled as an empty object. One preload added
to fix an unrelated N+1 would have put every field of `models.User` in the week response.

So: when a raw row must be reachable from a response, mark its associations `json:"-"` and
put the public data on the DTO. Regression tests live in `database/user_censor_test.go`
(`assertOnlyPublicFields` pins the exact key set that may go out) and
`models/user_serialization_test.go`.

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
- **Escape every value you interpolate into HTML.** Markup is built as template strings
  and assigned with `innerHTML`, so any value that originated outside the page — a note, a
  gear or action name, a track title, a Strava description — must go through the local
  `escapeHTML()` helper. That includes values inside a `<textarea>`: a `</textarea>` in the
  content closes the element and everything after it is parsed as markup. Attribute values
  need it too (the helper escapes `"`, so keep attributes double-quoted). This is not
  theoretical — an unescaped day note plus a missing ownership check on its write endpoint
  was a stored-XSS path to another user's session tokens, which are readable from
  `document.cookie` by design.
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
`controllers/`, plus request-level flow tests for the main `APIXxx` handlers (seasons
and goals, workout logging, accounts, admin, weekly debts and the wheel, push
notifications, the integrations against fake backends, OAuth, MCP) and the fault
sweeps below. Total statement coverage is about 90%.

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
- **HTTP handler tests** go through `newAPIHarness(t)` (`controllers/api_harness_test.go`):
  a real gin router with the real `Auth` middleware, the in-memory DB, the curated
  actions and OAuth client seeded, and a signing key installed. `h.user(email, admin)`
  creates a verified user and mints its access token; `h.expect(status, method, path,
  token, body)` / `h.ok(…)` send a request and decode the JSON; `field`/`idOf` dig into
  the response. The route table **mirrors `initRouter`** (which lives in `package main`
  and can't be called from here), minus the rate limiters (their state is
  process-global) — add a route there when a test needs it. Write these as **flows**
  (create → read → update → someone else tries → delete) and assert the actual status
  codes: several handlers answer a GET with 201, so don't assume.
- **Mail** is tested against `internal/smtptest`, an in-process SMTP server that records
  messages (`withFakeSMTP` in `utilities`, `withControllerSMTP` in `controllers`) — the
  real `go-mail` dialer runs, and a test can read the verification or reset code out of
  the mail it just triggered. **Web push** is tested against an `httptest` push endpoint
  with real VAPID keys and a real P-256 subscription (`notification_api_test.go`).
- **Integrations** are tested end to end against fake `httptest` backends that speak just
  enough of the real API: `withStrava`/`fakeStrava`, `fakeHevyAPI`, `fakeAudiobookshelf`,
  `stubSpotify`, a fake plex.tv + Plex Media Server pair, `withOllama`, and MCP through
  the official go-sdk client (`mcpSession`/`callTool`). Background work an endpoint
  starts with `goSafely` races with assertions — test the connect endpoint against an
  empty account and drive the sync function synchronously.
- **Robustness sweep** (`controllers/robustness_test.go`): every routed handler is called
  with random ids, then again with the database closed, and must not panic. It found the
  missing `return` in `APIRegisterSickleave`; keep new routes in the harness so they're
  swept too.
- **Fault-injection sweep** (`controllers/fault_test.go`, scenarios in
  `fault_scenarios*_test.go`): `installFaultInjector` registers GORM callbacks that fail
  exactly the Nth database operation. Each scenario runs once cleanly to count its
  operations, then once per operation with that one failing, on a fresh harness. A
  panic fails the test; a 2xx answer despite the failure is *reported* (logged at the
  end), not failed, because many reads deliberately degrade — see docs/wip.md for the
  ones that probably shouldn't. Scenarios are either HTTP requests or direct `call`s
  for background entry points (syncs, cron jobs). The sweep is skipped under `-short`.
- **Upstream fault sweep** (`controllers/upstream_fault_test.go`): every fake provider
  handler calls `injectUpstreamFault` first, which fails exactly the Nth upstream request
  as a 500, a non-JSON 200, or a dropped connection. `TestUpstreamFaultSweep` replays each
  integration scenario once per request and mode; it must never panic. New fakes must
  call the hook. Fakes that talk to rate-limited providers must reset the client-side
  limiter (see `withStrava`), or a long run blocks in `stravaWait`.
- **Closed-database sweep** (`database/closed_db_test.go`): every exported function in
  `database/` is called with zero-value arguments against a closed database and must
  return an error, never panic. The call list is generated —
  `go generate ./database` runs `gen_closed_db_calls.go` (a `//go:build ignore` file).
  Regenerate after adding a function.
- **bcrypt cost** is `models.PasswordHashCost` (14), a var solely so the `controllers`
  tests can drop it to `bcrypt.MinCost` in `TestMain`; the password flows hash hundreds
  of times.
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
- **Password tests are slow** outside `controllers/`: bcrypt runs at cost 14, so each
  hash *and each comparison* costs about a second. `controllers/` lowers
  `models.PasswordHashCost` in `TestMain`; elsewhere keep the case list short and put
  anything extravagant behind `testing.Short()`.
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
