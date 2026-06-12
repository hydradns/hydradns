# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

HydraDNS is a DNS-layer security and privacy gateway built as a monorepo of Git submodules orchestrated by Docker Compose. Each service lives in its own GitHub repository and is aggregated here under `apps/`.

## Build & Run Commands

### Full Stack (root)
- `make setup` — initialize submodules and project
- `make start` — `docker compose up -d` (all services)
- `make stop` — `docker compose down`
- `make update` — pull latest submodule changes (`git submodule update --remote --merge`)
- `make logs` / `make core-logs` / `make ui-logs` / `make scanner-logs` / `make landing-logs` — tail logs
- `make build-core` / `make build-ui` / `make build-landing` / `make build-scanner` — build individual services
- `make restart-core` / `make restart-ui` / etc. — rebuild and restart a single service

### Core (Go DNS Engine) — `apps/core/`
- `make build` — compile controlplane & dataplane binaries
- `make test` — run all tests with coverage
- `go test -v ./internal/dnsengine/...` — run a single package's tests
- `make fmt` / `make vet` / `make lint` — code quality (uses golangci-lint)
- `make proto-generate` — regenerate gRPC/protobuf code (requires buf)
- `make proto-lint` — lint proto definitions

### UI (Next.js Dashboard) — `apps/ui/`
- `npm run dev` — dev server on port 3000
- `npm run build` / `npm run lint`

### Landing (Vite + React) — `apps/landing/`
- `npm run dev` — dev server on port 3001
- `npm run build` / `npm run lint`
- `npm run test` / `npm run test:watch` — Vitest with jsdom

### CLI — `apps/cli/`
- `go build -o hydra .` — build CLI binary
- `./hydra --help` — list all commands
- `./hydra mcp` — start MCP server (JSON-RPC 2.0 over stdio)

### Scanner (Go) — `apps/scanner/`
- `make build` — compile scanner binary

### CLI (Go / Cobra) — `apps/cli/`
- `go build -o hydra` — produces the `hydra` binary at the package root
- `./hydra <command>` — `status`, `engine`, `block`, `blocklists`, `policies`, `metrics`, `logs`, `login`, `setup-router`, `mcp`
- `./hydra mcp` — runs the MCP stdio server (used by Gemini/Claude clients; ~9 tools incl. `create_policy` for batch ops)
- `go test ./...` — Cobra command tests live next to the commands (e.g. `cmd/setup_router_test.go`)
- API client lives in `apps/cli/api/client.go` and talks to the controlplane on `:8080`

## Architecture

```
Root (orchestrator)
├── apps/core       — Go 1.24, Gin, gRPC, GORM/SQLite
│   ├── cmd/controlplane/   — Admin API (host :8080)
│   ├── cmd/dataplane/      — DNS server; gRPC on :50051 (internal only, not published)
│   ├── internal/           — blocklist, dnsengine, policy, storage, grpc, metrics
│   └── proto/              — Protobuf definitions (buf for codegen → internal/gen/proto/)
│   NOTE: in production compose, controlplane + dataplane run as one combined `core` container
├── apps/ui         — Next.js 16, React 19, TypeScript, Tailwind v4, shadcn/ui (port 3000)
├── apps/landing    — Vite, React 18, TypeScript, Tailwind v3, shadcn/ui (port 3001)
├── apps/scanner    — Go 1.25, network scanning worker (no exposed port; commented out in compose)
├── apps/cli        — Cobra CLI + MCP stdio server (talks to controlplane API)
├── docker-compose.yml          — base: only `core` and `ui` are active; scanner/landing commented out
├── docker-compose.override.yml — local-dev overlay (auto-merged): WSL2-safe DNS port, `BLOCK_RESPONSE` env
├── docs/                       — phase plans, critiques, demo playbook, report.md
└── scripts/                    — setup.sh, install.sh, stress-test.sh
```

There is **no postgres or redis service** in compose. `core` writes to a Docker volume (`core-data`) backed by SQLite.

## Core Domain Concepts

### Intent vs Reality (Control Plane ↔ Data Plane)

The control plane maintains **desired state** in SQLite (source of truth). The data plane holds **actual runtime state**. When a change is made (e.g., toggle DNS engine), the control plane persists intent to DB, then applies it to the data plane via gRPC (`SetAcceptQueries`). Status endpoints return both desired and actual state combined. gRPC services defined in `proto/phantomdns/v1/status.proto`.

### DNS Query Pipeline (4-step, early exit)

