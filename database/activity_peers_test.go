package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// containsUserID reports whether the id is in the set.
func containsUserID(ids []uuid.UUID, id uuid.UUID) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}

// TestGetSeasonPeerUserIDsForUser covers the implicit social relation behind the front-page
// activity feed: everyone you have ever shared a season with, whether or not that season is
// still running, plus yourself. There is no friend-request system, so this query *is* the
// visibility rule.
func TestGetSeasonPeerUserIDsForUser(t *testing.T) {
	newTestDB(t)

	me := makeTestUser(t, "me@example.com", nil)
	pastMate := makeTestUser(t, "past@example.com", nil)
	currentMate := makeTestUser(t, "current@example.com", nil)
	stranger := makeTestUser(t, "stranger@example.com", nil)

	lastYear := time.Now().AddDate(-1, 0, 0)
	pastSeason := makeSeason(t, "Past season", lastYear, lastYear.AddDate(0, 2, 0), true)
	currentSeason := makeSeason(t, "Current season", time.Now().AddDate(0, -1, 0), time.Now().AddDate(0, 1, 0), true)
	otherSeason := makeSeason(t, "A season I never joined", lastYear, lastYear.AddDate(0, 2, 0), true)

	makeGoal(t, me.ID, pastSeason.ID, true)
	makeGoal(t, pastMate.ID, pastSeason.ID, true)
	makeGoal(t, me.ID, currentSeason.ID, true)
	makeGoal(t, currentMate.ID, currentSeason.ID, true)
	makeGoal(t, stranger.ID, otherSeason.ID, true)

	peerIDs, err := GetSeasonPeerUserIDsForUser(me.ID)
	if err != nil {
		t.Fatalf("GetSeasonPeerUserIDsForUser returned error: %v", err)
	}

	if !containsUserID(peerIDs, me.ID) {
		t.Errorf("the user is not their own peer, so their activities would be missing from their feed")
	}
	// A finished season still counts — that is the whole point of the widened feed.
	if !containsUserID(peerIDs, pastMate.ID) {
		t.Errorf("a past season-mate is not a peer")
	}
	if !containsUserID(peerIDs, currentMate.ID) {
		t.Errorf("a current season-mate is not a peer")
	}
	if containsUserID(peerIDs, stranger.ID) {
		t.Errorf("a user from a season the caller never joined is a peer")
	}
	if len(peerIDs) != 3 {
		t.Errorf("got %d peers %v, want exactly the 3 expected", len(peerIDs), peerIDs)
	}
}

// TestGetSeasonPeerUserIDsForUserDeduplicates covers that sharing several seasons with the
// same person yields one peer entry, not one per shared season — the id list goes straight
// into an IN clause.
func TestGetSeasonPeerUserIDsForUserDeduplicates(t *testing.T) {
	newTestDB(t)

	me := makeTestUser(t, "dedupe-me@example.com", nil)
	mate := makeTestUser(t, "dedupe-mate@example.com", nil)

	for i := 0; i < 3; i++ {
		season := makeSeason(t, "Season", time.Now().AddDate(0, -i-1, 0), time.Now().AddDate(0, -i, 0), true)
		makeGoal(t, me.ID, season.ID, true)
		makeGoal(t, mate.ID, season.ID, true)
	}

	peerIDs, err := GetSeasonPeerUserIDsForUser(me.ID)
	if err != nil {
		t.Fatalf("GetSeasonPeerUserIDsForUser returned error: %v", err)
	}
	if len(peerIDs) != 2 {
		t.Errorf("got %d peers %v, want 2 (self + the mate, deduplicated)", len(peerIDs), peerIDs)
	}
}

