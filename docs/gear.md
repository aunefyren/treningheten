# Gear tracking

Treningheten lets a user track **gear** — shoes, a bike, or other equipment — and
which workouts used it. Gear can be created manually or imported automatically from
Strava, and each gear item shows a total distance computed from the workouts it is
linked to.

Code: `models/gear.go`, `database/gear.go`, `controllers/gear.go`; Strava import lives
in `controllers/strava.go`.

## Data model

A `Gear` row (`models/gear.go`, table `gear`) is per-user equipment:

| Field | Purpose |
|---|---|
| `Name`, `Type`, `Brand`, `Model`, `Nickname` | Identity. `Type` is one of `shoe` / `bike` / `other`. |
| `Retired` | Hidden-by-default flag for gear no longer in use (still kept for history). |
| `IsPrimary` | The user's default gear; the builder pre-selects it. Only **one** gear per user is primary — promoting one demotes the rest (`UnsetPrimaryGearForUser`). |
| `StravaGearID` | The Strava equipment id (e.g. `g12345` shoe / `b12345` bike) when imported from Strava; **null** for manually created gear. |
| `Enabled` | App-level soft delete (deleting gear sets this false). |

Gear links to workouts through **`Operation.GearID`** (`*uuid.UUID`). One gear per
operation, mirroring Strava's per-activity `gear_id`.

> **Distance is not stored.** A gear's total distance is always computed by summing the
> `OperationSet.Distance` (km) of the operations linked to it
> (`GetGearDistanceTotalsForUser`) — one source of truth that works for manual gear too.
> It deliberately does **not** mirror Strava's lifetime total, so pre-Treningheten
> mileage is not included.

## Read shape

`GearObject` (`ConvertGearToGearObject`) is the enriched read form: the gear identity
plus the computed `Distance`. `BuildGearObjectsForUser` assembles a user's full gear
list in two queries (the gear rows + one grouped distance roll-up). Operations also
flatten their gear into `OperationObject.Gear` — but **identity only** (`Distance` left
at zero), to avoid a roll-up query per operation in list views.

## Selection — per operation, with a session-level convenience

Gear is **stored on the operation** (so a combined session that mixes activities can
hold different gear per activity, matching Strava). The builder edits it **per
operation**: each **moving** activity card carries its own gear selector, which writes
through the normal operation update — `PUT /api/auth/operations/:operation_id` with
`{ "gear_id": "<uuid>" | "" }` (empty clears it, `APIUpdateOperation` →
`resolveGearIDForUser`). Only moving activities show the selector, mirroring Strava's
shoe/bike model; a moving activity with no gear yet suggests the user's primary
(unpersisted until changed).

For a combined session that mixes **2+ moving activities**, a session-level "Set gear
for all" control assigns one gear to every operation at once:
`PUT /api/auth/exercises/:exercise_id/gear` with `{ "gear_id": "<uuid>" | null }`
(`APISetGearForExercise`). It is a convenience over the per-operation path, not the only
way in.

## Endpoints

All under `/api/auth` (`middlewares.Auth(false)`):

| Method | Path | Handler | Use |
|---|---|---|---|
| GET | `/gear` | `APIGetGearForUser` | List the user's gear with computed distance. |
| POST | `/gear` | `APICreateGear` | Create manual gear (name required; type defaults to `shoe`). |
| PUT | `/gear/:gear_id` | `APIUpdateGear` | Update gear. Partial — omitted fields are untouched. |
| DELETE | `/gear/:gear_id` | `APIDeleteGear` | Soft-delete gear (`Enabled = false`). |
| PUT | `/exercises/:exercise_id/gear` | `APISetGearForExercise` | Assign/clear gear on all of a session's operations. |

**Strava-sourced gear is identity-read-only:** `APIUpdateGear` rejects edits to
`Name`/`Type`/`Brand`/`Model` when `StravaGearID` is set (Strava owns those), but still
allows `Nickname`, `Retired`, and `IsPrimary`.

## Strava import

During a sync, `StravaSyncOperationForActivity` maps the activity's `gear_id` onto a
local gear row via `resolveStravaGearForUser`:

1. If the activity has **no** `gear_id`, the operation's existing gear is **left
   unchanged** (so a user-set gear on a non-gear Strava activity is preserved).
2. Otherwise it finds the user's gear by `StravaGearID`, or **creates** it the first
   time — fetching the equipment detail once via `StravaGetGear` (`GET /gear/{id}`) to
   resolve name/brand/model. The id prefix sets the type (`b…` → bike, `g…` → shoe). A
   failed detail fetch is non-fatal: the gear is imported with the id as its name.
3. `Operation.GearID` is set to the resolved gear.

Strava is the source of truth for the gear *on its own activities*: a re-sync re-links
the operation to whatever gear Strava reports. Imported gear is **kept until deleted in
Treningheten** — we never auto-remove a local gear row because Strava retired or deleted
it (a Strava `retired` flag is mirrored onto the row as information only).

## Related

- [strava.md](strava.md) — the sync pipeline that imports gear.
- [data-conventions.md](data-conventions.md) — the `Convert*Object` read layer and the
  durations/units conventions.
