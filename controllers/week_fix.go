package controllers

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"
	"github.com/aunefyren/treningheten/utilities"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Week reconciliation: the one place a finished season-week's debts are written. The Monday
// job runs it for last week, and the admin "Fix a week" tool runs it (or a dry run of it)
// after correcting a week. It compares what the week's results say should be stored with what
// is stored, and fixes the difference — so it is idempotent: a second run with no data change
// changes nothing. See docs/admin-week-fix.md.

// The two achievements tied to a single-winner debt: the winner's "Lucky bastard" and the
// loser's "What you deserve". A wheel spin hands out the same pair, at spin and view time.
var (
	debtWinnerAchievementID = uuid.MustParse("bb964360-6413-47c2-8400-ee87b40365a7")
	debtLoserAchievementID  = uuid.MustParse("d415fffc-ea99-4b27-8929-aeb02ae44da3")
)

// ErrDebtPartiallyGenerated means the week's results were computed and every participant was
// attempted, but the debt of at least one of them could not be checked or written. The
// achievements still ran, and the returned report is still valid for everyone else.
var ErrDebtPartiallyGenerated = errors.New("debt could not be generated for every loser")

// ErrWeekNotFinished: a week can only be reconciled once its Sunday is over.
var ErrWeekNotFinished = errors.New("the week is not over yet")

// ErrNotInSeason: the point in time is outside the season.
var ErrNotInSeason = errors.New("not in the middle of a season")

// weekReconcileMutex serialises reconciliations, so the Monday job and an admin (or two
// admins) can't both create the same missing debt.
var weekReconcileMutex sync.Mutex

type weekReconcileOptions struct {
	// DryRun works out and reports every change without writing or notifying anything.
	DryRun bool
	// Notify sends the usual e-mails and pushes for debts and achievements.
	Notify bool
}

// ProcessWeekOfSeason finalises a finished week of a season, as the Monday job does: debts,
// then the week's (and, in the season's last week, the season's) achievements, with
// notifications.
func ProcessWeekOfSeason(season models.Season, pointInTime time.Time) error {
	_, err := ReconcileWeekOfSeason(season, pointInTime, weekReconcileOptions{Notify: true})
	return err
}

// weekReconciliation is the state of one ReconcileWeekOfSeason run.
type weekReconciliation struct {
	options     weekReconcileOptions
	season      models.Season
	monday      time.Time
	sunday      time.Time
	weekNumber  int
	winners     []uuid.UUID
	report      *models.WeekReconcileReport
	entryByUser map[uuid.UUID]*models.WeekReconcileUser
}

