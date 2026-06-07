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

type MCPActivitySet struct {
	Repetitions  *float64 `json:"repetitions,omitempty"`
	Weight       *float64 `json:"weight,omitempty"`
	WeightUnit   string   `json:"weight_unit,omitempty"`
	Distance     *float64 `json:"distance,omitempty"`
	DistanceUnit string   `json:"distance_unit,omitempty"`
	TimeSeconds  *int64   `json:"time_seconds,omitempty"`
}

type MCPActivity struct {
	Date            time.Time        `json:"date"`
	Action          string           `json:"action" jsonschema:"the exercise type, e.g. Run, Bicycling, Weight Training"`
	Type            string           `json:"type" jsonschema:"moving, timing or lifting"`
	Note            string           `json:"note,omitempty"`
	DurationSeconds *int64           `json:"duration_seconds,omitempty"`
	Sets            []MCPActivitySet `json:"sets,omitempty"`
}

type MCPStatistics struct {
	ActivitiesAllTime   int     `json:"activities_all_time"`
	ActivitiesPastYear  int     `json:"activities_past_year"`
	ActivitiesPastMonth int     `json:"activities_past_month"`
	TotalDistance       float64 `json:"total_distance" jsonschema:"sum of all logged distances (mixed units, mostly km)"`
	TotalTimeSeconds    int64   `json:"total_time_seconds" jsonschema:"sum of all logged activity time in seconds"`
}
