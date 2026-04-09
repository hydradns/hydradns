# HydraDNS — Business Model

**Last updated:** 2026-04-09
**Status:** Draft — iterate as we learn from prospect conversations

---

## Model: Managed Security Appliance (Annual License)

Open-source software + hardware appliance + managed service.
Client pays for the device once, then an annual fee for management and protection.

---

## Pricing

| Component | Price | Notes |
|:----------|:------|:------|
| Hardware (Pi 4 + case + SD/SSD) | 8,000 - 10,000 INR | One-time. Client owns the device. |
| Installation + Setup | Free | Included in first year. Done on-site in 30 min. |
| Annual License + Management | 15,000 - 18,000 INR/year | Billed upfront annually. Includes: blocklist updates, threat monitoring, remote management, support. |
| **Total Year 1** | **23,000 - 28,000 INR** | |
| **Renewal Year 2+** | **15,000 - 18,000 INR/year** | |

### Discounts
| Term | Price | Savings |
|:-----|:------|:--------|
| 1 year | 15,000 INR | — |
| 2 years | 27,000 INR | 10% off |
| 3 years | 36,000 INR | 20% off |

### Comparison with Competitors
| Solution | Cost for 50-user school/year | Per-user? |
|:---------|:----------------------------|:----------|
| **HydraDNS** | 23,000 INR (year 1), 15,000 INR (year 2+) | No — flat fee |
| Cisco Umbrella | ~2,75,000 INR/year | Yes — $5.50/user/mo |
| NextDNS Business | ~16,000 INR/year | Yes — $19.90/mo per 50 users |
| Cloudflare Gateway | ~42,000 INR/year | Yes — $7/user/mo |
| Pi-hole (self-managed) | Free (but no one manages it) | No |

**Key advantage:** No per-user pricing. A 50-person school and a 500-person school pay the same. The 500-person school gets 18x better value than Cisco.

---

## What's Included in Annual License

| Service | Included |
|:--------|:---------|
| 92,000+ domain blocklist (auto-refreshed every 6h) | Yes |
| AI threat detection (DGA, entropy, tunneling) | Yes |
| Custom block/allow policies | Yes |
| Web dashboard access | Yes |
| Query logging and analytics | Yes |
| Remote monitoring and management | Yes (when built) |
| Software updates | Yes |
| Blocklist source updates | Yes |
| Support (email/WhatsApp) | Yes |
| On-site visit for hardware issues | 1 per year included |

---

## What's NOT Included (Potential Upsells)

| Add-on | Price | Notes |
|:-------|:------|:------|
| Additional Pi for redundancy/failover | 10,000 INR one-time | For critical networks (hospitals) |
| Custom content category filtering | 5,000 INR/year add-on | When feature is built |
| Compliance reporting (PDF annual report) | Included in base | Use as retention tool |
| Multi-site management | 10,000 INR/site/year | When feature is built |
| Priority support (4-hour response) | 5,000 INR/year add-on | For hospitals |

---

## Revenue Math

### Conservative (Year 1)
| Metric | Value |
|:-------|:------|
| Clients acquired | 5 |
| Avg first-year revenue per client | 25,000 INR |
| Hardware cost per client | ~5,000 INR (Pi 4 at cost) |
| **Gross revenue** | **1,25,000 INR** |
| **Gross profit** | **1,00,000 INR** |
| Time spent per client (setup + monthly check) | ~2 hours/month |

### Year 2 (with renewals + new clients)
| Metric | Value |
|:-------|:------|
| Retained clients (80% renewal) | 4 |
| New clients | 5-10 |
| Renewal revenue | 60,000 INR |
| New client revenue | 1,25,000 - 2,50,000 INR |
| **Total gross revenue** | **1,85,000 - 3,10,000 INR** |

### Break-even for full-time
| Metric | Value |
|:-------|:------|
| Target monthly income | 50,000 INR |
| Annual target | 6,00,000 INR |
| Clients needed at 15,000/year | ~40 clients |
| Clients needed at 18,000/year | ~34 clients |

---

## Sales Process

### Lead Generation
1. Personal network (schools, hospitals in Delhi NCR)
2. Walk-in visits to IT departments
3. Former WiJungle / Hosto contacts
4. LinkedIn outreach to school IT admins
5. Local business associations / chambers of commerce

### Sales Cycle
1. **Discovery call** (15 min) — learn their current setup, pain points
2. **Free trial** (1 week) — install Pi, let them see the dashboard
3. **Review meeting** (30 min) — show blocking stats from their own network
4. **Close** — annual contract, payment upfront
5. **Onboarding** — configure policies per their requirements

### The Free Trial Close
> "Let me install this for a week — free, no commitment. At the end of the week, I'll show you exactly what's been happening on your network. If you don't want to keep it, I take the device back. No cost."

This works because:
- After 1 week, the dashboard shows thousands of blocked queries
- Removing it feels like removing protection
- The data is specific to THEIR network — not a generic demo

---

## Contract Terms

- **Term:** 12 months, auto-renewing with 30-day cancellation notice
- **Payment:** Annual upfront (UPI/bank transfer/cheque)
- **Hardware:** Client owns the Pi after purchase. We manage it remotely.
- **SLA:** Best-effort support. No uptime guarantee for v1 (add SLA with ISO 27001).
- **Data:** All DNS query data stays on the client's device. We access remotely for management only.
- **Cancellation:** Pro-rata refund for unused months in first year only.

---

## Open Source Strategy

**Why keep it open-source:**
- Portfolio piece — shows real engineering capability
- Community contributions improve the product for free
- Trust signal — clients can verify there's no telemetry/backdoor
- Attracts developer attention and potential co-founders

**Why clients still pay:**
- They're not paying for the software — they're paying for the service
- No school IT admin will self-host, configure, and maintain a DNS filter
- The value is: "I manage it so you don't have to think about it"
- Analogy: Linux is free, but Red Hat charges for support. WordPress is free, but agencies charge for websites.

---

## Risks

| Risk | Mitigation |
|:-----|:-----------|
| Client churns after year 1 | Show annual report with blocking stats before renewal. "We blocked 2M queries." |
| Client's IT person sets it up themselves | Unlikely — your target is non-technical buyers. If they can DIY, they're not your customer. |
| Pi hardware fails | Keep spare Pis. Offer next-day replacement. |
| Competitor undercuts on price | You're already the cheapest managed option. Race to bottom = not your market. |
| Can't reach 40 clients for break-even | Start part-time alongside other work. HydraDNS is recurring — every new client compounds. |
| Internet outage at client site | HydraDNS still serves cached blocklist. When internet returns, everything resumes. |

---

## Metrics to Track

| Metric | Target |
|:-------|:-------|
| Monthly new leads | 10+ |
| Free trials installed | 3-5/month |
| Trial → paid conversion | 60%+ |
| Annual renewal rate | 80%+ |
| Avg revenue per client | 15,000-18,000 INR/year |
| Client satisfaction (NPS) | 8+/10 |
| Time to onboard new client | < 1 hour |
| Active managed devices | Track monthly |
