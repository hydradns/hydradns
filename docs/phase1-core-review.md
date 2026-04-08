# Phase 1: Core Code Review & Bug Fixes

**Date:** 2026-04-05
**Status:** Complete

## Context

The core DNS blocking feature was working but had several correctness bugs that would cause client hangs, a broken Docker setup, and zero test coverage. This was the first pass to make the core reliable.

## What We Found (Code Review)

### Critical Bugs
1. **Silent empty responses** — Three code paths in `engine.go` returned without sending any DNS response, causing clients to hang indefinitely:
   - Drain mode (acceptQueries=false) → no response
   - Policy evaluation failure → no response
   - Upstream nil response → potential nil dereference
2. **Domain normalization mismatch** — DNS queries arrive as FQDNs (`EXAMPLE.COM.`) but the Bloom filter was built with normalized domains (`example.com`). Mismatches caused policies to not trigger.
3. **Duplicate database migrations** — Both `db.go` and `store.go` called AutoMigrate with overlapping models.
4. **Docker Compose wrong Dockerfile path** — Referenced `Dockerfile` but actual file was `docker/controlplane.Dockerfile`. Build would fail.

### Important Issues
5. Unsafe type assertion in `status.go` could panic
6. Alpine `latest` tag in controlplane Dockerfile (unpinned)
7. Dead commented-out files (`handler.go` — 69 lines of dead code)
8. Hardcoded magic numbers (timeout 5s, retries 2, pool size 4)

## What We Fixed

| Fix | File |
|---|---|
| Send REFUSED in drain mode | `internal/dnsengine/engine.go` |
| Send SERVFAIL on policy eval error | `internal/dnsengine/engine.go` |
| Nil-check upstream response | `internal/dnsengine/engine.go` |
| Normalize domain once at top of ProcessDNSQuery | `internal/dnsengine/engine.go` |
| Fix respondRedirect to not append nil RR | `internal/dnsengine/engine.go` |
| Safe type assertion with ok check | `internal/dnsengine/status.go` |
| Consolidate migrations in db.go only | `internal/storage/db/db.go`, `repositories/store.go` |
| Fix Dockerfile path in docker-compose | `docker-compose.yml` |
| Pin alpine:3.20 | `docker/controlplane.Dockerfile` |
| Delete dead handler.go | `internal/dnsengine/handler.go` (deleted) |
| Add env var overrides for config/db/policies paths | `config.go`, `cmd/dataplane/main.go` |

## Verification

- `go build ./cmd/controlplane && go build ./cmd/dataplane` — clean
- `go vet ./...` — clean  
- Ran dataplane locally, tested with `dig`:
  - `google.com` → REFUSED (policy block) ✓
  - `010sec.com` → REFUSED (blocklist block) ✓
  - `github.com` → NOERROR + IP (upstream forward) ✓

## What We Deferred

- Mock handlers (wired in Phase 1b)
- Tests (written in Phase 1b)
- Query log retention, TLS, CORS config, regex/wildcard policies, CI pipeline

## Lessons Learned

1. **Always send a DNS response** — Every code path in a DNS handler must write a response. Silent returns = client hangs. Use REFUSED for "won't serve" and SERVFAIL for "can't serve".
2. **Normalize early, normalize once** — Domain normalization should happen at the entry point, not scattered across each subsystem. The engine, blocklist, and policy all had their own normalization.
3. **Hardcoded paths break local dev** — `/app/data/` and `/app/configs/` only work inside Docker. Env var overrides (`PHANTOM_CONFIG`, `PHANTOM_DB`, `PHANTOM_POLICIES`) were needed to run locally.
4. **Duplicate migrations are a time bomb** — Two places calling AutoMigrate with different model sets means some models get migrated twice and some get missed. Single source of truth.
