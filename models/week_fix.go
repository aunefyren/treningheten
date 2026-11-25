package models

import (
	"time"

	"github.com/google/uuid"
)

// The admin "Fix a week" tool (docs/admin-week-fix.md): request bodies and the report a week
// reconciliation returns.

// WeekReconcileRequest is the body of POST /api/admin/weeks/reconcile. Date is any day of the
// week, as YYYY-MM-DD in the server's time zone.
type WeekReconcileRequest struct {
	Date   string `json:"date" sanitize:"trim"`
	DryRun bool   `json:"dry_run"`
	Notify bool   `json:"notify"`
}

// AdminExerciseCreationRequest is the body of POST /api/admin/users/:user_id/exercises: one
// workout added to the user's day. Date is YYYY-MM-DD in the server's time zone.
type AdminExerciseCreationRequest struct {
	Date string `json:"date" sanitize:"trim"`
	Note string `json:"note" sanitize:"trim,collapse"`
}

// AdminSickleaveGrantRequest is the body of POST /api/admin/users/:user_id/sickleave: Count
// extra weeks of sick leave on the user's goal in the season.
type AdminSickleaveGrantRequest struct {
	SeasonID uuid.UUID `json:"season_id"`
	Count    int       `json:"count"`
}

// WeekReconcileReport is what reconciling one season-week found and did (or, in a dry run,
// would do). Changed is false when the stored state already matched the week's results.
type WeekReconcileReport struct {
	SeasonID   uuid.UUID           `json:"season_id"`
	SeasonName string              `json:"season_name"`
	WeekYear   int                 `json:"week_year"`
	WeekNumber int                 `json:"week_number"`
	WeekStart  time.Time           `json:"week_start"`
	WeekEnd    time.Time           `json:"week_end"`
	DryRun     bool                `json:"dry_run"`
	Changed    bool                `json:"changed"`
	Winners    int                 `json:"winners"`
	Losers     int                 `json:"losers"`
	Users      []WeekReconcileUser `json:"users"`
	Warnings   []string            `json:"warnings"`
}

// WeekReconcileUser is one participant's week and what reconciling it changed.
type WeekReconcileUser struct {
	User                  PublicUser                 `json:"user"`
	GoalID                uuid.UUID                  `json:"goal_id"`
	Goal                  int                        `json:"goal"`
	Workouts              int                        `json:"workouts"`
	WeekCompletion        float64                    `json:"week_completion"`
	Competing             bool                       `json:"competing"`
	SickLeave             bool                       `json:"sick_leave"`
	FullWeekParticipation bool                       `json:"full_week_participation"`
	CarryUsed             int                        `json:"carry_used"`
	GoalMet               bool                       `json:"goal_met"`
	WonWeek               bool                       `json:"won_week"`
	OwesSpin              bool                       `json:"owes_spin"`
	SickleaveLeft         int                        `json:"sickleave_left"`
	DebtBefore            *WeekReconcileDebt         `json:"debt_before"`
	DebtAfter             *WeekReconcileDebt         `json:"debt_after"`
	Changes               []string                   `json:"changes"`
	AchievementsGranted   []WeekReconcileAchievement `json:"achievements_granted"`
	AchievementsRevoked   []WeekReconcileAchievement `json:"achievements_revoked"`
	Warnings              []string                   `json:"warnings"`
}

// WeekReconcileDebt is a debt as the report shows it. A debt is Settled once it is paid or its
// wheel has been spun (Spun); reconciliation never changes a settled debt. ID is nil for a
// debt a dry run would create.
type WeekReconcileDebt struct {
	ID      *uuid.UUID  `json:"id"`
	Winner  *PublicUser `json:"winner"`
	Paid    bool        `json:"paid"`
	Spun    bool        `json:"spun"`
	Settled bool        `json:"settled"`
}

// WeekReconcileAchievement is an achievement reconciliation granted or revoked.
type WeekReconcileAchievement struct {
	AchievementID uuid.UUID `json:"achievement_id"`
	Name          string    `json:"name"`
	GivenAt       time.Time `json:"given_at"`
}
