package controllers

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"
)

// TestSanitizedAccountInput sends messy account input through the real endpoints: the
// e-mail is stored trimmed and lowercased and works in any case at login and reset, names
// are trimmed but stored unescaped, and the password is kept exactly as typed.
func TestSanitizedAccountInput(t *testing.T) {
	h := newAPIHarness(t)
	server := withControllerSMTP(t)
	files.ConfigFile.TreninghetenExternalURL = "https://trening.test"
	_, adminToken := h.user("admin@sanitize.test", true)
	code := field(t, h.expect(http.StatusCreated, "POST", "/api/admin/invites", adminToken, nil), "invitation").(string)

	password := "  Pass word123 "
	h.expect(http.StatusCreated, "POST", "/api/open/users", "", models.UserCreationRequest{
		FirstName: "  O'Brien \t", LastName: " <b>Bold</b> ", Email: "  Ola@Example.COM \t",
		Password: password, PasswordRepeat: password, InviteCode: " " + code + "\n",
	})
	verificationCode := lastMailMatching(t, server, `this code: <b>([^<]+)</b>`)
	if greeting := lastMailMatching(t, server, `Hello <b>([^<]*)</b>`); greeting != "O&#39;Brien" {
		t.Errorf("mail greets %q, want the name HTML-escaped by the mail itself", greeting)
	}

	var stored models.User
	if err := database.Instance.Where("email = ?", "ola@example.com").First(&stored).Error; err != nil {
		t.Fatalf("registered user not stored under the normalised e-mail: %v", err)
	}
	if stored.FirstName != "O'Brien" || stored.LastName != "<b>Bold</b>" {
		t.Errorf("names stored as %q %q, want them trimmed and unescaped", stored.FirstName, stored.LastName)
	}

	for _, username := range []string{"ola@example.com", "  OLA@Example.com "} {
		if status, body := passwordLogin(h, username, password); status != http.StatusOK {
			t.Errorf("login as %q: status = %d, body %v", username, status, body)
		}
	}
	if status, _ := passwordLogin(h, "ola@example.com", strings.TrimSpace(password)); status == http.StatusOK {
		t.Error("login succeeded with the password trimmed; it must be kept exactly as typed")
	}

	sent := len(server.Messages())
	h.ok("POST", "/api/open/users/reset", "", map[string]string{"email": " OLA@Example.com\t"})
	if len(server.Messages()) != sent+1 {
		t.Error("a reset requested with a differently cased, padded e-mail sent no mail")
	}

	_, login := passwordLogin(h, "ola@example.com", password)
	token := login["access_token"].(string)
	h.ok("POST", "/api/open/users/verify/"+verificationCode, token, nil)
	h.ok("POST", "/api/auth/users/"+stored.ID.String(), token, models.UserUpdateRequest{
		Email: " New.Address@Example.COM ", OldPassword: password,
	})
	database.Instance.Where("id = ?", stored.ID).First(&stored)
	if stored.Email != "new.address@example.com" {
		t.Errorf("changed e-mail stored as %q, want it trimmed and lowercased", stored.Email)
	}
}

