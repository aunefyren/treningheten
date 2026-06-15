package database

import (
	"database/sql"
	"io"
	"testing"

	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newTestDB spins up an isolated in-memory SQLite database, runs the full schema
// migration, and points the package-global Instance at it for the duration of the
// test. A single underlying connection is used so the in-memory database survives
// across GORM calls and stays isolated per test. State is restored on cleanup.
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

	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	// One connection keeps the :memory: database alive and isolated for the test.
	sqlDB.SetMaxOpenConns(1)

	gormDB, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	prev := Instance
	Instance = gormDB
	Migrate()

	t.Cleanup(func() {
		Instance = prev
		_ = sqlDB.Close()
	})
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