// ReconcileWeekOfSeason brings the stored debts and achievements of the season-week holding
// pointInTime in line with the week's results.
//
// Desired debts: none when nobody won the week; otherwise every participant who owes a spin
// gets one debt, assigned to the winner when there is exactly one, or left to the wheel
// (winner nil) when there are several. Stored debts are then fixed per participant:
//   - missing → created (with the single-winner achievements);
//   - no longer owed → switched off, its single-winner achievements revoked;
//   - wrong winner → re-assigned (or handed back to the wheel), achievements moved with it;
//   - duplicates → all but one switched off.
//
// A settled debt — paid, or its wheel already spun — is never changed; a disagreement with it
// becomes a warning in the report for the admin to judge.
//
// Weekly achievements (and season achievements in the season's last week) are granted once
// per day per user, so a re-run never doubles them. They are not revoked.
func ReconcileWeekOfSeason(season models.Season, pointInTime time.Time, options weekReconcileOptions) (models.WeekReconcileReport, error) {
	weekReconcileMutex.Lock()
	defer weekReconcileMutex.Unlock()

	if season.Start.After(pointInTime) || season.End.Before(pointInTime) {
		return models.WeekReconcileReport{}, ErrNotInSeason
	}

	monday, err := utilities.FindEarlierMonday(pointInTime)
	if err != nil {
		return models.WeekReconcileReport{}, errors.New("Failed to find earliest point in the week.")
	}
	sunday, err := utilities.FindNextSunday(pointInTime)
	if err != nil {
		return models.WeekReconcileReport{}, errors.New("Failed to find latest point in the week.")
	}
	if sunday.After(time.Now()) {
		return models.WeekReconcileReport{}, ErrWeekNotFinished
	}

	seasonObject, err := ConvertSeasonToSeasonObject(season)
	if err != nil {
		logger.Log.Info("Failed to convert season to season object. Error: " + err.Error())
		return models.WeekReconcileReport{}, errors.New("Failed to convert season to season object.")
	}

	weeks, err := RetrieveWeekResultsFromSeasonWithinTimeframe(monday, sunday, seasonObject)
	if err != nil {
		logger.Log.Info("Failed to retrieve the week for season. Error: " + err.Error())
		return models.WeekReconcileReport{}, errors.New("Failed to retrieve the week for season.")
	} else if len(weeks) != 1 {
		return models.WeekReconcileReport{}, errors.New("Failed to retrieve ONE week for season.")
	}
	week := weeks[0]

	weekYear, weekNumber := monday.ISOWeek()
	report := models.WeekReconcileReport{
		SeasonID:   season.ID,
		SeasonName: season.Name,
		WeekYear:   weekYear,
		WeekNumber: weekNumber,
		WeekStart:  monday,
		WeekEnd:    sunday,
		DryRun:     options.DryRun,
		Users:      []models.WeekReconcileUser{},
		Warnings:   []string{},
	}

	goals := map[uuid.UUID]models.GoalObject{}
	for _, goal := range seasonObject.Goals {
		goals[goal.ID] = goal
	}

	run := &weekReconciliation{
		options:     options,
		season:      season,
		monday:      monday,
		sunday:      sunday,
		weekNumber:  weekNumber,
		report:      &report,
		entryByUser: map[uuid.UUID]*models.WeekReconcileUser{},
	}

	losers := []uuid.UUID{}
	for _, result := range week.UserWeekResults {
		if result.OwesSpin() {
			losers = append(losers, result.UserID)
		} else if result.WonWeek() {
			run.winners = append(run.winners, result.UserID)
		}
		report.Users = append(report.Users, newWeekReconcileUser(result, goals[result.GoalID]))
	}
	for index := range report.Users {
		run.entryByUser[report.Users[index].User.ID] = &report.Users[index]
	}
	report.Winners = len(run.winners)
	report.Losers = len(losers)

	var desiredWinner *uuid.UUID
	if len(run.winners) == 1 {
		desiredWinner = &run.winners[0]
	}

	storedDebts, err := database.GetDebtsForSeasonBetweenDates(season.ID, monday, sunday)
	if err != nil {
		logger.Log.Error("Failed to get the week's debts. Error: " + err.Error())
		return report, errors.New("Failed to get the week's debts.")
	}
	debtsByLoser := map[uuid.UUID][]models.Debt{}
	for _, debt := range storedDebts {
		debtsByLoser[debt.LoserID] = append(debtsByLoser[debt.LoserID], debt)
	}

	failures := []error{}
	for _, result := range week.UserWeekResults {
		entry := run.entryByUser[result.UserID]
		owes := result.OwesSpin() && len(run.winners) > 0
		if err := run.reconcileUserDebt(entry, debtsByLoser[result.UserID], owes, desiredWinner); err != nil {
			logger.Log.Error("Failed to reconcile debt for '" + result.UserID.String() + "'. Error: " + err.Error())
			entry.Warnings = append(entry.Warnings, "The debt could not be fixed: "+err.Error())
			failures = append(failures, fmt.Errorf("reconcile debt for user %s: %w", result.UserID, err))
		}
		delete(debtsByLoser, result.UserID)
	}

	// Debts of people without a goal in the season (they left, or were disabled) are not the
	// week's to judge any more; the admin decides.
	for loserID, debts := range debtsByLoser {
		user, err := database.GetUserInformationIncludingDisabled(loserID)
		name := loserID.String()
		if err == nil {
			name = user.FirstName + " " + user.LastName
		}
		report.Warnings = append(report.Warnings, fmt.Sprintf("%s has %d debt(s) this week but no goal in the season any more; left as is.", name, len(debts)))
	}

	grant := func(userID uuid.UUID, achievementID uuid.UUID, achievementTime time.Time) error {
		granted, achievement, err := grantAchievement(userID, achievementID, achievementTime, achievementGrantOptions{OncePerDay: true, Notify: options.Notify, DryRun: options.DryRun})
		if err != nil {
			return err
		}
		if granted {
			run.recordAchievement(userID, achievement, achievementTime, true)
		}
		return nil
	}

	if err := GenerateAchievementsForWeek(week, nil, grant); err != nil {
		logger.Log.Error("Failed to generate the week's achievements. Error: " + err.Error())
		return report, errors.New("Failed to generate the week's achievements.")
	}

	seasonEndYear, seasonEndWeek := season.End.ISOWeek()
	if weekYear == seasonEndYear && weekNumber == seasonEndWeek {
		pastWeeks, err := RetrieveWeekResultsFromSeasonWithinTimeframe(seasonObject.Start, sunday, seasonObject)
		if err != nil {
			logger.Log.Error("Failed to get season results. Error: " + err.Error())
			return report, errors.New("Failed to get season results.")
		}
		if err := GenerateAchievementsForSeason(pastWeeks, nil, grant); err != nil {
			logger.Log.Error("Failed to generate the season's achievements. Error: " + err.Error())
			return report, errors.New("Failed to generate the season's achievements.")
		}
	}

	for _, entry := range report.Users {
		if len(entry.Changes) > 0 || len(entry.AchievementsGranted) > 0 || len(entry.AchievementsRevoked) > 0 {
			report.Changed = true
		}
	}

	if len(failures) > 0 {
		return report, fmt.Errorf("%w: %w", ErrDebtPartiallyGenerated, errors.Join(failures...))
	}
	return report, nil
}

