# HydraDNS — Business Alignment Plan

**Date:** 2026-04-07
**Purpose:** Bridge the gap between what's built and what's needed to put a Raspberry Pi on someone's router and get paid for it.

---

## Where We Are

### Built and Working
- DNS engine: blocklist filtering (Bloom filter), policy engine, upstream forwarding — all verified with dig
- Dashboard: 5 pages (home, DNS engine, blocklists, policies, query logs) wired to real API
- Docker Compose deployment: single core container, healthchecks, persistent volume
- CLI: 11 commands via `hydra` binary
- MCP server: 8 tools for AI-driven management
- 52 tests, 10 bugs fixed, async blocklist fetch (<1s startup)

### Done (Demo Blockers — Completed April 8)
All 5 critical items shipped in Tier 1:

| Item | Status | Implementation |
|---|---|---|
| Port 53 binding | DONE | Docker maps 53→1053, `DNS_LISTEN_ADDR` env var |
| Basic auth on dashboard | DONE | Bcrypt + bearer token, 3 auth endpoints, Next.js middleware |
| First-run setup wizard | DONE | 3-step: set password → choose DNS → pick blocklists |
| Auto-update blocklists | DONE | 6hr ticker, ETag caching, sources from DB only |
| Pi deployment (arm64) | DONE | Multi-arch Docker, install script, router docs for Indian ISPs |

Also shipped: CLI auth (`hydra login`), CI/CD pipeline, cross-compiled binaries, MIT license.

### Not Built Yet (Demo Polish — Tier 2)
These make the demo professional instead of just functional:

| Item | Why |
|---|---|
| AI suspicious DNS detection (demoable) | This is the differentiator — must be showable even if heuristic-based |
| Dashboard loading states + branding | First impressions matter — no blank screens, HydraDNS logo/name |
| Mobile-responsive dashboard | IT admin will pull it up on phone |
| Real Pi hardware test | End-to-end: boot → setup wizard → block domain → query log — on actual hardware |

### Not Built Yet (Pre-Revenue — Tier 3)
Must exist before charging money:

| Item | Why |
|---|---|
| Remote monitoring/management | You need to know when a client's Pi goes down before they call you |
| Query log retention/cleanup | Unbounded table will fill the Pi's SD card |
| TLS on dashboard | Serving admin UI over HTTP on a LAN is a bad look for a security product |
| Update mechanism | Push config/blocklist updates to deployed Pis without physical access |
| Domain input validation | A school IT admin will type garbage — handle it gracefully |

---

## The Demo Story

This is what you walk into a school or hospital and show. Every engineering decision below serves this narrative.

### The 5-Minute Pitch
> "Your network has no protection against malicious websites, phishing, or malware at the DNS level. Enterprise firewalls cost lakhs. This device costs ₹8,000 to install and ₹1,500/month. I plug it into your router, and within 30 minutes, every device on your network is protected. You get a dashboard showing what's being blocked, and I manage it remotely."

### The Demo Flow
1. **Plug Pi into their router** — physically do it in front of them
2. **Open dashboard on their laptop** — show it's already running, KPI cards updating
3. **Show blocklist in action** — open a known malware domain on their browser, show it blocked, show the query log entry appear in real-time
4. **Show the AI detection hook** — "this isn't just a blocklist, it catches suspicious domains that aren't on any list yet"
5. **Show policy control** — block a category (e.g., social media during school hours), demonstrate it works immediately
6. **Leave it running** — "try it for a week, I'll check in"

### What Must Work Flawlessly for This Demo
- Pi boots → dashboard is accessible within 2 minutes
- DNS redirect works → all LAN devices go through HydraDNS
- Blocking a known malicious domain → visible in query logs within seconds
- Adding a policy → takes effect immediately (no restart)
- Dashboard loads fast, doesn't error, looks professional

---

## Engineering Priorities (Aligned to Business)

### Tier 1: Demo-Ready — COMPLETE (April 8)

All items shipped. See report.md Tier 1 summary for details.

### Tier 2: Demo Polish (Now → April 18)

**6. Loading states on dashboard**
- Skeleton/spinner on all pages while API data loads
- Error states that say something useful ("Can't reach HydraDNS engine — is it running?")

**7. AI suspicious DNS detection — demo-ready**
- This is the differentiator in the pitch. Whatever state it's in, make it demoable.
- Even if it's heuristic-based (entropy scoring, DGA detection), package it as "AI-powered threat detection"
- Show a real example: generate a DGA-like domain, show it flagged

