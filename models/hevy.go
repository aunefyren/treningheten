package models

import (
	"strings"
	"time"
)

// HevyActionType maps a Hevy template "type" onto Treningheten's action type
// vocabulary (lifting/timing/moving). Lives here (not in controllers) so both the
// live import path and the catalog seeder produce identical Actions.
func HevyActionType(hevyType string) string {
	switch strings.ToLower(strings.TrimSpace(hevyType)) {
	case "duration":
		return "timing"
	case "distance_duration", "weight_distance", "short_distance_weight":
		return "moving"
	default:
		// weight_reps, reps_only, bodyweight_reps, weighted_bodyweight,
		// assisted_bodyweight, and anything unrecognised.
		return "lifting"
	}
}

// ToAction builds a global Action from an official-catalog Hevy template. The caller
// assigns the ID (random in the live path, deterministic in the seeder) — everything
// else here matches what an on-demand import would create.
func (t HevyExerciseTemplate) ToAction() Action {
	templateID := strings.TrimSpace(t.ID)
	return Action{
		Enabled:        true,
		Name:           strings.TrimSpace(t.Title),
		NorwegianName:  strings.TrimSpace(t.Title),
		Type:           HevyActionType(t.Type),
		BodyPart:       strings.TrimSpace(t.PrimaryMuscleGroup),
		HevyTemplateID: &templateID,
		HasLogo:        false,
	}
}

// HevyUserInfo mirrors the Hevy API "UserInfo" object returned by GET /v1/user/info.
type HevyUserInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// HevyUserInfoResponse is the wrapper Hevy returns around the user info object.
type HevyUserInfoResponse struct {
	Data HevyUserInfo `json:"data"`
}

// HevyExerciseTemplate mirrors the Hevy API "ExerciseTemplate" object. IsCustom
// distinguishes Hevy's official catalog (false) from a user's private custom
// exercises (true) — only the former is imported into the global Action vocabulary.
type HevyExerciseTemplate struct {
	ID                    string   `json:"id"`
	Title                 string   `json:"title"`
	Type                  string   `json:"type"`
	PrimaryMuscleGroup    string   `json:"primary_muscle_group"`
	SecondaryMuscleGroups []string `json:"secondary_muscle_groups"`
	IsCustom              bool     `json:"is_custom"`
}

// HevyExerciseTemplatesResponse is one page of the GET /v1/exercise_templates result.
type HevyExerciseTemplatesResponse struct {
	Page              int                    `json:"page"`
	PageCount         int                    `json:"page_count"`
	ExerciseTemplates []HevyExerciseTemplate `json:"exercise_templates"`
}

// HevyWorkout mirrors the Hevy API "Workout" object (GET /v1/workouts).
type HevyWorkout struct {
	ID          string                `json:"id"`
	Title       string                `json:"title"`
	Description string                `json:"description"`
	StartTime   time.Time             `json:"start_time"`
	EndTime     time.Time             `json:"end_time"`
	UpdatedAt   time.Time             `json:"updated_at"`
	CreatedAt   time.Time             `json:"created_at"`
	Exercises   []HevyWorkoutExercise `json:"exercises"`
}

// HevyWorkoutExercise is one exercise within a workout. SupersetsID groups exercises
// performed as a superset (nil when not part of one).
type HevyWorkoutExercise struct {
	Index              int              `json:"index"`
	Title              string           `json:"title"`
	Notes              string           `json:"notes"`
	ExerciseTemplateID string           `json:"exercise_template_id"`
	SupersetsID        *int             `json:"supersets_id"`
	Sets               []HevyWorkoutSet `json:"sets"`
}

// HevyWorkoutSet is one set within an exercise. All measurement fields are pointers
// because Hevy omits/nulls those that don't apply to the exercise type.
type HevyWorkoutSet struct {
	Index           int      `json:"index"`
	Type            string   `json:"type"`
	WeightKg        *float64 `json:"weight_kg"`
	Reps            *float64 `json:"reps"`
	DistanceMeters  *float64 `json:"distance_meters"`
	DurationSeconds *float64 `json:"duration_seconds"`
	RPE             *float64 `json:"rpe"`
	CustomMetric    *float64 `json:"custom_metric"`
}

// HevyWorkoutsResponse is one page of the GET /v1/workouts result.
type HevyWorkoutsResponse struct {
	Page      int           `json:"page"`
	PageCount int           `json:"page_count"`
	Workouts  []HevyWorkout `json:"workouts"`
}

// HevyWorkoutEvent is one entry from GET /v1/workouts/events. Type is "updated" (Workout
// is populated) or "deleted" (ID + DeletedAt are populated).
type HevyWorkoutEvent struct {
	Type      string       `json:"type"`
	Workout   *HevyWorkout `json:"workout"`
	ID        string       `json:"id"`
	DeletedAt string       `json:"deleted_at"`
}

// HevyWorkoutEventsResponse is one page of the GET /v1/workouts/events result.
type HevyWorkoutEventsResponse struct {
	Page      int                `json:"page"`
	PageCount int                `json:"page_count"`
	Events    []HevyWorkoutEvent `json:"events"`
}
