# Enterprise Gap Analysis

**Written:** 2026-04-23
**Purpose:** Compare HydraDNS today against commercial DNS security products (Cisco Umbrella, Cloudflare Gateway, DNSFilter, Infoblox BloxOne, NextDNS Teams) to identify what a paying customer expects that HydraDNS does not yet ship. Feeds directly into ROADMAP.md.

## Scope

This is a product gap analysis, not a feature parity checklist. The question being answered is: **"If I walked into a prospect's office tomorrow, what would cost me the sale?"** — not "what could we conceivably build someday".

Ordered by severity of the gap from a buyer's perspective, tagged by the lowest tier of buyer that expects the feature.

---

## Tier A — Table stakes to sell to anyone outside a single household

These are the gaps that kill a deal in the first demo.

### Multi-user access control
- **What we have:** one `AdminCredential` singleton, one bearer token, no concept of users or roles.
- **What buyers expect:** admin / operator / read-only roles, each tied to an identity, with audit attribution per user.
- **Floor:** SSO via SAML and OIDC the moment a company has more than ~10 employees. MFA at the same cutoff.

### Audit trail with user attribution
- **What we have:** Gin access logs, no structured audit of policy/blocklist/engine-state changes.
- **What buyers expect:** "who changed what, when, from where" on every write, queryable and exportable.
- **Floor:** compliance reviewers ask for this in the first week of any evaluation.

### Encrypted DNS (DoH and DoT)
- **What we have:** plain-UDP and TCP on port 53.
- **What buyers expect:** DoH (DNS-over-HTTPS) and DoT (DNS-over-TLS) both inbound (clients resolve *to* HydraDNS encrypted) and outbound (upstream).
- **Floor:** modern Windows and iOS push DoH whether the admin opts in or not. A plain-:53 resolver is increasingly bypassable by default OS settings.

### Categorized threat intelligence
- **What we have:** community hosts lists (StevenBlack, OISD, URLhaus, AdGuard).
- **What buyers expect:** checkbox categories — "block gambling, adult, piracy, newly-registered domains" — fed by a curated database (~50 categories) maintained by a dedicated intel team, refreshed minute-by-minute.
- **Floor:** turning "pick a blocklist URL" into "flip a category switch" is the difference between a DIY tool and a commercial product.

### Per-client / per-group policy
- **What we have:** policies match on domain only; no client identity.
- **What buyers expect:** "finance team strict, lab network permissive" keyed on client IP, MAC, AD/Okta group, or endpoint-agent identity.
- **Floor:** day one for any multi-device environment.

---

## Tier B — Expected at SMB commercial (DNSFilter / NextDNS Teams)

These are what HydraDNS is up against in the $50–$500 / month range.

### Multi-site / multi-tenant control plane
Single pane of glass managing N locations, parent / child org hierarchy, policy inheritance. Today every HydraDNS box is an island.

### Endpoint agent
Lightweight Windows / macOS / iOS / Android client that enforces policy even when the user is off-network (coffee shop, hotel, roaming). DNSFilter's entire value proposition. Network-only DNS firewalls lose the moment a laptop leaves the LAN.

### Branded block page
When a user is blocked, the browser gets a redirect to a company-logo'd page with a self-service "Request unblock" button and a reason. HydraDNS returns `REFUSED` — browser shows a generic "site can't be reached" error.

### Reporting and export
Scheduled email / PDF reports, top-blocked-categories by department, compliance-ready exports (CSV, JSON). Live log tables do not satisfy an auditor.

### SIEM / log forwarding
Syslog, Splunk HEC, Microsoft Sentinel, S3, Datadog. Integration is the whole point for security teams.

### HA / redundancy
Two boxes in active-active or active-passive. Single-node means one reboot equals a network outage, and no enterprise signs up for that.

### DNS tunneling / exfil detection
Long labels, high entropy, unusual NXDOMAIN ratios, fast-flux, DGA detection. DNS firewalls are supposed to catch this — it's a security feature, not a filtering feature.

---

## Tier C — Expected at true enterprise (Umbrella / Zscaler)

Adjacent-market stuff the big vendors bundle. Mostly not worth chasing as a solo dev.

- **Anycast / geo-distributed resolvers** with <20ms SLA globally, 99.99%+ uptime with credit-backed SLA.
- **24/7 NOC + support tiers** with named TAM, escalation paths. Procurement pays more for the phone number than the software.
- **Compliance paperwork.** SOC 2 Type II, ISO 27001, HIPAA BAAs, GDPR DPAs, sub-processor lists, pen-test reports on request. Procurement gates new vendors on this.
- **SSE / SASE adjacency.** Cloudflare Gateway, Zscaler, Netskope extend DNS filtering into URL filtering, CASB, DLP, browser isolation. Pure-DNS players are increasingly "features" of broader stacks.
- **Zero-touch provisioning.** MDM integration (Jamf, Intune), QR-code enrollment, auto-config-profile push for remote devices.
- **API-first + GitOps.** Terraform providers, Kubernetes operators, policy-as-code with PR-based review.
- **Billing / licensing / self-service.** Per-seat or per-query metering, auto-renewal, cloud marketplace listings, self-serve trial → upgrade.

---

## Where HydraDNS already wins

Not all bad news. These are genuine differentiators the incumbents cannot easily match.

- **Full custody.** Self-hosted, no queries leaving the LAN. NextDNS and Cloudflare cannot promise that.
- **AI-native via MCP.** No commercial competitor has a first-class conversational control surface. Real differentiator, and credible — agents are how SMBs will run IT in 3 years.
- **Open source, auditable.** Large buyers ask for source escrow; HydraDNS ships source by default.
- **Price floor.** "Free, GPL, your hardware" undercuts anything with per-seat pricing when the buyer is technical enough to self-host.

---

## Strategic read

1. **Pick the niche where strengths outweigh gaps.** Privacy-focused homelab / power user + small tech-forward teams. Do not try to out-Umbrella Cisco.
2. **The three gaps that cannot be deferred if HydraDNS wants paying customers:**
   1. **RBAC + SSO**
   2. **DoH / DoT (inbound + outbound)**
   3. **Categorized threat-intel feed**
   
   Everything else is deferrable; those three are the ones where a demo falls flat without them.
3. **The MCP story is the wedge.** Nobody is close. Lean into it instead of chasing feature checklists.
4. **HA + multi-site is a cliff.** Going from single-node to active-active with shared state is months of work. Probably the right v2.0 boundary rather than v1.1.

See `ROADMAP.md` phases 6+ for the concrete sequencing.
