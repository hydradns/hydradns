# HydraDNS — Certification Roadmap

**Sorted by cost (cheapest first), with difficulty rating and India market relevance.**

---

## Phase 1: Free / Near-Free (Do This Month)

| # | Certification | Cost | Time | Difficulty | How |
|---|:-------------|:-----|:-----|:-----------|:----|
| 1 | **Startup India (DPIIT Recognition)** | Free | 2-4 weeks | Easy | Self-declaration on startupindia.gov.in. Tax benefits, credibility badge, govt procurement access |
| 2 | **CERT-In Compliance** | Free | 4-8 weeks | Medium | Mandatory for cybersecurity providers. Implement: 6-hr incident reporting, 180-day log retention, NTP timestamps |
| 3 | **DPDP Act Compliance** | Free (legal review ~50K-2L INR) | 4-12 weeks | Medium | Self-assess. Publish privacy policy, data processing practices. Be ready before enforcement begins |
| 4 | **CIPA-Compatible** | Free | N/A | Easy | Marketing badge — no certification body. Just ensure your product can filter content per CIPA requirements |
| 5 | **GDPR Self-Assessment** | Free | 4-8 weeks | Medium | Publish data processing documentation. No formal cert exists — you declare compliance |
| 6 | **CSA STAR Level 1** | Free | 2-4 weeks | Easy | Self-assessment questionnaire (CAIQ). Published on CSA STAR Registry. Good for cloud credibility |
| 7 | **GeM Registration** | Free | 2-3 weeks | Easy | Government e-Marketplace. Essential for selling to govt schools and hospitals in India |

**Total Phase 1 cost: 0 - 2L INR**

---

## Phase 2: Low Cost (Months 4-8)

| # | Certification | Cost | Time | Difficulty | Why |
|---|:-------------|:-----|:-----|:-----------|:----|
| 8 | **NASSCOM Membership** | 15K-50K INR/yr | 2-4 weeks | Easy | Industry credibility. Recognized by Indian enterprises, hospitals, govt |
| 9 | **BIS (CRS) Registration** | 1-3L INR | 2-4 months | Medium | Required if selling the Pi hardware appliance. Software-only may not need it |
| 10 | **ISO 9001:2015** | 3-10L INR | 3-6 months | Medium | Quality management. Sometimes required in hospital/govt tenders |

---

## Phase 3: The Big One (Months 6-12)

| # | Certification | Cost | Time | Difficulty | Why |
|---|:-------------|:-----|:-----|:-----------|:----|
| 11 | **ISO 27001:2022** | 3-5L INR (Indian CBs) | 4-12 months | Hard | **THE most important cert for India.** Required in most govt, hospital, and education tenders. Use Indian auditors (BSI India, TUV India, IRQS) for lower cost |
| 12 | **STQC Certification** | 3-10L INR | 3-9 months | Medium-Hard | **Mandatory for Indian govt IT security procurement.** Through MeitY labs |
| 13 | **SOC 2 Type I** | 10-25L INR | 2-4 months | Medium-Hard | Point-in-time security audit. Expected by enterprise/hospital buyers |

---

## Phase 4: Scale (Year 2, When Revenue Supports It)

| # | Certification | Cost | Time | Difficulty | Why |
|---|:-------------|:-----|:-----|:-----------|:----|
| 14 | **SOC 2 Type II** | 25-50L INR | 6-12 months | Hard | Gold standard — proves sustained security over time |
| 15 | **ISO 27701** | 8-20L INR (incremental to 27001) | 2-4 months | Hard | Privacy management. Maps directly to DPDP Act. Strong for hospital data |
| 16 | **Common Criteria EAL2** | 80L+ INR | 6-24 months | Very Hard | Only for govt/defense contracts. Overkill for schools/SMBs |

---

## What To Do Right Now

### This Week
1. Register on **Startup India** (DPIIT) — free, takes 30 minutes to apply
2. Register on **GeM** (Government e-Marketplace) — free, opens govt procurement
3. Start **CERT-In compliance** — implement 180-day log retention (we need this feature in the code)

### This Month
4. Publish **privacy policy** and **data processing documentation** (DPDP Act readiness)
5. Complete **CSA STAR Level 1** self-assessment
6. Apply for **NASSCOM membership**

### Within 6 Months
7. Begin **ISO 27001** prep — this unlocks hospital and govt tenders

---

## CERT-In Requirements (Mandatory — Implement in Code)

These are legally required for any cybersecurity service provider in India:

| Requirement | Status in HydraDNS | Action Needed |
|:------------|:-------------------|:--------------|
| 6-hour incident reporting | Not implemented | Add incident detection + reporting mechanism |
| 180-day log retention | Not implemented | Add configurable retention (currently unbounded) |
| NTP-synchronized timestamps | Partial (uses system time) | Ensure NTP sync documented in deployment guide |
| Maintain logs of subscribers/customers | Not implemented | Add client tracking for managed service |
| Designate Point of Contact for CERT-In | Organizational | Appoint and register with CERT-In |

---

## Cost Summary by Phase

| Phase | Timeline | Total Cost | Certs Gained |
|:------|:---------|:-----------|:-------------|
| Phase 1 | Month 1-3 | 0 - 2L INR | DPIIT, CERT-In, DPDP, CIPA, GDPR, CSA STAR, GeM |
| Phase 2 | Month 4-8 | 2-5L INR | NASSCOM, BIS, ISO 9001 |
| Phase 3 | Month 6-12 | 10-20L INR | ISO 27001, STQC, SOC 2 Type I |
| Phase 4 | Year 2 | 30-70L INR | SOC 2 Type II, ISO 27701, CC EAL2 |

---

## Key Insight

**ISO 27001 is the unlock.** In India, the single certification that opens the most doors for school, hospital, and government sales is ISO 27001. It's achievable within 6-12 months at 3-5L INR through Indian auditors. Every other enterprise cert builds on it. Prioritize this after the free Phase 1 items.