1. **Blocklist check** — in-memory membership test (`internal/blocklist/memory.go`, `MemoryChecker`: atomic `map[string]struct{}` of all blocked domains + parent-domain walk). Reloaded on the 6h refresh; the DNS hot path never hits the DB. If blocked → respond per `BLOCK_RESPONSE`. (Historical note: this check used to run a per-query SQL `COUNT`, which capped throughput at ~500 QPS — see `docs/stress-test-plan.md` T1.)
2. **Policy evaluation** — Bloom filter for fast O(1) negative lookup, then exact domain match against `PolicySnapshot` (atomic rebuild on change). Multiple matches resolved by priority, then lexicographic ID
3. **Response cache** — TTL-respecting LRU (20k entries, `internal/dnsengine/cache.go`); only allowed queries are cached, never blocked/redirect responses
4. **Upstream forward** — pool-per-resolver with failover across all configured upstreams (1.5s per-attempt timeout, 2 retries each)

Query logging and statistics are written off the hot path by a bounded batched writer (`internal/dnsengine/querylog_writer.go`): non-blocking enqueue, single drain goroutine, batched inserts. Drops (counted) when its 4096-deep queue is full so logging can never stall resolution or grow memory without bound.

Domain normalization: lowercase + strip trailing dot (e.g., `EXAMPLE.COM.` → `example.com`).

### Blocklist Engine

Sources are fetched with ETag support (304 skip), SHA256 checksum tracking, and atomic persistence (transaction wraps snapshot + entries + metadata). Multiple format parsers: hosts, domain-list, ads-list — selected by `format` field on `BlocklistSource`. Blocklists auto-refresh on a configurable interval (default 6h, env: `BLOCKLIST_UPDATE_INTERVAL`).

### Policy Format

JSON file at `configs/policies.json`. Array of policies with `id`, `action` (BLOCK/ALLOW/REDIRECT), `domains`, optional `regexes`, `priority` (higher wins). Regexes are compiled/validated on load but **not yet evaluated at query time**. Wildcards also parsed but not evaluated.

### Authentication

Single admin user model (`AdminCredential` singleton in DB). Bearer token auth on all API endpoints except `/health`, `/api/v1/auth/status`, `/api/v1/auth/login`, `/api/v1/auth/setup`. Token is a UUID API key generated during setup. Password hashed with bcrypt.

**Auth flow:** First boot → setup wizard creates admin + returns token. Subsequent access → login with password → get token. Dashboard stores token in localStorage + cookie.

**Auth endpoints (unprotected):**
- `GET /api/v1/auth/status` — returns `{setup_complete: bool}`
- `POST /api/v1/auth/setup` — creates admin, optionally configures blocklists
- `POST /api/v1/auth/login` — validates password, returns token

## Configuration

- **Config file**: `configs/config.yaml` — top-level `dataplane` and `controlplane` keys
- **DataPlane config**: `listen_addr` (UDP/TCP), `upstream_resolvers` (list with failover), `grpc_server` (port/addr), `blocklist_update_interval`
- **Policy file**: `configs/policies.json` loaded from disk on dataplane startup
- **Config loaded as package singleton**: `config.DefaultConfig` with env var overrides (`DNS_LISTEN_ADDR`, `BLOCKLIST_UPDATE_INTERVAL`)
- **Environment**: `.env` file (gitignored) with fallback defaults. See `.env.example`.

| Env Variable | Default | Description |
|:-------------|:--------|:------------|
| `PHANTOM_CONFIG` | `/app/configs/config.yaml` | Path to config file |
| `PHANTOM_DB` | `/app/data/phantomdns.db` | SQLite database path |
| `PHANTOM_POLICIES` | `configs/policies.json` | Policy file path |
| `CORS_ORIGINS` | `http://localhost:3000` | Allowed CORS origins |
| `DNS_LISTEN_ADDR` | (from config) | Override DNS listen address |
| `BLOCKLIST_UPDATE_INTERVAL` | `6h` | Blocklist refresh interval |
| `HYDRA_API_URL` | `http://localhost:8080` | CLI/MCP API target |
| `HYDRA_TOKEN` | (none) | CLI auth token |
| `BLOCK_RESPONSE` | `zero` | Engine response for blocked domains: `zero` (A 0.0.0.0), `nxdomain` (RcodeNameError), `refused` (RcodeRefused). `zero` is the safe default; `nxdomain` is faster on Windows browsers but should be A/B tested first |
| `QUERY_LOG_RETENTION_DAYS` | `7` | Delete query logs older than N days; 0 disables |
| `QUERY_LOG_MAX_ROWS` | `1000000` | Keep at most N newest query-log rows (SD-card insurance); 0 disables |
| `QUERY_LOG_CLEANUP_INTERVAL` | `1h` | How often the retention loop runs |

### Compose port mapping (important for demos)

The dataplane listens on container port **1053**. The base compose maps it to host **:53**, but `docker-compose.override.yml` (gitignored, present locally) replaces that list to also expose **:5353** for WSL2/Windows hosts where :53 collides with `systemd-resolved` or the Windows DNS Client. Smoke-test paths:

```
curl http://localhost:8080/health                               # controlplane
dig @127.0.0.1 -p 5353 example.com                              # host-side, WSL-safe
docker exec hydradns-core-1 dig @127.0.0.1 -p 1053 example.com  # ground truth (always works)
```

