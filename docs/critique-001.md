# Code Critique #001: Post-Phase 3 Review

**Date:** 2026-04-07
**Reviewer:** Automated senior engineer critique agent
**Scope:** Full codebase after Phases 1-3

## Critical Issues (Must Fix)

### 1. DefaultConfig nil panic
**File:** `apps/core/internal/config/config.go`
**Problem:** `loadConfig()` returns nil on error. `DefaultConfig` is nil. Every access (`config.DefaultConfig.DataPlane...`) panics.
**Status:** FIXED

### 2. Entrypoint race condition
**File:** `apps/core/docker/entrypoint.sh`
**Problem:** `sleep 2` is not reliable — controlplane may try gRPC before dataplane is listening.
**Status:** FIXED — replaced with wait-loop polling gRPC port

### 3. Blocklist fetch blocks startup
**File:** `apps/core/cmd/dataplane/main.go`
**Problem:** `blEngine.UpdateSource()` runs synchronously on startup with 30s timeout. If GitHub is down, dataplane never starts.
**Status:** FIXED — moved to background goroutine, DNS serves immediately

### 4. Sidebar doesn't track active route
**File:** `apps/ui/components/app-sidebar.tsx`
**Problem:** `isActive` hardcoded to Dashboard only. Other pages never show as active.
**Status:** FIXED — uses `usePathname()` to match current route

### 5. No API fetch timeout in frontend
**File:** `apps/ui/lib/api.ts`
**Problem:** `fetch()` has no timeout. If API is down, requests hang forever.
**Status:** FIXED — added AbortController with 10s timeout

## High-Priority Issues (Should Fix Soon)

### 6. Blocklist count is N+1 query
**File:** `apps/core/cmd/controlplane/handlers/blocklists.go`
**Problem:** `blocklistFromSource()` runs a COUNT query per source in a loop.
**Status:** FIXED — single batch query

### 7. No loading states on dashboard pages
**Problem:** Pages show blank then content appears. No spinner or skeleton.
**Status:** DEFERRED — cosmetic, not blocking

### 8. getDnsEngineStatus error silently swallowed
**File:** `apps/ui/app/dashboard/page.tsx`
**Problem:** `.catch(() => {})` hides connection errors.
**Status:** FIXED

### 9. No deferred rollback in DeleteSource
**File:** `apps/core/internal/storage/repositories/blocklist.go`
**Problem:** If commit fails, rollback might not happen.
**Status:** FIXED — added defer tx.Rollback()

### 10. Secret in config.yaml
**File:** `apps/core/configs/config.yaml`
**Problem:** Anonymization secret hardcoded and committed.
**Status:** FIXED — reads from PHANTOM_ANON_SECRET env var

## Noted for Later (valid but not blocking)

| Issue | Severity | Notes |
|---|---|---|
| No UPDATE endpoints for policies/blocklists | Medium | Create + delete only |
| No pagination on query logs | Medium | Returns all 100 |
| No rate limiting | Low | DoS risk |
| No request body size limits | Low | Memory DoS risk |
| No input validation on domains | Low | Can create invalid policies |
| Metrics window not configurable | Low | Hardcoded 5min |
| No dataplane-specific health check | Low | Compose only checks controlplane |
| GORM AutoMigrate only adds, never removes | Low | Need versioned migrations eventually |
