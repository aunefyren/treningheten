package controllers

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

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
//
// Every task is tracked in backgroundWork until it returns, so waitForBackgroundWork can tell
// when the background has gone quiet.
func goSafely(name string, fn func()) {
	backgroundWork.start(name)
	go func() {
		defer backgroundWork.finish(name)
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Log.Error("Background task '" + name + "' panicked: " + fmt.Sprint(recovered) + "\n" + string(debug.Stack()))
			}
		}()
		fn()
	}()
}

// backgroundTracker counts the goSafely tasks in flight, by name. idle is closed whenever the
// count drops to zero and replaced when work starts again, so a waiter just waits for it.
type backgroundTracker struct {
	mutex   sync.Mutex
	running map[string]int
	total   int
	idle    chan struct{}
}

var backgroundWork = newBackgroundTracker()

func newBackgroundTracker() *backgroundTracker {
	idle := make(chan struct{})
	close(idle)
	return &backgroundTracker{running: map[string]int{}, idle: idle}
}

func (b *backgroundTracker) start(name string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.total == 0 {
		b.idle = make(chan struct{})
	}
	b.total++
	b.running[name]++
}

func (b *backgroundTracker) finish(name string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.total--
	if b.running[name]--; b.running[name] == 0 {
		delete(b.running, name)
	}
	if b.total == 0 {
		close(b.idle)
	}
}

// wait blocks until no task is in flight. Work a task starts before it returns is waited
// for too: the child is counted before the parent finishes. On timeout it names what is
// still running.
func (b *backgroundTracker) wait(timeout time.Duration) error {
	b.mutex.Lock()
	idle := b.idle
	b.mutex.Unlock()

	select {
	case <-idle:
		return nil
	case <-time.After(timeout):
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()
	names := []string{}
	for name, count := range b.running {
		names = append(names, fmt.Sprintf("%s ×%d", name, count))
	}
	sort.Strings(names)
	return errors.New("background work still running after " + timeout.String() + ": " + strings.Join(names, ", "))
}

// waitForBackgroundWork blocks until every goSafely task has returned, or reports which are
// still running after timeout. The tests use it so no task outlives the database and config
// it was started against.
func waitForBackgroundWork(timeout time.Duration) error {
	return backgroundWork.wait(timeout)
}
