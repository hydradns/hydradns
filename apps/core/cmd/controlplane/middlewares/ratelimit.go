// SPDX-License-Identifier: Apache-2.0
package middlewares

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// evictScanLimit bounds how many map entries evictLocked inspects looking
// for an already-expired one, regardless of maxEntries. Without this bound,
// a full map with no expired entries forces a scan of every one of
// maxEntries entries, under the single global mutex, on every request from
// a new key. An attacker with a large address pool (e.g. an IPv6 /64) can
// turn that into an O(maxEntries) lock hold on every login/setup request
// from a fresh IP.
const evictScanLimit = 64

// RateLimiter is a fixed-window, per-key budget tracker: at most `limit`
// calls to RecordFailure(key) are recorded within any `window`-long span
// before Blocked(key) starts returning true (with the remaining time until
// the window resets). Safe for concurrent use.
//
// Blocked and RecordFailure are deliberately separate: checking whether a
// key is currently blocked must not, by itself, consume any budget, so a
// caller can check-then-only-record-on-failure (see LoginThrottle) instead
// of every call, success or failure, counting against the same budget.
//
// Memory is bounded by maxEntries: once the entry map is full, a new key
// tries to evict an already-expired entry first (bounded to evictScanLimit
// entries inspected); if none is found in that sample, the entire map is
// reset rather than scanning the rest of it under the lock. A fixed-window
// limiter loses little by resetting occasionally under sustained pressure:
// every tracked key gets a fresh budget slightly early, which is no worse
// than the window boundary it would have hit anyway.
type RateLimiter struct {
	mu         sync.Mutex
	entries    map[string]*rateLimitEntry
	limit      int
	window     time.Duration
	maxEntries int
	now        func() time.Time
}

type rateLimitEntry struct {
	count       int
	windowStart time.Time
}

// NewRateLimiter builds a limiter allowing `limit` calls per `window` per
// key, capping the number of tracked keys at maxEntries.
func NewRateLimiter(limit int, window time.Duration, maxEntries int) *RateLimiter {
	return newRateLimiter(limit, window, maxEntries, time.Now)
}

// newRateLimiter is the internal constructor tests use to inject a clock,
// so window-expiry behavior can be tested deterministically instead of
// with a real sleep.
func newRateLimiter(limit int, window time.Duration, maxEntries int, now func() time.Time) *RateLimiter {
	return &RateLimiter{
		entries:    make(map[string]*rateLimitEntry),
		limit:      limit,
		window:     window,
		maxEntries: maxEntries,
		now:        now,
	}
}

// Blocked reports whether key is currently over budget, without recording
// an attempt. retryAfter is the remaining time until the window resets;
// only meaningful when blocked is true. Callers should check this before
// doing any expensive work for the request (bcrypt, a DB lookup) and
// before deciding whether the outcome even counts as a failure.
func (rl *RateLimiter) Blocked(key string) (blocked bool, retryAfter time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	e, ok := rl.entries[key]
	if !ok || now.Sub(e.windowStart) >= rl.window {
		return false, 0
	}
	if e.count >= rl.limit {
		return true, rl.window - now.Sub(e.windowStart)
	}
	return false, 0
}

// RecordFailure counts one failed attempt against key's budget, starting a
// fresh window if key is new or its previous window has expired. Callers
// must only invoke this after the request's outcome is known to be a
// failure (see LoginThrottle); a successful request must never call this,
// or it defeats the whole point of separating Blocked from RecordFailure.
func (rl *RateLimiter) RecordFailure(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	if e, ok := rl.entries[key]; ok && now.Sub(e.windowStart) < rl.window {
		e.count++
		return
	}

	// Key is new, or its window has expired: start a fresh window. Make
	// room first so the map never exceeds maxEntries.
	if _, ok := rl.entries[key]; !ok {
		rl.evictLocked(now)
	}
	rl.entries[key] = &rateLimitEntry{count: 1, windowStart: now}
}

// evictLocked frees room when the map is at capacity: it inspects up to
// evictScanLimit entries (Go's map iteration order is randomized, so this
// is an effectively random sample, not always the same entries) looking
// for one whose window has already expired. If it finds one, that single
// entry is deleted. This is the common case under normal traffic, where
// expired entries are plentiful. If the sample turns up nothing evictable
// (every entry it looked at is still live), the whole map is reset instead of
// scanning the remaining maxEntries-evictScanLimit entries under the lock.
// Called with rl.mu held.
func (rl *RateLimiter) evictLocked(now time.Time) {
	if len(rl.entries) < rl.maxEntries {
		return
	}
	scanned := 0
	for k, e := range rl.entries {
		if now.Sub(e.windowStart) >= rl.window {
			delete(rl.entries, k)
			return
		}
		scanned++
		if scanned >= evictScanLimit {
			break
		}
	}
	rl.entries = make(map[string]*rateLimitEntry, rl.maxEntries)
}

// LoginThrottle wraps a RateLimiter as gin middleware keyed by the
// request's client IP. Intended for POST /auth/login and POST
// /auth/setup only: unauthenticated endpoints where a brute-force
// attempt is otherwise unthrottled.
//
// Only failed attempts (a 4xx response: invalid credentials, a bad
// request body, setup-already-complete, etc.) consume budget; the
// middleware checks the budget before the handler runs (so an
// already-blocked caller never reaches bcrypt or the DB), then records a
// failure only after seeing the response status. Charging every call
// equally, success or failure, would let a legitimate admin behind a
// shared NAT (an office, a coaching centre, the demo's own reverse proxy)
// get locked out of their own appliance purely by the dashboard, the CLI,
// and a phone all re-authenticating successfully in the same window. A 5xx
// (the server's own fault: a bcrypt or DB error) does not count either: it
// is not evidence of a brute-force attempt or a mistyped password.
//
// The key MUST come from a client IP that cannot be spoofed by the
// caller. c.ClientIP() falls back to trusting X-Forwarded-For /
// X-Real-IP when gin has trusted proxies configured; this API calls
// r.SetTrustedProxies(nil) in main.go specifically so c.ClientIP() always
// resolves to the real socket address here, and a client cannot dodge
// its own per-IP budget (or frame another IP) by sending a fabricated
// X-Forwarded-For header.
func LoginThrottle(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if blocked, retryAfter := rl.Blocked(ip); blocked {
			secs := int(retryAfter.Seconds())
			if secs < 1 {
				secs = 1
			}
			c.Header("Retry-After", strconv.Itoa(secs))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"status": "error",
				"error":  "too many attempts, try again later",
			})
			return
		}

		c.Next()

		if status := c.Writer.Status(); status >= 400 && status < 500 {
			rl.RecordFailure(ip)
		}
	}
}
