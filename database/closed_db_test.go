//go:generate sh -c "go run gen_closed_db_calls.go . > closed_db_calls_test.go"

package database

import "testing"

// shortCircuitOnEmptyInput return early, before touching the database, when called with
// the zero-value arguments the sweep uses (an empty id list, a zero heart-rate peak, an
// empty playback list) — correctly, since there is nothing to read or write.
var shortCircuitOnEmptyInput = map[string]bool{
	"BumpObservedMaxHeartrate":                       true,
	"GetDebtsForUserIDsBetweenDates":                 true,
	"GetExerciseDaysForSharingUsersInListUsingDates": true,
	"GetSickleavesForGoalIDsBetweenDates":            true,
	"GetUnusedSickleavesForGoalIDs":                  true,
	"GetUsersByIDs":                                  true,
	"GetValidExercisesForUserIDsBetweenDates":        true,
	"ReplaceMediaPlaybackForExerciseProvider":        true,
}

// With the database gone, every exported data-access function must report the failure
// — never panic, and never return a nil error as if the read or write had worked. The
// call list is generated from the package's own exported functions
// (closed_db_calls_test.go), so a new function is swept once it's regenerated.
func TestEveryFunctionReportsADeadDatabase(t *testing.T) {
	newTestDB(t)
	sqlDB, err := Instance.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	for _, entry := range closedDBCalls {
		entry := entry
		t.Run(entry.name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Errorf("panicked with the database closed: %v", recovered)
				}
			}()
			if err := entry.call(); entry.returnsError && err == nil && !shortCircuitOnEmptyInput[entry.name] {
				t.Errorf("returned a nil error with the database closed")
			}
		})
	}
}
