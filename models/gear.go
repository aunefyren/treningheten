package models

import (
	"github.com/google/uuid"
)

// Gear is a piece of equipment (shoes, a bike, etc.) a user logs workouts
// against. It is linked to operations via Operation.GearID; a gear's total
// distance is not stored, it is computed from the operations linked to it (see
// docs/gear.md). StravaGearID is null for manually created gear and holds the
// Strava equipment id (e.g. "g12345" / "b12345") for gear imported from Strava.
type Gear struct {
	GormModel
	Enabled      bool      `json:"enabled" gorm:"not null; default: true"`
	UserID       uuid.UUID `json:"" gorm:"type:varchar(100); not null; index"`
	User         User      `json:"user" gorm:"foreignKey:UserID; references:ID"`
	Name         string    `json:"name" gorm:"type:varchar(191); not null"`
	Type         string    `json:"type" gorm:"not null; default: shoe"`
	Brand        *string   `json:"brand" gorm:"default: null"`
	Model        *string   `json:"model" gorm:"default: null"`
	Nickname     *string   `json:"nickname" gorm:"default: null"`
	Retired      bool      `json:"retired" gorm:"not null; default: false"`
	IsPrimary    bool      `json:"is_primary" gorm:"column:is_primary; not null; default: false"`
	StravaGearID *string   `json:"strava_gear_id" gorm:"type:varchar(191); default: null; index"`
}

// TableName pins the table to "gear" (the word is uncountable) instead of
// GORM's default pluralisation.
func (Gear) TableName() string {
	return "gear"
}

// GearObject is the enriched read shape: identity plus the computed total
// distance (km) summed from the operations linked to the gear.
type GearObject struct {
	GormModel
	Enabled      bool      `json:"enabled"`
	User         uuid.UUID `json:"user"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Brand        *string   `json:"brand"`
	Model        *string   `json:"model"`
	Nickname     *string   `json:"nickname"`
	Retired      bool      `json:"retired"`
	IsPrimary    bool      `json:"is_primary"`
	StravaGearID *string   `json:"strava_gear_id"`
	Distance     float64   `json:"distance"`
}

type GearCreationRequest struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Brand    *string `json:"brand"`
	Model    *string `json:"model"`
	Nickname *string `json:"nickname"`
}

// GearUpdateRequest fields are pointers so an omitted field leaves the stored
// value untouched. Identity fields on Strava-sourced gear are read-only and are
// rejected by the handler.
type GearUpdateRequest struct {
	Name      *string `json:"name"`
	Type      *string `json:"type"`
	Brand     *string `json:"brand"`
	Model     *string `json:"model"`
	Nickname  *string `json:"nickname"`
	Retired   *bool   `json:"retired"`
	IsPrimary *bool   `json:"is_primary"`
}
