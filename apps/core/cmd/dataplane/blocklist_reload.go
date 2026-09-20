// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"sync"
	"time"

	"github.com/hydradns/hydra-core/internal/blocklist"
	"github.com/hydradns/hydra-core/internal/logger"
	"github.com/hydradns/hydra-core/internal/storage/repositories"
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
// so the (potentially very large — millions of rows on a full blocklist
// set) List()+MemoryChecker.Reload() only happens when something in the
// blocklist DB state actually changed.
//
// Concurrency: Poll may be called from multiple goroutines at once (the
// signature-poll ticker and the post-fetch call the network-refresh loop
// makes). Rebuilds are single-flighted — at most one rebuildOnce runs at a
// time — and bursts are coalesced: if N signature changes are observed
// while a rebuild is already running, exactly one more rebuild happens
// after it finishes, not N. The rebuild itself always runs in its own
// goroutine so Poll never blocks its caller on a potentially multi-second
// full-entry-table read.
type blocklistReloader struct {
	engine blocklistSource
	mem    *blocklist.MemoryChecker

	mu         sync.Mutex
	lastSig    repositories.BlocklistSignature
	haveSig    bool
	rebuilding bool
	pending    bool
}

func newBlocklistReloader(engine blocklistSource, mem *blocklist.MemoryChecker) *blocklistReloader {
	return &blocklistReloader{engine: engine, mem: mem}
}

// Poll computes the current blocklist signature and, if it differs from
// the last one this reloader observed, ensures a rebuild happens (now, or
// immediately after the in-flight one if a rebuild is already running).
// It never blocks waiting for the rebuild itself.
func (r *blocklistReloader) Poll() {
	sig, err := r.engine.Signature()
	if err != nil {
		logger.Log.Errorf("blocklist signature check failed: %v", err)
		return
	}

	r.mu.Lock()
	if r.haveSig && sig == r.lastSig {
		r.mu.Unlock()
		return
	}
	r.lastSig = sig
	r.haveSig = true

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
// Poll), so rebuildOnce itself never runs concurrently with itself.
func (r *blocklistReloader) runRebuild() {
	for {
		r.rebuildOnce()

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

func (r *blocklistReloader) rebuildOnce() {
	start := time.Now()
	domains, err := r.engine.List()
	if err != nil {
		logger.Log.Errorf("failed to load blocklist domains into memory: %v", err)
		return
	}
	r.mem.Reload(domains)
	logger.Log.Infof("blocklist rebuild complete: %d domains in %s", len(domains), time.Since(start))
}

// startBlocklistPoll runs reloader.Poll() every interval in the
// background. interval <= 0 disables the loop entirely: propagation of
// CRUD changes then falls back to whatever else calls Poll() (the
// post-fetch call in the network-refresh loop, and the one-time initial
// call at startup), matching the pre-existing bound.
//
// Returns a stop func; production callers can ignore it (the loop is
// meant to run for the life of the process, like every other dataplane
// background loop). Tests use it to shut the goroutine down cleanly.
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
