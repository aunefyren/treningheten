package controllers

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/internal/smtptest"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// Tests for the admin "Fix a week" tool and the week reconciliation behind it (and behind the
// Monday job): docs/admin-week-fix.md.

// mondayAchievementID is "Case of the Mondays", a multi-delegation weekly achievement given
// for a workout on a Monday — handy for checking that re-runs don't double weekly grants.
var mondayAchievementID = uuid.MustParse("47f04b1f-4e19-40fe-ace3-3afa18378751")

// weekFix is a running season with an admin, and last week as the week being fixed.
type weekFix struct {
	t          *testing.T
	h          *apiHarness
	adminToken string
	season     models.Season
	lastWeek   time.Time
	mail       *smtptest.Server
}

func newWeekFix(t *testing.T) *weekFix {
	t.Helper()
	h := newAPIHarness(t)
	mail := withControllerSMTP(t)
	_, adminToken := h.user("admin@weekfix.test", true)
	return &weekFix{
		t:          t,
		h:          h,
		adminToken: adminToken,
		// Started two weeks ago, six weeks long: last week is a finished, mid-season week.
		season:   seedOngoingSeason(t, "Week fix season", 2, 6),
		lastWeek: wednesdayOfWeeksAgo(1),
		mail:     mail,
	}
}

// member joins the season with the given weekly goal and logs workouts on last Wednesday.
func (f *weekFix) member(email string, goal int, workouts int) (models.User, string, models.Goal) {
	f.t.Helper()
	user, token := f.h.user(email, false)
	seededGoal := seedGoal(f.t, user.ID, f.season, goal)
	if workouts > 0 {
		seedExerciseDayWithExercises(f.t, user.ID, f.lastWeek, workouts)
	}
	return user, token, seededGoal
}

func (f *weekFix) day() string {
	return f.lastWeek.Format("2006-01-02")
}

// reconcile runs POST /api/admin/weeks/reconcile for last week and returns the season's report.
func (f *weekFix) reconcile(dryRun bool, notify bool) models.WeekReconcileReport {
	f.t.Helper()
	reports := decodeReports(f.t, f.h.do("POST", "/api/admin/weeks/reconcile", f.adminToken,
		models.WeekReconcileRequest{Date: f.day(), DryRun: dryRun, Notify: notify}).Body.Bytes())
	return f.only(reports)
}

// overview runs GET /api/admin/weeks for last week and returns the season's report.
func (f *weekFix) overview() models.WeekReconcileReport {
	f.t.Helper()
	recorder := f.h.do("GET", "/api/admin/weeks?date="+f.day(), f.adminToken, nil)
	if recorder.Code != http.StatusOK {
		f.t.Fatalf("GET /api/admin/weeks: status %d, body %s", recorder.Code, recorder.Body.String())
	}
	return f.only(decodeReports(f.t, recorder.Body.Bytes()))
}

func (f *weekFix) only(reports []models.WeekReconcileReport) models.WeekReconcileReport {
	f.t.Helper()
	for _, report := range reports {
		if report.SeasonID == f.season.ID {
			return report
		}
	}
	f.t.Fatalf("no report for the season in %+v", reports)
	return models.WeekReconcileReport{}
}

