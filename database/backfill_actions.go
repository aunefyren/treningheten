package database

import (
	"strconv"

	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"
)

// backfillActionCategories is a one-time backfill for the Action.Category and Action.Equipment
// columns. Self-limiting: it only reads actions whose category is still empty, and gives every
// one of them a category, so the scan finds nothing on later boots. Hevy catalog actions get
// the category from their stored muscle group (BodyPart) and the equipment from the bracket in
// their title, exactly as models.HevyExerciseTemplate.ToAction does for new ones; every other
// action gets the default for its measurement type. Curated actions are then overwritten with
// their hand-picked values by SeedActions, which runs after Migrate.
func backfillActionCategories() {
	var actions []models.Action
	if err := Instance.Where("category IS NULL OR category = ?", "").Find(&actions).Error; err != nil {
		logger.Log.Warn("Failed to load actions needing a category. Error: " + err.Error())
		return
	}
	if len(actions) == 0 {
		return
	}

	updated := 0
	for _, action := range actions {
		fields := map[string]interface{}{"category": models.DefaultActionCategory(action.Type)}
		if action.HevyTemplateID != nil && *action.HevyTemplateID != "" {
			fields["category"] = models.HevyActionCategory(action.BodyPart)
			if action.Equipment == "" {
				fields["equipment"] = models.HevyTitleEquipment(action.Name)
			}
		}
		if err := Instance.Model(&models.Action{}).Where("id = ?", action.ID).Updates(fields).Error; err != nil {
			logger.Log.Warn("Failed to backfill category for action " + action.ID.String() + ". Error: " + err.Error())
			continue
		}
		updated++
	}
	logger.Log.Info("Backfilled categories for " + strconv.Itoa(updated) + " actions.")
}

// backfillHevyOperationNotes moves a Hevy catalog exercise's per-exercise note out of
// Operation.Note (the activity's title slot) into Operation.Description, where the importer now
// writes it. Only imports set Operation.Note, and custom Hevy exercises (no action) keep their
// title there, so the update is limited to catalog operations of Hevy sessions. Self-limiting:
// a moved row has a NULL note and no longer matches.
func backfillHevyOperationNotes() {
	result := Instance.Exec("UPDATE operations SET description = note, note = NULL " +
		"WHERE action_id IS NOT NULL AND note IS NOT NULL AND note <> '' " +
		"AND (description IS NULL OR description = '') " +
		"AND exercise_id IN (SELECT id FROM exercises WHERE hevy_workout_id IS NOT NULL)")
	if result.Error != nil {
		logger.Log.Warn("Failed to move Hevy exercise notes into descriptions. Error: " + result.Error.Error())
		return
	}
	if result.RowsAffected > 0 {
		logger.Log.Info("Moved " + strconv.FormatInt(result.RowsAffected, 10) + " Hevy exercise notes into descriptions.")
	}
}
