package controllers

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestBackgroundTrackerWait(t *testing.T) {
	t.Run("idle returns at once", func(t *testing.T) {
		tracker := newBackgroundTracker()
		if err := tracker.wait(time.Millisecond); err != nil {
			t.Errorf("idle wait = %v", err)
		}
	})

	t.Run("waits for nested work", func(t *testing.T) {
		tracker := newBackgroundTracker()
		var finished atomic.Int32
		// The parent counts its child before it finishes itself, as goSafely does when a
		// background task starts another.
		tracker.start("parent")
		go func() {
			defer tracker.finish("parent")
			tracker.start("child")
			go func() {
				defer tracker.finish("child")
				time.Sleep(20 * time.Millisecond)
				finished.Add(1)
			}()
		}()
		if err := tracker.wait(5 * time.Second); err != nil {
			t.Fatalf("wait = %v", err)
		}
		if finished.Load() != 1 {
			t.Errorf("wait returned before the child finished")
		}
	})

	t.Run("names what is stuck", func(t *testing.T) {
		tracker := newBackgroundTracker()
		release := make(chan struct{})
		tracker.start("stuck sync")
		tracker.start("stuck sync")
		go func() {
			<-release
			tracker.finish("stuck sync")
			tracker.finish("stuck sync")
		}()
		err := tracker.wait(10 * time.Millisecond)
		if err == nil || !strings.Contains(err.Error(), "stuck sync ×2") {
			t.Errorf("timeout error = %v, want the stuck task named", err)
		}
		close(release)
		if err := tracker.wait(5 * time.Second); err != nil {
			t.Errorf("wait after release = %v", err)
		}
	})

	t.Run("busy again after idle", func(t *testing.T) {
		tracker := newBackgroundTracker()
		tracker.start("first")
		tracker.finish("first")
		tracker.start("second")
		if err := tracker.wait(10 * time.Millisecond); err == nil {
			t.Errorf("wait returned while the second task runs")
		}
		tracker.finish("second")
		if err := tracker.wait(time.Second); err != nil {
			t.Errorf("wait after the second task = %v", err)
		}
	})
}

// A task that panics is still counted as finished, so a crash in background work can't
// leave every later wait hanging.
func TestGoSafelyReleasesAPanickedTask(t *testing.T) {
	quiesceBackgroundWork(t)
	goSafely("doomed", func() { panic("boom") })
	if err := waitForBackgroundWork(5 * time.Second); err != nil {
		t.Errorf("wait after a panic = %v", err)
	}
}
