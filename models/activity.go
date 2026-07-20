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
	OperationID          uuid.UUID  `json:"operation_id"`
	ExerciseID           uuid.UUID  `json:"exercise_id"`     // the session
	ExerciseDayID        uuid.UUID  `json:"exercise_day_id"` // the day (the /exercises/:id link)
	Date                 time.Time  `json:"date"`            // day date
	Time                 *time.Time `json:"time"`            // session start time (nullable)
	ActionID             *uuid.UUID `json:"action_id"`
	ActionName           string     `json:"action_name"`
	ActionType           string     `json:"action_type"`
	ActionHasLogo        bool       `json:"action_has_logo"`
	Note                 *string    `json:"note"`
	DistanceUnit         string     `json:"distance_unit"`
	WeightUnit           string     `json:"weight_unit"`
	Distance             float64    `json:"distance"`
	DurationSeconds      int64      `json:"duration_seconds"`
	Repetitions          float64    `json:"repetitions"`
	TopWeight            float64    `json:"top_weight"`
	SetCount             int        `json:"set_count"`
	HasStrava            bool       `json:"has_strava"`
	HevyWorkoutID        *string    `json:"hevy_workout_id"`    // session-level Hevy provenance; drives source resolution (strava/hevy/manual)
	CountsTowardGoal     bool       `json:"counts_toward_goal"` // session-level: false → shown but not tallied
	SessionActivityCount int        `json:"session_activity_count"`
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
