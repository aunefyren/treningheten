package controllers

import (
	"reflect"
	"testing"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// headlineAction builds an action for headline tests; verb "" means the action has none.
func headlineAction(name, category, verb string, hasLogo bool) *models.Action {
	action := &models.Action{Name: name, Category: category, HasLogo: hasLogo}
	if verb != "" {
		action.PastTenseVerb = &verb
	}
	return action
}

func TestDescribeSession(t *testing.T) {
	run := newSessionPart(headlineAction("Run", models.ActionCategoryCardio, "went for a run", true), "moving", nil)
	ride := newSessionPart(headlineAction("Bicycling", models.ActionCategoryCardio, "went for a bike ride", true), "moving", nil)
	padel := newSessionPart(headlineAction("Padel", models.ActionCategorySport, "played some padel", true), "timing", nil)
	yoga := newSessionPart(headlineAction("Yoga", models.ActionCategoryMobility, "did some yoga", true), "timing", nil)
	bench := newSessionPart(headlineAction("Bench Press (Barbell)", models.ActionCategoryStrength, "", false), "lifting", nil)
	squat := newSessionPart(headlineAction("Squat (Barbell)", models.ActionCategoryStrength, "", false), "lifting", nil)
	row := newSessionPart(headlineAction("Seated Row (Cable)", models.ActionCategoryStrength, "", false), "lifting", nil)
	tennis := newSessionPart(headlineAction("Tennis", models.ActionCategorySport, "played some tennis", true), "timing", nil)
	golf := newSessionPart(headlineAction("Golf", models.ActionCategorySport, "played some golf", true), "timing", nil)
	swim := newSessionPart(headlineAction("Swim", models.ActionCategoryCardio, "went for a swim", true), "moving", nil)
	hike := newSessionPart(headlineAction("Hike", models.ActionCategoryCardio, "went for a hike", true), "moving", nil)

	tests := []struct {
		name        string
		sourceTitle string
		parts       []sessionPart
		want        models.SessionHeadline
	}{
		{"empty session", "", nil,
			models.SessionHeadline{Title: "Workout", PastTenseVerb: "worked out", Category: models.ActionCategoryGeneral, Logos: []string{}}},
		{"empty session keeps a source title", "Morning", nil,
			models.SessionHeadline{Title: "Morning", PastTenseVerb: "worked out", Category: models.ActionCategoryGeneral, Logos: []string{}}},
		{"single activity uses its own phrase", "", []sessionPart{padel},
			models.SessionHeadline{Title: "Padel", PastTenseVerb: "played some padel", Category: models.ActionCategorySport, Logos: []string{"Padel"}}},
		{"source title wins", "Pad padel", []sessionPart{padel},
			models.SessionHeadline{Title: "Pad padel", PastTenseVerb: "played some padel", Category: models.ActionCategorySport, Logos: []string{"Padel"}}},
		{"one lift without a phrase falls back to its category", "", []sessionPart{bench},
			models.SessionHeadline{Title: "Bench Press (Barbell)", PastTenseVerb: "did some strength training", Category: models.ActionCategoryStrength, Logos: []string{strengthLogoAction}}},
		{"repeated activity", "", []sessionPart{run, run},
			models.SessionHeadline{Title: "Run", PastTenseVerb: "went for a run", Category: models.ActionCategoryCardio, Logos: []string{"Run"}}},
		{"all strength", "", []sessionPart{bench, squat, row},
			models.SessionHeadline{Title: "Strength training", PastTenseVerb: "did some strength training", Category: models.ActionCategoryStrength, Logos: []string{strengthLogoAction}}},
		{"strength dominates a warm-up", "", []sessionPart{run, bench, squat},
			models.SessionHeadline{Title: "Strength training", PastTenseVerb: "did some strength training", Category: models.ActionCategoryStrength, Logos: []string{"Run", strengthLogoAction}}},
		{"one cardio name dominates", "", []sessionPart{run, run, bench},
			models.SessionHeadline{Title: "Run", PastTenseVerb: "went for a run", Category: models.ActionCategoryCardio, Logos: []string{"Run", strengthLogoAction}}},
		{"several cardio names", "", []sessionPart{run, ride},
			models.SessionHeadline{Title: "Run + Bicycling", PastTenseVerb: "did some cardio", Category: models.ActionCategoryCardio, Logos: []string{"Run", "Bicycling"}}},
		{"many cardio names are counted", "", []sessionPart{run, ride, swim, hike},
			models.SessionHeadline{Title: "Run + Bicycling + 2 more", PastTenseVerb: "did some cardio", Category: models.ActionCategoryCardio, Logos: []string{"Run", "Bicycling", "Swim", "Hike"}}},
		{"mixed cardio and strength", "", []sessionPart{run, bench},
			models.SessionHeadline{Title: "Run + strength training", PastTenseVerb: "did a range of workouts", Category: models.SessionCategoryMixed, Logos: []string{"Run", strengthLogoAction}}},
		{"mixed with a category label", "", []sessionPart{bench, tennis, golf, yoga, run, ride},
			models.SessionHeadline{Title: "Sports + cardio + 2 more", PastTenseVerb: "did a range of workouts", Category: models.SessionCategoryMixed, Logos: []string{strengthLogoAction, "Tennis", "Golf", "Yoga", "Run", "Bicycling"}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := describeSession(tc.sourceTitle, tc.parts); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("describeSession() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestNewSessionPart(t *testing.T) {
	note := "  My Secret Move "
	blank := "  "
	tests := []struct {
		name          string
		action        *models.Action
		operationType string
		note          *string
		want          sessionPart
	}{
		{"action with logo and phrase", headlineAction("Padel", models.ActionCategorySport, "played some padel", true), "timing", nil,
			sessionPart{name: "Padel", category: models.ActionCategorySport, verb: "played some padel", logo: "Padel"}},
		{"action without category defaults from type", headlineAction("Gravel", "", "", false), "moving", nil,
			sessionPart{name: "Gravel", category: models.ActionCategoryCardio}},
		{"strength without logo borrows weight training", headlineAction("Curl", models.ActionCategoryStrength, "", false), "lifting", nil,
			sessionPart{name: "Curl", category: models.ActionCategoryStrength, logo: strengthLogoAction}},
		{"custom exercise named by its note", nil, "timing", &note,
			sessionPart{name: "My Secret Move", category: models.ActionCategoryGeneral}},
		{"custom lift", nil, "lifting", &note,
			sessionPart{name: "My Secret Move", category: models.ActionCategoryStrength, logo: strengthLogoAction}},
		{"bare activity", nil, "timing", &blank,
			sessionPart{name: "Activity", category: models.ActionCategoryGeneral}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := newSessionPart(tc.action, tc.operationType, tc.note); got != tc.want {
				t.Fatalf("newSessionPart() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestSessionPartsVerb(t *testing.T) {
	if got := sessionPartsVerb("unknown", nil); got != "worked out" {
		t.Fatalf("unknown category verb = %q, want worked out", got)
	}
}

func TestCapitalizeFirst(t *testing.T) {
	tests := map[string]string{"": "", "strength training": "Strength training", "Run": "Run"}
	for value, want := range tests {
		if got := capitalizeFirst(value); got != want {
			t.Errorf("capitalizeFirst(%q) = %q, want %q", value, got, want)
		}
	}
}

func TestSessionHeadlineForExercise(t *testing.T) {
	bench := headlineAction("Bench Press (Barbell)", models.ActionCategoryStrength, "", false)
	squat := headlineAction("Squat (Barbell)", models.ActionCategoryStrength, "", false)
	lifts := []models.OperationObject{{Action: bench, Type: "lifting"}, {Action: squat, Type: "lifting"}}
	hevyID := "workout-1"

	tests := []struct {
		name     string
		exercise models.ExerciseObject
		want     string
	}{
		{"Hevy keeps its workout title", models.ExerciseObject{Note: "Push day", HevyWorkoutID: &hevyID, Operations: lifts}, "Push day"},
		{"Strava keeps its activity name", models.ExerciseObject{Note: "Pad padel", StravaID: []string{"1"}, Operations: lifts}, "Pad padel"},
		{"manual note is not a title", models.ExerciseObject{Note: "knee felt off", Operations: lifts}, "Strength training"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sessionHeadlineForExercise(tc.exercise).Title; got != tc.want {
				t.Fatalf("title = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSessionHeadlineForFeedItem(t *testing.T) {
	actionID := uuid.New()
	note := "Garage thing"
	hevyID := "workout-1"
	activities := []models.ActivityFeedItem{
		{ActionID: &actionID, ActionName: "Padel", ActionCategory: models.ActionCategorySport, ActionPastTenseVerb: "played some padel", ActionHasLogo: true, ActionType: "timing"},
	}

	t.Run("single action uses its phrase and logo", func(t *testing.T) {
		got := sessionHeadlineForFeedItem(models.SessionFeedItem{Activities: activities})
		want := models.SessionHeadline{Title: "Padel", PastTenseVerb: "played some padel", Category: models.ActionCategorySport, Logos: []string{"Padel"}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("headline = %+v, want %+v", got, want)
		}
	})
	t.Run("Strava source title", func(t *testing.T) {
		if got := sessionHeadlineForFeedItem(models.SessionFeedItem{Note: "Pad padel", HasStrava: true, Activities: activities}).Title; got != "Pad padel" {
			t.Fatalf("title = %q, want Pad padel", got)
		}
	})
	t.Run("Hevy source title", func(t *testing.T) {
		if got := sessionHeadlineForFeedItem(models.SessionFeedItem{Note: "Legs", HevyWorkoutID: &hevyID, Activities: activities}).Title; got != "Legs" {
			t.Fatalf("title = %q, want Legs", got)
		}
	})
	t.Run("custom activity without action", func(t *testing.T) {
		got := sessionHeadlineForFeedItem(models.SessionFeedItem{Note: "ignored", Activities: []models.ActivityFeedItem{{Note: &note}}})
		if got.Title != "Garage thing" || got.Category != models.ActionCategoryGeneral {
			t.Fatalf("headline = %+v, want Garage thing / general", got)
		}
	})
}
