// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hydradns/hydra-core/internal/blocklist"
	"github.com/hydradns/hydra-core/internal/storage/repositories"
)

// fakeBlocklistSource is a test double for blocklistSource. sig is read
// under mu so tests can mutate it concurrently with the reloader's Poll
// goroutines. listCalls counts List() invocations (i.e. rebuilds).
// If block is non-nil, List() waits on it before returning, letting tests
// hold a rebuild "in flight" to exercise coalescing/single-flight.
type fakeBlocklistSource struct {
	mu     sync.Mutex
	sig    repositories.BlocklistSignature
	sigErr error

	listCalls   int32
	maxInFlight int32
	inFlight    int32

	block chan struct{} // if set, List() waits for a send/close before returning
}

func (f *fakeBlocklistSource) Signature() (repositories.BlocklistSignature, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sig, f.sigErr
}

func (f *fakeBlocklistSource) setSignature(s repositories.BlocklistSignature) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sig = s
}

func (f *fakeBlocklistSource) List() ([]string, error) {
	n := atomic.AddInt32(&f.inFlight, 1)
	for {
		max := atomic.LoadInt32(&f.maxInFlight)
		if n <= max || atomic.CompareAndSwapInt32(&f.maxInFlight, max, n) {
			break
		}
	}
	atomic.AddInt32(&f.listCalls, 1)
	if f.block != nil {
		<-f.block
	}
	atomic.AddInt32(&f.inFlight, -1)
	return []string{"blocked.example"}, nil
}

func (f *fakeBlocklistSource) calls() int {
	return int(atomic.LoadInt32(&f.listCalls))
}

// waitUntil polls cond every 2ms until it's true or the timeout elapses.
func waitUntil(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if cond() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("condition not met within %s", timeout)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestBlocklistReloader_NoRebuildWhenUnchanged(t *testing.T) {
	fake := &fakeBlocklistSource{sig: repositories.BlocklistSignature{SourceCount: 1}}
	r := newBlocklistReloader(fake, blocklist.NewMemoryChecker())

	r.Poll() // first call always rebuilds (nothing observed yet)
	waitUntil(t, time.Second, func() bool { return fake.calls() == 1 })

	// Same signature, repeatedly: must not trigger another rebuild.
	for i := 0; i < 5; i++ {
		r.Poll()
	}
	time.Sleep(20 * time.Millisecond)
	if got := fake.calls(); got != 1 {
		t.Errorf("List() called %d times, want 1 (no rebuild on unchanged signature)", got)
	}
}

func TestBlocklistReloader_OneRebuildPerChange(t *testing.T) {
	fake := &fakeBlocklistSource{}
	r := newBlocklistReloader(fake, blocklist.NewMemoryChecker())

	for i := int64(1); i <= 4; i++ {
		fake.setSignature(repositories.BlocklistSignature{SourceCount: i})
		r.Poll()
		waitUntil(t, time.Second, func() bool { return fake.calls() == int(i) })
	}
}

func TestBlocklistReloader_CoalescesBurstsDuringRebuild(t *testing.T) {
	fake := &fakeBlocklistSource{block: make(chan struct{})}
	mem := blocklist.NewMemoryChecker()
	r := newBlocklistReloader(fake, mem)

	fake.setSignature(repositories.BlocklistSignature{SourceCount: 1})
	r.Poll() // starts a rebuild that blocks inside List()
	waitUntil(t, time.Second, func() bool { return atomic.LoadInt32(&fake.inFlight) == 1 })

	// Several changes land while the rebuild is in flight.
	for i := int64(2); i <= 6; i++ {
		fake.setSignature(repositories.BlocklistSignature{SourceCount: i})
		r.Poll()
	}

	// Release the first (blocked) rebuild.
	close(fake.block)

	// Expect exactly one more rebuild — not one per queued change.
	waitUntil(t, time.Second, func() bool { return fake.calls() == 2 })
	time.Sleep(30 * time.Millisecond) // give any extra (buggy) rebuilds a chance to show up
	if got := fake.calls(); got != 2 {
		t.Errorf("List() called %d times, want exactly 2 (1 initial + 1 coalesced pass)", got)
	}
}

func TestBlocklistReloader_SingleFlightUnderConcurrentTriggers(t *testing.T) {
	fake := &fakeBlocklistSource{}
	r := newBlocklistReloader(fake, blocklist.NewMemoryChecker())

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			fake.setSignature(repositories.BlocklistSignature{SourceCount: int64(i)})
			r.Poll()
		}(i)
	}
	wg.Wait()

	// Let any in-flight/coalesced rebuild settle.
	waitUntil(t, time.Second, func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		return !r.rebuilding
	})

	if max := atomic.LoadInt32(&fake.maxInFlight); max > 1 {
		t.Errorf("max concurrent List() calls = %d, want at most 1 (single-flight)", max)
	}
}

func TestBlocklistReloader_SignatureErrorDoesNotRebuild(t *testing.T) {
	fake := &fakeBlocklistSource{sigErr: errTest}
	r := newBlocklistReloader(fake, blocklist.NewMemoryChecker())

	r.Poll()
	time.Sleep(20 * time.Millisecond)
	if got := fake.calls(); got != 0 {
		t.Errorf("List() called %d times, want 0 when Signature() errors", got)
	}
}

func TestStartBlocklistPoll_ZeroDisables(t *testing.T) {
	fake := &fakeBlocklistSource{sig: repositories.BlocklistSignature{SourceCount: 1}}
	r := newBlocklistReloader(fake, blocklist.NewMemoryChecker())

	stop := startBlocklistPoll(r, 0)
	defer stop()

	time.Sleep(50 * time.Millisecond)
	if got := fake.calls(); got != 0 {
		t.Errorf("List() called %d times, want 0 with interval=0 (poll disabled)", got)
	}
}

func TestStartBlocklistPoll_PositiveIntervalPolls(t *testing.T) {
	fake := &fakeBlocklistSource{}
	r := newBlocklistReloader(fake, blocklist.NewMemoryChecker())

	stop := startBlocklistPoll(r, 5*time.Millisecond)
	defer stop()

	fake.setSignature(repositories.BlocklistSignature{SourceCount: 1})
	waitUntil(t, time.Second, func() bool { return fake.calls() >= 1 })

	fake.setSignature(repositories.BlocklistSignature{SourceCount: 2})
	waitUntil(t, time.Second, func() bool { return fake.calls() >= 2 })
}

// errTest is a sentinel error for tests.
type testError string

func (e testError) Error() string { return string(e) }

const errTest = testError("boom")
