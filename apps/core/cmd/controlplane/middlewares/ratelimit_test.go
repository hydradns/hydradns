// SPDX-License-Identifier: GPL-3.0-or-later
package middlewares

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// fakeClock lets tests advance time deterministically instead of sleeping.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock() *fakeClock { return &fakeClock{t: time.Now()} }

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func newTestLimiter(limit int, window time.Duration, maxEntries int, clock *fakeClock) *RateLimiter {
	return newRateLimiter(limit, window, maxEntries, clock.now)
}

// --- RateLimiter unit tests ---
//
// RecordFailure is the only thing that consumes budget: an Allow() that
// consumed budget on every call, including successful logins, could lock
// out a legitimate admin behind a shared NAT. Blocked() is a pure
// check-without-consuming.

func TestRateLimiter_AllowsUpToLimit(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(3, time.Minute, 100, clock)

	for i := 0; i < 3; i++ {
		if blocked, _ := rl.Blocked("1.1.1.1"); blocked {
			t.Fatalf("attempt %d: expected not blocked", i)
		}
		rl.RecordFailure("1.1.1.1")
	}
}

func TestRateLimiter_TripsAfterLimit(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(3, time.Minute, 100, clock)

	for i := 0; i < 3; i++ {
		rl.RecordFailure("1.1.1.1")
	}
	blocked, retryAfter := rl.Blocked("1.1.1.1")
	if !blocked {
		t.Fatal("expected the 4th attempt within the window to be blocked")
	}
	if retryAfter <= 0 || retryAfter > time.Minute {
		t.Errorf("expected a positive retry-after within the window, got %v", retryAfter)
	}
}

func TestRateLimiter_IndependentPerKey(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(1, time.Minute, 100, clock)

	if blocked, _ := rl.Blocked("1.1.1.1"); blocked {
		t.Fatal("expected first IP's first attempt to be allowed")
	}
	rl.RecordFailure("1.1.1.1")
	if blocked, _ := rl.Blocked("1.1.1.1"); !blocked {
		t.Fatal("expected first IP's second attempt to be denied")
	}

	// A different key must not be affected by the first key's exhausted budget.
	if blocked, _ := rl.Blocked("2.2.2.2"); blocked {
		t.Fatal("expected a different IP to have its own independent budget")
	}
}

func TestRateLimiter_SuccessDoesNotConsumeBudget(t *testing.T) {
	// Repeatedly checking Blocked() (what a successful, non-failing
	// request does under LoginThrottle) must never by itself exhaust the
	// budget.
	clock := newFakeClock()
	rl := newTestLimiter(2, time.Minute, 100, clock)

	for i := 0; i < 50; i++ {
		if blocked, _ := rl.Blocked("1.1.1.1"); blocked {
			t.Fatalf("attempt %d: Blocked() alone must never trip the limiter", i)
		}
	}
}

func TestRateLimiter_FailuresFromOtherKeysDoNotAffectThisKey(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(2, time.Minute, 100, clock)

	rl.RecordFailure("1.1.1.1")
	rl.RecordFailure("1.1.1.1") // 1.1.1.1 now at its limit

	for i := 0; i < 5; i++ {
		rl.RecordFailure("2.2.2.2")
	}

	blocked, _ := rl.Blocked("1.1.1.1")
	if !blocked {
		t.Error("expected 1.1.1.1 to remain throttled regardless of 2.2.2.2's activity")
	}
}

func TestRateLimiter_WindowExpiryResetsBudget(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(2, 5*time.Minute, 100, clock)

	rl.RecordFailure("1.1.1.1")
	rl.RecordFailure("1.1.1.1")
	if blocked, _ := rl.Blocked("1.1.1.1"); !blocked {
		t.Fatal("expected the 3rd attempt to be denied within the window")
	}

	clock.advance(5*time.Minute + time.Second)

	if blocked, _ := rl.Blocked("1.1.1.1"); blocked {
		t.Error("expected the budget to reset once the window has fully elapsed")
	}
}

func TestRateLimiter_BoundedMemory_EvictsUnderPressure(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(10, time.Minute, 5, clock)

	for i := 0; i < 50; i++ {
		rl.RecordFailure(fmt.Sprintf("client-%d", i))
	}

	rl.mu.Lock()
	n := len(rl.entries)
	rl.mu.Unlock()
	if n > 5 {
		t.Errorf("expected the entry map capped at maxEntries=5, got %d entries", n)
	}
}

