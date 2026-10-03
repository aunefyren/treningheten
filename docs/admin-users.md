# Disabling users

The admin panel (`/admin`) has a **Users** module listing every account, disabled ones
included, with a Disable / Enable button per row (not on your own row).

## What disabling does

- **Access stops at once.** `middlewares.Auth` / `Authenticate` check `users.enabled` on
  every request, so existing sessions, refresh tokens, PATs, OAuth clients and MCP all
  get `403 account disabled` without any token revocation.
- **The user leaves every unfinished season.** `DisableGoalsForUserInUnfinishedSeasons`
  disables their goals in seasons whose `End` is still ahead (ongoing *and* upcoming), so
  they drop out of weekly processing, debts and the wheel. It runs on every disable
  request, not just a state change, so a retry after a partial failure finishes the job.
- **History stays.** Goals in finished seasons are untouched, and the read paths that
  render history resolve users with `GetUserInformationIncludingDisabled` (goal, debt,
  wheel view, invite and exercise-day conversion; `GetUsersByIDs` for the season goal
  list). Before this, a single disabled user with a goal made `ConvertGoalsToGoalObjects`
  fail and broke the season views for everyone.
- **Re-enabling restores access only.** The user rejoins seasons themselves.

An admin cannot disable their own account (400). Everything else that looks users up
by `GetUserInformation` (profiles, the user picker, OAuth/PAT ownership) still treats a
disabled user as absent.

## Code

- `GET /api/admin/users` → `APIAdminGetUsers`: every user as `models.AdminUser` (an
  allowlist: name, email, admin/enabled/verified flags; no credentials).
- `PUT /api/admin/users/:user_id/enabled` `{ "enabled": bool }` → `APIAdminSetUserEnabled`
  (`controllers/admin_user.go`). Answers `{ "user": AdminUser, "message": … }`.
- Database: `GetAllUsersIncludingDisabled`, `GetUserByIDIncludingDisabled`,
  `SetUserEnabled`, `ToAdminUser` (`database/user.go`),
  `DisableGoalsForUserInUnfinishedSeasons` (`database/goal.go`).
- Frontend: `getAdminUsers` / `placeAdminUsers` / `setUserEnabled` in `web/js/admin.js`.
  The same list fills every user picker on the page (`placeUserSelects`, enabled users
  only).
- Tests: `controllers/admin_user_api_test.go`, `database/user_admin_test.go`.
