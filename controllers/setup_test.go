package controllers

import (
	"database/sql"
	"io"
	"os"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

// TestMain stubs logger.Log with a discarding logger so controller helpers that log
// (including Migrate() and the data-access layer) don't nil-panic in tests. The real
// InitLogger writes to a config file we don't want to touch from tests.
func TestMain(m *testing.M) {
	if logger.Log == nil {
		l := logrus.New()
		l.SetOutput(io.Discard)
		logger.Log = l
	}
	os.Exit(m.Run())
}

// newControllerTestDB points database.Instance at an isolated in-memory SQLite database
// with the full schema migrated, restoring the previous instance on cleanup. This
// mirrors database.newTestDB, which is unexported and lives in the database package.
// Foreign-key enforcement is left off so tests can seed rows by ID without building the
// full parent graph (season → prize → goal → user); the logic under test queries by ID.
func newControllerTestDB(t *testing.T) {
	t.Helper()

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

	prev := database.Instance
	database.Instance = gormDB
	database.Migrate()

	t.Cleanup(func() {
		database.Instance = prev
		_ = sqlDB.Close()
	})
}

// seedExerciseDayWithExercises inserts one exercise day for a user on the given date,
// plus `count` valid (enabled, on) exercises under it. Week completion counts exercise
// rows, so `count` is what the week's exercises total. The date should sit inside the
// target week (e.g. a Wednesday) to stay clear of the Monday/Sunday range boundaries.
func seedExerciseDayWithExercises(t *testing.T, userID uuid.UUID, date time.Time, count int) {
	t.Helper()

	day := models.ExerciseDay{
		Date:    date,
		Enabled: true,
		UserID:  &userID,
	}
	day.ID = uuid.New()
	if err := database.Instance.Omit("User", "Goal").Create(&day).Error; err != nil {
		t.Fatalf("failed to seed exercise day: %v", err)
	}

	for i := 0; i < count; i++ {
		exercise := models.Exercise{
			Enabled:       true,
			IsOn:          true,
			ExerciseDayID: day.ID,
		}
		exercise.ID = uuid.New()
		if err := database.Instance.Omit("ExerciseDay").Create(&exercise).Error; err != nil {
			t.Fatalf("failed to seed exercise: %v", err)
		}
	}
}

// createTestUser inserts a minimal enabled, verified user via the real registration path
// and returns it.
func createTestUser(t *testing.T, email string, firstName string) models.User {
	t.Helper()

	admin := false
	user := models.User{
		FirstName: firstName,
		LastName:  "User",
		Email:     email,
		Password:  "hashed-password",
		Admin:     &admin,
		Enabled:   true,
		Verified:  true,
	}
	user.ID = uuid.New()

	created, err := database.RegisterUserInDB(user)
	if err != nil {
		t.Fatalf("failed to register test user: %v", err)
	}
	return created
}

// seedSickleave inserts an enabled sick-leave row for a goal on the given date.
func seedSickleave(t *testing.T, goalID uuid.UUID, date time.Time, used bool) {
	t.Helper()

	sickleave := models.Sickleave{
		Enabled: true,
		GoalID:  goalID,
		Used:    used,
		Date:    date,
	}
	sickleave.ID = uuid.New()
	if err := database.Instance.Omit("Goal").Create(&sickleave).Error; err != nil {
		t.Fatalf("failed to seed sickleave: %v", err)
	}
}
