# Wheel appearance customization

Users can personalize how they appear on the prize wheel (the wheel-of-fortune that
awards a season's prize). Each person can set a **color**, a **border color**, and an
**emoji** from their account page; anything left unset is auto-assigned.

See [seasons-and-goals.md](seasons-and-goals.md) for the wheel's role in the
competition.

## Stored fields

Three nullable fields on the `User` model (`models/user.go`), all public (not stripped
by the user censor, so they're visible to others on the wheel):

| Field | JSON | Meaning |
|---|---|---|
| `WheelColor` | `wheel_color` | Segment fill color (hex `#rrggbb`) |
| `WheelBorderColor` | `wheel_border_color` | Segment border color (hex) |
| `WheelEmoji` | `wheel_emoji` | Emoji prefixed to the name on the segment |

`null`/empty means "auto". Columns are added by `AutoMigrate` (no migration file).

## Editing (account page)

The **"Wheel appearance"** section (`web/js/account.js`, `renderWheelSection`) offers:

- a **curated swatch grid** (the wheel's 20 distinct palette colors) for color and
  border — one click to select,
- a **custom hex** picker (`<input type="color">`) for full freedom,
- an **emoji** text field, limited to a single emoji — the input keeps only the first
  grapheme cluster (`firstGrapheme` via `Intl.Segmenter`), so flags/skin-tones/ZWJ
  sequences count as one,
- a **live preview** chip showing the name on the chosen color, border and emoji.

Changes are saved immediately via `PATCH /api/auth/users/:id`
(`UserPartialUpdateRequest` → `APIPartialUpdateUser`). An empty value clears the field
(reverts to auto). Styling is in `web/css/main.css` (`.wheel-*`).

### Validation (server-side)

`APIPartialUpdateUser` validates before storing (`utilities/validation.go`):

- colors must match `^#[0-9a-fA-F]{6}$` (`ValidHexColor`),
- the emoji must be short and contain a non-ASCII symbol rune (`ValidWheelEmoji`),
  which blocks plain text and overly long strings while allowing single emoji, flags
  and ZWJ sequences.

## Rendering (the wheel)

`web/js/wheel.js` `placeWheel()` builds the segments. The candidate list comes from
`GET /api/auth/debts/:id` → `winners[].user`, a full `User` object, so the wheel fields
arrive with no backend change.

### Color assignment — `assignWheelColors`

Designed so the wheel is always **distinct and stable**, while honoring choices:

1. **Honor explicit picks** verbatim. Two users *may* pick the same color — that is
   accepted (the picker is a preference, not a claim).
2. **Auto-assign everyone else** deterministically: candidates without a pick are
   ordered by user id (stable across spins — unlike the old per-spin random colors)
   and given the first **unused** palette color, so auto colors never collide with each
   other or with explicit picks.
3. If the 20-color palette is exhausted, fall back to a **golden-angle HSL** spread for
   continued distinctness.

### Border & emoji

- A set `wheel_border_color` becomes the segment's `strokeStyle` (with a `lineWidth`);
  unset means no custom border.
- A set `wheel_emoji` is prefixed to the segment `text` (e.g. `🔥 Alex`).

### Readability

Segment labels stay legible on any fill: the text color is chosen from the fill's
perceived brightness (`readableTextColor` — black on light, white on dark). This
applies to picked and auto colors alike, including custom hex. The text outline is
deliberately omitted — Winwheel strokes it on top of the fill with miter joins, which
looks rough on rotated glyphs, so fill-only (bold) text reads cleaner.

### Long-name fit

Winwheel draws `outer`/horizontal segment text from the rim inward with no fitting of
its own, so a long first name (or a wide emoji) used to spill past the wide part of the
wedge toward the pointy center. `fitSegmentFontSize` (`web/js/wheel.js`) measures each
label with the canvas `measureText` and, if it exceeds the segment's radial budget,
shrinks **that segment's** `textFontSize` (down to a 16px floor) — short names keep the
full 34px. Labels are never truncated; a long name only gets a smaller font. The
measurement is done in the wheel's base (logical) units, which is correct because
Winwheel multiplies both geometry and font by `scaleFactor` internally, so the fitted
size is scale-independent.

### Crispness (high-DPI / mobile)

`placeWheel` sizes the canvas backing store to `1000 × devicePixelRatio` (capped at
3×) and passes that ratio as Winwheel's `scaleFactor`, while keeping the displayed
(CSS) size at the logical 1000px (responsive via `max-width:100%`). This avoids the
browser upscaling a fixed low-res bitmap onto high-DPI screens, which was causing
aliased borders and text. Because Winwheel scales geometry/fonts but not stroke
widths, the segment border/text-outline widths and the manual pointer triangle are
multiplied by the scale factor too.

## Key code locations

- `models/user.go` — the three `Wheel*` fields + `UserPartialUpdateRequest`.
- `controllers/user.go` — `APIPartialUpdateUser` validation.
- `utilities/validation.go` — `ValidHexColor`, `ValidWheelEmoji`.
- `web/js/wheel.js` — `assignWheelColors`, `readableTextColor`, segment building.
- `web/js/account.js` — `renderWheelSection` and the save/preview helpers.
- `web/css/main.css` — `.wheel-*` styles.

## Related

- [seasons-and-goals.md](seasons-and-goals.md) — the wheel and prizes
