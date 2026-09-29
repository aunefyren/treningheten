package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/aunefyren/treningheten/database"

	"gorm.io/gorm"
)

var errInjectedFault = errors.New("injected database fault")

// faultInjector fails exactly the Nth database operation (query, create, update, delete,
// row or raw) issued after it is armed. It is installed as GORM callbacks that run
// before each operation, so the failing statement is never executed — like a lost
// connection at that point in the request.
type faultInjector struct {
	mutex  sync.Mutex
	armed  bool
	failAt int
	seen   int
}

func installFaultInjector(t *testing.T, db *gorm.DB) *faultInjector {
	t.Helper()

	injector := &faultInjector{}
	hook := func(tx *gorm.DB) {
		injector.mutex.Lock()
		defer injector.mutex.Unlock()
		if !injector.armed {
			return
		}
		injector.seen++
		if injector.seen == injector.failAt {
			_ = tx.AddError(errInjectedFault)
		}
	}

	callbacks := db.Callback()
	registrations := []error{
		callbacks.Create().Before("gorm:create").Register("fault:create", hook),
		callbacks.Query().Before("gorm:query").Register("fault:query", hook),
		callbacks.Update().Before("gorm:update").Register("fault:update", hook),
		callbacks.Delete().Before("gorm:delete").Register("fault:delete", hook),
		callbacks.Row().Before("gorm:row").Register("fault:row", hook),
		callbacks.Raw().Before("gorm:raw").Register("fault:raw", hook),
	}
	for _, err := range registrations {
		if err != nil {
			t.Fatalf("failed to register fault callback: %v", err)
		}
	}
	return injector
}

// arm starts counting operations; failAt 0 only counts.
func (f *faultInjector) arm(failAt int) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.armed, f.failAt, f.seen = true, failAt, 0
}

// disarm stops counting and returns how many operations were seen.
func (f *faultInjector) disarm() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.armed = false
	return f.seen
}

// faultRequest is the request a scenario wants swept.
type faultRequest struct {
	method, path, token string
	body                any
	// call, when set, is invoked instead of an HTTP request — for background entry
	// points (syncs, cron jobs). A nil error counts as 200, an error as 500.
	call func() error
}

// faultScenario builds its data on a fresh harness (faults off) and returns the request
// under test. wantStatus is the status of the fault-free run.
type faultScenario struct {
	name       string
	wantStatus int
	setup      func(t *testing.T, h *apiHarness) faultRequest
}

// sweepFaults runs a scenario once cleanly to count its database operations, then once
// per operation with exactly that one failing, each on a fresh database. Every faulted
// run must answer without panicking and must not report success — a handler that
// swallows a failed write and answers 2xx would be caught here.
func sweepFaults(t *testing.T, scenario faultScenario) {
	t.Helper()

	run := func(failAt int) (status int, operations int, body string) {
		var result *httptest.ResponseRecorder
		t.Run(fmt.Sprintf("fault-%d", failAt), func(t *testing.T) {
			h := newAPIHarness(t)
			injector := installFaultInjector(t, database.Instance)
			request := scenario.setup(t, h)

			injector.arm(failAt)
			defer func() {
				operations = injector.disarm()
				if recovered := recover(); recovered != nil {
					t.Errorf("%s: panicked with query %d failing: %v", scenario.name, failAt, recovered)
				}
			}()
			if request.call != nil {
				result = httptest.NewRecorder()
				if err := request.call(); err != nil {
					result.Code = http.StatusInternalServerError
					result.Body.WriteString(err.Error())
				} else {
					result.Code = http.StatusOK
				}
				return
			}
			result = h.do(request.method, request.path, request.token, request.body)
		})
		if result == nil {
			return 0, operations, ""
		}
		return result.Code, operations, result.Body.String()
	}

	status, total, body := run(0)
	if status != scenario.wantStatus {
		t.Fatalf("%s: clean run status = %d, want %d; body: %.300s", scenario.name, status, scenario.wantStatus, body)
	}

	for failAt := 1; failAt <= total; failAt++ {
		faulted, _, faultedBody := run(failAt)
		if faulted >= 200 && faulted < 300 {
			// Reported, not failed: many handlers deliberately degrade (log and return
			// partial data) when a secondary lookup fails. Which of these should be
			// errors instead is an open question — see docs/wip.md.
			swallowed.record(scenario.name, faulted, faultedBody)
		}
	}
}

// swallowedFaults collects scenarios that answered 2xx with a database operation failing.
type swallowedFaults struct {
	mutex  sync.Mutex
	counts map[string]int
}

var swallowed = &swallowedFaults{counts: map[string]int{}}

func (s *swallowedFaults) record(name string, status int, body string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.counts[name]++
}
