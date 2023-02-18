package models

import (
	"time"

	"gorm.io/gorm"
)

type Season struct {
	gorm.Model
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description"`
	Start       time.Time `json:"start" gorm:"not null"`
	End         time.Time `json:"end" gorm:"not null"`
	Enabled     bool      `json:"enabled" gorm:"not null;default: true"`
}

type SeasonCreationRequest struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
}

type SeasonObject struct {
	gorm.Model
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Start       time.Time    `json:"start"`
	End         time.Time    `json:"end"`
	Enabled     bool         `json:"enabled"`
	Goals       []GoalObject `json:"goals"`
}

type SeasonLeaderboard struct {
	CurrentStreak     int           `json:"current_streak"`
	CurrentCompletion float64       `json:"current_completion"`
	UserGoal          GoalObject    `json:"goal"`
	Season            SeasonObject  `json:"season"`
	Weeks             []WeekResults `json:"weeks"`
}

type WeekResults struct {
	WeekNumber      int               `json:"week_number"`
	WeekYear        int               `json:"week_year"`
	UserWeekResults []UserWeekResults `json:"users"`
}

type UserWeekResults struct {
	WeekCompletion float64 `json:"week_completion"`
	CurrentStreak  int     `json:"current_streak"`
	User           User    `json:"user"`
}
