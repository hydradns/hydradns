// SPDX-License-Identifier: Apache-2.0
package dnsengine

import (
	"container/list"
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

const (
	defaultCacheMaxEntries = 20000
	// minCacheTTL guards against upstreams handing out 0/1s TTLs that
	// would make the cache useless; maxCacheTTL bounds staleness if an
	// upstream hands out a week-long TTL. Trade-off: a record whose real
	// TTL is below the floor can be served (with TTL aged down to 1) for
	// up to minCacheTTL after it expired.
	minCacheTTL = 5 * time.Second
	maxCacheTTL = 1 * time.Hour
	// Negative answers (RFC 2308): lifetime comes from the SOA in the
	// Authority section when present, clamped to maxNegativeTTL;
	// negativeCacheTTL is the fallback for NXDOMAIN without a SOA.
	negativeCacheTTL = 30 * time.Second
	maxNegativeTTL   = 5 * time.Minute
)

type cacheKey struct {
	name  string // lowercased FQDN with trailing dot
	qtype uint16
	class uint16
}

type cacheEntry struct {
	key      cacheKey
	msg      *dns.Msg
	storedAt time.Time
	expires  time.Time
}

// ResponseCache is a bounded LRU cache of upstream DNS responses keyed by
// (qname, qtype, qclass). Entries expire at the answer's minimum TTL,
// clamped to [minCacheTTL, maxCacheTTL]. Served responses get their TTLs
// decremented by time spent in cache.
type ResponseCache struct {
	mu      sync.Mutex
	entries map[cacheKey]*list.Element
	lru     *list.List // front = most recently used
	max     int

	hits   atomic.Uint64
	misses atomic.Uint64
}

func NewResponseCache(maxEntries int) *ResponseCache {
	if maxEntries < 1 {
		maxEntries = defaultCacheMaxEntries
	}
	return &ResponseCache{
		entries: make(map[cacheKey]*list.Element),
		lru:     list.New(),
		max:     maxEntries,
	}
}

// cacheKeyFor returns the cache key for a query, or ok=false if the query
// is not cacheable (no question or multiple questions).
func cacheKeyFor(q *dns.Msg) (cacheKey, bool) {
	if q == nil || len(q.Question) != 1 {
		return cacheKey{}, false
	}
	question := q.Question[0]
	return cacheKey{
		name:  dns.CanonicalName(question.Name),
		qtype: question.Qtype,
		class: question.Qclass,
	}, true
}

// Get returns a response ready to send for the given query, or nil on miss.
// The returned message is a copy with the query's ID and Question section
// and with TTLs reduced by the entry's age.
func (c *ResponseCache) Get(q *dns.Msg) *dns.Msg {
	key, ok := cacheKeyFor(q)
	if !ok {
		return nil
	}

	c.mu.Lock()
	elem, found := c.entries[key]
	if !found {
		c.mu.Unlock()
		c.misses.Add(1)
		return nil
	}
	entry := elem.Value.(*cacheEntry)
	if time.Now().After(entry.expires) {
		c.lru.Remove(elem)
		delete(c.entries, key)
		c.mu.Unlock()
		c.misses.Add(1)
		return nil
	}
	c.lru.MoveToFront(elem)
	resp := entry.msg.Copy()
	age := time.Since(entry.storedAt)
	c.mu.Unlock()

	c.hits.Add(1)
	resp.Id = q.Id
	// Echo the client's Question verbatim so case-randomizing resolvers
	// (DNS 0x20) accept the answer.
	resp.Question = append([]dns.Question(nil), q.Question...)
	ageTTLs(resp, age)
	return resp
}

// Set stores an upstream response. Truncated responses and rcodes other
// than NOERROR/NXDOMAIN are not cached. Empty NOERROR responses are only
// cached when the Authority section carries a SOA (a genuine NODATA answer
// per RFC 2308); an empty answer without one may be a referral.
func (c *ResponseCache) Set(q, resp *dns.Msg) {
	key, ok := cacheKeyFor(q)
	if !ok || resp == nil || resp.Truncated {
		return
	}
	if resp.Rcode != dns.RcodeSuccess && resp.Rcode != dns.RcodeNameError {
		return
	}
	if resp.Rcode == dns.RcodeSuccess && len(resp.Answer) == 0 && findSOA(resp) == nil {
		return
	}

	stored := resp.Copy()
	stripOPT(stored)

	ttl := cacheTTL(stored)
	now := time.Now()
	entry := &cacheEntry{
		key:      key,
		msg:      stored,
		storedAt: now,
		expires:  now.Add(ttl),
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if elem, found := c.entries[key]; found {
		elem.Value = entry
		c.lru.MoveToFront(elem)
		return
	}
	c.entries[key] = c.lru.PushFront(entry)
	for len(c.entries) > c.max {
		oldest := c.lru.Back()
		if oldest == nil {
			break
		}
		c.lru.Remove(oldest)
		delete(c.entries, oldest.Value.(*cacheEntry).key)
	}
}

// Stats returns cumulative hit/miss counts and the current entry count.
func (c *ResponseCache) Stats() (hits, misses uint64, size int) {
	c.mu.Lock()
	size = len(c.entries)
	c.mu.Unlock()
	return c.hits.Load(), c.misses.Load(), size
}

// findSOA returns the SOA record from the Authority section, or nil.
func findSOA(resp *dns.Msg) *dns.SOA {
	for _, rr := range resp.Ns {
		if soa, ok := rr.(*dns.SOA); ok {
			return soa
		}
	}
	return nil
}

// cacheTTL derives the cache lifetime from the response: the minimum TTL
// across answer records, clamped. Negative answers use the RFC 2308 value
// min(SOA TTL, SOA MINIMUM) when a SOA is present, negativeCacheTTL otherwise.
func cacheTTL(resp *dns.Msg) time.Duration {
	if len(resp.Answer) == 0 {
		if soa := findSOA(resp); soa != nil {
			secs := soa.Minttl
			if soa.Hdr.Ttl < secs {
				secs = soa.Hdr.Ttl
			}
			ttl := time.Duration(secs) * time.Second
			if ttl < minCacheTTL {
				return minCacheTTL
			}
			if ttl > maxNegativeTTL {
				return maxNegativeTTL
			}
			return ttl
		}
		return negativeCacheTTL
	}
	min := resp.Answer[0].Header().Ttl
	for _, rr := range resp.Answer[1:] {
		if t := rr.Header().Ttl; t < min {
			min = t
		}
	}
	ttl := time.Duration(min) * time.Second
	if ttl < minCacheTTL {
		return minCacheTTL
	}
	if ttl > maxCacheTTL {
		return maxCacheTTL
	}
	return ttl
}

// ageTTLs reduces every record's TTL by the entry's age, flooring at 1s.
func ageTTLs(m *dns.Msg, age time.Duration) {
	dec := uint32(age / time.Second)
	if dec == 0 {
		return
	}
	for _, section := range [][]dns.RR{m.Answer, m.Ns, m.Extra} {
		for _, rr := range section {
			h := rr.Header()
			if h.Ttl > dec {
				h.Ttl -= dec
			} else {
				h.Ttl = 1
			}
		}
	}
}

// stripOPT removes EDNS0 OPT pseudo-records so a cached response is not
// replayed with the original client's EDNS parameters.
func stripOPT(m *dns.Msg) {
	if len(m.Extra) == 0 {
		return
	}
	extra := m.Extra[:0]
	for _, rr := range m.Extra {
		if rr.Header().Rrtype != dns.TypeOPT {
			extra = append(extra, rr)
		}
	}
	m.Extra = extra
}
