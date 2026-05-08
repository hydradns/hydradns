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

## Architecture

```
Root (orchestrator)
├── apps/core       — Go 1.24, Gin, gRPC, GORM/SQLite
│   ├── cmd/controlplane/   — Admin API (port 8080), auth middleware, handlers
│   ├── cmd/dataplane/      — DNS server (UDP/TCP on 1053), gRPC on 50051
│   ├── internal/           — blocklist, dnsengine, policy, storage, grpc, config
│   └── proto/              — Protobuf definitions (buf for codegen → internal/gen/proto/)
├── apps/ui         — Next.js 16, React 19, TypeScript, Tailwind v4, shadcn/ui (port 3000)
├── apps/landing    — Vite, React 18, TypeScript, Tailwind v3, shadcn/ui (port 3001)
├── apps/scanner    — Go, network scanning worker (no exposed port)
├── apps/cli        — Go + Cobra CLI (11 commands) + MCP server (8 tools)
│   ├── cmd/        — Cobra commands (status, engine, block, policies, blocklists, logs, mcp)
│   ├── api/        — Shared HTTP client for control plane API
│   └── mcp/        — MCP JSON-RPC 2.0 server
└── docker-compose.yml
    └── hydra-net bridge network
```

**Docker port mapping:** Core runs on 1053 internally, Docker maps 53→1053 on the host. API on 8080.

## Core Domain Concepts

### Intent vs Reality (Control Plane ↔ Data Plane)

The control plane maintains **desired state** in SQLite (source of truth). The data plane holds **actual runtime state**. When a change is made (e.g., toggle DNS engine), the control plane persists intent to DB, then applies it to the data plane via gRPC (`SetAcceptQueries`). Status endpoints return both desired and actual state combined. gRPC services defined in `proto/phantomdns/v1/status.proto`.

### DNS Query Pipeline (3-step, early exit)

1. **Blocklist check** — if domain is blocklisted → respond REFUSED immediately (hardest block)
2. **Policy evaluation** — Bloom filter for fast O(1) negative lookup, then exact domain match against `PolicySnapshot` (atomic rebuild on change). Multiple matches resolved by priority, then lexicographic ID
3. **Upstream forward** — pool-per-resolver with failover across all configured upstreams (5s timeout, 2 retries each)

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

## SQLite Setup

Pure-Go SQLite driver (`glebarez/sqlite`), WAL mode for concurrency, single-writer (`MaxOpenConns=1`). GORM auto-migrates all models on startup: Policy, DNSQuery, DomainPolicy, Action, Category, Statistics, SystemState, BlocklistSource, BlocklistSnapshot, BlocklistEntry, AdminCredential.

## API Response Envelope

All control plane responses use a standard envelope:
```json
{"status": "success", "data": {...}, "error": "message if error"}
```

## Submodule Workflow

Each app is a separate Git repo. Clone with `git clone --recursive`. Work inside each `apps/<service>` directory and push to that service's repo. Run `make update` from root to sync.

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
