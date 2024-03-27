package models

import (
	"time"

	"github.com/google/uuid"
)

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
}

type SeasonCreationRequest struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Prize       uuid.UUID `json:"prize_id"`
	Sickleave   int       `json:"sickleave"`
	TimeZone    string    `json:"timezone"`
	JoinAnytime bool      `json:"join_anytime"`
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
}

type SeasonLeaderboard struct {
	UserGoal    GoalObject    `json:"goal"`
	Season      SeasonObject  `json:"season"`
	PastWeeks   []WeekResults `json:"past_weeks"`
	CurrentWeek WeekResults   `json:"this_week"`
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
}

type UserStreak struct {
	UserID uuid.UUID `json:"user_id"`
	Streak int       `json:"streak"`
}
