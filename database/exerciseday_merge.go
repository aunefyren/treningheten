package database

import (
	"strconv"
	"strings"

	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// mergeDuplicateExerciseDays is a one-time cleanup for days created before the SQLite
// midnight fix (see docs/data-conventions.md): a whole-day lookup missed the existing
// midnight-stamped day, so a second enabled day was created for the same user and date.
// For each such (user, date) the oldest day is kept; the other days' sessions move onto it,
// their notes are appended to its note and a missing goal is adopted, and the duplicates
// are disabled. Self-limiting: once no duplicates remain it only reads the enabled days.
func mergeDuplicateExerciseDays() {
	var days []models.ExerciseDay
	err := Instance.Where("enabled = ?", true).
		Where("user_id IS NOT NULL").
		Order("created_at ASC").
		Find(&days).Error
	if err != nil {
		logger.Log.Warn("Failed to read exercise days for duplicate merge. Error: " + err.Error())
		return
	}

	merged := 0
	for _, group := range groupDuplicateExerciseDays(days) {
		if err := mergeExerciseDayGroup(group[0], group[1:]); err != nil {
			logger.Log.Warn("Failed to merge duplicate exercise days for " + group[0].ID.String() + ". Error: " + err.Error())
			continue
		}
		merged += len(group) - 1
	}

	if merged > 0 {
		logger.Log.Info("Merged " + strconv.Itoa(merged) + " duplicate exercise days.")
	}
}

// groupDuplicateExerciseDays returns the groups of days that share a user and calendar
// date, keeping the input order within each group (so with days sorted oldest first, the
// keeper is group[0]). Days without a duplicate are left out.
func groupDuplicateExerciseDays(days []models.ExerciseDay) [][]models.ExerciseDay {
	groups := map[string][]models.ExerciseDay{}
	var keys []string
	for _, day := range days {
		if day.UserID == nil {
			continue
		}
		key := day.UserID.String() + "|" + day.Date.Format("2006-01-02")
		if _, seen := groups[key]; !seen {
			keys = append(keys, key)
		}
		groups[key] = append(groups[key], day)
	}

	var duplicates [][]models.ExerciseDay
	for _, key := range keys {
		if len(groups[key]) > 1 {
			duplicates = append(duplicates, groups[key])
		}
	}
	return duplicates
}

// mergeExerciseDayGroup folds duplicates into keeper in one transaction.
func mergeExerciseDayGroup(keeper models.ExerciseDay, duplicates []models.ExerciseDay) error {
	duplicateIDs := make([]uuid.UUID, 0, len(duplicates))
	note := keeper.Note
	goalID := keeper.GoalID
	for _, duplicate := range duplicates {
		duplicateIDs = append(duplicateIDs, duplicate.ID)
		note = mergeExerciseDayNotes(note, duplicate.Note)
		if goalID == nil {
			goalID = duplicate.GoalID
		}
	}

	return Instance.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&models.Exercise{}).
			Where("exercise_day_id IN ?", duplicateIDs).
			Update("exercise_day_id", keeper.ID).Error
		if err != nil {
			return err
		}

		keeperChanges := map[string]any{}
		if note != keeper.Note {
			keeperChanges["note"] = note
		}
		if keeper.GoalID == nil && goalID != nil {
			keeperChanges["goal_id"] = *goalID
		}
		if len(keeperChanges) > 0 {
			if err := tx.Model(&models.ExerciseDay{}).Where("id = ?", keeper.ID).Updates(keeperChanges).Error; err != nil {
				return err
			}
		}

		return tx.Model(&models.ExerciseDay{}).
			Where("id IN ?", duplicateIDs).
			Update("enabled", false).Error
	})
}

// mergeExerciseDayNotes appends addition to note on its own line, skipping blanks and
// repeats.
func mergeExerciseDayNotes(note string, addition string) string {
	addition = strings.TrimSpace(addition)
	if addition == "" || addition == strings.TrimSpace(note) {
		return note
	}
	if strings.TrimSpace(note) == "" {
		return addition
	}
	return note + "\n" + addition
}
