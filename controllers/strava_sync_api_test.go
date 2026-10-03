package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"
)

// fakeStrava serves just enough of the Strava API for a week sync: the token exchange,
// the activity list, activity detail, streams and gear. revoke makes the token endpoint
// answer 401 (a revoked connection).
type fakeStrava struct {
	activities []models.StravaGetActivitiesRequestReply
	revoke     atomic.Bool
	tokenCalls atomic.Int32
	gearCalls  atomic.Int32
}

func (f *fakeStrava) handler(t *testing.T) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if injectUpstreamFault(writer) {
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		path := request.URL.Path

		switch {
		case path == "/oauth/token":
			f.tokenCalls.Add(1)
			if f.revoke.Load() {
				writer.WriteHeader(http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"access_token": "access-" + request.FormValue("grant_type"), "refresh_token": "refresh-token",
				"expires_at": time.Now().Add(time.Hour).Unix(), "athlete": map[string]any{"id": 4242},
			})
		case request.Header.Get("Authorization") == "":
			t.Errorf("Strava API call without a bearer token: %s", path)
			writer.WriteHeader(http.StatusUnauthorized)
		case path == "/api/v3/athlete/activities":
			if request.URL.Query().Get("page") != "" && request.URL.Query().Get("page") != "1" {
				_, _ = writer.Write([]byte("[]"))
				return
			}
			_ = json.NewEncoder(writer).Encode(f.activities)
		case strings.HasSuffix(path, "/streams"):
			_ = json.NewEncoder(writer).Encode(richStravaStreams())
		case strings.HasPrefix(path, "/api/v3/activities/"):
			for _, activity := range f.activities {
				if strings.HasSuffix(path, "/"+jsonNumber(activity.ID)) {
					description := "Felt strong"
					activity.Description = &description
					_ = json.NewEncoder(writer).Encode(activity)
					return
				}
			}
			writer.WriteHeader(http.StatusNotFound)
		case strings.HasPrefix(path, "/api/v3/gear/"):
			f.gearCalls.Add(1)
			_ = json.NewEncoder(writer).Encode(models.StravaGear{ID: strings.TrimPrefix(path, "/api/v3/gear/"), Name: "Race shoes", BrandName: "Brand", ModelName: "Fast"})
		default:
			t.Errorf("unexpected Strava request: %s %s", request.Method, path)
			writer.WriteHeader(http.StatusNotFound)
		}
	}
}

func jsonNumber(id int64) string {
	encoded, _ := json.Marshal(id)
	return string(encoded)
}

// todayAt returns a time today at the given minute past midnight, so activities stay on
// today's date whatever time the test runs.
func todayAt(minute int) time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, minute, 0, 0, time.Local)
}

func stravaActivity(id int64, sportType string, start time.Time, gearID *string, commute bool) models.StravaGetActivitiesRequestReply {
	activity := models.StravaGetActivitiesRequestReply{
		ID: id, Name: sportType + " activity", SportType: sportType, Type: sportType,
		Distance: 5000, MovingTime: 1500, ElapsedTime: 1600, TotalElevationGain: 30,
		StartDate:      start.UTC(),
		StartDateLocal: time.Date(start.Year(), start.Month(), start.Day(), start.Hour(), start.Minute(), 0, 0, time.UTC),
		Timezone:       "(GMT+01:00) Europe/Oslo",
		GearID:         gearID, Commute: commute, Visibility: "everyone",
	}
	activity.Athlete.ID = 4242
	return activity
}

func withStrava(t *testing.T) *fakeStrava {
	t.Helper()

	fake := &fakeStrava{}
	stubStrava(t, fake.handler(t))
	// The client-side rate limiter is process-wide (90 calls / 15 min, sized for the real
	// Strava). Every fake is a fresh "Strava", so start each test with an empty window —
	// otherwise a long run of tests blocks in stravaWait for up to 15 minutes.
	stravaRateMu.Lock()
	stravaRateTimes = nil
	stravaRateMu.Unlock()
	previous := files.ConfigFile
	files.ConfigFile.StravaEnabled = true
	key, _ := files.GenerateSecureKey(32)
	files.ConfigFile.StravaTokenKey = key
	t.Cleanup(func() {
		files.ConfigFile.StravaEnabled = previous.StravaEnabled
		files.ConfigFile.StravaTokenKey = previous.StravaTokenKey
	})
	return fake
}

