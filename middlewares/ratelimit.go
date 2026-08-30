package middlewares

import (
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Rate limiting for the unauthenticated surface. Everything here is in-memory and
// per-process: Treningheten is a single self-hosted binary, so there is no second
// instance to share state with, and a limiter that survives a restart is not worth a
// Redis dependency. The point is to make credential stuffing slow and to stop a
// handful of clients from burning every core on bcrypt, not to be an exact accountant.
//
// Limits are deliberately code constants rather than config: they are generous enough
// that a real user never meets them, and an operator who needs different numbers is
// better served by the reverse proxy in front.
const (
	// The OAuth token endpoint. A browser login is one request; a refresh is one an
	// hour. Twenty in five minutes leaves room for a user fumbling their password.
	tokenRateLimit  = 20
	tokenRateWindow = 5 * time.Minute

	// The /api/open group: registration, password reset request, reset-code check,
	// e-mail verification. All of these are once-in-a-while actions.
	openRateLimit  = 15
	openRateWindow = 5 * time.Minute

	// Dynamic client registration (RFC 7591). Unauthenticated write access, so it
	// gets the tightest budget.
	registerRateLimit  = 5
	registerRateWindow = time.Hour

	// Per-account lockout on the password grant. Counted per e-mail address, so an
	// attacker rotating IPs still can't grind one account, and cleared on a
	// successful login so a user who eventually remembers their password is fine.
	// The lock lifts as the failures age out of the window rather than on a separate
	// timer, so there is only one clock to reason about.
	loginFailureLimit  = 5
	loginFailureWindow = 15 * time.Minute

	// Concurrent password hashes. bcrypt at cost 14 is deliberately ~1s of a core,
	// which is a fine defence against offline cracking and a liability when it is
	// reachable unauthenticated: without a cap, enough simultaneous logins saturate
	// the box even while every individual caller stays inside its rate limit. Half
	// the cores leaves the rest of the app responsive. Callers that cannot get a slot
	// within passwordCostWaitTimeout are shed with a 503 rather than queued forever.
	passwordCostWaitTimeout = 5 * time.Second
)

// slidingWindow counts events per key inside a moving time window. Keys are client
// IPs (or, for the login tracker, account identifiers).
type slidingWindow struct {
	mutex  sync.Mutex
	events map[string][]time.Time
	limit  int
	window time.Duration
}

func newSlidingWindow(limit int, window time.Duration) *slidingWindow {
	limiter := &slidingWindow{
		events: map[string][]time.Time{},
		limit:  limit,
		window: window,
	}
	go limiter.janitor()
	return limiter
}

// prune drops events that have fallen out of the window. The caller holds the mutex.
func (limiter *slidingWindow) prune(key string, now time.Time) []time.Time {
	cutoff := now.Add(-limiter.window)
	kept := limiter.events[key][:0]
	for _, stamp := range limiter.events[key] {
		if stamp.After(cutoff) {
			kept = append(kept, stamp)
		}
	}
	if len(kept) == 0 {
		delete(limiter.events, key)
		return nil
	}
	limiter.events[key] = kept
	return kept
}

// allow records a hit for key and reports whether it fits inside the window. When it
// does not, retryAfter says how long until the oldest hit expires and a slot frees up.
func (limiter *slidingWindow) allow(key string) (allowed bool, retryAfter time.Duration) {
	now := time.Now()

	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	current := limiter.prune(key, now)
	if len(current) >= limiter.limit {
		return false, limiter.window - now.Sub(current[0])
	}

	limiter.events[key] = append(current, now)
	return true, 0
}

// count reports how many hits are currently inside the window, without recording one.
func (limiter *slidingWindow) count(key string) int {
	now := time.Now()

	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	return len(limiter.prune(key, now))
}

// record adds a hit without applying the limit. Used by the login failure tracker,
// where the decision to block is taken separately from the decision to count.
func (limiter *slidingWindow) record(key string) {
	now := time.Now()

	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	limiter.events[key] = append(limiter.prune(key, now), now)
}

func (limiter *slidingWindow) forget(key string) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	delete(limiter.events, key)
}

// janitor evicts keys nobody has touched for a window's length, so a long-running
// instance that has been scanned doesn't hold every IP it ever saw.
func (limiter *slidingWindow) janitor() {
	for {
		time.Sleep(limiter.window)

		now := time.Now()
		limiter.mutex.Lock()
		for key := range limiter.events {
			limiter.prune(key, now)
		}
		limiter.mutex.Unlock()
	}
}

