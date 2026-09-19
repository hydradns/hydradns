// SPDX-License-Identifier: GPL-3.0-or-later
package dnsengine

import (
	"sync"
	"testing"
	"time"

	"github.com/hydradns/hydra-core/internal/storage/models"
	"github.com/hydradns/hydra-core/internal/threat"
	"github.com/hydradns/hydra-core/internal/utils"
)

// --- Fakes for the query-log path ---

type fakeQueryLogRepo struct {
	mu      sync.Mutex
	batches [][]*models.DNSQuery
}

func (f *fakeQueryLogRepo) Save(*models.DNSQuery) error { return nil }

func (f *fakeQueryLogRepo) SaveBatch(qs []*models.DNSQuery) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]*models.DNSQuery, len(qs))
	copy(cp, qs)
	f.batches = append(f.batches, cp)
	return nil
}

func (f *fakeQueryLogRepo) ListRecent(int) ([]models.DNSQuery, error) { return nil, nil }
func (f *fakeQueryLogRepo) DeleteOlderThan(time.Time) (int64, error)  { return 0, nil }
func (f *fakeQueryLogRepo) EnforceRowCap(int64) (int64, error)        { return 0, nil }
func (f *fakeQueryLogRepo) Count() (int64, error)                     { return 0, nil }

func (f *fakeQueryLogRepo) rows() []*models.DNSQuery {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*models.DNSQuery
	for _, b := range f.batches {
		out = append(out, b...)
	}
	return out
}

type fakeStatisticsRepo struct{}

func (fakeStatisticsRepo) Save(*models.Statistics) error               { return nil }
func (fakeStatisticsRepo) ListRecent(int) ([]models.Statistics, error) { return nil, nil }
func (fakeStatisticsRepo) IncrementCounter(string) error               { return nil }
func (fakeStatisticsRepo) AddCounters(a, b, c, o int64) error          { return nil }
func (fakeStatisticsRepo) SeedSingleton() error                        { return nil }

// engineWithLogWriter builds a minimal Engine with a real QueryLogWriter
// backed by fakeQueryLogRepo, so logQuery's full path (including the
// batched async writer) is exercised, not just the in-memory struct build.
func engineWithLogWriter(anonymize bool) (*Engine, *fakeQueryLogRepo) {
	repo := &fakeQueryLogRepo{}
	e := &Engine{
		logWriter:          NewQueryLogWriter(repo, fakeStatisticsRepo{}),
		anonymizeClientIPs: anonymize,
	}
	return e, repo
}

// --- Tests ---

func TestLogQuery_DisabledStoresRawClientIP(t *testing.T) {
	e, repo := engineWithLogWriter(false)

	const raw = "203.0.113.7:54321"
	e.logQuery("example.com", raw, "allow", threat.Result{})
	e.logWriter.Shutdown()

	rows := repo.rows()
	if len(rows) != 1 {
		t.Fatalf("expected 1 stored row, got %d", len(rows))
	}
	if rows[0].ClientIP != raw {
		t.Errorf("disabled anonymization must store the raw client IP byte-for-byte: got %q, want %q", rows[0].ClientIP, raw)
	}
}

func TestLogQuery_EnabledStoresAnonymizedClientIP(t *testing.T) {
	utils.InitSecret("test-engine-secret")
	defer func() { utils.InitSecret("") }()

	e, repo := engineWithLogWriter(true)

	const raw = "203.0.113.7:54321"
	e.logQuery("example.com", raw, "allow", threat.Result{})
	e.logWriter.Shutdown()

	rows := repo.rows()
	if len(rows) != 1 {
		t.Fatalf("expected 1 stored row, got %d", len(rows))
	}
	if rows[0].ClientIP == raw {
		t.Errorf("enabled anonymization must not store the raw client IP, got %q", rows[0].ClientIP)
	}
	if rows[0].ClientIP == "" {
		t.Errorf("expected a non-empty anonymized value")
	}
}

func TestLogQuery_EnabledIsStableForSameIPWithinOneInstall(t *testing.T) {
	utils.InitSecret("test-engine-secret")
	defer func() { utils.InitSecret("") }()

	e, repo := engineWithLogWriter(true)

	e.logQuery("a.com", "203.0.113.7:1111", "allow", threat.Result{})
	e.logQuery("b.com", "203.0.113.7:2222", "allow", threat.Result{})
	e.logWriter.Shutdown()

	rows := repo.rows()
	if len(rows) != 2 {
		t.Fatalf("expected 2 stored rows, got %d", len(rows))
	}
	if rows[0].ClientIP != rows[1].ClientIP {
		t.Errorf("same device (same IP, different port) across two queries must anonymize to the same value: %q != %q", rows[0].ClientIP, rows[1].ClientIP)
	}
}

