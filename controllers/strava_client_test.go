package controllers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/aunefyren/treningheten/files"
)

// stubStrava points both Strava endpoints at a test server and installs app credentials,
// restoring everything afterwards.
func stubStrava(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(handler)

	previousAPI, previousOAuth := stravaAPIBaseURL, stravaOAuthTokenURL
	previousID, previousSecret := files.ConfigFile.StravaClientID, files.ConfigFile.StravaClientSecret
	stravaAPIBaseURL = server.URL + "/api/v3"
	stravaOAuthTokenURL = server.URL + "/oauth/token"
	files.ConfigFile.StravaClientID = "the-client-id"
	files.ConfigFile.StravaClientSecret = "the-client-secret"

	t.Cleanup(func() {
		stravaAPIBaseURL, stravaOAuthTokenURL = previousAPI, previousOAuth
		files.ConfigFile.StravaClientID, files.ConfigFile.StravaClientSecret = previousID, previousSecret
		server.Close()
	})

	return server
}

// TestStravaAuthorizeSessionInvalid covers the sentinel that decides whether a failed token
// exchange clears the user's connection. Strava answers 400/401 for a used authorization
// code or a revoked token — permanent failures the user must reconnect from — while a 429
// or 5xx is transient and must leave the stored credential alone.
func TestStravaAuthorizeSessionInvalid(t *testing.T) {
	permanent := []int{http.StatusBadRequest, http.StatusUnauthorized}
	for _, status := range permanent {
		t.Run("authorize "+http.StatusText(status), func(t *testing.T) {
			stubStrava(t, func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(status)
			})
			_, err := StravaAuthorize("the-code")
			if !errors.Is(err, ErrStravaSessionInvalid) {
				t.Errorf("error = %v, want ErrStravaSessionInvalid so the connection is cleared", err)
			}
		})

		t.Run("reauthorize "+http.StatusText(status), func(t *testing.T) {
			stubStrava(t, func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(status)
			})
			_, err := StravaReauthorize("the-refresh-token")
			if !errors.Is(err, ErrStravaSessionInvalid) {
				t.Errorf("error = %v, want ErrStravaSessionInvalid so the connection is cleared", err)
			}
		})
	}

	transient := []int{http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway}
	for _, status := range transient {
		t.Run("authorize "+http.StatusText(status)+" is transient", func(t *testing.T) {
			stubStrava(t, func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(status)
			})
			_, err := StravaAuthorize("the-code")
			if err == nil {
				t.Fatalf("status %d returned no error", status)
			}
			if errors.Is(err, ErrStravaSessionInvalid) {
				t.Errorf("status %d was treated as permanent; it would wrongly clear the connection", status)
			}
		})

		t.Run("reauthorize "+http.StatusText(status)+" is transient", func(t *testing.T) {
			stubStrava(t, func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(status)
			})
			_, err := StravaReauthorize("the-refresh-token")
			if err == nil {
				t.Fatalf("status %d returned no error", status)
			}
			if errors.Is(err, ErrStravaSessionInvalid) {
				t.Errorf("status %d was treated as permanent; it would wrongly clear the connection", status)
			}
		})
	}
}

// TestStravaAuthorize covers the successful code exchange: the app credentials and the
// authorization-code grant are sent, and the returned token set is parsed.
func TestStravaAuthorize(t *testing.T) {
	var gotQuery url.Values
	stubStrava(t, func(writer http.ResponseWriter, request *http.Request) {
		gotQuery = request.URL.Query()
		writer.Write([]byte(`{"access_token":"at","refresh_token":"rt","expires_at":1893456000,"athlete":{"id":12345}}`))
	})

	authorization, err := StravaAuthorize("the-code")
	if err != nil {
		t.Fatalf("StravaAuthorize returned error: %v", err)
	}
	if authorization.AccessToken != "at" || authorization.RefreshToken != "rt" {
		t.Errorf("authorization = %+v, want the parsed token set", authorization)
	}

	if gotQuery.Get("grant_type") != "authorization_code" {
		t.Errorf("grant_type = %q, want authorization_code", gotQuery.Get("grant_type"))
	}
	if gotQuery.Get("code") != "the-code" {
		t.Errorf("code = %q, want the authorization code", gotQuery.Get("code"))
	}
	if gotQuery.Get("client_id") != "the-client-id" || gotQuery.Get("client_secret") != "the-client-secret" {
		t.Errorf("app credentials were not sent: %v", gotQuery)
	}
}

// TestStravaReauthorizeUsesRefreshGrant covers that the refresh path sends the stored
// refresh token under the refresh_token grant — mixing the two grants up would make every
// token renewal fail and disconnect users.
func TestStravaReauthorizeUsesRefreshGrant(t *testing.T) {
	var gotQuery url.Values
	stubStrava(t, func(writer http.ResponseWriter, request *http.Request) {
		gotQuery = request.URL.Query()
		writer.Write([]byte(`{"access_token":"at","refresh_token":"rt2","expires_at":1893456000}`))
	})

	authorization, err := StravaReauthorize("the-refresh-token")
	if err != nil {
		t.Fatalf("StravaReauthorize returned error: %v", err)
	}
	if authorization.RefreshToken != "rt2" {
		t.Errorf("refresh token = %q, want the rotated token", authorization.RefreshToken)
	}
	if gotQuery.Get("grant_type") != "refresh_token" {
		t.Errorf("grant_type = %q, want refresh_token", gotQuery.Get("grant_type"))
	}
	if gotQuery.Get("refresh_token") != "the-refresh-token" {
		t.Errorf("refresh_token = %q, want the stored token", gotQuery.Get("refresh_token"))
	}
	if gotQuery.Get("code") != "" {
		t.Errorf("the refresh grant sent a code parameter: %v", gotQuery)
	}
}