func TestRateLimiter_EvictsExpiredEntriesBeforeOldestLive(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(10, time.Minute, 2, clock)

	rl.RecordFailure("stale") // window starts at t0
	clock.advance(2 * time.Minute)
	rl.RecordFailure("fresh") // window starts later, "stale" is now expired

	// Map is now at capacity (2). A third distinct key must evict the
	// expired "stale" entry, not the live "fresh" one.
	rl.RecordFailure("newcomer")

	rl.mu.Lock()
	_, staleStillPresent := rl.entries["stale"]
	_, freshStillPresent := rl.entries["fresh"]
	rl.mu.Unlock()
	if staleStillPresent {
		t.Error("expected the expired entry to be evicted first")
	}
	if !freshStillPresent {
		t.Error("expected the live entry to survive eviction")
	}
}

// TestRateLimiter_EvictionUnderAllLiveEntriesStaysBounded proves eviction
// under sustained pressure (every entry live, none expired) must not scan
// the entire map on every insert: it resets the map instead once a bounded
// sample finds nothing to reclaim. We can't easily assert "didn't do O(n)
// work" from outside, so this asserts the observable contract instead: the
// map never exceeds maxEntries even when every single entry is live, the
// worst case for a naive "scan everything" eviction strategy.
func TestRateLimiter_EvictionUnderAllLiveEntriesStaysBounded(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(10, time.Hour, 100, clock)

	for i := 0; i < 500; i++ {
		rl.RecordFailure(fmt.Sprintf("client-%d", i))
	}

	rl.mu.Lock()
	n := len(rl.entries)
	rl.mu.Unlock()
	if n > 100 {
		t.Errorf("expected the entry map capped at maxEntries=100, got %d entries", n)
	}
}

// --- LoginThrottle middleware tests ---
//
// The stub handler's status is configurable so tests can distinguish
// "successful login" (must not consume budget) from "failed login" (must
// consume budget).

func newThrottledRouter(rl *RateLimiter, status int) *gin.Engine {
	r := gin.New()
	// Mirrors main.go's r.SetTrustedProxies(nil): ClientIP() must resolve
	// to the real socket address, never a client-supplied
	// X-Forwarded-For, or the spoofing test below would be meaningless.
	if err := r.SetTrustedProxies(nil); err != nil {
		panic(err)
	}
	r.POST("/api/v1/auth/login", LoginThrottle(rl), func(c *gin.Context) {
		c.JSON(status, gin.H{"status": "success"})
	})
	return r
}

func doPost(r *gin.Engine, path, remoteAddr string, xff string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, nil)
	req.RemoteAddr = remoteAddr
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestLoginThrottle_TripsWithRetryAfterHeaderOnFailures(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(2, 5*time.Minute, 1000, clock)
	r := newThrottledRouter(rl, http.StatusUnauthorized)

	doPost(r, "/api/v1/auth/login", "203.0.113.5:1111", "")
	doPost(r, "/api/v1/auth/login", "203.0.113.5:2222", "")
	rec := doPost(r, "/api/v1/auth/login", "203.0.113.5:3333", "")

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected a Retry-After header on 429")
	}
}

// TestLoginThrottle_SuccessfulLoginsDoNotConsumeBudget verifies a
// legitimate admin behind a shared NAT, or the dashboard/CLI/phone all
// re-authenticating in the same window, must never be locked out purely
// by succeeding repeatedly.
func TestLoginThrottle_SuccessfulLoginsDoNotConsumeBudget(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(2, 5*time.Minute, 1000, clock)
	r := newThrottledRouter(rl, http.StatusOK)

	for i := 0; i < 20; i++ {
		rec := doPost(r, "/api/v1/auth/login", "203.0.113.5:1111", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("attempt %d: expected 200, got %d", i, rec.Code)
		}
	}
}

