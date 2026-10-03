package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func TestDisabledUserLookups(t *testing.T) {
	newTestDB(t)

	active := makeTestUser(t, "active@example.com", func(user *models.User) { user.FirstName = "Bea" })
	disabled := makeTestUser(t, "disabled@example.com", func(user *models.User) { user.FirstName = "Al" })
	if err := SetUserEnabled(disabled.ID, false); err != nil {
		t.Fatalf("SetUserEnabled returned error: %v", err)
	}

	t.Run("enabled-only lookup misses the disabled user", func(t *testing.T) {
		if _, err := GetUserInformation(disabled.ID); err == nil {
			t.Errorf("GetUserInformation found a disabled user")
		}
	})

	t.Run("including-disabled lookup finds them, censored", func(t *testing.T) {
		user, err := GetUserInformationIncludingDisabled(disabled.ID)
		if err != nil {
			t.Fatalf("GetUserInformationIncludingDisabled returned error: %v", err)
		}
		if user.ID != disabled.ID {
			t.Errorf("got user %v, want %v", user.ID, disabled.ID)
		}
		assertOnlyPublicFields(t, user, "GetUserInformationIncludingDisabled")

		if _, err := GetUserInformationIncludingDisabled(uuid.New()); err == nil {
			t.Errorf("expected an error for an unknown user")
		}
	})

	t.Run("full-row getter returns nil on miss", func(t *testing.T) {
		user, err := GetUserByIDIncludingDisabled(disabled.ID)
		if err != nil || user == nil || user.Enabled {
			t.Fatalf("GetUserByIDIncludingDisabled: got (%+v, %v), want the disabled row", user, err)
		}
		missing, err := GetUserByIDIncludingDisabled(uuid.New())
		if err != nil || missing != nil {
			t.Errorf("unknown user: got (%v, %v), want (nil, nil)", missing, err)
		}
	})

	t.Run("admin list includes everyone, ordered by name", func(t *testing.T) {
		users, err := GetAllUsersIncludingDisabled()
		if err != nil {
			t.Fatalf("GetAllUsersIncludingDisabled returned error: %v", err)
		}
		if len(users) != 2 || users[0].ID != disabled.ID || users[1].ID != active.ID {
			t.Errorf("got %d users in the wrong order: %+v", len(users), users)
		}
	})

	t.Run("re-enabling and repeating are both fine", func(t *testing.T) {
		for _, enabled := range []bool{true, true} {
			if err := SetUserEnabled(disabled.ID, enabled); err != nil {
				t.Fatalf("SetUserEnabled(%v) returned error: %v", enabled, err)
			}
		}
		if _, err := GetUserInformation(disabled.ID); err != nil {
			t.Errorf("re-enabled user not found: %v", err)
		}
	})
}

func TestToAdminUser(t *testing.T) {
	user := models.User{FirstName: "Ada", LastName: "Admin", Email: "ada@example.com", Password: "secret", Admin: boolPtr(true), Enabled: true, Verified: true}
	user.ID = uuid.New()

	adminUser := ToAdminUser(user)
	want := models.AdminUser{ID: user.ID, FirstName: "Ada", LastName: "Admin", Email: "ada@example.com", Admin: user.Admin, Enabled: true, Verified: true}
	if adminUser != want {
		t.Errorf("ToAdminUser: got %+v, want %+v", adminUser, want)
	}
}

func TestDisableGoalsForUserInUnfinishedSeasons(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "leaver@example.com", nil)
	other := makeTestUser(t, "stayer@example.com", nil)
	now := time.Now()

	finished := makeSeason(t, "Finished", now.AddDate(0, -3, 0), now.AddDate(0, -1, 0), true)
	ongoing := makeSeason(t, "Ongoing", now.AddDate(0, -1, 0), now.AddDate(0, 1, 0), true)
	upcoming := makeSeason(t, "Upcoming", now.AddDate(0, 1, 0), now.AddDate(0, 2, 0), true)

	finishedGoal := makeGoal(t, user.ID, finished.ID, true)
	ongoingGoal := makeGoal(t, user.ID, ongoing.ID, true)
	upcomingGoal := makeGoal(t, user.ID, upcoming.ID, true)
	alreadyOff := makeGoal(t, user.ID, ongoing.ID, false)
	otherGoal := makeGoal(t, other.ID, ongoing.ID, true)

	disabled, err := DisableGoalsForUserInUnfinishedSeasons(user.ID, now)
	if err != nil {
		t.Fatalf("DisableGoalsForUserInUnfinishedSeasons returned error: %v", err)
	}
	if disabled != 2 {
		t.Errorf("disabled %d goals, want 2", disabled)
	}

	cases := []struct {
		name    string
		goalID  uuid.UUID
		enabled bool
	}{
		{"finished season kept", finishedGoal.ID, true},
		{"ongoing season left", ongoingGoal.ID, false},
		{"upcoming season left", upcomingGoal.ID, false},
		{"already disabled stays disabled", alreadyOff.ID, false},
		{"other user untouched", otherGoal.ID, true},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			var goal models.Goal
			if err := Instance.Where("goals.id = ?", testCase.goalID).First(&goal).Error; err != nil {
				t.Fatal(err)
			}
			if goal.Enabled != testCase.enabled {
				t.Errorf("enabled: got %v, want %v", goal.Enabled, testCase.enabled)
			}
		})
	}

	again, err := DisableGoalsForUserInUnfinishedSeasons(user.ID, now)
	if err != nil || again != 0 {
		t.Errorf("second run: got (%d, %v), want (0, nil)", again, err)
	}
}
