package utilities

import (
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/internal/smtptest"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// withFakeSMTP points the SMTP config at a fresh in-process server for the test.
func withFakeSMTP(t *testing.T) *smtptest.Server {
	t.Helper()

	server := smtptest.Start(t)
	previous := files.ConfigFile
	files.ConfigFile.SMTPHost = server.Host
	files.ConfigFile.SMTPPort = server.Port
	files.ConfigFile.SMTPUsername = ""
	files.ConfigFile.SMTPPassword = ""
	files.ConfigFile.SMTPFrom = "noreply@test.local"
	files.ConfigFile.TreninghetenName = "Treningheten"
	files.ConfigFile.TreninghetenExternalURL = "https://trening.test"
	files.ConfigFile.TreninghetenEnvironment = "production"
	t.Cleanup(func() { files.ConfigFile = previous })
	return server
}

// onlyMessage asserts exactly one mail arrived, to the given address, and returns its data
// with quoted-printable soft line breaks removed so the body can be searched.
func onlyMessage(t *testing.T, server *smtptest.Server, to string) string {
	t.Helper()

	messages := server.Messages()
	if len(messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(messages))
	}
	if len(messages[0].To) != 1 || messages[0].To[0] != to {
		t.Errorf("recipients = %v, want [%s]", messages[0].To, to)
	}
	data := strings.ReplaceAll(messages[0].Data, "=\r\n", "")
	return strings.ReplaceAll(data, "=3D", "=")
}

func TestUserMailsReachTheSMTPServer(t *testing.T) {
	code := "VERIFY123"
	user := models.User{FirstName: "Ann", Email: "ann@test.local", VerificationCode: &code, ResetCode: stringPointer("RESET456")}
	season := models.Season{Name: "Autumn"}

	tests := []struct {
		name    string
		send    func() error
		subject string
		body    string
	}{
		{"verification", func() error { return SendSMTPVerificationEmail(user) }, "Please verify your account", "VERIFY123"},
		{"reset", func() error { return SendSMTPResetEmail(user) }, "", "RESET456"},
		{"sunday reminder", func() error { return SendSMTPSundayReminderEmail(user, season, time.Now()) }, "", "Ann"},
		{"week lost", func() error { return SendSMTPForWeekLost(user, 12) }, "", "12"},
		{"wheel spin", func() error { return SendSMTPForWheelSpin(user, 13) }, "", "13"},
		{"wheel spin check", func() error { return SendSMTPForWheelSpinCheck(user, 14) }, "", "14"},
		{"wheel spin win", func() error { return SendSMTPForWheelSpinWin(user, 15) }, "", "15"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := withFakeSMTP(t)
			if err := test.send(); err != nil {
				t.Fatalf("send failed: %v", err)
			}
			data := onlyMessage(t, server, "ann@test.local")
			if test.subject != "" && !strings.Contains(data, "Subject: "+test.subject) {
				t.Errorf("subject %q missing from:\n%s", test.subject, data)
			}
			if !strings.Contains(data, test.body) {
				t.Errorf("body is missing %q:\n%s", test.body, data)
			}
		})
	}
}

func TestTestEnvironmentRedirectsMailToTheTestAddress(t *testing.T) {
	server := withFakeSMTP(t)
	files.ConfigFile.TreninghetenEnvironment = "test"
	files.ConfigFile.TreninghetenTestEmail = "sink@test.local"

	code := "C"
	if err := SendSMTPVerificationEmail(models.User{FirstName: "Real", Email: "real@user.local", VerificationCode: &code}); err != nil {
		t.Fatalf("send failed: %v", err)
	}
	onlyMessage(t, server, "sink@test.local")
}

func TestMailFailsWhenTheServerIsUnreachable(t *testing.T) {
	withFakeSMTP(t)
	files.ConfigFile.SMTPHost = "127.0.0.1"
	files.ConfigFile.SMTPPort = 1 // nothing listens there

	code := "C"
	if err := SendSMTPVerificationEmail(models.User{Email: "a@test.local", VerificationCode: &code}); err == nil {
		t.Error("expected an error when the SMTP server can't be reached")
	}
}

func TestSeasonStartMailGoesToEveryGoalHolder(t *testing.T) {
	newUtilitiesTestDB(t)
	server := withFakeSMTP(t)

	var goals []models.GoalObject
	for _, email := range []string{"one@test.local", "two@test.local"} {
		user := models.User{FirstName: "Runner", Email: email, Enabled: true}
		user.ID = uuid.New()
		if err := database.Instance.Create(&user).Error; err != nil {
			t.Fatalf("failed to seed user: %v", err)
		}
		goals = append(goals, models.GoalObject{User: models.PublicUser{ID: user.ID, FirstName: "Runner"}, ExerciseInterval: 3})
	}
	// A goal whose user doesn't exist is skipped, not fatal.
	goals = append(goals, models.GoalObject{User: models.PublicUser{ID: uuid.New()}})

	if err := SendSMTPSeasonStartEmail(models.SeasonObject{Goals: goals}); err != nil {
		t.Fatalf("SendSMTPSeasonStartEmail: %v", err)
	}

	messages := server.Messages()
	if len(messages) != 2 {
		t.Fatalf("messages = %d, want one per existing goal holder (2)", len(messages))
	}
	recipients := messages[0].To[0] + "," + messages[1].To[0]
	if !strings.Contains(recipients, "one@test.local") || !strings.Contains(recipients, "two@test.local") {
		t.Errorf("recipients = %s", recipients)
	}
}

func stringPointer(value string) *string { return &value }
