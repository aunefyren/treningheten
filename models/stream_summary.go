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

// OperationStreamRollup holds the scalar summaries precomputed from an activity's Strava
// stream and stored as columns on the Operation, so the activity list
// (GetActivityFeedForUser) can surface heart rate, cadence, temperature and elevation gain
// without loading and parsing the stream blob for every row. Each field is nil when the
// stream lacks that channel. The heavier per-km/zone detail stays behind get_activity_streams
// and the get_activity include parameter.
type OperationStreamRollup struct {
	AvgHeartrate   *int
	MaxHeartrate   *int
	AvgCadence     *int
	TempC          *int
	ElevationGainM *float64
}

// ComputeStreamRollup derives the stored list scalars from a stream. Heart rate and cadence
// average only positive samples (both read 0 while paused); the max heart rate reuses the
// plausible cap so a dropout can't inflate it; temperature averages all samples (it can be
// negative); elevation gain sums the positive altitude changes. Shared by the Strava sync and
// the one-time backfill so the two stay identical.
func ComputeStreamRollup(s *StravaActivityStreams) OperationStreamRollup {
	rollup := OperationStreamRollup{}
	if s == nil {
		return rollup
	}

	if s.Heartrate != nil {
		// Cap at the plausible max so a sensor dropout can't drag the average up, matching how
		// the max (ObservedMaxHeartrate) already filters spikes.
		if avg, ok := avgPositive(s.Heartrate.Data, maxPlausibleHeartrate); ok {
			rollup.AvgHeartrate = &avg
		}
		if peak := ObservedMaxHeartrate(s); peak > 0 {
			rollup.MaxHeartrate = &peak
		}
	}
	if s.Cadence != nil {
		if avg, ok := avgPositive(s.Cadence.Data, 0); ok {
			rollup.AvgCadence = &avg
		}
	}
	if s.Temp != nil && len(s.Temp.Data) > 0 {
		sum := 0
		for _, v := range s.Temp.Data {
			sum += v
		}
		avg := int(roundHalf(float64(sum) / float64(len(s.Temp.Data))))
		rollup.TempC = &avg
	}
	if s.Altitude != nil && len(s.Altitude.Data) > 1 {
		gain := 0.0
		prev := s.Altitude.Data[0]
		for _, v := range s.Altitude.Data[1:] {
			if d := v - prev; d > 0 {
				gain += d
			}
			prev = v
		}
		gain = roundHalf(gain*10) / 10
		rollup.ElevationGainM = &gain
	}
	return rollup
}

// avgPositive returns the rounded mean of the strictly-positive samples, and false when there
// are none (the channel was all zero/paused or empty). Samples above max are treated as sensor
// dropouts and skipped; max <= 0 disables the cap.
func avgPositive(data []int, max int) (int, bool) {
	sum, count := 0, 0
	for _, v := range data {
		if v <= 0 || (max > 0 && v > max) {
			continue
		}
		sum += v
		count++
	}
	if count == 0 {
		return 0, false
	}
	return int(roundHalf(float64(sum) / float64(count))), true
}

// roundHalf rounds to the nearest integer, half away from zero. Kept local to models so the
// rollup has no dependency on the controllers' rounding helpers.
func roundHalf(v float64) float64 {
	if v < 0 {
		return float64(int(v - 0.5))
	}
	return float64(int(v + 0.5))
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
	Analysis         *StreamAnalysis        `json:"analysis,omitempty" jsonschema:"precomputed derived metrics (aerobic decoupling, first/second-half splits, pace consistency, breaks, HR-by-gradient) — the kind of second-order analysis a coach reads off the raw stream"`
	HRMaxBasis       string                 `json:"hr_max_basis,omitempty" jsonschema:"how the HR zones were anchored: 'max' (the athlete's configured maximum), 'age' (220 minus their age AT THE TIME OF THIS ACTIVITY — the age is taken from the activity's own date, so an old activity's zones do not drift as the athlete ages), 'reserve' (heart-rate reserve / Karvonen, using their resting and max HR), or 'observed' (the peak heart rate in this activity, used when nothing is configured). Only the HR zones depend on the athlete's current settings; every other field here (segments, elevation, route) is derived purely from the immutable recorded stream and is identical whenever the activity is viewed"`
	HRMaxBpm         int                    `json:"hr_max_bpm,omitempty" jsonschema:"the maximum heart rate the zone boundaries were derived from"`
	HRRestBpm        int                    `json:"hr_rest_bpm,omitempty" jsonschema:"the resting heart rate used for reserve (Karvonen) zones, when applicable"`
}

