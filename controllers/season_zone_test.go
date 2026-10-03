package controllers

import (
	"net/http"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"
	"github.com/aunefyren/treningheten/utilities"

	"github.com/google/uuid"
)

// TestSeasonCreatedFromAnotherZoneProcessesItsFinalWeek covers that a season is pinned to
// the server's zone whatever zone the admin's browser sends. Built in the request's zone,
// the final Sunday's 23:59:59 fell outside the Monday–Sunday window weekly processing
// computes in time.Local, so the last week came back empty and was never processed.
func TestSeasonCreatedFromAnotherZoneProcessesItsFinalWeek(t *testing.T) {
	h := newAPIHarness(t)
	_, adminToken := h.user("admin@zone.test", true)
	prizeID := seedPrize(t, "Zone prize")

	// A zone far from any CI or developer machine, so the request and server zones differ.
	auckland, err := time.LoadLocation("Pacific/Auckland")
	if err != nil {
		t.Skipf("zone database unavailable: %v", err)
	}
	monday := nextMonday(1)
	start := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, auckland)
	end := start.AddDate(0, 0, 13)

	h.expect(http.StatusCreated, "POST", "/api/admin/seasons", adminToken, models.SeasonCreationRequest{
		Name: "Antipodes season", Start: start, End: end, Prize: prizeID, TimeZone: "Pacific/Auckland",
	})

	var stored models.Season
	if err := database.Instance.Where("name = ?", "Antipodes season").First(&stored).Error; err != nil {
		t.Fatalf("season not stored: %v", err)
	}

	wantStart := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
	wantEnd := time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 59, time.Local)
	if !stored.Start.Equal(wantStart) {
		t.Errorf("start = %v, want the picked Monday at midnight server time %v", stored.Start, wantStart)
	}
	if !stored.End.Equal(wantEnd) {
		t.Errorf("end = %v, want the picked Sunday 23:59:59 server time %v", stored.End, wantEnd)
	}

	seasonObject, err := ConvertSeasonToSeasonObject(stored)
	if err != nil {
		t.Fatal(err)
	}
	finalMonday, err := utilities.FindEarlierMonday(stored.End)
	if err != nil {
		t.Fatal(err)
	}
	finalSunday, err := utilities.FindNextSunday(stored.End)
	if err != nil {
		t.Fatal(err)
	}
	weeks, err := RetrieveWeekResultsFromSeasonWithinTimeframe(finalMonday, finalSunday, seasonObject)
	if err != nil {
		t.Fatalf("RetrieveWeekResultsFromSeasonWithinTimeframe returned error: %v", err)
	}
	if len(weeks) != 1 {
		t.Errorf("final week results = %d weeks, want exactly 1", len(weeks))
	}
}

// TestSeasonBoundariesInServerZoneKeepsThePickedDates covers that only the zone changes:
// the calendar dates the admin picked survive whatever offset they arrived in.
func TestSeasonBoundariesInServerZoneKeepsThePickedDates(t *testing.T) {
	for _, zoneName := range []string{"UTC", "Pacific/Auckland", "America/Los_Angeles"} {
		t.Run(zoneName, func(t *testing.T) {
			zone, err := time.LoadLocation(zoneName)
			if err != nil {
				t.Skipf("zone database unavailable: %v", err)
			}
			start, end := seasonBoundariesInServerZone(
				time.Date(2026, 3, 2, 0, 0, 0, 0, zone),
				time.Date(2026, 3, 15, 0, 0, 0, 0, zone),
			)
			if start.Location() != time.Local || end.Location() != time.Local {
				t.Errorf("boundaries in %v / %v, want time.Local", start.Location(), end.Location())
			}
			if start.Format("2006-01-02 15:04:05") != "2026-03-02 00:00:00" {
				t.Errorf("start = %v", start)
			}
			if end.Format("2006-01-02 15:04:05") != "2026-03-15 23:59:59" {
				t.Errorf("end = %v", end)
			}
		})
	}
}

// TestFinalWeekResolvesOnlyInTheServerZone pins the mechanism behind the fix, with no
// database round-trip in the way: the same picked dates resolve to exactly one final week
// when built in the server's zone, and not when built in a zone far from it.
func TestFinalWeekResolvesOnlyInTheServerZone(t *testing.T) {
	newControllerTestDB(t)
	auckland, err := time.LoadLocation("Pacific/Auckland")
	if err != nil {
		t.Skipf("zone database unavailable: %v", err)
	}
	if _, offset := time.Now().In(time.Local).Zone(); offset >= 10*3600 {
		t.Skip("the server zone is too close to Auckland for the contrast to show")
	}

	monday := nextMonday(1)
	pickedStart := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
	pickedEnd := pickedStart.AddDate(0, 0, 13)

	finalWeeks := func(start, end time.Time) int {
		t.Helper()
		season := models.Season{Name: "Zone season", Start: start, End: end}
		season.ID = uuid.New()
		seasonObject, err := ConvertSeasonToSeasonObject(season)
		if err != nil {
			t.Fatal(err)
		}
		finalMonday, _ := utilities.FindEarlierMonday(end)
		finalSunday, _ := utilities.FindNextSunday(end)
		weeks, _ := RetrieveWeekResultsFromSeasonWithinTimeframe(finalMonday, finalSunday, seasonObject)
		return len(weeks)
	}

	serverStart, serverEnd := seasonBoundariesInServerZone(pickedStart, pickedEnd)
	if got := finalWeeks(serverStart, serverEnd); got != 1 {
		t.Errorf("server-zone season: final week results = %d, want 1", got)
	}

	foreignStart := time.Date(pickedStart.Year(), pickedStart.Month(), pickedStart.Day(), 0, 0, 0, 0, auckland)
	foreignEnd := time.Date(pickedEnd.Year(), pickedEnd.Month(), pickedEnd.Day(), 23, 59, 59, 59, auckland)
	if got := finalWeeks(foreignStart, foreignEnd); got == 1 {
		t.Error("an Auckland-zoned season resolved its final week too; the test no longer shows the zone mismatch")
	}
}
