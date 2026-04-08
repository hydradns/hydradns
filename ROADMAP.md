# HydraDNS Product Roadmap

## Definition of "Complete"

The product is shippable when someone can:
1. `docker compose up` on a Pi or VPS and have DNS filtering working in under 5 minutes
2. Open a dashboard to see what's happening and manage blocklists/policies without touching config files
3. Optionally use a CLI (and eventually MCP) for power-user / AI-driven management
4. Visit a landing page that explains what it does and how to get it

For **job interviews / portfolio**, Phases 1-3 are the credible PoC. Phases 4-5 are differentiators.

---

## Phase 1 — Solid Core (Current)

**Goal:** The DNS engine is correct, reliable, and testable.

### Done
- DNS query pipeline (blocklist → policy → upstream)
- Blocklist fetch + parse + store (StevenBlack hosts format)
- Policy engine (exact domain match, Bloom filter)
- gRPC control/data plane split
- Bug fixes (SERVFAIL on errors, domain normalization, nil guards)
- Config via env vars for local dev

### Remaining
- Unit tests for critical path: dnsengine, policy, blocklist, storage repos
- Wire mock handlers to real storage (blocklists CRUD, policies CRUD, query logs)
- Remove all mock data from control plane handlers

### Exit Criteria
`go test ./...` passes with meaningful coverage on the query pipeline, and every API endpoint hits real storage.

---

## Phase 2 — The Dashboard

**Goal:** Non-technical user can see and control everything from the browser.

- API client layer in UI (connect Next.js to control plane REST API)
- Dashboard home: live stats (queries/sec, blocked %, top blocked domains, system health)
- Blocklist management page: add/remove sources, entry counts, toggle enable/disable
- Policy management page: create/edit/delete with domain input, action picker, priority
- Query log viewer: searchable/filterable table of recent DNS queries with block reason
- DNS engine toggle: start/stop button with live status indicator

### Exit Criteria
You can manage HydraDNS entirely from `localhost:3000` without touching any config file or CLI.

---

## Phase 3 — Plug-and-Play Deployment

**Goal:** One-command setup on Raspberry Pi, works as network-wide DNS.

- Single Dockerfile entrypoint (combined controlplane + dataplane for Pi)
- Setup wizard / first-run flow (detect network, configure upstream DNS, create initial blocklists)
- Listen on port 53 with proper privilege handling
- DHCP/router integration docs for common routers
- Health monitoring (self-check endpoint, auto-restart, systemd service file)
- Persistent config (survive reboots, retain blocklists/policies/logs)
- Update mechanism (scheduled blocklist refresh, container self-update)

### Exit Criteria
Flash a Pi, run one script, point your router at it — whole network is filtered, survives reboots, updates blocklists automatically.

---

## Phase 4 — CLI + MCP

**Goal:** Power users and AI agents can manage HydraDNS programmatically.

- `hydra` CLI wrapping control plane API: `hydra status`, `hydra block <domain>`, `hydra logs`, `hydra blocklist add <url>`
- Shell completions (bash/zsh/fish)
- MCP server exposing CLI tools as MCP tools for Claude
- MCP demo flow: "Block all social media" → Claude calls MCP tools → domains blocked

### Exit Criteria
`hydra block tiktok.com` works from terminal, and Claude via MCP can do the same conversationally.

---

## Phase 5 — Polish & Ship

**Goal:** It looks and feels like a real product.

- Landing page: hero, features, architecture diagram, install instructions, demo GIF
- README overhaul: badges, screenshots, quick start, architecture section
- GitHub release pipeline: CI builds Docker images, pushes to GHCR, tagged releases
- Pi install script: `curl -fsSL https://... | bash` one-liner
- Demo video: 2-min screencast (install → dashboard → block → unblock)
- License & contributing guide

### Exit Criteria
Someone lands on the GitHub repo or landing page and can go from zero to running in 5 minutes.
