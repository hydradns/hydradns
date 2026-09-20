// Handles upstream DNS resolvers with connection pooling, retry, and failover.
// SPDX-License-Identifier: GPL-3.0-or-later
package dnsengine

import (
	"time"

	"github.com/hydradns/hydradns/apps/core/internal/logger"
	"github.com/miekg/dns"
)

type UpstreamManager struct {
	pools []*UpstreamPool
}

// NewUpstreamManager builds a pool for each configured resolver
func NewUpstreamManager(resolvers []string, poolSize int) (*UpstreamManager, error) {
	manager := &UpstreamManager{}
	for _, addr := range resolvers {
		pool, err := NewUpstreamPool(addr, poolSize)
		if err != nil {
			return nil, err
		}
		manager.pools = append(manager.pools, pool)
	}
	return manager, nil
}

func (m *UpstreamManager) Close() {
	for _, pool := range m.pools {
		pool.Close()
	}
}

// Exchange forwards query to resolvers with retry+failover.
//
// Each attempt goes out with a fresh random ID rather than the client's:
// the client-chosen ID is attacker-predictable, and reusing one ID across
// attempts lets a late answer to a timed-out attempt satisfy the next one.
// The original ID is restored on the query and stamped on the response so
// the client sees its own ID.
func (m *UpstreamManager) Exchange(q *dns.Msg, timeout time.Duration, maxRetries int) (*dns.Msg, error) {
	origID := q.Id
	defer func() { q.Id = origID }()

	var lastErr error
	for _, pool := range m.pools {
		for attempt := 0; attempt < maxRetries; attempt++ {
			q.Id = dns.Id()
			resp, err := pool.Exchange(q, timeout)
			if err == nil {
				resp.Id = origID
				return resp, nil
			}
			lastErr = err
			logger.Log.Warnf("upstream %s failed (attempt %d): %v", pool.upstreamAddr, attempt+1, err)
		}
	}
	return nil, lastErr
}
