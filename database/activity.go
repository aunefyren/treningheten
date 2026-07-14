package database

import (
	"strings"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// activityFeedBase builds the filtered, joined, grouped query for the activity feed. Both
// the page fetch and the total count derive from this, so their WHERE/HAVING stay in lockstep.
// It walks the enabled chain operations→exercises→exercise_days→users (same as
// GetOperationsByUserID) and LEFT JOINs operation_sets for the per-activity aggregates.
// GROUP BY lists every selected non-aggregated column (not just o.id) so it is valid under
// MySQL's ONLY_FULL_GROUP_BY; sqlite (used by the tests) is indifferent to the extra columns.
func activityFeedBase(userID uuid.UUID, filter models.ActivityFeedFilter) *gorm.DB {
	query := Instance.Table("`operations` AS `o`").
		Joins("JOIN `exercises` AS `e` ON `o`.`exercise_id` = `e`.`id` AND `e`.`enabled` = 1").
		Joins("JOIN `exercise_days` AS `d` ON `e`.`exercise_day_id` = `d`.`id` AND `d`.`enabled` = 1").
		Joins("LEFT JOIN `actions` AS `a` ON `o`.`action_id` = `a`.`id`").
		Joins("LEFT JOIN `operation_sets` AS `os` ON `os`.`operation_id` = `o`.`id` AND `os`.`enabled` = 1").
		Where("`o`.`enabled` = 1").
		Where("`e`.`is_on` = 1").
		Where("`d`.`user_id` = ?", userID).
		Group("`o`.`id`, `o`.`exercise_id`, `e`.`exercise_day_id`, `d`.`date`, `e`.`time`, `o`.`action_id`, `a`.`name`, `a`.`type`, `a`.`has_logo`, `o`.`note`, `o`.`distance_unit`, `o`.`weight_unit`")

	if filter.ActionID != nil {
		query = query.Where("`o`.`action_id` = ?", *filter.ActionID)
	}
	if filter.Start != nil {
		query = query.Where("`d`.`date` >= ?", *filter.Start)
	}
	if filter.End != nil {
		query = query.Where("`d`.`date` <= ?", *filter.End)
	}
	if strings.TrimSpace(filter.Query) != "" {
		// Case-fold both sides so "cosmo" finds "Cosmo" regardless of the column collation,
		// and search every note level (operation / session / day) plus the action name —
		// a name in the session or day note should be findable, not just the activity note.
		like := "%" + strings.ToLower(strings.TrimSpace(filter.Query)) + "%"
		query = query.Where(
			"(LOWER(`o`.`note`) LIKE ? OR LOWER(`e`.`note`) LIKE ? OR LOWER(`d`.`note`) LIKE ? OR LOWER(`a`.`name`) LIKE ?)",
			like, like, like, like,
		)
	}
	if filter.HasDistance {
		query = query.Having("COALESCE(SUM(`os`.`distance`), 0) > 0")
	}

	return query
}

// GetActivityFeedForUser returns one page of the activity timeline plus the total number of
// matching activities. Metrics are aggregated from each operation's enabled sets. Sort is
// mapped to a whitelisted column (defaulting to date) and the ordering keeps a session's
// activities adjacent so the client can group them in browse mode.
func GetActivityFeedForUser(userID uuid.UUID, filter models.ActivityFeedFilter) ([]models.ActivityFeedItem, int64, error) {
	items := []models.ActivityFeedItem{}

	sortColumns := map[string]string{
		"date":     "`d`.`date`",
		"distance": "distance",
		"duration": "duration_seconds",
		"weight":   "top_weight",
		"reps":     "repetitions",
	}
	sortColumn, ok := sortColumns[filter.Sort]
	if !ok {
		sortColumn = "`d`.`date`"
	}
	order := "DESC"
	if strings.EqualFold(filter.Order, "asc") {
		order = "ASC"
	}
	// The chosen key first, then day/session/creation so a session's activities stay
	// contiguous (browse grouping) and the order is deterministic.
	orderBy := sortColumn + " " + order + ", `d`.`date` DESC, `e`.`time` DESC, `o`.`created_at` ASC"

	pageQuery := activityFeedBase(userID, filter).
		Select("`o`.`id` AS operation_id, " +
			"`o`.`exercise_id` AS exercise_id, " +
			"`e`.`exercise_day_id` AS exercise_day_id, " +
			"`d`.`date` AS date, " +
			"`e`.`time` AS time, " +
			"`o`.`action_id` AS action_id, " +
			"COALESCE(`a`.`name`, '') AS action_name, " +
			"COALESCE(`a`.`type`, '') AS action_type, " +
			"COALESCE(`a`.`has_logo`, 0) AS action_has_logo, " +
			"`o`.`note` AS note, " +
			"`o`.`distance_unit` AS distance_unit, " +
			"`o`.`weight_unit` AS weight_unit, " +
			"COALESCE(SUM(`os`.`distance`), 0) AS distance, " +
			"COALESCE(SUM(`os`.`time`), 0) AS duration_seconds, " +
			"COALESCE(SUM(`os`.`repetitions`), 0) AS repetitions, " +
			"COALESCE(MAX(`os`.`weight`), 0) AS top_weight, " +
			"COUNT(`os`.`id`) AS set_count, " +
			"(COALESCE(SUM(CASE WHEN `os`.`strava_id` IS NOT NULL AND `os`.`strava_id` <> '' THEN 1 ELSE 0 END), 0) > 0) AS has_strava").
		Order(orderBy).
		Limit(filter.Limit).
		Offset(filter.Offset)

	if record := pageQuery.Scan(&items); record.Error != nil {
		return []models.ActivityFeedItem{}, 0, record.Error
	}

	// Total matching activities (for has_more and result counts). Wrapping the grouped id
	// query in a subquery counts groups correctly even with the has_distance HAVING.
	var total int64
	countSub := activityFeedBase(userID, filter).Select("`o`.`id`")
	if record := Instance.Table("(?) AS `sub`", countSub).Count(&total); record.Error != nil {
		return []models.ActivityFeedItem{}, 0, record.Error
	}

	// SessionActivityCount = all enabled activities in each returned session (independent of
	// the feed filter), so a browse card can show the true session size.
	if len(items) > 0 {
		seen := map[uuid.UUID]bool{}
		exerciseIDs := make([]uuid.UUID, 0, len(items))
		for _, item := range items {
			if !seen[item.ExerciseID] {
				seen[item.ExerciseID] = true
				exerciseIDs = append(exerciseIDs, item.ExerciseID)
			}
		}

		type sessionCount struct {
			ExerciseID uuid.UUID
			Count      int
		}
		counts := []sessionCount{}
		if record := Instance.Table("`operations`").
			Select("`exercise_id` AS exercise_id, COUNT(*) AS count").
			Where("`enabled` = 1").
			Where("`exercise_id` IN ?", exerciseIDs).
			Group("`exercise_id`").
			Scan(&counts); record.Error != nil {
			return []models.ActivityFeedItem{}, 0, record.Error
		}

		countByExercise := map[uuid.UUID]int{}
		for _, c := range counts {
			countByExercise[c.ExerciseID] = c.Count
		}
		for i := range items {
			items[i].SessionActivityCount = countByExercise[items[i].ExerciseID]
		}
	}

	return items, total, nil
}
