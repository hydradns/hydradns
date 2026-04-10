# HydraDNS Progress Report

> This file tracks progress, decisions, and lessons learned across all phases.  
> Updated after each work session.

---

## Project Status

| Phase | Status | Date Started | Date Completed |
|---|---|---|---|
| Phase 1: Solid Core | Complete | 2026-04-05 | 2026-04-05 |
| Phase 2: The Dashboard | Complete | 2026-04-07 | 2026-04-07 |
| Phase 3: Plug-and-Play Deployment | Complete | 2026-04-07 | 2026-04-07 |
| Phase 4: CLI + MCP | Complete | 2026-04-07 | 2026-04-07 |
| Phase 5: Polish & Ship | Complete | 2026-04-07 | 2026-04-07 |
| Tier 1: Demo-Ready | Complete | 2026-04-08 | 2026-04-08 |
| Tier 2: Demo Polish | Complete | 2026-04-08 | 2026-04-08 |
| E2E Verification | Complete | 2026-04-08 | 2026-04-08 |

---

## Phase 1 Summary (2026-04-05)

**Goal:** Make the DNS engine correct, reliable, and testable.

### What was done
- Code review of entire `apps/core` (67 Go files)
- Fixed 10 bugs (6 critical, 4 important) — see [phase1-core-review.md](phase1-core-review.md)
- Wrote 52 unit/integration tests across 5 packages — see [phase1b-tests-and-handlers.md](phase1b-tests-and-handlers.md)
- Wired all mock handlers to real SQLite storage
- Added PolicyRepository and extended BlocklistRepository
- Verified with live dig tests (blocklist block, policy block, upstream forward all work)

### Key metrics
- **Tests:** 0 → 52 (5 packages)
- **Mock handlers:** 6 → 0
- **Bugs fixed:** 10
- **DNS query verified working:** blocklist block (REFUSED), policy block (REFUSED), upstream forward (NOERROR)

### Key decisions
| Decision | Why |
|---|---|
| SQLite stays (not Postgres) | Single-writer is fine for v1, simpler deployment on Pi |
| Resolvers are config-only | Infrastructure config, not user data — no CRUD needed |
| Policies store domains as JSON text | Simple for CRUD, no need for per-domain querying yet |
| In-memory SQLite for tests | Fast, tests real SQL, no mock complexity |
| Env var overrides for paths | Docker uses `/app/...`, local dev uses relative paths |

### Lessons learned (carry forward)
1. Every DNS handler code path must send a response — never return silently
2. Normalize domains once at entry point, not in each subsystem
3. Single source of truth for migrations (db.go only)
4. Hardcoded container paths need env var escape hatches for local dev
5. Mock interfaces at system boundaries, not internal seams

---

## Phase 2 Summary (2026-04-07)

**Goal:** Non-technical users can manage HydraDNS from the browser.

### What was done
- Created API client layer (`lib/api.ts` + `lib/types.ts`) with typed functions for all 11 endpoints
- Built 5 dashboard pages: Home, DNS Engine, Blocklists, Policies, Query Logs — see [phase2-dashboard.md](phase2-dashboard.md)
- Extracted shared dashboard layout with sidebar
- Rewrote KPI cards from financial to DNS stats
- Removed 12 unused mock/financial components

### Key metrics
- **Pages:** 0 functional → 5 (all wired to real API)
- **Components removed:** 12 (mock financial data)
- **New files:** 9 (2 lib, 1 layout, 5 pages, 1 modified component)
- **Build:** Clean, 0 errors, 0 lint warnings

### Key decisions
| Decision | Why |
|---|---|
| Native fetch, not SWR/React Query | Minimal deps, no new packages |
| Inline forms, not modals | No dialog component, simpler UX |
| Auto-refresh via setInterval | Real-time feel without WebSockets |
| Shared layout.tsx | DRY sidebar across all dashboard pages |

### Lessons learned
1. Extract layout early — sidebar should be in layout.tsx from the start
2. Type API responses first — catches field name mismatches before page code
3. Delete aggressively — 12 files of financial mock were noise
4. CORS config will bite on deployment — hardcoded localhost needs env config

---

## Phase 3 Summary (2026-04-07)

**Goal:** `docker compose up` works, deployable to Pi.

