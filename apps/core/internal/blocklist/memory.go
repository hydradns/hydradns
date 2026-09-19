// SPDX-License-Identifier: GPL-3.0-or-later
package blocklist

import (
	"strings"
	"sync/atomic"
)

// MemoryChecker is an in-memory blocklist membership test. It holds the
// full set of blocked domains in a map and answers IsBlocked without
// touching the database, so the DNS hot path never does per-query SQL.
//
// The domain set is swapped atomically on Reload (called after each
// periodic blocklist refresh), so reads are lock-free and never block on
// an update. This satisfies dnsengine.BlocklistChecker structurally.
type MemoryChecker struct {
	// domains holds *map[string]struct{}; swapped wholesale on Reload.
	domains atomic.Pointer[map[string]struct{}]
}

// NewMemoryChecker returns a checker with an empty set. Until the first
// Reload completes, IsBlocked returns false (fail-open), so DNS can serve
// immediately on startup while the blocklist loads in the background.
func NewMemoryChecker() *MemoryChecker {
	m := &MemoryChecker{}
	empty := make(map[string]struct{})
	m.domains.Store(&empty)
	return m
}

// Reload replaces the blocked-domain set atomically. Domains are
// normalized (lowercased, trailing dot stripped) to match query-time
// normalization.
func (m *MemoryChecker) Reload(domains []string) {
	set := make(map[string]struct{}, len(domains))
	for _, d := range domains {
		set[normalize(d)] = struct{}{}
	}
	m.domains.Store(&set)
}

// Count returns the number of blocked domains currently loaded.
func (m *MemoryChecker) Count() int {
	return len(*m.domains.Load())
}

// IsBlocked reports whether the domain or any of its parent domains is in
// the set. e.g. www.ads.example.com matches a block on ads.example.com or
// example.com. The public suffix itself (the last label) is never treated
// as a match, mirroring the original DB query's candidate set.
func (m *MemoryChecker) IsBlocked(domain string) (bool, error) {
	d := normalize(domain)
	// Single-label inputs (no dot) are never blocked. This matches the
	// original DB query, whose candidate set was empty for single-label
	// domains, and avoids blocking bare names like "localhost" /
	// "broadcasthost" that appear in standard hosts blocklists.
	if strings.IndexByte(d, '.') < 0 {
		return false, nil
	}
	set := *m.domains.Load()
	if len(set) == 0 {
		return false, nil
	}

	// Walk from the full domain up to (but not including) the last label.
	for {
		if _, ok := set[d]; ok {
			return true, nil
		}
		i := strings.IndexByte(d, '.')
		if i < 0 {
			return false, nil
		}
		rest := d[i+1:]
		// Stop once only the final label remains (no more dots): we do
		// not block on a bare TLD.
		if strings.IndexByte(rest, '.') < 0 {
			return false, nil
		}
		d = rest
	}
}

func normalize(d string) string {
	return strings.TrimSuffix(strings.ToLower(d), ".")
}
