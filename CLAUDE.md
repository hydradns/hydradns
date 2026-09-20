# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

HydraDNS is a DNS-layer security and privacy gateway built as a single monorepo orchestrated by Docker Compose. Each service lives under `apps/` in this repository.

## Build & Run Commands

### Full Stack (root)
- `make setup` — one-time local setup (.env)
- `make start` — `docker compose up -d` (all services)
- `make stop` — `docker compose down`
- `make update` — `git pull --ff-only`
- `make logs` / `make core-logs` / `make ui-logs` / `make scanner-logs` — tail logs
- `make build-core` / `make build-ui` / `make build-scanner` — build individual services
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


### Scanner (Go) — `apps/scanner/`
- `make build` — compile scanner binary

### CLI (Go / Cobra) — `apps/cli/`
- `go build -o hydra` — produces the `hydra` binary at the package root
- `./hydra <command>` — `status`, `engine`, `block`, `unblock`, `blocklists`, `policies`, `metrics`, `logs`, `setup`, `login`, `setup-router`, `mcp`, `update`, `version`
- `./hydra mcp` — runs the MCP server: stdio JSON-RPC 2.0 by default, or `--http` for an HTTP transport (management traffic only, requires a bearer token via `--http-token`/`HYDRA_MCP_TOKEN`) for driving a fleet remotely. 14 tools registered in `apps/cli/mcp/server.go`: `get_status`, `toggle_engine`, `block_domain`, `unblock_domain`, `list_policies`, `list_blocklists`, `get_query_logs`, `get_metrics`, `create_policy`, `delete_policy`, `bulk_unblock`, `get_weekly_summary`, `explain_anomaly`, `compare_to_last_month`. Tool access is scoped by the `MCP_ROLE` env var (`admin` default, `operator` — can't `toggle_engine`, `reporter` — read-only tools only, per each tool's explicit registration flag in `toolRegistry()`, not a `get_`/`list_` name guess — `explain_anomaly` and `compare_to_last_month` are read-only too); see `apps/cli/mcp/roles.go`.
- `go test ./...` — Cobra command tests live next to the commands (e.g. `cmd/setup_router_test.go`)
- API client lives in `apps/cli/api/client.go` and talks to the controlplane on `:8080`
- The token file (`~/.hydra/token`) is written atomically (temp file, `fsync`, rename), file mode `0600` in a directory forced to `0700`, and a symlinked directory or token path is refused outright rather than followed (`apps/cli/cmd/token_store.go`).
- `api.New()` prints a one-line stderr warning when the configured API URL is plain `http://` to a non-local (non-loopback/private/link-local) address — the login password and bearer token would otherwise cross the network in cleartext with no indication (`apps/cli/api/client.go`, `warnIfInsecure`).
- The MCP server's `initialize` response reports the CLI's own `Version` (the same value `hydra version` prints, ldflags-overridable) as `serverInfo.version`, not a hardcoded string (`apps/cli/mcp/server.go`, `apps/cli/cmd/mcp.go`).

## Architecture