// TestStravaGetActivities covers the activity list request: a bearer token, the caller's
// before/after window, and the paging parameters the sync relies on.
func TestStravaGetActivities(t *testing.T) {
	var gotPath, gotAuth string
	var gotQuery url.Values
	stubStrava(t, func(writer http.ResponseWriter, request *http.Request) {
		gotPath = request.URL.Path
		gotAuth = request.Header.Get("Authorization")
		gotQuery = request.URL.Query()
		writer.Write([]byte(`[
			{"id":111,"name":"Morning Walk","sport_type":"Walk","commute":true},
			{"id":222,"name":"Evening Run","sport_type":"Run"}
		]`))
	})

	activities, err := StravaGetActivities("the-access-token", 1754300000, 1754200000)
	if err != nil {
		t.Fatalf("StravaGetActivities returned error: %v", err)
	}
	if len(activities) != 2 {
		t.Fatalf("got %d activities, want 2", len(activities))
	}
	if activities[0].ID != 111 || activities[0].SportType != "Walk" || !activities[0].Commute {
		t.Errorf("first activity = %+v, want the parsed walk", activities[0])
	}
	if activities[1].SportType != "Run" {
		t.Errorf("second activity sport type = %q, want Run", activities[1].SportType)
	}

	if gotPath != "/api/v3/athlete/activities" {
		t.Errorf("requested %q, want the athlete activities endpoint", gotPath)
	}
	if gotAuth != "Bearer the-access-token" {
		t.Errorf("Authorization = %q, want a bearer token", gotAuth)
	}
	if gotQuery.Get("before") != "1754300000" || gotQuery.Get("after") != "1754200000" {
		t.Errorf("window parameters = %v, want the caller's before/after", gotQuery)
	}
	if gotQuery.Get("per_page") != "30" || gotQuery.Get("page") != "1" {
		t.Errorf("paging parameters = %v", gotQuery)
	}
}

// TestStravaGetActivitiesFailures covers that a non-200 or a malformed body is an error
// rather than an empty activity list — an empty list would look like "nothing new to
// import" and hide a broken connection.
func TestStravaGetActivitiesFailures(t *testing.T) {
	stubStrava(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusTooManyRequests)
	})
	if _, err := StravaGetActivities("the-access-token", 1, 0); err == nil {
		t.Errorf("a rate-limited response returned no error")
	}

	stubStrava(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(`{"not":"an array"}`))
	})
	if _, err := StravaGetActivities("the-access-token", 1, 0); err == nil {
		t.Errorf("a malformed body returned no error")
	}
}

// TestStravaGetActivity covers the detailed-activity fetch, which is the only source of an
// activity's description (the list payload has none).
func TestStravaGetActivity(t *testing.T) {
	var gotPath string
	stubStrava(t, func(writer http.ResponseWriter, request *http.Request) {
		gotPath = request.URL.Path
		writer.Write([]byte(`{"id":111,"name":"Morning Walk","sport_type":"Walk","description":"felt good"}`))
	})

	activity, err := StravaGetActivity("the-access-token", "111")
	if err != nil {
		t.Fatalf("StravaGetActivity returned error: %v", err)
	}
	if activity.Description == nil || *activity.Description != "felt good" {
		t.Errorf("description = %v, want the detailed activity's", activity.Description)
	}
	if gotPath != "/api/v3/activities/111" {
		t.Errorf("requested %q, want the activity endpoint", gotPath)
	}
}

// TestStravaGetGear covers the lazy gear lookup used to name a gear id on first sight.
func TestStravaGetGear(t *testing.T) {
	var gotPath string
	stubStrava(t, func(writer http.ResponseWriter, request *http.Request) {
		gotPath = request.URL.Path
		writer.Write([]byte(`{"id":"g123","name":"Trail Shoes","nickname":"Muddy","retired":false,"distance":1234.5}`))
	})

	gear, err := StravaGetGear("the-access-token", "g123")
	if err != nil {
		t.Fatalf("StravaGetGear returned error: %v", err)
	}
	if gear.Name != "Trail Shoes" || gear.ID != "g123" {
		t.Errorf("gear = %+v, want the parsed gear", gear)
	}
	if gotPath != "/api/v3/gear/g123" {
		t.Errorf("requested %q, want the gear endpoint", gotPath)
	}

	t.Run("a missing gear is an error", func(t *testing.T) {
		stubStrava(t, func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusNotFound)
		})
		if _, err := StravaGetGear("the-access-token", "g404"); err == nil {
			t.Errorf("a 404 returned no error")
		}
	})
}
