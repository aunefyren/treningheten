package models

// maxPlausibleHeartrate caps a believable human heart rate; samples above it are treated
// as sensor dropouts so a single glitch can't inflate a stored maximum.
const maxPlausibleHeartrate = 226

// ObservedMaxHeartrate returns the highest plausible heart-rate sample in a stream, or 0
// when there is none. Shared by the Strava sync (to maintain a user's all-time observed
// max) and the one-time backfill, so both filter dropouts identically.
func ObservedMaxHeartrate(s *StravaActivityStreams) int {
	if s == nil || s.Heartrate == nil {
		return 0
	}
	max := 0
	for _, v := range s.Heartrate.Data {
		if v > max && v <= maxPlausibleHeartrate {
			max = v
		}
	}
	return max
}

// StreamSummary is the processed, presentation-ready view of one activity's Strava
// sensor streams. It is the single shared shape consumed by both the MCP
// get_activity_streams tool (embedded in MCPWorkoutStreams) and the /exercises detail
// page (attached to OperationObject) — so the summary math lives in one place and the
// two surfaces can never drift. Only channels the workout actually recorded are set.
type StreamSummary struct {
	Available       []string `json:"available,omitempty" jsonschema:"sensor channels present in this workout"`
	HasGPS          bool     `json:"has_gps"`
	DurationSeconds int64    `json:"duration_seconds,omitempty"`

	Heartrate   *StreamStat          `json:"heartrate_bpm,omitempty" jsonschema:"heart rate, beats per minute"`
	Cadence     *StreamStat          `json:"cadence_rpm,omitempty" jsonschema:"cadence, revolutions/steps per minute"`
	Speed       *StreamSpeedStat     `json:"speed,omitempty"`
	Elevation   *StreamElevationStat `json:"elevation,omitempty"`
	Power       *StreamPowerStat     `json:"power,omitempty"`
	Temperature *StreamStat          `json:"temperature_c,omitempty" jsonschema:"temperature, degrees Celsius"`

	Segments         []StreamSegment        `json:"segments,omitempty" jsonschema:"per-unit-distance splits (per km or per mile per the activity's distance_unit); the final split may be shorter than a full unit"`
	Route            *StreamRoute           `json:"route,omitempty" jsonschema:"a summary of the GPS path (present only when the activity has GPS)"`
	ElevationProfile []StreamElevationPoint `json:"elevation_profile,omitempty" jsonschema:"a down-sampled altitude-over-distance profile of the activity"`
	HRZones          []StreamHRZone         `json:"hr_zones,omitempty" jsonschema:"time spent in each of five heart-rate zones"`
	HRMaxBasis       string                 `json:"hr_max_basis,omitempty" jsonschema:"how the HR zones were anchored: 'max' (the athlete's configured maximum), 'age' (220 minus their age AT THE TIME OF THIS ACTIVITY — the age is taken from the activity's own date, so an old activity's zones do not drift as the athlete ages), 'reserve' (heart-rate reserve / Karvonen, using their resting and max HR), or 'observed' (the peak heart rate in this activity, used when nothing is configured). Only the HR zones depend on the athlete's current settings; every other field here (segments, elevation, route) is derived purely from the immutable recorded stream and is identical whenever the activity is viewed"`
	HRMaxBpm         int                    `json:"hr_max_bpm,omitempty" jsonschema:"the maximum heart rate the zone boundaries were derived from"`
	HRRestBpm        int                    `json:"hr_rest_bpm,omitempty" jsonschema:"the resting heart rate used for reserve (Karvonen) zones, when applicable"`
}

