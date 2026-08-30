package middlewares

import (
	"testing"
	"time"
)

func TestSlidingWindowAllowsUpToLimitThenBlocks(t *testing.T) {
	limiter := &slidingWindow{events: map[string][]time.Time{}, limit: 3, window: time.Minute}

	for attempt := 1; attempt <= 3; attempt++ {
		if allowed, _ := limiter.allow("1.2.3.4"); !allowed {
			t.Fatalf("attempt %d should have been allowed", attempt)
		}
	}

	allowed, retryAfter := limiter.allow("1.2.3.4")
	if allowed {
		t.Fatal("the fourth attempt should have been blocked")
	}
	if retryAfter <= 0 || retryAfter > time.Minute {
		t.Fatalf("retryAfter should be inside the window, got %v", retryAfter)
	}
}

func TestSlidingWindowIsPerKey(t *testing.T) {
	limiter := &slidingWindow{events: map[string][]time.Time{}, limit: 1, window: time.Minute}

	limiter.allow("1.2.3.4")
	if allowed, _ := limiter.allow("5.6.7.8"); !allowed {
		t.Fatal("a different key must have its own budget")
	}
}

func TestSlidingWindowForgetsExpiredEvents(t *testing.T) {
	limiter := &slidingWindow{events: map[string][]time.Time{}, limit: 2, window: 50 * time.Millisecond}

	limiter.allow("1.2.3.4")
	limiter.allow("1.2.3.4")
	if allowed, _ := limiter.allow("1.2.3.4"); allowed {
		t.Fatal("the limit should be reached")
	}

	time.Sleep(60 * time.Millisecond)

	if allowed, _ := limiter.allow("1.2.3.4"); !allowed {
		t.Fatal("events older than the window should have expired")
	}
	// The key is pruned rather than accumulating forever.
	if got := len(limiter.events["1.2.3.4"]); got != 1 {
		t.Fatalf("expected 1 retained event, got %d", got)
	}
}

func TestLoginFailureLockoutAndReset(t *testing.T) {
	t.Cleanup(func() { ClearLoginFailures("victim@example.com") })

	for attempt := 0; attempt < loginFailureLimit-1; attempt++ {
		RegisterLoginFailure("victim@example.com")
	}
	if locked, _ := LoginAttemptLocked("victim@example.com"); locked {
		t.Fatal("an account should not lock before the failure limit")
	}

	RegisterLoginFailure("victim@example.com")
	locked, retryAfter := LoginAttemptLocked("victim@example.com")
	if !locked {
		t.Fatal("the account should be locked at the failure limit")
	}
	if retryAfter != loginFailureWindow {
		t.Fatalf("expected retryAfter %v, got %v", loginFailureWindow, retryAfter)
	}

	// Another account is unaffected.
	if locked, _ := LoginAttemptLocked("someone-else@example.com"); locked {
		t.Fatal("the lockout must be scoped to one account")
	}

	// A successful login clears the count.
	ClearLoginFailures("victim@example.com")
	if locked, _ := LoginAttemptLocked("victim@example.com"); locked {
		t.Fatal("a successful login should clear the failure count")
	}
}

func TestPasswordCostSlotsAreBoundedAndReleased(t *testing.T) {
	capacity := cap(passwordCostSlots)
	if capacity < 1 {
		t.Fatalf("expected at least one password cost slot, got %d", capacity)
	}

	releases := []func(){}
	for taken := 0; taken < capacity; taken++ {
		release, ok := AcquirePasswordCostSlot()
		if !ok {
			t.Fatalf("slot %d should have been available", taken)
		}
		releases = append(releases, release)
	}

	// Every slot is held, so nothing further may enter bcrypt.
	select {
	case passwordCostSlots <- struct{}{}:
		<-passwordCostSlots
		t.Fatal("the budget should be exhausted")
	default:
	}

	for _, release := range releases {
		release()
	}

	release, ok := AcquirePasswordCostSlot()
	if !ok {
		t.Fatal("a released slot should be reusable")
	}
	release()
}
