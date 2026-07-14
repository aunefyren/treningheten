package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"
)

func TestGetAdminStatsCounts(t *testing.T) {
	newTestDB(t)

	now := time.Now()

	// Two enabled users, one of them in an ongoing season and Strava-connected.
	active := makeTestUser(t, "statsactive@example.com", func(u *models.User) { u.StravaCode = strPtr("r:token") })
	makeTestUser(t, "statsplain@example.com", nil)
	// A disabled user must not be counted anywhere.
	makeTestUser(t, "statsdisabled@example.com", func(u *models.User) { u.Enabled = false })

	// Ongoing season with an enabled goal for the active user.
	season := makeSeason(t, "StatsSeason", now.Add(-24*time.Hour), now.Add(24*time.Hour), true)
	makeGoal(t, active.ID, season.ID, true)

	// A push subscription for the active user.
	makeSubscription(t, active.ID, "stats-endpoint", false, false, false)

	// One enabled achievement, granted once.
	ach := makeAchievement(t, "StatsAch", "cat", 1, true)
	makeDelegation(t, active.ID, ach.ID, false)

	counts, err := GetAdminStatsCounts(now)
	if err != nil {
		t.Fatalf("GetAdminStatsCounts returned error: %v", err)
	}

	if counts.TotalUsers != 2 {
		t.Errorf("TotalUsers: got %d, want 2", counts.TotalUsers)
	}
	if counts.UsersInSeasonNow != 1 {
		t.Errorf("UsersInSeasonNow: got %d, want 1", counts.UsersInSeasonNow)
	}
	if counts.UsersWithNotifications != 1 {
		t.Errorf("UsersWithNotifications: got %d, want 1", counts.UsersWithNotifications)
	}
	if counts.UsersWithStrava != 1 {
		t.Errorf("UsersWithStrava: got %d, want 1", counts.UsersWithStrava)
	}
	if counts.AchievementsTotal != 1 {
		t.Errorf("AchievementsTotal: got %d, want 1", counts.AchievementsTotal)
	}
	if counts.AchievementDelegations != 1 {
		t.Errorf("AchievementDelegations: got %d, want 1", counts.AchievementDelegations)
	}
}