func decodeReports(t *testing.T, body []byte) []models.WeekReconcileReport {
	t.Helper()
	var decoded struct {
		Error   string                       `json:"error"`
		Reports []models.WeekReconcileReport `json:"reports"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("invalid reconcile response %s: %v", body, err)
	}
	if decoded.Error != "" {
		t.Fatalf("reconcile answered an error: %s", body)
	}
	return decoded.Reports
}

func entryFor(t *testing.T, report models.WeekReconcileReport, userID uuid.UUID) models.WeekReconcileUser {
	t.Helper()
	for _, entry := range report.Users {
		if entry.User.ID == userID {
			return entry
		}
	}
	t.Fatalf("no report entry for user %s in %+v", userID, report.Users)
	return models.WeekReconcileUser{}
}

// debtsOf returns the user's enabled debts in the season, oldest first.
func debtsOf(t *testing.T, seasonID uuid.UUID, loserID uuid.UUID) []models.Debt {
	t.Helper()
	debts := []models.Debt{}
	if err := database.Instance.Where("debts.enabled = ?", true).Where("debts.season_id = ?", seasonID).
		Where("debts.loser_id = ?", loserID).Order("debts.created_at asc").Find(&debts).Error; err != nil {
		t.Fatal(err)
	}
	return debts
}

// onlyDebt fails unless the user has exactly one enabled debt, and returns it.
func onlyDebt(t *testing.T, seasonID uuid.UUID, loserID uuid.UUID) models.Debt {
	t.Helper()
	debts := debtsOf(t, seasonID, loserID)
	if len(debts) != 1 {
		t.Fatalf("user %s has %d enabled debts, want 1: %+v", loserID, len(debts), debts)
	}
	return debts[0]
}

func delegationCount(t *testing.T, userID uuid.UUID, achievementID uuid.UUID) int64 {
	t.Helper()
	var count int64
	if err := database.Instance.Model(&models.AchievementDelegation{}).Where("enabled = ?", true).
		Where("user_id = ?", userID).Where("achievement_id = ?", achievementID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}

// rowCounts is every debt and delegation row (enabled or not) and their enabled counts: the
// fingerprint a no-op run must leave untouched.
type rowCounts struct {
	debts, enabledDebts, delegations, enabledDelegations, wheelviews int64
}

func countRows(t *testing.T) rowCounts {
	t.Helper()
	counts := rowCounts{}
	database.Instance.Model(&models.Debt{}).Count(&counts.debts)
	database.Instance.Model(&models.Debt{}).Where("enabled = ?", true).Count(&counts.enabledDebts)
	database.Instance.Model(&models.AchievementDelegation{}).Count(&counts.delegations)
	database.Instance.Model(&models.AchievementDelegation{}).Where("enabled = ?", true).Count(&counts.enabledDelegations)
	database.Instance.Model(&models.Wheelview{}).Count(&counts.wheelviews)
	return counts
}

// winnerOf reads the debt's winner, failing on a wheel debt.
func winnerOf(t *testing.T, debt models.Debt) uuid.UUID {
	t.Helper()
	if debt.WinnerID == nil {
		t.Fatalf("debt %s has no winner", debt.ID)
	}
	return *debt.WinnerID
}

// stopCounting switches off every workout the user logged, as if they deleted them.
func stopCounting(t *testing.T, userID uuid.UUID) {
	t.Helper()
	days := database.Instance.Model(&models.ExerciseDay{}).Select("id").Where("user_id = ?", userID)
	if err := database.Instance.Model(&models.Exercise{}).Where("exercise_day_id IN (?)", days).Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}
}

// insertDebt stores a debt directly, as old data or a race would have left it.
func insertDebt(t *testing.T, seasonID uuid.UUID, loserID uuid.UUID, winnerID *uuid.UUID, date time.Time, paid bool, createdAt time.Time) models.Debt {
	t.Helper()
	debt := models.Debt{Date: date, SeasonID: seasonID, LoserID: loserID, WinnerID: winnerID, Paid: paid, Enabled: true}
	debt.ID = uuid.New()
	debt.CreatedAt = createdAt
	if err := database.Instance.Omit("Season", "Loser", "Winner").Create(&debt).Error; err != nil {
		t.Fatal(err)
	}
	return debt
}

func hasChange(entry models.WeekReconcileUser, fragment string) bool {
	for _, change := range entry.Changes {
		if strings.Contains(change, fragment) {
			return true
		}
	}
	return false
}

func hasWarning(warnings []string, fragment string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, fragment) {
			return true
		}
	}
	return false
}

// assertNoOp runs the reconciliation again and checks it changes nothing at all.
func (f *weekFix) assertNoOp() {
	f.t.Helper()
	before := countRows(f.t)
	mailBefore := len(f.mail.Messages())
	report := f.reconcile(false, true)
	if report.Changed {
		f.t.Errorf("a second run changed something: %+v", report.Users)
	}
	if after := countRows(f.t); after != before {
		f.t.Errorf("a second run changed rows: before %+v, after %+v", before, after)
	}
	if mails := len(f.mail.Messages()); mails != mailBefore {
		f.t.Errorf("a second run sent %d mails", mails-mailBefore)
	}
}

// TestWeekFixForgotToLog is the issue's main flow: a member forgot to log, so the Monday job
// gave them a debt to the week's only winner. The admin adds the workout, previews the fix
// (which writes nothing), applies it — the debt and its achievements go — and a second run is
// a no-op.
func TestWeekFixForgotToLog(t *testing.T) {
	f := newWeekFix(t)
	winner, _, _ := f.member("winner@weekfix.test", 1, 1)
	forgot, _, _ := f.member("forgot@weekfix.test", 1, 0)

	if err := ProcessWeekOfSeason(f.season, f.lastWeek); err != nil {
		t.Fatal(err)
	}
	debt := onlyDebt(t, f.season.ID, forgot.ID)
	if winnerOf(t, debt) != winner.ID {
		t.Fatalf("debt winner = %v, want the only winner", debt.WinnerID)
	}
	if delegationCount(t, winner.ID, debtWinnerAchievementID) != 1 || delegationCount(t, forgot.ID, debtLoserAchievementID) != 1 {
		t.Fatal("the single-winner debt did not come with its two achievements")
	}

	// The Monday job (or the admin) running again leaves everything as it is.
	f.assertNoOp()

	// The admin adds the forgotten workout to last Tuesday.
	tuesday := f.lastWeek.AddDate(0, 0, -1).Format("2006-01-02")
	created := f.h.expect(http.StatusCreated, "POST", "/api/admin/users/"+forgot.ID.String()+"/exercises", f.adminToken,
		models.AdminExerciseCreationRequest{Date: tuesday})
	if note := field(t, created, "exercise", "note"); note != adminAddedWorkoutNote {
		t.Errorf("note = %v, want the default %q", note, adminAddedWorkoutNote)
	}

	// The overview is a dry run: it shows the fix without making it.
	before := countRows(t)
	preview := f.overview()
	if !preview.DryRun || !preview.Changed {
		t.Fatalf("preview = %+v, want a dry run with changes", preview)
	}
	entry := entryFor(t, preview, forgot.ID)
	if !entry.WonWeek || entry.OwesSpin || entry.Workouts != 1 || entry.DebtBefore == nil || entry.DebtAfter != nil {
		t.Errorf("preview entry = %+v, want a won week whose debt goes", entry)
	}
	if !hasChange(entry, "Removed the debt") || len(entry.AchievementsRevoked) != 1 {
		t.Errorf("preview entry changes = %v, revoked = %v", entry.Changes, entry.AchievementsRevoked)
	}
	if len(entryFor(t, preview, winner.ID).AchievementsRevoked) != 1 {
		t.Error("the preview does not revoke the winner's debt achievement")
	}
	if after := countRows(t); after != before {
		t.Fatalf("the preview wrote rows: before %+v, after %+v", before, after)
	}

	// Apply.
	report := f.reconcile(false, false)
	if report.DryRun || !report.Changed || report.Losers != 0 || report.Winners != 2 {
		t.Errorf("report = %+v, want an applied fix with two winners and no losers", report)
	}
	if debts := debtsOf(t, f.season.ID, forgot.ID); len(debts) != 0 {
		t.Errorf("the forgotten week still has debts: %+v", debts)
	}
	if delegationCount(t, winner.ID, debtWinnerAchievementID) != 0 || delegationCount(t, forgot.ID, debtLoserAchievementID) != 0 {
		t.Error("the debt's achievements were not revoked")
	}

	f.assertNoOp()
}

// TestWeekFixRipple covers the case from the planning discussion: L and U lost to W, the only
// winner. Fixing U makes U a winner too, so L's debt is no longer W's — it goes to the wheel,
// and W's two debt achievements and L's go with the old debts.
func TestWeekFixRipple(t *testing.T) {
	f := newWeekFix(t)
	w, _, _ := f.member("w@ripple.test", 1, 1)
	u, _, _ := f.member("u@ripple.test", 1, 0)
	l, lToken, _ := f.member("l@ripple.test", 1, 0)

	f.reconcile(false, false)
	if winnerOf(t, onlyDebt(t, f.season.ID, l.ID)) != w.ID || winnerOf(t, onlyDebt(t, f.season.ID, u.ID)) != w.ID {
		t.Fatal("both losers should owe the only winner")
	}
	if got := delegationCount(t, w.ID, debtWinnerAchievementID); got != 2 {
		t.Fatalf("winner holds %d debt achievements, want one per debt (2)", got)
	}

	seedExerciseDayWithExercises(t, u.ID, f.lastWeek, 1)
	report := f.reconcile(false, false)

	if debts := debtsOf(t, f.season.ID, u.ID); len(debts) != 0 {
		t.Errorf("U still has a debt: %+v", debts)
	}
	lDebt := onlyDebt(t, f.season.ID, l.ID)
	if lDebt.WinnerID != nil {
		t.Errorf("L's debt winner = %v, want the wheel (nil) now that there are two winners", lDebt.WinnerID)
	}
	if !hasChange(entryFor(t, report, l.ID), "Handed the debt to the wheel") {
		t.Errorf("L's changes = %v", entryFor(t, report, l.ID).Changes)
	}
	if got := delegationCount(t, w.ID, debtWinnerAchievementID); got != 0 {
		t.Errorf("W keeps %d debt achievements, want 0", got)
	}
	if delegationCount(t, l.ID, debtLoserAchievementID) != 0 || delegationCount(t, u.ID, debtLoserAchievementID) != 0 {
		t.Error("the losers' debt achievements were not revoked")
	}

	f.assertNoOp()

	// L can now spin the wheel for it.
	unchosen := f.h.ok("GET", "/api/auth/debts/unchosen", lToken, nil)
	if debts, _ := unchosen["debt"].([]any); len(debts) != 1 {
		t.Errorf("L's unchosen debts = %v, want the wheel debt", unchosen)
	}
}

// TestWeekFixRippleTowardsOneWinner is the reverse ripple: a second winner's workouts turn
// out not to count, so the wheel debt now belongs to the remaining winner, and the former
// winner owes them too.
func TestWeekFixRippleTowardsOneWinner(t *testing.T) {
	f := newWeekFix(t)
	w1, _, _ := f.member("w1@reverse.test", 1, 1)
	w2, _, _ := f.member("w2@reverse.test", 1, 1)
	l, _, _ := f.member("l@reverse.test", 1, 0)

	f.reconcile(false, false)
	if onlyDebt(t, f.season.ID, l.ID).WinnerID != nil {
		t.Fatal("with two winners the debt should be the wheel's")
	}

	stopCounting(t, w2.ID)
	report := f.reconcile(false, false)

	if winnerOf(t, onlyDebt(t, f.season.ID, l.ID)) != w1.ID {
		t.Error("L's debt should go to the only winner left")
	}
	if winnerOf(t, onlyDebt(t, f.season.ID, w2.ID)) != w1.ID {
		t.Error("the former winner should now owe the only winner")
	}
	if !hasChange(entryFor(t, report, l.ID), "Moved the debt") || !hasChange(entryFor(t, report, w2.ID), "Added a debt") {
		t.Errorf("changes: L %v, W2 %v", entryFor(t, report, l.ID).Changes, entryFor(t, report, w2.ID).Changes)
	}
	if delegationCount(t, w1.ID, debtWinnerAchievementID) != 2 || delegationCount(t, l.ID, debtLoserAchievementID) != 1 || delegationCount(t, w2.ID, debtLoserAchievementID) != 1 {
		t.Error("the single-winner debt achievements were not granted for both debts")
	}

	f.assertNoOp()
}

// TestWeekFixSwapsTheOnlyWinner: in one fix the only winner's workouts stop counting and a
// loser's are added, so the new only winner takes over every debt. The loser who stays a loser
// keeps one "What you deserve"; the old winner loses both "Lucky bastard"s.
func TestWeekFixSwapsTheOnlyWinner(t *testing.T) {
	f := newWeekFix(t)
	oldWinner, _, _ := f.member("old@swap.test", 1, 1)
	newWinner, _, _ := f.member("new@swap.test", 1, 0)
	loser, _, _ := f.member("loser@swap.test", 1, 0)

	f.reconcile(false, false)

	stopCounting(t, oldWinner.ID)
	seedExerciseDayWithExercises(t, newWinner.ID, f.lastWeek, 1)
	f.reconcile(false, false)

	if winnerOf(t, onlyDebt(t, f.season.ID, loser.ID)) != newWinner.ID {
		t.Error("the loser's debt should move to the new only winner")
	}
	if winnerOf(t, onlyDebt(t, f.season.ID, oldWinner.ID)) != newWinner.ID {
		t.Error("the old winner should owe the new one")
	}
	if debts := debtsOf(t, f.season.ID, newWinner.ID); len(debts) != 0 {
		t.Errorf("the new winner still owes: %+v", debts)
	}

	checks := []struct {
		name          string
		user          uuid.UUID
		achievementID uuid.UUID
		want          int64
	}{
		{"old winner's wins", oldWinner.ID, debtWinnerAchievementID, 0},
		{"new winner's wins", newWinner.ID, debtWinnerAchievementID, 2},
		{"new winner's old loss", newWinner.ID, debtLoserAchievementID, 0},
		{"remaining loser's loss", loser.ID, debtLoserAchievementID, 1},
		{"old winner's new loss", oldWinner.ID, debtLoserAchievementID, 1},
	}
	for _, check := range checks {
		if got := delegationCount(t, check.user, check.achievementID); got != check.want {
			t.Errorf("%s: %d delegations, want %d", check.name, got, check.want)
		}
	}

	f.assertNoOp()
}

// TestWeekFixLeavesSettledDebtsAlone: a paid or spun debt is never changed, however the week
// turns out now; the report warns instead.
func TestWeekFixLeavesSettledDebtsAlone(t *testing.T) {
	t.Run("paid single-winner debt", func(t *testing.T) {
		f := newWeekFix(t)
		_, winnerToken, _ := f.member("winner@paid.test", 1, 1)
		loser, _, _ := f.member("loser@paid.test", 1, 0)
		f.reconcile(false, false)
		debt := onlyDebt(t, f.season.ID, loser.ID)
		f.h.ok("POST", "/api/auth/debts/"+debt.ID.String()+"/received", winnerToken, nil)

		seedExerciseDayWithExercises(t, loser.ID, f.lastWeek, 1)
		before := countRows(t)
		report := f.reconcile(false, false)

		entry := entryFor(t, report, loser.ID)
		if !hasWarning(entry.Warnings, "already paid") || entry.DebtAfter == nil || !entry.DebtAfter.Settled {
			t.Errorf("entry = %+v, want the paid debt kept with a warning", entry)
		}
		if report.Changed {
			t.Error("a warning alone is not a change")
		}
		if after := countRows(t); after != before {
			t.Errorf("rows changed: before %+v, after %+v", before, after)
		}
		if onlyDebt(t, f.season.ID, loser.ID).ID != debt.ID {
			t.Error("the paid debt was replaced")
		}
	})

	t.Run("spun wheel debt the week no longer calls for", func(t *testing.T) {
		f := newWeekFix(t)
		f.member("w1@spun.test", 1, 1)
		f.member("w2@spun.test", 1, 1)
		loser, loserToken, _ := f.member("loser@spun.test", 1, 0)
		f.reconcile(false, false)
		debt := onlyDebt(t, f.season.ID, loser.ID)
		f.h.ok("POST", "/api/auth/debts/"+debt.ID.String()+"/choose", loserToken, nil)

		seedExerciseDayWithExercises(t, loser.ID, f.lastWeek, 1)
		before := countRows(t)
		report := f.reconcile(false, false)

		entry := entryFor(t, report, loser.ID)
		if !hasWarning(entry.Warnings, "already spun") || entry.DebtBefore == nil || !entry.DebtBefore.Spun {
			t.Errorf("entry = %+v, want the spun debt kept with a warning", entry)
		}
		if after := countRows(t); after != before {
			t.Errorf("rows changed: before %+v, after %+v", before, after)
		}
	})

	t.Run("spun debt whose winner no longer won", func(t *testing.T) {
		f := newWeekFix(t)
		w1, _, _ := f.member("w1@spunlost.test", 1, 1)
		w2, _, _ := f.member("w2@spunlost.test", 1, 1)
		loser, loserToken, _ := f.member("loser@spunlost.test", 1, 0)
		f.reconcile(false, false)
		debt := onlyDebt(t, f.season.ID, loser.ID)
		f.h.ok("POST", "/api/auth/debts/"+debt.ID.String()+"/choose", loserToken, nil)
		spunWinner := winnerOf(t, onlyDebt(t, f.season.ID, loser.ID))
		otherWinner := w1.ID
		if spunWinner == w1.ID {
			otherWinner = w2.ID
		}

		stopCounting(t, spunWinner)
		report := f.reconcile(false, false)

		if !hasWarning(entryFor(t, report, loser.ID).Warnings, "different result") {
			t.Errorf("loser's warnings = %v", entryFor(t, report, loser.ID).Warnings)
		}
		if winnerOf(t, onlyDebt(t, f.season.ID, loser.ID)) != spunWinner {
			t.Error("the spun debt was re-assigned")
		}
		// The spun winner now lost the week and owes the remaining winner.
		if winnerOf(t, onlyDebt(t, f.season.ID, spunWinner)) != otherWinner {
			t.Error("the former winner should owe the only winner left")
		}
		f.assertNoOp()
	})

	t.Run("spun debt that still matches", func(t *testing.T) {
		f := newWeekFix(t)
		f.member("w1@spunok.test", 1, 1)
		f.member("w2@spunok.test", 1, 1)
		loser, loserToken, _ := f.member("loser@spunok.test", 1, 0)
		f.reconcile(false, false)
		debt := onlyDebt(t, f.season.ID, loser.ID)
		f.h.ok("POST", "/api/auth/debts/"+debt.ID.String()+"/choose", loserToken, nil)

		report := f.reconcile(false, false)
		entry := entryFor(t, report, loser.ID)
		if len(entry.Warnings) != 0 || len(entry.Changes) != 0 {
			t.Errorf("a spun debt that matches the week should be silent: %+v", entry)
		}
	})
}

// TestWeekFixCollapsesDuplicateDebts: old bugs and races left some weeks with several debts
// for one loser. The fix keeps one — a settled one if there is one, else the oldest.
func TestWeekFixCollapsesDuplicateDebts(t *testing.T) {
	t.Run("unsettled duplicates keep the oldest", func(t *testing.T) {
		f := newWeekFix(t)
		f.member("w1@dupe.test", 1, 1)
		f.member("w2@dupe.test", 1, 1)
		loser, _, _ := f.member("loser@dupe.test", 1, 0)
		oldest := insertDebt(t, f.season.ID, loser.ID, nil, f.lastWeek, false, time.Now().Add(-3*time.Hour))
		insertDebt(t, f.season.ID, loser.ID, nil, f.lastWeek, false, time.Now().Add(-2*time.Hour))
		insertDebt(t, f.season.ID, loser.ID, nil, f.lastWeek.AddDate(0, 0, 2), false, time.Now().Add(-time.Hour))

		// The single-debt lookup used to report "no debt" when more than one matched.
		if found, ok, err := database.GetDebtForWeekForUserInSeasonID(f.lastWeek, loser.ID, f.season.ID); err != nil || !ok || found.ID != oldest.ID {
			t.Errorf("GetDebtForWeekForUserInSeasonID = %v, %v, %v; want the oldest", found.ID, ok, err)
		}

		report := f.reconcile(false, false)
		if kept := onlyDebt(t, f.season.ID, loser.ID); kept.ID != oldest.ID {
			t.Errorf("kept debt %s, want the oldest %s", kept.ID, oldest.ID)
		}
		entry := entryFor(t, report, loser.ID)
		removed := 0
		for _, change := range entry.Changes {
			if strings.Contains(change, "duplicate") {
				removed++
			}
		}
		if removed != 2 {
			t.Errorf("changes = %v, want two duplicates removed", entry.Changes)
		}
		f.assertNoOp()
	})

	t.Run("a paid duplicate is the one kept", func(t *testing.T) {
		f := newWeekFix(t)
		winner, _, _ := f.member("winner@dupepaid.test", 1, 1)
		loser, _, _ := f.member("loser@dupepaid.test", 1, 0)
		insertDebt(t, f.season.ID, loser.ID, &winner.ID, f.lastWeek, false, time.Now().Add(-2*time.Hour))
		paid := insertDebt(t, f.season.ID, loser.ID, &winner.ID, f.lastWeek, true, time.Now().Add(-time.Hour))

		f.reconcile(false, false)
		if kept := onlyDebt(t, f.season.ID, loser.ID); kept.ID != paid.ID {
			t.Errorf("kept debt %s, want the paid one %s", kept.ID, paid.ID)
		}
		f.assertNoOp()
	})

	t.Run("two settled debts are both kept, with a warning", func(t *testing.T) {
		f := newWeekFix(t)
		winner, _, _ := f.member("winner@twopaid.test", 1, 1)
		loser, _, _ := f.member("loser@twopaid.test", 1, 0)
		insertDebt(t, f.season.ID, loser.ID, &winner.ID, f.lastWeek, true, time.Now().Add(-2*time.Hour))
		insertDebt(t, f.season.ID, loser.ID, &winner.ID, f.lastWeek, true, time.Now().Add(-time.Hour))

		report := f.reconcile(false, false)
		if got := len(debtsOf(t, f.season.ID, loser.ID)); got != 2 {
			t.Errorf("%d debts left, want both paid ones", got)
		}
		if !hasWarning(entryFor(t, report, loser.ID).Warnings, "more than one debt") {
			t.Errorf("warnings = %v", entryFor(t, report, loser.ID).Warnings)
		}
	})
}

// TestWeekFixRemovesDebtsTheWeekNoLongerCallsFor covers each reason a stored, unsettled debt
// can stop being owed.
func TestWeekFixRemovesDebtsTheWeekNoLongerCallsFor(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, f *weekFix, loser models.User, goal models.Goal)
	}{
		{"sick leave used for the week", func(t *testing.T, f *weekFix, loser models.User, goal models.Goal) {
			leave := models.Sickleave{GoalID: goal.ID, Used: true, Date: f.lastWeek, Enabled: true}
			leave.ID = uuid.New()
			if err := database.Instance.Omit("Goal").Create(&leave).Error; err != nil {
				t.Fatal(err)
			}
		}},
		{"no longer competing", func(t *testing.T, f *weekFix, loser models.User, goal models.Goal) {
			if err := database.Instance.Model(&models.Goal{}).Where("id = ?", goal.ID).Update("competing", false).Error; err != nil {
				t.Fatal(err)
			}
		}},
		{"joined during the week", func(t *testing.T, f *weekFix, loser models.User, goal models.Goal) {
			if err := database.Instance.Model(&models.Goal{}).Where("id = ?", goal.ID).Update("created_at", f.lastWeek).Error; err != nil {
				t.Fatal(err)
			}
		}},
		{"workouts added", func(t *testing.T, f *weekFix, loser models.User, goal models.Goal) {
			seedExerciseDayWithExercises(t, loser.ID, f.lastWeek, 1)
		}},
		{"made up with saved workouts", func(t *testing.T, f *weekFix, loser models.User, goal models.Goal) {
			database.Instance.Model(&models.Season{}).Where("id = ?", f.season.ID).Updates(map[string]any{"carry_over_max": 2, "carry_over_weeks": 2})
			f.season.CarryOverMax, f.season.CarryOverWeeks = 2, 2
			// Two in the week before (goal 1: one saved), then opt in to spend it.
			seedExerciseDayWithExercises(t, loser.ID, wednesdayOfWeeksAgo(2), 2)
			seedCarryOverUse(t, goal.ID, f.lastWeek)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newWeekFix(t)
			winner, _, _ := f.member("winner@stale.test", 1, 1)
			loser, _, goal := f.member("loser@stale.test", 1, 0)
			f.reconcile(false, false)
			if winnerOf(t, onlyDebt(t, f.season.ID, loser.ID)) != winner.ID {
				t.Fatal("setup: the loser should owe the winner")
			}

			test.setup(t, f, loser, goal)
			report := f.reconcile(false, false)

			if debts := debtsOf(t, f.season.ID, loser.ID); len(debts) != 0 {
				t.Errorf("debt kept: %+v (entry %+v)", debts, entryFor(t, report, loser.ID))
			}
			if delegationCount(t, winner.ID, debtWinnerAchievementID) != 0 || delegationCount(t, loser.ID, debtLoserAchievementID) != 0 {
				t.Error("the debt's achievements were not revoked")
			}
			f.assertNoOp()
		})
	}
}

// TestWeekFixNoWinnersMeansNoDebts: when nobody met the goal nobody owes anyone, and a debt
// left over from before (a winner whose workouts no longer count) is removed.
func TestWeekFixNoWinnersMeansNoDebts(t *testing.T) {
	f := newWeekFix(t)
	onlyWinner, _, _ := f.member("winner@nowin.test", 1, 1)
	a, _, _ := f.member("a@nowin.test", 1, 0)
	b, _, _ := f.member("b@nowin.test", 1, 0)
	f.reconcile(false, false)

	stopCounting(t, onlyWinner.ID)
	report := f.reconcile(false, false)
	if report.Winners != 0 {
		t.Fatalf("winners = %d, want 0", report.Winners)
	}
	for _, user := range []models.User{onlyWinner, a, b} {
		if debts := debtsOf(t, f.season.ID, user.ID); len(debts) != 0 {
			t.Errorf("%s has debts in a week nobody won: %+v", user.Email, debts)
		}
	}
	f.assertNoOp()
}

// TestWeekFixDryRunWritesNothing: every kind of change, previewed — no row and no mail.
func TestWeekFixDryRunWritesNothing(t *testing.T) {
	f := newWeekFix(t)
	w, _, _ := f.member("w@dry.test", 1, 1)
	u, _, _ := f.member("u@dry.test", 1, 0)
	l, _, _ := f.member("l@dry.test", 1, 0)
	f.reconcile(false, false)
	insertDebt(t, f.season.ID, l.ID, &w.ID, f.lastWeek, false, time.Now())
	seedExerciseDayWithExercises(t, u.ID, f.lastWeek, 1)
	// A Monday workout earns a weekly achievement the preview should list, not give.
	seedExerciseDayWithExercises(t, w.ID, f.lastWeek.AddDate(0, 0, -2), 1)

	before := countRows(t)
	mails := len(f.mail.Messages())
	report := f.reconcile(true, true)

	if !report.DryRun || !report.Changed {
		t.Fatalf("report = %+v, want a dry run with changes", report)
	}
	if !hasChange(entryFor(t, report, u.ID), "Removed the debt") || !hasChange(entryFor(t, report, l.ID), "duplicate") ||
		!hasChange(entryFor(t, report, l.ID), "Handed the debt to the wheel") {
		t.Errorf("the preview misses changes: U %v, L %v", entryFor(t, report, u.ID).Changes, entryFor(t, report, l.ID).Changes)
	}
	granted := false
	for _, achievement := range entryFor(t, report, w.ID).AchievementsGranted {
		if achievement.AchievementID == mondayAchievementID {
			granted = true
		}
	}
	if !granted {
		t.Errorf("the preview does not list the Monday achievement: %+v", entryFor(t, report, w.ID).AchievementsGranted)
	}
	if after := countRows(t); after != before {
		t.Errorf("a dry run wrote rows: before %+v, after %+v", before, after)
	}
	if got := len(f.mail.Messages()); got != mails {
		t.Errorf("a dry run sent %d mails", got-mails)
	}
}

// TestWeekFixNotifications: an admin fix is silent unless "Notify affected users" is ticked;
// the Monday job always notifies.
func TestWeekFixNotifications(t *testing.T) {
	t.Run("silent by default", func(t *testing.T) {
		f := newWeekFix(t)
		f.member("winner@silent.test", 1, 1)
		loser, _, _ := f.member("loser@silent.test", 1, 0)
		f.reconcile(false, false)
		onlyDebt(t, f.season.ID, loser.ID)
		if got := len(f.mail.Messages()); got != 0 {
			t.Errorf("a silent fix sent %d mails", got)
		}
	})

	t.Run("notify sends the debt mails", func(t *testing.T) {
		f := newWeekFix(t)
		f.member("winner@loud.test", 1, 1)
		f.member("loser@loud.test", 1, 0)
		f.reconcile(false, true)
		// Week lost to the loser, wheel-spin win to the winner.
		if got := len(f.mail.Messages()); got != 2 {
			t.Errorf("sent %d mails, want 2", got)
		}
	})

	t.Run("the Monday job notifies", func(t *testing.T) {
		f := newWeekFix(t)
		f.member("winner@monday.test", 1, 1)
		f.member("loser@monday.test", 1, 0)
		if err := ProcessWeekOfSeason(f.season, f.lastWeek); err != nil {
			t.Fatal(err)
		}
		if got := len(f.mail.Messages()); got != 2 {
			t.Errorf("sent %d mails, want 2", got)
		}
	})
}

// TestWeekFixWeeklyAchievementsAreNotDoubled: re-running a week used to hand out its
// multi-delegation achievements again on every run.
func TestWeekFixWeeklyAchievementsAreNotDoubled(t *testing.T) {
	f := newWeekFix(t)
	member, _, _ := f.member("member@double.test", 1, 0)
	seedExerciseDayWithExercises(t, member.ID, f.lastWeek.AddDate(0, 0, -2), 1) // last Monday

	for run := 0; run < 3; run++ {
		if err := ProcessWeekOfSeason(f.season, f.lastWeek); err != nil {
			t.Fatal(err)
		}
	}
	f.reconcile(false, true)
	if got := delegationCount(t, member.ID, mondayAchievementID); got != 1 {
		t.Errorf("Monday achievement given %d times, want once", got)
	}
}

// TestWeekFixSeasonsLastWeek: the last week also runs the season achievements, and those
// don't double on a re-run either.
func TestWeekFixSeasonsLastWeek(t *testing.T) {
	h := newAPIHarness(t)
	withControllerSMTP(t)
	_, adminToken := h.user("admin@lastweek.test", true)
	member, _ := h.user("member@lastweek.test", false)
	rival, _ := h.user("rival@lastweek.test", false)
	// One week long, ended last Sunday.
	season := seedOngoingSeason(t, "One week season", 1, 1)
	seedGoal(t, member.ID, season, 1)
	seedGoal(t, rival.ID, season, 1)
	seedExerciseDayWithExercises(t, member.ID, wednesdayOfWeeksAgo(1), 1)

	day := wednesdayOfWeeksAgo(1).Format("2006-01-02")
	h.ok("POST", "/api/admin/weeks/reconcile", adminToken, models.WeekReconcileRequest{Date: day})
	before := countRows(t)
	if before.enabledDelegations == 0 {
		t.Fatal("the season's last week granted no achievements at all")
	}
	reports := decodeReports(t, h.ok2("POST", "/api/admin/weeks/reconcile", adminToken, models.WeekReconcileRequest{Date: day}))
	if len(reports) != 1 || reports[0].Changed {
		t.Errorf("second run of the last week = %+v, want unchanged", reports)
	}
	if after := countRows(t); after != before {
		t.Errorf("rows changed on the second run: before %+v, after %+v", before, after)
	}
}

// ok2 is ok returning the raw body.
func (h *apiHarness) ok2(method, path, token string, body any) []byte {
	h.t.Helper()
	recorder := h.do(method, path, token, body)
	if recorder.Code != http.StatusOK {
		h.t.Fatalf("%s %s: status = %d; body: %s", method, path, recorder.Code, recorder.Body.String())
	}
	return recorder.Body.Bytes()
}

// TestWeekFixDebtOfSomeoneWhoLeft: a debt held by someone without a goal in the season any
// more is not the week's to judge; it is reported and left alone.
func TestWeekFixDebtOfSomeoneWhoLeft(t *testing.T) {
	f := newWeekFix(t)
	winner, _, _ := f.member("winner@left.test", 1, 1)
	leaver, _, goal := f.member("leaver@left.test", 1, 0)
	f.reconcile(false, false)
	debt := onlyDebt(t, f.season.ID, leaver.ID)

	if err := database.DisableGoalInDBUsingGoalID(goal.ID); err != nil {
		t.Fatal(err)
	}
	report := f.reconcile(false, false)

	if !hasWarning(report.Warnings, "no goal in the season") {
		t.Errorf("report warnings = %v", report.Warnings)
	}
	if onlyDebt(t, f.season.ID, leaver.ID).ID != debt.ID {
		t.Error("the leaver's debt was changed")
	}
	if delegationCount(t, winner.ID, debtWinnerAchievementID) != 1 {
		t.Error("the winner's achievement for it was touched")
	}
}

// TestWeekFixTwoSeasons: a week shared by two seasons is fixed in both, independently.
func TestWeekFixTwoSeasons(t *testing.T) {
	f := newWeekFix(t)
	other := seedOngoingSeason(t, "Other season", 3, 6)
	winner, _, _ := f.member("winner@two.test", 1, 1)
	loser, _, _ := f.member("loser@two.test", 1, 0)
	seedGoal(t, winner.ID, other, 1)
	seedGoal(t, loser.ID, other, 3) // misses here too

	reports := decodeReports(t, f.h.ok2("POST", "/api/admin/weeks/reconcile", f.adminToken, models.WeekReconcileRequest{Date: f.day()}))
	if len(reports) != 2 {
		t.Fatalf("got %d reports, want one per season", len(reports))
	}
	if len(debtsOf(t, f.season.ID, loser.ID)) != 1 || len(debtsOf(t, other.ID, loser.ID)) != 1 {
		t.Error("the loser should owe a debt in each season")
	}

	// One workout fixes the first season (goal 1) but not the other (goal 3).
	seedExerciseDayWithExercises(t, loser.ID, f.lastWeek, 1)
	f.h.ok("POST", "/api/admin/weeks/reconcile", f.adminToken, models.WeekReconcileRequest{Date: f.day()})
	if len(debtsOf(t, f.season.ID, loser.ID)) != 0 {
		t.Error("the debt in the season whose goal is now met should go")
	}
	if len(debtsOf(t, other.ID, loser.ID)) != 1 {
		t.Error("the debt in the season whose goal is still missed should stay")
	}
}

// TestWeekFixRequestValidation covers the endpoints' guards.
func TestWeekFixRequestValidation(t *testing.T) {
	f := newWeekFix(t)
	_, memberToken, _ := f.member("member@validate.test", 1, 0)
	thisWeek := time.Now().Format("2006-01-02")

	f.h.expect(http.StatusForbidden, "GET", "/api/admin/weeks?date="+f.day(), memberToken, nil)
	f.h.expect(http.StatusForbidden, "POST", "/api/admin/weeks/reconcile", memberToken, models.WeekReconcileRequest{Date: f.day()})
	f.h.expect(http.StatusBadRequest, "POST", "/api/admin/weeks/reconcile", f.adminToken, "{broken")
	f.h.expect(http.StatusBadRequest, "POST", "/api/admin/weeks/reconcile", f.adminToken, models.WeekReconcileRequest{})
	f.h.expect(http.StatusBadRequest, "POST", "/api/admin/weeks/reconcile", f.adminToken, models.WeekReconcileRequest{Date: "24.11.2026"})
	f.h.expect(http.StatusBadRequest, "GET", "/api/admin/weeks", f.adminToken, nil)
	f.h.expect(http.StatusBadRequest, "GET", "/api/admin/weeks?date=nope", f.adminToken, nil)

	// The running week isn't over: no debts can be judged yet.
	f.h.expect(http.StatusBadRequest, "GET", "/api/admin/weeks?date="+thisWeek, f.adminToken, nil)
	f.h.expect(http.StatusBadRequest, "POST", "/api/admin/weeks/reconcile", f.adminToken, models.WeekReconcileRequest{Date: thisWeek})
	if _, err := ReconcileWeekOfSeason(f.season, time.Now(), weekReconcileOptions{}); err != ErrWeekNotFinished {
		t.Errorf("reconciling the running week: err = %v, want ErrWeekNotFinished", err)
	}

	// Outside the season.
	if _, err := ReconcileWeekOfSeason(f.season, f.season.Start.AddDate(0, 0, -10), weekReconcileOptions{}); err != ErrNotInSeason {
		t.Errorf("reconciling before the season: err = %v, want ErrNotInSeason", err)
	}

	// A week no season ran: nothing to do.
	long := f.season.Start.AddDate(-1, 0, 0).Format("2006-01-02")
	answer := f.h.ok("GET", "/api/admin/weeks?date="+long, f.adminToken, nil)
	if answer["message"] != "No season ran that week." {
		t.Errorf("answer = %v", answer)
	}
	if reports, _ := answer["reports"].([]any); len(reports) != 0 {
		t.Errorf("reports = %v, want none", answer["reports"])
	}
}

// TestAdminAddWorkout covers POST /api/admin/users/:user_id/exercises.
func TestAdminAddWorkout(t *testing.T) {
	f := newWeekFix(t)
	member, memberToken, _ := f.member("member@addworkout.test", 2, 0)
	outsider, _ := f.h.user("outsider@addworkout.test", false)
	path := "/api/admin/users/" + member.ID.String() + "/exercises"

	// Creates the day and a counting session with the note.
	created := f.h.expect(http.StatusCreated, "POST", path, f.adminToken, models.AdminExerciseCreationRequest{Date: f.day(), Note: "  Logged   by the admin  "})
	if note := field(t, created, "exercise", "note"); note != "Logged by the admin" {
		t.Errorf("note = %q, want it trimmed and collapsed", note)
	}
	day, err := database.GetExerciseDayByUserIDAndDate(member.ID, f.lastWeek)
	if err != nil || day == nil {
		t.Fatalf("the day was not created: %v", err)
	}

	// A second one lands on the same day; the week's count follows.
	f.h.expect(http.StatusCreated, "POST", path, f.adminToken, models.AdminExerciseCreationRequest{Date: f.day()})
	if again, _ := database.GetExerciseDayByUserIDAndDate(member.ID, f.lastWeek); again == nil || again.ID != day.ID {
		t.Error("the second workout did not reuse the day")
	}
	if entry := entryFor(t, f.overview(), member.ID); entry.Workouts != 2 || !entry.GoalMet {
		t.Errorf("week entry = %+v, want two workouts and the goal met", entry)
	}

	// A day holds three counting workouts.
	f.h.expect(http.StatusCreated, "POST", path, f.adminToken, models.AdminExerciseCreationRequest{Date: f.day()})
	f.h.expect(http.StatusBadRequest, "POST", path, f.adminToken, models.AdminExerciseCreationRequest{Date: f.day()})

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	beforeSeason := f.season.Start.AddDate(0, 0, -3).Format("2006-01-02")
	tests := []struct {
		name   string
		path   string
		token  string
		body   any
		status int
	}{
		{"not an admin", path, memberToken, models.AdminExerciseCreationRequest{Date: f.day()}, http.StatusForbidden},
		{"broken body", path, f.adminToken, "{broken", http.StatusBadRequest},
		{"bad user id", "/api/admin/users/nope/exercises", f.adminToken, models.AdminExerciseCreationRequest{Date: f.day()}, http.StatusBadRequest},
		{"unknown user", "/api/admin/users/" + uuid.NewString() + "/exercises", f.adminToken, models.AdminExerciseCreationRequest{Date: f.day()}, http.StatusNotFound},
		{"missing date", path, f.adminToken, models.AdminExerciseCreationRequest{}, http.StatusBadRequest},
		{"bad date", path, f.adminToken, models.AdminExerciseCreationRequest{Date: "last tuesday"}, http.StatusBadRequest},
		{"future day", path, f.adminToken, models.AdminExerciseCreationRequest{Date: tomorrow}, http.StatusBadRequest},
		{"note too long", path, f.adminToken, models.AdminExerciseCreationRequest{Date: f.lastWeek.AddDate(0, 0, 1).Format("2006-01-02"), Note: strings.Repeat("a", 256)}, http.StatusBadRequest},
		{"no goal in a season that day", path, f.adminToken, models.AdminExerciseCreationRequest{Date: beforeSeason}, http.StatusBadRequest},
		{"user not in the season", "/api/admin/users/" + outsider.ID.String() + "/exercises", f.adminToken, models.AdminExerciseCreationRequest{Date: f.day()}, http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f.h.expect(test.status, "POST", test.path, test.token, test.body)
		})
	}
}

// TestAdminGrantSickleave covers POST /api/admin/users/:user_id/sickleave.
func TestAdminGrantSickleave(t *testing.T) {
	f := newWeekFix(t)
	member, memberToken := f.h.user("member@sick.test", false)
	// Join through the API so the season's allowance (2) is created as usual.
	f.h.expect(http.StatusCreated, "POST", "/api/auth/goals", memberToken, models.GoalCreationRequest{ExerciseInterval: 1, Competing: true, SeasonID: f.season.ID})
	outsider, _ := f.h.user("outsider@sick.test", false)
	path := "/api/admin/users/" + member.ID.String() + "/sickleave"

	granted := f.h.expect(http.StatusCreated, "POST", path, f.adminToken, models.AdminSickleaveGrantRequest{SeasonID: f.season.ID, Count: 3})
	if left := field(t, granted, "sickleave_left"); left != float64(5) {
		t.Errorf("sickleave_left = %v, want 2 + 3", left)
	}
	if entry := entryFor(t, f.overview(), member.ID); entry.SickleaveLeft != 5 {
		t.Errorf("the week overview shows %d weeks of sick leave left, want 5", entry.SickleaveLeft)
	}

	tests := []struct {
		name   string
		path   string
		token  string
		body   any
		status int
	}{
		{"not an admin", path, memberToken, models.AdminSickleaveGrantRequest{SeasonID: f.season.ID, Count: 1}, http.StatusForbidden},
		{"broken body", path, f.adminToken, "{broken", http.StatusBadRequest},
		{"bad user id", "/api/admin/users/nope/sickleave", f.adminToken, models.AdminSickleaveGrantRequest{SeasonID: f.season.ID, Count: 1}, http.StatusBadRequest},
		{"zero weeks", path, f.adminToken, models.AdminSickleaveGrantRequest{SeasonID: f.season.ID, Count: 0}, http.StatusBadRequest},
		{"negative weeks", path, f.adminToken, models.AdminSickleaveGrantRequest{SeasonID: f.season.ID, Count: -2}, http.StatusBadRequest},
		{"too many weeks", path, f.adminToken, models.AdminSickleaveGrantRequest{SeasonID: f.season.ID, Count: maxSickleaveGrant + 1}, http.StatusBadRequest},
		{"no season", path, f.adminToken, models.AdminSickleaveGrantRequest{Count: 1}, http.StatusBadRequest},
		{"unknown season", path, f.adminToken, models.AdminSickleaveGrantRequest{SeasonID: uuid.New(), Count: 1}, http.StatusBadRequest},
		{"user not in the season", "/api/admin/users/" + outsider.ID.String() + "/sickleave", f.adminToken, models.AdminSickleaveGrantRequest{SeasonID: f.season.ID, Count: 1}, http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f.h.expect(test.status, "POST", test.path, test.token, test.body)
		})
	}
}

// TestGrantAchievementOptions covers the once-per-day rule and dry runs behind the week
// processing's grants, and that one-off grants still stack.
func TestGrantAchievementOptions(t *testing.T) {
	newAPIHarness(t)
	user := makeControllerTestUser(t, "grant@options.test")
	sunday := time.Date(2026, 3, 8, 23, 59, 59, 0, time.Local)

	granted, _, err := grantAchievement(user.ID, mondayAchievementID, sunday, achievementGrantOptions{OncePerDay: true, DryRun: true})
	if err != nil || !granted {
		t.Fatalf("dry run: granted = %v, err = %v", granted, err)
	}
	if delegationCount(t, user.ID, mondayAchievementID) != 0 {
		t.Fatal("a dry run wrote a delegation")
	}

	for run := 0; run < 2; run++ {
		granted, _, err = grantAchievement(user.ID, mondayAchievementID, sunday, achievementGrantOptions{OncePerDay: true})
		if err != nil || granted != (run == 0) {
			t.Fatalf("run %d: granted = %v, err = %v", run, granted, err)
		}
	}
	// Same day, other hour: still the same day.
	if granted, _, _ = grantAchievement(user.ID, mondayAchievementID, sunday.Add(-20*time.Hour), achievementGrantOptions{OncePerDay: true}); granted {
		t.Error("granted twice on the same day")
	}
	// Another day is another grant.
	if granted, _, _ = grantAchievement(user.ID, mondayAchievementID, sunday.AddDate(0, 0, 7), achievementGrantOptions{OncePerDay: true}); !granted {
		t.Error("not granted for another day")
	}
	// One-off grants (wheel wins) still stack on one day.
	if err := GiveUserAnAchievement(user.ID, mondayAchievementID, sunday, 0); err != nil {
		t.Fatal(err)
	}
	if got := delegationCount(t, user.ID, mondayAchievementID); got != 3 {
		t.Errorf("%d delegations, want 3", got)
	}

	// Revoking takes back one same-day delegation, never another day's.
	revoked, _, err := revokeAchievementForDay(user.ID, mondayAchievementID, sunday.AddDate(0, 0, 1), false)
	if err != nil || revoked {
		t.Errorf("revoked a delegation from another day: %v, %v", revoked, err)
	}
	if revoked, _, err = revokeAchievementForDay(user.ID, mondayAchievementID, sunday, true); err != nil || !revoked {
		t.Errorf("dry-run revoke = %v, %v", revoked, err)
	}
	if got := delegationCount(t, user.ID, mondayAchievementID); got != 3 {
		t.Errorf("a dry-run revoke changed the count to %d", got)
	}
	if revoked, _, err = revokeAchievementForDay(user.ID, mondayAchievementID, sunday, false); err != nil || !revoked {
		t.Errorf("revoke = %v, %v", revoked, err)
	}
	if got := delegationCount(t, user.ID, mondayAchievementID); got != 2 {
		t.Errorf("%d delegations after a revoke, want 2", got)
	}
}