### What was done
- Combined controlplane + dataplane into single core container (avoids SQLite sharing issues)
- Created entrypoint.sh that runs both processes with proper shutdown handling
- Fixed config.yaml: port mismatch (8086→8080), gRPC address (dataplane→localhost)
- Made CORS configurable via `CORS_ORIGINS` env var
- Created Dockerfiles for all 4 services (core, ui, scanner, landing)
- Fixed Next.js config conflict (.mjs overriding .ts)
- Fixed system state default (DNSEnabled was false, causing all queries to be REFUSED)
- Rewrote docker-compose.yml with healthchecks, volumes, correct ports
- See [phase3-deployment.md](phase3-deployment.md) for details

### Key metrics
- **Dockerfiles created:** 5 (core combined, ui, scanner, landing, entrypoint)
- **Config bugs found & fixed:** 3 (port mismatch, gRPC address, system state default)
- **Core container verified:** health check, API endpoints, DNS block + forward all working
- **Tests:** 52 still passing after default change

### Bug found during testing
System state defaults to `DNSEnabled=false`. On fresh DB, controlplane tells dataplane to stop. Our Phase 1 fix (REFUSED instead of silent drop) made this visible — all queries got REFUSED. Fixed by defaulting to `true`.

### Lessons learned
1. Combined container > split for SQLite — can't share SQLite across container boundaries
2. Default system state to enabled — fresh install should work out of the box
3. Test in container, not just locally — pre-existing local DB masked the default state bug
4. Next.js .mjs config overrides .ts — remove one or the other

---

## Critique #001 Fixes (2026-04-07)

**Trigger:** Automated code critique agent reviewed all 3 phases. Found 10 actionable issues.

### Critical fixes applied
| Fix | Impact |
|---|---|
| DefaultConfig nil panic — returns sensible defaults when config missing | Prevented crash on startup without config file |
| Entrypoint race — polls /proc/net/tcp for gRPC port instead of sleep 2 | Reliable startup sequencing |
| Blocklist fetch async — moved to goroutine, DNS serves immediately | Startup <1s instead of 30s, no longer fails if GitHub is down |
| Sidebar active route — uses `usePathname()` | Correct page highlighted in nav |
| API fetch timeout — 10s AbortController | No more infinite hangs when API is down |

### High-priority fixes applied
| Fix | Impact |
|---|---|
| Blocklist N+1 → batch COUNT query | Single SQL query instead of N queries for N sources |
| DeleteSource deferred rollback | Transaction safety guaranteed |
| Dashboard error handling — Promise.all with catch | Both API calls handled, errors shown to user |
| Config secret → env var placeholder | No more hardcoded secret in committed config |

### Bug found during verification
Async blocklist fetch revealed that DNS server starts and serves within <1s. Previously blocked 30s+ on startup. The gRPC port poll in entrypoint correctly detects readiness.

See [critique-001.md](critique-001.md) for full list of findings.

---

## Phase 4 Summary (2026-04-07)

**Goal:** CLI and MCP server for terminal and AI-driven management.

### What was done
- Built Go CLI (`hydra`) with 11 commands using Cobra: status, engine (enable/disable), metrics, block, unblock, policies (delete), blocklists (add/delete), logs, mcp
- Created shared API client (`api/client.go`) with 13 typed methods wrapping control plane REST API
- Built MCP server (`mcp/server.go`) implementing JSON-RPC 2.0 over stdio with 8 tools
- `hydra mcp` command starts MCP session — compatible with Claude Code and other MCP clients

### Key metrics
- **CLI commands:** 11 (including subcommands)
- **MCP tools:** 8 (get_status, toggle_engine, block_domain, unblock_domain, list_policies, list_blocklists, get_query_logs, get_metrics)
- **New files:** 12 (1 main, 1 api client, 9 cmd files, 1 mcp server)
- **Binary size:** ~9.5MB
- **Build:** Clean, zero errors

### Key decisions
| Decision | Why |
|---|---|
| Custom MCP handler, no SDK | JSON-RPC 2.0 over stdio is simple enough to implement directly |
| Shared API client | CLI commands and MCP tools use the same typed HTTP client |
| Single binary, two modes | `hydra` for CLI, `hydra mcp` for AI integration — no separate install |
| Line-delimited JSON on stdio | Standard MCP transport, works with all MCP hosts |

