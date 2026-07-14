package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	GormModel
	FirstName                  string     `json:"first_name" gorm:"not null"`
	LastName                   string     `json:"last_name" gorm:"not null"`
	Email                      string     `json:"email" gorm:"unique; not null"`
	Password                   string     `json:"password" gorm:"not null"`
	Admin                      *bool      `json:"admin" gorm:"not null; default: false"`
	Enabled                    bool       `json:"enabled" gorm:"not null; default: false"`
	Verified                   bool       `json:"verified" gorm:"not null; default: false"`
	VerificationCode           *string    `json:"verification_code"`
	VerificationCodeExpiration *time.Time `json:"verification_code_expiration"`
	ResetCode                  *string    `json:"reset_code"`
	ResetExpiration            *time.Time `json:"reset_expiration"`
	SundayAlert                bool       `json:"sunday_alert" gorm:"not null; default: false"`
	BirthDate                  *time.Time `json:"birth_date" gorm:"default: null"`
	StravaCode                 *string    `json:"strava_code" gorm:"default: null"`
	StravaIgnoreWalks          *bool      `json:"strava_walks" gorm:"column:strava_walks;default: true"`
	StravaID                   *string    `json:"strava_id" gorm:"default: null"`
	StravaPublic               *bool      `json:"strava_public" gorm:"default: true"`
	StravaSkipHevyDuplicates   *bool      `json:"strava_skip_hevy" gorm:"default: false"`
	HevyAPIKey                 *string    `json:"-" gorm:"default: null"`
	HevyLastSync               *time.Time `json:"-" gorm:"default: null"`
	HevyProfileURL             *string    `json:"hevy_profile_url" gorm:"default: null"`
	HevyPublic                 *bool      `json:"hevy_public" gorm:"default: true"`
	HevyConnected              bool       `json:"hevy_connected" gorm:"-"`
	WheelColor                 *string    `json:"wheel_color" gorm:"default: null"`
	WheelBorderColor           *string    `json:"wheel_border_color" gorm:"default: null"`
	WheelEmoji                 *string    `json:"wheel_emoji" gorm:"default: null"`
	ShareActivities            *bool      `json:"share_activities" gorm:"default: true"`
	ShareStatistics            *bool      `json:"share_statistics" gorm:"default: true"`
}

type UserCreationRequest struct {
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Email          string `json:"email"`
	Password       string `json:"password"`
	PasswordRepeat string `json:"password_repeat"`
	InviteCode     string `json:"invite_code"`
}

type UserUpdateRequest struct {
	Email           string     `json:"email"`
	Password        string     `json:"password"`
	PasswordRepeat  string     `json:"password_repeat"`
	ProfileImage    string     `json:"profile_image"`
	OldPassword     string     `json:"password_old"`
	BirthDate       *time.Time `json:"birth_date"`
	ShareActivities *bool      `json:"share_activities"`
	ShareStatistics *bool      `json:"share_statistics"`
}

type UserPartialUpdateRequest struct {
	SundayAlert              *bool   `json:"sunday_alert"`
	StravaIgnoreWalks        *bool   `json:"strava_walks"`
	StravaPublic             *bool   `json:"strava_public"`
	StravaSkipHevyDuplicates *bool   `json:"strava_skip_hevy"`
	HevyPublic               *bool   `json:"hevy_public"`
	WheelColor               *string `json:"wheel_color"`
	WheelBorderColor         *string `json:"wheel_border_color"`
	WheelEmoji               *string `json:"wheel_emoji"`
}

type UserUpdatePasswordRequest struct {
	ResetCode      string `json:"reset_code"`
	Password       string `json:"password"`
	PasswordRepeat string `json:"password_repeat"`
}

type UserStravaCodeUpdateRequest struct {
	StravaCode string `json:"strava_code"`
}

type UserHevyAPIKeyUpdateRequest struct {
	HevyAPIKey string `json:"hevy_api_key"`
}

type UserWithTickets struct {
	User    User `json:"user"`
	Tickets int  `json:"tickets"`
}

func (user *User) HashPassword(password string) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return err
	}
	user.Password = string(bytes)
	return nil
}

func (user *User) CheckPassword(providedPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(providedPassword))
	if err != nil {
		return err
	}
	return nil
}

type UserStatisticsReply struct {
	ExercisesAllTime   int `json:"exercises_all_time"`
	ExercisesPastYear  int `json:"exercises_past_year"`
	ExercisesPastMonth int `json:"exercises_past_month"`
	StreakWeeks        int `json:"streak_weeks"`
	StreakWeeksTop     int `json:"streak_weeks_top"`
	StreakDays         int `json:"streak_days"`
	StreakDaysTop      int `json:"streak_days_top"`
	SeasonsJoined      int `json:"seasons_joined"`
	ActivityStatistics struct {
		Action    *Action                   `json:"action"`
		PastMonth UserStatisticsCompilation `json:"past_month"`
		PastYear  UserStatisticsCompilation `json:"past_year"`
		AllTime   UserStatisticsCompilation `json:"all_time"`
	} `json:"activity_statistics"`
}

type UserStatisticsCompilation struct {
	Sums     UserStatisticsSumCompilation     `json:"sums"`
	Averages UserStatisticsAverageCompilation `json:"averages"`
	Tops     UserStatisticsTopCompilation     `json:"tops"`
}

type UserStatisticsTopCompilation struct {
	Distance              float64    `json:"distance"`
	DistanceExerciseDayID *uuid.UUID `json:"distance_exercise_day_id"`
	Time                  int64      `json:"time"`
	TimeExerciseDayID     *uuid.UUID `json:"time_exercise_day_id"`
	Weight                float64    `json:"weight"`
	WeightExerciseDayID   *uuid.UUID `json:"weight_exercise_day_id"`
}

type UserStatisticsSumCompilation struct {
	Distance   float64 `json:"distance"`
	Time       int64   `json:"time"`
	Weight     float64 `json:"weight"`
	Operations int64   `json:"operations"`
}

type UserStatisticsAverageCompilation struct {
	Distance float64 `json:"distance"`
	Time     int64   `json:"time"`
	Weight   float64 `json:"weight"`
}
