package controllers

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/internal/testdb"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
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
	// Real bcrypt cost is ~1s per hash or compare; the flows and the fault sweep hash
	// hundreds of times. The cost value itself is not what these tests check.
	models.PasswordHashCost = bcrypt.MinCost
	// Tests wait for background work before restoring state; the seconds-long pause before
	// an achievement's push would make every one of those waits that long.
	achievementPushDelayUnit = 0
	os.Exit(m.Run())
}

// newControllerTestDB points database.Instance at an isolated database (see internal/testdb)
// with the full schema migrated, restoring the previous instance on cleanup. This
// mirrors database.newTestDB, which is unexported and lives in the database package.
// Foreign-key enforcement is left off so tests can seed rows by ID without building the
// full parent graph (season → prize → goal → user); the logic under test queries by ID.
func newControllerTestDB(t *testing.T) {
	t.Helper()

	// Silent: the fault sweeps fail thousands of queries on purpose, and GORM would print
	// every one of them, burying real failures in the CI log.
	gormDB := testdb.Open(t, &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})

	quiesceBackgroundWork(t)
	prev := database.Instance
	database.Instance = gormDB
	database.Migrate()

	restoreAfterBackgroundWork(t, func() { database.Instance = prev })
}

// backgroundWorkTimeout bounds how long a test waits for goSafely tasks to finish. They
// talk to in-process stubs, so anything near this is a hang, not slowness.
const backgroundWorkTimeout = 60 * time.Second

// quiesceBackgroundWork waits until no goSafely task is in flight. Call it before
// changing package state (database.Instance, files.ConfigFile, a stubbed URL): a task
// still running from an earlier request would otherwise read the state half-changed.
func quiesceBackgroundWork(t testing.TB) {
	t.Helper()
	if err := waitForBackgroundWork(backgroundWorkTimeout); err != nil {
		t.Errorf("%v", err)
	}
}

// restoreAfterBackgroundWork registers a cleanup that waits for in-flight goSafely tasks
// before restoring package state, so no task outlives the database, config or stub it was
// started against. Use it instead of t.Cleanup for any restore of package state: cleanups
// run last-registered first, so every restore has to wait, not just one of them. The wait
// is cheap once the background is quiet.
func restoreAfterBackgroundWork(t testing.TB, restore func()) {
	t.Helper()
	t.Cleanup(func() {
		quiesceBackgroundWork(t)
		restore()
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
