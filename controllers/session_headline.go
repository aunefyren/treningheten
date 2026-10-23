package controllers

import (
	"slices"
	"strconv"
	"strings"

	"github.com/aunefyren/treningheten/models"
)

// strengthLogoAction is the action whose logo stands for strength work in the feed, so a
// Hevy session of logo-less lifts still shows one strength icon instead of none (or many).
const strengthLogoAction = "Weight Training"

// dominantShareNumerator/Denominator: a category covering at least two thirds of a session's
// activities names the session ("Strength training" for six lifts and a treadmill warm-up).
const (
	dominantShareNumerator   = 2
	dominantShareDenominator = 3
)

// sessionCategoryVerbs is the feed phrase for a session dominated by a category but not by
// one single action ("did some strength training" for a Hevy push day).
var sessionCategoryVerbs = map[string]string{
	models.ActionCategoryStrength: "did some strength training",
	models.ActionCategoryCardio:   "did some cardio",
	models.ActionCategorySport:    "played some sports",
	models.ActionCategoryMobility: "did some mobility work",
	models.ActionCategoryGeneral:  "worked out",
}

// sessionCategoryLabels names a category inside a generated title when it holds more than one
// distinct activity ("Run + strength training").
var sessionCategoryLabels = map[string]string{
	models.ActionCategoryStrength: "strength training",
	models.ActionCategoryCardio:   "cardio",
	models.ActionCategorySport:    "sports",
	models.ActionCategoryMobility: "mobility",
	models.ActionCategoryGeneral:  "workout",
}

const (
	sessionMixedVerb    = "did a range of workouts"
	sessionDefaultVerb  = "worked out"
	sessionDefaultTitle = "Workout"
	sessionDefaultName  = "Activity"
	sessionMaxTitleRuns = 2 // generated titles name at most two parts, then "+ N more"
)

// sessionPart is the slice of one activity that describeSession needs: what it is called,
// its category, its own feed phrase (empty when the action has none) and the logo it shows
// in the feed (empty for none).
type sessionPart struct {
	name     string
	category string
	verb     string
	logo     string
}

// newSessionPart builds a sessionPart from an activity's action (nil for a custom or bare
// activity), falling back to its note for the name and to its measurement type for the
// category. Strength activities without a logo of their own borrow strengthLogoAction.
func newSessionPart(action *models.Action, operationType string, note *string) sessionPart {
	part := sessionPart{name: sessionDefaultName, category: models.DefaultActionCategory(operationType)}
	if action != nil {
		part.name = action.Name
		if action.Category != "" {
			part.category = action.Category
		}
		if action.PastTenseVerb != nil {
			part.verb = strings.TrimSpace(*action.PastTenseVerb)
		}
		if action.HasLogo {
			part.logo = action.Name
		}
	} else if note != nil && strings.TrimSpace(*note) != "" {
		part.name = strings.TrimSpace(*note)
	}
	if part.logo == "" && part.category == models.ActionCategoryStrength {
		part.logo = strengthLogoAction
	}
	return part
}

// sessionHeadlineForExercise describes a converted session. A session imported from Strava or
// Hevy keeps the source's own title (its Note); a manual session's Note is a free-text note,
// not a title, so its title is generated.
func sessionHeadlineForExercise(exercise models.ExerciseObject) models.SessionHeadline {
	parts := make([]sessionPart, 0, len(exercise.Operations))
	for _, operation := range exercise.Operations {
		parts = append(parts, newSessionPart(operation.Action, operation.Type, operation.Note))
	}
	sourceTitle := ""
	if len(exercise.StravaID) > 0 || exercise.HevyWorkoutID != nil {
		sourceTitle = exercise.Note
	}
	return describeSession(sourceTitle, parts)
}

// sessionHeadlineForFeedItem is sessionHeadlineForExercise for the /exercises list rows,
// which carry the action fields flattened instead of the action itself.
func sessionHeadlineForFeedItem(session models.SessionFeedItem) models.SessionHeadline {
	parts := make([]sessionPart, 0, len(session.Activities))
	for _, item := range session.Activities {
		var action *models.Action
		if item.ActionID != nil {
			verb := item.ActionPastTenseVerb
			action = &models.Action{Name: item.ActionName, Category: item.ActionCategory, HasLogo: item.ActionHasLogo, PastTenseVerb: &verb}
		}
		parts = append(parts, newSessionPart(action, item.ActionType, item.Note))
	}
	sourceTitle := ""
	if session.HasStrava || session.HevyWorkoutID != nil {
		sourceTitle = session.Note
	}
	return describeSession(sourceTitle, parts)
}

