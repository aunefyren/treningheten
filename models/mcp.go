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
	HasStreams       bool             `json:"has_streams" jsonschema:"true if this activity has Strava sensor streams; call get_activity_streams for the time-series detail"`
	HasSoundtrack    bool             `json:"has_soundtrack" jsonschema:"true if listening history (music/podcast/audiobook) was matched to this session; call get_activity_soundtrack for the tracks. The soundtrack is a session-level fact, so every activity in the same session shares it"`
	CountsTowardGoal bool             `json:"counts_toward_goal" jsonschema:"whether the parent session tallies toward the user's weekly goal; false means it is logged but deliberately excluded (a session-level flag, so every activity in the same session shares it)"`
	Sets             []MCPActivitySet `json:"sets,omitempty"`
}

// MCPActivitySummary is the slim, aggregated view of one activity (operation) used by the
// list_activities search. It mirrors the /exercises timeline's query-time aggregation: per-set
// distance/time/reps are SUMmed and the heaviest set weight is TopWeight, so it never loads the
// operation/set/stream tree. For per-set detail, tags, description and soundtrack, drill into
// one activity with get_activity by its id.
type MCPActivitySummary struct {
	ID                   string     `json:"id" jsonschema:"stable id for this activity (the operation id); pass to get_activity, get_activity_streams or get_activity_soundtrack"`
	Date                 time.Time  `json:"date" jsonschema:"the calendar day of the session"`
	Time                 *time.Time `json:"time,omitempty" jsonschema:"the session's start time, when known"`
	Action               string     `json:"action" jsonschema:"the exercise type, e.g. Run, Bicycling, Weight Training, or a specific movement like Bench Press"`
	Type                 string     `json:"type" jsonschema:"moving, timing or lifting"`
	Note                 string     `json:"note,omitempty" jsonschema:"the user's short manual note on this activity"`
	Distance             float64    `json:"distance" jsonschema:"total distance summed across the activity's sets, in distance_unit"`
	DistanceUnit         string     `json:"distance_unit,omitempty"`
	DurationSeconds      int64      `json:"duration_seconds" jsonschema:"total time summed across the activity's sets, in seconds"`
	Repetitions          float64    `json:"repetitions" jsonschema:"total repetitions summed across the activity's sets"`
	TopWeight            float64    `json:"top_weight" jsonschema:"heaviest weight recorded on any of the activity's sets, in weight_unit"`
	WeightUnit           string     `json:"weight_unit,omitempty"`
	SetCount             int        `json:"set_count" jsonschema:"number of sets in this activity"`
	HasStreams           bool       `json:"has_streams" jsonschema:"true if this activity has Strava sensor streams; call get_activity_streams for the time-series detail"`
	Source               string     `json:"source" jsonschema:"where this activity came from: strava, hevy or manual"`
	CountsTowardGoal     bool       `json:"counts_toward_goal" jsonschema:"whether the parent session tallies toward the user's weekly goal; false means it is logged but deliberately excluded"`
	SessionID            string     `json:"session_id" jsonschema:"id of the parent session; activities sharing a session_id were logged together"`
	SessionActivityCount int        `json:"session_activity_count" jsonschema:"total number of activities in the parent session, independent of the current search filter"`
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
	CadenceRpm   []int       `json:"cadence_rpm,omitempty"`
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
