// SPDX-License-Identifier: GPL-3.0-or-later
package blocklist

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
)

func TestMemoryChecker_ParentMatching(t *testing.T) {
	m := NewMemoryChecker()
	m.Reload([]string{"ads.google.com", "doubleclick.net", "evil.example.org"})

	cases := []struct {
		domain string
		want   bool
	}{
		{"ads.google.com", true},          // exact
		{"www.ads.google.com", true},      // child of blocked parent
		{"x.y.ads.google.com", true},      // deep child
		{"ADS.GOOGLE.COM.", true},         // case + trailing dot normalized
		{"google.com", false},             // parent of a blocked domain is not blocked
		{"com", false},                    // bare TLD never blocked
		{"doubleclick.net", true},         // exact
		{"sub.doubleclick.net", true},     // child
		{"notdoubleclick.net", false},     // sibling, not a child
		{"evil.example.org", true},        // exact
		{"example.org", false},            // parent not blocked
		{"safe.com", false},               // unrelated
		{"", false},                       // empty
		{"localhost", false},              // single label
	}
	for _, c := range cases {
		got, err := m.IsBlocked(c.domain)
		if err != nil {
			t.Fatalf("IsBlocked(%q) error: %v", c.domain, err)
		}
		if got != c.want {
			t.Errorf("IsBlocked(%q) = %v, want %v", c.domain, got, c.want)
		}
	}
}

// Standard hosts blocklists contain single-label entries like "localhost"
// and "broadcasthost". The old DB checker never matched single-label
// queries (empty candidate set); the in-memory checker must not either,
// or it would REFUSE local name resolution.
func TestMemoryChecker_SingleLabelNeverBlocked(t *testing.T) {
	m := NewMemoryChecker()
	m.Reload([]string{"localhost", "broadcasthost", "ip6-localhost", "ads.google.com"})

	for _, d := range []string{"localhost", "broadcasthost", "ip6-localhost", "wpad"} {
		if got, _ := m.IsBlocked(d); got {
			t.Errorf("IsBlocked(%q) = true, want false (single-label must never block)", d)
		}
	}
	// Multi-label entries still work.
	if got, _ := m.IsBlocked("ads.google.com"); !got {
		t.Error("multi-label entry should still block")
	}
}

func TestMemoryChecker_EmptyFailsOpen(t *testing.T) {
	m := NewMemoryChecker()
	// No Reload yet, nothing is blocked.
	if got, _ := m.IsBlocked("ads.google.com"); got {
		t.Error("empty checker must not block (fail-open during startup)")
	}
	if m.Count() != 0 {
		t.Errorf("Count = %d, want 0", m.Count())
	}
}

func TestMemoryChecker_ReloadSwaps(t *testing.T) {
	m := NewMemoryChecker()
	m.Reload([]string{"a.com"})
	if got, _ := m.IsBlocked("a.com"); !got {
		t.Error("a.com should be blocked after first reload")
	}
	// New set replaces the old one wholesale.
	m.Reload([]string{"b.com"})
	if got, _ := m.IsBlocked("a.com"); got {
		t.Error("a.com should no longer be blocked after reload")
	}
	if got, _ := m.IsBlocked("b.com"); !got {
		t.Error("b.com should be blocked after reload")
	}
	if m.Count() != 1 {
		t.Errorf("Count = %d, want 1", m.Count())
	}
}

// Reads must stay correct and race-free while a Reload runs concurrently.
func TestMemoryChecker_ConcurrentReadDuringReload(t *testing.T) {
	m := NewMemoryChecker()
	m.Reload([]string{"blocked.com"})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 2000; j++ {
				// Result may be either set's answer, but must never panic
				// or read a torn map.
				_, _ = m.IsBlocked("blocked.com")
				_, _ = m.IsBlocked("www.other.com")
			}
		}()
	}
	for i := 0; i < 50; i++ {
		m.Reload([]string{"blocked.com", "other.com"})
		m.Reload([]string{"blocked.com"})
	}
	wg.Wait()
}

// TestMemoryChecker_ReloadMemoryFootprint gives a rough (not exact:
// GC timing and map bucket growth make this noisy) estimate of bytes per
// entry for the blocklist design doc's "1GB Pi holds two full sets during
// a rebuild" concern. Run with -v to see the reported figure; it is not
// asserted against a hard threshold because GOGC/allocator behavior varies
// too much across Go versions/platforms for a tight bound to be reliable
// signal rather than flakiness.
//
// Measured in this environment (go1.24-ish, linux/amd64) with 300k
// ~35-byte synthetic domains: ~30 bytes/entry heap growth for the
// map[string]struct{} (this run's HeapAlloc delta ÷ entry count; see the
// t.Logf output; GC timing makes the exact number noisy run to run,
// roughly 20-40 bytes/entry observed). Extrapolated: a few million entries
// (a large aggregated blocklist set) is tens to ~100MB for one set,
// comfortably fine on a 1GB Pi for one set, but Reload briefly holds
// old+new simultaneously (the atomic swap only drops the old map after the
// new one is built), so peak usage during a rebuild is roughly double the
// steady-state figure. This is why Poll single-flights rebuilds
// (cmd/dataplane/blocklist_reload.go) instead of ever running two
// concurrently, which would otherwise make that peak worse under a change
// burst.
func TestMemoryChecker_ReloadMemoryFootprint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory footprint measurement in -short mode")
	}
	const n = 300_000
	domains := make([]string, n)
	for i := 0; i < n; i++ {
		domains[i] = fmt.Sprintf("synthetic-domain-%d.example.com", i)
	}

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	m := NewMemoryChecker()
	m.Reload(domains)

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	if m.Count() != n {
		t.Fatalf("Count() = %d, want %d", m.Count(), n)
	}

	// HeapAlloc can go down across a GC cycle on a quiet system; guard
	// against a negative/garbage bytes-per-entry figure rather than report
	// nonsense.
	if after.HeapAlloc > before.HeapAlloc {
		delta := after.HeapAlloc - before.HeapAlloc
		t.Logf("blocklist set: %d entries, ~%d bytes heap growth, ~%.1f bytes/entry",
			n, delta, float64(delta)/float64(n))
	} else {
		t.Logf("blocklist set: %d entries; heap shrank across GC (delta not meaningful on this run)", n)
	}
}
