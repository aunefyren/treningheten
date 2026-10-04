package models

import (
	"time"

	"github.com/google/uuid"

	"golang.org/x/crypto/bcrypt"
)

// User is both the GORM row and the shape returned by the user endpoints. The credential
// and recovery fields are `json:"-"` on purpose: a bcrypt hash, a live password-reset code
// or an OAuth credential must never reach a response body, and relying on CensorUserObject
// alone proved too easy to get wrong (see docs/conventions.md, "Never serialize a
// credential"). Connection state is surfaced through the derived HevyConnected /
// StravaConnected booleans instead.
type User struct {
	GormModel
	FirstName                  string     `json:"first_name" gorm:"not null"`
	LastName                   string     `json:"last_name" gorm:"not null"`
	Email                      string     `json:"email" gorm:"unique; not null"`
	Password                   string     `json:"-" gorm:"not null"`
	Admin                      *bool      `json:"admin" gorm:"not null; default: false"`
	Enabled                    bool       `json:"enabled" gorm:"not null; default: false"`
	Verified                   bool       `json:"verified" gorm:"not null; default: false"`
	VerificationCode           *string    `json:"-"`
	VerificationCodeExpiration *time.Time `json:"-"`
	ResetCode                  *string    `json:"-"`
	ResetExpiration            *time.Time `json:"-"`
	SundayAlert                bool       `json:"sunday_alert" gorm:"not null; default: false"`
	BirthDate                  *time.Time `json:"birth_date" gorm:"default: null"`
	MaxHeartrate               *int       `json:"max_heartrate" gorm:"default: null"`
	RestingHeartrate           *int       `json:"resting_heartrate" gorm:"default: null"`
	// ObservedMaxHeartrate is the highest heart rate seen across the user's imported
	// activities — maintained on Strava sync and used to anchor HR zones when no explicit
	// max is set. System-derived (not user-editable); NULL means "not yet computed" (a
	// legacy row the backfill still owes), which is why new users start at 0.
	ObservedMaxHeartrate *int    `json:"observed_max_heartrate" gorm:"default: null"`
	StravaCode           *string `json:"-" gorm:"default: null"`
	// Deprecated: superseded by UserActivityGoalSetting (Walking → doesn't count). No longer
	// read on import; the Migrate() backfill converts any lingering true value to a goal
	// setting and clears it. Default is now false so new users don't re-trigger that migration.
	StravaIgnoreWalks        *bool      `json:"strava_walks" gorm:"column:strava_walks;default: false"`
	StravaID                 *string    `json:"strava_id" gorm:"default: null"`
	StravaPublic             *bool      `json:"strava_public" gorm:"default: true"`
	StravaSkipHevyDuplicates *bool      `json:"strava_skip_hevy" gorm:"default: false"`
	HevyAPIKey               *string    `json:"-" gorm:"default: null"`
	HevyLastSync             *time.Time `json:"-" gorm:"default: null"`
	HevyProfileURL           *string    `json:"hevy_profile_url" gorm:"default: null"`
	HevyPublic               *bool      `json:"hevy_public" gorm:"default: true"`
	HevyConnected            bool       `json:"hevy_connected" gorm:"-"`
	StravaConnected          bool       `json:"strava_connected" gorm:"-"`
	// IntegrationHealth is set only when a user reads their own account: the health of
	// their Strava / Hevy connections, keyed by provider. See docs/integration-health.md.
	IntegrationHealth map[string]IntegrationHealthObject `json:"integration_health,omitempty" gorm:"-"`
	WheelColor        *string                            `json:"wheel_color" gorm:"default: null"`
	WheelBorderColor  *string                            `json:"wheel_border_color" gorm:"default: null"`
	WheelEmoji        *string                            `json:"wheel_emoji" gorm:"default: null"`
	ShareActivities   *bool                              `json:"share_activities" gorm:"default: true"`
	ShareStatistics   *bool                              `json:"share_statistics" gorm:"default: true"`
}