// describeSession names a session from its activities:
//
//  1. Title: the source title when given; otherwise the single activity's name when there is
//     only one kind; otherwise "Strength training" when strength dominates (two thirds or
//     more), the dominant category's activity names when another category does, and up to
//     two category runs ("Run + strength training") when nothing does.
//  2. Phrase: the single activity's own phrase, the dominant category's phrase, or "did a
//     range of workouts" for a genuinely mixed session.
//  3. Logos: each activity's logo, de-duplicated in order.
func describeSession(sourceTitle string, parts []sessionPart) models.SessionHeadline {
	headline := models.SessionHeadline{Logos: []string{}}
	for _, part := range parts {
		if part.logo != "" && !slices.Contains(headline.Logos, part.logo) {
			headline.Logos = append(headline.Logos, part.logo)
		}
	}

	categories, byCategory := groupSessionParts(parts)
	switch {
	case len(parts) == 0:
		headline.Title = sessionDefaultTitle
		headline.PastTenseVerb = sessionDefaultVerb
		headline.Category = models.ActionCategoryGeneral
	case len(distinctSessionNames(parts)) == 1:
		headline.Title = parts[0].name
		headline.PastTenseVerb = sessionPartsVerb(parts[0].category, parts)
		headline.Category = parts[0].category
	case len(byCategory[categories[0]])*dominantShareDenominator >= len(parts)*dominantShareNumerator:
		dominant := categories[0]
		names := distinctSessionNames(byCategory[dominant])
		headline.Category = dominant
		headline.PastTenseVerb = sessionCategoryVerbs[dominant]
		if dominant == models.ActionCategoryStrength {
			headline.Title = capitalizeFirst(sessionCategoryLabels[dominant])
		} else if len(names) == 1 {
			headline.Title = names[0]
			headline.PastTenseVerb = sessionPartsVerb(dominant, byCategory[dominant])
		} else {
			headline.Title = joinSessionRuns(names)
		}
	default:
		runs := make([]string, 0, len(categories))
		for _, category := range categories {
			names := distinctSessionNames(byCategory[category])
			if category != models.ActionCategoryStrength && len(names) == 1 {
				runs = append(runs, names[0])
			} else {
				runs = append(runs, sessionCategoryLabels[category])
			}
		}
		headline.Title = capitalizeFirst(joinSessionRuns(runs))
		headline.PastTenseVerb = sessionMixedVerb
		headline.Category = models.SessionCategoryMixed
	}

	if strings.TrimSpace(sourceTitle) != "" {
		headline.Title = strings.TrimSpace(sourceTitle)
	}
	return headline
}

// groupSessionParts buckets parts by category and returns the categories ordered by how many
// activities they hold (ties keep first-appearance order).
func groupSessionParts(parts []sessionPart) ([]string, map[string][]sessionPart) {
	byCategory := map[string][]sessionPart{}
	categories := []string{}
	for _, part := range parts {
		if _, seen := byCategory[part.category]; !seen {
			categories = append(categories, part.category)
		}
		byCategory[part.category] = append(byCategory[part.category], part)
	}
	// Stable insertion sort: a session has a handful of categories at most.
	for i := 1; i < len(categories); i++ {
		for j := i; j > 0 && len(byCategory[categories[j]]) > len(byCategory[categories[j-1]]); j-- {
			categories[j], categories[j-1] = categories[j-1], categories[j]
		}
	}
	return categories, byCategory
}

// sessionPartsVerb is the phrase for a run of same-named parts: the action's own phrase when
// it has one, otherwise its category's.
func sessionPartsVerb(category string, parts []sessionPart) string {
	for _, part := range parts {
		if part.verb != "" {
			return part.verb
		}
	}
	if verb, ok := sessionCategoryVerbs[category]; ok {
		return verb
	}
	return sessionDefaultVerb
}

// distinctSessionNames lists the parts' names without repeats, in order.
func distinctSessionNames(parts []sessionPart) []string {
	names := []string{}
	for _, part := range parts {
		if !slices.Contains(names, part.name) {
			names = append(names, part.name)
		}
	}
	return names
}

// joinSessionRuns joins up to sessionMaxTitleRuns names with " + " and counts the rest.
func joinSessionRuns(runs []string) string {
	if len(runs) <= sessionMaxTitleRuns {
		return strings.Join(runs, " + ")
	}
	return strings.Join(runs[:sessionMaxTitleRuns], " + ") + " + " + strconv.Itoa(len(runs)-sessionMaxTitleRuns) + " more"
}

// capitalizeFirst upper-cases the first letter of a generated title ("strength training" →
// "Strength training"); action names keep their own casing.
func capitalizeFirst(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}
