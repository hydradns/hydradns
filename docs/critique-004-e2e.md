# Critique #004: End-to-End Test Review

**Date:** 2026-04-08
**Scope:** Full Docker E2E test — auth, DNS, threat detection, policies, blocklists, statistics
**Method:** Fresh container, setup wizard, DNS queries, API verification

## Bugs Found and Fixed

### HIGH

| # | Bug | Root Cause | Fix |
|---|-----|-----------|-----|
| 1 | Dashboard always shows 0 queries | `Statistics.IncrementCounter()` never called from DNS pipeline | Added `statistics` field to Engine, call `IncrementCounter(action)` in async `logQuery()` |
| 2 | API-created policies never enforced by DNS engine | Dataplane loads policies from JSON file only, never from DB | Added `reloadPolicies()` that merges file + DB policies, polls DB every 5s |

### MEDIUM

| # | Bug | Status |
|---|-----|--------|
| 3 | Blocklist entries accumulate across restarts | Deferred — single-session demo OK, needs snapshot cleanup for production |
| 4 | Blocklist IsBlocked does SQL query per DNS request | Deferred — acceptable for demo scale, needs in-memory cache for Pi production |

### LOW

| # | Bug | Status |
|---|-----|--------|
| 5 | google.com blocked by StevenBlack | Not actually a bug — only `ads.google.com` is blocked, `google.com` itself is not |

## E2E Test Results (After Fixes)

| Test | Expected | Actual |
|:-----|:---------|:-------|
| Auth: pre-setup status | `setup_complete: false` | PASS |
| Auth: protected route before setup | 403 | PASS |
| Auth: setup with blocklist | token returned | PASS |
| Auth: duplicate setup | 409 | PASS |
| Auth: login | same token | PASS |
| Auth: no token | 401 | PASS |
| Auth: with token | data returned | PASS |
| DNS: github.com | allow, resolved | PASS |
| DNS: tiktok.com (API policy) | REFUSED | PASS |
| DNS: ads.google.com (blocklist) | REFUSED | PASS |
| DNS: DGA hex domain | flagged, threat score 0.9 | PASS |
| Stats: dashboard summary | 4 queries, 2 blocked, 50% | PASS |
| Query logs | 4 entries with threat fields | PASS |
| Blocklist: auto-refresh | 92,283 domains loaded | PASS |
| Policy: DB-created enforced | tiktok.com blocked within 6s | PASS |
