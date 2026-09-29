package controllers

import (
	"net/http"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// seedGoal inserts a competing goal that predates the season, so the holder counts as
// having participated in every full week.
func seedGoal(t *testing.T, userID uuid.UUID, season models.Season, interval int) models.Goal {
	t.Helper()

	goal := models.Goal{SeasonID: season.ID, UserID: userID, ExerciseInterval: interval, Competing: true, Enabled: true}
	goal.ID = uuid.New()
	goal.CreatedAt = season.Start.AddDate(0, 0, -1)
	if err := database.Instance.Omit("Season", "User").Create(&goal).Error; err != nil {
		t.Fatalf("failed to seed goal: %v", err)
	}
	return goal
}

// wednesdayOfWeeksAgo returns noon on the Wednesday `weeks` weeks before this one.
func wednesdayOfWeeksAgo(weeks int) time.Time {
	day := time.Now().AddDate(0, 0, -7*weeks)
	for day.Weekday() != time.Wednesday {
		if day.Weekday() == time.Sunday || day.Weekday() > time.Wednesday {
			day = day.AddDate(0, 0, -1)
		} else {
			day = day.AddDate(0, 0, 1)
		}
	}
	return time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, time.Local)
}

// TestWeeklyResultsDebtAndWheel runs the Monday processing for a finished season:
// two members met their goal, one missed it, so the one who missed gets a debt with no
// winner chosen yet, spins the wheel, and the chosen winner marks the prize received.
func TestWeeklyResultsDebtAndWheel(t *testing.T) {
	h := newAPIHarness(t)
	withControllerSMTP(t)

	_, adminToken := h.user("admin@debt.test", true)
	alice, aliceToken := h.user("alice@debt.test", false)
	dave, daveToken := h.user("dave@debt.test", false)
	bob, bobToken := h.user("bob@debt.test", false)

	// Three weeks long, ending last Sunday.
	season := seedOngoingSeason(t, "Finished season", 3, 3)
	seedGoal(t, alice.ID, season, 1)
	seedGoal(t, dave.ID, season, 1)
	seedGoal(t, bob.ID, season, 2)

	for weeksAgo := 1; weeksAgo <= 3; weeksAgo++ {
		wednesday := wednesdayOfWeeksAgo(weeksAgo)
		seedExerciseDayWithExercises(t, alice.ID, wednesday, 2)
		seedExerciseDayWithExercises(t, dave.ID, wednesday, 1)
		seedExerciseDayWithExercises(t, bob.ID, wednesday, 1)
	}

	lastWeek := wednesdayOfWeeksAgo(1)
	h.expect(http.StatusBadRequest, "POST", "/api/admin/debts", adminToken, "{broken")
	h.ok("POST", "/api/admin/debts", adminToken, models.DebtCreationRequest{Date: lastWeek})
	// The Monday cron does the same for last week; it must not double-issue the debt.
	ProcessLastWeek()

	// Winners have nothing to spin.
	if unchosen := h.ok("GET", "/api/auth/debts/unchosen", aliceToken, nil); unchosen["debt"] != nil {
		t.Errorf("a winner has an unchosen debt: %v", unchosen)
	}

	unchosen := h.ok("GET", "/api/auth/debts/unchosen", bobToken, nil)
	debts, _ := unchosen["debt"].([]any)
	if len(debts) != 1 {
		t.Fatalf("bob's unchosen debts = %v, want exactly one", unchosen)
	}
	debtID := idOf(t, debts[0])
	debtPath := "/api/auth/debts/" + debtID

	// Bob must spin before logging a new week.
	h.expect(http.StatusBadRequest, "POST", "/api/auth/exercises/week", bobToken, models.WeekCreationRequest{Days: currentWeekDays(0), TimeZone: "UTC"})

	h.ok("GET", debtPath, bobToken, nil)
	h.expect(http.StatusBadRequest, "GET", "/api/auth/debts/nope", bobToken, nil)
	h.expect(http.StatusUnauthorized, "POST", debtPath+"/choose", aliceToken, nil)
	h.expect(http.StatusBadRequest, "POST", "/api/auth/debts/nope/choose", bobToken, nil)

	chosen := h.ok("POST", debtPath+"/choose", bobToken, nil)
	winnerID := idOf(t, chosen, "winner")
	if winnerID != alice.ID.String() && winnerID != dave.ID.String() {
		t.Fatalf("winner %s is not one of the two who met their goal", winnerID)
	}
	h.expect(http.StatusBadRequest, "POST", debtPath+"/choose", bobToken, nil)

	winnerToken := aliceToken
	if winnerID == dave.ID.String() {
		winnerToken = daveToken
	}
	h.ok("GET", "/api/auth/debts", winnerToken, nil)
	h.ok("GET", "/api/auth/debts", bobToken, nil)
	h.ok("POST", debtPath+"/received", winnerToken, nil)
	h.expect(http.StatusBadRequest, "POST", "/api/auth/debts/nope/received", winnerToken, nil)

	// Processing awarded achievements; everyone can see their own and others'.
	achievements := h.ok("GET", "/api/auth/achievements", bobToken, nil)
	if len(achievements) == 0 {
		t.Error("no achievements response")
	}
	h.ok("GET", "/api/auth/achievements?user="+alice.ID.String(), bobToken, nil)
	h.expect(http.StatusBadRequest, "GET", "/api/auth/achievements?user=nope", bobToken, nil)

	// Admin can hand out an achievement directly.
	h.expect(http.StatusCreated, "POST", "/api/admin/users/"+bob.ID.String()+"/achievement-delegations", adminToken,
		models.AchievementDelegationCreationRequest{AchievementID: uuid.MustParse("7f2d49ad-d056-415e-aa80-0ada6db7cc00")})
	h.expect(http.StatusBadRequest, "POST", "/api/admin/users/nope/achievement-delegations", adminToken,
		models.AchievementDelegationCreationRequest{AchievementID: uuid.New()})
}

