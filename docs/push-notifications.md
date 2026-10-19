# Push notifications

Web push per device: a user subscribes a device from `/account` (or the front-page prompt),
and the server pushes through every subscription that opted into the category
(`controllers/notification.go`). The service worker (`web/js/service-worker.js`) shows each
push and opens its page on click.

| Category | Sent for | Opens | `additional_data` |
|---|---|---|---|
| `achievement` | a new achievement | `/achievements` | — |
| `news` | a news post | `/news` | — |
| `alert` | Sunday reminders, a lost week | `/` | — |
| `debt` | a wheel to spin, a spin to check, a win | `/wheel?debt_id=…` | the debt id |
| `account` | a broken integration ([integration-health.md](integration-health.md)) | `/account` | — |

All pushes share the notification tag `Message`, so a new one replaces the one before it on
the device rather than stacking.

## Clearing what you've already seen

The service worker stores each notification's `category` and `additionalData` in its
`data`. Pages close the ones they answer with `closeNotifications(category, additionalData?)`
(`web/js/functions.js`, through `ServiceWorkerRegistration.getNotifications`, which works in
the installed PWA too):

- `/achievements` closes `achievement` notifications;
- `/news` closes `news`;
- `/wheel?debt_id=…` closes the `debt` notification **for that debt** only.

`account` notifications are deliberately left until the connection is fixed, and `alert`
ones are reminders with no page to "see". A notification shown before the service worker
started storing its category can't be matched and stays until dismissed.
