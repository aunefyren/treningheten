package models

import (
	"time"

	"github.com/google/uuid"
)

// SeasonListItem is a lightweight season view for the potential/countdown lists, which
// only need identity, dates, participant count and the caller's own goal — not the full,
// per-goal-resolved SeasonObject.
type SeasonListItem struct {
	ID               uuid.UUID  `json:"id"`
	Name             string     `json:"name"`
	Start            time.Time  `json:"start"`
	End              time.Time  `json:"end"`
	JoinAnytime      *bool      `json:"join_anytime"`
	ParticipantCount int        `json:"participant_count"`
	UserGoalID       *uuid.UUID `json:"user_goal_id"`
}

type Season struct {
	GormModel
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description"`
	Start       time.Time `json:"start" gorm:"not null"`
	End         time.Time `json:"end" gorm:"not null"`
	PrizeID     uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Prize       Prize     `json:"prize"`
	Sickleave   int       `json:"sickleave"`
	JoinAnytime *bool     `json:"join_anytime" gorm:"not null; default: false"`
	Enabled     bool      `json:"enabled" gorm:"not null; default: true"`
	// Saved workouts (docs/flexible-workouts.md). CarryOverMax is the most a participant
	// can hold at once; 0 turns the feature off, which is the default and what every season
	// created before it has. CarryOverWeeks is how many weeks a saved workout lasts (1 = only
	// the week after it was earned); 0 while the feature is off.
	CarryOverMax   int `json:"carry_over_max" gorm:"not null; default: 0"`
	CarryOverWeeks int `json:"carry_over_weeks" gorm:"not null; default: 0"`
}

type SeasonCreationRequest struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Prize       uuid.UUID `json:"prize_id"`
	Sickleave   int       `json:"sickleave"`
	// Deprecated: ignored. Season boundaries are always built in the server's configured
	// zone (see seasonBoundariesInServerZone); the field is still accepted so older clients
	// keep working.
	TimeZone    string `json:"timezone"`
	JoinAnytime bool   `json:"join_anytime"`
	// Saved workouts: both optional, and leaving them out keeps the season's feature off.
	CarryOverMax   int `json:"carry_over_max"`
	CarryOverWeeks int `json:"carry_over_weeks"`
}

type SeasonObject struct {
	GormModel
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Start       time.Time    `json:"start"`
	End         time.Time    `json:"end"`
	Enabled     bool         `json:"enabled"`
	Goals       []GoalObject `json:"goals"`
	Prize       Prize        `json:"prize"`
	Sickleave   int          `json:"sickleave"`
	JoinAnytime *bool        `json:"join_anytime"`
	// Saved workouts rule, copied from Season (0 / 0 = off).
	CarryOverMax   int `json:"carry_over_max"`
	CarryOverWeeks int `json:"carry_over_weeks"`
}

type SeasonLeaderboard struct {
	UserGoal    *GoalObject   `json:"goal"`
	Season      SeasonObject  `json:"season"`
	PastWeeks   []WeekResults `json:"past_weeks"`
	CurrentWeek *WeekResults  `json:"this_week"`
}

type WeekResults struct {
	WeekNumber      int               `json:"week_number"`
	WeekYear        int               `json:"week_year"`
	WeekDate        time.Time         `json:"week_date"`
	UserWeekResults []UserWeekResults `json:"users"`
}

type WeekResultsPersonal struct {
	WeekNumber      int                    `json:"week_number"`
	WeekYear        int                    `json:"week_year"`
	WeekDate        time.Time              `json:"week_date"`
	UserWeekResults UserWeekResultPersonal `json:"user"`
}

type UserWeekResults struct {
	WeekCompletion        float64     `json:"week_completion"`
	CurrentStreak         int         `json:"current_streak"`
	UserID                uuid.UUID   `json:"user_id"`
	SickLeave             bool        `json:"sick_leave"`
	Competing             bool        `json:"competing"`
	Debt                  *DebtObject `json:"debt"`
	GoalID                uuid.UUID   `json:"goal_id"`
	FullWeekParticipation bool        `json:"full_week_participation"`
	// Saved workouts (docs/flexible-workouts.md), worked out by nextCarryBank in the season
	// walk. WeekCompletion stays the raw workouts / goal; GoalMet adds CarryUsed.
	// CarriedIn is how many saved workouts were available entering the week, and
	// CarryExpiries the use-by day (the Sunday of the last week it can be used) of each of
	// them, oldest first. CarryOptIn says the participant chose "Use saved workouts" for the
	// week; CarryUsed is how many the week spent (all or nothing: only when they covered the
	// whole shortfall) and CarryEarned how many it saved. For the running week these are
	// what would happen if the week ended now.
	CarriedIn     int         `json:"carried_in"`
	CarryExpiries []time.Time `json:"carry_expiries"`
	CarryOptIn    bool        `json:"carry_opt_in"`
	CarryUsed     int         `json:"carry_used"`
	CarryEarned   int         `json:"carry_earned"`
}

type UserWeekResultPersonal struct {
	WeekCompletionInterval int         `json:"week_completion_interval"`
	ExerciseGoal           int         `json:"exercise_goal"`
	CurrentStreak          int         `json:"current_streak"`
	UserID                 uuid.UUID   `json:"user_id"`
	GoalID                 uuid.UUID   `json:"goal_id"`
	SickLeave              bool        `json:"sick_leave"`
	Competing              bool        `json:"competing"`
	Debt                   *DebtObject `json:"debt"`
	FullWeekParticipation  bool        `json:"full_week_participation"`
	CarryUsed              int         `json:"carry_used"`
	GoalMet                bool        `json:"goal_met"`
}

type UserStreak struct {
	UserID uuid.UUID `json:"user_id"`
	Streak int       `json:"streak"`
}