func newWeekReconcileUser(result models.UserWeekResults, goal models.GoalObject) models.WeekReconcileUser {
	user := goal.User
	user.ID = result.UserID
	return models.WeekReconcileUser{
		User:                  user,
		GoalID:                result.GoalID,
		Goal:                  goal.ExerciseInterval,
		Workouts:              int(math.Round(result.WeekCompletion * float64(goal.ExerciseInterval))),
		WeekCompletion:        result.WeekCompletion,
		Competing:             result.Competing,
		SickLeave:             result.SickLeave,
		FullWeekParticipation: result.FullWeekParticipation,
		CarryUsed:             result.CarryUsed,
		GoalMet:               result.GoalMet(),
		WonWeek:               result.WonWeek(),
		OwesSpin:              result.OwesSpin(),
		SickleaveLeft:         goal.SickleaveLeft,
		Changes:               []string{},
		AchievementsGranted:   []models.WeekReconcileAchievement{},
		AchievementsRevoked:   []models.WeekReconcileAchievement{},
		Warnings:              []string{},
	}
}

// storedDebt is a debt with whether its wheel has been spun.
type storedDebt struct {
	debt models.Debt
	spun bool
}

func (stored storedDebt) settled() bool {
	return stored.debt.Paid || stored.spun
}

// reconcileUserDebt fixes one participant's debts for the week. owes says whether they should
// have a debt; desiredWinner is its winner (nil: the wheel decides).
func (run *weekReconciliation) reconcileUserDebt(entry *models.WeekReconcileUser, debts []models.Debt, owes bool, desiredWinner *uuid.UUID) error {
	stored := []storedDebt{}
	for _, debt := range debts {
		spun, err := database.DebtHasWheelviews(debt.ID)
		if err != nil {
			return errors.New("failed to check the wheel for the debt")
		}
		stored = append(stored, storedDebt{debt: debt, spun: spun})
	}

	// The debt to keep: a settled one if any (it can't be changed anyway), else the oldest.
	keeper := -1
	for index, candidate := range stored {
		if candidate.settled() {
			keeper = index
			break
		}
	}
	if keeper == -1 && len(stored) > 0 {
		keeper = 0
	}
	if keeper >= 0 {
		entry.DebtBefore = run.debtState(stored[keeper])
		entry.DebtAfter = entry.DebtBefore
	}

	for index, duplicate := range stored {
		if index == keeper {
			continue
		}
		if duplicate.settled() {
			entry.Warnings = append(entry.Warnings, "Has more than one debt this week, and an extra one is already paid or spun; left as is.")
			continue
		}
		applied, err := run.disableDebt(entry, duplicate.debt)
		if err != nil {
			return err
		}
		if applied {
			entry.Changes = append(entry.Changes, "Removed a duplicate debt for the week.")
		}
	}

	switch {
	case keeper < 0 && !owes:
		return nil

	case keeper < 0:
		return run.createDebt(entry, desiredWinner)

	case !owes:
		kept := stored[keeper]
		if kept.settled() {
			entry.Warnings = append(entry.Warnings, "Has a debt the week no longer calls for, but it is already "+settledHow(kept)+"; left as is.")
			return nil
		}
		applied, err := run.disableDebt(entry, kept.debt)
		if err != nil {
			return err
		}
		if applied {
			entry.DebtAfter = nil
			entry.Changes = append(entry.Changes, "Removed the debt: the week no longer owes a spin.")
		}
		return nil

	default:
		kept := stored[keeper]
		if run.winnerMatches(kept, desiredWinner) {
			return nil
		}
		if kept.settled() {
			entry.Warnings = append(entry.Warnings, "The debt was decided with a different result than the week now has, but it is already "+settledHow(kept)+"; left as is.")
			return nil
		}
		return run.reassignDebt(entry, kept.debt, desiredWinner)
	}
}

