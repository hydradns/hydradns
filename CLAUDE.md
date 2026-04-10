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
- `make build-docker` / `make destroy-docker` — container lifecycle

### UI (Next.js Dashboard) — `apps/ui/`
- `npm run dev` — dev server on port 3000
- `npm run build` / `npm run lint`

### Landing (Vite + React) — `apps/landing/`
- `npm run dev` — dev server on port 3001
- `npm run build` / `npm run lint`
- `npm run test` / `npm run test:watch` — Vitest with jsdom

### Scanner (Go) — `apps/scanner/`
- `make build` — compile scanner binary

## Architecture

```
Root (orchestrator)
├── apps/core       — Go 1.24, Gin, gRPC, GORM/SQLite
│   ├── cmd/controlplane/   — Admin API (port 8080)
│   ├── cmd/dataplane/      — DNS server (gRPC on 50051)
│   ├── internal/           — blocklist, dnsengine, policy, storage, grpc, metrics
│   └── proto/              — Protobuf definitions (buf for codegen → internal/gen/proto/)
├── apps/ui         — Next.js 16, React 19, TypeScript, Tailwind v4, shadcn/ui (port 3000)
├── apps/landing    — Vite, React 18, TypeScript, Tailwind v3, shadcn/ui (port 3001)
├── apps/scanner    — Go 1.25, network scanning worker (no exposed port)
├── apps/cli        — placeholder, not yet populated
└── docker-compose.yml
    ├── postgres (15-alpine, port 5432) — not used by core yet
    ├── redis (7-alpine, port 6379) — not used by core yet
    └── hydra-net bridge network
```

## Core Domain Concepts

### Intent vs Reality (Control Plane ↔ Data Plane)

The control plane maintains **desired state** in SQLite (source of truth). The data plane holds **actual runtime state**. When a change is made (e.g., toggle DNS engine), the control plane persists intent to DB, then applies it to the data plane via gRPC (`SetAcceptQueries`). Status endpoints return both desired and actual state combined. gRPC services defined in `proto/phantomdns/v1/status.proto`.

### DNS Query Pipeline (3-step, early exit)

1. **Blocklist check** — if domain is blocklisted → respond REFUSED immediately (hardest block)
2. **Policy evaluation** — Bloom filter for fast O(1) negative lookup, then exact domain match against `PolicySnapshot` (atomic rebuild on change). Multiple matches resolved by priority, then lexicographic ID
3. **Upstream forward** — pool-per-resolver with failover across all configured upstreams (5s timeout, 2 retries each)

Domain normalization: lowercase + strip trailing dot (e.g., `EXAMPLE.COM.` → `example.com`).

### Blocklist Engine

Sources are fetched with ETag support (304 skip), SHA256 checksum tracking, and atomic persistence (transaction wraps snapshot + entries + metadata). Multiple format parsers: hosts, domain-list, ads-list — selected by `format` field on `BlocklistSource`.

### Policy Format

JSON file at `configs/policies.json`. Array of policies with `id`, `action` (BLOCK/ALLOW/REDIRECT), `domains`, optional `regexes`, `priority` (higher wins). Regexes are compiled/validated on load but **not yet evaluated at query time** (TODO in code). Wildcards also parsed but not evaluated.

## Configuration

- **Config file**: `configs/config.yaml` — top-level `dataplane` and `controlplane` keys
- **DataPlane config**: `listen_addr` (UDP/TCP), `upstream_resolvers` (list with failover), `grpc_server` (port/addr)
- **Policy file**: `configs/policies.json` loaded from disk on dataplane startup
- **Config loaded as package singleton**: `config.DefaultConfig`
- **Environment**: `.env` file (gitignored) with fallback defaults. See `.env.example`.

## SQLite Setup

Pure-Go SQLite driver (`glebarez/sqlite`), WAL mode for concurrency, single-writer (`MaxOpenConns=1`). GORM auto-migrates all models on startup: Policy, DNSQuery, DomainPolicy, Action, Category, Statistics, SystemState, BlocklistSource, BlocklistSnapshot, BlocklistEntry.

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
{"status": "success|error", "data": {...}, "error": "message if error"}
```

## Submodule Workflow

Each app is a separate Git repo. Clone with `git clone --recursive`. Work inside each `apps/<service>` directory and push to that service's repo. Run `make update` from root to sync.

## CI (Core)

GitHub Actions on push/PR to main: golangci-lint, gosec security scan, tests on Go 1.21/1.22 with Codecov, Docker build verification, govulncheck.
