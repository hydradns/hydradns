# HydraDNS Stress Test Plan — "Juice Out the Box"

**Date:** 2026-06-11
**Purpose:** Find the real limits of one HydraDNS box before putting it on a paying customer's network. Every number we discover here becomes either a sales claim ("handles X devices") or a fix ticket. No number, no claim.

> **No Pi yet.** All numbers below are from the dev laptop (22 cores, WSL2), load
> generated *inside the core container* to remove docker-proxy and host-network
> noise from the DNS path. The laptop *is* the current production hardware (router
> points at the laptop's static IP). When a Pi arrives, T8 re-runs the redline on
> it — that number, not the laptop's, goes in sales material.

---

## T1 RESULTS (2026-06-11) — two ceilings found and lifted

T1 immediately surfaced a hard throughput ceiling and, once lifted, a memory bomb.
Both are now fixed and verified. Load generated with dnspyre inside the container.

**Ceiling #1 — throughput capped at ~500 QPS (every query did a SQL blocklist read).**
Root cause: `IsBlocked` ran a SQL `COUNT` against the 92k-row `blocklist_entries`
table on **every** query (blocked *and* allowed — the check precedes the cache),
serialized through the single SQLite connection (`MaxOpenConns=1`) that was also
absorbing the async write flood. Engine self-latency under load: p50=50ms,
p99=5000ms, grade=bad. Fix: in-memory `MemoryChecker` (atomic map, reloaded on the
6h refresh). The advertised "Bloom filter" never existed for the blocklist; this is
the first time the hot path is actually in-memory.

**Ceiling #2 — memory ballooned to 2GB under load (goroutine-per-query logging).**
Unmasked once throughput was free: every query spawned a goroutine doing INSERT +
UPDATE on the single connection; at high QPS they piled up faster than SQLite drained.
Fix: bounded (4096) single-goroutine batched writer; non-blocking enqueue drops
(counted) when full, so logging can never stall DNS or OOM the box.

| Offered load | Throughput | p99 (blocked) | **RSS before** | **RSS after** |
|---|---|---|---|---|
| 500 QPS | tracks offered | 0-2ms | 98 MiB | 82 MiB |
| 1,000 QPS | tracks offered | 1-2ms | 199 MiB | 76 MiB |
| 2,000 QPS | tracks offered | 0-62ms | 443 MiB | 80 MiB |
| 5,000 QPS | ~4,800 | 23ms | 1,007 MiB | 82 MiB |
| 10,000 QPS | ~9,500 | 22-28ms | **2,063 MiB** | **87 MiB** |

**Net: throughput ceiling ~500 → ~9,500 QPS (≈19x); RSS under max load 2,063 → 87 MiB
(flat). Engine self-latency grade bad → excellent (p50 5ms, p99 20ms).**
Against the design target (500 sustained / 2,000 burst), the box now has ~5x headroom
on the laptop.

---

## T2-lite RESULTS (2026-06-11) — sustained soak + retention

Not the full 24h (no Pi yet), but a 3-minute sustained soak at 1,500 QPS (≈10x a
coaching center's sustained load) to check for leaks and validate retention.

**Query-log retention shipped** (the T2 blocker). Background loop in the dataplane,
runs at startup then hourly:
- `QUERY_LOG_RETENTION_DAYS` (default 7) — delete rows older than N days.
- `QUERY_LOG_MAX_ROWS` (default 1,000,000) — keep at most N newest rows (SD-card insurance).
Verified live: first boot with retention pruned **7,235** stale rows from the real DB.
(SQLite `DELETE` reuses pages, so the `.db` file settles at its high-water mark rather
than shrinking — bounded, not growing. No auto-`VACUUM`: it locks the DB and would
stall DNS.)

**Soak: 248,166 queries at 1,379 QPS over 180s.**

| Metric | Result |
|---|---|
| RSS trajectory | 119→127→131→133→139→139→**114** MiB (bounded sawtooth, GC returns to baseline) |
| Engine errors | **0 / 248,166** (server-side); 24 client-side tail timeouts (0.01%) |
| Engine self-latency | p50/p95/p99 = 5ms, grade **excellent** |
| Post-soak idle RSS | 114 MiB |
| Serving correctness | ads blocked, real domains resolve, `localhost` not blocked — all intact |

**Pass.** Memory is bounded (oscillates ~114-140 MiB, reclaims to baseline — vs the old
98→2,063 MiB monotonic climb). No leak signature over 180s. Caveat: a true 24h soak on
the Pi (folded into T8) is the final confirmation that the sawtooth peak doesn't trend
up over hours; 3 minutes can't prove that, but return-to-baseline is strong evidence.

---

**Baseline already measured (2026-06-11, WSL2, 250 sequential queries):**

| Metric | Pre-cache | Post-cache + timeout fix |
|---|---|---|
| p50 | 8ms | 3ms |
| p90 | 20ms | 4ms |
| p99 | 1,980ms | 19ms |
| Dropped | 7/250 (2.8%) | 0/250 |
| Direct-to-8.8.8.8 control | p99 223ms, 7.6% loss | same path |

---

## What a real customer network looks like (size the test to this)

| Deployment | Devices | Est. sustained QPS | Burst QPS |
|---|---|---|---|
| PG/hostel (30 tenants) | ~60 | 20-40 | 200 |
| Coaching center | ~100 | 30-80 | 400 |
| Small office (50 staff) | ~120 | 40-100 | 500 |
| School office + labs | ~200 | 60-150 | 800 |

**Design target: sustain 500 QPS with p99 < 100ms; survive 2,000 QPS bursts without dropping queries.** That covers every customer in the pipeline with 3x headroom.

---

## Tooling

- **dnsperf** (`apt install dnsperf`) — primary load generator. The existing `scripts/stress-test.sh` already wraps it.
- **tc netem** — inject packet loss/latency on the upstream path (jerk simulation).
- **docker stats / `docker exec ... cat /proc/1/status`** — RSS, goroutine pressure.
- **`/api/v1/dns/metrics`** — engine's own p50/p95/p99 + error counts.
- Query file: reuse the 70% normal / 25% blocked / 5% suspicious mix from `stress-test.sh`, **plus a cache-buster variant** (unique random subdomains, e.g. `q000001.stress.example.com`) to force the uncached path.

---

## Test Matrix

### T1 — Throughput ceiling (find the redline)
Ramp dnsperf: 500 → 1,000 → 2,000 → 5,000 → 10,000 QPS, 60s each, 20 client sockets.
Run three mixes separately:
- **Blocked-only** (pure blocklist hits): measures the engine floor — no network involved. Expect 10k+ QPS.
- **Cached-only** (repeat 20 popular domains): measures cache path. Expect near engine floor.
- **Cache-buster** (all unique domains): measures upstream-bound path. This is the bottleneck — every query holds a slot on a shared UDP socket per upstream.

**Record per step:** achieved QPS, lost queries, p50/p95/p99, container CPU/RSS.
**Redline = the step where loss > 0.1% or p99 > 250ms.** That number, divided by ~0.8 QPS/device, is the honest "supports N devices" claim.

### T2 — Sustained soak (the renewal-killer test)
500 QPS, realistic mix (80% repeat domains, 15% blocked, 5% unique), **minimum 1 hour; 24h before first paid install.**
Watch for the three predicted failure points:

1. **Goroutine-per-query logging.** `engine.go logQuery()` spawns one goroutine per query, all funneling into SQLite with `MaxOpenConns=1`. At 500 QPS that's 500 goroutines/sec contending for one connection. Predicted symptom: goroutine count and RSS climb until the box stalls or OOMs.
   *Measure:* `expvar`/pprof goroutine count every 60s; RSS trend.
   *Fix ready if it falls over:* buffered channel + single batching writer (INSERT 100 rows/transaction).
2. **Unbounded DNSQuery table.** 500 QPS × 24h = 43M rows. The Pi's SD card will fill and SQLite will slow.
   *Measure:* DB file size every 10 min; query-log API latency at hour 0 vs hour 24.
   *Fix ready:* retention cleanup job (already on the Tier 3 list, becomes mandatory).
3. **Cache memory.** 20k-entry LRU; verify RSS plateaus rather than grows after the cache fills.

**Pass:** zero lost queries, flat RSS after warmup, flat goroutine count, DB growth linear and within SD budget.

### T3 — Hostile upstream (jerk simulation)
The April-style WSL2 numbers showed the real internet drops packets. Reproduce deliberately:
```
# on the docker host, against the upstream path
tc qdisc add dev eth0 root netem loss 10% delay 50ms 20ms
```
Run 200 QPS realistic mix for 5 min under: 5% loss, 10% loss, 20% loss, 200ms added latency.
**Pass:** cached domains stay < 5ms regardless of loss (cache shields them); uncached p99 stays under ~3.2s (1.5s timeout × 2 retries) and **client-visible error rate < 0.5%** because retry+failover absorbs the loss.

### T4 — Upstream failover
While running 200 QPS, block the primary resolver entirely:
```
iptables -A OUTPUT -d 8.8.8.8 -j DROP    # then later -D to restore
```
**Measure:** how long until queries flow normally via 1.1.1.1, and what clients experience during the window.
**Pass:** no SERVFAIL storm; steady state restored < 5s. *Known gap to confirm:* there is no health-marking on upstreams — every uncached query re-tries the dead upstream first, paying 2 × 1.5s every time until it's unblocked. If T4 shows this, the fix is a circuit breaker (skip an upstream for 30s after N consecutive failures).

### T5 — Burst and recovery
Idle → instant 5,000 QPS for 10s → idle. Three cycles.
**Pass:** no crash, no stuck goroutines after burst (count returns to baseline), queries during burst degrade gracefully (slower, not dropped silently).

### T6 — Blocklist scale
Load a 1M-domain list (concat several public lists) alongside StevenBlack.
**Measure:** RSS delta, startup time, blocked-query latency before/after, snapshot rebuild time while serving 200 QPS (the atomic rebuild must not stall queries).
**Pass:** blocked-path latency unchanged; rebuild causes no query loss.

### T7 — Restart under fire (demo-day insurance)
At 200 QPS: `docker restart hydradns-core-1`.
**Measure:** total outage seconds, time to first answered query, cache cold-start penalty, any DB corruption (`PRAGMA integrity_check`).
**Pass:** serving within 15s, no corruption, no manual intervention.

### T8 — On the actual Pi (192.168.1.50)
Re-run T1 redline, T2 (24h), and T5 on the Pi — the WSL2 numbers don't transfer.
Pi-specific watches: CPU thermal throttling (`vcgencmd measure_temp` per minute — sustained load in an Indian summer closet is the real environment), SD card write latency during log flushes, total power draw if on a cheap adapter.
**The Pi redline from T1 is the number that goes in the sales material**, not the laptop number.

---

## Pass = a claims sheet you can say out loud

Every test feeds a line you can use with a buyer without flinching:

| Claim | Source |
|---|---|
| "Supports N devices" | T1 redline on Pi (T8) ÷ 0.8 QPS/device, halved for safety |
| "Most lookups answered on-premises in under 5ms — your internet gets faster" | T1 cached mix + T3 |
| "Survives your ISP having a bad day" | T3, T4 |
| "Runs for 24 hours flat-out without degrading" | T2 |
| "Recovers from a power cut in under a minute" | T7 + Pi boot time |

## Execution order

1. T1 + T5 now (laptop) — 1 evening. Finds the goroutine-logging ceiling immediately.
2. T2 1-hour + T3 + T4 — next session. Expect to come out of this with the batching-writer and circuit-breaker fixes.
3. T6, T7 — after those fixes land.
4. T8 full pass on the Pi — the week before the first install. 24h soak runs while you do discovery conversations.