// TestGetSeasonPeerUserIDsForUserIgnoresDisabledGoals covers that a withdrawn membership
// (a disabled Goal) does not create or preserve a peer relation, on either side.
func TestGetSeasonPeerUserIDsForUserIgnoresDisabledGoals(t *testing.T) {
	newTestDB(t)

	me := makeTestUser(t, "disabled-me@example.com", nil)
	withdrawnMate := makeTestUser(t, "withdrawn@example.com", nil)
	season := makeSeason(t, "Season", time.Now().AddDate(0, -1, 0), time.Now().AddDate(0, 1, 0), true)

	makeGoal(t, me.ID, season.ID, true)
	makeGoal(t, withdrawnMate.ID, season.ID, false)

	peerIDs, err := GetSeasonPeerUserIDsForUser(me.ID)
	if err != nil {
		t.Fatalf("GetSeasonPeerUserIDsForUser returned error: %v", err)
	}
	if containsUserID(peerIDs, withdrawnMate.ID) {
		t.Errorf("a user whose goal is disabled is still a peer")
	}

	// And the reverse: my own disabled goal doesn't make their season-mates my peers.
	otherSeason := makeSeason(t, "Other season", time.Now().AddDate(0, -1, 0), time.Now().AddDate(0, 1, 0), true)
	otherMate := makeTestUser(t, "othermate@example.com", nil)
	makeGoal(t, me.ID, otherSeason.ID, false)
	makeGoal(t, otherMate.ID, otherSeason.ID, true)

	peerIDs, err = GetSeasonPeerUserIDsForUser(me.ID)
	if err != nil {
		t.Fatalf("GetSeasonPeerUserIDsForUser returned error: %v", err)
	}
	if containsUserID(peerIDs, otherMate.ID) {
		t.Errorf("a disabled membership of mine still exposed that season's members")
	}
}

// TestGetSeasonPeerUserIDsForUserWithoutSeasons covers a brand-new user: no seasons yet, so
// no peers — but still themselves, so their own activities show on their front page.
func TestGetSeasonPeerUserIDsForUserWithoutSeasons(t *testing.T) {
	newTestDB(t)

	me := makeTestUser(t, "newcomer@example.com", nil)

	peerIDs, err := GetSeasonPeerUserIDsForUser(me.ID)
	if err != nil {
		t.Fatalf("GetSeasonPeerUserIDsForUser returned error: %v", err)
	}
	if len(peerIDs) != 1 || peerIDs[0] != me.ID {
		t.Errorf("peers = %v, want just the user themselves", peerIDs)
	}
}

// TestGetExerciseDaysForSharingUsersInListUsingDates covers the feed's day query: it is
// restricted to the given users, still honours the per-user sharing opt-out, and stays
// inside the requested week.
func TestGetExerciseDaysForSharingUsersInListUsingDates(t *testing.T) {
	newTestDB(t)

	sharing := makeTestUser(t, "sharing@example.com", nil)
	notSharing := makeTestUser(t, "notsharing@example.com", func(user *models.User) {
		user.ShareActivities = boolPtr(false)
	})
	outsider := makeTestUser(t, "outsider@example.com", nil)

	// Force the opt-out: a false zero value is dropped by the insert (default:true).
	if err := Instance.Model(&models.User{}).Where("id = ?", notSharing.ID).Update("share_activities", false).Error; err != nil {
		t.Fatalf("failed to clear share_activities: %v", err)
	}

	monday := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	sunday := monday.AddDate(0, 0, 6)

	makeDay(t, sharing.ID, monday.AddDate(0, 0, 2))    // in the week
	makeDay(t, sharing.ID, monday.AddDate(0, 0, -3))   // the week before
	makeDay(t, notSharing.ID, monday.AddDate(0, 0, 2)) // opted out
	makeDay(t, outsider.ID, monday.AddDate(0, 0, 2))   // not a peer

	days, err := GetExerciseDaysForSharingUsersInListUsingDates([]uuid.UUID{sharing.ID, notSharing.ID}, monday, sunday)
	if err != nil {
		t.Fatalf("GetExerciseDaysForSharingUsersInListUsingDates returned error: %v", err)
	}

	if len(days) != 1 {
		t.Fatalf("got %d days %v, want only the sharing peer's in-week day", len(days), days)
	}
	if days[0].UserID == nil || *days[0].UserID != sharing.ID {
		t.Errorf("day belongs to %v, want the sharing peer", days[0].UserID)
	}

	// An empty peer list must return nothing rather than degrading to "everyone".
	days, err = GetExerciseDaysForSharingUsersInListUsingDates(nil, monday, sunday)
	if err != nil {
		t.Fatalf("GetExerciseDaysForSharingUsersInListUsingDates returned error: %v", err)
	}
	if len(days) != 0 {
		t.Errorf("an empty peer list returned %d days, want none", len(days))
	}
}