func settledHow(stored storedDebt) string {
	if stored.debt.Paid {
		return "paid"
	}
	return "spun"
}

// winnerMatches reports whether a stored debt already has the winner the week calls for. With
// several winners the wheel decides, so an unspun debt must have no winner, while a spun one
// must have landed on one of this week's winners.
func (run *weekReconciliation) winnerMatches(stored storedDebt, desiredWinner *uuid.UUID) bool {
	winnerID := stored.debt.WinnerID
	if desiredWinner != nil {
		return winnerID != nil && *winnerID == *desiredWinner
	}
	if winnerID == nil {
		return true
	}
	if !stored.spun {
		return false
	}
	for _, winner := range run.winners {
		if winner == *winnerID {
			return true
		}
	}
	return false
}

func (run *weekReconciliation) debtState(stored storedDebt) *models.WeekReconcileDebt {
	id := stored.debt.ID
	return &models.WeekReconcileDebt{
		ID:      &id,
		Winner:  publicUserOrNil(stored.debt.WinnerID),
		Paid:    stored.debt.Paid,
		Spun:    stored.spun,
		Settled: stored.settled(),
	}
}

func publicUserOrNil(userID *uuid.UUID) *models.PublicUser {
	if userID == nil {
		return nil
	}
	user, err := database.GetUserInformationIncludingDisabled(*userID)
	if err != nil {
		user = models.PublicUser{ID: *userID, FirstName: "Deleted", LastName: "Deleted"}
	}
	return &user
}

// createDebt writes the missing debt for the week, with the single-winner achievements.
func (run *weekReconciliation) createDebt(entry *models.WeekReconcileUser, winnerID *uuid.UUID) error {
	debt := models.Debt{
		Date:     run.monday,
		SeasonID: run.season.ID,
		LoserID:  entry.User.ID,
		WinnerID: winnerID,
		Enabled:  true,
	}
	debt.ID = uuid.New()

	if !run.options.DryRun {
		if _, err := database.RegisterDebtInDB(debt); err != nil {
			return errors.New("failed to save the debt")
		}
	}

	after := &models.WeekReconcileDebt{Winner: publicUserOrNil(winnerID)}
	if !run.options.DryRun {
		after.ID = &debt.ID
	}
	entry.DebtAfter = after

	if winnerID == nil {
		entry.Changes = append(entry.Changes, "Added a debt; the wheel picks the winner.")
	} else {
		entry.Changes = append(entry.Changes, "Added a debt to "+after.Winner.FirstName+" "+after.Winner.LastName+", the week's only winner.")
		run.grantDebtAchievement(*winnerID, debtWinnerAchievementID)
		run.grantDebtAchievement(entry.User.ID, debtLoserAchievementID)
	}

	if run.options.Notify && !run.options.DryRun {
		notifyDebt(debt, run.weekNumber)
	}
	return nil
}

