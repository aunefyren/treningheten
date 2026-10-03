package database

import (
	"io"
	"testing"

	"github.com/aunefyren/treningheten/internal/testdb"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// newTestDB spins up an isolated database (in-memory SQLite, or Postgres/MySQL when the
// testdb environment variables are set), runs the full schema migration, and points the
// package-global Instance at it for the duration of the test. State is restored on
// cleanup.
//
// logger.Log is stubbed to a discarding logger because Migrate() (and most database
// helpers) log, and the real InitLogger writes to a config/ file we don't want in tests.
func newTestDB(t *testing.T) {
	t.Helper()

	if logger.Log == nil {
		l := logrus.New()
		l.SetOutput(io.Discard)
		logger.Log = l
	}

	gormDB := testdb.Open(t, nil)

	prev := Instance
	Instance = gormDB
	Migrate()

	t.Cleanup(func() { Instance = prev })
}

// boolPtr / strPtr are small helpers for the many *bool / *string model fields.
func boolPtr(b bool) *bool    { return &b }
func strPtr(s string) *string { return &s }

// makeTestUser inserts a minimal enabled user and returns it. Fields can be tweaked
// via the mutate callback before the row is written.
func makeTestUser(t *testing.T, email string, mutate func(*models.User)) models.User {
	t.Helper()

	user := models.User{
		FirstName: "Test",
		LastName:  "User",
		Email:     email,
		Password:  "hashed-password",
		Admin:     boolPtr(false),
		Enabled:   true,
		Verified:  true,
	}
	user.ID = uuid.New()

	if mutate != nil {
		mutate(&user)
	}

	created, err := RegisterUserInDB(user)
	if err != nil {
		t.Fatalf("failed to register test user: %v", err)
	}
	return created
}