// StreamAnalysis holds second-order metrics derived from the raw stream — the numbers a
// coach hand-computes to judge a session. Every field is optional: each needs specific
// channels (decoupling and the halves/gradient views need heart rate plus a distance or
// speed signal), so a field is nil when its inputs are absent. All are computed purely from
// the immutable recorded stream, so they are stable whenever the activity is viewed.
type StreamAnalysis struct {
	DecouplingPct     *float64                 `json:"decoupling_pct,omitempty" jsonschema:"aerobic decoupling: the percent drop in pace-to-heart-rate efficiency from the first half to the second half of the activity. Near 0 (under ~5%) means a well-controlled aerobic effort held pace without HR drift; higher values indicate cardiac drift or fatigue. A negative value means efficiency improved in the second half (e.g. a warm-up or negative split)"`
	SplitHalves       *StreamSplitHalves       `json:"split_halves,omitempty" jsonschema:"first-half vs second-half comparison (time, distance, avg HR, avg pace) — positive/negative split detection in two rows"`
	PaceStdDevSeconds *float64                 `json:"pace_std_dev_seconds,omitempty" jsonschema:"standard deviation of the per-split pace, in seconds per unit distance; pacing consistency in a single number, where lower is steadier. Uses only the full-length splits"`
	Breaks            *StreamBreaks            `json:"breaks,omitempty" jsonschema:"stops and walk breaks detected as sustained stretches of near-zero speed"`
	HRByGradient      []StreamHRGradientBucket `json:"hr_by_gradient,omitempty" jsonschema:"time and average heart rate bucketed by terrain gradient — shows how much the terrain, rather than effort, drove heart rate"`
}

// StreamSplitHalves compares the first and second halves of the activity, split at the
// midpoint of elapsed moving time.
type StreamSplitHalves struct {
	First  StreamHalf `json:"first"`
	Second StreamHalf `json:"second"`
}

// StreamHalf is the aggregate for one half of a split-halves comparison.
type StreamHalf struct {
	ElapsedSeconds  int64   `json:"elapsed_seconds"`
	DistanceKm      float64 `json:"distance_km,omitempty"`
	AvgHeartrateBpm *int    `json:"avg_heartrate_bpm,omitempty"`
	AvgPaceMinKm    float64 `json:"avg_pace_min_km,omitempty"`
}

// StreamBreak is one detected pause: when it started and ended (seconds from workout start)
// and how long it lasted.
type StreamBreak struct {
	FromSeconds     int `json:"from_seconds"`
	ToSeconds       int `json:"to_seconds"`
	DurationSeconds int `json:"duration_seconds"`
}

// StreamBreaks summarizes the pauses in an activity: how many, their combined duration, and
// the individual breaks in order.
type StreamBreaks struct {
	Count                int           `json:"count"`
	TotalDurationSeconds int           `json:"total_duration_seconds"`
	Breaks               []StreamBreak `json:"breaks,omitempty"`
}

// StreamHRGradientBucket is time-in-gradient: how long was spent on terrain in this grade
// band and the average heart rate (and pace) there.
type StreamHRGradientBucket struct {
	Label           string   `json:"label" jsonschema:"human-readable gradient band, e.g. 'steep descent', 'flat', 'steep climb'"`
	MinGradePct     *float64 `json:"min_grade_pct,omitempty" jsonschema:"lower bound of the band in percent grade; null for the open-ended first (descent) band"`
	MaxGradePct     *float64 `json:"max_grade_pct,omitempty" jsonschema:"upper bound of the band in percent grade; null for the open-ended last (climb) band"`
	Seconds         int64    `json:"seconds"`
	AvgHeartrateBpm *int     `json:"avg_heartrate_bpm,omitempty"`
	AvgPaceMinKm    float64  `json:"avg_pace_min_km,omitempty"`
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
