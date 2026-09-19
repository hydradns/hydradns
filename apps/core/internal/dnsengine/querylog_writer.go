// SPDX-License-Identifier: GPL-3.0-or-later
package dnsengine

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/lopster568/phantomDNS/internal/logger"
	"github.com/lopster568/phantomDNS/internal/storage/models"
	"github.com/lopster568/phantomDNS/internal/storage/repositories"
)

const (
	// logQueueSize bounds memory: at most this many query logs are held
	// in flight. Sized so a burst is absorbed but the queue can never grow
	// without limit (the old goroutine-per-query path could).
	logQueueSize = 4096
	// logBatchSize and logFlushInterval control how the writer coalesces
	// queued logs into DB writes: flush when either is reached.
	logBatchSize     = 256
	logFlushInterval = 500 * time.Millisecond
)

// QueryLogWriter drains DNS query logs off a bounded channel and writes
// them to the DB in batches via a single goroutine. This keeps the DNS
// hot path free of database work and bounds memory under load: enqueue is
// non-blocking and drops (counted) when the queue is full, so logging can
// never stall query resolution or OOM the box.
type QueryLogWriter struct {
	ch         chan *models.DNSQuery
	queryLog   repositories.QueryLogRepository
	statistics repositories.StatisticsRepository

	dropped atomic.Uint64
	// lastDroppedLog is read/written only by the writer goroutine (in
	// flush), so it needs no synchronization.
	lastDroppedLog uint64

	wg   sync.WaitGroup
	stop chan struct{}
	once sync.Once
}

func NewQueryLogWriter(ql repositories.QueryLogRepository, stats repositories.StatisticsRepository) *QueryLogWriter {
	w := &QueryLogWriter{
		ch:         make(chan *models.DNSQuery, logQueueSize),
		queryLog:   ql,
		statistics: stats,
		stop:       make(chan struct{}),
	}
	w.wg.Add(1)
	go w.run()
	return w
}

// Enqueue submits a query log without blocking. If the queue is full
// (writer can't keep up), the entry is dropped and counted — resolution
// is never delayed by logging.
func (w *QueryLogWriter) Enqueue(q *models.DNSQuery) {
	if w == nil || q == nil {
		return
	}
	select {
	case w.ch <- q:
	default:
		w.dropped.Add(1)
	}
}

// Dropped returns the cumulative count of log entries dropped due to a
// full queue. The writer also logs a warning when this advances (see
// flush), so sustained drops are visible without polling this.
func (w *QueryLogWriter) Dropped() uint64 {
	if w == nil {
		return 0
	}
	return w.dropped.Load()
}

func (w *QueryLogWriter) run() {
	defer w.wg.Done()
	ticker := time.NewTicker(logFlushInterval)
	defer ticker.Stop()

	batch := make([]*models.DNSQuery, 0, logBatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		w.flush(batch)
		batch = batch[:0]
	}

	for {
		select {
		case q := <-w.ch:
			batch = append(batch, q)
			if len(batch) >= logBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-w.stop:
			// Drain whatever is already queued, then exit.
			for {
				select {
				case q := <-w.ch:
					batch = append(batch, q)
					if len(batch) >= logBatchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		}
	}
}

func (w *QueryLogWriter) flush(batch []*models.DNSQuery) {
	if err := w.queryLog.SaveBatch(batch); err != nil {
		logger.Log.Errorf("Failed to write query log batch (%d entries): %v", len(batch), err)
	}

	var allowed, blocked, redirected, other int64
	for _, q := range batch {
		switch q.Action {
		case "block":
			blocked++
		case "redirect":
			redirected++
		case "allow", "flagged": // flagged is allowed-but-suspicious, still forwarded
			allowed++
		default: // "error" and anything unexpected: count toward total only
			other++
		}
	}
	if w.statistics != nil {
		if err := w.statistics.AddCounters(allowed, blocked, redirected, other); err != nil {
			logger.Log.Errorf("Failed to apply batched statistics: %v", err)
		}
	}

	// Surface dropped logs (full queue = sustained overload) at most once
	// per flush, so the soak test and ops can see backpressure happening.
	if d := w.dropped.Load(); d > w.lastDroppedLog {
		logger.Log.Warnf("query log writer dropped %d entries under load (queue full)", d-w.lastDroppedLog)
		w.lastDroppedLog = d
	}
}

// Shutdown stops the writer after draining queued entries. Idempotent.
func (w *QueryLogWriter) Shutdown() {
	if w == nil {
		return
	}
	w.once.Do(func() { close(w.stop) })
	w.wg.Wait()
}
