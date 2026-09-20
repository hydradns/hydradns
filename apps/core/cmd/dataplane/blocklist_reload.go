// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"sync"
	"time"

	"github.com/hydradns/hydradns/apps/core/internal/blocklist"
	"github.com/hydradns/hydradns/apps/core/internal/logger"
	"github.com/hydradns/hydradns/apps/core/internal/storage/repositories"
)

// blocklistSource is the subset of *blocklist.Engine the reloader needs.
// Narrowed to an interface so tests can supply a fake without a real DB.
type blocklistSource interface {
	Signature() (repositories.BlocklistSignature, error)
	List() ([]string, error)
}

// blocklistReloader closes the propagation gap described in CLAUDE.md's
// "DNS Query Pipeline" section: without it, a blocklist add / enable /
// disable / delete / edit made via the dashboard, CLI, or MCP tools only
// reaches the DNS hot path's in-memory set on the next periodic
// fetch-loop pass (BLOCKLIST_UPDATE_INTERVAL, default 6h) or at restart,
// while policy changes already propagate within about 5 seconds
// (cmd/dataplane/main.go's reloadPolicies ticker). Poll is meant to be
// called frequently (see startBlocklistPoll) with a cheap signature check
// so the (potentially very large, millions of rows on a full blocklist
// set) List()+MemoryChecker.Reload() only happens when something in the
// blocklist DB state actually changed.
//
// Concurrency: Poll may be called from multiple goroutines at once (the
// signature-poll ticker and the post-fetch call the network-refresh loop
// makes). Rebuilds are single-flighted (at most one rebuildOnce runs at a
// time), and bursts are coalesced: if N signature changes are observed
// while a rebuild is already running, exactly one more rebuild happens
// after it finishes, not N. The rebuild itself always runs in its own
// goroutine so Poll never blocks its caller on a potentially multi-second
// full-entry-table read.
//
// Failure handling: lastSig/haveSig are only advanced *after*
// mem.Reload() has succeeded for the signature that triggered the rebuild.
// A List() error leaves them exactly as they were: never a populated set
// silently replaced by an empty one, and the next Poll() (whether from the
// signature-poll ticker or the 6h refresh pass) sees the same "changed"
// condition and retries. The poll interval (default 5s) is the retry
// cadence; there is no separate backoff timer for the periodic path, so a
// persistent error costs one List() call per poll tick, not a hot loop.
type blocklistReloader struct {
	engine blocklistSource
	mem    *blocklist.MemoryChecker

	mu      sync.Mutex
	lastSig repositories.BlocklistSignature
	// haveSig is true only once a rebuild has actually *succeeded* for
	// lastSig. Setting it (or lastSig) before the rebuild it triggered had
	// run would make a failed rebuild indistinguishable from a successful
	// one to every later Poll() call.
	haveSig    bool
	rebuilding bool
	pending    bool
	// pendingSig is the most recently observed signature that still needs
	// a rebuild attempt, set by whichever of Poll/ForceRebuild most
	// recently triggered or coalesced into the in-flight rebuild. The
	// rebuild goroutine reads it at the start of each pass.
	pendingSig repositories.BlocklistSignature
	// consecutiveFailures rate-limits error logging: only the first
	// failure in a streak logs an ERROR line, and a subsequent success
	// after a streak logs one recovery line. Without this, a persistent
	// outage would emit one ERROR per poll tick forever.
	consecutiveFailures int
}

func newBlocklistReloader(engine blocklistSource, mem *blocklist.MemoryChecker) *blocklistReloader {
	return &blocklistReloader{engine: engine, mem: mem}
}

// Poll computes the current blocklist signature and, if it differs from the
// last one this reloader successfully rebuilt from, ensures a rebuild
// happens (now, or immediately after the in-flight one if a rebuild is
// already running). It never blocks waiting for the rebuild itself.
func (r *blocklistReloader) Poll() {
	r.poll(false)
}

// ForceRebuild triggers a rebuild unconditionally, ignoring the signature
// comparison. This is the safety net the 6h refreshSources pass uses (see
// main.go): cheap relative to a 6h cadence, and it lets the in-memory set
// self-heal from any bug that might otherwise desynchronize it from the DB
// without a matching signature change. Still funnels through the same
// single-flight/coalescing machinery as Poll, so it never races a
// concurrently-running rebuild.
func (r *blocklistReloader) ForceRebuild() {
	r.poll(true)
}

func (r *blocklistReloader) poll(force bool) {
	sig, err := r.engine.Signature()
	if err != nil {
		logger.Log.Errorf("blocklist signature check failed: %v", err)
		if !force {
			return
		}
		// A forced rebuild's whole point is to resync independent of the
		// signature check, so still attempt it even without a fresh
		// signature: worst case it reuses the zero value, which only
		// risks a spurious "unchanged" skip on some future Poll(), not a
		// missed or empty rebuild now.
	}

	r.mu.Lock()
	if !force && r.haveSig && sig == r.lastSig {
		r.mu.Unlock()
		return
	}
	r.pendingSig = sig
	if r.rebuilding {
		// A rebuild is already running and started reading the DB before
		// this change landed, so it won't reflect it. Flag exactly one
		// more pass instead of starting a second rebuild concurrently.
		r.pending = true
		r.mu.Unlock()
		return
	}
	r.rebuilding = true
	r.mu.Unlock()

	go r.runRebuild()
}

