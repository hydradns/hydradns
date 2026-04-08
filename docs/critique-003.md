# Critique #003: Tier 2 Demo Polish Review

**Date:** 2026-04-08
**Scope:** AI threat detection, query logging, dashboard error states, branding
**Findings:** 10 issues (0 critical, 2 high, 5 medium, 3 low)

## Fixed Issues

| # | Severity | Issue | Fix |
|---|---|---|---|
| H-1 | HIGH | Synchronous DB write in DNS hot path blocks resolution | Made `logQuery` async with `go func()` |
| M-1 | MEDIUM | CDN domains (cloudfront.net, etc.) flagged as hex DGA | Added infra domain allowlist, raised hex threshold to 16+ chars |
| M-2 | MEDIUM | Shannon entropy uses byte count instead of rune count | Changed to `utf8.RuneCountInString()` |
| M-3 | MEDIUM | Base64 pattern matches normal long brand domains | Require actual base64 indicators (+, /, =) in pattern |
| M-4 | MEDIUM | Policy eval errors skip query logging | Added `logQuery("error")` before SERVFAIL return |
| M-5 | MEDIUM | Empty query log returns `null` instead of `[]` (frontend crash) | Changed to `make([]QueryLogEntry, 0)` |

## Also fixed
- Raised digit ratio threshold from 0.4 to 0.5 (reduces false positives on infrastructure hostnames)

## Deferred Issues

| # | Severity | Issue | Why deferred |
|---|---|---|---|
| H-2 | HIGH | Timestamp field relies on repo code, no autoCreateTime tag | Works correctly today; low risk since all writes go through repo |
| L-1 | LOW | AWS EC2 hostnames may trigger digit ratio check | Rare in consumer/school networks |
| L-2 | LOW | SRV records (6 labels) may trigger subdomain depth | Edge case; threshold at 5 is reasonable for demo |
| L-3 | LOW | No AbortController cleanup on unmount in logs page | React 18 handles this gracefully |
