// SPDX-License-Identifier: GPL-3.0-or-later
package middlewares

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter is a fixed-window, per-key request limiter: at most `limit`
// calls to Allow(key) succeed within any `window`-long span before Allow
// starts returning false (with the remaining time until the window
// resets). Safe for concurrent use.
//
// Memory is bounded by maxEntries: once the entry map is full, a new key
// evicts either an already-expired entry or (failing that) the oldest
// window, so a flood of distinct source IPs cannot grow this map without
// bound.
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

// Allow reports whether the caller identified by key may proceed. When it
// returns false, retryAfter is how long the caller should wait before the
// window resets.
func (rl *RateLimiter) Allow(key string) (allowed bool, retryAfter time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	if e, ok := rl.entries[key]; ok && now.Sub(e.windowStart) < rl.window {
		if e.count >= rl.limit {
			return false, rl.window - now.Sub(e.windowStart)
		}
		e.count++
		return true, 0
	}

	// Key is new, or its window has expired: start a fresh window. Make
	// room first so the map never exceeds maxEntries.
	if _, ok := rl.entries[key]; !ok {
		rl.evictLocked(now)
	}
	rl.entries[key] = &rateLimitEntry{count: 1, windowStart: now}
	return true, 0
}

// evictLocked frees one slot when the map is at capacity: it prefers
// deleting any entry whose window has already expired, and falls back to
// the single oldest window otherwise. Called with rl.mu held.
func (rl *RateLimiter) evictLocked(now time.Time) {
	if len(rl.entries) < rl.maxEntries {
		return
	}
	var oldestKey string
	var oldestStart time.Time
	first := true
	for k, e := range rl.entries {
		if now.Sub(e.windowStart) >= rl.window {
			delete(rl.entries, k)
			return
		}
		if first || e.windowStart.Before(oldestStart) {
			oldestKey, oldestStart = k, e.windowStart
			first = false
		}
	}
	if oldestKey != "" {
		delete(rl.entries, oldestKey)
	}
}

// LoginThrottle wraps a RateLimiter as gin middleware keyed by the
// request's client IP. Intended for POST /auth/login and POST
// /auth/setup only — unauthenticated endpoints where a brute-force
// attempt is otherwise unthrottled.
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
		allowed, retryAfter := rl.Allow(ip)
		if !allowed {
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
	}
}
