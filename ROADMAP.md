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

---

## Tier 3 — Post-Demo Hardening (April 28+)

**Goal:** Move from "demoable portfolio piece" to "safe to leave running for a paying customer".

### Live bugs (blocking — fix first)
- `UNIQUE constraint failed: statistics.id` on every query, logged repeatedly. Breaks stats increment intermittently. _Fixed on `apps/core feat/bypass-mitigations`._
- Blocklist ingestion persists the source row but `domains_count` never leaves 0. Fetch/parse/snapshot pipeline is not being triggered (or is silently failing) on `POST /api/v1/blocklists`. _Fixed on `apps/core feat/bypass-mitigations` (immediate async fetch on create)._

### Operability
- **Query log retention** — rotation job + configurable max-age so `DNSQuery` stops growing forever
- **TLS on dashboard and gRPC** — stop shipping `grpc.WithInsecure()` and plain-text admin auth
- **Update mechanism** — `hydra update` or container self-pull so Pis stay current without SSH
- **Remote monitoring** — per-Pi heartbeat + basic alerting for multi-customer deploys
- **Bare-metal systemd unit** for Pi deploys that want to skip Docker

### Feature gaps
- **Regex / wildcard policy evaluation** (parsed but not enforced at query time)
- **Policy + blocklist edit UI** (today: create + delete only)
- **Query log pagination** (hard-capped at 100 entries, no paging)
- **Settings page** (UI stub, no backend wiring)
- **`/api/v1/dns/resolvers`** returns mock data — wire upstream editing
- **Scanner** currently only reads `/etc/resolv.conf`; LAN client enumeration not implemented

### Polish
- WebSocket for live query streaming (replace polling)
- Mobile-responsive dashboard (deferred out of the demo cycle)
- CORS env-driven config for production deploys

### Exit Criteria
A fresh Pi running HydraDNS for a month doesn't log a single `UNIQUE constraint` error, blocklists show real domain counts within 60s of adding them, the dashboard is reachable over HTTPS, and `hydra update` can ship a new version without manual intervention.

---

# Commercial Product Roadmap

Phases 6–9 move HydraDNS from "demoable portfolio piece + free homelab tool" to "something a small business will pay for, and eventually an enterprise will evaluate." Sequencing is based on `docs/enterprise-gap-analysis.md` — read that first for the competitive landscape.

**Solo-dev reality check:** each phase below is an eyeball estimate assuming evenings + weekends. Real calendar time depends on whether HydraDNS stays a side project or gets dedicated time.

---

## Phase 6 — Commercial Floor (est. 2–3 months FTE)

**Goal:** HydraDNS stops being a toy. Anyone running >1 user on it has a product they can trust, and a non-technical buyer sees something that looks like a real commercial SKU.

These are the **three non-negotiable gaps** from the gap analysis plus the operability items the Tier 3 hardening already named.

### Multi-user + RBAC
- Drop the `AdminCredential` singleton in favour of a proper `User` model with roles: `admin`, `operator`, `read_only`
- Per-user bearer tokens with rotation
- Audit log table: `actor_id`, `action`, `target`, `before`, `after`, `ip`, `ts` on every write endpoint
- Dashboard: user-management page, per-user MFA (TOTP), session timeout

_Backend landed on `apps/core feat/rbac-and-audit` (8 commits) — User/Token/AuditEvent models, repos, middleware with RequireRole, audit on every mutating handler, GET /audit, user + token CRUD. Dashboard UI is the remaining piece._

### SSO (SAML + OIDC)
- OIDC first (Okta, Google Workspace, Microsoft Entra all support it, less painful than SAML)
- Role mapping from IdP group claims onto HydraDNS roles
- SAML as a follow-up once a prospect asks

### Encrypted DNS — DoH + DoT
- DoH listener on `:443` (or `:5443` for coexistence with a reverse proxy)
- DoT listener on `:853`
- Outbound upstream support for DoH and DoT resolvers (Cloudflare 1.1.1.1, Quad9, Google)
- ACME / Let's Encrypt integration so certs aren't a manual chore

### Categorized threat intelligence
- New `Category` concept in the data model: "ads", "malware", "phishing", "adult", "gambling", "tracking", "newly-registered", "cryptomining", "c2"
- Ship a default set of curated feeds mapped to categories. For v1, use existing community feeds (OISD categories, URLhaus, Hagezi) behind a category abstraction
- Dashboard: category toggles, not list URLs. List URLs become an advanced option
- Policy model gains `category` as an alternative to `domains`

### Operability (rolled up from Tier 3)
- Fix the `statistics.id` UNIQUE bug ✅
- Fix blocklist ingestion so `domains_count` actually populates ✅
- Query log retention + rotation
- TLS on gRPC + dashboard
- `hydra update` + Docker self-pull

### Bypass mitigations (shipped on `feat/bypass-mitigations`)
- DoH/DoT bootstrap blocklist baked into the engine — invisible to the dashboard, defeats default-on browser DoH for >80% of cases ✅
- Router config script (`hydra setup-router`) generates pfSense/MikroTik/OpenWrt/ASUS firewall rules that lock outbound DNS to the HydraDNS Pi and optionally blackhole known DoH provider IPs on `:443` ✅
- `BLOCK_RESPONSE` env switch (`zero` / `nxdomain` / `refused`) so operators can A/B test response shapes per deployment ✅