// reassignDebt moves an unsettled debt to the winner the week now calls for, or back to the
// wheel, moving the single-winner achievements with it.
func (run *weekReconciliation) reassignDebt(entry *models.WeekReconcileUser, debt models.Debt, winnerID *uuid.UUID) error {
	if !run.options.DryRun {
		err := database.SetDebtWinnerIfUnchanged(debt.ID, debt.WinnerID, winnerID)
		if errors.Is(err, database.ErrDebtChanged) {
			entry.Warnings = append(entry.Warnings, "The debt changed while it was being fixed (spun or paid); run the fix again.")
			return nil
		} else if err != nil {
			return errors.New("failed to update the debt")
		}
	}

	previousWinner := debt.WinnerID
	if previousWinner != nil {
		run.revokeDebtAchievement(*previousWinner, debtWinnerAchievementID)
		if winnerID == nil {
			run.revokeDebtAchievement(entry.User.ID, debtLoserAchievementID)
		}
	}
	if winnerID != nil {
		run.grantDebtAchievement(*winnerID, debtWinnerAchievementID)
		if previousWinner == nil {
			run.grantDebtAchievement(entry.User.ID, debtLoserAchievementID)
		}
	}

	debt.WinnerID = winnerID
	after := *entry.DebtBefore
	after.Winner = publicUserOrNil(winnerID)
	entry.DebtAfter = &after

	if winnerID == nil {
		entry.Changes = append(entry.Changes, fmt.Sprintf("Handed the debt to the wheel: the week now has %d winners.", len(run.winners)))
	} else {
		entry.Changes = append(entry.Changes, "Moved the debt to "+after.Winner.FirstName+" "+after.Winner.LastName+", the week's only winner.")
	}

	if run.options.Notify && !run.options.DryRun {
		notifyDebt(debt, run.weekNumber)
	}
	return nil
}

// disableDebt switches an unsettled debt off and revokes the achievements a single-winner
// debt came with. applied is false when the debt changed under us (a warning is recorded).
func (run *weekReconciliation) disableDebt(entry *models.WeekReconcileUser, debt models.Debt) (applied bool, err error) {
	if !run.options.DryRun {
		err := database.DisableDebtIfUnchanged(debt.ID, debt.WinnerID)
		if errors.Is(err, database.ErrDebtChanged) {
			entry.Warnings = append(entry.Warnings, "A debt changed while it was being fixed (spun or paid); run the fix again.")
			return false, nil
		} else if err != nil {
			return false, errors.New("failed to remove the debt")
		}
	}

	if debt.WinnerID != nil {
		run.revokeDebtAchievement(*debt.WinnerID, debtWinnerAchievementID)
		run.revokeDebtAchievement(debt.LoserID, debtLoserAchievementID)
	}
	return true, nil
}

// grantDebtAchievement gives one of the single-winner debt achievements, dated the week's
// Sunday. Each debt carries its own pair, so this is not once-per-day.
func (run *weekReconciliation) grantDebtAchievement(userID uuid.UUID, achievementID uuid.UUID) {
	granted, achievement, err := grantAchievement(userID, achievementID, run.sunday, achievementGrantOptions{Notify: run.options.Notify, DryRun: run.options.DryRun})
	if err != nil {
		logger.Log.Warn("Failed to give debt achievement to '" + userID.String() + "'. Error: " + err.Error())
		run.warnUser(userID, "An achievement for the debt could not be given: "+err.Error())
		return
	}
	if granted {
		run.recordAchievement(userID, achievement, run.sunday, true)
	}
}

// revokeDebtAchievement takes back one single-winner debt achievement given for the week.
func (run *weekReconciliation) revokeDebtAchievement(userID uuid.UUID, achievementID uuid.UUID) {
	revoked, achievement, err := revokeAchievementForDay(userID, achievementID, run.sunday, run.options.DryRun)
	if err != nil {
		logger.Log.Warn("Failed to revoke debt achievement from '" + userID.String() + "'. Error: " + err.Error())
		run.warnUser(userID, "An achievement for the debt could not be revoked: "+err.Error())
		return
	}
	if revoked {
		run.recordAchievement(userID, achievement, run.sunday, false)
	}
}