// TestLoginThrottle_FailuresStillTripAfterManySuccesses proves the budget
// tracks failures specifically, not just "is exempt from limiting
// entirely": mix successes and failures, and confirm the failures alone
// are what eventually trips the limiter.
func TestLoginThrottle_FailuresStillTripAfterManySuccesses(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(2, 5*time.Minute, 1000, clock)
	rOK := newThrottledRouter(rl, http.StatusOK)
	rFail := newThrottledRouter(rl, http.StatusUnauthorized)

	for i := 0; i < 10; i++ {
		doPost(rOK, "/api/v1/auth/login", "203.0.113.5:1111", "")
	}

	doPost(rFail, "/api/v1/auth/login", "203.0.113.5:2222", "")
	doPost(rFail, "/api/v1/auth/login", "203.0.113.5:3333", "")
	rec := doPost(rFail, "/api/v1/auth/login", "203.0.113.5:4444", "")

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected the 3rd failure to trip the limiter regardless of prior successes, got %d", rec.Code)
	}
}

func TestLoginThrottle_ServerErrorDoesNotConsumeBudget(t *testing.T) {
	// A 5xx is the server's own fault (e.g. a bcrypt or DB error), not
	// evidence of a brute-force attempt or a mistyped password: only
	// 4xx (401/400/409 etc.) counts against the budget.
	clock := newFakeClock()
	rl := newTestLimiter(1, 5*time.Minute, 1000, clock)
	r := newThrottledRouter(rl, http.StatusInternalServerError)

	for i := 0; i < 10; i++ {
		rec := doPost(r, "/api/v1/auth/login", "203.0.113.5:1111", "")
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("attempt %d: expected the handler's 500 to pass through, got %d", i, rec.Code)
		}
	}
}

func TestLoginThrottle_DifferentIPsAreIndependent(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(1, 5*time.Minute, 1000, clock)
	r := newThrottledRouter(rl, http.StatusUnauthorized)

	rec1 := doPost(r, "/api/v1/auth/login", "203.0.113.5:1111", "")
	rec2 := doPost(r, "/api/v1/auth/login", "203.0.113.9:1111", "")

	if rec1.Code != http.StatusUnauthorized || rec2.Code != http.StatusUnauthorized {
		t.Errorf("expected both distinct IPs' first attempt to reach the handler, got %d and %d", rec1.Code, rec2.Code)
	}
}

func TestLoginThrottle_SpoofedXFFDoesNotBypassLimiter(t *testing.T) {
	// With gin's default (untrusted-proxy) ClientIP resolution (the same
	// posture main.go configures via SetTrustedProxies(nil)), a caller
	// cannot dodge its own budget by sending a different X-Forwarded-For
	// on every request; ClientIP() must key off the real socket address.
	clock := newFakeClock()
	rl := newTestLimiter(1, 5*time.Minute, 1000, clock)
	r := newThrottledRouter(rl, http.StatusUnauthorized)

	doPost(r, "/api/v1/auth/login", "203.0.113.5:1111", "1.2.3.4")
	rec := doPost(r, "/api/v1/auth/login", "203.0.113.5:2222", "5.6.7.8")

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected the spoofed X-Forwarded-For to be ignored and the real IP throttled, got %d", rec.Code)
	}
}

func TestLoginThrottle_WindowExpiryAllowsRetry(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(1, 5*time.Minute, 1000, clock)
	r := newThrottledRouter(rl, http.StatusUnauthorized)

	doPost(r, "/api/v1/auth/login", "203.0.113.5:1111", "")
	rec := doPost(r, "/api/v1/auth/login", "203.0.113.5:2222", "")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 before the window elapses, got %d", rec.Code)
	}

	clock.advance(5*time.Minute + time.Second)

	rec = doPost(r, "/api/v1/auth/login", "203.0.113.5:3333", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected a fresh window to allow the request through again, got %d", rec.Code)
	}
}

func TestLoginThrottle_BlockedRequestNeverReachesHandler(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(1, 5*time.Minute, 1000, clock)
	reached := 0
	r := gin.New()
	if err := r.SetTrustedProxies(nil); err != nil {
		panic(err)
	}
	r.POST("/api/v1/auth/login", LoginThrottle(rl), func(c *gin.Context) {
		reached++
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error"})
	})

	doPost(r, "/api/v1/auth/login", "203.0.113.5:1111", "")
	doPost(r, "/api/v1/auth/login", "203.0.113.5:2222", "")
	if reached != 1 {
		t.Errorf("expected the handler to run exactly once before the limiter trips, got %d", reached)
	}
}
