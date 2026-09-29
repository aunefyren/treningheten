package utilities

import (
	"database/sql"
	"io"
	"os"
	"testing"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

// TestMain stubs logger.Log with a discarding logger; the real one writes to a file
// under config/.
func TestMain(m *testing.M) {
	if logger.Log == nil {
		l := logrus.New()
		l.SetOutput(io.Discard)
		logger.Log = l
	}
	os.Exit(m.Run())
}

// newUtilitiesTestDB points database.Instance at an isolated, migrated in-memory SQLite
// database for the test (the same approach as database.newTestDB, which is unexported).
func newUtilitiesTestDB(t *testing.T) {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	gormDB, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	previous := database.Instance
	database.Instance = gormDB
	database.Migrate()

	t.Cleanup(func() {
		database.Instance = previous
		_ = sqlDB.Close()
	})
}
