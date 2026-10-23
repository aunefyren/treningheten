package models

import (
	"time"

	"github.com/google/uuid"
)

// ── Query rows ───────────────────────────────────────────────────────────────
// Flat rows the statistics queries scan into. All of them come from the user's visible
// workouts only: enabled day, enabled and switched-on workout (hidden duplicates and deleted
// workouts drop out), enabled activity and sets. See docs/statistics.md.

// StatisticsSessionRow is one workout with its duration (the workout's own duration, else
// its summed set times — the same cascade as the /exercises feed).
type StatisticsSessionRow struct {
	ExerciseID       uuid.UUID `json:"exercise_id"`
	Date             time.Time `json:"date"`
	CountsTowardGoal bool      `json:"counts_toward_goal"`
	DurationSeconds  int64     `json:"duration_seconds"`
}

// StatisticsActivityRow is one activity with its sets rolled up and its stream rollups.
type StatisticsActivityRow struct {
	OperationID       uuid.UUID  `json:"operation_id"`
	ExerciseID        uuid.UUID  `json:"exercise_id"`
	ExerciseDayID     uuid.UUID  `json:"exercise_day_id"`
	Date              time.Time  `json:"date"`
	Time              *time.Time `json:"time"`
	ActionID          *uuid.UUID `json:"action_id"`
	ActionName        string     `json:"action_name"`
	StravaName        string     `json:"strava_name"`
	Type              string     `json:"type"`
	DistanceUnit      string     `json:"distance_unit"`
	WeightUnit        string     `json:"weight_unit"`
	Distance          float64    `json:"distance"`
	SetSeconds        int64      `json:"set_seconds"`
	MovingSeconds     int64      `json:"moving_seconds"`
	Repetitions       float64    `json:"repetitions"`
	TopWeight         float64    `json:"top_weight"`
	SetCount          int        `json:"set_count"`
	OperationDuration *int64     `json:"operation_duration"`
	AvgHeartrate      *int       `json:"avg_heartrate"`
	MaxHeartrate      *int       `json:"max_heartrate"`
	AvgCadence        *int       `json:"avg_cadence"`
	ElevationGainM    *float64   `json:"elevation_gain_m"`
	AvgWatts          *int       `json:"avg_watts"`
}

// StatisticsSetRow is one weighted or counted set, for the strength records.
type StatisticsSetRow struct {
	OperationID   uuid.UUID  `json:"operation_id"`
	ExerciseDayID uuid.UUID  `json:"exercise_day_id"`
	Date          time.Time  `json:"date"`
	Time          *time.Time `json:"time"`
	ActionID      uuid.UUID  `json:"action_id"`
	Type          string     `json:"type"`
	WeightUnit    string     `json:"weight_unit"`
	Repetitions   float64    `json:"repetitions"`
	Weight        float64    `json:"weight"`
}

// BestEffortRow is a stored best effort joined to the workout it belongs to.
type BestEffortRow struct {
	OperationID    uuid.UUID  `json:"operation_id"`
	ExerciseDayID  uuid.UUID  `json:"exercise_day_id"`
	ActionID       uuid.UUID  `json:"action_id"`
	StravaName     string     `json:"strava_name"`
	DistanceM      int        `json:"distance_m"`
	ElapsedSeconds int        `json:"elapsed_seconds"`
	Date           time.Time  `json:"date"`
	Time           *time.Time `json:"time"`
}

// ── Shared pieces ────────────────────────────────────────────────────────────

// UnitAmount is an amount in one unit. Distances and weights are reported per unit and never
// summed across units (see docs/data-conventions.md).
type UnitAmount struct {
	Unit   string  `json:"unit"`
	Amount float64 `json:"amount"`
}

// ── Personal records ─────────────────────────────────────────────────────────

// PersonalRecordEntry is one point in a record's history: a value, when it was set and the
// day it was set on.
type PersonalRecordEntry struct {
	Value         float64   `json:"value"`
	Unit          string    `json:"unit"`
	Reps          *float64  `json:"reps,omitempty"`
	SetAt         time.Time `json:"set_at"`
	ExerciseDayID uuid.UUID `json:"exercise_day_id"`
	OperationID   uuid.UUID `json:"operation_id"`
}

// PersonalRecord is the current best for one record key, with the best it beat.
type PersonalRecord struct {
	Key          string               `json:"key"`
	Label        string               `json:"label"`
	Sport        string               `json:"sport,omitempty" jsonschema:"the activity type's Strava sport, so a best effort can be shown as pace (foot sports) or speed"`
	Kind         string               `json:"kind" jsonschema:"what the value measures: time (seconds, lower is better), distance, duration (seconds), elevation (metres), weight or reps"`
	Current      PersonalRecordEntry  `json:"current"`
	Previous     *PersonalRecordEntry `json:"previous"`
	Fresh        bool                 `json:"fresh"`
	Improvements int                  `json:"improvements"`
	// AllTimeRecord is true when this is the best ever; always so without a date range. With
	// one, Current is the best inside the range and AllTimeBest the record to compare it with.
	AllTimeRecord bool                 `json:"all_time_record"`
	AllTimeBest   *PersonalRecordEntry `json:"all_time_best,omitempty"`
}

// PersonalRecordGroup is every record for one activity type.
type PersonalRecordGroup struct {
	Action     Action           `json:"action"`
	Activities int              `json:"activities"`
	Records    []PersonalRecord `json:"records"`
}

