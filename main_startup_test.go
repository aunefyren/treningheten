package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codnect.io/chrono"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"
)

// inTempInstall runs the test from an empty directory, the way a fresh install starts,
// and restores the process-wide config, logger and database afterwards.
func inTempInstall(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	previousConfig, previousLogger, previousDB := files.ConfigFile, logger.Log, database.Instance
	t.Cleanup(func() {
		if database.Instance != nil && database.Instance != previousDB {
			if sqlDB, err := database.Instance.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}
		files.ConfigFile, logger.Log, database.Instance = previousConfig, previousLogger, previousDB
	})
	return dir
}

func TestStartupOnAFreshInstall(t *testing.T) {
	dir := inTempInstall(t)
	withArgs(t, "--generateinvite", "true", "--disablesmtp", "true", "--timezone", "Europe/Oslo")

	if err := startup(); err != nil {
		t.Fatalf("startup: %v", err)
	}

	for _, file := range []string{"config/config.json", "config/data.db", "config/treningheten.log"} {
		if _, err := os.Stat(filepath.Join(dir, file)); err != nil {
			t.Errorf("%s was not created: %v", file, err)
		}
	}
	var invites int64
	database.Instance.Model(&models.Invite{}).Count(&invites)
	if invites != 1 {
		t.Errorf("invites = %d, want the one generated at startup", invites)
	}
	var achievements, actions int64
	database.Instance.Model(&models.Achievement{}).Count(&achievements)
	database.Instance.Model(&models.Action{}).Count(&actions)
	if achievements == 0 || actions == 0 {
		t.Errorf("seeds missing: %d achievements, %d actions", achievements, actions)
	}
	if client, err := database.GetOAuthClientByClientID(models.FirstPartyClientID); err != nil || !client.FirstParty {
		t.Errorf("first-party OAuth client not seeded: %v", err)
	}
	log, _ := os.ReadFile(filepath.Join(dir, "config/treningheten.log"))
	if !strings.Contains(string(log), "generated new invite code") {
		t.Error("the invite code was not logged")
	}
}

func TestStartupDropsAnInvalidTimeZone(t *testing.T) {
	inTempInstall(t)
	withArgs(t, "--timezone", "Not/AZone", "--disablesmtp", "true")

	if err := startup(); err != nil {
		t.Fatalf("startup: %v", err)
	}
	if files.ConfigFile.Timezone != "" {
		t.Errorf("timezone = %q, want the invalid value cleared", files.ConfigFile.Timezone)
	}
}

func TestStartupFailsOnAnUnreachableDatabase(t *testing.T) {
	inTempInstall(t)
	withArgs(t, "--disablesmtp", "true", "--dbip", "127.0.0.1", "--dbport", "1")
	if err := os.MkdirAll("config", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("config/config.json", []byte(`{"db_type":"mysql"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := startup(); err == nil || !strings.Contains(err.Error(), "database") {
		t.Errorf("startup against a closed MySQL port: err = %v, want a database error", err)
	}
}

func TestScheduleJobsRespectsIntegrationFlags(t *testing.T) {
	inTempInstall(t)
	scheduler := chrono.NewDefaultTaskScheduler()
	t.Cleanup(func() { <-scheduler.Shutdown() })

	files.ConfigFile.Ollama.Enabled = false
	files.ConfigFile.StravaEnabled = false
	files.ConfigFile.HevyEnabled = false
	files.ConfigFile.Media.Enabled = false
	if got := scheduleJobs(scheduler); got != 2 {
		t.Errorf("jobs with every integration off = %d, want the two core jobs", got)
	}

	files.ConfigFile.Ollama.Enabled = true
	files.ConfigFile.StravaEnabled = true
	files.ConfigFile.HevyEnabled = true
	files.ConfigFile.Media.Enabled = true
	if got := scheduleJobs(scheduler); got != 6 {
		t.Errorf("jobs with every integration on = %d, want 6", got)
	}
}
