package controllers

import (
	"sort"
	"time"

	"github.com/aunefyren/treningheten/models"
)

// personalStreakResult holds a user's season-independent activity streaks.
type personalStreakResult struct {
	DayCurrent  int
	DayBest     int
	WeekCurrent int
	WeekBest    int
}

// computePersonalStreaks derives the user's day and ISO-week activity streaks from the
// enriched day tree. A day (or week) counts when it contains at least one enabled,
// "on" exercise. Streaks are season- and goal-independent. "Best" is the longest run
// ever; "current" is only non-zero while still alive — for days that means activity
// today or yesterday, for weeks this ISO week or last.
//
// This is the single source of truth shared by the MCP get_statistics tool and the
// HTTP /users/:user_id/statistics endpoint, so the two cannot drift.
func computePersonalStreaks(dayObjects []models.ExerciseDayObject) personalStreakResult {
	daySeen := map[string]bool{}
	weekSeen := map[string]bool{}
	var dayStarts, weekStarts []time.Time

	for _, day := range dayObjects {
		active := false
		for _, exercise := range day.Exercises {
			if exercise.Enabled && exercise.IsOn {
				active = true
				break
			}
		}
		if !active {
			continue
		}

		ds := dayStart(day.Date)
		if key := ds.Format("2006-01-02"); !daySeen[key] {
			daySeen[key] = true
			dayStarts = append(dayStarts, ds)
		}
		ws := isoWeekStart(day.Date)
		if key := ws.Format("2006-01-02"); !weekSeen[key] {
			weekSeen[key] = true
			weekStarts = append(weekStarts, ws)
		}
	}

	sort.Slice(dayStarts, func(i, j int) bool { return dayStarts[i].Before(dayStarts[j]) })
	sort.Slice(weekStarts, func(i, j int) bool { return weekStarts[i].Before(weekStarts[j]) })

	now := time.Now()
	dayAnchors := []time.Time{dayStart(now), dayStart(now.AddDate(0, 0, -1))}
	weekAnchors := []time.Time{isoWeekStart(now), isoWeekStart(now.AddDate(0, 0, -7))}

	return personalStreakResult{
		DayCurrent:  currentRun(dayStarts, 1, dayAnchors),
		DayBest:     longestRun(dayStarts, 1),
		WeekCurrent: currentRun(weekStarts, 7, weekAnchors),
		WeekBest:    longestRun(weekStarts, 7),
	}
}

// longestRun returns the longest chain of entries spaced exactly stepDays apart.
// sorted must be ascending and de-duplicated.
func longestRun(sorted []time.Time, stepDays int) int {
	best, run := 0, 0
	for i, t := range sorted {
		if i > 0 && t.Equal(sorted[i-1].AddDate(0, 0, stepDays)) {
			run++
		} else {
			run = 1
		}
		if run > best {
			best = run
		}
	}
	return best
}

// currentRun returns the run ending at the most recent entry, but only if that entry
// matches one of the "alive" anchors (otherwise the streak has lapsed and is 0).
func currentRun(sorted []time.Time, stepDays int, aliveAnchors []time.Time) int {
	if len(sorted) == 0 {
		return 0
	}
	last := sorted[len(sorted)-1]
	alive := false
	for _, a := range aliveAnchors {
		if last.Equal(a) {
			alive = true
			break
		}
	}
	if !alive {
		return 0
	}

	run := 1
	for i := len(sorted) - 1; i > 0; i-- {
		if sorted[i].Equal(sorted[i-1].AddDate(0, 0, stepDays)) {
			run++
		} else {
			break
		}
	}
	return run
}

// dayStart truncates to midnight UTC so day comparisons are stable regardless of the
// stored time-of-day or zone.
func dayStart(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

// isoWeekStart returns midnight UTC on the Monday of t's ISO week.
func isoWeekStart(t time.Time) time.Time {
	d := dayStart(t)
	weekday := int(d.Weekday())
	if weekday == 0 { // Sunday -> 7 so Monday is the week start
		weekday = 7
	}
	return d.AddDate(0, 0, -(weekday - 1))
}