// RecentPersonalRecord is a record set inside an overview period, with its activity type.
type RecentPersonalRecord struct {
	ActionID   uuid.UUID      `json:"action_id"`
	ActionName string         `json:"action_name"`
	Record     PersonalRecord `json:"record"`
}

// ── Overview ─────────────────────────────────────────────────────────────────

// StatisticsTotals sums a period.
type StatisticsTotals struct {
	Workouts        int          `json:"workouts"`
	DurationSeconds int64        `json:"duration_seconds"`
	ActiveDays      int          `json:"active_days"`
	Distances       []UnitAmount `json:"distances"`
}

// StatisticsBucket is one week (or month) of a period.
type StatisticsBucket struct {
	Start           time.Time    `json:"start"`
	Workouts        int          `json:"workouts"`
	DurationSeconds int64        `json:"duration_seconds"`
	Distances       []UnitAmount `json:"distances"`
}

// StatisticsMixItem is one activity type's share of a period.
type StatisticsMixItem struct {
	ActionID        uuid.UUID `json:"action_id"`
	Name            string    `json:"name"`
	Activities      int       `json:"activities"`
	DurationSeconds int64     `json:"duration_seconds"`
}

// StatisticsStreaks are the personal streaks (see docs/streaks.md).
type StatisticsStreaks struct {
	WeekCurrent int `json:"week_current"`
	WeekBest    int `json:"week_best"`
	DayCurrent  int `json:"day_current"`
	DayBest     int `json:"day_best"`
}

// StatisticsOverview is the Overview tab of /statistics for one period.
type StatisticsOverview struct {
	Period        string                 `json:"period"`
	Start         time.Time              `json:"start"`
	End           time.Time              `json:"end"`
	Bucket        string                 `json:"bucket"`
	Totals        StatisticsTotals       `json:"totals"`
	Previous      *StatisticsTotals      `json:"previous"`
	Buckets       []StatisticsBucket     `json:"buckets"`
	Mix           []StatisticsMixItem    `json:"mix"`
	Streaks       StatisticsStreaks      `json:"streaks"`
	RecentRecords []RecentPersonalRecord `json:"recent_records"`
}

// ── Activity type statistics ─────────────────────────────────────────────────

// StatisticsBest is the best single activity or set for one measure in a period.
type StatisticsBest struct {
	Value         float64   `json:"value"`
	Unit          string    `json:"unit"`
	Reps          *float64  `json:"reps,omitempty"`
	Date          time.Time `json:"date"`
	ExerciseDayID uuid.UUID `json:"exercise_day_id"`
	OperationID   uuid.UUID `json:"operation_id"`
}

type ActionStatisticsTotals struct {
	Distances       []UnitAmount `json:"distances"`
	DurationSeconds int64        `json:"duration_seconds"`
	MovingSeconds   int64        `json:"moving_seconds"`
	Repetitions     float64      `json:"repetitions"`
	Volume          []UnitAmount `json:"volume" jsonschema:"weight × reps over every set, per weight unit"`
}

// ActionStatisticsAverages are per-activity means; a field is nil when no activity in the
// period has that measure.
type ActionStatisticsAverages struct {
	Distance        *UnitAmount `json:"distance"`
	DurationSeconds *int64      `json:"duration_seconds"`
	Repetitions     *float64    `json:"repetitions"`
	Volume          *UnitAmount `json:"volume"`
}

type ActionStatisticsBests struct {
	LongestDistance *StatisticsBest `json:"longest_distance"`
	LongestDuration *StatisticsBest `json:"longest_duration"`
	MostReps        *StatisticsBest `json:"most_reps"`
	HeaviestSet     *StatisticsBest `json:"heaviest_set"`
}

// ActionStatisticsSensors come from the per-activity stream rollups: heart rate, cadence and
// power are averaged weighted by each activity's duration, elevation gain is summed.
type ActionStatisticsSensors struct {
	AvgHeartrate   *int     `json:"avg_heartrate"`
	MaxHeartrate   *int     `json:"max_heartrate"`
	AvgCadence     *int     `json:"avg_cadence"`
	CadenceUnit    string   `json:"cadence_unit,omitempty"`
	AvgWatts       *int     `json:"avg_watts"`
	ElevationGainM *float64 `json:"elevation_gain_m"`
}

// ActionTrendPoint is one activity in the trend chart.
type ActionTrendPoint struct {
	Date               time.Time `json:"date"`
	ExerciseDayID      uuid.UUID `json:"exercise_day_id"`
	OperationID        uuid.UUID `json:"operation_id"`
	Distance           float64   `json:"distance"`
	DistanceUnit       string    `json:"distance_unit"`
	DurationSeconds    int64     `json:"duration_seconds"`
	MovingSeconds      int64     `json:"moving_seconds"`
	AvgHeartrate       *int      `json:"avg_heartrate"`
	Repetitions        float64   `json:"repetitions"`
	TopWeight          float64   `json:"top_weight"`
	WeightUnit         string    `json:"weight_unit"`
	EstimatedOneRepMax float64   `json:"estimated_one_rep_max"`
}

type ActionStatistics struct {
	Action     Action                   `json:"action"`
	Activities int                      `json:"activities"`
	Workouts   int                      `json:"workouts"`
	Totals     ActionStatisticsTotals   `json:"totals"`
	Averages   ActionStatisticsAverages `json:"averages"`
	Bests      ActionStatisticsBests    `json:"bests"`
	Sensors    ActionStatisticsSensors  `json:"sensors"`
	Trend      []ActionTrendPoint       `json:"trend"`
	Media      *ActionMediaStatistics   `json:"media"`
}