var (
	tokenLimiter    = newSlidingWindow(tokenRateLimit, tokenRateWindow)
	openLimiter     = newSlidingWindow(openRateLimit, openRateWindow)
	registerLimiter = newSlidingWindow(registerRateLimit, registerRateWindow)

	loginFailures = newSlidingWindow(loginFailureLimit, loginFailureWindow)
)

// rateLimit builds a gin middleware that limits by client IP. Note this relies on
// the engine's trusted-proxy configuration being correct — see SetTrustedProxies in
// main.go — since ClientIP() otherwise believes a spoofed X-Forwarded-For.
func rateLimit(limiter *slidingWindow) gin.HandlerFunc {
	return func(context *gin.Context) {
		allowed, retryAfter := limiter.allow(context.ClientIP())
		if !allowed {
			writeTooManyRequests(context, retryAfter)
			return
		}
		context.Next()
	}
}

// RateLimitTokenEndpoint throttles POST /api/oauth/token.
func RateLimitTokenEndpoint() gin.HandlerFunc { return rateLimit(tokenLimiter) }

// RateLimitOpenEndpoints throttles the unauthenticated /api/open group.
func RateLimitOpenEndpoints() gin.HandlerFunc { return rateLimit(openLimiter) }

// RateLimitClientRegistration throttles OAuth dynamic client registration.
func RateLimitClientRegistration() gin.HandlerFunc { return rateLimit(registerLimiter) }

func writeTooManyRequests(context *gin.Context, retryAfter time.Duration) {
	seconds := int(retryAfter.Seconds()) + 1
	if seconds < 1 {
		seconds = 1
	}
	context.Header("Retry-After", strconv.Itoa(seconds))
	context.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests. Try again later."})
	context.Abort()
}

// LoginAttemptLocked reports whether an account has failed too many password grants
// recently, and how long the caller should wait. The key is the submitted username,
// normalized by the caller.
func LoginAttemptLocked(accountKey string) (locked bool, retryAfter time.Duration) {
	if loginFailures.count(accountKey) < loginFailureLimit {
		return false, 0
	}
	return true, loginFailureWindow
}

// RegisterLoginFailure counts a failed password grant against the account.
func RegisterLoginFailure(accountKey string) { loginFailures.record(accountKey) }

// ClearLoginFailures forgets an account's failures after a successful login.
func ClearLoginFailures(accountKey string) { loginFailures.forget(accountKey) }

// AbortLoginLocked writes the lockout response.
func AbortLoginLocked(context *gin.Context, retryAfter time.Duration) {
	writeTooManyRequests(context, retryAfter)
}

// passwordCostSlots bounds how many goroutines may be inside a bcrypt hash or compare
// at once. Sized from the machine rather than a constant so a one-core VM doesn't let
// four requests monopolise it.
var passwordCostSlots = make(chan struct{}, maxConcurrentPasswordCost())

func maxConcurrentPasswordCost() int {
	slots := runtime.NumCPU() / 2
	if slots < 1 {
		slots = 1
	}
	return slots
}

// AcquirePasswordCostSlot blocks until a bcrypt slot is free, returning a release
// function. It reports false if the wait timed out, in which case the caller must not
// hash and should shed the request (see AbortPasswordCostBusy) — the alternative is an
// unbounded queue of second-long operations.
func AcquirePasswordCostSlot() (release func(), ok bool) {
	timer := time.NewTimer(passwordCostWaitTimeout)
	defer timer.Stop()

	select {
	case passwordCostSlots <- struct{}{}:
		return func() { <-passwordCostSlots }, true
	case <-timer.C:
		return func() {}, false
	}
}

// AbortPasswordCostBusy writes the response for a request shed because every password
// hashing slot was busy. 503 (not 429) because it is the server that is saturated, not
// this caller that misbehaved.
func AbortPasswordCostBusy(context *gin.Context) {
	context.Header("Retry-After", strconv.Itoa(int(passwordCostWaitTimeout.Seconds())))
	context.JSON(http.StatusServiceUnavailable, gin.H{"error": "Server is busy. Try again shortly."})
	context.Abort()
}
