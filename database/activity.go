package database

import (
	"strings"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// activityFeedSelectColumns is the per-activity (Operation) projection: one row per activity,
// with its own sets aggregated. It is what the session feed hangs inside each session.
const activityFeedSelectColumns = "o.id AS operation_id, " +
	"o.exercise_id AS exercise_id, " +
	"e.exercise_day_id AS exercise_day_id, " +
	"d.date AS date, " +
	"e.time AS time, " +
	"o.action_id AS action_id, " +
	"COALESCE(a.name, '') AS action_name, " +
	"COALESCE(a.type, '') AS action_type, " +
	"COALESCE(a.has_logo, false) AS action_has_logo, " +
	"o.note AS note, " +
	"o.distance_unit AS distance_unit, " +
	"o.weight_unit AS weight_unit, " +
	"COALESCE(SUM(os.distance), 0) AS distance, " +
	"COALESCE(SUM(os.time), 0) AS duration_seconds, " +
	"COALESCE(SUM(os.moving_time), 0) AS moving_seconds, " +
	"COALESCE(SUM(os.repetitions), 0) AS repetitions, " +
	"COALESCE(MAX(os.weight), 0) AS top_weight, " +
	"COUNT(os.id) AS set_count, " +
	"(COALESCE(SUM(CASE WHEN os.strava_id IS NOT NULL AND os.strava_id <> '' THEN 1 ELSE 0 END), 0) > 0) AS has_strava, " +
	// Operation-level rollups are single-valued per group; MAX() reads the value while
	// staying valid under ONLY_FULL_GROUP_BY, and preserves NULL when unset.
	"MAX(o.avg_heartrate) AS avg_heartrate, " +
	"MAX(o.max_heartrate) AS max_heartrate, " +
	"MAX(o.avg_cadence) AS avg_cadence, " +
	"MAX(o.temp_c) AS temp_c, " +
	"MAX(o.elevation_gain_m) AS elevation_gain_m, " +
	"e.hevy_workout_id AS hevy_workout_id, " +
	"e.counts_toward_goal AS counts_toward_goal, " +
	"e.private AS private"

// activityRowsBase builds the unfiltered operation-rooted query: it walks the enabled chain
// operations→exercises→exercise_days→users (same as GetOperationsByUserID) and LEFT JOINs
// operation_sets for the per-activity aggregates. GROUP BY lists every selected
// non-aggregated column (not just o.id) so it is valid under MySQL's ONLY_FULL_GROUP_BY;
// sqlite (used by the tests) is indifferent to the extra columns.
func activityRowsBase(userID uuid.UUID) *gorm.DB {
	return Instance.Table("operations AS o").
		Joins("JOIN exercises AS e ON o.exercise_id = e.id AND e.enabled = true").
		Joins("JOIN exercise_days AS d ON e.exercise_day_id = d.id AND d.enabled = true").
		Joins("LEFT JOIN actions AS a ON o.action_id = a.id").
		Joins("LEFT JOIN operation_sets AS os ON os.operation_id = o.id AND os.enabled = true").
		Where("o.enabled = true").
		Where("e.is_on = true").
		Where("d.user_id = ?", userID).
		Group("o.id, o.exercise_id, e.exercise_day_id, d.date, e.time, o.action_id, a.name, a.type, a.has_logo, o.note, o.distance_unit, o.weight_unit, e.hevy_workout_id, e.counts_toward_goal, e.private")
}

// sessionFeedBase builds the filtered, grouped query for the session feed. The root table is
// `exercises` and the joins down to operations/sets are LEFT joins, which is the whole point:
// a session logged with the front page's "+" button has no Operation rows at all, and an
// operation-rooted query drops it from the timeline entirely.
//
// The activity-level filters (action, free text, has-distance) are expressed as EXISTS
// subqueries rather than as predicates on the join, so matching a session never narrows the
// aggregate — a card that matched on its run still reports the whole workout's totals, and
// still lists every activity in it.
func sessionFeedBase(userID uuid.UUID, filter models.ActivityFeedFilter) *gorm.DB {
	query := Instance.Table("exercises AS e").
		Joins("JOIN exercise_days AS d ON e.exercise_day_id = d.id AND d.enabled = true").
		Joins("LEFT JOIN operations AS o ON o.exercise_id = e.id AND o.enabled = true").
		Joins("LEFT JOIN operation_sets AS os ON os.operation_id = o.id AND os.enabled = true").
		Where("e.enabled = true").
		Where("e.is_on = true").
		Where("d.user_id = ?", userID).
		Group("e.id, e.exercise_day_id, d.date, e.time, e.note, e.duration, e.hevy_workout_id, e.counts_toward_goal, e.private")

	if filter.ActionID != nil {
		query = query.Where("EXISTS (SELECT 1 FROM operations AS fo WHERE fo.exercise_id = e.id AND fo.enabled = true AND fo.action_id = ?)", *filter.ActionID)
	}
	if name := strings.TrimSpace(filter.ActionName); name != "" {
		query = query.Where("EXISTS (SELECT 1 FROM operations AS fo JOIN actions AS fa ON fo.action_id = fa.id WHERE fo.exercise_id = e.id AND fo.enabled = true AND LOWER(fa.name) LIKE ?)", "%"+strings.ToLower(name)+"%")
	}
	if filter.Start != nil {
		query = query.Where("d.date >= ?", *filter.Start)
	}
	if filter.End != nil {
		query = query.Where("d.date <= ?", *filter.End)
	}
	if strings.TrimSpace(filter.Query) != "" {
		// Every note level (session / day / activity) plus the action name, case-folded on
		// both sides so the match is independent of column collation.
		like := "%" + strings.ToLower(strings.TrimSpace(filter.Query)) + "%"
		query = query.Where(
			"(LOWER(e.note) LIKE ? OR LOWER(d.note) LIKE ? OR EXISTS (SELECT 1 FROM operations AS fo LEFT JOIN actions AS fa ON fo.action_id = fa.id WHERE fo.exercise_id = e.id AND fo.enabled = true AND (LOWER(fo.note) LIKE ? OR LOWER(fa.name) LIKE ?)))",
			like, like, like, like,
		)
	}
	if filter.HasDistance {
		// Session-level: any distance anywhere in the workout.
		query = query.Having("COALESCE(SUM(os.distance), 0) > 0")
	}

	return query
}

// GetSessionFeedForUser returns one page of the /exercises timeline — sessions, newest first
// by default — plus the total number of matching sessions, so pagination and the result count
// are measured in workouts rather than in exercise rows.
//
// Two queries: the session page (aggregates rolled up across the whole workout), then one
// fetch of every activity belonging to the returned sessions, attached in feed order. The
// second query is deliberately unfiltered — a matched session shows all of its activities.
func GetSessionFeedForUser(userID uuid.UUID, filter models.ActivityFeedFilter) ([]models.SessionFeedItem, int64, error) {
	sessions := []models.SessionFeedItem{}

	sortColumns := map[string]string{
		"date":     "d.date",
		"distance": "distance",
		"duration": "duration_seconds",
		"weight":   "top_weight",
		"reps":     "repetitions",
	}
	sortColumn, ok := sortColumns[filter.Sort]
	if !ok {
		sortColumn = "d.date"
	}
	order := "DESC"
	if strings.EqualFold(filter.Order, "asc") {
		order = "ASC"
	}
	// The chosen key first, then day/start/creation so the order is deterministic and days
	// stay contiguous for the browse grouping.
	orderBy := sortColumn + " " + order + ", d.date DESC, e.time DESC, e.created_at ASC"

	pageQuery := sessionFeedBase(userID, filter).
		Select("e.id AS exercise_id, " +
			"e.exercise_day_id AS exercise_day_id, " +
			"d.date AS date, " +
			"e.time AS time, " +
			"e.note AS note, " +
			"COALESCE(SUM(os.distance), 0) AS distance, " +
			// The session's dominant units. A workout that genuinely mixes them is rare, and
			// each nested activity still carries its own exact unit.
			"COALESCE(MAX(o.distance_unit), '') AS distance_unit, " +
			"COALESCE(MAX(o.weight_unit), '') AS weight_unit, " +
			// The session's own duration wins over the summed set times (same cascade as
			// resolveSessionWindow), so a session logged with only a duration still reads
			// as one rather than as zero.
			"CASE WHEN e.duration > 0 THEN e.duration ELSE COALESCE(SUM(os.time), 0) END AS duration_seconds, " +
			"COALESCE(SUM(os.moving_time), 0) AS moving_seconds, " +
			"COALESCE(SUM(os.repetitions), 0) AS repetitions, " +
			"COALESCE(MAX(os.weight), 0) AS top_weight, " +
			"COUNT(os.id) AS set_count, " +
			"(COALESCE(SUM(CASE WHEN os.strava_id IS NOT NULL AND os.strava_id <> '' THEN 1 ELSE 0 END), 0) > 0) AS has_strava, " +
			"COUNT(DISTINCT o.id) AS activity_count, " +
			"e.hevy_workout_id AS hevy_workout_id, " +
			"e.counts_toward_goal AS counts_toward_goal, " +
			"e.private AS private").
		Order(orderBy).
		Limit(filter.Limit).
		Offset(filter.Offset)

	if record := pageQuery.Scan(&sessions); record.Error != nil {
		return []models.SessionFeedItem{}, 0, record.Error
	}

	// Total matching sessions. Wrapping the grouped id query in a subquery counts groups
	// correctly even with the has_distance HAVING.
	var total int64
	countSub := sessionFeedBase(userID, filter).Select("e.id")
	if record := Instance.Table("(?) AS sub", countSub).Count(&total); record.Error != nil {
		return []models.SessionFeedItem{}, 0, record.Error
	}

	if len(sessions) == 0 {
		return sessions, total, nil
	}

	exerciseIDs := make([]uuid.UUID, 0, len(sessions))
	for _, session := range sessions {
		exerciseIDs = append(exerciseIDs, session.ExerciseID)
	}

	activities, err := getActivityRowsForExercises(userID, exerciseIDs)
	if err != nil {
		return []models.SessionFeedItem{}, 0, err
	}

	for i := range sessions {
		sessions[i].Activities = activities[sessions[i].ExerciseID]
		if sessions[i].Activities == nil {
			sessions[i].Activities = []models.ActivityFeedItem{}
		}
		sessions[i].ActivityCount = len(sessions[i].Activities)
		// Mirror the count onto each nested activity so the field never reads 0 in a payload
		// where the session already knows the answer.
		for j := range sessions[i].Activities {
			sessions[i].Activities[j].SessionActivityCount = sessions[i].ActivityCount
		}
	}

	return sessions, total, nil
}

// getActivityRowsForExercises loads every enabled activity of the given sessions, keyed by
// session, using the same projection as the operation-rooted feed. Ordered by creation so the
// activities read in the order they were logged.
func getActivityRowsForExercises(userID uuid.UUID, exerciseIDs []uuid.UUID) (map[uuid.UUID][]models.ActivityFeedItem, error) {
	byExercise := map[uuid.UUID][]models.ActivityFeedItem{}
	if len(exerciseIDs) == 0 {
		return byExercise, nil
	}

	items := []models.ActivityFeedItem{}
	query := activityRowsBase(userID).
		Where("o.exercise_id IN ?", exerciseIDs).
		Select(activityFeedSelectColumns).
		Order("o.created_at ASC")

	if record := query.Scan(&items); record.Error != nil {
		return nil, record.Error
	}

	for _, item := range items {
		byExercise[item.ExerciseID] = append(byExercise[item.ExerciseID], item)
	}
	return byExercise, nil
}