func TestStravaConnectSyncAndReshape(t *testing.T) {
	h := newAPIHarness(t)
	fake := withStrava(t)

	gearID := "g1234"
	fake.activities = []models.StravaGetActivitiesRequestReply{
		stravaActivity(1001, "Run", todayAt(1), &gearID, false),
		stravaActivity(1002, "Ride", todayAt(2), nil, true),
	}

	_, adminToken := h.user("admin@strava.test", true)
	user, token := h.user("athlete@strava.test", false)
	userPath := "/api/auth/users/" + user.ID.String()

	// Connecting stores the one-time code and runs the first sync right away.
	h.ok("POST", userPath+"/strava", token, models.UserStravaCodeUpdateRequest{StravaCode: "one-time-code"})

	stored, err := database.GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.StravaID == nil || *stored.StravaID != "4242" {
		t.Errorf("strava id = %v, want 4242 from the athlete object", stored.StravaID)
	}
	if stored.StravaCode == nil || !strings.HasPrefix(*stored.StravaCode, "r:") || strings.Contains(*stored.StravaCode, "refresh-token") {
		t.Errorf("stored credential = %v, want an encrypted r: refresh token", stored.StravaCode)
	}

	operations := h.ok("GET", "/api/auth/operations", token, nil)
	if got := len(field(t, operations, "operations").([]any)); got != 2 {
		t.Fatalf("operations after sync = %d, want one per Strava activity", got)
	}

	// Gear was resolved from Strava once and linked.
	gear := h.ok("GET", "/api/auth/gear", token, nil)
	gearList := field(t, gear, "gear").([]any)
	if len(gearList) != 1 || field(t, gearList[0], "name") != "Race shoes" {
		t.Errorf("gear = %v, want the Strava shoe", gearList)
	}

	// A second sync (now via the stored refresh token) is idempotent.
	if err := StravaSyncWeekForUser(stored, time.Now()); err != nil {
		t.Fatalf("resync: %v", err)
	}
	operations = h.ok("GET", "/api/auth/operations", token, nil)
	operationList := field(t, operations, "operations").([]any)
	if len(operationList) != 2 {
		t.Errorf("operations after resync = %d, want still 2", len(operationList))
	}
	if fake.gearCalls.Load() != 1 {
		t.Errorf("gear fetched %d times, want once (then cached locally)", fake.gearCalls.Load())
	}

	// The commute is tagged from the list payload.
	sawCommute := false
	for _, operation := range operationList {
		if tags, _ := field(t, operation, "tags").([]any); len(tags) > 0 && tags[0] == "commute" {
			sawCommute = true
		}
	}
	if !sawCommute {
		t.Error("the commute activity was not tagged")
	}

	// Combine the two sessions of the day into one, then split them again.
	// Combine takes the Strava activity ids, not session ids.
	stravaIDs := []string{"1001", "1002"}
	_, strangerToken := h.user("stranger@strava.test", false)
	h.expect(http.StatusBadRequest, "POST", "/api/auth/exercises/strava-combine", strangerToken, stravaIDs)
	h.expect(http.StatusCreated, "POST", "/api/auth/exercises/strava-combine", token, stravaIDs)

	combined := h.ok("GET", "/api/auth/operations", token, nil)
	combinedList := field(t, combined, "operations").([]any)
	masterSession := field(t, combinedList[0], "exercise").(string)
	for _, operation := range combinedList {
		if field(t, operation, "exercise") != masterSession {
			t.Fatal("combine left the activities in different sessions")
		}
	}
	h.expect(http.StatusCreated, "POST", "/api/auth/exercises/"+masterSession+"/strava-divide", token, nil)
	h.expect(http.StatusBadRequest, "POST", "/api/auth/exercises/nope/strava-divide", token, nil)

	// Re-pull one set's detail from Strava.
	operationID := idOf(t, combinedList[0])
	sets := h.ok("GET", "/api/auth/operation-sets?operation_id="+operationID, token, nil)
	setID := idOf(t, field(t, sets, "operation_sets").([]any)[0])
	h.ok("POST", "/api/auth/operation-sets/"+setID+"/strava-sync", token, nil)

	// Richer reads now that there are streams, gear and Strava metadata.
	h.ok("GET", "/api/auth/operations/"+operationID, token, nil)
	h.ok("GET", userPath+"/statistics", token, nil)
	h.ok("GET", userPath+"/activities", token, nil)
	h.ok("GET", "/api/auth/activities", token, nil)
	h.ok("POST", userPath+"/strava-sync?pointInTime="+strconv.FormatInt(time.Now().Unix(), 10), token, nil)

	// Admin bulk sync, by user and by Strava activity id.
	h.expect(http.StatusAccepted, "POST", "/api/admin/strava/sync-activities-for-users", adminToken, models.StravaSyncActivitiesForUsersRequest{UserIDs: []string{user.ID.String()}})
	SyncStravaActivitiesForUsers([]models.User{stored}, []string{"1001"})
	StravaSyncWeekForAllUsers()

	// A revoked connection is kept, but marked broken so the user is told to reconnect
	// (it used to be cleared, which looked like "never connected" on the account page).
	withInlineIntegrationRecovery(t)
	fake.revoke.Store(true)
	refreshed, _ := database.GetAllUserInformation(user.ID)
	if err := StravaSyncWeekForUser(refreshed, time.Now()); !errors.Is(err, ErrStravaSessionInvalid) {
		t.Fatalf("sync against a revoked token = %v, want ErrStravaSessionInvalid", err)
	}
	kept, _ := database.GetAllUserInformation(user.ID)
	if kept.StravaCode == nil || kept.StravaID == nil {
		t.Error("a revoked Strava connection was cleared; it should be kept and marked broken")
	}
	self := h.ok("GET", userPath, token, nil)
	if field(t, self, "user", "integration_health", "strava", "status") != models.IntegrationStatusAuthFailed {
		t.Errorf("own user health = %v, want strava auth_failed", field(t, self, "user", "integration_health"))
	}

	// Once Strava accepts the credential again, the next sync clears the status.
	fake.revoke.Store(false)
	if err := StravaSyncWeekForUser(kept, time.Now()); err != nil {
		t.Fatalf("sync after recovery: %v", err)
	}
	self = h.ok("GET", userPath, token, nil)
	if field(t, self, "user", "integration_health", "strava", "status") != models.IntegrationStatusOK {
		t.Errorf("own user health after recovery = %v", field(t, self, "user", "integration_health"))
	}

	// Disconnecting drops a status along with the connection.
	recordIntegrationFailure(user.ID, models.IntegrationProviderStrava, integrationAuthErrorFor(ErrStravaSessionInvalid, ""))
	h.ok("DELETE", userPath+"/strava", token, nil)
	if row, _ := database.GetIntegrationStatus(user.ID, models.IntegrationProviderStrava); row != nil {
		t.Errorf("status survived the Strava disconnect: %+v", row)
	}
}

