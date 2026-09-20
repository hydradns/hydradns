// SPDX-License-Identifier: GPL-3.0-or-later
package dnsengine

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hydradns/hydradns/apps/core/internal/storage/models"
	"github.com/hydradns/hydradns/apps/core/internal/storage/repositories"
)

type fakeQueryLog struct {
	mu      sync.Mutex
	saved   []*models.DNSQuery
	batches int
}

func (f *fakeQueryLog) Save(q *models.DNSQuery) error { return nil }
func (f *fakeQueryLog) SaveBatch(qs []*models.DNSQuery) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saved = append(f.saved, qs...)
	f.batches++
	return nil
}
func (f *fakeQueryLog) ListRecent(limit int) ([]models.DNSQuery, error) { return nil, nil }
func (f *fakeQueryLog) DeleteOlderThan(time.Time) (int64, error)        { return 0, nil }
func (f *fakeQueryLog) EnforceRowCap(int64) (int64, error)              { return 0, nil }
func (f *fakeQueryLog) Count() (int64, error)                           { return int64(f.count()), nil }
func (f *fakeQueryLog) ListPage(repositories.QueryLogFilter) ([]models.DNSQuery, error) {
	return nil, nil
}
func (f *fakeQueryLog) CountFiltered(repositories.QueryLogFilter) (int64, error) { return 0, nil }
func (f *fakeQueryLog) BypassAttempts(time.Time, int) (repositories.BypassSummary, error) {
	return repositories.BypassSummary{}, nil
}
func (f *fakeQueryLog) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.saved)
}
func (f *fakeQueryLog) batchCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.batches
}

type fakeStats struct {
	allowed, blocked, redirected, other atomic.Int64
	calls                               atomic.Int64
}

func (s *fakeStats) Save(*models.Statistics) error               { return nil }
func (s *fakeStats) ListRecent(int) ([]models.Statistics, error) { return nil, nil }
func (s *fakeStats) IncrementCounter(string) error               { return nil }
func (s *fakeStats) SeedSingleton() error                        { return nil }
func (s *fakeStats) AddCounters(a, b, r, o int64) error {
	s.allowed.Add(a)
	s.blocked.Add(b)
	s.redirected.Add(r)
	s.other.Add(o)
	s.calls.Add(1)
	return nil
}

func TestWriterBatchesAndCounts(t *testing.T) {
	ql := &fakeQueryLog{}
	st := &fakeStats{}
	w := NewQueryLogWriter(ql, st)

	for i := 0; i < 300; i++ {
		w.Enqueue(&models.DNSQuery{Domain: "a.com", Action: "block"})
	}
	for i := 0; i < 100; i++ {
		w.Enqueue(&models.DNSQuery{Domain: "b.com", Action: "allow"})
	}
	for i := 0; i < 50; i++ {
		w.Enqueue(&models.DNSQuery{Domain: "c.com", Action: "flagged"}) // counts as allowed
	}
	for i := 0; i < 20; i++ {
		w.Enqueue(&models.DNSQuery{Domain: "d.com", Action: "error"}) // total only, not allowed
	}
	w.Shutdown() // drains remaining batch

	if got := ql.count(); got != 470 {
		t.Errorf("saved %d query logs, want 470", got)
	}
	if got := st.blocked.Load(); got != 300 {
		t.Errorf("blocked count = %d, want 300", got)
	}
	// 100 allow + 50 flagged = 150 allowed; "error" must NOT inflate this.
	if got := st.allowed.Load(); got != 150 {
		t.Errorf("allowed count = %d, want 150 (allow + flagged, excluding error)", got)
	}
	if got := st.other.Load(); got != 20 {
		t.Errorf("other count = %d, want 20 (error actions, total-only)", got)
	}
	// 470 entries should collapse into far fewer DB batches than 470.
	if b := ql.batchCount(); b == 0 || b > 10 {
		t.Errorf("batch count = %d, want a small number (batching not working?)", b)
	}
}

func TestWriterDropsWhenFull(t *testing.T) {
	// A writer whose DB is wedged can't drain; enqueue must not block and
	// must count drops once the bounded queue fills.
	block := make(chan struct{})
	ql := &blockingQueryLog{release: block}
	w := NewQueryLogWriter(ql, &fakeStats{})

	// Push well past queue capacity without blocking.
	done := make(chan struct{})
	go func() {
		for i := 0; i < logQueueSize*3; i++ {
			w.Enqueue(&models.DNSQuery{Domain: "x.com", Action: "allow"})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Enqueue blocked when queue was full — must be non-blocking")
	}
	if w.Dropped() == 0 {
		t.Error("expected some dropped entries when queue overflowed")
	}
	close(block) // let the writer drain so Shutdown can return
	w.Shutdown()
}

type blockingQueryLog struct {
	release chan struct{}
	once    sync.Once
}

func (b *blockingQueryLog) Save(*models.DNSQuery) error { return nil }
func (b *blockingQueryLog) SaveBatch([]*models.DNSQuery) error {
	// Block the first flush until released, simulating a stalled DB.
	b.once.Do(func() { <-b.release })
	return nil
}
func (b *blockingQueryLog) ListRecent(int) ([]models.DNSQuery, error) { return nil, nil }
func (b *blockingQueryLog) DeleteOlderThan(time.Time) (int64, error)  { return 0, nil }
func (b *blockingQueryLog) EnforceRowCap(int64) (int64, error)        { return 0, nil }
func (b *blockingQueryLog) Count() (int64, error)                     { return 0, nil }
func (b *blockingQueryLog) ListPage(repositories.QueryLogFilter) ([]models.DNSQuery, error) {
	return nil, nil
}
func (b *blockingQueryLog) CountFiltered(repositories.QueryLogFilter) (int64, error) { return 0, nil }
func (b *blockingQueryLog) BypassAttempts(time.Time, int) (repositories.BypassSummary, error) {
	return repositories.BypassSummary{}, nil
}