// StreamStat is a simple avg/min/max summary of a single sensor channel.
type StreamStat struct {
	Avg float64 `json:"avg"`
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type StreamSpeedStat struct {
	AvgKmh       float64 `json:"avg_kmh"`
	MaxKmh       float64 `json:"max_kmh"`
	AvgPaceMinKm float64 `json:"avg_pace_min_km" jsonschema:"average pace in minutes per kilometre"`
}

type StreamElevationStat struct {
	GainM        float64      `json:"gain_m" jsonschema:"total elevation gained (sum of positive altitude changes), in metres"`
	LossM        float64      `json:"loss_m" jsonschema:"total elevation lost (sum of negative altitude changes, as a positive number), in metres"`
	MinM         float64      `json:"min_m"`
	MaxM         float64      `json:"max_m"`
	BiggestClimb *StreamClimb `json:"biggest_climb,omitempty" jsonschema:"the single largest sustained ascent in the activity"`
}

// StreamClimb describes one sustained ascent: how much was gained, over how far, and at
// what average gradient. FromPoint/ToPoint index the raw stream so it can be located.
type StreamClimb struct {
	GainM      float64 `json:"gain_m"`
	DistanceKm float64 `json:"distance_km"`
	GradePct   float64 `json:"grade_pct" jsonschema:"average gradient of the climb, percent"`
	FromPoint  int     `json:"from_point"`
	ToPoint    int     `json:"to_point"`
}

// StreamElevationPoint is one sample of the altitude-over-distance profile.
type StreamElevationPoint struct {
	DistanceKm float64 `json:"distance_km"`
	AltitudeM  float64 `json:"altitude_m"`
}

type StreamPowerStat struct {
	AvgW   float64 `json:"avg_w"`
	MaxW   float64 `json:"max_w"`
	WorkKj float64 `json:"work_kj" jsonschema:"total mechanical work, in kilojoules"`
}

// StreamSegment is one split of a moving activity — one distance unit long, except the
// final split which may be shorter. Distances are in the activity's distance_unit;
// pace/speed carry both forms so runners (pace) and cyclists (speed) each read what they
// expect. FromPoint/ToPoint are indices into the raw stream so a client can highlight the
// split on the route.
type StreamSegment struct {
	Index           int     `json:"index" jsonschema:"1-based split number"`
	DistanceUnit    string  `json:"distance_unit"`
	Distance        float64 `json:"distance" jsonschema:"length of this split in distance_unit (the last split may be a partial unit)"`
	ElapsedSeconds  int64   `json:"elapsed_seconds"`
	AvgSpeedKmh     float64 `json:"avg_speed_kmh,omitempty"`
	AvgPaceMinKm    float64 `json:"avg_pace_min_km,omitempty" jsonschema:"average pace in minutes per kilometre over this split"`
	AvgHeartrateBpm *int    `json:"avg_heartrate_bpm,omitempty"`
	AvgCadenceRpm   *int    `json:"avg_cadence_rpm,omitempty"`
	AvgWatts        *int    `json:"avg_watts,omitempty"`
	ElevationGainM  float64 `json:"elevation_gain_m,omitempty"`
	FromPoint       int     `json:"from_point" jsonschema:"first raw-sample index of this split"`
	ToPoint         int     `json:"to_point" jsonschema:"last raw-sample index of this split"`
}

// StreamRoute is a compact summary of a GPS path: enough to place, size and sketch the
// route without shipping every point. Polyline is a down-sampled overview, not the full
// trace (the raw latlng series is still available via the stream series when needed).
type StreamRoute struct {
	PointCount  int         `json:"point_count"`
	DistanceKm  float64     `json:"distance_km" jsonschema:"total GPS path length in kilometres (haversine over the recorded points)"`
	Start       []float64   `json:"start,omitempty" jsonschema:"[lat, lng] of the first GPS point"`
	End         []float64   `json:"end,omitempty" jsonschema:"[lat, lng] of the last GPS point"`
	BoundingBox *StreamBBox `json:"bounding_box,omitempty"`
	Polyline    [][]float64 `json:"polyline,omitempty" jsonschema:"a down-sampled [lat, lng] overview of the route"`
}

type StreamBBox struct {
	MinLat float64 `json:"min_lat"`
	MinLng float64 `json:"min_lng"`
	MaxLat float64 `json:"max_lat"`
	MaxLng float64 `json:"max_lng"`
}

// StreamHRZone is time-in-zone for one of five heart-rate zones. MaxBpm is 0 for the
// open-ended top zone. Percent is of total time with a valid heart-rate reading.
type StreamHRZone struct {
	Zone    int     `json:"zone" jsonschema:"1 (easy) to 5 (maximal)"`
	Name    string  `json:"name"`
	MinBpm  int     `json:"min_bpm"`
	MaxBpm  int     `json:"max_bpm" jsonschema:"upper bound of the zone; 0 means open-ended (the top zone)"`
	Seconds int64   `json:"seconds"`
	Percent float64 `json:"percent" jsonschema:"share of heart-rate time spent in this zone, 0-100"`
}
