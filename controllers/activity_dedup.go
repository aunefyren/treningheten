package controllers

import (
	"errors"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// duplicateActivityWindow is how close two imported sessions from different sources must
// start to be treated as the same workout (e.g. a Hevy workout and its auto-push to Strava).
const duplicateActivityWindow = 15 * time.Minute

// activitySources lists the import sources duplicate detection can rank, in the default
// order (Hevy first: it carries the richer set/rep data).
var activitySources = []string{models.IntegrationProviderHevy, models.IntegrationProviderStrava}

// defaultActivitySourcePriority is the ranking a user gets when they first connect a second
// import source without having chosen one.
func defaultActivitySourcePriority() string {
	return strings.Join(activitySources, ",")
}

// parseActivitySourcePriority splits a stored ranking into its sources, highest first.
// nil and empty both yield an empty list (detection off).
func parseActivitySourcePriority(raw *string) []string {
	sources := []string{}
	if raw == nil {
		return sources
	}
	for _, part := range strings.Split(*raw, ",") {
		source := strings.ToLower(strings.TrimSpace(part))
		if source != "" {
			sources = append(sources, source)
		}
	}
	return sources
}

// normaliseActivitySourcePriority validates a client-sent ranking (known sources, no
// repeats) and returns it in its stored form. An empty ranking is valid and turns detection
// off.
func normaliseActivitySourcePriority(raw string) (string, error) {
	sources := parseActivitySourcePriority(&raw)
	for index, source := range sources {
		if !slices.Contains(activitySources, source) {
			return "", errors.New("unknown activity source: " + source)
		}
		if slices.Contains(sources[:index], source) {
			return "", errors.New("activity source listed twice: " + source)
		}
	}
	return strings.Join(sources, ","), nil
}

// sourcedExercise is an imported session tagged with the source it came from.
type sourcedExercise struct {
	exercise models.Exercise
	source   string
}

// classifyImportedExercises tags each session with its import source. Sessions that are
// neither a Hevy workout nor exactly one Strava activity are dropped: a *combined* Strava
// session (see APIStravaCombine) is the user's own curation and is never hidden or allowed to
// hide anything.
func classifyImportedExercises(exercises []models.Exercise) ([]sourcedExercise, error) {
	exerciseIDs := make([]uuid.UUID, 0, len(exercises))
	for _, exercise := range exercises {
		exerciseIDs = append(exerciseIDs, exercise.ID)
	}
	stravaCounts, err := database.CountStravaActivitiesForExercises(exerciseIDs)
	if err != nil {
		return nil, err
	}

	classified := []sourcedExercise{}
	for _, exercise := range exercises {
		switch {
		case exercise.HevyWorkoutID != nil:
			classified = append(classified, sourcedExercise{exercise, models.IntegrationProviderHevy})
		case stravaCounts[exercise.ID] == 1:
			classified = append(classified, sourcedExercise{exercise, models.IntegrationProviderStrava})
		}
	}
	return classified, nil
}

// hideDuplicateExercise hides loser behind winnerID. Anything already hidden behind loser
// moves to winnerID, so a winner going away restores its duplicates in one step.
func hideDuplicateExercise(loserID uuid.UUID, winnerID uuid.UUID) error {
	if err := database.SetExerciseSupersededBy(loserID, &winnerID); err != nil {
		return err
	}
	return database.RepointSupersededExercises(loserID, winnerID)
}

// resolveImportedDuplicates compares a freshly synced session against the user's other
// imported sessions starting within duplicateActivityWindow, and hides whichever ranks lower
// in the user's ActivitySourcePriority. Reports whether the session itself ended up hidden.
// A no-op when detection is off, the session isn't an import, or its source isn't ranked.
func resolveImportedDuplicates(user models.User, exercise models.Exercise) (bool, error) {
	priority := parseActivitySourcePriority(user.ActivitySourcePriority)
	if len(priority) < 2 || exercise.Time == nil {
		return false, nil
	}

	candidates, err := database.GetImportedExercisesForUserNearTime(user.ID, *exercise.Time, duplicateActivityWindow)
	if err != nil {
		return false, err
	}
	classified, err := classifyImportedExercises(candidates)
	if err != nil {
		return false, err
	}

	incomingIndex := slices.IndexFunc(classified, func(candidate sourcedExercise) bool { return candidate.exercise.ID == exercise.ID })
	if incomingIndex < 0 {
		return false, nil
	}
	incoming := classified[incomingIndex]
	incomingRank := slices.Index(priority, incoming.source)
	if incomingRank < 0 {
		return false, nil
	}

	var winner *sourcedExercise
	winnerRank := incomingRank
	losers := []sourcedExercise{}
	for index, candidate := range classified {
		rank := slices.Index(priority, candidate.source)
		if candidate.source == incoming.source || rank < 0 {
			continue
		}
		if rank < winnerRank {
			winner = &classified[index]
			winnerRank = rank
		} else if rank > incomingRank {
			losers = append(losers, candidate)
		}
	}

	if winner != nil {
		return true, hideDuplicateExercise(incoming.exercise.ID, winner.exercise.ID)
	}
	for _, loser := range losers {
		if err := hideDuplicateExercise(loser.exercise.ID, incoming.exercise.ID); err != nil {
			return false, err
		}
	}
	return false, nil
}

// restoreDuplicatesOf brings back every session hidden behind exerciseID, for when that
// session goes away (its Hevy workout was deleted). Each restored session is resolved again,
// since another duplicate may still outrank it.
func restoreDuplicatesOf(user models.User, exerciseID uuid.UUID) error {
	hidden, err := database.GetExercisesSupersededBy(exerciseID)
	if err != nil {
		return err
	}
	for _, exercise := range hidden {
		if err := database.SetExerciseSupersededBy(exercise.ID, nil); err != nil {
			return err
		}
		exercise.Enabled = true
		exercise.SupersededByID = nil
		if _, err := resolveImportedDuplicates(user, exercise); err != nil {
			return err
		}
	}
	return nil
}

// reconcileDuplicateActivitiesForUser re-applies the user's ActivitySourcePriority to their
// whole history: every hidden duplicate is restored, then each overlapping cross-source pair
// is resolved again, highest-ranked sessions first. Idempotent; run after the ranking
// changes, so turning detection off restores everything and a flipped ranking swaps which
// copy shows. Already-processed weeks are not recalculated (see docs/activity-dedup.md).
func reconcileDuplicateActivitiesForUser(user models.User) error {
	hidden, err := database.GetSupersededExercisesForUser(user.ID)
	if err != nil {
		return err
	}
	for _, exercise := range hidden {
		if err := database.SetExerciseSupersededBy(exercise.ID, nil); err != nil {
			return err
		}
	}

	priority := parseActivitySourcePriority(user.ActivitySourcePriority)
	if len(priority) < 2 {
		return nil
	}

	exercises, err := database.GetImportedExercisesForUser(user.ID)
	if err != nil {
		return err
	}
	classified, err := classifyImportedExercises(exercises)
	if err != nil {
		return err
	}
	sort.SliceStable(classified, func(a, b int) bool { return classified[a].exercise.Time.Before(*classified[b].exercise.Time) })

	ranks := make([]int, len(classified))
	order := []int{}
	for index, candidate := range classified {
		ranks[index] = slices.Index(priority, candidate.source)
		if ranks[index] >= 0 {
			order = append(order, index)
		}
	}
	sort.SliceStable(order, func(a, b int) bool { return ranks[order[a]] < ranks[order[b]] })

	hiddenIndexes := map[int]bool{}
	for _, winnerIndex := range order {
		if hiddenIndexes[winnerIndex] {
			continue
		}
		for _, loserIndex := range neighboursWithinWindow(classified, winnerIndex) {
			if hiddenIndexes[loserIndex] || ranks[loserIndex] <= ranks[winnerIndex] {
				continue
			}
			if err := hideDuplicateExercise(classified[loserIndex].exercise.ID, classified[winnerIndex].exercise.ID); err != nil {
				return err
			}
			hiddenIndexes[loserIndex] = true
		}
	}

	logger.Log.Debug("Reconciled duplicate activities for user " + user.ID.String() + ".")
	return nil
}

// neighboursWithinWindow returns the indexes of sessions (sorted by time) starting within
// duplicateActivityWindow of classified[index], excluding the session itself.
func neighboursWithinWindow(classified []sourcedExercise, index int) []int {
	start := *classified[index].exercise.Time
	neighbours := []int{}
	for other := index - 1; other >= 0 && start.Sub(*classified[other].exercise.Time) <= duplicateActivityWindow; other-- {
		neighbours = append(neighbours, other)
	}
	for other := index + 1; other < len(classified) && classified[other].exercise.Time.Sub(start) <= duplicateActivityWindow; other++ {
		neighbours = append(neighbours, other)
	}
	return neighbours
}
