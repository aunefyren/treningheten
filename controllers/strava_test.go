package controllers

import (
	"reflect"
	"testing"

	"github.com/aunefyren/treningheten/models"
)

func intPtr(i int) *int { return &i }

func TestStravaDerivedTags(t *testing.T) {
	tests := []struct {
		name     string
		activity models.StravaGetActivitiesRequestReply
		want     []string
	}{
		{"none", models.StravaGetActivitiesRequestReply{}, []string{}},
		{"commute only", models.StravaGetActivitiesRequestReply{Commute: true}, []string{models.TagCommute}},
		{"race run", models.StravaGetActivitiesRequestReply{WorkoutType: intPtr(1)}, []string{models.TagRace}},
		{"race ride", models.StravaGetActivitiesRequestReply{WorkoutType: intPtr(11)}, []string{models.TagRace}},
		{"long run", models.StravaGetActivitiesRequestReply{WorkoutType: intPtr(2)}, []string{models.TagLongRun}},
		{"workout run", models.StravaGetActivitiesRequestReply{WorkoutType: intPtr(3)}, []string{models.TagWorkout}},
		{"workout ride", models.StravaGetActivitiesRequestReply{WorkoutType: intPtr(12)}, []string{models.TagWorkout}},
		{"unknown workout type", models.StravaGetActivitiesRequestReply{WorkoutType: intPtr(99)}, []string{}},
		{"commute + race", models.StravaGetActivitiesRequestReply{Commute: true, WorkoutType: intPtr(1)}, []string{models.TagCommute, models.TagRace}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stravaDerivedTags(tt.activity)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stravaDerivedTags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeStravaTags(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		derived  []string
		want     models.TagList
	}{
		{
			name:     "preserves user tags and refreshes managed",
			existing: []string{models.TagForACause, models.TagCommute},
			derived:  []string{models.TagRace},
			want:     models.TagList{models.TagForACause, models.TagRace},
		},
		{
			name:     "drops stale managed tag no longer derived",
			existing: []string{models.TagCommute},
			derived:  []string{},
			want:     models.TagList{},
		},
		{
			name:     "keeps user-only tags when nothing derived",
			existing: []string{models.TagWithPet, models.TagRecovery},
			derived:  []string{},
			want:     models.TagList{models.TagWithPet, models.TagRecovery},
		},
		{
			name:     "dedupes derived already present as user tag is ignored (managed wins)",
			existing: []string{models.TagRace},
			derived:  []string{models.TagRace},
			want:     models.TagList{models.TagRace},
		},
		{
			name:     "drops invalid existing tags",
			existing: []string{"bogus", models.TagWithKid},
			derived:  []string{models.TagWorkout},
			want:     models.TagList{models.TagWithKid, models.TagWorkout},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeStravaTags(tt.existing, tt.derived)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeStravaTags() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStravaActivityIsPrivate covers the mapping from Strava's two overlapping privacy
// fields onto the session's Private flag — notably that `followers_only` is not private,
// since Treningheten's audience is the same trusted circle as a user's followers.
func TestStravaActivityIsPrivate(t *testing.T) {
	tests := []struct {
		name       string
		private    bool
		visibility string
		want       bool
	}{
		{"public activity", false, "everyone", false},
		{"legacy private bool", true, "everyone", true},
		{"only_me visibility", false, "only_me", true},
		{"only_me with padding and case", false, " Only_Me ", true},
		{"followers_only stays visible", false, "followers_only", false},
		{"missing visibility falls back to the bool", false, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			activity := models.StravaGetActivitiesRequestReply{Private: tt.private, Visibility: tt.visibility}
			if got := stravaActivityIsPrivate(activity); got != tt.want {
				t.Errorf("stravaActivityIsPrivate(private=%v, visibility=%q) = %v, want %v", tt.private, tt.visibility, got, tt.want)
			}
		})
	}
}
