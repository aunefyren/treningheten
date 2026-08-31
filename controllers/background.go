package controllers

import (
	"fmt"
	"runtime/debug"

	"github.com/aunefyren/treningheten/logger"
)

// goSafely runs fn on its own goroutine behind a panic guard.
//
// The handlers fire a lot of work off into the background — achievement grants, Strava/Hevy
// syncs, Ollama cache refreshes, media pulls — whose outcome the response deliberately ignores.
// A panic in any of them is unrecoverable from the goroutine that spawned it, so a single nil
// dereference in a best-effort achievement grant takes down the whole server, dropping every
// in-flight request with it. That is never the right trade for work whose failure the caller
// already treats as ignorable: log it with its stack and keep serving.
//
// name identifies the task in the log — make it the thing a reader would grep for.
//
// It is not a licence to ignore errors: fn should still log its own, and anything whose failure
// the user must hear about does not belong on a background goroutine in the first place.
func goSafely(name string, fn func()) {
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Log.Error("Background task '" + name + "' panicked: " + fmt.Sprint(recovered) + "\n" + string(debug.Stack()))
			}
		}()
		fn()
	}()
}