// TestSanitizedLabelsAndNotes checks the stored form of labels (trimmed, inner whitespace
// collapsed, first letter capitalised) and free text (trimmed only, never HTML-escaped).
func TestSanitizedLabelsAndNotes(t *testing.T) {
	h := newAPIHarness(t)
	_, adminToken := h.user("admin@labels.test", true)
	_, token := h.user("member@labels.test", false)

	send := func(t *testing.T, method, path, token string, body any) map[string]any {
		t.Helper()
		response := h.do(method, path, token, body)
		if response.Code < 200 || response.Code > 299 {
			t.Fatalf("%s %s: status = %d, body %s", method, path, response.Code, response.Body.String())
		}
		return decodeBody(t, response.Body.Bytes())
	}
	check := func(t *testing.T, what, got, want string) {
		t.Helper()
		if got != want {
			t.Errorf("%s = %q, want %q", what, got, want)
		}
	}

	t.Run("gear", func(t *testing.T) {
		created := send(t, "POST", "/api/auth/gear", token, map[string]any{
			"name": "  easy   run shoes ", "type": " Bike ", "brand": "  Nike  ", "nickname": " old   faithful ",
		})
		gearID := idOf(t, created, "gear")
		var gear models.Gear
		database.Instance.Where("id = ?", gearID).First(&gear)
		check(t, "gear name", gear.Name, "Easy run shoes")
		check(t, "gear type", gear.Type, "bike")
		check(t, "gear brand", *gear.Brand, "Nike")
		check(t, "gear nickname", *gear.Nickname, "old faithful")

		send(t, "PUT", "/api/auth/gear/"+gearID, token, map[string]any{"name": " trail\tshoes ", "type": " OTHER "})
		database.Instance.Where("id = ?", gearID).First(&gear)
		check(t, "updated gear name", gear.Name, "Trail shoes")
		check(t, "updated gear type", gear.Type, "other")
	})

	t.Run("prize and season", func(t *testing.T) {
		send(t, "POST", "/api/admin/prizes", adminToken, map[string]any{"name": "  gift   card ", "quantity": 1})
		var prize models.Prize
		database.Instance.Where("name = ?", "Gift card").First(&prize)
		check(t, "prize name", prize.Name, "Gift card")

		start := nextMonday(2)
		send(t, "POST", "/api/admin/seasons", adminToken, models.SeasonCreationRequest{
			Name: " spring   league ", Description: "  first  half\n", Start: start, End: start.AddDate(0, 0, 27), Prize: prize.ID,
		})
		var season models.Season
		database.Instance.Where("name = ?", "Spring league").First(&season)
		check(t, "season name", season.Name, "Spring league")
		check(t, "season description", season.Description, "first  half")
	})

	t.Run("personal access token", func(t *testing.T) {
		created := send(t, "POST", "/api/auth/pats", token, map[string]any{"name": "  ci   bot ", "scope": " api:read ", "expires_in_days": 30})
		check(t, "PAT name", field(t, created, "data", "pat", "name").(string), "Ci bot")
	})

	t.Run("news", func(t *testing.T) {
		send(t, "POST", "/api/admin/news", adminToken, map[string]any{"title": "  Big news  ", "body": " Tom & Jerry's <run>\n", "date": time.Now()})
		var news models.News
		database.Instance.Where("title = ?", "Big news").First(&news)
		check(t, "news body", news.Body, "Tom & Jerry's <run>")
	})

	t.Run("action", func(t *testing.T) {
		send(t, "POST", "/api/auth/actions", adminToken, map[string]any{
			"name": "  cable   fly ", "norwegian_name": " kabel  flyes ", "type": " lifting ", "body_part": " chest ",
		})
		var action models.Action
		database.Instance.Where("name = ?", "Cable Fly").First(&action)
		// Curated exercise names keep their full title case.
		check(t, "action name", action.Name, "Cable Fly")
		check(t, "action Norwegian name", action.NorwegianName, "Kabel Flyes")
		check(t, "action type", action.Type, "lifting")
	})

	t.Run("workout notes", func(t *testing.T) {
		dayID := idOf(t, h.ok("GET", "/api/auth/exercise-days/week?today=true", token, nil), "exercise")

		created := send(t, "POST", "/api/auth/exercises", token, map[string]any{"exercise_day_id": dayID, "is_on": true, "note": "  felt   good \n"})
		exerciseID := idOf(t, created, "exercise")
		var exercise models.Exercise
		database.Instance.Where("id = ?", exerciseID).First(&exercise)
		check(t, "new session note", exercise.Note, "felt   good")

		send(t, "PUT", "/api/auth/exercises/"+exerciseID, token, map[string]any{"is_on": true, "note": " <script>x</script> "})
		database.Instance.Where("id = ?", exerciseID).First(&exercise)
		check(t, "updated session note", exercise.Note, "<script>x</script>")

		var day models.ExerciseDay
		send(t, "POST", "/api/auth/exercise-days/"+dayID, token, map[string]any{"note": "  O'Brien's  day "})
		database.Instance.Where("id = ?", dayID).First(&day)
		check(t, "day note", day.Note, "O'Brien's  day")

		// The week form saves notes through a different path, which used to HTML-escape them.
		days := currentWeekDays(0)
		todayIndex := (int(time.Now().Weekday()) + 6) % 7
		days[todayIndex].ExerciseInterval = 1
		days[todayIndex].Note = " Tom & Jerry's \t"
		send(t, "POST", "/api/auth/exercises/week", token, models.WeekCreationRequest{Days: days, TimeZone: " UTC "})
		database.Instance.Where("id = ?", dayID).First(&day)
		check(t, "week-form day note", day.Note, "Tom & Jerry's")
	})
}
