// SPDX-License-Identifier: GPL-3.0-or-later
package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestRateLimiter_AllowsUpToLimit(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(3, time.Minute, 100, clock)

	for i := 0; i < 3; i++ {
		ok, _ := rl.Allow("1.1.1.1")
		if !ok {
			t.Fatalf("attempt %d: expected allowed", i)
		}
	}
}

func TestRateLimiter_TripsAfterLimit(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(3, time.Minute, 100, clock)

	for i := 0; i < 3; i++ {
		rl.Allow("1.1.1.1")
	}
	ok, retryAfter := rl.Allow("1.1.1.1")
	if ok {
		t.Fatal("expected the 4th attempt within the window to be denied")
	}
	if retryAfter <= 0 || retryAfter > time.Minute {
		t.Errorf("expected a positive retry-after within the window, got %v", retryAfter)
	}
}

func TestRateLimiter_IndependentPerKey(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(1, time.Minute, 100, clock)

	ok, _ := rl.Allow("1.1.1.1")
	if !ok {
		t.Fatal("expected first IP's first attempt to be allowed")
	}
	ok, _ = rl.Allow("1.1.1.1")
	if ok {
		t.Fatal("expected first IP's second attempt to be denied")
	}

	// A different key must not be affected by the first key's exhausted budget.
	ok, _ = rl.Allow("2.2.2.2")
	if !ok {
		t.Fatal("expected a different IP to have its own independent budget")
	}
}

func TestRateLimiter_SuccessDoesNotResetOtherKeys(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(2, time.Minute, 100, clock)

	rl.Allow("1.1.1.1")
	rl.Allow("1.1.1.1") // 1.1.1.1 now at its limit

	// Simulate a bunch of successful calls from an unrelated key.
	for i := 0; i < 5; i++ {
		rl.Allow("2.2.2.2")
	}

	ok, _ := rl.Allow("1.1.1.1")
	if ok {
		t.Error("expected 1.1.1.1 to remain throttled regardless of 2.2.2.2's activity")
	}
}

func TestRateLimiter_WindowExpiryResetsBudget(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(2, 5*time.Minute, 100, clock)

	rl.Allow("1.1.1.1")
	rl.Allow("1.1.1.1")
	ok, _ := rl.Allow("1.1.1.1")
	if ok {
		t.Fatal("expected the 3rd attempt to be denied within the window")
	}

	clock.advance(5*time.Minute + time.Second)

	ok, _ = rl.Allow("1.1.1.1")
	if !ok {
		t.Error("expected the budget to reset once the window has fully elapsed")
	}
}

func TestRateLimiter_BoundedMemory_EvictsUnderPressure(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(10, time.Minute, 5, clock)

	for i := 0; i < 50; i++ {
		rl.Allow("client-" + strconv.Itoa(i))
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

	rl.Allow("stale") // window starts at t0
	clock.advance(2 * time.Minute)
	rl.Allow("fresh") // window starts later, "stale" is now expired

	// Map is now at capacity (2). A third distinct key must evict the
	// expired "stale" entry, not the live "fresh" one.
	rl.Allow("newcomer")

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

// --- LoginThrottle middleware tests ---

func newThrottledRouter(rl *RateLimiter) *gin.Engine {
	r := gin.New()
	// Mirrors main.go's r.SetTrustedProxies(nil): ClientIP() must resolve
	// to the real socket address, never a client-supplied
	// X-Forwarded-For, or the spoofing test below would be meaningless.
	if err := r.SetTrustedProxies(nil); err != nil {
		panic(err)
	}
	r.POST("/api/v1/auth/login", LoginThrottle(rl), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
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

func TestLoginThrottle_TripsWithRetryAfterHeader(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(2, 5*time.Minute, 1000, clock)
	r := newThrottledRouter(rl)

	doPost(r, "/api/v1/auth/login", "203.0.113.5:1111", "")
	doPost(r, "/api/v1/auth/login", "203.0.113.5:2222", "")
	rec := doPost(r, "/api/v1/auth/login", "203.0.113.5:3333", "")

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected a Retry-After header on 429")
	}
	if rec.Header().Get("Content-Type") == "" {
		t.Log("no content-type asserted; body envelope checked separately")
	}
}

func TestLoginThrottle_DifferentIPsAreIndependent(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(1, 5*time.Minute, 1000, clock)
	r := newThrottledRouter(rl)

	rec1 := doPost(r, "/api/v1/auth/login", "203.0.113.5:1111", "")
	rec2 := doPost(r, "/api/v1/auth/login", "203.0.113.9:1111", "")

	if rec1.Code != http.StatusOK || rec2.Code != http.StatusOK {
		t.Errorf("expected both distinct IPs' first attempt to succeed, got %d and %d", rec1.Code, rec2.Code)
	}
}

func TestLoginThrottle_SpoofedXFFDoesNotBypassLimiter(t *testing.T) {
	// With gin's default (untrusted-proxy) ClientIP resolution — the same
	// posture main.go configures via SetTrustedProxies(nil) — a caller
	// cannot dodge its own budget by sending a different X-Forwarded-For
	// on every request; ClientIP() must key off the real socket address.
	clock := newFakeClock()
	rl := newTestLimiter(1, 5*time.Minute, 1000, clock)
	r := newThrottledRouter(rl)

	doPost(r, "/api/v1/auth/login", "203.0.113.5:1111", "1.2.3.4")
	rec := doPost(r, "/api/v1/auth/login", "203.0.113.5:2222", "5.6.7.8")

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected the spoofed X-Forwarded-For to be ignored and the real IP throttled, got %d", rec.Code)
	}
}

func TestLoginThrottle_WindowExpiryAllowsRetry(t *testing.T) {
	clock := newFakeClock()
	rl := newTestLimiter(1, 5*time.Minute, 1000, clock)
	r := newThrottledRouter(rl)

	doPost(r, "/api/v1/auth/login", "203.0.113.5:1111", "")
	rec := doPost(r, "/api/v1/auth/login", "203.0.113.5:2222", "")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 before the window elapses, got %d", rec.Code)
	}

	clock.advance(5*time.Minute + time.Second)

	rec = doPost(r, "/api/v1/auth/login", "203.0.113.5:3333", "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected a fresh window to allow the request again, got %d", rec.Code)
	}
}
