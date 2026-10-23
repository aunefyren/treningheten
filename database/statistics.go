package database

import (
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StatisticsFilter narrows the statistics queries. Start and End are inclusive calendar-day
// bounds on the workout's day (only the date part is used); nil leaves that side open.
type StatisticsFilter struct {
	Start    *time.Time
	End      *time.Time
	ActionID *uuid.UUID
}

func applyStatisticsDates(query *gorm.DB, filter StatisticsFilter) *gorm.DB {
	// Whole-day bounds as strings in the server zone; see docs/data-conventions.md.
	if filter.Start != nil {
		query = query.Where("d.date >= ?", filter.Start.Format("2006-01-02")+" 00:00:00")
	}
	if filter.End != nil {
		query = query.Where("d.date <= ?", filter.End.Format("2006-01-02")+" 23:59:59")
	}
	return query
}

// GetStatisticsSessionRows returns one row per visible workout with its duration. Workouts
// without activities (the front page's "+") are included: they are workouts too.
func GetStatisticsSessionRows(userID uuid.UUID, filter StatisticsFilter) ([]models.StatisticsSessionRow, error) {
	rows := []models.StatisticsSessionRow{}
	query := Instance.Table("exercises AS e").
		Select("e.id AS exercise_id, d.date AS date, e.counts_toward_goal AS counts_toward_goal, "+
			"CASE WHEN e.duration > 0 THEN e.duration ELSE COALESCE(SUM(os.time), 0) END AS duration_seconds").
		Joins("JOIN exercise_days AS d ON d.id = e.exercise_day_id AND d.enabled = true").
		Joins("LEFT JOIN operations AS o ON o.exercise_id = e.id AND o.enabled = true").
		Joins("LEFT JOIN operation_sets AS os ON os.operation_id = o.id AND os.enabled = true").
		Where("e.enabled = true").
		Where("e.is_on = true").
		Where("d.user_id = ?", userID).
		Group("e.id, d.date, e.counts_toward_goal, e.duration").
		Order("d.date ASC")
	if filter.ActionID != nil {
		query = query.Where("EXISTS (SELECT 1 FROM operations AS fo WHERE fo.exercise_id = e.id AND fo.enabled = true AND fo.action_id = ?)", *filter.ActionID)
	}
	record := applyStatisticsDates(query, filter).Scan(&rows)
	return rows, record.Error
}

// GetStatisticsActivityRows returns one row per visible activity: its sets rolled up (summed
// distance, time, moving time and reps; the heaviest set) and its stream rollup columns.
func GetStatisticsActivityRows(userID uuid.UUID, filter StatisticsFilter) ([]models.StatisticsActivityRow, error) {
	rows := []models.StatisticsActivityRow{}
	query := Instance.Table("operations AS o").
		Select("o.id AS operation_id, e.id AS exercise_id, d.id AS exercise_day_id, d.date AS date, e.time AS time, "+
			"o.action_id AS action_id, COALESCE(a.name, '') AS action_name, COALESCE(a.strava_name, '') AS strava_name, "+
			"o.type AS type, o.distance_unit AS distance_unit, o.weight_unit AS weight_unit, "+
			"COALESCE(SUM(os.distance), 0) AS distance, COALESCE(SUM(os.time), 0) AS set_seconds, "+
			"COALESCE(SUM(os.moving_time), 0) AS moving_seconds, COALESCE(SUM(os.repetitions), 0) AS repetitions, "+
			"COALESCE(MAX(os.weight), 0) AS top_weight, COUNT(os.id) AS set_count, "+
			"o.duration AS operation_duration, o.avg_heartrate AS avg_heartrate, o.max_heartrate AS max_heartrate, "+
			"o.avg_cadence AS avg_cadence, o.elevation_gain_m AS elevation_gain_m, o.avg_watts AS avg_watts").
		Joins("JOIN exercises AS e ON e.id = o.exercise_id AND e.enabled = true AND e.is_on = true").
		Joins("JOIN exercise_days AS d ON d.id = e.exercise_day_id AND d.enabled = true").
		Joins("LEFT JOIN actions AS a ON a.id = o.action_id").
		Joins("LEFT JOIN operation_sets AS os ON os.operation_id = o.id AND os.enabled = true").
		Where("o.enabled = true").
		Where("d.user_id = ?", userID).
		Group("o.id, e.id, d.id, d.date, e.time, o.action_id, a.name, a.strava_name, o.type, o.distance_unit, o.weight_unit, " +
			"o.duration, o.avg_heartrate, o.max_heartrate, o.avg_cadence, o.elevation_gain_m, o.avg_watts").
		Order("d.date ASC, e.time ASC")
	if filter.ActionID != nil {
		query = query.Where("o.action_id = ?", *filter.ActionID)
	}
	record := applyStatisticsDates(query, filter).Scan(&rows)
	return rows, record.Error
}

// GetStatisticsSetRows returns every visible set that carries a weight or a rep count, on
// activities with an activity type, for the strength records and bests.
func GetStatisticsSetRows(userID uuid.UUID, filter StatisticsFilter) ([]models.StatisticsSetRow, error) {
	rows := []models.StatisticsSetRow{}
	query := Instance.Table("operation_sets AS os").
		Select("os.operation_id AS operation_id, d.id AS exercise_day_id, d.date AS date, e.time AS time, "+
			"o.action_id AS action_id, o.type AS type, o.weight_unit AS weight_unit, "+
			"COALESCE(os.repetitions, 0) AS repetitions, COALESCE(os.weight, 0) AS weight").
		Joins("JOIN operations AS o ON o.id = os.operation_id AND o.enabled = true").
		Joins("JOIN exercises AS e ON e.id = o.exercise_id AND e.enabled = true AND e.is_on = true").
		Joins("JOIN exercise_days AS d ON d.id = e.exercise_day_id AND d.enabled = true").
		Where("os.enabled = true").
		Where("o.action_id IS NOT NULL").
		Where("(os.weight > 0 OR os.repetitions > 0)").
		Where("d.user_id = ?", userID).
		Order("d.date ASC, e.time ASC")
	if filter.ActionID != nil {
		query = query.Where("o.action_id = ?", *filter.ActionID)
	}
	record := applyStatisticsDates(query, filter).Scan(&rows)
	return rows, record.Error
}

// GetActionTracksForUser returns the GPS track of every visible activity of one type in the
// period, for the heatmap: one latlng series per activity, from its first stored stream.
func GetActionTracksForUser(userID uuid.UUID, filter StatisticsFilter) ([][][]float64, error) {
	tracks := [][][]float64{}
	var rows []struct {
		OperationID uuid.UUID
		Streams     *models.StravaStreamsJSON
	}
	query := Instance.Table("operation_sets AS os").
		Select("os.operation_id AS operation_id, os.strava_streams AS streams").
		Joins("JOIN operations AS o ON o.id = os.operation_id AND o.enabled = true").
		Joins("JOIN exercises AS e ON e.id = o.exercise_id AND e.enabled = true AND e.is_on = true").
		Joins("JOIN exercise_days AS d ON d.id = e.exercise_day_id AND d.enabled = true").
		Where("os.enabled = true").
		Where("os.strava_streams IS NOT NULL").
		Where("d.user_id = ?", userID).
		Order("d.date ASC")
	if filter.ActionID != nil {
		query = query.Where("o.action_id = ?", *filter.ActionID)
	}
	if record := applyStatisticsDates(query, filter).Scan(&rows); record.Error != nil {
		return tracks, record.Error
	}

	seen := map[uuid.UUID]bool{}
	for _, row := range rows {
		if seen[row.OperationID] || row.Streams == nil || row.Streams.LatLng == nil {
			continue
		}
		seen[row.OperationID] = true
		track := [][]float64{}
		for _, point := range row.Streams.LatLng.Data {
			if len(point) == 2 {
				track = append(track, point)
			}
		}
		if len(track) > 1 {
			tracks = append(tracks, track)
		}
	}
	return tracks, nil
}
