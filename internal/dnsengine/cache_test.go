// SPDX-License-Identifier: GPL-3.0-or-later
package dnsengine

import (
	"testing"
	"time"

	"github.com/miekg/dns"
)

func newQuery(name string, qtype uint16) *dns.Msg {
	q := new(dns.Msg)
	q.SetQuestion(dns.Fqdn(name), qtype)
	q.Id = dns.Id()
	return q
}

func newAnswer(q *dns.Msg, ttl uint32, ip string) *dns.Msg {
	resp := new(dns.Msg)
	resp.SetReply(q)
	rr, err := dns.NewRR(q.Question[0].Name + " " +
		"IN A " + ip)
	if err != nil {
		panic(err)
	}
	rr.Header().Ttl = ttl
	resp.Answer = append(resp.Answer, rr)
	return resp
}

func TestCacheHitReturnsCopyWithQueryID(t *testing.T) {
	c := NewResponseCache(10)
	q1 := newQuery("example.com", dns.TypeA)
	c.Set(q1, newAnswer(q1, 300, "93.184.216.34"))

	q2 := newQuery("example.com", dns.TypeA)
	q2.Id = 0xBEEF
	got := c.Get(q2)
	if got == nil {
		t.Fatal("expected cache hit")
	}
	if got.Id != 0xBEEF {
		t.Errorf("response ID = %#x, want query ID %#x", got.Id, 0xBEEF)
	}
	if len(got.Answer) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(got.Answer))
	}

	// Mutating the returned message must not corrupt the cached copy.
	got.Answer[0].Header().Ttl = 0
	again := c.Get(newQuery("example.com", dns.TypeA))
	if again == nil || again.Answer[0].Header().Ttl == 0 {
		t.Error("cached entry was mutated by caller")
	}
}

func TestCacheMissOnDifferentTypeAndName(t *testing.T) {
	c := NewResponseCache(10)
	q := newQuery("example.com", dns.TypeA)
	c.Set(q, newAnswer(q, 300, "1.2.3.4"))

	if c.Get(newQuery("example.com", dns.TypeAAAA)) != nil {
		t.Error("AAAA query must not hit A cache entry")
	}
	if c.Get(newQuery("other.com", dns.TypeA)) != nil {
		t.Error("different name must miss")
	}
}

func TestCacheCaseInsensitiveKey(t *testing.T) {
	c := NewResponseCache(10)
	q := newQuery("Example.COM", dns.TypeA)
	c.Set(q, newAnswer(q, 300, "1.2.3.4"))

	mixed := newQuery("eXaMpLe.CoM", dns.TypeA)
	got := c.Get(mixed)
	if got == nil {
		t.Fatal("expected case-insensitive hit")
	}
	// DNS 0x20: the answer must echo the client's question casing.
	if got.Question[0].Name != mixed.Question[0].Name {
		t.Errorf("question = %q, want client casing %q", got.Question[0].Name, mixed.Question[0].Name)
	}
}

func TestCacheExpiry(t *testing.T) {
	c := NewResponseCache(10)
	q := newQuery("short.com", dns.TypeA)
	c.Set(q, newAnswer(q, 300, "1.2.3.4"))

	// Force the entry into the past.
	c.mu.Lock()
	for _, elem := range c.entries {
		elem.Value.(*cacheEntry).expires = time.Now().Add(-time.Second)
	}
	c.mu.Unlock()

	if c.Get(newQuery("short.com", dns.TypeA)) != nil {
		t.Error("expired entry must miss")
	}
	if _, _, size := c.Stats(); size != 0 {
		t.Errorf("expired entry should be evicted, size = %d", size)
	}
}

func TestCacheTTLAging(t *testing.T) {
	c := NewResponseCache(10)
	q := newQuery("aged.com", dns.TypeA)
	c.Set(q, newAnswer(q, 300, "1.2.3.4"))

	c.mu.Lock()
	for _, elem := range c.entries {
		elem.Value.(*cacheEntry).storedAt = time.Now().Add(-100 * time.Second)
	}
	c.mu.Unlock()

	got := c.Get(newQuery("aged.com", dns.TypeA))
	if got == nil {
		t.Fatal("expected hit")
	}
	ttl := got.Answer[0].Header().Ttl
	if ttl > 200 || ttl < 195 {
		t.Errorf("TTL = %d, want ~200 (300 - 100s age)", ttl)
	}
}

func TestCacheDoesNotStoreFailuresOrTruncated(t *testing.T) {
	c := NewResponseCache(10)

	q := newQuery("servfail.com", dns.TypeA)
	resp := new(dns.Msg)
	resp.SetRcode(q, dns.RcodeServerFailure)
	c.Set(q, resp)
	if c.Get(newQuery("servfail.com", dns.TypeA)) != nil {
		t.Error("SERVFAIL must not be cached")
	}

	q2 := newQuery("trunc.com", dns.TypeA)
	tr := newAnswer(q2, 300, "1.2.3.4")
	tr.Truncated = true
	c.Set(q2, tr)
	if c.Get(newQuery("trunc.com", dns.TypeA)) != nil {
		t.Error("truncated response must not be cached")
	}
}

