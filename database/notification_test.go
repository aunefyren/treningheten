package database

import (
	"testing"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// makeSubscription persists a push subscription for the given user and returns it.
func makeSubscription(t *testing.T, userID uuid.UUID, endpoint string, sunday, achievement, news bool) models.Subscription {
	t.Helper()
	sub := models.Subscription{
		UserID:           userID,
		Endpoint:         endpoint,
		P256Dh:           "p256",
		Auth:             "auth",
		Enabled:          true,
		SundayAlert:      sunday,
		AchievementAlert: achievement,
		NewsAlert:        news,
	}
	sub.ID = uuid.New()
	id, err := CreateSubscriptionInDB(sub)
	if err != nil {
		t.Fatalf("CreateSubscriptionInDB returned error: %v", err)
	}
	if id != sub.ID {
		t.Fatalf("CreateSubscriptionInDB returned id %v, want %v", id, sub.ID)
	}
	return sub
}

func TestCreateAndGetSubscription(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "sub@example.com", nil)
	makeSubscription(t, user.ID, "endpoint-1", false, false, false)

	all, err := GetAllSubscriptionsForUserByUserID(user.ID)
	if err != nil {
		t.Fatalf("GetAllSubscriptionsForUserByUserID returned error: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("got %d subscriptions, want 1", len(all))
	}

	found, ok, err := GetAllSubscriptionForUserByUserIDAndEndpoint(user.ID, "endpoint-1")
	if err != nil {
		t.Fatalf("GetAllSubscriptionForUserByUserIDAndEndpoint returned error: %v", err)
	}
	if !ok || found.Endpoint != "endpoint-1" {
		t.Errorf("expected to find subscription, got ok=%v", ok)
	}

	_, ok, err = GetAllSubscriptionForUserByUserIDAndEndpoint(user.ID, "unknown")
	if err != nil {
		t.Fatalf("GetAllSubscriptionForUserByUserIDAndEndpoint(missing) returned error: %v", err)
	}
	if ok {
		t.Errorf("expected not found for unknown endpoint")
	}
}

func TestSubscriptionAlertFilters(t *testing.T) {
	newTestDB(t)

	user1 := makeTestUser(t, "alerts1@example.com", nil)
	user2 := makeTestUser(t, "alerts2@example.com", nil)

	makeSubscription(t, user1.ID, "sunday-ep", true, false, false)
	makeSubscription(t, user1.ID, "ach-ep", false, true, false)
	makeSubscription(t, user2.ID, "news-ep", false, false, true)

	sunday, ok, err := GetAllSubscriptionsForSundayAlerts()
	if err != nil {
		t.Fatalf("GetAllSubscriptionsForSundayAlerts returned error: %v", err)
	}
	if !ok || len(sunday) != 1 || sunday[0].Endpoint != "sunday-ep" {
		t.Errorf("sunday alerts: got ok=%v len=%d", ok, len(sunday))
	}

	ach, ok, err := GetAllSubscriptionsForAchievementsForUserID(user1.ID)
	if err != nil {
		t.Fatalf("GetAllSubscriptionsForAchievementsForUserID returned error: %v", err)
	}
	if !ok || len(ach) != 1 || ach[0].Endpoint != "ach-ep" {
		t.Errorf("achievement alerts: got ok=%v len=%d", ok, len(ach))
	}

	// user2 has no achievement subscription.
	_, ok, err = GetAllSubscriptionsForAchievementsForUserID(user2.ID)
	if err != nil {
		t.Fatalf("GetAllSubscriptionsForAchievementsForUserID(user2) returned error: %v", err)
	}
	if ok {
		t.Errorf("expected user2 to have no achievement subscriptions")
	}

	news, ok, err := GetAllSubscriptionsForNews()
	if err != nil {
		t.Fatalf("GetAllSubscriptionsForNews returned error: %v", err)
	}
	if !ok || len(news) != 1 || news[0].Endpoint != "news-ep" {
		t.Errorf("news alerts: got ok=%v len=%d", ok, len(news))
	}
}

func TestUpdateSubscriptionForUserByUserIDAndEndpoint(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "subupdate@example.com", nil)
	makeSubscription(t, user.ID, "ep", false, false, false)

	if err := UpdateSubscriptionForUserByUserIDAndEndpoint(user.ID, "ep", true, true, true); err != nil {
		t.Fatalf("UpdateSubscriptionForUserByUserIDAndEndpoint returned error: %v", err)
	}

	found, ok, err := GetAllSubscriptionForUserByUserIDAndEndpoint(user.ID, "ep")
	if err != nil || !ok {
		t.Fatalf("failed to reload subscription: err=%v ok=%v", err, ok)
	}
	if !found.SundayAlert || !found.AchievementAlert || !found.NewsAlert {
		t.Errorf("expected all alerts enabled, got sunday=%v ach=%v news=%v", found.SundayAlert, found.AchievementAlert, found.NewsAlert)
	}

	// Updating a non-existent subscription must error.
	if err := UpdateSubscriptionSundayReminderByEndpointAndUserID(user.ID, "no-such-ep", true); err == nil {
		t.Errorf("expected error updating unknown subscription")
	}
}

func TestUpdateSubscriptionSave(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "subsave@example.com", nil)
	sub := makeSubscription(t, user.ID, "ep-save", false, false, false)

	sub.Endpoint = "ep-renamed"
	sub.NewsAlert = true
	updated, err := UpdateSubscription(sub)
	if err != nil {
		t.Fatalf("UpdateSubscription returned error: %v", err)
	}
	if updated.Endpoint != "ep-renamed" {
		t.Errorf("endpoint: got %q, want %q", updated.Endpoint, "ep-renamed")
	}

	found, ok, err := GetAllSubscriptionForUserByUserIDAndEndpoint(user.ID, "ep-renamed")
	if err != nil || !ok {
		t.Fatalf("failed to reload renamed subscription: err=%v ok=%v", err, ok)
	}
	if !found.NewsAlert {
		t.Errorf("expected news alert persisted after save")
	}
}
