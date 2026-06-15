package database

import (
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func GetGearForUser(userID uuid.UUID) (gear []models.Gear, err error) {
	gear = []models.Gear{}
	err = nil

	record := Instance.Where("`gear`.enabled = ?", 1).
		Where("`gear`.user_id = ?", userID).
		Order("`gear`.retired asc, `gear`.name asc").
		Find(&gear)

	if record.Error != nil {
		return gear, record.Error
	}

	return
}

func GetGearByIDAndUserID(gearID uuid.UUID, userID uuid.UUID) (gear *models.Gear, err error) {
	gear = nil
	err = nil

	found := models.Gear{}
	record := Instance.Where("`gear`.enabled = ?", 1).
		Where("`gear`.id = ?", gearID).
		Where("`gear`.user_id = ?", userID).
		Find(&found)

	if record.Error != nil {
		return nil, record.Error
	} else if record.RowsAffected == 0 {
		return nil, nil
	}

	return &found, nil
}

// GetGearByID fetches a gear row by id without user scoping — used when
// flattening an operation that already belongs to the requesting user.
func GetGearByID(gearID uuid.UUID) (gear *models.Gear, err error) {
	gear = nil
	err = nil

	found := models.Gear{}
	record := Instance.Where("`gear`.id = ?", gearID).
		Find(&found)

	if record.Error != nil {
		return nil, record.Error
	} else if record.RowsAffected == 0 {
		return nil, nil
	}

	return &found, nil
}

func GetGearByStravaGearIDAndUserID(stravaGearID string, userID uuid.UUID) (gear *models.Gear, err error) {
	gear = nil
	err = nil

	found := models.Gear{}
	record := Instance.Where("`gear`.enabled = ?", 1).
		Where("`gear`.user_id = ?", userID).
		Where("`gear`.strava_gear_id = ?", stravaGearID).
		Find(&found)

	if record.Error != nil {
		return nil, record.Error
	} else if record.RowsAffected == 0 {
		return nil, nil
	}

	return &found, nil
}

func CreateGearInDB(gear models.Gear) (models.Gear, error) {
	record := Instance.Create(&gear)
	if record.Error != nil {
		return gear, record.Error
	}
	return gear, nil
}

func UpdateGearInDB(gear models.Gear) (models.Gear, error) {
	record := Instance.Save(&gear)
	if record.Error != nil {
		return gear, record.Error
	}
	return gear, nil
}

// UnsetPrimaryGearForUser clears the is_primary flag on every gear belonging to
// a user except exceptID, so a single gear can be promoted to primary.
func UnsetPrimaryGearForUser(userID uuid.UUID, exceptID uuid.UUID) error {
	record := Instance.Model(&models.Gear{}).
		Where("`gear`.user_id = ?", userID).
		Where("`gear`.id != ?", exceptID).
		Update("is_primary", false)
	return record.Error
}

// GetGearDistanceTotalsForUser returns each gear's total logged distance (km),
// summed from the operation sets of the operations linked to it. Distance is not
// stored on the gear — this is the single source of truth (see docs/gear.md).
func GetGearDistanceTotalsForUser(userID uuid.UUID) (map[uuid.UUID]float64, error) {
	type gearDistanceRow struct {
		GearID   uuid.UUID
		Distance float64
	}

	rows := []gearDistanceRow{}
	record := Instance.Model(&models.OperationSet{}).
		Select("`operations`.gear_id as gear_id, COALESCE(SUM(`operation_sets`.distance), 0) as distance").
		Joins("JOIN `operations` on `operation_sets`.operation_id = `operations`.id").
		Joins("JOIN `exercises` on `operations`.exercise_id = `exercises`.id").
		Joins("JOIN `exercise_days` on `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`operation_sets`.enabled = ?", 1).
		Where("`operations`.enabled = ?", 1).
		Where("`exercises`.enabled = ?", 1).
		Where("`exercise_days`.enabled = ?", 1).
		Where("`exercise_days`.user_id = ?", userID).
		Where("`operations`.gear_id IS NOT NULL").
		Group("`operations`.gear_id").
		Scan(&rows)

	if record.Error != nil {
		return nil, record.Error
	}

	totals := map[uuid.UUID]float64{}
	for _, row := range rows {
		totals[row.GearID] = row.Distance
	}

	return totals, nil
}
