# Activity heatmap

A private, per-user density heatmap of where the user's GPS activities have taken them,
shown inside the **Activity statistics** section of the `/statistics` page. Because it
lives in that section, it inherits the existing **activity-type** and **date-range**
filters for free — the heatmap reflects whatever activity and period the user selects.

## Why this shape

The idea is borrowed from the [running-heatmap](https://github.com/moresamwilson/running-heatmap)
project (a Python/Jupyter tool that builds heatmaps from a downloaded Strava export). We
did **not** reuse its code: it is a different language/runtime and is built around a
one-off offline export. Treningheten already stores per-activity GPS `latlng` streams
from the hourly Strava sync (`OperationSet.StravaStreams`, see [strava.md](strava.md)),
so the entire data-ingestion half of that project is unnecessary — only the
visualization is reimplemented, deliberately kept simple.

## Data flow (no dedicated endpoint)

There is **no separate heatmap API**. The Activity-statistics endpoint
(`GET /api/auth/actions/:action_id/statistics?start=&end=`, `APIGetActionStatistics`)
already returns the matching `OperationObject`s, and `ConvertOperationSetToOperationSetObject`
copies `StravaStreams` onto each set — so the `latlng` data is already on the wire,
filtered to the chosen activity and period. Adding a second request would just re-fetch
the same data, so the client renders straight from the statistics response.

Client side (`web/js/statistics.js`):

- `renderActivityHeatmap(operations)` is called at the end of `placeActivityStatistics`.
- `extractHeatmapPoints` flattens `operations[].operation_sets[].strava_streams.latlng.data`
  into `[lat, lng]` points, thinned by a per-track stride and capped overall to keep the
  render light.
- `densestCenter` buckets points into ~1 km cells and returns the centroid of the busiest
  cell, so the map **opens on the most-frequented cluster** rather than fitting all points
  (which would zoom out to include far-away one-off activities).
- Rendering uses **Leaflet** + **Leaflet.heat** (loaded from cdnjs in `statistics.html`,
  matching the existing Chart.js CDN usage) over an OpenStreetMap tile basemap. The map
  container is `.heatmap-canvas`; it starts hidden, so `invalidateSize()` is called once
  it is shown. `resetActivityHeatmap` clears the layer whenever the filter changes.

Only activities with GPS movement carry `latlng` streams; for any other activity type (or
a period with no GPS data) the section shows a "no GPS data" note. A caption states that
only activities with GPS data appear on the map.

## Privacy

GPS tracks reveal where a user lives, so this is strictly private. `APIGetActionStatistics`
is scoped to the **token's user** (it reads only that user's exercise days), and the
heatmap is only rendered on the owner's own `/statistics` view — it is **not** part of
public profiles. If this is ever extended to public/shared heatmaps, start/end points
(near home) should be clipped or fuzzed first, and the external OSM basemap tiles
reconsidered for a privacy-minded self-hosted deployment.

## Possible future work

- Multi-layer views (pace / heart-rate / gradient), which would need server-side grid
  aggregation and raster/image generation rather than the current client-side density layer.
- Self-hosting the Leaflet assets and/or tiles instead of using CDNs.
- A lighter dedicated endpoint if the full statistics payload (which already ships entire
  streams) becomes a concern.
