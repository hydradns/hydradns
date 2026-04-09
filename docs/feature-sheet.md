# HydraDNS — Product Feature Sheet

**DNS Security Gateway for Schools, Hospitals, and Small Businesses**

---

## What It Does

HydraDNS protects every device on your network by filtering DNS queries — the system that converts website names to IP addresses. Malware, phishing, ads, and unwanted content are blocked before they can ever connect. No software to install on individual devices.

---

## Feature List

### Threat Protection
| Feature | HydraDNS | Pi-hole (Free) | NextDNS ($2/mo) | Cisco Umbrella ($5+/user/mo) |
|:--------|:--------:|:--------------:|:---------------:|:---------------------------:|
| Blocklist-based filtering (92K+ domains) | Yes | Yes | Yes | Yes |
| AI threat detection (DGA, entropy, tunneling) | Yes | No | No | Yes (ML-based) |
| Subdomain blocking (block x.com = block www.x.com) | Yes | Yes | Yes | Yes |
| DoH bypass prevention | Yes | No | N/A | Yes |
| Real-time threat scoring | Yes | No | No | Yes |
| Custom block/allow policies | Yes | Yes | Yes | Yes |
| Newly registered domain blocking | Planned | No | Yes | Yes |

### Filtering & Policy Control
| Feature | HydraDNS |
|:--------|:---------|
| Domain-level blocking | Yes |
| Subdomain auto-matching | Yes (block parent = block all subdomains) |
| Policy priority system | Yes (highest priority wins) |
| Per-policy action (Block / Allow / Redirect) | Yes |
| Real-time policy enforcement | Yes (< 5 seconds) |
| DNS search domain stripping | Yes (handles router-appended suffixes) |
| IPv4 + IPv6 blocking | Yes (0.0.0.0 + ::) |
| Multiple blocklist sources | Yes (auto-refresh every 6 hours) |
| ETag-based efficient updates | Yes (skip unchanged lists) |

### Dashboard & Visibility
| Feature | HydraDNS |
|:--------|:---------|
| Real-time web dashboard | Yes |
| Total / blocked / allowed query counts | Yes |
| Block rate percentage | Yes |
| Query log with search and filter | Yes |
| Threat detection column (score + method) | Yes |
| Suspicious query highlighting | Yes (red rows) |
| DNS engine status and control | Yes |
| Blocklist source management | Yes |
| Policy management (create/delete) | Yes |
| Auto-refresh stats (5s interval) | Yes |

### AI-Powered Threat Detection (Hydra Intelligence)
| Detection Method | What It Catches |
|:-----------------|:----------------|
| Shannon entropy analysis | Random-looking domains (DGA malware) |
| Hexadecimal pattern detection | C2 beacon domains (e.g., a1b2c3d4e5f6...) |
| Base64 pattern detection | Data exfiltration attempts |
| Digit ratio analysis | Algorithmically generated domains |
| Subdomain depth analysis | DNS tunneling attempts |
| Infrastructure allowlist | Prevents false positives on CDN/cloud domains |

### Deployment
| Feature | HydraDNS |
|:--------|:---------|
| One-command install | Yes (`curl ... \| bash`) |
| Docker-based deployment | Yes |
| Raspberry Pi support (arm64) | Yes |
| Multi-architecture builds (amd64 + arm64) | Yes |
| Setup wizard (browser-based) | Yes (2-step: password + blocklists) |
| No per-device software needed | Yes (network-level) |
| Port 53 standard DNS | Yes |
| Persistent data across restarts | Yes (Docker volumes) |
| Health monitoring endpoint | Yes (`/health`) |

### Authentication & Security
| Feature | HydraDNS |
|:--------|:---------|
| Admin password protection | Yes (bcrypt) |
| API token authentication | Yes (Bearer token) |
| Dashboard login required | Yes |
| Pre-setup route protection | Yes (403 until configured) |
| CORS configuration | Yes (configurable origins) |

### Management & Integration
| Feature | HydraDNS |
|:--------|:---------|
| REST API (full CRUD) | Yes (17 endpoints) |
| CLI tool (`hydra`) | Yes (11 commands) |
| MCP server (AI integration) | Yes (8 tools, JSON-RPC 2.0) |
| Claude Code / AI assistant compatible | Yes |
| Environment variable configuration | Yes |
| Configurable blocklist refresh interval | Yes (default 6h) |

---

## Competitive Positioning

### vs. Pi-hole / AdGuard Home (Free, Self-Hosted)
**They have:** Blocklist filtering, web dashboard, community support
**We add:** AI threat detection, MCP/AI integration, setup wizard, built-in auth, subdomain auto-matching, Docker-first deployment, professional dashboard

### vs. NextDNS / Cloudflare Gateway ($2-7/user/month)
**They have:** Cloud-hosted, per-user policies, client apps, global network
**We offer:** Self-hosted (no per-user fees), data stays on your network, no cloud dependency, one-time hardware cost + flat management fee, works on any network without internet dependency for cached data

### vs. Cisco Umbrella / Palo Alto ($5-12/user/month)
**They have:** ML/AI threat intelligence, global network, SIEM integration, compliance certifications, SLA guarantees
**We offer:** 90% of the protection at 5% of the cost, no per-user licensing, physical device on premises (tangible), managed service model, ideal for Indian SMB/school/hospital budget

---

## Pricing Model

| Component | Cost |
|:----------|:-----|
| Hardware (Raspberry Pi 4) | ~8,000 INR (one-time) |
| Installation + Setup | Included |
| Monthly Management | 1,500 - 5,000 INR/month |
| Software | Open-source (free) |
| Per-user fees | None |
| Per-device fees | None |

**Compare:** Cisco Umbrella for a 50-person school = ~$275/month (~23,000 INR/month). HydraDNS = 1,500 INR/month. **15x cheaper.**

---

## Technical Specifications

| Spec | Value |
|:-----|:------|
| DNS Engine | Go 1.24, miekg/dns library |
| Database | SQLite (WAL mode, pure-Go driver) |
| API Framework | Gin (Go) |
| Dashboard | Next.js 16, React 19, TypeScript |
| Container | Alpine Linux, multi-stage Docker build |
| Binary size | ~15 MB (controlplane + dataplane) |
| Blocklist capacity | 92,283 domains (StevenBlack), expandable |
| Threat detection methods | 5 heuristic analyzers |
| Policy enforcement latency | < 5 seconds |
| Blocklist refresh | Configurable (default 6h), ETag caching |
| Authentication | bcrypt + UUID Bearer tokens |
| Supported architectures | amd64, arm64 |

---

## Roadmap (Planned Features)

| Feature | Timeline |
|:--------|:---------|
| Remote monitoring (heartbeat + alerts) | Tier 3 |
| Query log retention (auto-cleanup) | Tier 3 |
| TLS on dashboard | Tier 3 |
| Category-based filtering (60+ categories) | Future |
| Time-based policies (e.g., block social media during school hours) | Future |
| Multi-site management | Future |
| Email/SMS alerts | Future |
| SIEM log export | Future |
| CIPA compliance certification | Future |
