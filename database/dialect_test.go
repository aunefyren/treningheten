package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/internal/testdb"
)

func TestInstantParam(t *testing.T) {
	newTestDB(t)

	oslo, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		t.Fatal(err)
	}
	instant := time.Date(2024, 6, 15, 14, 0, 0, 0, oslo)

	got := instantParam(instant)
	if testdb.Backend() == "sqlite" {
		if got != "2024-06-15 12:00:00" {
			t.Errorf("instantParam on sqlite = %v, want the UTC text form", got)
		}
		return
	}
	if parameter, ok := got.(time.Time); !ok || !parameter.Equal(instant) {
		t.Errorf("instantParam = %v, want the time itself", got)
	}
}