// runRebuild drives rebuildOnce, looping exactly once more if a change was
// coalesced in while it was running, then exiting. Only one goroutine can
// be inside this function at a time (guarded by the rebuilding flag in
// poll), so rebuildOnce itself never runs concurrently with itself.
func (r *blocklistReloader) runRebuild() {
	for {
		r.mu.Lock()
		sig := r.pendingSig
		r.mu.Unlock()

		r.rebuildOnce(sig)

		r.mu.Lock()
		if r.pending {
			r.pending = false
			r.mu.Unlock()
			continue
		}
		r.rebuilding = false
		r.mu.Unlock()
		return
	}
}

// rebuildOnce attempts one List()+MemoryChecker.Reload() pass. On success,
// sig becomes the new lastSig/haveSig, committed only now, never before
// the rebuild it describes has actually landed in mem. On failure, mem is
// left untouched (never swapped for an empty set) and
// lastSig/haveSig are left exactly as they were, so the condition that
// causes the next Poll()/ForceRebuild() to trigger a rebuild is unchanged
// and it will retry.
func (r *blocklistReloader) rebuildOnce(sig repositories.BlocklistSignature) {
	start := time.Now()
	domains, err := r.engine.List()
	if err != nil {
		r.mu.Lock()
		r.consecutiveFailures++
		streak := r.consecutiveFailures
		r.mu.Unlock()
		if streak == 1 {
			logger.Log.Errorf("failed to load blocklist domains into memory (will retry on next poll): %v", err)
		}
		return
	}

	r.mem.Reload(domains)

	r.mu.Lock()
	recovered := r.consecutiveFailures > 0
	r.consecutiveFailures = 0
	r.lastSig = sig
	r.haveSig = true
	r.mu.Unlock()

	if recovered {
		logger.Log.Infof("blocklist rebuild recovered after prior failures: %d domains in %s", len(domains), time.Since(start))
	} else {
		logger.Log.Infof("blocklist rebuild complete: %d domains in %s", len(domains), time.Since(start))
	}
}

// loaded reports whether a rebuild has ever succeeded. Used by
// runInitialBlocklistLoad to decide whether to keep retrying.
func (r *blocklistReloader) loaded() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.haveSig
}

// pollAndWait triggers Poll() and blocks (busy-polling, this is only ever
// called from the dedicated startup-retry goroutine, never the DNS hot
// path) until that rebuild attempt, including any pass coalesced in while
// it ran, has settled, then reports whether the reloader is loaded.
func (r *blocklistReloader) pollAndWait() bool {
	r.Poll()
	for {
		r.mu.Lock()
		rebuilding := r.rebuilding
		have := r.haveSig
		r.mu.Unlock()
		if !rebuilding {
			return have
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// runInitialBlocklistLoad blocks until the reloader has successfully loaded
// the blocklist at least once, retrying on failure: a single fire-and-forget
// `go reloader.Poll()` would leave the in-memory blocklist permanently empty
// after one transient error (e.g. "database is locked" while the control
// plane's AutoMigrate runs concurrently against the same SQLite file).
// Meant to be run in its own goroutine so it doesn't delay the DNS server
// starting up. backoff is the delay between attempts; <= 0 falls back to
// a sensible default so the initial load still retries even when the
// periodic signature poll is disabled via BLOCKLIST_POLL_INTERVAL=0.
func runInitialBlocklistLoad(r *blocklistReloader, backoff time.Duration) {
	if backoff <= 0 {
		backoff = 5 * time.Second
	}
	for {
		if r.pollAndWait() {
			return
		}
		time.Sleep(backoff)
	}
}

// startBlocklistPoll runs reloader.Poll() every interval in the
// background. interval <= 0 disables the loop entirely: propagation of
// CRUD changes then falls back to whatever else calls Poll() (the
// post-fetch call in the network-refresh loop, and the initial load
// retried by runInitialBlocklistLoad at startup), matching the
// pre-existing bound.
//
// Returns a stop func. Production callers must wire this into shutdown
// (see main.go) rather than discard it, since it's the only handle on the
// background goroutine; tests use it the same way to shut it down cleanly.
func startBlocklistPoll(reloader *blocklistReloader, interval time.Duration) (stop func()) {
	if interval <= 0 {
		return func() {}
	}
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				reloader.Poll()
			case <-done:
				return
			}
		}
	}()
	var once sync.Once
	return func() { once.Do(func() { close(done) }) }
}
