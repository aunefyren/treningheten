package models

import (
	"time"

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
	StravaPadel                *bool      `json:"strava_padel" gorm:"default: false"`
	StravaWalks                *bool      `json:"strava_walks" gorm:"default: true"`
	StravaID                   *string    `json:"strava_id" gorm:"default: null"`
	StravaPublic               *bool      `json:"strava_public" gorm:"default: true"`
	ShareActivities            *bool      `json:"share_activities" gorm:"default: false"`
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
}

type UserPartialUpdateRequest struct {
	SundayAlert  *bool `json:"sunday_alert"`
	StravaPadel  *bool `json:"strava_padel"`
	StravaWalks  *bool `json:"strava_walks"`
	StravaPublic *bool `json:"strava_public"`
}

type UserUpdatePasswordRequest struct {
	ResetCode      string `json:"reset_code"`
	Password       string `json:"password"`
	PasswordRepeat string `json:"password_repeat"`
}

type UserStravaCodeUpdateRequest struct {
	StravaCode string `json:"strava_code"`
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