**8. Dashboard cosmetics**
- HydraDNS branding (logo, name, consistent colors)
- Mobile-responsive (IT admin will check on phone)
- Dark mode is fine as default but should look polished

### Tier 3: Pre-Revenue (Weeks 5-8, before first paying customer)

**9. Remote monitoring**
- Pi phones home to a central endpoint you control (heartbeat + metrics)
- You get alerted if a Pi goes offline
- Can push config updates remotely

**10. Query log retention**
- Configurable retention period (default 7 days)
- Background cleanup goroutine
- Dashboard shows "last 7 days" not "all time"

**11. Update mechanism**
- `hydra update` CLI command or auto-update on heartbeat
- At minimum: pull new Docker images and restart
- Blocklist updates already handled by Tier 1 item

**12. TLS on dashboard**
- Self-signed cert auto-generated on first run
- Or Let's Encrypt if Pi has a public hostname (unlikely for LAN — self-signed is fine)

---

## Customer Conversations Plan

### Who to Talk To (5 minimum by April 28)

| Prospect Type | How to Find Them | What to Learn |
|---|---|---|
| School IT admin | Personal network, local schools in Delhi NCR | Do they have any DNS filtering? What do they spend on IT security? Who approves purchases? |
| Hospital IT staff | Walk in and ask for IT department | Same + compliance requirements (patient data, network isolation) |
| Small business (10-50 employees) | Local markets, industrial areas | Do they even have a dedicated IT person? Who manages their router? |
| WiJungle contacts | Former colleagues | What do SMB clients actually ask for? What do they reject as too expensive? |
| Hosto clients | Existing relationship | Would they pay for DNS security on their office network? |

### Discovery Questions (Don't Pitch — Listen)
1. "What happens today if someone on your network visits a malicious website?"
2. "Have you ever had a malware or phishing incident? What happened?"
3. "What do you currently spend on network security per year?"
4. "If I could protect every device on your network for ₹1,500/month, would that be interesting?"
5. "Who would need to approve a purchase like this?"

### What You're Validating
- **Problem exists:** Do they care about DNS security, or is it not even on their radar?
- **Willingness to pay:** Is ₹1,500-5,000/month in the range they'd consider?
- **Decision maker:** Can the IT admin buy this, or does it go to a principal/director/procurement?
- **Deployment reality:** Will they let you plug a device into their router?

---

## GitHub Cleanup (Do This Week)

This was planned weeks ago. No more deferring.

- [ ] Delete empty repos: hydra-cli, hydra-central-controller, hydradns (org-name repo)
- [ ] Fix hydra-ui README (replace Next.js boilerplate with real description) or archive
- [ ] Fix naming: update hydra-core description from "Phantom Core" to "HydraDNS Core"
- [ ] Update resume: "Phantom DNS" → "HydraDNS" everywhere
- [ ] Cut v0.1.0 tag on hydra-core (Tier 1 is done — do this NOW)

---

## Timeline Summary

```
April 5-8   (Done)       Phases 1-5 + Tier 1: Core, dashboard, deployment, CLI, auth, setup wizard, Pi-ready
April 8-18  (Now)        Tier 2: AI detection demoable, dashboard polish, real Pi test
April 18-28 (Week 3-4)   5 prospect conversations + first demos
April 28+   (Weeks 5-8)  Tier 3: remote mgmt, log retention, TLS → first paying customer
End of May               Target: 2-3 demos done, 1 paying customer (stretch)
```

---

## Success Criteria

| Milestone | Metric | Deadline |
|---|---|---|
| Demo-ready | Pi boots, blocks, dashboard works — tested on real hardware | DONE (April 8) |
| Market signal | 5 prospect conversations completed | April 28 |
| First demo | Live demo at a school, hospital, or SMB | May 7 |
| First revenue | 1 paying customer | May 31 (stretch) |

---

## Open Risks

| Risk | Mitigation |
|---|---|
| Pi SD card reliability | Use USB SSD boot or high-endurance SD card. Document for clients. |
| Router DNS redirect varies wildly | Build a cheat sheet for top 5 router brands in India. Offer to configure it during install. |
| School/hospital procurement is slow | Offer a free 2-week trial. Decision is easier when it's already running. |
| Single point of failure (one Pi) | For v1, accept it. Failover to upstream DNS if Pi is down (configure on router as secondary DNS). |
| You're one person | Don't take more than 5-10 clients until remote management exists. |
