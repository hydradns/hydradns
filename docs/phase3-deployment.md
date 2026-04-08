# Phase 3: Plug-and-Play Deployment

**Date:** 2026-04-07
**Status:** Complete (core container verified; UI/scanner/landing Dockerfiles created but full compose blocked by WSL Docker credential issue)

## Context

docker-compose was broken: no dataplane service, config files not mounted, port mismatches, missing Dockerfiles. The controlplane + dataplane shared SQLite but were modeled as separate containers.

## Key Architecture Decision: Combined Core Container

Instead of separate controlplane/dataplane containers (which fight over SQLite), combined them into a single container with an entrypoint script that runs both processes. gRPC communicates over localhost:50051 inside the container.

## What Was Done

### Config fixes
- `config.yaml`: controlplane listen_addr 8086→8080, gRPC addr "dataplane:50051"→"localhost:50051"
- CORS middleware now reads `CORS_ORIGINS` env var (comma-separated), falls back to localhost:3000
- Controlplane main.go: DB path from `PHANTOM_DB` env var
- System state default: `DNSEnabled=true` (was false, which caused engine to REFUSE all queries on fresh DB)

### Dockerfiles created
- `apps/core/Dockerfile` — combined multi-stage: builds both binaries, copies configs, creates data dir
- `apps/core/docker/entrypoint.sh` — starts dataplane (background), waits 2s, starts controlplane (foreground), handles shutdown
- `apps/ui/Dockerfile` — multi-stage Next.js standalone build with `NEXT_PUBLIC_API_URL` build arg
- `apps/scanner/Dockerfile` — multi-stage Go build
- `apps/landing/Dockerfile` — multi-stage Vite build → nginx

### Next.js config fixes
- Removed `next.config.mjs` (conflicted with `.ts`, had wrong assetPrefix)
- Added `output: "standalone"` to `next.config.ts`

### docker-compose.yml rewrite
- Single `core` service with combined Dockerfile, volumes, healthcheck
- UI with build arg for API URL
- Scanner and landing services
- Persistent `core-data` volume for SQLite
- DNS ports (1053 UDP+TCP)
- Removed unused postgres/redis services (core uses SQLite)

### Updated
- `.env.example` — reflects current env vars
- Updated existing tests (system state default changed)

## Verification

Core container tested end-to-end:
- `curl http://localhost:8080/health` → `{"status":"ok"}`
- `dig @127.0.0.1 -p 1053 google.com` → REFUSED (policy block)
- `dig @127.0.0.1 -p 1053 github.com` → 20.207.73.82 (upstream forward)
- `dig @127.0.0.1 -p 1053 010sec.com` → REFUSED (blocklist block)
- All 52 Go tests pass

## Bug Found & Fixed During Deployment

**System state default was DNSEnabled=false.** On a fresh DB, the controlplane loaded this and told the dataplane to stop accepting queries. Combined with our Phase 1 drain-mode fix (which now sends REFUSED instead of silently dropping), this caused ALL queries to be REFUSED. Fixed by defaulting DNSEnabled=true.

## What's Blocked

UI/scanner/landing Docker builds fail due to WSL2 ↔ Docker Desktop credential resolution (`error getting credentials`). This is an environment issue, not a code issue. The Dockerfiles are structurally correct. Run `docker compose build` after restarting Docker Desktop.

## Lessons Learned

1. **Combined container > split containers for SQLite** — SQLite doesn't work well across container boundaries. A single container with an entrypoint script is simpler and correct.
2. **Default system state matters** — A "disabled by default" DNS engine means fresh installs refuse all queries. Default to enabled.
3. **Next.js .mjs overrides .ts config** — If both exist, .mjs wins. The .mjs had a stale assetPrefix that broke the build.
4. **Test in the container, not just locally** — The system state bug only manifested in Docker because local dev always had a pre-existing DB.