func TestStravaEndpointsWhenDisabledOrDisconnected(t *testing.T) {
	h := newAPIHarness(t)
	user, token := h.user("nostrava@strava.test", false)
	userPath := "/api/auth/users/" + user.ID.String()

	h.expect(http.StatusBadRequest, "POST", userPath+"/strava", token, models.UserStravaCodeUpdateRequest{StravaCode: "x"})
	h.expect(http.StatusBadRequest, "POST", userPath+"/strava-sync", token, nil)

	withStrava(t)
	h.expect(http.StatusBadRequest, "POST", userPath+"/strava-sync", token, nil)
	h.expect(http.StatusBadRequest, "POST", userPath+"/strava-sync?pointInTime=never", token, nil)
	h.ok("DELETE", userPath+"/strava", token, nil)

	bad := "x:garbage"
	for _, code := range []*string{nil, &bad} {
		if _, err := StravaGetAuthorizationForUser(models.User{StravaCode: code}); err == nil {
			t.Errorf("StravaCode %v: expected an error", code)
		}
	}
}

// richStravaStreams is a 30-minute, one-sample-per-second run with every channel: a hill
// in the middle, a walk break and a full stop, heart rate drifting up, and cadence,
// power and temperature — enough for splits, zones, breaks and gradient analysis.
func richStravaStreams() models.StravaActivityStreams {
	const seconds = 1800
	streams := models.StravaActivityStreams{
		Time:           &models.StravaStream[int]{},
		Heartrate:      &models.StravaStream[int]{},
		LatLng:         &models.StravaStream[[]float64]{},
		Altitude:       &models.StravaStream[float64]{},
		VelocitySmooth: &models.StravaStream[float64]{},
		Cadence:        &models.StravaStream[int]{},
		Watts:          &models.StravaStream[int]{},
		Temp:           &models.StravaStream[int]{},
	}
	latitude, altitude := 59.9, 10.0
	for second := 0; second < seconds; second++ {
		speed := 3.2
		switch {
		case second >= 600 && second < 660:
			speed = 1.2 // walk break
		case second >= 1200 && second < 1230:
			speed = 0 // stopped at a light
		}
		switch {
		case second >= 800 && second < 1000:
			altitude += 0.1
		case second >= 1000 && second < 1100:
			altitude -= 0.2
		}
		latitude += speed / 111000
		streams.Time.Data = append(streams.Time.Data, second)
		streams.Heartrate.Data = append(streams.Heartrate.Data, 120+second/40)
		streams.LatLng.Data = append(streams.LatLng.Data, []float64{latitude, 10.7})
		streams.Altitude.Data = append(streams.Altitude.Data, altitude)
		streams.VelocitySmooth.Data = append(streams.VelocitySmooth.Data, speed)
		streams.Cadence.Data = append(streams.Cadence.Data, 80+second%5)
		streams.Watts.Data = append(streams.Watts.Data, 200+second%30)
		streams.Temp.Data = append(streams.Temp.Data, 12+second/600)
	}
	return streams
}
