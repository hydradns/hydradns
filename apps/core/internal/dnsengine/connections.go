// SPDX-License-Identifier: Apache-2.0
package dnsengine

import (
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hydradns/hydradns/apps/core/internal/logger"
	"github.com/miekg/dns"
)

const (
	defaultDialTimeout  = 5 * time.Second
	defaultQueryTimeout = 5 * time.Second
	defaultKeepAlive    = 30 * time.Second
	maxRetries          = 3
	// maxStaleReads caps how many mismatched datagrams one exchange will
	// discard before giving up, so the shared-socket mutex can't be pinned
	// by a flood.
	maxStaleReads = 16
)

// UDPClient is a small wrapper around a reusable UDP socket for a single upstream.
// We serialize access with a mutex because we use the same socket for many goroutines.
type UDPClient struct {
	upstreamAddr string
	mu           sync.Mutex
	conn         *dns.Conn
}

func newUDPClient(upstreamAddr string) (*UDPClient, error) {
	d := net.Dialer{Timeout: defaultDialTimeout, KeepAlive: defaultKeepAlive}
	raw, err := d.Dial("udp", upstreamAddr)
	if err != nil {
		return nil, err
	}
	return &UDPClient{
		upstreamAddr: upstreamAddr,
		conn:         &dns.Conn{Conn: raw},
	}, nil
}

// Exchange sends a query and reads a response using the shared UDP socket.
// It serializes access and applies the provided timeout.
func (u *UDPClient) Exchange(q *dns.Msg, timeout time.Duration) (*dns.Msg, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if err := u.conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}
	if err := u.conn.WriteMsg(q); err != nil {
		return nil, err
	}
	// The socket is shared across queries, so a late answer to an earlier
	// timed-out query may still be sitting in the buffer. A response is
	// only accepted if both the ID and the echoed Question match: ID
	// alone is 16 bits and collides under load (and is spoofable). The
	// deadline bounds the loop in time, maxStaleReads bounds it in
	// iterations so a datagram flood can't pin the socket mutex.
	for stale := 0; stale < maxStaleReads; stale++ {
		resp, err := u.conn.ReadMsg()
		if err != nil {
			return nil, err
		}
		if resp.Id == q.Id && questionMatches(q, resp) {
			return resp, nil
		}
	}
	return nil, errors.New("too many mismatched datagrams from upstream")
}

// questionMatches reports whether resp echoes q's question section
// (case-insensitive name, same type and class). Anything else is a stale
// or forged datagram and must not be accepted, let alone cached.
func questionMatches(q, resp *dns.Msg) bool {
	if len(q.Question) != 1 || len(resp.Question) != 1 {
		return false
	}
	a, b := q.Question[0], resp.Question[0]
	return a.Qtype == b.Qtype && a.Qclass == b.Qclass && strings.EqualFold(a.Name, b.Name)
}

// Close closes the underlying UDP socket.
func (u *UDPClient) Close() error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.conn == nil {
		return nil
	}
	err := u.conn.Close()
	u.conn = nil
	return err
}

// UpstreamPool manages outbound queries to a single upstream resolver.
// - UDP: a single long-lived socket (UDPClient) is used for the fast path.
// - TCP: a small pool of persistent TCP connections is maintained and reused.
type UpstreamPool struct {
	upstreamAddr string
	// UDP client (fast path)
	udp *UDPClient
	// TCP pool
	mu       sync.Mutex
	conns    []*net.TCPConn // fixed length == maxConns
	inUse    []bool
	maxConns int
	next     int // simple round-robin cursor for fairness
	// dialer used to create new TCP connections
	dialer net.Dialer
	closed bool
}

// NewUpstreamPool creates a pool for the given upstream address.
// maxConns must be >= 1 (the number of TCP connections to maintain).
func NewUpstreamPool(upstreamAddr string, maxConns int) (*UpstreamPool, error) {
	if maxConns < 1 {
		return nil, errors.New("maxConns must be >= 1")
	}

	udp, err := newUDPClient((upstreamAddr))
	if err != nil {
		return nil, err
	}

	// Preallocate fixed-size slices for easier invariant reasoning.
	conns := make(([]*net.TCPConn), maxConns)
	inUse := make([]bool, maxConns)

	return &UpstreamPool{
		upstreamAddr: upstreamAddr,
		udp:          udp,
		conns:        conns,
		inUse:        inUse,
		maxConns:     maxConns,
		dialer: net.Dialer{
			Timeout:   defaultDialTimeout,
			KeepAlive: defaultKeepAlive,
		},
	}, nil
}

