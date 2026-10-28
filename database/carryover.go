package database

import (
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// "Use saved workouts" opt-ins (models.CarryOverUse), one enabled row per goal and week.
// See docs/flexible-workouts.md.

// GetCarryOverUsesForGoalIDsBetweenDates returns the enabled opt-ins for the given goals
// whose date falls between the two dates (inclusive, whole days), so the season walk can
// bucket them by (goal, week) without a query per week. Returns an empty slice when no goal
// IDs are supplied.
func GetCarryOverUsesForGoalIDsBetweenDates(goalIDs []uuid.UUID, startDate time.Time, endDate time.Time) ([]models.CarryOverUse, error) {
	uses := []models.CarryOverUse{}
	if len(goalIDs) == 0 {
		return uses, nil
	}

	startDayString := startDate.Format("2006-01-02") + " 00:00:00"
	endDayString := endDate.Format("2006-01-02") + " 23:59:59"

	record := Instance.
		Where("carry_over_uses.enabled = ?", true).
		Where("carry_over_uses.goal_id IN ?", goalIDs).
		Where("carry_over_uses.date >= ?", startDayString).
		Where("carry_over_uses.date <= ?", endDayString).
		Find(&uses)
	if record.Error != nil {
		return []models.CarryOverUse{}, record.Error
	}

	return uses, nil
}

// GetCarryOverUseForGoalWithinWeek returns the goal's enabled opt-in for the Monday–Sunday
// week holding the given date, or nil when there is none.
func GetCarryOverUseForGoalWithinWeek(pointInTime time.Time, goalID uuid.UUID) (*models.CarryOverUse, error) {

	monday, sunday := weekBounds(pointInTime)

	var use models.CarryOverUse
	record := Instance.
		Where("carry_over_uses.enabled = ?", true).
		Where("carry_over_uses.goal_id = ?", goalID).
		Where("carry_over_uses.date >= ?", monday.Format("2006-01-02")+" 00:00:00").
		Where("carry_over_uses.date <= ?", sunday.Format("2006-01-02")+" 23:59:59").
		Order("carry_over_uses.created_at").
		Limit(1).
		Find(&use)
	if record.Error != nil {
		return nil, record.Error
	} else if record.RowsAffected != 1 {
		return nil, nil
	}

	return &use, nil
}

// CreateCarryOverUseInDB stores an opt-in.
func CreateCarryOverUseInDB(use models.CarryOverUse) error {
	record := Instance.Omit("Goal").Create(&use)
	if record.Error != nil {
		return record.Error
	}
	return nil
}

// DisableCarryOverUsesForGoalWithinWeek undoes the goal's opt-in for the week holding the
// given date. It reports how many rows it disabled (0 when the week had none).
func DisableCarryOverUsesForGoalWithinWeek(pointInTime time.Time, goalID uuid.UUID) (int64, error) {

	monday, sunday := weekBounds(pointInTime)

	record := Instance.Model(&models.CarryOverUse{}).
		Where("carry_over_uses.enabled = ?", true).
		Where("carry_over_uses.goal_id = ?", goalID).
		Where("carry_over_uses.date >= ?", monday.Format("2006-01-02")+" 00:00:00").
		Where("carry_over_uses.date <= ?", sunday.Format("2006-01-02")+" 23:59:59").
		Update("enabled", false)
	if record.Error != nil {
		return 0, record.Error
	}

	return record.RowsAffected, nil
}

// weekBounds returns the Monday and Sunday of the ISO week holding the given date, in its
// own zone.
func weekBounds(pointInTime time.Time) (time.Time, time.Time) {
	offset := (int(pointInTime.Weekday()) + 6) % 7 // days since Monday
	monday := pointInTime.AddDate(0, 0, -offset)
	return monday, monday.AddDate(0, 0, 6)
}
