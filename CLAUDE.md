# CLAUDE.md

Guidance for AI agents working in this repository. Keep this file short: it is loaded into
every session. Detail belongs in `docs/` — this file points there.

## What this is

Treningheten is a self-hosted workout tracker built around competitive "seasons": users join a
season with a weekly workout goal, log workouts, and build streaks on a leaderboard, with a
wheel of fortune, achievements, sick leave and integrations (Strava, Hevy, Plex, Spotify,
Audiobookshelf, Ollama, MCP). It is a single Go binary — Gin + GORM — serving a JSON API and a
templated vanilla-JS frontend from `web/`. Architecture: [`docs/architecture.md`](docs/architecture.md).

## How we work

Work is coordinated through Paseo. **Planning agents** (Opus) discuss, decide and plan
together with the maintainer; **coding agents** (Sonnet) implement an agreed plan.

- **GitHub issues are the backlog.** Plans, ideas, bugs and open decisions live in issues
  (`gh issue list`, `gh issue view <n>`), not in Markdown files. Don't create WIP/TODO/plan
  files in the repo.
- **Planning (Opus):** talk features, architecture and design through with the maintainer
  *before* code is written — propose, then confirm direction. Record the outcome in the issue in
  the shape of the "Planned work" template (`.github/ISSUE_TEMPLATE/planned-work.md`): goal,
  decided approach, files involved, docs to update, tests, open questions. Labels: `idea`
  (not planned yet), `needs-decision` (open questions block the work), `security`, `bug`,
  `enhancement`.
- **Implementing (Sonnet):** work from an issue whose plan is decided. If the plan turns out
  to be wrong or incomplete, stop and say so in the issue/PR instead of improvising a
  different design.
- **Worktrees, always.** Every agent works in its own git worktree on its own branch — never
  in the main checkout and never on `main`. Several agents share this machine and its paths:
  don't touch files outside your worktree, don't run `git stash` (the stash is shared), and
  don't assume ports or containers are free.
- **Branches and PRs.** Branch `feat/<issue>-<slug>` or `fix/<issue>-<slug>` (`docs/`,
  `chore/` likewise). PR titles keep the commit style (`feature: …`, `fix: …`, `docs: …`) and
  the body says `Closes #<issue>`. CI (`.github/workflows/go.yml`) must pass. The maintainer
  merges — agents never merge or push to `main`.
- **GitHub Actions changes go in `.github/potential-workflows/`.** The agent token can't push
  to `.github/workflows/`, so never touch that directory. Stage complete workflow files under
  their real names, and list the `git mv`/`git rm` commands at the top of the PR body for the
  maintainer. Procedure and workflow overview: [`docs/ci.md`](docs/ci.md).
- **Nothing lasts outside the repo.** Agent memory is per worktree path and is lost with it.
  A fact worth keeping goes into the relevant `docs/*.md`, or into the issue.

## Rules

- **Keep docs in sync.** When a feature is added or changed, update its `docs/*.md` (create
  one if needed, and list it in [`docs/README.md`](docs/README.md)), plus this file or
  `README.md` if they are affected. A PR that changes behaviour without touching docs is
  incomplete.
- **Follow [`docs/conventions.md`](docs/conventions.md)** — naming, error handling, API
  shapes, frontend JS, tests, migrations. Key rule: camelCase wherever the language allows,
  with initialisms uppercase (`userID`, struct field `ID`). Older code is refactored when it
  is meaningfully touched, never in sweeping renames.
- **Read the feature doc first.** Before changing a feature, read its doc (index:
  [`docs/README.md`](docs/README.md)). For the data layer, read
  [`docs/data-model.md`](docs/data-model.md) and
  [`docs/data-conventions.md`](docs/data-conventions.md); for UI,
  [`docs/styleguide.md`](docs/styleguide.md).
- **Add tests when finishing a feature**, reusing the existing harnesses (below).
- **Don't screenshot-verify frontend changes.** No throwaway HTML harnesses or headless
  browsers: they drift from the real JS-built markup, and headless Edge here ignores the
  viewport size, so `@media` breakpoints never trigger. Reason about the CSS and let the
  maintainer look at the real page.

## Gotchas

- **Durations are seconds.** `*time.Duration` fields (`Operation.Duration`,
  `OperationSet.Time`/`MovingTime`) hold a raw seconds count — `int64(*d)`, never
  `d.Seconds()`.
- **New models go in `Migrate()`** (`database/client.go`). There are no migration files;
  queries must run on SQLite, MySQL and Postgres ([conventions](docs/conventions.md#portable-sql)).
- **New routes go in `initRouter`** (`main.go`) *and* in the test harness route table
  (`controllers/api_harness_test.go`).
- **Background work uses `goSafely`**, never a bare `go`.
- **Reads go through `Convert*Object`** — flatten the `Object` tree rather than re-walking
  raw models.
- **Password hashing/comparison must take `middlewares.AcquirePasswordCostSlot()`.**
- **Integration URL `var`s** (`stravaAPIBaseURL`, `hevyAPIBaseURL`, …) exist only so tests
  can redirect them — never reassign them at runtime.
- **The frontend is Go templates**, not a build: `web/js/*.js` can use `{{ .appVersion }}`
  etc. See [architecture](docs/architecture.md#frontend-serving).
- **The version is the `{{RELEASE_TAG}}` placeholder** in `files/config.go`, replaced at
  release time.

## Commands

```bash
go build .                # ./treningheten
go run . --generateinvite true   # run locally; prints an invite code for the first user
go vet ./...
gofmt -w <files>
go test -short ./...      # fast suite: skips the slow fault-injection sweeps
go test ./controllers/    # full suite for one package (the sweeps take minutes)
go generate ./database    # after adding an exported database function
```

Run `go vet ./...` and the short suite before opening a PR, plus the full suite for the
packages you touched. CI also runs `-race` and the full suite on Postgres and MySQL.

Test harnesses (details in [conventions.md → Tests](docs/conventions.md#tests)):
`newTestDB(t)` for `database/`, `newControllerTestDB(t)` and `newAPIHarness(t)` for
`controllers/`, `internal/smtptest` for mail, fake `httptest` backends for integrations.

The live Docker harness (`docker-test/`, see its [README](docker-test/README.md)) runs the app
with Mailpit, Audiobookshelf and optional MariaDB/Postgres/Ollama.