// Close shuts down the pool and closes all sockets. It is safe to call multiple times.
func (p *UpstreamPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true

	var firstErr error
	if err := p.udp.Close(); err != nil {
		firstErr = err
	}
	for i, conn := range p.conns {
		if conn != nil {
			if err := conn.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
			p.conns[i] = nil
			p.inUse[i] = false
		}
	}
	return firstErr
}

// getTCPConn returns an index and an active *net.TCPConn. If a slot contains nil,
// it will attempt to dial and fill that slot. If all slots are busy, returns error.
// Note: caller must call releaseTCPConn(idx, hadErr) when done.
func (p *UpstreamPool) getTCPConn() (*net.TCPConn, int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, -1, errors.New("upstream pool is closed")
	}

	// Simple round-robin to ensure fairness.
	start := p.next
	for i := 0; i < p.maxConns; i++ {
		idx := (start + i) % p.maxConns
		if !p.inUse[idx] {
			// reserve a slot
			p.inUse[idx] = true
			p.next = (idx + 1) % p.maxConns
			// if there's already a connection, return it
			if p.conns[idx] != nil {
				return p.conns[idx], idx, nil
			}
			// otherwise, try to dial a new connection
			// mark that slot as expected to be filled; but we already set inUse true so others won't take it.
			p.mu.Unlock() // unlock while dialing
			raw, err := p.dialer.Dial("tcp", p.upstreamAddr)
			p.mu.Lock() // relock to update state
			if err != nil {
				p.inUse[idx] = false // release the slot on error
				return nil, -1, err
			}
			// successful dial: store typed *net.TCPConn
			if tcp, ok := raw.(*net.TCPConn); ok {
				p.conns[idx] = tcp
				return tcp, idx, nil
			}
			// unlikely: non-TCP conn returned
			_ = raw.Close()
			p.inUse[idx] = false
			return nil, -1, errors.New("unexpected non-tcp connection")
		}
	}
	// all slots are busy
	return nil, -1, errors.New("upstream pool exhausted")
}

// releaseTCPConn releases the connection at index idx back to the pool.
// If hadErr is true, the connection is closed and the slot becomes nil so
// the next user dials fresh; either way the slot is freed for reuse.
func (p *UpstreamPool) releaseTCPConn(idx int, hadErr bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// guard index
	if idx < 0 || idx >= len(p.conns) {
		return
	}

	if hadErr && p.conns[idx] != nil {
		_ = p.conns[idx].Close()
		p.conns[idx] = nil
	}
	p.inUse[idx] = false
}

// Exchange implements the UDP-fastpath -> TCP-fallback behavior.
// 1) Try UDP with shared socket (fast). If UDP returns a response and TC == false, return it.
// 2) Otherwise (error or truncated), use a pooled TCP connection and return that response.
//
// Important: callers should not assume Exchange is cheap; it performs network IO and may block.
func (p *UpstreamPool) Exchange(q *dns.Msg, timeout time.Duration) (*dns.Msg, error) {
	// First, try UDP (fast path)
	if p.udp != nil {
		resp, err := p.udp.Exchange(q, timeout)
		if err == nil && resp != nil && !resp.Truncated {
			return resp, nil
		}
		// else fall through to TCP fallback (either error or truncated)
	}

	// Fallback, TCP using pooled connections
	tcpConn, idx, err := p.getTCPConn()
	if err != nil {
		return nil, err
	}

	var hadErr bool
	// Wrapped in a closure so hadErr is read at return time, not captured
	// by value when the defer is declared.
	defer func() { p.releaseTCPConn(idx, hadErr) }()

	// wrap with dns.Conn for framing (length-prefix) and convenience
	dnsConn := &dns.Conn{Conn: tcpConn}
	if err := tcpConn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}

	if err := dnsConn.WriteMsg(q); err != nil {
		hadErr = true
		logger.Log.Errorf("Failed to write DNS query to TCP connection: %v", err)
		return nil, err
	}

	resp, err := dnsConn.ReadMsg()
	if err != nil {
		hadErr = true
		logger.Log.Errorf("Failed to read DNS response from TCP connection: %v", err)
		return nil, err
	}
	// A reused TCP conn can hold a stale response from a prior exchange
	// that timed out between write and read. Don't trust it; close the
	// conn so the next user starts clean.
	if resp.Id != q.Id || !questionMatches(q, resp) {
		hadErr = true
		return nil, errors.New("mismatched response on pooled TCP connection")
	}
	return resp, nil
}
