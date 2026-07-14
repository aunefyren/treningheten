package database

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"

	"github.com/sirupsen/logrus"
)

// withSQLiteConfig points files.ConfigFile.DBLocation at a fresh temp path and the package
// Instance at a throwaway value, restoring both on cleanup. Connect and InitializeSQLiteDB
// read the config global and (re)assign Instance, so they must be isolated from other tests.
// It also ensures logger.Log is set, since Connect logs and this test path does not go
// through newTestDB (which normally stubs the logger).
func withSQLiteConfig(t *testing.T) string {
	t.Helper()
	if logger.Log == nil {
		l := logrus.New()
		l.SetOutput(io.Discard)
		logger.Log = l
	}

	prevInstance := Instance
	prevLocation := files.ConfigFile.DBLocation

	location := filepath.Join(t.TempDir(), "test.db")
	files.ConfigFile.DBLocation = location

	t.Cleanup(func() {
		Instance = prevInstance
		files.ConfigFile.DBLocation = prevLocation
	})
	return location
}

func TestConnectUnrecognizedType(t *testing.T) {
	if err := Connect("mongodb", "Local", "", "", "", 0, "", false, ""); err == nil {
		t.Errorf("expected error for an unrecognized database type")
	}
}

func TestConnectSQLite(t *testing.T) {
	withSQLiteConfig(t)

	// The DB file does not exist yet, so Connect initializes it, then opens and migrates.
	if err := Connect("sqlite", "Local", "", "", "", 0, "", false, ""); err != nil {
		t.Fatalf("Connect(sqlite) returned error: %v", err)
	}
	if Instance == nil {
		t.Fatal("Connect(sqlite) left Instance nil")
	}

	// The connection is usable: migrate and round-trip a row.
	Migrate()
	user := makeTestUser(t, "connect@example.com", nil)
	if _, err := GetAllUserInformation(user.ID); err != nil {
		t.Errorf("round-trip after Connect failed: %v", err)
	}

	// Connecting again with the file now present exercises the already-exists path.
	if err := Connect("sqlite", "Local", "", "", "", 0, "", false, ""); err != nil {
		t.Errorf("Connect(sqlite) on an existing file returned error: %v", err)
	}
}

func TestInitializeSQLiteDB(t *testing.T) {
	location := withSQLiteConfig(t)

	if err := InitializeSQLiteDB(); err != nil {
		t.Fatalf("InitializeSQLiteDB returned error: %v", err)
	}
	if _, err := filepath.Abs(location); err != nil {
		t.Fatalf("bad location: %v", err)
	}

	// Pointing at an unwritable path (a directory that does not exist) surfaces an error.
	files.ConfigFile.DBLocation = filepath.Join(location, "nonexistent-dir", "db.sqlite")
	if err := InitializeSQLiteDB(); err == nil {
		t.Errorf("expected error initializing SQLite at an invalid path")
	}
}
