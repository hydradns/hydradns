#!/usr/bin/env bash
#
# HydraDNS Stress Test
# Usage: ./stress-test.sh [SERVER_IP] [DURATION_SECONDS]
#
set -euo pipefail

SERVER="${1:-127.0.0.1}"
DURATION="${2:-30}"
PORT="${3:-53}"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

info()  { echo -e "${GREEN}[+]${NC} $1"; }
warn()  { echo -e "${YELLOW}[!]${NC} $1"; }

# Check for dnsperf
if ! command -v dnsperf &>/dev/null; then
    warn "dnsperf not found. Installing..."
    sudo apt-get update -qq && sudo apt-get install -y -qq dnsperf 2>/dev/null || {
        warn "dnsperf install failed. Using dig-based fallback."
        USE_DIG=true
    }
fi

# Generate test domains file
QUERY_FILE=$(mktemp)
trap "rm -f $QUERY_FILE" EXIT

info "Generating test query file..."

# Mix of normal, blocked, and suspicious domains
NORMAL_DOMAINS=(
    google.com github.com wikipedia.org stackoverflow.com
    amazon.com microsoft.com apple.com cloudflare.com
    netflix.com spotify.com linkedin.com medium.com
    reddit.com twitter.com youtube.com twitch.tv
    npmjs.com pypi.org golang.org rust-lang.org
    ubuntu.com debian.org archlinux.org fedoraproject.org
)

BLOCKED_DOMAINS=(
    ads.google.com tracking.example.com pixel.facebook.com
    adservice.google.com analytics.yahoo.com ad.doubleclick.net
    pagead2.googlesyndication.com facebook-ads.com ad-display.smartadserver.com
    ads.yahoo.com ads.twitter.com graph.facebook.com
)

SUSPICIOUS_DOMAINS=(
    a1b2c3d4e5f6a7b8c9d0.evil.com
    xkq7mz9plw2vb8nt.malware.org
    a.b.c.d.e.f.g.evil.com
    deadbeefcafebabe1234.bad.net
)

# Build query file with realistic mix (70% normal, 25% blocked, 5% suspicious)
for i in $(seq 1 10000); do
    rand=$((RANDOM % 100))
    if [ $rand -lt 70 ]; then
        domain=${NORMAL_DOMAINS[$((RANDOM % ${#NORMAL_DOMAINS[@]}))]}
    elif [ $rand -lt 95 ]; then
        domain=${BLOCKED_DOMAINS[$((RANDOM % ${#BLOCKED_DOMAINS[@]}))]}
    else
        domain=${SUSPICIOUS_DOMAINS[$((RANDOM % ${#SUSPICIOUS_DOMAINS[@]}))]}
    fi
    echo "$domain A" >> "$QUERY_FILE"
done

TOTAL_QUERIES=$(wc -l < "$QUERY_FILE")
info "Generated $TOTAL_QUERIES queries (70% normal, 25% blocked, 5% suspicious)"

echo ""
echo "========================================"
echo "  HydraDNS Stress Test"
echo "  Server: ${SERVER}:${PORT}"
echo "  Duration: ${DURATION}s"
echo "========================================"
echo ""

if [ "${USE_DIG:-false}" = "true" ]; then
    # Fallback: parallel dig queries
    info "Running dig-based stress test (${DURATION}s)..."

    RESULTS=$(mktemp)
    START=$(date +%s)
    COUNT=0
    ERRORS=0

    while true; do
        NOW=$(date +%s)
        ELAPSED=$((NOW - START))
        [ $ELAPSED -ge $DURATION ] && break

        # Fire 10 parallel queries
        for j in $(seq 1 10); do
            domain=${NORMAL_DOMAINS[$((RANDOM % ${#NORMAL_DOMAINS[@]}))]}
            dig @${SERVER} -p ${PORT} ${domain} +short +timeout=2 +tries=1 &>/dev/null && true || ERRORS=$((ERRORS + 1))
            COUNT=$((COUNT + 1))
        done
        wait
    done

    QPS=$((COUNT / DURATION))
    info "Results:"
    echo "  Total queries:    $COUNT"
    echo "  Duration:         ${DURATION}s"
    echo "  Queries/sec:      $QPS"
    echo "  Errors:           $ERRORS"
    rm -f "$RESULTS"
else
    # Use dnsperf for accurate benchmarking
    info "Running dnsperf benchmark..."
    echo ""

    # Test 1: Baseline throughput
    info "Test 1: Maximum throughput (no rate limit)"
    dnsperf -s "$SERVER" -p "$PORT" -d "$QUERY_FILE" -l "$DURATION" -c 20 -Q 10000 2>&1 | \
        grep -E "Queries sent|Queries completed|Run time|Queries per second|Average Latency|Latency StdDev"

    echo ""
    sleep 3

    # Test 2: Sustained load at 500 QPS
    info "Test 2: Sustained load (500 QPS)"
    dnsperf -s "$SERVER" -p "$PORT" -d "$QUERY_FILE" -l "$DURATION" -c 10 -Q 500 2>&1 | \
        grep -E "Queries sent|Queries completed|Run time|Queries per second|Average Latency|Latency StdDev"

    echo ""
    sleep 3

    # Test 3: Sustained load at 1000 QPS
    info "Test 3: Sustained load (1000 QPS)"
    dnsperf -s "$SERVER" -p "$PORT" -d "$QUERY_FILE" -l "$DURATION" -c 20 -Q 1000 2>&1 | \
        grep -E "Queries sent|Queries completed|Run time|Queries per second|Average Latency|Latency StdDev"

    echo ""
    sleep 3

    # Test 4: Burst test (5000 QPS)
    info "Test 4: Burst load (5000 QPS, 10s)"
    dnsperf -s "$SERVER" -p "$PORT" -d "$QUERY_FILE" -l 10 -c 50 -Q 5000 2>&1 | \
        grep -E "Queries sent|Queries completed|Run time|Queries per second|Average Latency|Latency StdDev"
fi

echo ""
echo "========================================"
echo "  Test Complete"
echo "========================================"

# Check HydraDNS stats after test
TOKEN="${HYDRA_TOKEN:-}"
if [ -n "$TOKEN" ]; then
    echo ""
    info "HydraDNS Dashboard Stats:"
    curl -s -H "Authorization: Bearer $TOKEN" http://${SERVER}:8080/api/v1/dashboard/summary 2>/dev/null | \
        python3 -c "
import sys,json
d = json.load(sys.stdin).get('data',{})
print(f\"  Total Queries:  {d.get('total_queries',0)}\")
print(f\"  Blocked:        {d.get('blocked_queries',0)}\")
print(f\"  Allowed:        {d.get('allowed_queries',0)}\")
print(f\"  Block Rate:     {d.get('block_rate_percent',0):.1f}%\")
" 2>/dev/null || echo "  (could not fetch stats)"

    echo ""
    info "DNS Metrics:"
    curl -s -H "Authorization: Bearer $TOKEN" http://${SERVER}:8080/api/v1/dns/metrics 2>/dev/null | \
        python3 -c "
import sys,json
d = json.load(sys.stdin).get('data',{})
q = d.get('queries',{})
l = d.get('latency_ms',{})
print(f\"  Total:   {q.get('total',0)}\")
print(f\"  Errors:  {q.get('errors',0)}\")
print(f\"  P50:     {l.get('p50',0)}ms\")
print(f\"  P95:     {l.get('p95',0)}ms\")
print(f\"  P99:     {l.get('p99',0)}ms\")
print(f\"  Grade:   {d.get('grade','unknown')}\")
" 2>/dev/null || echo "  (could not fetch metrics)"
fi