func (run *weekReconciliation) recordAchievement(userID uuid.UUID, achievement models.Achievement, givenAt time.Time, granted bool) {
	entry, found := run.entryByUser[userID]
	if !found {
		return
	}
	change := models.WeekReconcileAchievement{AchievementID: achievement.ID, Name: achievement.Name, GivenAt: givenAt}
	if granted {
		entry.AchievementsGranted = append(entry.AchievementsGranted, change)
	} else {
		entry.AchievementsRevoked = append(entry.AchievementsRevoked, change)
	}
}

func (run *weekReconciliation) warnUser(userID uuid.UUID, warning string) {
	if entry, found := run.entryByUser[userID]; found {
		entry.Warnings = append(entry.Warnings, warning)
	} else {
		run.report.Warnings = append(run.report.Warnings, warning)
	}
}

// notifyDebt sends the e-mails and pushes for a newly issued (or re-assigned) debt: to the
// loser and the winner when there is one winner, otherwise the loser's call to spin.
func notifyDebt(debt models.Debt, weekNumber int) {
	if debt.WinnerID != nil {
		winnerID := *debt.WinnerID

		if loser, err := database.GetAllUserInformation(debt.LoserID); err != nil {
			logger.Log.Info("Failed to get object for user '" + debt.LoserID.String() + "'. Ignoring. Error: " + err.Error())
		} else if err := utilities.SendSMTPForWeekLost(loser, weekNumber); err != nil {
			logger.Log.Info("Failed to notify user '" + debt.LoserID.String() + "' by e-mail. Ignoring. Error: " + err.Error())
		}
		if err := PushNotificationsForWeekLost(debt.LoserID); err != nil {
			logger.Log.Info("Failed to notify user '" + debt.LoserID.String() + "' by push. Ignoring. Error: " + err.Error())
		}

		if winner, err := database.GetAllUserInformation(winnerID); err != nil {
			logger.Log.Info("Failed to get object for user '" + winnerID.String() + "'. Ignoring. Error: " + err.Error())
		} else if err := utilities.SendSMTPForWheelSpinWin(winner, weekNumber); err != nil {
			logger.Log.Info("Failed to notify user '" + winnerID.String() + "' by e-mail. Ignoring. Error: " + err.Error())
		}
		if err := PushNotificationsForWheelSpinWin(winnerID, debt); err != nil {
			logger.Log.Info("Failed to notify user '" + winnerID.String() + "' by push. Ignoring. Error: " + err.Error())
		}
		return
	}

	if loser, err := database.GetAllUserInformation(debt.LoserID); err != nil {
		logger.Log.Info("Failed to get object for user '" + debt.LoserID.String() + "'. Ignoring. Error: " + err.Error())
	} else if err := utilities.SendSMTPForWheelSpin(loser, weekNumber); err != nil {
		logger.Log.Info("Failed to notify user '" + debt.LoserID.String() + "' by e-mail. Ignoring. Error: " + err.Error())
	}
	if err := PushNotificationsForWheelSpin(debt.LoserID, debt); err != nil {
		logger.Log.Info("Failed to notify user '" + debt.LoserID.String() + "' by push. Ignoring. Error: " + err.Error())
	}
}

// parseAdminDate reads a YYYY-MM-DD day in the server's time zone, at noon so the day is
// unambiguous for season boundary checks.
func parseAdminDate(value string) (time.Time, error) {
	day, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, time.Local), nil
}