### Lessons learned
1. MCP protocol is simple — initialize + tools/list + tools/call covers the full lifecycle
2. Shared client layer prevents drift between CLI and MCP tool behavior
3. Cobra's PersistentPreRun is the right hook for initializing shared state (API client)

---

## Phase 5 Summary (2026-04-07)

**Goal:** It looks and feels like a real product.

### What was done
- Overhauled root README with ASCII architecture diagram, badges, quick start, CLI usage, MCP setup, configuration reference, project structure
- Created GitHub Actions CI pipeline (ci.yml): lint, test, build for core/cli/dashboard/landing + Docker build verification
- Created GitHub Actions release pipeline (release.yml): Docker images to GHCR + cross-compiled CLI binaries (linux/darwin, amd64/arm64) on tagged releases
- Created install script (`scripts/install.sh`): one-liner curl | bash setup for Pi/VPS with dependency checks, health polling, and network info display
- Added MIT LICENSE and CONTRIBUTING.md
- Fixed landing page: updated QuickStart commands to match monorepo (`git clone --recursive`, `docker compose up -d`), fixed repo URLs in footer/open-source sections, corrected license reference (GPL-3.0 -> MIT)

### Key metrics
- **New files:** 6 (README rewrite, LICENSE, CONTRIBUTING.md, ci.yml, release.yml, install.sh)
- **Landing page fixes:** 4 files updated (QuickStartSection, OpenSourceSection, Footer — commands, URLs, license)
- **CI jobs:** 5 (core, cli, dashboard, landing, docker build)
- **Release artifacts:** 4 CLI binaries + 3 Docker images per tag
- **Landing build:** Clean after all changes

### Key decisions
| Decision | Why |
|---|---|
| MIT license | Standard permissive license, simplest for portfolio/open-source |
| Separate CI and release workflows | CI runs on every push/PR; release only on tags — avoids wasting GHCR quota |
| Cross-compile CLI for 4 targets | linux/darwin x amd64/arm64 covers Pi (arm64), Mac (both), and servers |
| Install script with health polling | Better UX than "wait 30s" — shows when service is actually ready |

### Lessons learned
1. Landing page had stale repo URLs pointing to hydra-core instead of hydradns monorepo — always grep for URLs when restructuring
2. License mismatch (footer said GPL-3.0) — single source of truth should be the LICENSE file
3. Install script should detect docker compose v2 vs v1 — can't assume `docker compose` exists everywhere

---

## Tier 1: Demo-Ready Summary (2026-04-08)

**Goal:** Pi can be plugged into a router for a live demo at a school/hospital.

### What was done

**1. Port 53 binding**
- Docker maps host port 53 → container port 1053 (no setcap, no root needed)
- Added `DNS_LISTEN_ADDR` env var override in config loader
- Updated all docs/README/install.sh from port 1053 to 53

**2. Basic authentication**
- New `AdminCredential` model (singleton, bcrypt hash + UUID API key)
- Auth repository with `IsSetup()`, `CreateAdmin()`, `ValidateAPIKey()`
- Bearer token middleware on all API routes (exempt: health, auth endpoints)
- Three new endpoints: `GET /auth/status`, `POST /auth/setup`, `POST /auth/login`
- Before first setup, all routes are open (so setup wizard works)
- After setup, all routes require `Authorization: Bearer <token>`

**3. First-run setup wizard**
- 3-step UI at `/setup`: set password → choose DNS provider → pick blocklists
- Login page at `/login` with automatic redirect to setup if not configured
- Next.js middleware redirects unauthenticated users to `/login`
- Token stored in localStorage (API calls) + cookie (middleware checks)
- Logout button wired in sidebar nav

**4. Blocklist auto-update**
- Removed hardcoded StevenBlack source from dataplane startup
- Sources now loaded from DB (created via API or setup wizard)
- `time.Ticker` goroutine refreshes all enabled sources every 6 hours (configurable via `BLOCKLIST_UPDATE_INTERVAL`)
- ETag persistence: unchanged sources skip download on refresh
- New `refreshBlocklists()` helper with per-source error handling

**5. Pi deployment (arm64)**
- Multi-arch Dockerfile using `TARGETARCH`/`TARGETOS` build args
- Release workflow builds `linux/amd64,linux/arm64` via QEMU + buildx
- Install script detects architecture, disables `systemd-resolved` on port 53 conflict
- New `docs/pi-deployment.md` with router DNS setup for TP-Link, D-Link, Netgear, JioFiber, Airtel Xstream