func TestCacheNegativeNXDOMAIN(t *testing.T) {
	c := NewResponseCache(10)
	q := newQuery("nxdomain.com", dns.TypeA)
	resp := new(dns.Msg)
	resp.SetRcode(q, dns.RcodeNameError)
	c.Set(q, resp)

	got := c.Get(newQuery("nxdomain.com", dns.TypeA))
	if got == nil {
		t.Fatal("NXDOMAIN should be negatively cached")
	}
	if got.Rcode != dns.RcodeNameError {
		t.Errorf("rcode = %d, want NXDOMAIN", got.Rcode)
	}
}

func TestCacheEvictionAtCapacity(t *testing.T) {
	c := NewResponseCache(3)
	names := []string{"a.com", "b.com", "c.com", "d.com"}
	for _, n := range names {
		q := newQuery(n, dns.TypeA)
		c.Set(q, newAnswer(q, 300, "1.2.3.4"))
	}

	if _, _, size := c.Stats(); size != 3 {
		t.Errorf("size = %d, want 3 (capacity)", size)
	}
	if c.Get(newQuery("a.com", dns.TypeA)) != nil {
		t.Error("oldest entry (a.com) should have been evicted")
	}
	if c.Get(newQuery("d.com", dns.TypeA)) == nil {
		t.Error("newest entry (d.com) should be present")
	}
}

func TestCacheStripsOPT(t *testing.T) {
	c := NewResponseCache(10)
	q := newQuery("edns.com", dns.TypeA)
	resp := newAnswer(q, 300, "1.2.3.4")
	opt := new(dns.OPT)
	opt.Hdr.Name = "."
	opt.Hdr.Rrtype = dns.TypeOPT
	resp.Extra = append(resp.Extra, opt)
	c.Set(q, resp)

	got := c.Get(newQuery("edns.com", dns.TypeA))
	if got == nil {
		t.Fatal("expected hit")
	}
	for _, rr := range got.Extra {
		if rr.Header().Rrtype == dns.TypeOPT {
			t.Error("cached response must not carry the original OPT record")
		}
	}
}

func TestCacheNODATARequiresSOA(t *testing.T) {
	c := NewResponseCache(10)

	// Empty NOERROR without SOA (looks like a referral) must not be cached.
	q := newQuery("referral.com", dns.TypeA)
	resp := new(dns.Msg)
	resp.SetReply(q)
	c.Set(q, resp)
	if c.Get(newQuery("referral.com", dns.TypeA)) != nil {
		t.Error("empty NOERROR without SOA must not be cached")
	}

	// Empty NOERROR with SOA is genuine NODATA and is cacheable.
	q2 := newQuery("nodata.com", dns.TypeA)
	resp2 := new(dns.Msg)
	resp2.SetReply(q2)
	soa, err := dns.NewRR("nodata.com. 300 IN SOA ns1.nodata.com. admin.nodata.com. 1 7200 3600 1209600 60")
	if err != nil {
		t.Fatal(err)
	}
	resp2.Ns = append(resp2.Ns, soa)
	c.Set(q2, resp2)
	if c.Get(newQuery("nodata.com", dns.TypeA)) == nil {
		t.Error("NODATA with SOA should be cached")
	}
}

func TestNegativeTTLFromSOA(t *testing.T) {
	q := newQuery("neg.com", dns.TypeA)
	resp := new(dns.Msg)
	resp.SetRcode(q, dns.RcodeNameError)
	soa, err := dns.NewRR("neg.com. 300 IN SOA ns1.neg.com. admin.neg.com. 1 7200 3600 1209600 60")
	if err != nil {
		t.Fatal(err)
	}
	resp.Ns = append(resp.Ns, soa)

	// RFC 2308: min(SOA TTL 300, SOA MINIMUM 60) = 60s.
	if got := cacheTTL(resp); got != 60*time.Second {
		t.Errorf("negative TTL = %v, want 60s from SOA minimum", got)
	}
}

func TestQuestionMatches(t *testing.T) {
	q := newQuery("example.com", dns.TypeA)

	good := new(dns.Msg)
	good.SetReply(q)
	good.Question[0].Name = "EXAMPLE.com." // 0x20 casing must still match
	if !questionMatches(q, good) {
		t.Error("case-insensitive question should match")
	}

	wrongName := new(dns.Msg)
	wrongName.SetReply(q)
	wrongName.Question[0].Name = "evil.com."
	if questionMatches(q, wrongName) {
		t.Error("different name must not match")
	}

	wrongType := new(dns.Msg)
	wrongType.SetReply(q)
	wrongType.Question[0].Qtype = dns.TypeAAAA
	if questionMatches(q, wrongType) {
		t.Error("different qtype must not match")
	}

	empty := new(dns.Msg)
	empty.SetReply(q)
	empty.Question = nil
	if questionMatches(q, empty) {
		t.Error("missing question section must not match")
	}
}

func TestCacheTTLClamping(t *testing.T) {
	if got := cacheTTL(newAnswer(newQuery("x.com", dns.TypeA), 1, "1.2.3.4")); got != minCacheTTL {
		t.Errorf("1s TTL should clamp to %v, got %v", minCacheTTL, got)
	}
	if got := cacheTTL(newAnswer(newQuery("x.com", dns.TypeA), 86400, "1.2.3.4")); got != maxCacheTTL {
		t.Errorf("1d TTL should clamp to %v, got %v", maxCacheTTL, got)
	}
}
