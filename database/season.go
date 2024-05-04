package database

import (
	"aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// Verify if season with name exists
func VerifyUniqueSeasonName(providedSeasonName string) (bool, error) {
	var season models.Season
	seasonrecords := Instance.Where("`seasons`.enabled = ?", 1).Where("`seasons`.name = ?", providedSeasonName).Find(&season)
	if seasonrecords.Error != nil {
		return false, seasonrecords.Error
	}
	if seasonrecords.RowsAffected != 1 {
		return true, nil
	}
	return false, nil
}

// Get all enabled seasons
func GetAllEnabledSeasons() ([]models.Season, error) {
	var seasons []models.Season
	seasonrecord := Instance.Order("start desc").Where("`seasons`.enabled = ?", 1).Find(&seasons)
	if seasonrecord.Error != nil {
		return []models.Season{}, seasonrecord.Error
	}
	return seasons, nil
}

// Get season by ID
func GetSeasonByID(seasonID uuid.UUID) (*models.Season, error) {
	var season models.Season
	seasonrecord := Instance.Where("`seasons`.enabled = ?", 1).Where("`seasons`.ID = ?", seasonID).Find(&season)
	if seasonrecord.Error != nil {
		return nil, seasonrecord.Error
	} else if seasonrecord.RowsAffected != 1 {
		return nil, nil
	}
	return &season, nil
}

// Create new season
func CreateSeasonInDB(season models.Season) error {
	record := Instance.Create(&season)
	if record.Error != nil {
		return record.Error
	}
	return nil
}