`BLOCK_RESPONSE` env var (set in override) controls how the dataplane answers a blocked query: `zero` (default; A/AAAA → 0.0.0.0/::), `nxdomain`, or `refused`. See `respondBlocked` in `apps/core/cmd/dataplane`/`engine.go`.

## SQLite Setup

Pure-Go SQLite driver (`glebarez/sqlite`), WAL mode for concurrency, single-writer (`MaxOpenConns=1`). GORM auto-migrates all models on startup: Policy, DNSQuery, DomainPolicy, Action, Category, Statistics, SystemState, BlocklistSource, BlocklistSnapshot, BlocklistEntry, AdminCredential.

### Query-log retention

The `dns_queries` table is bounded by a background loop in the dataplane (`startQueryLogRetention`), runs once at startup then on `QUERY_LOG_CLEANUP_INTERVAL` (default 1h). Two limits, both via env:
- `QUERY_LOG_RETENTION_DAYS` (default 7) — delete rows older than N days; 0 disables.
- `QUERY_LOG_MAX_ROWS` (default 1,000,000) — keep at most N newest rows (SD-card insurance); 0 disables.

Note: SQLite `DELETE` reuses freed pages rather than shrinking the file, so the `.db` size settles at its high-water mark (bounded by retention) and does not auto-`VACUUM` — `VACUUM` is avoided deliberately because it locks the DB and would stall DNS. Compliance note: CERT-In wants 180-day retention; that conflicts with SD-card capacity at scale, so long-retention customers need a bigger disk or external log shipping (see `docs/certifications-roadmap.md`).

## Known Incomplete Features

Many control plane API handlers use **in-memory mock data**, not the real storage/engines:
- `handlers/blocklists.go` — mock slice, doesn't use the real blocklist engine or DB
- `handlers/policies.go` — mock slice, doesn't use the real policy engine
- `handlers/dns.go` — `/dns/resolvers` uses mock data

**What IS wired up**: `/dns/engine` GET/POST (real gRPC to dataplane), `/dns/metrics` (real gRPC), and the full blocklist DB storage layer (just not called from the API).

The UI has no API client layer yet — `NEXT_PUBLIC_API_URL` is set but unused in code.

The scanner currently only detects the system resolver via `/etc/resolv.conf` and runs a basic UDP resolution check.

## API Response Envelope

All control plane responses use a standard envelope:
```json
{"status": "success", "data": {...}, "error": "message if error"}
```

## Submodule Workflow

Each app is a separate Git repo (see `.gitmodules`). Clone with `git clone --recursive`. Work inside each `apps/<service>` directory and push to that service's repo. Run `make update` from root to sync. Submodule URLs all live under `github.com/hydradns/`.

## Docs & session retrospectives

`docs/` holds phase plans, critiques (`critique-NNN.md`), and `report.md` (rolling session log). New phases get a `phaseN-*-plan.md` before implementation and a `phaseN-*.md` retrospective after. `report.md` is updated at the end of each working session.

## CI

Two GitHub Actions workflows:
- `ci.yml` — on push/PR to main: vet + test core, vet + build CLI, lint + build dashboard, lint + build landing, Docker build verification
- `release.yml` — on tag push: multi-arch Docker images (amd64/arm64) to GHCR + cross-compiled CLI binaries

## Known Incomplete Features

- Regex/wildcard policy evaluation — parsed but not evaluated at query time
- Query log retention — DNSQuery table grows unbounded
- TLS on gRPC — uses `grpc.WithInsecure()`
- TLS on dashboard — HTTPS not terminated anywhere
- Port 53 binding on bare metal — needs `CAP_NET_BIND_SERVICE` (Docker handles this via port mapping)
- `/dns/resolvers` endpoint returns mock data (resolvers are config-only)
- Scanner only detects system resolver via `/etc/resolv.conf`
- Policy / blocklist edit UI — only create + delete today, no inline edit
- Query log pagination — hard-capped at ~100 entries, no paging UI
- Settings page — UI stub exists, no backend wiring
- CORS origins — defaults to `http://localhost:3000`; production needs `CORS_ORIGINS` env override
- Update mechanism — no `hydra update` or container self-update flow
- Remote monitoring — no heartbeat or alert pipeline for distributed Pis

## Known Live Bugs (caught in real stack)

- **`UNIQUE constraint failed: statistics.id`** in core logs during query counting. Causes intermittent stats increment failures. Reproducible on a fresh volume. _Fix landed on `apps/core feat/bypass-mitigations` branch._
- **Blocklist ingestion leaves `domains_count` at 0** after `POST /api/v1/blocklists`. Source row persists, fetch/parse pipeline never populates the snapshot. Seen with StevenBlack, OISD Big, URLhaus, AdGuard Tracking. _Fix landed on `apps/core feat/bypass-mitigations` branch (kicks off async UpdateSource on creation)._