// reconcileWeekForDate runs the reconciliation for every season ongoing on the day and writes
// the response.
func reconcileWeekForDate(context *gin.Context, dateValue string, options weekReconcileOptions) {
	date, err := parseAdminDate(dateValue)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Pick a day as YYYY-MM-DD."})
		context.Abort()
		return
	}

	sunday, err := utilities.FindNextSunday(date)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find the end of the week."})
		context.Abort()
		return
	} else if sunday.After(time.Now()) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "That week isn't over yet. Pick a finished week."})
		context.Abort()
		return
	}

	seasons, err := GetOngoingSeasonsFromDB(date)
	if err != nil {
		logger.Log.Error("Failed to get seasons for the week. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get seasons for the week."})
		context.Abort()
		return
	}

	reports := []models.WeekReconcileReport{}
	partial := false
	for _, season := range seasons {
		report, err := ReconcileWeekOfSeason(season, date, options)
		if errors.Is(err, ErrDebtPartiallyGenerated) {
			logger.Log.Error("Week fixed for some users only. Error: " + err.Error())
			partial = true
		} else if err != nil {
			logger.Log.Error("Failed to reconcile the week for season '" + season.ID.String() + "'. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process the week for season " + season.Name + "."})
			context.Abort()
			return
		}
		reports = append(reports, report)
	}

	if partial {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "The week could not be fixed for every user. Check the report and the log, then run it again.", "reports": reports})
		context.Abort()
		return
	}

	message := "Week fixed."
	switch {
	case len(seasons) == 0:
		message = "No season ran that week."
	case options.DryRun:
		message = "Week checked."
	}
	context.JSON(http.StatusOK, gin.H{"message": message, "reports": reports})
}

// APIAdminGetWeek is the "Fix a week" overview: a dry run of the reconciliation for every
// season ongoing on ?date=YYYY-MM-DD. Nothing is written.
func APIAdminGetWeek(context *gin.Context) {
	reconcileWeekForDate(context, context.Query("date"), weekReconcileOptions{DryRun: true})
}

// APIAdminReconcileWeek reconciles (or with dry_run, previews) the finished week holding the
// given day, for every season ongoing then.
func APIAdminReconcileWeek(context *gin.Context) {
	var request models.WeekReconcileRequest
	if err := bindJSON(context, &request); err != nil {
		logger.Log.Info("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	reconcileWeekForDate(context, request.Date, weekReconcileOptions{DryRun: request.DryRun, Notify: request.Notify})
}

// adminAddedWorkoutNote is the note on a workout an admin adds without writing one.
const adminAddedWorkoutNote = "Added by admin"

// APIAdminCreateExerciseForUser adds one goal-counting workout to a user's day — for the
// workouts someone forgot to log. The user must have a goal in a season running that day.
func APIAdminCreateExerciseForUser(context *gin.Context) {
	userID, err := uuid.Parse(context.Param("user_id"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse user ID."})
		context.Abort()
		return
	}

	var request models.AdminExerciseCreationRequest
	if err := bindJSON(context, &request); err != nil {
		logger.Log.Info("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	date, err := parseAdminDate(request.Date)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Pick a day as YYYY-MM-DD."})
		context.Abort()
		return
	}
	if utilities.SetClockToMinimum(date).After(time.Now()) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You can't add workouts on days in the future."})
		context.Abort()
		return
	}

	note := request.Note
	if note == "" {
		note = adminAddedWorkoutNote
	}
	if len(note) > 255 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "The note is too long."})
		context.Abort()
		return
	}

	user, err := database.GetUserByIDIncludingDisabled(userID)
	if err != nil {
		logger.Log.Error("Failed to get user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user."})
		context.Abort()
		return
	} else if user == nil {
		context.JSON(http.StatusNotFound, gin.H{"error": "User not found."})
		context.Abort()
		return
	}

	seasons, err := GetOngoingSeasonsFromDB(date)
	if err != nil {
		logger.Log.Error("Failed to get seasons. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get seasons."})
		context.Abort()
		return
	}
	hasGoal := false
	for _, season := range seasons {
		goal, err := database.GetGoalFromUserWithinSeason(season.ID, userID)
		if err != nil {
			logger.Log.Error("Failed to get goal. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get goal."})
			context.Abort()
			return
		} else if goal != nil {
			hasGoal = true
			break
		}
	}
	if !hasGoal {
		context.JSON(http.StatusBadRequest, gin.H{"error": "The user has no goal in a season running that day."})
		context.Abort()
		return
	}

	exerciseDay, err := database.GetExerciseDayByUserIDAndDate(userID, date)
	if err != nil {
		logger.Log.Error("Failed to get exercise day. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise day."})
		context.Abort()
		return
	}
	if exerciseDay == nil {
		newDay := models.ExerciseDay{
			Date:    utilities.SetClockToMinimum(date),
			UserID:  &userID,
			Enabled: true,
		}
		newDay.ID = uuid.New()
		if _, err := database.CreateExerciseDayForGoalInDatabase(newDay); err != nil {
			logger.Log.Error("Failed to create exercise day. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create exercise day."})
			context.Abort()
			return
		}
		exerciseDay = &newDay
	} else {
		dayObject, err := ConvertExerciseDayToExerciseDayObject(*exerciseDay)
		if err != nil {
			logger.Log.Error("Failed to convert exercise day. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert exercise day."})
			context.Abort()
			return
		}
		if dayObject.ExerciseInterval >= maxWorkoutsPerDay {
			context.JSON(http.StatusBadRequest, gin.H{"error": dayWorkoutLimitMessage})
			context.Abort()
			return
		}
	}

	dayTime := exerciseDay.Date
	exercise := models.Exercise{
		Note:             note,
		Enabled:          true,
		IsOn:             true,
		CountsTowardGoal: true,
		ExerciseDayID:    exerciseDay.ID,
		Time:             &dayTime,
	}
	exercise.ID = uuid.New()

	exercise, err = database.CreateExerciseInDB(exercise)
	if err != nil {
		logger.Log.Error("Failed to create exercise. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create exercise."})
		context.Abort()
		return
	}

	exerciseObject, err := ConvertExerciseToExerciseObject(exercise)
	if err != nil {
		logger.Log.Error("Failed to convert exercise. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert exercise."})
		context.Abort()
		return
	}

	logger.Log.Info("Admin added a workout on " + request.Date + " for user '" + userID.String() + "'.")
	context.JSON(http.StatusCreated, gin.H{"message": "Workout added.", "exercise": exerciseObject})
}

