package models

type Prize struct {
	GormModel
	Name     string `json:"name" gorm:"not null"`
	Quantity int    `json:"quantity" gorm:"not null"`
	Enabled  bool   `json:"enabled" gorm:"not null;default: true"`
}

type PrizeCreationRequest struct {
	Name     string `json:"name" sanitize:"trim,collapse,capfirst"`
	Quantity int    `json:"quantity"`
}
