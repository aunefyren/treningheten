package controllers

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func TestGoalInviteGearNewsEdges(t *testing.T) {
	h := newAPIHarness(t)
	_, adminToken := h.user("admin@edges2.test", true)
	user, token := h.user("user@edges2.test", false)

	// Goals: unknown season, a started season that doesn't allow late joining, unknown goal.
	h.expect(http.StatusBadRequest, "POST", "/api/auth/goals", token, models.GoalCreationRequest{ExerciseInterval: 2, SeasonID: uuid.New()})
	closed := seedSeason(t, "Closed season", time.Now().AddDate(0, 0, -7), time.Now().AddDate(0, 0, 30), false)
	h.expect(http.StatusBadRequest, "POST", "/api/auth/goals", token, models.GoalCreationRequest{ExerciseInterval: 2, SeasonID: closed.ID})
	if code := h.do("DELETE", "/api/auth/goals/"+uuid.NewString(), token, nil).Code; code < 400 {
		t.Errorf("withdrawing an unknown goal: status = %d", code)
	}

	// A used invite can't be deleted.
	code, _ := database.GenerateRandomInvite()
	h.expect(http.StatusCreated, "POST", "/api/open/users", "", models.UserCreationRequest{
		FirstName: "In", LastName: "Vited", Email: "invited@edges2.test", Password: "Password123", PasswordRepeat: "Password123", InviteCode: code,
	})
	for _, invite := range field(t, h.ok("GET", "/api/admin/invites", adminToken, nil), "invites").([]any) {
		if field(t, invite, "code") == code {
			if code := h.do("DELETE", "/api/admin/invites/"+idOf(t, invite), adminToken, nil).Code; code != http.StatusConflict {
				t.Errorf("deleting a used invite: status = %d, want %d", code, http.StatusConflict)
			}
		}
	}

	// Gear: Strava-owned identity is read-only; names can't be blank.
	stravaID := "g999"
	stravaGear := models.Gear{UserID: user.ID, Name: "From Strava", Type: "shoe", StravaGearID: &stravaID, Enabled: true}
	stravaGear.ID = uuid.New()
	if err := database.Instance.Create(&stravaGear).Error; err != nil {
		t.Fatal(err)
	}
	h.expect(http.StatusBadRequest, "PUT", "/api/auth/gear/"+stravaGear.ID.String(), token, models.GearUpdateRequest{Name: stringPtr("Renamed")})
	h.ok("PUT", "/api/auth/gear/"+stravaGear.ID.String(), token, models.GearUpdateRequest{Retired: boolPtr(true)})
	manual := h.expect(http.StatusCreated, "POST", "/api/auth/gear", token, models.GearCreationRequest{Name: "Manual", Type: "bike"})
	h.expect(http.StatusBadRequest, "PUT", "/api/auth/gear/"+idOf(t, manual, "gear"), token, models.GearUpdateRequest{Name: stringPtr("   ")})

	// News needs a real body; a day note has a length limit.
	h.expect(http.StatusBadRequest, "POST", "/api/admin/news", adminToken, models.NewsCreationRequest{Title: "Title", Body: "abc", Date: time.Now()})
	day := h.ok("GET", "/api/auth/exercise-days/week?today=true", token, nil)
	h.expect(http.StatusBadRequest, "POST", "/api/auth/exercise-days/"+idOf(t, day, "exercise"), token, models.ExerciseDayUpdateRequest{Note: strings.Repeat("n", 300)})

	// Exercise days for a goal that isn't the caller's.
	if code := h.do("GET", "/api/auth/exercise-days?goal="+uuid.NewString(), token, nil).Code; code < 400 {
		t.Errorf("exercise days for an unknown goal: status = %d", code)
	}
}

func TestMediaAdminAndConnectEdges(t *testing.T) {
	h := newAPIHarness(t)
	withMedia(t)
	_, adminToken := h.user("admin@media2.test", true)
	_, token := h.user("user@media2.test", false)

	h.expect(http.StatusBadRequest, "POST", "/api/auth/media/audiobookshelf/connect", token, models.AudiobookshelfConnectRequest{ServerURL: "https://abs.example", Token: " "})
	h.expect(http.StatusBadRequest, "POST", "/api/auth/media/audiobookshelf/connect", token, "{broken")
	// With no targets, the admin sync covers every user with a media connection (none).
	h.expect(http.StatusAccepted, "POST", "/api/admin/media/sync-for-users", adminToken, models.MediaSyncForUsersRequest{})
	h.expect(http.StatusNotFound, "POST", "/api/admin/media/sync-for-users", adminToken, models.MediaSyncForUsersRequest{UserIDs: []string{uuid.NewString()}})
}