HydraDNS is a single monorepo (no Git submodules, no `.gitmodules`). Go services share
a workspace via the root `go.work` (`apps/core`, `apps/cli`, `apps/scanner` — `apps/ui`
is Next.js/npm and isn't part of the Go workspace). The marketing/landing site used to
live here as `apps/landing`; it has moved out to the separate `hydradns/hydradns-landing`
repo and no longer exists in this tree.

```
Root (orchestrator)
├── apps/core       — Go 1.24, Gin, gRPC, GORM/SQLite
│   ├── cmd/controlplane/   — Admin API (host :8080)
│   ├── cmd/dataplane/      — DNS server; gRPC on :50051 (internal only, not published)
│   ├── internal/           — blocklist, dnsengine, policy, storage, grpc, metrics
│   └── proto/              — Protobuf definitions (buf for codegen → internal/gen/proto/)
│   NOTE: in production compose, controlplane + dataplane run as one combined `core` container
├── apps/ui         — Next.js 16, React 19, TypeScript, Tailwind v4, shadcn/ui (port 3000)
├── apps/scanner    — Go 1.25, network scanning worker (no exposed port; commented out in compose)
├── apps/cli        — Cobra CLI + MCP server (stdio or HTTP transport; talks to controlplane API)
├── docker-compose.yml — only `core` and `ui` are active; scanner is commented out, landing was
│                        removed entirely (it now builds/deploys from its own repo)
├── go.work                     — Go workspace covering apps/core, apps/cli, apps/scanner
├── docs/                       — user-facing docs; docs/internal/ (gitignored) — phase plans, critiques, report.md
└── scripts/                    — setup.sh, install.sh, stress-test.sh
```

There is **no postgres or redis service** in compose. `core` writes to a Docker volume (`core-data`) backed by SQLite.

Note: this worktree has no `docker-compose.override.yml` (it's gitignored and operator-provided);
the local-dev WSL2 workaround it describes elsewhere in this file is a convention, not something
guaranteed present in a fresh clone.

## Core Domain Concepts

### Intent vs Reality (Control Plane ↔ Data Plane)

The control plane maintains **desired state** in SQLite (source of truth). The data plane holds **actual runtime state**. When a change is made (e.g., toggle DNS engine), the control plane persists intent to DB, then applies it to the data plane via gRPC (`SetAcceptQueries`). Status endpoints return both desired and actual state combined. gRPC services defined in `proto/hydradns/v1/status.proto`.

### DNS Query Pipeline (early exit)

Every query first runs through the heuristic threat detector (`internal/threat`, entropy +
DGA-pattern + length + subdomain-depth scoring), non-blocking and score-only — it never
blocks a query by itself, it only tags the eventual query-log row `IsSuspicious`/`flagged`
(`internal/dnsengine/engine.go`). There is no NXDOMAIN-burst or fast-flux heuristic, and
there's no auto-block-by-threshold — flagged-but-allowed queries still resolve.

Then, in order, with early exit:

0. **DoH/DoT bootstrap interception** — known DoH provider bootstrap hostnames (Cloudflare,
   Google, Quad9, etc. — `internal/dnsengine/doh_bootstrap.go`) always get NXDOMAIN,
   regardless of `BLOCK_RESPONSE`, so the browser falls back to system DNS. Invisible to the
   dashboard, not user-editable.
1. **Blocklist check** — in-memory membership test (`internal/blocklist/memory.go`, `MemoryChecker`: atomic `map[string]struct{}` of all blocked domains + parent-domain walk). Built from enabled sources only and rebuilt when the dataplane's blocklist signature poll sees a change (`BLOCKLIST_POLL_INTERVAL`, default 5s) or after a source refresh; the DNS hot path never hits the DB. If blocked → respond per `BLOCK_RESPONSE`. (Historical note: this check used to run a per-query SQL `COUNT`, which capped throughput at ~500 QPS — see `docs/internal/stress-test-plan.md` T1.)
2. **Policy evaluation** — Bloom filter for fast O(1) negative lookup, then exact domain match against `PolicySnapshot` (atomic rebuild on change). Multiple matches resolved by priority, then lexicographic ID
3. **Response cache** — TTL-respecting LRU (20k entries, `internal/dnsengine/cache.go`); only allowed queries are cached, never blocked/redirect responses
4. **Upstream forward** — pool-per-resolver with failover across all configured upstreams (1.5s per-attempt timeout, 2 retries each)

Query logging and statistics are written off the hot path by a bounded batched writer (`internal/dnsengine/querylog_writer.go`): non-blocking enqueue, single drain goroutine, batched inserts. Drops (counted) when its 4096-deep queue is full so logging can never stall resolution or grow memory without bound.

Domain normalization: lowercase + strip trailing dot (e.g., `EXAMPLE.COM.` → `example.com`).

### Blocklist Engine

Sources are fetched with ETag support (304 skip), SHA256 checksum tracking, and atomic persistence (transaction wraps snapshot + entries + metadata). Three format parsers, keyed by the `format` field on `BlocklistSource` (`apps/core/internal/blocklist/parser/`): `hosts`, `adblock`, `domains`. Blocklists auto-refresh on a configurable interval (default 6h, env: `BLOCKLIST_UPDATE_INTERVAL`); creating a source via the API also triggers an immediate async fetch so `domains_count` doesn't sit at 0 waiting for the next cycle.

The source URL must be `http://` or `https://` (`validateBlocklistURL`, `cmd/controlplane/handlers/blocklists.go`); enforced identically on create, update, and the setup wizard's optional blocklist bootstrap. Each new snapshot fully replaces a source's entries in one transaction (delete-then-insert, `SaveSnapshotWithEntries`) — a domain removed upstream stops being blocked after the next fetch, and editing a source's URL only takes effect once that next (immediately-triggered) download completes, not synchronously in the API response. Only the last 10 `BlocklistSnapshot` metadata rows are kept per source (`snapshotRetentionPerSource`, oldest pruned); this caps metadata only, never the live entries. If a dataplane rebuild of the in-memory set fails, it keeps its previous (still-enforced) contents and retries on the next `BLOCKLIST_POLL_INTERVAL` tick rather than going empty (`cmd/dataplane/blocklist_reload.go`).

The fetcher (`internal/blocklist/fetcher/http_client.go`) is a plain `http.Client` with no `CheckRedirect` override, so it follows redirects to any `http(s)` host, including private/internal addresses — there is no SSRF protection. The threat model assumes whoever adds a blocklist source URL is trusted; see `docs/limitations.md`.

### Policy Format

JSON file at `configs/policies.json`. Array of policies with `id`, `action` (BLOCK/ALLOW/REDIRECT), `domains`, optional `regexes`, `priority` (higher wins). Regexes are compiled/validated on load but **not yet evaluated at query time**. Wildcards also parsed but not evaluated.

The API additionally validates `action` on both create and update (`validatePolicyAction`, `cmd/controlplane/handlers/policies.go`): it must be `BLOCK`, `ALLOW`, or `REDIRECT` (case-insensitive), and `REDIRECT` requires a non-empty, parseable `redirect_ip`. This closes what used to be a create-only check — an update used to accept any string and silently turn a policy into a no-op.

### Authentication & RBAC

Multi-user, role-based. The old single-admin `AdminCredential` singleton has been replaced
by a `User` model (`internal/storage/models/user.model.go`) with three roles:
`admin`, `operator`, `read_only` (constants `RoleAdmin`/`RoleOperator`/`RoleReadOnly`).
Auth is per-user bearer tokens (SHA-256 hashed at rest, plaintext shown once on creation),
not a single shared API key — see `internal/storage/models/token.model.go` and
`cmd/controlplane/handlers/tokens.go`. Password hashed with bcrypt.

**Role boundary:** `admin` bypasses every role check. Read endpoints are open to any
authenticated user, including `read_only`. Mutating endpoints (`POST`/`DELETE` on
`/dns/engine`, `/policies`, `/blocklists`) are wrapped in `middlewares.RequireRole(RoleOperator)`
(`cmd/controlplane/middlewares/auth.go`), so `operator` is effectively "operator or admin"
and `read_only` cannot write. `GET /audit` requires `operator` or above (read_only is denied —
audit history is treated as sensitive). User management (`POST/DELETE /users`,
`POST /users/:id/disable`) is admin-only; any user can `PATCH` their own email/password but
not their own role. Every mutating handler writes an `AuditEvent` row via
`cmd/controlplane/audit`.

`AdminCredential` still exists in the schema, but only to support
`migrateAdminSingletonToUser` (`internal/storage/db/db.go`): on first boot of a build that
knows about RBAC, an existing singleton admin is copied into a `User` + a non-expiring
`Token` so pre-RBAC installs keep working through the upgrade.

**Auth flow:** First boot → `POST /api/v1/auth/setup` creates the first `admin` User and
mints a non-expiring token (also optionally seeds blocklist sources). Subsequent access →
`POST /api/v1/auth/login` with email + password → a 90-day bearer token
(`repositories.DefaultTokenExpiry`). Dashboard stores the token in localStorage + cookie.
Login without an `email` field falls back to "the one user" if exactly one exists
(backward-compat with the pre-RBAC single-admin flow); it 401s otherwise.

**Auth endpoints (unprotected):**
- `GET /api/v1/auth/status` — returns `{setup_complete: bool}` (true once any `User` row exists)
- `POST /api/v1/auth/setup` — creates the first admin user, optionally configures blocklists; 409 once setup is complete
- `POST /api/v1/auth/login` — validates credentials, returns a token

Both `/auth/login` and `/auth/setup` are throttled per client IP (fixed window, 10 attempts
/ 5 minutes by default, shared across the two endpoints; 100 attempts / 5 minutes when
`HYDRA_DEMO_MODE=true`), so the correctness of the throttle depends on `TRUSTED_PROXIES`
being set correctly behind a reverse proxy (see the Configuration table below) — otherwise
every request behind that proxy is bucketed under one IP. Only a failed attempt (HTTP 4xx —
bad credentials, a malformed request, setup-already-done) counts against the budget; a
successful login/setup and a 5xx (the server's own fault) do not. The counter is in-memory
and resets on restart.

Not implemented: per-user MFA/TOTP, SSO/OIDC/SAML, session timeout beyond the 90-day token
expiry, and a CLI for user/token management (dashboard-only today, via `/api/v1/users` and
`/api/v1/tokens`).

## Configuration

- **Config file**: `configs/config.yaml` — top-level `dataplane` and `controlplane` keys
- **DataPlane config**: `listen_addr` (UDP/TCP), `upstream_resolvers` (list with failover), `grpc_server` (port/addr), `blocklist_update_interval`
- **Policy file**: `configs/policies.json` loaded from disk on dataplane startup
- **Config loaded as package singleton**: `config.DefaultConfig` with env var overrides (`DNS_LISTEN_ADDR`, `BLOCKLIST_UPDATE_INTERVAL`)
- **Environment**: `.env` file (gitignored) with fallback defaults. See `.env.example`.

| Env Variable | Default | Description |
|:-------------|:--------|:------------|
| `HYDRA_CONFIG` | `/app/configs/config.yaml` | Path to config file (`internal/config/config.go`) |
| `HYDRA_DB` | `/app/data/hydradns.db` (`db.DefaultDBPath`) | SQLite database path. Resolved via `db.ResolveDBPath` (`internal/storage/db/dbpath.go`), which falls back to a pre-rename `phantomdns.db` in the same directory if that's the only DB file an existing install has — a one-time upgrade path, not the default for new installs |
| `HYDRA_POLICIES` | `/app/configs/policies.json` | Policy file path (`cmd/dataplane/main.go`) |
| `CORS_ORIGINS` | `http://localhost:3000,http://127.0.0.1:3000` | Comma-separated allowed CORS origins (`cmd/controlplane/middlewares/cors.go`). The shipped `docker-compose.yml` sets `http://localhost:3000`. `*` still works but logs a warning |
| `CORS_ALLOW_SAME_HOST` | `true` | Also allow an Origin whose hostname equals the request's Host hostname when that hostname is an IP literal or `localhost` (dashboard opened by LAN IP). Named hosts need a `CORS_ORIGINS` entry. Set `false` to disable |
| `DNS_LISTEN_ADDR` | (from config, normally `0.0.0.0:1053`) | Override DNS listen address. Not forwarded by `docker-compose.yml` on purpose — the host-facing DNS port mapping there is fixed at `53:1053/...`, so changing this alone inside the container would only break that mapping, not move it |
| `BLOCKLIST_UPDATE_INTERVAL` | `6h` | How often blocklist sources are re-downloaded |
| `HYDRA_DEMO_MODE` | `false` | Public read-only demo: a `DemoGuard` middleware rejects every mutation before auth, a `read_only` demo user and synthetic data are seeded (`cmd/controlplane/demoseed`), client IPs are masked in responses. Refuses to start on a database that has real users. See `demo/README.md` |
| `BLOCKLIST_POLL_INTERVAL` | `5s` | How often the dataplane checks the DB for blocklist changes (add, toggle, delete, finished download) and rebuilds the in-memory set; `0` disables |
| `TRUSTED_PROXIES` | (empty) | Comma-separated CIDRs/IPs allowed to set `X-Forwarded-For` for client-IP purposes (login/setup throttle, audit log). Empty means no proxy is trusted — `c.ClientIP()` always resolves to the real socket address (`cmd/controlplane/main.go`) |
| `HYDRA_API_URL` | `http://localhost:8080` | CLI/MCP API target |
| `HYDRA_TOKEN` | (none) | CLI/MCP bearer token; if unset the CLI also tries `~/.hydra/token` (`apps/cli/cmd/root.go`) |
| `HYDRA_MCP_TOKEN` | (none) | Bearer token required by `hydra mcp --http` (or use `--http-token`) |
| `MCP_ROLE` | `admin` | Scopes which MCP tools a caller may invoke: `admin` (all), `operator` (all but `toggle_engine`), `reporter` (tools flagged read-only in `toolRegistry()`, an explicit per-tool list, not a `get_`/`list_` name guess). Unrecognized values safe-default to `reporter` (`apps/cli/mcp/roles.go`) |
| `HYDRA_UPDATE_URL` | (built-in release feed) | Override the release feed `hydra update` checks (`apps/cli/cmd/update.go`) |
| `HYDRA_ANONYMIZE_CLIENT_IPS` | `false` | Opt-in: hash client IPs (HMAC-SHA256, truncated to 64 bits) before writing them to the query log, instead of storing them as-is. Off by default — per-device visibility in the query log is treated as a core feature (`internal/config/config.go`, `internal/dnsengine/anonymize.go`) |
| `HYDRA_ANON_SECRET` | (generated per-install) | HMAC key for `utils.AnonymizeIP`, only used when anonymization is enabled |
| `BLOCK_RESPONSE` | `zero` | Engine response for blocked domains: `zero` (A 0.0.0.0), `nxdomain` (RcodeNameError), `refused` (RcodeRefused). `zero` is the safe default; `nxdomain` is faster on Windows browsers but should be A/B tested first |
| `QUERY_LOG_RETENTION_DAYS` | `7` | Delete query logs older than N days; 0 disables |
| `QUERY_LOG_MAX_ROWS` | `1000000` | Keep at most N newest query-log rows (SD-card insurance); 0 disables |
| `QUERY_LOG_CLEANUP_INTERVAL` | `1h` | How often the retention loop runs |
| `NEXT_PUBLIC_API_URL` | `http://localhost:8080` | Dashboard API base URL override, inlined at build time. Left at the default, the dashboard derives the API URL at runtime from the page's own protocol and hostname on port 8080 (`apps/ui/lib/api-base.ts`); set it only for a reverse proxy or a non-default API host/port |
| `NEXT_PUBLIC_SHOW_BYPASS_PANEL` | unset (hidden) | Build-time flag to show the DoH-bypass-attempts panel on the dashboard; set to `true` for technical/internal deployments (`apps/ui/app/dashboard/page.tsx`) |

The shipped `docker-compose.yml` forwards `BLOCK_RESPONSE`, `BLOCKLIST_UPDATE_INTERVAL`,
`BLOCKLIST_POLL_INTERVAL`, `QUERY_LOG_RETENTION_DAYS`, `QUERY_LOG_MAX_ROWS`,
`QUERY_LOG_CLEANUP_INTERVAL`, `HYDRA_ANONYMIZE_CLIENT_IPS`, `HYDRA_ANON_SECRET`, and
`TRUSTED_PROXIES` from `.env` into the `core` container's environment, each defaulting to
the code's own default when unset. `DNS_LISTEN_ADDR` is the one exception (see its row
above). `HYDRA_DEMO_MODE` is deliberately never forwarded by this file — demo mode has its
own `demo/docker-compose.demo.yml`.

Boolean env vars (`HYDRA_DEMO_MODE`, `CORS_ALLOW_SAME_HOST`, `HYDRA_ANONYMIZE_CLIENT_IPS`) all
parse through one shared function, `config.ParseBoolEnvValue`/`MustParseBoolEnv`
(`internal/config/config.go`): accepted spellings are `true/1/yes/on` and `false/0/no/off`,
case-insensitive, whitespace-trimmed; an unset or empty value keeps the default.
`HYDRA_DEMO_MODE` and `CORS_ALLOW_SAME_HOST` go through `MustParseBoolEnv`, so an
unrecognized value (a typo, `"1 "` with unexpected characters, etc.) is fatal at startup,
naming both the variable and the value in the error.
`HYDRA_ANONYMIZE_CLIENT_IPS` is parsed the same way but does not use `MustParseBoolEnv` (it
runs inside `config.DefaultConfig`'s package-level initializer, before `FatalFunc` can be
overridden in a test); an unrecognized value there logs a warning and keeps the previously
configured value instead of exiting.

`CORS_ORIGINS` entries are split on commas, trimmed, and each validated as either `*` or an
`http(s)://host` with no path (`cmd/controlplane/middlewares/cors.go`); an invalid entry stops
startup with an error naming `CORS_ORIGINS` and the offending entry, instead of reaching
gin-contrib/cors (which panics). Setting it to `*` disables `Access-Control-Allow-Credentials`
(a wildcard origin with credentials is invalid per the CORS spec; this API only ever
authenticates via a Bearer header, never cookies) and logs a startup warning.

Query-log IP note: client IP anonymization is implemented and wired into `Engine.logQuery`
(`internal/dnsengine/engine.go`, `internal/dnsengine/anonymize.go`) but **off by default**.
With `HYDRA_ANONYMIZE_CLIENT_IPS` unset (or `dataplane.anonymization.enabled: false`, the
shipped default), client IP addresses are stored as-is in `dns_queries` — per-device activity
is visible in the query log, which is treated as the correct default for a home/office DNS
firewall. Enabling it hashes the IP (HMAC-SHA256 of the raw address, not a subnet mask —
masking first would collapse every device on one LAN's /24 onto the same hash, see the
comment on `utils.AnonymizeIP`) with a per-install secret so a device's own queries still
group together without exposing the raw address.

### Compose port mapping (important for demos)

The dataplane listens on container port **1053**. The base compose maps it to host **:53**. Some operators add a gitignored `docker-compose.override.yml` to also expose **:5353** for WSL2/Windows hosts where :53 collides with `systemd-resolved` or the Windows DNS Client — that file is not part of this repo and isn't present in a fresh checkout. Smoke-test paths:

```
curl http://localhost:8080/health                               # controlplane
dig @127.0.0.1 -p 5353 example.com                              # host-side, WSL-safe
docker exec hydradns-core-1 dig @127.0.0.1 -p 1053 example.com  # ground truth (always works)
```

`BLOCK_RESPONSE` env var controls how the dataplane answers a blocked query: `zero` (default; A/AAAA → 0.0.0.0/::), `nxdomain`, or `refused`. See `respondBlocked` in `apps/core/internal/dnsengine/engine.go`.

## SQLite Setup

Pure-Go SQLite driver (`glebarez/sqlite`), WAL mode for concurrency, single-writer (`MaxOpenConns=1`). `busy_timeout` is set to 30 seconds via `PRAGMA busy_timeout=30000;` (`internal/storage/db/db.go`), so the controlplane and dataplane — which both call `db.InitDB`/`AutoMigrate` independently against the same file and can start at the same time in the combined `core` container — wait for each other instead of one failing with `SQLITE_BUSY`. GORM auto-migrates all models on startup, run by both processes (`internal/storage/db/db.go`): Policy, DNSQuery, DomainPolicy, Action, Category, Statistics, SystemState, BlocklistSource, BlocklistSnapshot, BlocklistEntry, AdminCredential (legacy, migration-only), User, Token, AuditEvent.

Two indexes — `dns_queries.action` and `blocklist_entries.source_id` — are new as of this branch (not present on `origin/main`). On a fresh install this is instant; on an existing large database, `CREATE INDEX` runs synchronously at startup and can make the first start after upgrading noticeably slower than a normal restart, especially on SD-card storage. See the Upgrade notes in `CHANGELOG.md` and `docs/releasing.md`.

### Query-log retention

The `dns_queries` table is bounded by a background loop in the dataplane (`startQueryLogRetention`), runs once at startup then on `QUERY_LOG_CLEANUP_INTERVAL` (default 1h). Two limits, both via env:
- `QUERY_LOG_RETENTION_DAYS` (default 7) — delete rows older than N days; 0 disables.
- `QUERY_LOG_MAX_ROWS` (default 1,000,000) — keep at most N newest rows (SD-card insurance); 0 disables.

Note: SQLite `DELETE` reuses freed pages rather than shrinking the file, so the `.db` size settles at its high-water mark (bounded by retention) and does not auto-`VACUUM` — `VACUUM` is avoided deliberately because it locks the DB and would stall DNS. Compliance note: CERT-In wants 180-day retention; that conflicts with SD-card capacity at scale, so long-retention customers need a bigger disk or external log shipping (see `docs/internal/certifications-roadmap.md`).

## Known Incomplete Features

Re-verified directly against this monorepo's code (no more pinned-submodule vs. upstream
split — everything below is main, checked at the branch point in this worktree).

- Regex/wildcard policy evaluation — still parsed and validated on load but not evaluated
  at query time; `Engine.Evaluate` only does exact + parent-domain matching
  (`apps/core/internal/policy/engine.go`, see the `TODO: wildcard/regex` comment)
- TLS on gRPC — still `grpc.WithInsecure() // TLS later` (`apps/core/internal/grpc/controlplane/client.go`)
- TLS on dashboard — no HTTPS termination anywhere in compose or the Next.js server
- Encrypted upstream/listener DNS (DoH/DoT) — upstream resolvers are plain UDP (`8.8.8.8:53`
  style) and there is no DoH/DoT listener; the only DoH-related code is the *bootstrap
  blocklist* that NXDOMAINs known DoH provider hostnames to keep clients on plain DNS
  (`apps/core/internal/dnsengine/doh_bootstrap.go`) — that is bypass mitigation, not
  encrypted DNS support
- DNSSEC validation — not implemented anywhere in `internal/dnsengine`
- Port 53 binding on bare metal — needs `CAP_NET_BIND_SERVICE` (Docker handles this via port mapping)
- `/dns/resolvers` — reads real upstream resolvers from `config.DefaultConfig`, but is
  read-only; no CRUD (`apps/core/cmd/controlplane/handlers/dns.go`, `ListResolvers`)
- Scanner only detects the system resolver via `/etc/resolv.conf` and runs a basic UDP resolution check
- Blocklist rebuild cost — blocklist create/toggle/delete/URL edits are live within about 5s of
  the change (or of the new download finishing) via the signature poll
  (`cmd/dataplane/blocklist_reload.go`); each source's entries are replaced per snapshot
  (`SaveSnapshotWithEntries`). The rebuild is still a full read of all enabled entries and holds
  the dataplane's single SQLite connection (`MaxOpenConns=1`) while it runs, so query-log writes
  queue behind it
- Resolver CRUD — the dashboard client has create/update/delete calls for `/dns/resolvers` but
  the control plane only serves `GET`; those calls 404
- Dashboard recent-activity feed — `GetAnalyticsSummary` returns the newest 100 rows unpaginated;
  the Logs page uses `GET /analytics/logs` with server-side paging and filters
- Bypass attempts — `GET /analytics/bypass` reports every attempt as protocol `doh`; the dataplane
  sees only the bootstrap hostname lookup and cannot tell DoH from DoT or DoQ
- Settings page — dashboard page exists (`apps/ui/app/dashboard/settings/page.tsx`) and calls
  `getSettings`/`updateSettings`, but there is no `/settings` route or handler anywhere on
  the control plane — backend wiring is still absent
- CORS origins — explicit allowlist plus an automatic same-host rule for IP literals and
  `localhost`; a dashboard reached by a named host (`pi.local`, a reverse-proxy domain) needs
  its origin added to `CORS_ORIGINS`
- Container self-update — none for the `core`/`ui` containers; the CLI does have `hydra
  update` self-update for the `hydra` binary itself (`apps/cli/cmd/update.go`)
- Remote monitoring — no heartbeat/alert pipeline anywhere in `apps/core`
- Query-log client IPs are stored raw by default — anonymization exists and works
  (`HYDRA_ANONYMIZE_CLIENT_IPS`, see the Configuration table above) but is opt-in, off by
  default
- Per-user MFA/TOTP, SSO/OIDC/SAML — not implemented; RBAC (see Authentication above) covers
  roles and per-user tokens only
- CLI has no `hydra users`/`hydra tokens` commands — user/token management is dashboard- or
  API-only today

The UI has a full API client (`lib/api.ts`, `lib/auth.ts`) — all dashboard pages poll the real API.

## API Response Envelope

All control plane responses use a standard envelope:
```json
{"status": "success", "data": {...}, "error": "message if error"}
```


## Docs & session retrospectives

`docs/internal/` (gitignored, local only) holds phase plans, critiques (`critique-NNN.md`), and `report.md` (rolling session log). New phases get a `phaseN-*-plan.md` before implementation and a `phaseN-*.md` retrospective after. `report.md` is updated at the end of each working session.

## CI

Two GitHub Actions workflows (`.github/workflows/`):
- `ci.yml` — on push/PR to main, four jobs: `core` (Go vet + `go test -race` + build controlplane
  and dataplane binaries), `cli` (Go vet + test + build), `dashboard` (Next.js `npm run lint` +
  `npm test` + `npm run build`), `docker` (builds the `core` and `ui` Docker images, depends on
  the other three)
- `release.yml` — on tag push (`v*`): multi-arch (linux/amd64, linux/arm64) Docker images for
  `core` and `ui` pushed to GHCR, plus a separate `release-cli` job that cross-compiles the
  `hydra` binary for linux/darwin × amd64/arm64 and attaches them to the GitHub release

## Known Live Bugs (caught in real stack)

- Resolved: the dashboard's vitest suite (`apps/ui`) previously failed 28 of 47 tests, but
  this was a Node-version artifact, not a product bug — Node 22+ defines global
  `localStorage`/`sessionStorage` that shadow jsdom's under Vitest. Fixed in
  `apps/ui/vitest.setup.ts`; the suite is 77/77 (14 files) as of this branch, and the
  `dashboard` CI job now runs `npm test`.
- The `statistics.id` UNIQUE-constraint bug and blocklist ingestion leaving `domains_count`
  at 0 are both fixed: query counting no longer collides (see `internal/storage/repositories`)
  and `CreateBlocklist`/the setup wizard both trigger an immediate async
  `BlocklistEngine.UpdateSource` fetch on create.
- The default `apps/core/configs/policies.json` no longer ships any block-google/block-apple
  test policy; it seeds only `block-ads` and `block-malware` example policies.

No other known live bugs as of this worktree's branch point.