### Exit Criteria
A 20-person company's IT admin can set up HydraDNS, hook it to their Okta, give finance "read-only" access, enforce DoH for all laptops, turn on "block malware + phishing + gambling" as categories, and receive a weekly email report — all without touching a config file.

---

## Phase 7 — SMB Competitive (est. 4–6 months FTE)

**Goal:** HydraDNS can credibly win head-to-head against DNSFilter and NextDNS Teams for a company of 50–500 people.

### Multi-site control plane
- Central "tenant" service that manages N "sites" (each site = one HydraDNS box)
- Site inherits policies from the tenant root; overrides allowed
- Per-site auth for the box, per-tenant auth for the control plane
- Likely means splitting the current SQLite-bundled control plane into a separate tenant service + per-site local controller

### Per-client / per-group policy
- Client identity model: by IP range, MAC (via DHCP lease ingestion), or endpoint-agent-reported identity
- AD/LDAP/Okta group sync so policies can target "finance" not IPs
- Policy priority resolution accounts for client-identity match

### Endpoint agent (v1)
- Start with macOS + Windows. Mobile later
- The agent is essentially a local DoH client that ignores OS DNS and talks directly to the tenant's HydraDNS site
- Out-of-network fallback: direct connection to a cloud-hosted relay (operated by HydraDNS Inc, not the customer's Pi)
- This is the phase where a cloud-hosted component becomes unavoidable

### Branded block page
- Walled-garden redirect with customer logo, reason, contact link, and a "request access" form that opens a ticket in the admin queue

### Reporting + SIEM
- Scheduled PDF / CSV reports, emailed on a cron
- Top-N dashboards (domains, categories, clients) with date ranges
- Syslog output (RFC 5424) + Splunk HEC + generic webhook for every query event and every audit event

### HA (active-passive first)
- Two-box deployment, shared config via the tenant control plane, health-based DNS failover via router config or keepalived
- Active-active with shared state is phase 8; active-passive is good enough for SMB

### DNS tunneling detection
- Heuristics: long labels, high per-query entropy, NXDOMAIN bursts, fast-flux domain lookups
- Flag as "suspicious" in logs, optional auto-block by threshold

### Exit Criteria
HydraDNS wins a side-by-side POC against DNSFilter for a mid-market customer: it matches on filtering quality, wins on price / privacy / self-hostability, and doesn't get knocked out by missing features in procurement's checklist.

---

## Phase 8 — The MCP Wedge (parallel track, est. 2 months FTE)

**Goal:** Lean into the one place HydraDNS is genuinely ahead of the incumbents: agent-first control. This is a parallel track, not sequential — spin it alongside Phase 6/7 work.

- **MCP tool coverage:** every write operation in the dashboard has an equivalent MCP tool. Today we have 9; we'll need closer to 30 as the product grows.
- **MCP guardrails:** role scoping on tokens (an agent with the "reporter" role cannot `toggle_engine`). Rate limits. Confirmation prompts surfaced to the client for destructive ops.
- **Agent-first UX patterns:** `create_policy` batch ops already exist. Build more batch-first tools: `bulk_unblock`, `suggest_categories_for_client`, `explain_why_blocked`.
- **First-party agent experience:** publish a ready-to-install MCP config for Claude Desktop, Claude Code, Gemini CLI, and Cursor. A "one-click" registration in the dashboard that copies the right config block.
- **Agent-driven reporting:** instead of a static PDF, "ask the agent" — `get_weekly_summary`, `explain_anomaly`, `compare_to_last_month`.
- **Marketing play:** put MCP front and center on the landing page. No competitor is doing this. It's the moat.

### Exit Criteria
A customer's IT admin can manage HydraDNS end-to-end via conversation with Claude. Pitch decks and case studies lead with "the AI-native DNS firewall", not with the feature list.

---

## Phase 9 — Enterprise Floor (deferred, 12+ months)

**Goal:** HydraDNS is a credible choice for a 5,000-person enterprise. Reachable only with funding or team.

### Compliance
- SOC 2 Type I → Type II (external audit, cost $20k–$50k)
- ISO 27001, optionally
- HIPAA BAA templates
- GDPR DPA templates, sub-processor list, data-residency controls
- Penetration test report on request

### Deep HA + scale
- Active-active with shared state (move control plane off SQLite onto Postgres or a distributed KV)
- Anycast or geo-distributed POPs for the cloud-hosted relay
- 99.99% uptime SLA with credit-backed commitment

### API + GitOps
- Terraform provider, Kubernetes operator, policy-as-code with PR-based review
- Bulk import / export with schema validation

### Deployment surface
- Virtual appliance (OVA, VHD), AWS / Azure / GCP marketplace listings, Helm chart
- Per-seat / per-query metering, self-service trial → paid upgrade

### Support + ops
- 24/7 on-call rotation (needs team)
- Named TAM for top-tier accounts
- Customer Slack / Teams shared channels

### Exit Criteria
HydraDNS is on an enterprise shortlist alongside Cisco Umbrella and Cloudflare Gateway. Procurement stops flagging the vendor as "too small".

---

## Prioritization heuristic

Pick the next thing to build by answering, in order:

1. **Is there a paying customer asking for it?** Do that first.
2. **Is it a Phase 6 item?** Do that next. Phase 6 is the gate to charging money at all.
3. **Is it an MCP improvement (Phase 8)?** Do that in parallel; it's cheap and it's the wedge.
4. **Is it a Phase 7 item?** Only once Phase 6 is complete.
5. **Is it a Phase 9 item?** Not yet. Defer until a funded team exists.
