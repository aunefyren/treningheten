# Invites

Registration needs an invite code. Admins generate codes on `/admin` (the **Invites**
module: `POST /api/admin/invites`, listed by `GET /api/admin/invites`, withdrawn with
`DELETE` while unused). There is no invite e-mail; the admin shares the code or link
themselves.

## Invite links

`/register?invite=CODE` fills in the code on the register page (`applyInviteFromURL` in
`web/js/register.js`), changes the intro to "You've been invited…", and focuses the e-mail
field. The code stays editable in case a link was mangled.

- On `/admin`, each **unused** invite has a **Copy link** button (`copyInviteLink`), built
  from the page's own origin. Where the clipboard isn't available (a non-HTTPS origin), the
  link is shown in a prompt to copy by hand.
- `--generateinvite true` logs the new code on startup and, when
  `treningheten_external_url` is configured, the ready-made link too
  (`utilities.InviteLink`).

The bigger onboarding flow (season sign-up, connections and notifications after registering)
is still open in [#20](https://github.com/aunefyren/treningheten/issues/20).
