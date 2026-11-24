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

Each notification is tagged by kind (`notificationTag` in the service worker): `debt-<debt
id>` for a debt, otherwise the category. A new push replaces an older one of the same kind
(a second spin check for the same debt, a lost week over that week's Sunday reminder) but
leaves other kinds alone, so Monday's wheel spin no longer wipes out an unread Sunday
reminder.

## Delivery: urgency, TTL and logging

`PushNotificationToSubscriptions` sends every push with **`Urgency: high`**, since each one
is shown to the user. With the default normal urgency, a sleeping Android phone (Doze) has
the push service hold the message until its next maintenance window.

The **TTL** is how long the push service keeps a message it can't deliver yet; once it
expires, the message is dropped without a trace. It is set per notification:
`pushTTLSundayReminder` (6 hours, until Sunday ends) for the Sunday reminder and
`pushTTLDefault` (24 hours) for everything else. It used to be 30 seconds for all, which
lost reminders to phones that were asleep at 18:00.

Every push logs its outcome at info level: category, subscription and user ID, the push
service's host (never the full endpoint, which works as a credential) and the HTTP status.
A 404/410 disables the subscription; any other non-2xx is logged as not delivered and not
counted. A device that never gets anything while the log shows a 201 for it was accepted by
the push service, so look at the device (muted, battery optimisation, notification
permission). A network error such as `dial tcp 0.0.0.0:443` means the server's own DNS
blocks that push service's host.

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
