package controllers

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/google/uuid"
)

// withVAPIDKeys installs a freshly generated VAPID key pair for the test.
func withVAPIDKeys(t *testing.T) {
	t.Helper()

	private, public, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("failed to generate VAPID keys: %v", err)
	}
	previous := files.ConfigFile
	files.ConfigFile.VAPIDPublicKey = public
	files.ConfigFile.VAPIDSecretKey = private
	files.ConfigFile.VAPIDContact = "mailto:admin@test.local"
	files.ConfigFile.TreninghetenEnvironment = "production"
	t.Cleanup(func() { files.ConfigFile = previous })
}

// pushEndpoint is a fake push service that answers every delivery with status.
func pushEndpoint(t *testing.T, status int) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	var deliveries atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if injectUpstreamFault(writer) {
			return
		}
		if request.Header.Get("Authorization") == "" || request.Header.Get("Content-Encoding") != "aes128gcm" {
			t.Errorf("push delivery without VAPID auth or encryption: %v", request.Header)
		}
		deliveries.Add(1)
		writer.WriteHeader(status)
	}))
	t.Cleanup(server.Close)
	return server, &deliveries
}

// browserSubscription builds a subscription with real P-256 / auth keys, as a browser's
// PushManager would hand out, so the payload encryption actually runs.
func browserSubscription(t *testing.T, endpoint string) models.SubscriptionOriginal {
	t.Helper()

	key, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	secret := make([]byte, 16)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}
	return models.SubscriptionOriginal{
		Endpoint: endpoint,
		Keys: models.SubscriptionOriginalKeys{
			P256Dh: base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes()),
			Auth:   base64.RawURLEncoding.EncodeToString(secret),
		},
	}
}

func TestPushSubscriptionsAndDelivery(t *testing.T) {
	h := newAPIHarness(t)
	withVAPIDKeys(t)

	_, adminToken := h.user("admin@push.test", true)
	user, token := h.user("phone@push.test", false)

	live, liveDeliveries := pushEndpoint(t, http.StatusCreated)
	gone, goneDeliveries := pushEndpoint(t, http.StatusGone)

	subscribe := func(endpoint string) int {
		request := models.SubscriptionCreationRequest{Subscription: browserSubscription(t, endpoint)}
		request.Settings.SundayAlert = true
		request.Settings.AchievementAlert = true
		request.Settings.NewsAlert = true
		return h.do("POST", "/api/auth/notifications/subscribe", token, request).Code
	}
	if code := subscribe(live.URL); code != http.StatusCreated {
		t.Fatalf("subscribe: status = %d, want 201", code)
	}
	// Subscribing the same endpoint again updates it in place.
	if code := subscribe(live.URL); code != http.StatusOK {
		t.Fatalf("re-subscribe: status = %d, want 200", code)
	}
	if code := subscribe(gone.URL); code != http.StatusCreated {
		t.Fatalf("subscribe second device: status = %d, want 201", code)
	}
	h.expect(http.StatusBadRequest, "POST", "/api/auth/notifications/subscribe", token, "{nope")

	found := h.expect(http.StatusCreated, "POST", "/api/auth/notifications/subscription", token, models.SubscriptionGetRequest{Endpoint: live.URL})
	if field(t, found, "subscription", "news_alert") != true {
		t.Errorf("news_alert not stored: %v", found)
	}
	h.expect(http.StatusBadRequest, "POST", "/api/auth/notifications/subscription", token, models.SubscriptionGetRequest{Endpoint: "https://unknown.test"})
	h.expect(http.StatusCreated, "POST", "/api/auth/notifications/subscription/update", token, models.SubscriptionUpdateRequest{
		Endpoint: live.URL, SundayAlert: true, AchievementAlert: true, NewsAlert: true,
	})

	// An admin test push reaches both devices; the one answering 410 is then disabled.
	pushed := h.expect(http.StatusCreated, "POST", "/api/admin/notifications/push/all-devices", adminToken, models.NotificationCreationRequest{
		Title: "Hi", Body: "Test", UserID: user.ID, Category: "test",
	})
	if pushed["amount"] != float64(1) {
		t.Errorf("amount = %v, want 1 (the gone device doesn't count)", pushed["amount"])
	}
	if liveDeliveries.Load() != 1 || goneDeliveries.Load() != 1 {
		t.Fatalf("deliveries live/gone = %d/%d, want 1/1", liveDeliveries.Load(), goneDeliveries.Load())
	}

	// Every notification kind reaches the live device only.
	debt := models.Debt{}
	debt.ID = uuid.New()
	for name, push := range map[string]func() error{
		"achievements":     func() error { return PushNotificationsForAchievements(user.ID) },
		"news":             PushNotificationsForNews,
		"sunday alerts":    PushNotificationsForSundayAlerts,
		"week lost":        func() error { return PushNotificationsForWeekLost(user.ID) },
		"wheel spin":       func() error { return PushNotificationsForWheelSpin(user.ID, debt) },
		"wheel spin check": func() error { return PushNotificationsForWheelSpinCheck(user.ID, debt) },
		"wheel spin win":   func() error { return PushNotificationsForWheelSpinWin(user.ID, debt) },
	} {
		before := liveDeliveries.Load()
		if err := push(); err != nil {
			t.Errorf("%s: %v", name, err)
		}
		if liveDeliveries.Load() != before+1 {
			t.Errorf("%s: live deliveries %d → %d, want one more", name, before, liveDeliveries.Load())
		}
	}
	if goneDeliveries.Load() != 1 {
		t.Errorf("the disabled device kept receiving pushes: %d", goneDeliveries.Load())
	}

	// The test environment mutes pushes entirely.
	files.ConfigFile.TreninghetenEnvironment = "test"
	before := liveDeliveries.Load()
	_ = PushNotificationsForNews()
	_ = PushNotificationsForAchievements(user.ID)
	if liveDeliveries.Load() != before {
		t.Error("pushes were sent in the test environment")
	}
}
