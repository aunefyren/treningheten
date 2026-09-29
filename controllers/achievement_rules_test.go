package controllers

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// hasAchievement reports whether the user was awarded the achievement.
func hasAchievement(t *testing.T, userID uuid.UUID, achievementID string) bool {
	t.Helper()
	var count int64
	database.Instance.Model(&models.AchievementDelegation{}).
		Where("user_id = ? AND achievement_id = ?", userID, achievementID).Count(&count)
	return count > 0
}

// processSeasonWeeks runs the Monday job for every week of the season, oldest first.
func processSeasonWeeks(t *testing.T, season models.Season) {
	t.Helper()
	for wednesday := season.Start.AddDate(0, 0, 2); wednesday.Before(season.End); wednesday = wednesday.AddDate(0, 0, 7) {
		if err := ProcessWeekOfSeason(season, wednesday, true, true, nil); err != nil {
			t.Fatalf("processing week of %s: %v", wednesday.Format("2006-01-02"), err)
		}
	}
}

// A five-week season over Advent and Christmas 2025: one member trains every day
// (beating a three-a-week goal every week), the other never does. Processing each week
// must award the calendar achievements on their exact days, the streak and
// over-achiever achievements, and the season-end ones.
func TestWeeklyProcessingAwardsCalendarAndStreakAchievements(t *testing.T) {
	h := newAPIHarness(t)
	withControllerSMTP(t)
	eager, _ := h.user("eager@rules.test", false)
	idle, _ := h.user("idle@rules.test", false)

	// Seasons are created in the server's zone, as in production (main sets time.Local
	// from the configured timezone); mixing zones is what broke CI running in UTC.
	zone := time.Local
	start := time.Date(2025, 11, 24, 0, 0, 0, 0, zone) // a Monday
	end := time.Date(2025, 12, 28, 23, 59, 59, 0, zone)
	season := seedSeason(t, "Advent season", start, end, false)
	seedGoal(t, eager.ID, season, 3)
	seedGoal(t, idle.ID, season, 3)

	for day := start; day.Before(end); day = day.AddDate(0, 0, 1) {
		seedExerciseDayWithExercises(t, eager.ID, day, 1)
	}

	processSeasonWeeks(t, season)

	for name, achievementID := range map[string]string{
		"first advent":           "5276382c-fdae-410b-a298-5107a3ff3089",
		"second advent":          "6c991ba6-d0ae-4022-9410-6558e376ec5e",
		"third advent":           "7ef923b5-21aa-4478-a658-68078f499620",
		"last advent":            "720b036c-7d24-418f-88e6-a0e84147efda",
		"christmas eve":          "c4a131a6-2aa6-49fb-98e5-fa797152a9a4",
		"three-week streak":      "8875597e-d8f5-4514-b96f-c51ecce4eb1f",
		"beat the weekly goal":   "f7fad558-3e59-4812-9b13-4c30a91c04b9",
		"won a week (wheel win)": "bb964360-6413-47c2-8400-ee87b40365a7",
	} {
		if !hasAchievement(t, eager.ID, achievementID) {
			t.Errorf("the everyday member was not awarded %s", name)
		}
	}
	if hasAchievement(t, idle.ID, "5276382c-fdae-410b-a298-5107a3ff3089") {
		t.Error("the idle member got a training achievement")
	}

	// Their achievement list renders, including for another viewer.
	token := h.ok("GET", "/api/auth/achievements?user="+eager.ID.String(), mustToken(t, idle), nil)
	if len(token) == 0 {
		t.Error("empty achievements response")
	}
}

func TestSeventeenthOfMayAchievement(t *testing.T) {
	h := newAPIHarness(t)
	withControllerSMTP(t)
	patriot, _ := h.user("patriot@rules.test", false)
	other, _ := h.user("other@rules.test", false)

	// Seasons are created in the server's zone, as in production (main sets time.Local
	// from the configured timezone); mixing zones is what broke CI running in UTC.
	zone := time.Local
	season := seedSeason(t, "May season", time.Date(2025, 5, 12, 0, 0, 0, 0, zone), time.Date(2025, 5, 18, 23, 59, 59, 0, zone), false)
	seedGoal(t, patriot.ID, season, 1)
	seedGoal(t, other.ID, season, 1)
	seedExerciseDayWithExercises(t, patriot.ID, time.Date(2025, 5, 17, 0, 0, 0, 0, zone), 1)

	processSeasonWeeks(t, season)

	if !hasAchievement(t, patriot.ID, "ab0b1bf0-c57b-469f-a6ba-5d195f1b896d") {
		t.Error("training on 17 May was not awarded")
	}
}

func mustToken(t *testing.T, user models.User) string {
	t.Helper()
	token, _, err := auth.GenerateAccessToken(user.ID, false, auth.ScopeForUser(false), models.FirstPartyClientID)
	if err != nil {
		t.Fatal(err)
	}
	return token
}