func TestLogQuery_EnabledDiffersAcrossTwoSecrets(t *testing.T) {
	const raw = "203.0.113.7:54321"

	utils.InitSecret("secret-one")
	e1, repo1 := engineWithLogWriter(true)
	e1.logQuery("a.com", raw, "allow", threat.Result{})
	e1.logWriter.Shutdown()

	utils.InitSecret("secret-two")
	e2, repo2 := engineWithLogWriter(true)
	e2.logQuery("a.com", raw, "allow", threat.Result{})
	e2.logWriter.Shutdown()
	defer func() { utils.InitSecret("") }()

	got1, got2 := repo1.rows()[0].ClientIP, repo2.rows()[0].ClientIP
	if got1 == got2 {
		t.Errorf("two different install secrets must not anonymize the same IP to the same value, both were %q", got1)
	}
}

func TestLogQuery_EnabledIPv6Works(t *testing.T) {
	utils.InitSecret("test-engine-secret")
	defer func() { utils.InitSecret("") }()

	e, repo := engineWithLogWriter(true)
	e.logQuery("a.com", "[2001:db8::1]:5353", "allow", threat.Result{})
	e.logWriter.Shutdown()

	rows := repo.rows()
	if len(rows) != 1 {
		t.Fatalf("expected 1 stored row, got %d", len(rows))
	}
	if rows[0].ClientIP == "" || rows[0].ClientIP == "[2001:db8::1]:5353" {
		t.Errorf("expected an anonymized IPv6 value, got %q", rows[0].ClientIP)
	}
}

// TestLogQuery_EnabledWithoutInitSecretDoesNotPanic covers the case where
// the engine is constructed (or a test runs) without ever calling
// utils.InitSecret. Behaviour must be defined: AnonymizeIP's no-secret
// fallback (coarse masking, no hash) applies — never a panic, and never
// the raw untouched clientIP.
func TestLogQuery_EnabledWithoutInitSecretDoesNotPanic(t *testing.T) {
	utils.InitSecret("") // explicitly simulate "never initialized"
	defer func() { utils.InitSecret("") }()

	e, repo := engineWithLogWriter(true)

	const raw = "203.0.113.7:54321"
	e.logQuery("example.com", raw, "allow", threat.Result{})
	e.logWriter.Shutdown()

	rows := repo.rows()
	if len(rows) != 1 {
		t.Fatalf("expected 1 stored row, got %d", len(rows))
	}
	if rows[0].ClientIP == raw {
		t.Errorf("expected the no-secret fallback (masked IP) rather than the raw client IP, got %q", rows[0].ClientIP)
	}
}

// --- anonymizeClientIP unit tests ---

func TestAnonymizeClientIP_StripsPortIPv4(t *testing.T) {
	utils.InitSecret("test-secret")
	defer func() { utils.InitSecret("") }()

	withPort := anonymizeClientIP("203.0.113.7:54321")
	bare := utils.AnonymizeIP("203.0.113.7")
	if withPort == "" {
		t.Fatalf("expected a non-empty anonymized value")
	}
	if withPort != bare {
		t.Errorf("anonymizeClientIP(%q) = %q, want it to match AnonymizeIP of the bare host %q", "203.0.113.7:54321", withPort, bare)
	}
}

func TestAnonymizeClientIP_StripsPortIPv6(t *testing.T) {
	utils.InitSecret("test-secret")
	defer func() { utils.InitSecret("") }()

	withPort := anonymizeClientIP("[2001:db8::1]:5353")
	bare := utils.AnonymizeIP("2001:db8::1")
	if withPort == "" {
		t.Fatalf("expected a non-empty anonymized value")
	}
	if withPort != bare {
		t.Errorf("anonymizeClientIP(%q) = %q, want it to match AnonymizeIP of the bare host %q", "[2001:db8::1]:5353", withPort, bare)
	}
}

func TestAnonymizeClientIP_PlainIPWithoutPort(t *testing.T) {
	utils.InitSecret("test-secret")
	defer func() { utils.InitSecret("") }()

	got := anonymizeClientIP("203.0.113.7")
	want := utils.AnonymizeIP("203.0.113.7")
	if got != want {
		t.Errorf("anonymizeClientIP(bare ip) = %q, want %q", got, want)
	}
}