// PublicUser is what one user is allowed to see of another. It exists as a distinct type
// so the safe set is an allowlist the compiler enforces: a field added to User is invisible
// to every cross-user response until somebody adds it here on purpose.
//
// That matters because the previous design — censor named fields, return a User — is a
// blocklist, and it had already failed silently. BirthDate, MaxHeartrate, RestingHeartrate
// and ObservedMaxHeartrate were added to User long after CensorUserObject was written, so
// GET /api/auth/users served every user's date of birth and resting heart rate to any
// authenticated caller, a read-only PAT included. Nothing rendered them; they inherited
// serialization for free. `json:"-"` is not the answer for these — unlike a credential they
// are legitimately visible to their owner on /account — so the fix is the type, not the tag.
//
// The JSON tags match User's exactly, so this is invisible to clients.
//
// Naming: deliberately not "UserObject". In this codebase the *Object suffix means an
// enriched read model (ConvertExerciseToExerciseObject resolves relations and does real
// work). This is the opposite — a reduction — and should not borrow that word.
//
// See docs/wip.md (S15) and docs/conventions.md.
type PublicUser struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`

	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Admin     *bool  `json:"admin"`

	// Profile links, each shown only when its owner opted in. StravaID is cleared
	// alongside StravaPublic by CensorUserObject, so a private Strava id is not merely
	// unrendered but absent.
	StravaID       *string `json:"strava_id"`
	StravaPublic   *bool   `json:"strava_public"`
	HevyProfileURL *string `json:"hevy_profile_url"`
	HevyPublic     *bool   `json:"hevy_public"`

	// ShareStatistics is a visibility flag rather than personal data: the profile page
	// reads it to decide whether to request the statistics block at all.
	ShareStatistics *bool `json:"share_statistics"`

	// Wheel appearance is rendered for other users on the prize wheel.
	WheelColor       *string `json:"wheel_color"`
	WheelBorderColor *string `json:"wheel_border_color"`
	WheelEmoji       *string `json:"wheel_emoji"`
}

// AdminUser is what an admin sees of a user on the admin page: identity plus the account
// flags they manage. Like PublicUser it is an allowlist, so credentials and personal
// health data stay out of the admin list too.
type AdminUser struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Admin     *bool     `json:"admin"`
	Enabled   bool      `json:"enabled"`
	Verified  bool      `json:"verified"`
}

// UserEnabledRequest toggles a user's access from the admin page. A pointer, so a missing
// field is rejected rather than read as "disable".
type UserEnabledRequest struct {
	Enabled *bool `json:"enabled"`
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
	Email           string `json:"email"`
	Password        string `json:"password"`
	PasswordRepeat  string `json:"password_repeat"`
	ProfileImage    string `json:"profile_image"`
	OldPassword     string `json:"password_old"`
	ShareActivities *bool  `json:"share_activities"`
	ShareStatistics *bool  `json:"share_statistics"`
}

// UserTrainingProfileRequest replaces the settings that shape how a user's workouts are
// read (age and heart-rate zone anchors). Every field is written as sent, so a null
// clears it. Not password-gated: none of it is a credential.
type UserTrainingProfileRequest struct {
	BirthDate        *time.Time `json:"birth_date"`
	MaxHeartrate     *int       `json:"max_heartrate"`
	RestingHeartrate *int       `json:"resting_heartrate"`
}

type UserPartialUpdateRequest struct {
	SundayAlert              *bool   `json:"sunday_alert"`
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
	User    PublicUser `json:"user"`
	Tickets int        `json:"tickets"`
}

// PasswordHashCost is the bcrypt cost for account passwords (~1s of a core at 14, see
// docs/security.md). A var solely so tests can lower it — never reassign it at runtime.
var PasswordHashCost = 14

func (user *User) HashPassword(password string) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), PasswordHashCost)
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
	// MinimumSampleSize is the floor below which a derived figure is withheld — the
	// averages, and the headline activity the breakdown is built around. It is sent so the
	// client can say *why* something is missing without hardcoding the number and drifting
	// from it. See controllers.userStatisticsMinSampleSize.
	MinimumSampleSize int `json:"minimum_sample_size"`
	// Each window is nil when it holds fewer than MinimumSampleSize sessions of the headline
	// activity — see controllers.publishWindow. Action is nil (and every window with it)
	// when there is no activity to headline at all.
	ActivityStatistics struct {
		Action    *Action                    `json:"action"`
		PastMonth *UserStatisticsCompilation `json:"past_month"`
		PastYear  *UserStatisticsCompilation `json:"past_year"`
		AllTime   *UserStatisticsCompilation `json:"all_time"`
	} `json:"activity_statistics"`
}

type UserStatisticsCompilation struct {
	Sums     UserStatisticsSumCompilation     `json:"sums"`
	Averages UserStatisticsAverageCompilation `json:"averages"`
	Tops     UserStatisticsTopCompilation     `json:"tops"`
}

// UserStatisticsTopCompilation is the best single session in a window. It is deliberately
// value-only: the exercise-day ids this used to carry named a specific session, with its
// date, on a profile any authenticated user can read — which is precisely what the Private
// flag withholds from the feeds. Nothing consumed them either (the web client never rendered
// a link, and MCP get_statistics has its own DTO), and the id only ever resolved for the
// owner, so they are gone rather than gated. See docs/wip.md, S14.
type UserStatisticsTopCompilation struct {
	Distance float64 `json:"distance"`
	Time     int64   `json:"time"`
	Weight   float64 `json:"weight"`
}

type UserStatisticsSumCompilation struct {
	Distance   float64 `json:"distance"`
	Time       int64   `json:"time"`
	Weight     float64 `json:"weight"`
	Operations int64   `json:"operations"`
}

// UserStatisticsAverageCompilation holds a window's per-session averages. The fields are
// pointers because an average over a handful of sessions approaches the sessions themselves
// — "1 operation, avg distance 21 km" simply republishes that one workout — so a window
// below userStatisticsMinSampleSize reports null rather than a number that is really an
// individual. Null means "not enough data", never zero.
type UserStatisticsAverageCompilation struct {
	Distance *float64 `json:"distance"`
	Time     *int64   `json:"time"`
	Weight   *float64 `json:"weight"`
}
