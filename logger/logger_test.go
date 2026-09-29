package logger

import (
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func withLogDir(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.Mkdir("config", 0o700); err != nil {
		t.Fatal(err)
	}
	previous := Log
	t.Cleanup(func() { Log = previous })
}

func TestInitLoggerWritesJSONToTheLogFile(t *testing.T) {
	withLogDir(t)

	InitLogger("debug")
	if Log.GetLevel() != logrus.DebugLevel {
		t.Errorf("level = %v, want debug", Log.GetLevel())
	}
	Log.Debug("hello from the test")

	written, err := os.ReadFile("config/treningheten.log")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(written), `"msg":"hello from the test"`) {
		t.Errorf("log file does not contain the JSON entry:\n%s", written)
	}
}

func TestInitLoggerFallsBackToInfo(t *testing.T) {
	withLogDir(t)

	InitLogger("chatty")
	if Log.GetLevel() != logrus.InfoLevel {
		t.Errorf("level = %v, want the info fallback", Log.GetLevel())
	}
}
