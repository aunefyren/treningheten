package models

import "time"

// MCP DTOs are flattened, LLM-friendly views of the operation-centric data model.
// Durations are exposed in seconds rather than nanoseconds.

type MCPProfile struct {
	ID          string    `json:"id" jsonschema:"the user's unique id"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	Email       string    `json:"email"`
	Admin       bool      `json:"admin"`
	MemberSince time.Time `json:"member_since"`
}

type MCPWeight struct {
	Date   time.Time `json:"date"`
	Weight float64   `json:"weight"`
}

// MCPSeason is the authenticated user's personal view of a season. The goal fields
// (weekly_goal, competing, sickleave_left) are only populated when the user has
// joined the season.
type MCPSeason struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	Start          time.Time `json:"start"`
	End            time.Time `json:"end"`
	Status         string    `json:"status" jsonschema:"upcoming, ongoing or ended, relative to now"`
	Joined         bool      `json:"joined" jsonschema:"whether the authenticated user participates in this season"`
	WeeklyGoal     *int      `json:"weekly_goal,omitempty" jsonschema:"the user's target number of workouts per week in this season"`
	Competing      *bool     `json:"competing,omitempty" jsonschema:"whether the user competes for the prize (vs participating casually)"`
	SickleaveTotal int       `json:"sickleave_total" jsonschema:"sick-leave weeks the season allows"`
	SickleaveLeft  *int      `json:"sickleave_left,omitempty" jsonschema:"sick-leave weeks the user has remaining"`
}

// MCPAchievement is the catalog achievement plus the authenticated user's earned
// state. Description is "Hidden" for hidden achievements the user has not yet earned.
type MCPAchievement struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	Category     string     `json:"category,omitempty"`
	Earned       bool       `json:"earned" jsonschema:"whether the authenticated user has unlocked this achievement"`
	TimesEarned  int        `json:"times_earned" jsonschema:"how many times the user has been awarded it (can exceed 1 for repeatable achievements)"`
	LastEarnedAt *time.Time `json:"last_earned_at,omitempty"`
}

// MCPAchievementDelegation is a single award of an achievement to the authenticated
// user (the user owns every delegation returned).
type MCPAchievementDelegation struct {
	ID              string    `json:"id"`
	AchievementID   string    `json:"achievement_id"`
	AchievementName string    `json:"achievement_name"`
	Category        string    `json:"category,omitempty"`
	GivenAt         time.Time `json:"given_at"`
	Seen            bool      `json:"seen" jsonschema:"whether the user has acknowledged this award"`
}

type MCPActivitySet struct {
	Repetitions       *float64 `json:"repetitions,omitempty"`
	Weight            *float64 `json:"weight,omitempty"`
	WeightUnit        string   `json:"weight_unit,omitempty"`
	Distance          *float64 `json:"distance,omitempty"`
	DistanceUnit      string   `json:"distance_unit,omitempty"`
	TimeSeconds       *int64   `json:"time_seconds,omitempty" jsonschema:"elapsed time for the set, in seconds"`
	MovingTimeSeconds *int64   `json:"moving_time_seconds,omitempty" jsonschema:"active/moving time (excludes pauses), in seconds; usually only present for Strava-imported activities"`
}

type MCPActivity struct {
	ID               string           `json:"id" jsonschema:"stable id for this activity; pass to get_activity or get_activity_streams"`
	Date             time.Time        `json:"date"`
	Action           string           `json:"action" jsonschema:"the exercise type, e.g. Run, Bicycling, Weight Training, or a specific movement like Bench Press"`
	Type             string           `json:"type" jsonschema:"moving, timing or lifting"`
	Source           string           `json:"source" jsonschema:"where this activity came from: strava (imported from Strava), hevy (imported from Hevy), or manual (logged in the app)"`
	Note             string           `json:"note,omitempty" jsonschema:"the user's short manual note on this activity"`
	Description      string           `json:"description,omitempty" jsonschema:"longer free-text description; for Strava-imported activities this is the athlete's description from Strava, and for Hevy custom exercises it is the user's per-exercise note. Belongs to this activity's action specifically"`
	Tags             []string         `json:"tags,omitempty" jsonschema:"workout category tags from a fixed vocabulary: race, long-run, workout, commute, for-a-cause, recovery, with-pet, with-kid"`
	Equipment        string           `json:"equipment,omitempty"`
	DurationSeconds  *int64           `json:"duration_seconds,omitempty"`
	HasStreams       bool             `json:"has_streams" jsonschema:"true if this activity has Strava sensor streams. For the ANALYSIS layer (per-km/mile splits, heart-rate zones, aerobic decoupling, walk/stop breaks, HR-by-gradient) re-call get_activity with the include parameter — that is far cheaper than get_activity_streams, which is only needed for the raw second-by-second series"`
	HasSoundtrack    bool             `json:"has_soundtrack" jsonschema:"true if listening history (music/podcast/audiobook) was matched to this session; call get_activity_soundtrack for the tracks. The soundtrack is a session-level fact, so every activity in the same session shares it"`
	CountsTowardGoal bool             `json:"counts_toward_goal" jsonschema:"whether the parent session tallies toward the user's weekly goal; false means it is logged but deliberately excluded (a session-level flag, so every activity in the same session shares it)"`
	Sets             []MCPActivitySet `json:"sets,omitempty"`
	// AnalysisHint is an in-band nudge: on a stream-backed activity fetched WITHOUT include,
	// it names the include blocks the caller can re-request for the analysis layer, so the
	// depth is discoverable at the moment of need rather than only in the tool description.
	// Empty once include was supplied (or when there are no streams to analyse).
	AnalysisHint string `json:"analysis_hint,omitempty" jsonschema:"present only when this stream-backed activity was fetched without include; tells you how to re-call get_activity to obtain the analysis blocks"`
	// StreamSummary carries the processed sensor blocks (segments, hr_zones, elevation,
	// route, elevation_profile, analysis) a caller opted into via get_activity's include
	// parameter. It is present only for stream-backed activities when include is non-empty,
	// and holds just the requested blocks (the raw time-series stays in get_activity_streams).
	StreamSummary *StreamSummary `json:"stream_summary,omitempty"`
	// BestEfforts is attached when include holds "efforts": the activity's fastest stretch over
	// each tracked distance, and whether it is the user's current record for that distance.
	BestEfforts []MCPBestEffort `json:"best_efforts,omitempty" jsonschema:"present when include has 'efforts': this activity's fastest time over each tracked distance it covered (runs, walks and hikes: 1 km, 1 mile, 5 km, 10 km, half marathon, marathon; rides: 10, 20, 40 and 100 km)"`
}

// MCPBestEffort is an activity's fastest time over one tracked distance.
type MCPBestEffort struct {
	Label            string  `json:"label" jsonschema:"the distance's name, e.g. 5 km or Half marathon"`
	DistanceM        int     `json:"distance_m" jsonschema:"the distance in metres"`
	ElapsedSeconds   int     `json:"elapsed_seconds" jsonschema:"the fastest time over the distance in seconds, pauses inside the stretch included (stopwatch time)"`
	PaceMinKm        float64 `json:"pace_min_km,omitempty" jsonschema:"that time as minutes per kilometre; for foot sports"`
	SpeedKmh         float64 `json:"speed_kmh,omitempty" jsonschema:"that time as km/h; for rides"`
	IsPersonalRecord bool    `json:"is_personal_record" jsonschema:"true when this is the user's fastest ever time over the distance for this activity type"`
}

// MCPActivitySummary is the slim, aggregated view of one activity (operation), nested inside
// the workout it belongs to in the list_workouts search. It mirrors the /exercises timeline's
// query-time aggregation: per-set distance/time/reps are SUMmed and the heaviest set weight is
// TopWeight, so it never loads the operation/set/stream tree. For per-set detail, tags,
// description and soundtrack, drill into one activity with get_activity by its id.
type MCPActivitySummary struct {
	ID                   string     `json:"id" jsonschema:"stable id for this activity (the operation id); pass to get_activity, get_activity_streams or get_activity_soundtrack"`
	Date                 time.Time  `json:"date" jsonschema:"the calendar day of the session"`
	Time                 *time.Time `json:"time,omitempty" jsonschema:"the session's start time, when known"`
	Action               string     `json:"action" jsonschema:"the exercise type, e.g. Run, Bicycling, Weight Training, or a specific movement like Bench Press"`
	Type                 string     `json:"type" jsonschema:"moving, timing or lifting"`
	Note                 string     `json:"note,omitempty" jsonschema:"the user's short manual note on this activity"`
	Distance             float64    `json:"distance" jsonschema:"total distance summed across the activity's sets, in distance_unit"`
	DistanceUnit         string     `json:"distance_unit,omitempty"`
	DurationSeconds      int64      `json:"duration_seconds" jsonschema:"total elapsed time summed across the activity's sets, in seconds"`
	MovingSeconds        int64      `json:"moving_seconds,omitempty" jsonschema:"active/moving time (excludes pauses), in seconds; usually only present for Strava-imported activities"`
	AvgPaceMinKm         float64    `json:"avg_pace_min_km,omitempty" jsonschema:"average pace in minutes per kilometre, from moving time when available; only for activities that record distance"`
	AvgHeartrateBpm      *int       `json:"avg_heartrate_bpm,omitempty" jsonschema:"average heart rate over the activity, from its Strava stream; absent without one"`
	MaxHeartrateBpm      *int       `json:"max_heartrate_bpm,omitempty" jsonschema:"peak heart rate over the activity, from its Strava stream"`
	AvgCadenceRpm        *int       `json:"avg_cadence_rpm,omitempty" jsonschema:"average cadence in cadence_unit, from its Strava stream"`
	CadenceUnit          string     `json:"cadence_unit,omitempty" jsonschema:"unit of avg_cadence_rpm: 'spm' (steps per minute, both legs) for runs, walks and hikes; 'rpm' otherwise"`
	TemperatureC         *int       `json:"temperature_c,omitempty" jsonschema:"average recorded temperature in Celsius, from its Strava stream"`
	ElevationGainM       *float64   `json:"elevation_gain_m,omitempty" jsonschema:"total elevation gained in metres, from its Strava stream"`
	AvgWatts             *int       `json:"avg_watts,omitempty" jsonschema:"average power in watts over the activity, zero samples (coasting) included, from its Strava stream; absent without a power meter"`
	Repetitions          float64    `json:"repetitions" jsonschema:"total repetitions summed across the activity's sets"`
	TopWeight            float64    `json:"top_weight" jsonschema:"heaviest weight recorded on any of the activity's sets, in weight_unit"`
	WeightUnit           string     `json:"weight_unit,omitempty"`
	SetCount             int        `json:"set_count" jsonschema:"number of sets in this activity"`
	HasStreams           bool       `json:"has_streams" jsonschema:"true if this activity has Strava sensor streams. For splits, heart-rate zones and derived metrics (decoupling, breaks) call get_activity with include; get_activity_streams is only for the raw second-by-second series"`
	Source               string     `json:"source" jsonschema:"where this activity came from: strava, hevy or manual"`
	CountsTowardGoal     bool       `json:"counts_toward_goal" jsonschema:"whether the parent session tallies toward the user's weekly goal; false means it is logged but deliberately excluded"`
	SessionID            string     `json:"session_id" jsonschema:"id of the parent session; activities sharing a session_id were logged together"`
	SessionActivityCount int        `json:"session_activity_count" jsonschema:"total number of activities in the parent session, independent of the current search filter"`
}

// MCPWorkoutSummary is one workout session in the list_workouts search — the unit the search
// pages, counts and sorts, with the activities it was made of nested inside it.
//
// The search used to return activities (operations) directly, which made a session with no
// activities unreachable: the app's most common way to log a workout (the front page's "+"
// button) creates a session with zero operations, so those workouts existed on the user's
// timeline but not in anything an LLM could see, and "how many workouts did I do" answered in
// exercise rows. Metrics are summed across the whole session; DurationSeconds prefers the
// session's own recorded duration over the summed set times, so a workout logged with nothing
// but a duration still reports one. See docs/mcp.md and docs/exercises.md.
type MCPWorkoutSummary struct {
	ID               string               `json:"id" jsonschema:"stable id for this workout (the session id); pass to get_activity_soundtrack. Per-activity detail is addressed by the ids inside activities"`
	Date             time.Time            `json:"date" jsonschema:"the calendar day of the workout"`
	Time             *time.Time           `json:"time,omitempty" jsonschema:"the workout's start time, when known"`
	Note             string               `json:"note,omitempty" jsonschema:"the user's short manual note on the whole workout"`
	Distance         float64              `json:"distance" jsonschema:"total distance summed across every set of every activity, in distance_unit"`
	DistanceUnit     string               `json:"distance_unit,omitempty"`
	DurationSeconds  int64                `json:"duration_seconds" jsonschema:"how long the workout took, in seconds: the session's own recorded duration when it has one, otherwise the summed set times"`
	MovingSeconds    int64                `json:"moving_seconds,omitempty" jsonschema:"active/moving time (excludes pauses), in seconds; usually only present for Strava-imported workouts"`
	Repetitions      float64              `json:"repetitions" jsonschema:"total repetitions summed across every set of every activity"`
	TopWeight        float64              `json:"top_weight" jsonschema:"heaviest weight recorded on any set of any activity, in weight_unit"`
	WeightUnit       string               `json:"weight_unit,omitempty"`
	SetCount         int                  `json:"set_count" jsonschema:"number of sets across the whole workout"`
	HasStreams       bool                 `json:"has_streams" jsonschema:"true if any activity in this workout has Strava sensor streams; the activity ids inside are what get_activity and get_activity_streams take"`
	Source           string               `json:"source" jsonschema:"where this workout came from: strava, hevy or manual"`
	CountsTowardGoal bool                 `json:"counts_toward_goal" jsonschema:"whether this workout tallies toward the user's weekly goal; false means it is logged but deliberately excluded"`
	ActivityCount    int                  `json:"activity_count" jsonschema:"how many activities the workout is made of; 0 means the user logged the workout without recording what they did"`
	Activities       []MCPActivitySummary `json:"activities" jsonschema:"the activities this workout was made of, in the order they were logged; empty when the user just logged that they worked out"`
}

// MCPSoundtrackTrack is one item the user listened to during a session, flattened
// for the LLM. Durations are seconds; times are absolute so the LLM can place a
// track within the workout (and, with get_activity_streams, correlate it to effort).
type MCPSoundtrackTrack struct {
	Type               string     `json:"type" jsonschema:"song, podcast or audiobook"`
	Title              string     `json:"title"`
	Artist             *string    `json:"artist,omitempty"`
	Album              *string    `json:"album,omitempty"`
	Provider           string     `json:"provider" jsonschema:"where the play was recorded: plex, spotify or audiobookshelf"`
	StartedAt          time.Time  `json:"started_at"`
	EndedAt            *time.Time `json:"ended_at,omitempty"`
	StartedBefore      bool       `json:"started_before,omitempty" jsonschema:"true when the item was already playing when the session began, so started_at is the session start rather than the moment the play began"`
	TrackLengthSeconds *int64     `json:"track_length_seconds,omitempty" jsonschema:"full length of the item in seconds (not necessarily how long it was played)"`
}

// MCPWorkoutSoundtrack is the listening history matched to one session. Like streams,
// it is fetched on demand (it can be long) rather than inlined into the activity.
// The soundtrack is session-level: any activity id belonging to the session returns
// the same tracks.
type MCPWorkoutSoundtrack struct {
	HasSoundtrack bool                 `json:"has_soundtrack"`
	Message       string               `json:"message,omitempty"`
	RetrievedAt   *time.Time           `json:"retrieved_at,omitempty" jsonschema:"when the listening history for this session was last pulled"`
	Tracks        []MCPSoundtrackTrack `json:"tracks,omitempty" jsonschema:"tracks in play order (earliest started_at first)"`
}

// The per-channel stat shapes now live on the shared StreamSummary (stream_summary.go)
// so the MCP tool and the /exercises detail page compute them once. These aliases keep
// the historical MCP type names working for any external reference.
type (
	MCPStreamStat    = StreamStat
	MCPSpeedStat     = StreamSpeedStat
	MCPElevationStat = StreamElevationStat
	MCPPowerStat     = StreamPowerStat
)

// MCPStreamSeries is a downsampled, time-aligned slice of the raw sensor streams.
// All arrays share the same indexing as t_seconds. Only streams the workout
// actually recorded are populated.
type MCPStreamSeries struct {
	ReturnedPoints      int   `json:"returned_points"`
	TotalPointsInWindow int   `json:"total_points_in_window" jsonschema:"how many raw samples exist in the requested window before downsampling"`
	SampledEverySeconds int   `json:"sampled_every_seconds" jsonschema:"approx spacing between returned samples; 1 is full fidelity. Re-call with a smaller window and resolution=1 to zoom in"`
	WindowFromSeconds   int   `json:"window_from_seconds"`
	WindowToSeconds     int   `json:"window_to_seconds"`
	TSeconds            []int `json:"t_seconds" jsonschema:"seconds elapsed from workout start for each sample"`

	HeartrateBpm []int       `json:"heartrate_bpm,omitempty"`
	AltitudeM    []float64   `json:"altitude_m,omitempty"`
	SpeedKmh     []float64   `json:"speed_kmh,omitempty"`
	CadenceRpm   []int       `json:"cadence_rpm,omitempty" jsonschema:"cadence per sample, in the summary's cadence_unit (steps per minute for runs, walks and hikes)"`
	Watts        []int       `json:"watts,omitempty"`
	TempC        []int       `json:"temperature_c,omitempty"`
	LatLng       [][]float64 `json:"latlng,omitempty" jsonschema:"GPS coordinate pairs [lat, lng]"`
}

// MCPWorkoutStreams is the processed view of one workout's Strava sensor data.
// Streams exist ONLY for GPS/sensor activities imported from Strava (runs, rides,
// etc.); strength and manually-logged workouts have none (HasStreams=false). The
// whole-workout summary (header stats, segments, route, HR zones) is the shared
// StreamSummary, embedded so its fields flatten into the tool's JSON; the downsampled
// Series is specific to this tool.
type MCPWorkoutStreams struct {
	HasStreams    bool   `json:"has_streams"`
	Message       string `json:"message,omitempty"`
	StreamSummary        // embedded: available, has_gps, duration_seconds, header stats, segments, route, hr_zones

	Series *MCPStreamSeries `json:"series,omitempty"`
}

// MCPStatWindow holds the totals for one time window. Distance is normalized to km;
// time is in seconds and counts each activity once (activity duration when present,
// else the sum of its set times).
type MCPStatWindow struct {
	Workouts    int     `json:"workouts" jsonschema:"number of logged workouts in this window — the unit the app counts toward goals and shows on /statistics. A workout holds zero or more activities"`
	Activities  int     `json:"activities" jsonschema:"number of logged activities of ANY type (runs, rides, strength, etc.) in this window"`
	DistanceKm  float64 `json:"distance_km" jsonschema:"total logged distance in this window, normalized to kilometres; only activities that record distance contribute, so strength sessions add to activities but not distance_km"`
	TimeSeconds int64   `json:"time_seconds" jsonschema:"total logged activity time in this window, in seconds; only activities that record a duration contribute, so this may cover fewer activities than the activities count"`
}

// MCPStatistics reports training totals over three windows. The windows are rolling
// and computed relative to the current time on each call, not fixed calendar periods.
type MCPStatistics struct {
	PastMonth MCPStatWindow `json:"past_month" jsonschema:"trailing ~1 month up to now"`
	PastYear  MCPStatWindow `json:"past_year" jsonschema:"trailing 12 months up to now"`
	AllTime   MCPStatWindow `json:"all_time" jsonschema:"all recorded activity, no time limit"`
	Streaks   MCPStreaks    `json:"streaks"`
	// Overview is the /statistics Overview tab for one calendar period, present only when the
	// caller passed period.
	Overview *MCPOverview `json:"overview,omitempty" jsonschema:"present only when period was passed: the same overview the app's /statistics page shows — totals for the period and the previous one of equal length (workouts, active seconds, active days, distance per unit), one bucket per week (per month for all), the activity mix by time, personal streaks and records set in the period"`
}

// MCPStreaks groups the user's streaks. Only personal (season-independent) streaks
// are reported; season/goal streaks are not included.
type MCPStreaks struct {
	Personal MCPPersonalStreaks `json:"personal" jsonschema:"activity streaks independent of seasons and goals: a day or week counts when any activity was logged, regardless of whether a weekly goal was met"`
}

type MCPPersonalStreaks struct {
	WeekCurrent int `json:"week_current" jsonschema:"current run of consecutive ISO weeks each containing at least one logged activity; 0 if the streak is no longer alive (nothing logged this week or last week)"`
	WeekBest    int `json:"week_best" jsonschema:"longest run of consecutive weeks with activity, ever"`
	DayCurrent  int `json:"day_current" jsonschema:"current run of consecutive days each with at least one logged activity; 0 if nothing was logged today or yesterday"`
	DayBest     int `json:"day_best" jsonschema:"longest run of consecutive days with activity, ever"`
}

// MCPOverview is the /statistics Overview for one period, with string ids (uuid.UUID would be
// described as a byte array in the tool's output schema).
type MCPOverview struct {
	Period        string             `json:"period"`
	Start         time.Time          `json:"start"`
	End           time.Time          `json:"end"`
	Bucket        string             `json:"bucket" jsonschema:"week or month: the size of each entry in buckets"`
	Totals        StatisticsTotals   `json:"totals" jsonschema:"workouts, active seconds, active days and distance per unit (never mixed) in the period"`
	Previous      *StatisticsTotals  `json:"previous,omitempty" jsonschema:"the same totals for the previous period of equal length; absent for all"`
	Buckets       []StatisticsBucket `json:"buckets" jsonschema:"one entry per week (Monday start) or month of the period, empty ones included"`
	Mix           []MCPMixItem       `json:"mix" jsonschema:"activity types in the period, most time first"`
	Streaks       StatisticsStreaks  `json:"streaks"`
	RecentRecords []MCPRecentRecord  `json:"recent_records" jsonschema:"personal records set in the period, newest first, at most six"`
}

// MCPMixItem is one activity type's share of a period.
type MCPMixItem struct {
	ActionID        string `json:"action_id"`
	Action          string `json:"action"`
	Activities      int    `json:"activities"`
	DurationSeconds int64  `json:"duration_seconds"`
}

// MCPRecentRecord is a record set inside an overview period.
type MCPRecentRecord struct {
	ActionID string            `json:"action_id"`
	Action   string            `json:"action"`
	Record   MCPPersonalRecord `json:"record"`
}

// ── Personal records ─────────────────────────────────────────────────────────

// MCPPersonalRecords is every personal record, grouped by activity type, most logged first.
type MCPPersonalRecords struct {
	ActivityTypes []MCPRecordGroup `json:"activity_types"`
}

// MCPRecordGroup is one activity type's records.
type MCPRecordGroup struct {
	ActionID   string              `json:"action_id" jsonschema:"the activity type's id; pass as action to get_personal_records or get_activity_type_statistics"`
	Action     string              `json:"action" jsonschema:"the activity type, e.g. Run or Bench Press (Barbell)"`
	Activities int                 `json:"activities" jsonschema:"how many activities of this type the user has logged, ever"`
	Records    []MCPPersonalRecord `json:"records"`
}

// MCPPersonalRecord is the best the user has done for one measure, and what it beat.
type MCPPersonalRecord struct {
	Key           string           `json:"key" jsonschema:"stable record id within the activity type; pass as key to get_personal_records for its history. effort_<metres> (fastest time over a distance), longest_distance, longest_duration, most_elevation, heaviest_set, estimated_one_rep_max, most_reps, most_volume"`
	Label         string           `json:"label" jsonschema:"human name, e.g. 5 km, Longest distance, Heaviest set"`
	Kind          string           `json:"kind" jsonschema:"what value measures: time (seconds, LOWER is better), distance (in unit), duration (seconds), elevation (metres), weight (in unit) or reps"`
	Value         float64          `json:"value"`
	Unit          string           `json:"unit" jsonschema:"s for times and durations, m for elevation, km or miles for distance, kg or lb for weight, reps for reps"`
	Reps          *float64         `json:"reps,omitempty" jsonschema:"for heaviest_set and estimated_one_rep_max: the reps of the set behind it"`
	PaceMinKm     float64          `json:"pace_min_km,omitempty" jsonschema:"for a foot-sport effort_ record: the time as minutes per kilometre"`
	SpeedKmh      float64          `json:"speed_kmh,omitempty" jsonschema:"for a ride effort_ record: the time as km/h"`
	SetAt         time.Time        `json:"set_at" jsonschema:"when the record was set"`
	ActivityID    string           `json:"activity_id" jsonschema:"the activity it was set in; pass to get_activity"`
	Fresh         bool             `json:"fresh" jsonschema:"true when set in the last 30 days"`
	Improvements  int              `json:"improvements" jsonschema:"how many times this record has been beaten since it was first set"`
	Previous      *MCPRecordEntry  `json:"previous,omitempty" jsonschema:"the best it beat; absent when the record has never been improved (with from/to: within the range)"`
	AllTimeRecord bool             `json:"all_time_record" jsonschema:"true when this is the user's best ever; always true without from/to"`
	AllTimeBest   *MCPRecordEntry  `json:"all_time_best,omitempty" jsonschema:"with from/to, when the range's best is not the all-time record: that record, to compare with"`
	History       []MCPRecordEntry `json:"history,omitempty" jsonschema:"present only when key was passed: every value that beat the ones before it, oldest first (within the range when from/to are passed)"`
}

// MCPRecordEntry is one value in a record's history.
type MCPRecordEntry struct {
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	Reps       *float64  `json:"reps,omitempty"`
	SetAt      time.Time `json:"set_at"`
	ActivityID string    `json:"activity_id" jsonschema:"pass to get_activity"`
}

// ── Activity type statistics ─────────────────────────────────────────────────

// MCPActivityTypeStatistics is one activity type over a date range: what the app's
// /statistics Activity types tab shows, plus one trend point per activity.
type MCPActivityTypeStatistics struct {
	ActionID       string                   `json:"action_id"`
	Action         string                   `json:"action"`
	From           string                   `json:"from" jsonschema:"first day of the range, YYYY-MM-DD"`
	To             string                   `json:"to" jsonschema:"last day of the range, YYYY-MM-DD"`
	Activities     int                      `json:"activities"`
	Workouts       int                      `json:"workouts" jsonschema:"how many distinct workouts those activities belong to"`
	Totals         ActionStatisticsTotals   `json:"totals" jsonschema:"distances and weight volume are per unit and never mixed"`
	Averages       ActionStatisticsAverages `json:"averages" jsonschema:"per-activity means; a field is null when no activity has that measure"`
	Bests          MCPActivityTypeBests     `json:"bests" jsonschema:"the best single activity or set in the range (for all-time bests use get_personal_records)"`
	Sensors        ActionStatisticsSensors  `json:"sensors" jsonschema:"from each activity's sensor stream: heart rate, cadence (cadence_unit: spm or rpm) and power averaged weighted by moving time, the highest heart rate, elevation gain summed"`
	Trend          []MCPTrendPoint          `json:"trend" jsonschema:"one point per activity, oldest first — use it to judge whether the user is getting faster, stronger or fitter"`
	TrendTruncated bool                     `json:"trend_truncated,omitempty" jsonschema:"true when the range held more activities than the trend returns; only the most recent are included"`
}

// MCPActivityTypeBests are the range's bests, each with the activity it came from.
type MCPActivityTypeBests struct {
	LongestDistance *MCPRecordEntry `json:"longest_distance,omitempty"`
	LongestDuration *MCPRecordEntry `json:"longest_duration,omitempty" jsonschema:"value in seconds"`
	MostReps        *MCPRecordEntry `json:"most_reps,omitempty" jsonschema:"reps in a single set"`
	HeaviestSet     *MCPRecordEntry `json:"heaviest_set,omitempty"`
}

// MCPTrendPoint is one activity in an activity type's trend.
type MCPTrendPoint struct {
	ActivityID         string    `json:"activity_id" jsonschema:"pass to get_activity"`
	Date               time.Time `json:"date"`
	Distance           float64   `json:"distance,omitempty"`
	DistanceUnit       string    `json:"distance_unit,omitempty"`
	DurationSeconds    int64     `json:"duration_seconds,omitempty"`
	MovingSeconds      int64     `json:"moving_seconds,omitempty"`
	PaceMinKm          float64   `json:"pace_min_km,omitempty" jsonschema:"minutes per kilometre from moving time (else duration); for distance activities"`
	SpeedKmh           float64   `json:"speed_kmh,omitempty" jsonschema:"km/h from moving time (else duration); for distance activities"`
	AvgHeartrateBpm    *int      `json:"avg_heartrate_bpm,omitempty"`
	Repetitions        float64   `json:"repetitions,omitempty"`
	TopWeight          float64   `json:"top_weight,omitempty" jsonschema:"heaviest set weight, in weight_unit"`
	WeightUnit         string    `json:"weight_unit,omitempty"`
	EstimatedOneRepMax float64   `json:"estimated_one_rep_max,omitempty" jsonschema:"best Epley estimate (weight × (1 + reps/30)) over the activity's sets of 1–10 reps, in weight_unit"`
}