// TestSingleWinnerIsAssignedDirectly covers the branch where exactly one member met the
// goal: the debt is issued with that winner already set, so there is nothing to spin.
func TestSingleWinnerIsAssignedDirectly(t *testing.T) {
	h := newAPIHarness(t)
	withControllerSMTP(t)

	_, adminToken := h.user("admin@single.test", true)
	winner, winnerToken := h.user("winner@single.test", false)
	loser, loserToken := h.user("loser@single.test", false)

	season := seedOngoingSeason(t, "Running season", 2, 5)
	seedGoal(t, winner.ID, season, 1)
	seedGoal(t, loser.ID, season, 3)
	seedExerciseDayWithExercises(t, winner.ID, wednesdayOfWeeksAgo(1), 1)

	h.ok("POST", "/api/admin/debts", adminToken, models.DebtCreationRequest{Date: wednesdayOfWeeksAgo(1), TargetUser: &loser.ID})

	if unchosen := h.ok("GET", "/api/auth/debts/unchosen", loserToken, nil); unchosen["debt"] != nil {
		t.Errorf("a pre-assigned debt shows up as unchosen: %v", unchosen)
	}
	overview := h.ok("GET", "/api/auth/debts", winnerToken, nil)
	if overview["overview"] == nil {
		t.Errorf("winner's overview is empty: %v", overview)
	}

	// Season-level reads now have a week of results behind them.
	seasonPath := "/api/auth/seasons/" + season.ID.String()
	h.ok("GET", seasonPath+"/leaderboard", winnerToken, nil)
	h.ok("GET", seasonPath+"/weeks", winnerToken, nil)
	h.ok("GET", seasonPath+"/weeks-personal", loserToken, nil)
}

// A winner whose account no longer resolves is shown as "Deleted" rather than failing the
// whole conversion. (The fallback used to dereference the never-preloaded Winner
// association and panic.)
func TestConvertDebtWithMissingWinnerFallsBackToDeletedUser(t *testing.T) {
	newControllerTestDB(t)

	loser := makeControllerTestUser(t, "loser@convert.test")
	season := seedOngoingSeason(t, "Convert season", 1, 4)
	missingWinner := uuid.New()

	debt := models.Debt{Date: time.Now(), SeasonID: season.ID, LoserID: loser.ID, WinnerID: &missingWinner, Enabled: true}
	debt.ID = uuid.New()

	object, err := ConvertDebtToDebtObject(debt)
	if err != nil {
		t.Fatalf("ConvertDebtToDebtObject: %v", err)
	}
	if object.Winner == nil || object.Winner.FirstName != "Deleted" {
		t.Errorf("winner = %+v, want the Deleted placeholder", object.Winner)
	}
}