// maxSickleaveGrant caps one sick-leave top-up, so a typo can't hand out a season's worth.
const maxSickleaveGrant = 10

// APIAdminGrantSickleave adds unused sick-leave weeks to a user's goal in a season — for
// someone who has used up their allowance and is granted more.
func APIAdminGrantSickleave(context *gin.Context) {
	userID, err := uuid.Parse(context.Param("user_id"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse user ID."})
		context.Abort()
		return
	}

	var request models.AdminSickleaveGrantRequest
	if err := bindJSON(context, &request); err != nil {
		logger.Log.Info("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	if request.Count < 1 || request.Count > maxSickleaveGrant {
		context.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Grant between 1 and %d weeks of sick leave.", maxSickleaveGrant)})
		context.Abort()
		return
	}
	if request.SeasonID == uuid.Nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Pick a season."})
		context.Abort()
		return
	}

	goal, err := database.GetGoalFromUserWithinSeason(request.SeasonID, userID)
	if err != nil {
		logger.Log.Error("Failed to get goal. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get goal."})
		context.Abort()
		return
	} else if goal == nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "The user has no goal in that season."})
		context.Abort()
		return
	}

	for i := 0; i < request.Count; i++ {
		sickleave := models.Sickleave{GoalID: goal.ID}
		sickleave.ID = uuid.New()
		if err := database.CreateSickleave(sickleave); err != nil {
			logger.Log.Error("Failed to create sick leave. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Only %d of %d weeks of sick leave could be granted.", i, request.Count)})
			context.Abort()
			return
		}
	}

	unused, _, err := database.GetUnusedSickleaveForGoalWithinWeek(goal.ID)
	if err != nil {
		logger.Log.Error("Failed to count sick leave. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Sick leave granted, but the new total could not be read."})
		context.Abort()
		return
	}

	logger.Log.Info(fmt.Sprintf("Admin granted %d week(s) of sick leave to user '%s' in season '%s'.", request.Count, userID, request.SeasonID))
	context.JSON(http.StatusCreated, gin.H{"message": "Sick leave granted.", "sickleave_left": len(unused)})
}
