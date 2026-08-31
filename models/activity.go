package models

import (
	"time"

	"github.com/google/uuid"
)

// ActivityFeedItem is one activity (Operation) in the /exercises timeline, pre-aggregated
// for listing so the client never has to load the operation/set/Strava-stream tree. The
// metric fields are summed from the operation's enabled sets (distance/time/reps) with the
// heaviest set weight as TopWeight; durations hold a seconds count (repo convention).
// SessionActivityCount is how many activities the parent session has in total (not just the
// ones matching the current filter), so a browse card can say "2 activities" honestly.
// HR is intentionally omitted from the list (it lives in the stream blobs) — the detail
// page carries it. Shaped so precomputed Operation rollup columns can back this later
// without changing the JSON.
type ActivityFeedItem struct {
	OperationID     uuid.UUID  `json:"operation_id"`
	ExerciseID      uuid.UUID  `json:"exercise_id"`     // the session
	ExerciseDayID   uuid.UUID  `json:"exercise_day_id"` // the day (the /exercises/:id link)
	Date            time.Time  `json:"date"`            // day date
	Time            *time.Time `json:"time"`            // session start time (nullable)
	ActionID        *uuid.UUID `json:"action_id"`
	ActionName      string     `json:"action_name"`
	ActionType      string     `json:"action_type"`
	ActionHasLogo   bool       `json:"action_has_logo"`
	Note            *string    `json:"note"`
	DistanceUnit    string     `json:"distance_unit"`
	WeightUnit      string     `json:"weight_unit"`
	Distance        float64    `json:"distance"`
	DurationSeconds int64      `json:"duration_seconds"`
	MovingSeconds   int64      `json:"moving_seconds"` // summed moving time (excludes pauses); usually Strava-only
	Repetitions     float64    `json:"repetitions"`
	TopWeight       float64    `json:"top_weight"`
	SetCount        int        `json:"set_count"`
	HasStrava       bool       `json:"has_strava"`
	// Rollup scalars precomputed on the Operation from its Strava stream (nil without one), so
	// the list can show them without touching the stream blob. See models.ComputeStreamRollup.
	AvgHeartrate         *int     `json:"avg_heartrate"`
	MaxHeartrate         *int     `json:"max_heartrate"`
	AvgCadence           *int     `json:"avg_cadence"`
	TempC                *int     `json:"temp_c"`
	ElevationGainM       *float64 `json:"elevation_gain_m"`
	HevyWorkoutID        *string  `json:"hevy_workout_id"`    // session-level Hevy provenance; drives source resolution (strava/hevy/manual)
	CountsTowardGoal     bool     `json:"counts_toward_goal"` // session-level: false → shown but not tallied
	Private              bool     `json:"private"`            // session-level: true → hidden from everyone else's feeds
	SessionActivityCount int      `json:"session_activity_count"`
}

// ActivityFeedFilter is the parsed, validated query for GetActivityFeedForUser. Sort is one
// of date|distance|duration|weight|reps and Order is asc|desc (both validated by the
// controller). Limit/Offset drive pagination. A nil pointer means "no filter on that field".
type ActivityFeedFilter struct {
	ActionID    *uuid.UUID
	ActionName  string // case-insensitive substring on the action name; the MCP search filters by name (LLMs have names, not action ids). The web feed leaves this empty and filters by ActionID.
	Start       *time.Time
	End         *time.Time
	Query       string
	HasDistance bool
	Sort        string
	Order       string
	Limit       int
	Offset      int
}

// SessionFeedItem is one workout session (Exercise) in the /exercises timeline — the unit
// the feed pages, counts and sorts. It exists because the feed used to be rooted in
// Operation, which made a session with no activities invisible: the front page's "+" button
// (and the week calendar) create an Exercise with zero Operations, so the most common way to
// log a workout never reached the timeline at all. Rooting in the session fixes that and
// makes the counts honest ("showing 20 of 340" now means workouts).
//
// Metrics are summed across every enabled set of every enabled activity in the session, so
// they describe the whole workout regardless of which activity matched the filter.
// DurationSeconds cascades the session's own Duration over the summed set times (mirroring
// resolveSessionWindow), so a bare session that recorded only a duration still reads as one.
// Units are the session's dominant ones; a session that genuinely mixes them still shows the
// exact per-activity unit on each nested row.
//
// CountsTowardGoal and Private live here rather than on ActivityFeedItem because they are
// session-level facts — the old per-activity rendering repeated them on every row of a
// session. Activities holds every activity in the session (never only the matching ones), so
// a card describes the workout truthfully; ActivityCount is len(Activities).
type SessionFeedItem struct {
	ExerciseID       uuid.UUID  `json:"exercise_id"`
	ExerciseDayID    uuid.UUID  `json:"exercise_day_id"` // the /exercises/:id link target
	Date             time.Time  `json:"date"`
	Time             *time.Time `json:"time"` // session start (nullable — bare sessions may have none)
	Note             string     `json:"note"`
	Distance         float64    `json:"distance"`
	DistanceUnit     string     `json:"distance_unit"`
	WeightUnit       string     `json:"weight_unit"`
	DurationSeconds  int64      `json:"duration_seconds"`
	MovingSeconds    int64      `json:"moving_seconds"`
	Repetitions      float64    `json:"repetitions"`
	TopWeight        float64    `json:"top_weight"`
	SetCount         int        `json:"set_count"`
	HasStrava        bool       `json:"has_strava"`
	HevyWorkoutID    *string    `json:"hevy_workout_id"`
	CountsTowardGoal bool       `json:"counts_toward_goal"`
	Private          bool       `json:"private"`
	ActivityCount    int        `json:"activity_count"`
	// gorm:"-" — this is filled by a second query, not scanned from the session row; without
	// it GORM reads the slice as a relation and refuses the Scan.
	Activities []ActivityFeedItem `json:"activities" gorm:"-"`
}