**6. CLI auth**
- `--token` persistent flag with `HYDRA_TOKEN` env fallback and `~/.hydra/token` file
- `hydra login` command: prompts for password, saves token
- All CLI commands and MCP tools now send `Authorization: Bearer` header

### Key metrics
- **New files:** 10 (model, repo, middleware, handler, auth.ts, middleware.ts, login page, setup page, login cmd, pi-deployment.md)
- **Modified files:** 15 (config, db, store, routes, main, dataplane, engine, Dockerfile, release.yml, install.sh, api.ts, page.tsx, nav-user, root.go, client.go)
- **Build:** Go vet clean, all 52 tests pass, Next.js builds 11 routes, CLI builds clean
- **Auth endpoints:** 3 (status, setup, login)
- **Setup wizard steps:** 3 (password, DNS, blocklists)

### Key decisions
| Decision | Why |
|---|---|
| Docker port mapping 53→1053 | No setcap/root needed, simplest approach |
| Auth bypass when not setup | Setup wizard must work before admin exists |
| Token in localStorage + cookie | localStorage for API, cookie for Next.js middleware (edge runtime can't read localStorage) |
| Blocklist sources from DB only | Setup wizard creates them, periodic refresh fetches them — no hardcoded sources |
| systemd-resolved auto-disable | Most Ubuntu/Pi OS systems have it running on port 53 |

---

## Tier 2: Demo Polish Summary (2026-04-08)

**Goal:** Make the demo professional and differentiated.

### What was done

**1. AI Suspicious DNS Detection**
- New `internal/threat/` package with heuristic domain analysis
- 5 detection methods: Shannon entropy, hex DGA, base64 patterns, digit ratio, subdomain depth
- Infrastructure domain allowlist (cloudfront.net, amazonaws.com, etc.) to reduce false positives
- Integrated into DNS query pipeline — every query scored, suspicious ones marked as "flagged"
- 6 unit tests covering normal domains, hex DGA, entropy, length, and deep subdomains

**2. Query Logging Wired Into Pipeline**
- DNS engine now logs every query to SQLite with action, threat score, detection method
- Async writes (goroutine) to avoid blocking DNS resolution
- Extended DNSQuery model with `IsSuspicious`, `ThreatScore`, `DetectionMethod`, `ThreatReason` fields
- API response includes all threat fields

**3. Dashboard Error States**
- All 5 dashboard pages now show error banners when API calls fail
- Login and setup pages show "API unreachable" message instead of redirect loops
- Logs page shows threat column with score percentage and detection method
- Suspicious rows highlighted with red tint

**4. Branding**
- Sidebar logo replaced with HydraDNS "H" mark + text
- User info updated from placeholder to "Admin"
- Removed unused icon imports

### Key metrics
- **New files:** 2 (detector.go, detector_test.go)
- **Modified files:** 10 (engine.go, dns_query model, analytics handler, types.ts, logs page, sidebar, 4 dashboard pages with error states)
- **Detection methods:** 5 (entropy, dga_hex, dga_base64, dga_digits, subdomain_depth)
- **Tests:** 52 → 58 (6 new threat tests)
- **Critique findings:** 10 found, 7 fixed, 3 deferred

### Lessons learned
1. Synchronous DB writes in hot paths are a performance trap — always async for logging
2. CDN domains look like DGA — always have an infrastructure allowlist
3. Empty slices vs nil matter for JSON serialization — `null` vs `[]` causes frontend crashes

See [critique-003.md](critique-003.md) for full review findings.

---

## MCP Go-Live Session (2026-04-10)

**Goal:** Get MCP server working end-to-end with a real AI client (Gemini CLI).

### What was done

**1. MCP Protocol Fixes**
- Fixed: server was sending response to `notifications/initialized` — notifications must get no response
- Fixed: JSON responses could have null fields that fail Zod schema validation in MCP SDK clients
- Added custom `MarshalJSON` that ensures exactly `result` OR `error` in each response, never both

**2. Submodule Registration**
- `apps/cli` was a working git submodule internally but missing from `.gitmodules`
- `git pull --recurse-submodules` on other machines didn't fetch CLI code
- Registered `hydra-cli` in `.gitmodules` and setup.sh

**3. New MCP Tool: `create_policy`**
- AI was calling `block_domain` 7 times for "block social media" — creating 7 separate policies
- Added `create_policy` tool that accepts name, action, domains array, and priority
- AI now creates one grouped policy (e.g. "Block Social Media" with all domains)

**4. Live Verification**
- MCP server connected to Gemini CLI: 🟢 hydradns - Ready (9 tools)
- Tested: "Check status, block all social media, show policies" — Gemini called get_status, create_policy, list_policies correctly
- DNS blocking confirmed: `dig @localhost facebook.com` returns REFUSED after AI-created policy

### Key metrics
- **MCP tools:** 8 → 9 (added create_policy)
- **Protocol fixes:** 2 (notification response, JSON serialization)
- **Verified with:** Gemini CLI (Google)
- **End-to-end:** AI prompt → MCP tool call → DNS policy created → domain blocked on network

### Lessons learned
1. MCP notifications must get zero response — even `{"id":null}` breaks Zod validation
2. MCP SDK clients validate response JSON strictly — use custom marshaling to prevent null fields
3. AI defaults to calling single-item tools N times — provide batch tools with clear descriptions to guide it
4. Submodules must be in `.gitmodules` for recursive clone/pull — internal `.git/modules/` isn't enough

---

## Deferred Items (Tech Debt)

These are known issues that were intentionally deferred. Track them here so they don't get lost.

| Item | Severity | Deferred Since | Notes |
|---|---|---|---|
| Regex/wildcard policy evaluation | Medium | Phase 1 | Parsed but not evaluated at query time |
| Query log retention / cleanup | Medium | Phase 1 | DNSQuery table grows unbounded |
| TLS on gRPC | Medium | Phase 1 | Uses `grpc.WithInsecure()` |
| CORS hardcoded to localhost:3000 | Low | Phase 1 | Needs env config for production |
| BlocklistEntry composite index | Low | Phase 1 | (SourceID, Domain) for faster queries |
| SQLite single-connection bottleneck | Low | Phase 1 | Fine for v1, revisit under load |
| Hardcoded magic numbers in engine | Low | Phase 1 | Timeout, retries, pool size, TTL |
| Edit/update for policies & blocklists | Low | Phase 2 | Only create/delete, no edit UI |
| Pagination for query logs | Low | Phase 2 | Shows all 100 entries at once |
| Settings page | Low | Phase 2 | No backend support yet |
| WebSocket for live log streaming | Low | Phase 2 | Using polling (setInterval) instead |
| Systemd service file for Pi | Low | Phase 3 | Docker-only deployment for now |

---

## Architecture Decisions Record

### ADR-001: Submodule monorepo structure
- **Decision:** Each service is a separate Git repo, aggregated via submodules
- **Context:** Solo dev, want independent deployment per service
- **Consequence:** More complex git workflow, but clean separation

### ADR-002: Control plane / data plane split via gRPC
- **Decision:** Separate processes communicating over gRPC
- **Context:** Data plane handles DNS (latency-critical), control plane handles admin API
- **Consequence:** Can scale/restart independently, but adds operational complexity

### ADR-003: SQLite over PostgreSQL for v1
- **Decision:** Core uses SQLite with WAL mode, single connection
- **Context:** Simpler deployment (especially on Pi), no external DB dependency
- **Consequence:** Limited to single-writer, but good enough for home/small-office scale

### ADR-004: Client-side data fetching for dashboard
- **Decision:** Use client components with native fetch + useEffect, not server components or SWR
- **Context:** Dashboard needs auto-refresh (polling), all data from external API not same-origin
- **Consequence:** No SSR for dashboard pages, but simpler and no new dependencies

### ADR-006: Combined core container
- **Decision:** Run controlplane + dataplane in a single Docker container with entrypoint script
- **Context:** Both processes share SQLite; separate containers can't share a SQLite file safely
- **Consequence:** Simpler deployment (one container), but can't scale processes independently

### ADR-005: Inline forms over modals
- **Decision:** Expandable inline forms for create operations, not dialog/modal overlays
- **Context:** No dialog component installed, Sheet exists but inline is simpler for forms
- **Consequence:** More vertical scrolling but no z-index/focus-trap complexity
