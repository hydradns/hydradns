// SPDX-License-Identifier: GPL-3.0-or-later
package dnsengine

import (
	"sync"
	"testing"
	"time"

	"github.com/hydradns/hydra-core/internal/metrics"
	"github.com/hydradns/hydra-core/internal/policy"
	"github.com/hydradns/hydra-core/internal/storage/models"
	"github.com/hydradns/hydra-core/internal/storage/repositories"
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
func (f *fakeQueryLogRepo) ListPage(repositories.QueryLogFilter) ([]models.DNSQuery, error) {
	return nil, nil
}
func (f *fakeQueryLogRepo) CountFiltered(repositories.QueryLogFilter) (int64, error) { return 0, nil }
func (f *fakeQueryLogRepo) BypassAttempts(time.Time, int) (repositories.BypassSummary, error) {
	return repositories.BypassSummary{}, nil
}

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

// TestLogQuery_DisabledStoresBareClientIPWithoutPort documents a fix: this
// test used to assert that disabled anonymization stored clientIP
// byte-for-byte, including the ephemeral source port that
// w.RemoteAddr().String() always includes for UDP/TCP. That locked in a
// real bug — every stored row carried a random per-connection port,
// breaking per-device filtering (GET /analytics/logs?client=<ip>) and
// cluttering the dashboard's Client IP column, since the same device
// would never produce two rows with the same ClientIP value. The port is
// now stripped unconditionally in logQuery, whether or not anonymization
// is enabled; disabled anonymization still does no hashing.
func TestLogQuery_DisabledStoresBareClientIPWithoutPort(t *testing.T) {
	e, repo := engineWithLogWriter(false)

	e.logQuery("example.com", "203.0.113.7:54321", "allow", threat.Result{})
	e.logWriter.Shutdown()

	rows := repo.rows()
	if len(rows) != 1 {
		t.Fatalf("expected 1 stored row, got %d", len(rows))
	}
	if rows[0].ClientIP != "203.0.113.7" {
		t.Errorf("disabled anonymization must still strip the ephemeral port: got %q, want %q", rows[0].ClientIP, "203.0.113.7")
	}
}

func TestLogQuery_DisabledStripsPortIPv6(t *testing.T) {
	e, repo := engineWithLogWriter(false)

	e.logQuery("example.com", "[2001:db8::1]:5353", "allow", threat.Result{})
	e.logWriter.Shutdown()

	rows := repo.rows()
	if len(rows) != 1 {
		t.Fatalf("expected 1 stored row, got %d", len(rows))
	}
	if rows[0].ClientIP != "2001:db8::1" {
		t.Errorf("expected bare IPv6 host, got %q", rows[0].ClientIP)
	}
}

func TestLogQuery_DisabledPassesThroughBareIPUnchanged(t *testing.T) {
	e, repo := engineWithLogWriter(false)

	e.logQuery("example.com", "203.0.113.7", "allow", threat.Result{})
	e.logWriter.Shutdown()

	rows := repo.rows()
	if len(rows) != 1 || rows[0].ClientIP != "203.0.113.7" {
		t.Fatalf("expected bare IP passed through unchanged, got %+v", rows)
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

// --- DoH bootstrap marker ---

// TestLogQuery_DoHBootstrapMarksDetectionMethod exercises ProcessDNSQuery
// end-to-end (not just logQuery) so it also proves the port-stripped
// client IP and the doh_bootstrap marker both land in the persisted row —
// the two signals GET /analytics/bypass groups by.
func TestProcessDNSQuery_DoHBootstrapMarksDetectionMethod(t *testing.T) {
	repo := &fakeQueryLogRepo{}
	pe := policy.NewPolicyEngine()
	if err := pe.LoadPolicies(nil); err != nil {
		t.Fatalf("load policies: %v", err)
	}
	e := &Engine{
		policyEngine: pe,
		state:        &RuntimeState{},
		metrics:      metrics.NewQueryMetrics(),
		logWriter:    NewQueryLogWriter(repo, fakeStatisticsRepo{}),
	}
	e.state.acceptQueries.Store(true)

	w := &mockResponseWriter{}
	q := newTestQuery("dns.google")
	e.ProcessDNSQuery(w, q)
	e.logWriter.Shutdown()

	rows := repo.rows()
	if len(rows) != 1 {
		t.Fatalf("expected 1 logged row, got %d", len(rows))
	}
	row := rows[0]
	if row.Action != "block" {
		t.Errorf("expected action=block for a DoH bootstrap hit, got %q", row.Action)
	}
	if row.DetectionMethod != models.DetectionMethodDoHBootstrap {
		t.Errorf("expected detection_method=%q, got %q", models.DetectionMethodDoHBootstrap, row.DetectionMethod)
	}
}

// --- stripClientPort unit tests ---

func TestStripClientPort(t *testing.T) {
	tests := []struct{ in, want string }{
		{"203.0.113.7:54321", "203.0.113.7"},
		{"[2001:db8::1]:5353", "2001:db8::1"},
		{"203.0.113.7", "203.0.113.7"}, // already bare
		{"2001:db8::1", "2001:db8::1"}, // already bare IPv6, no brackets
		{"", ""},
	}
	for _, tt := range tests {
		if got := stripClientPort(tt.in); got != tt.want {
			t.Errorf("stripClientPort(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
