# HydraDNS Demo Playbook

**Purpose:** Step-by-step guide to prepare for and deliver a live demo at a school, hospital, or SMB.
**Last verified:** 2026-04-08 (all E2E tests pass)

---

## Pre-Demo Checklist (Do the night before)

### Hardware
- [ ] Raspberry Pi 4 (2GB+ RAM) with power supply
- [ ] Ethernet cable (Pi must be wired, not WiFi)
- [ ] MicroSD card with Raspberry Pi OS (or USB SSD for reliability)
- [ ] Your laptop (for dashboard demo)
- [ ] Phone (to show mobile dashboard access)

### Software Setup (on the Pi)
```bash
# 1. Install Docker (if not already)
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
# Log out and back in

# 2. Install HydraDNS
curl -fsSL https://raw.githubusercontent.com/hydradns/hydradns/main/scripts/install.sh | bash

# 3. Verify it's running
docker compose ps        # Should show core as "healthy"
curl http://localhost:8080/health   # Should return {"status":"ok"}
```

### Pre-Configure (so the demo is instant)
```bash
# Complete setup in advance with a demo password
curl -X POST http://localhost:8080/api/v1/auth/setup \
  -H "Content-Type: application/json" \
  -d '{
    "password": "hydrademo",
    "blocklists": [
      {
        "id": "stevenblack",
        "name": "StevenBlack Unified",
        "url": "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
        "format": "hosts"
      }
    ]
  }'

# Wait 60 seconds for blocklist to load, then verify
sleep 60
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"password":"hydrademo"}'
# Save the token for CLI use
```

### Verify Before Leaving Home
```bash
# DNS works
dig @localhost ads.google.com    # Should return REFUSED (blocked)
dig @localhost github.com        # Should resolve normally

# Dashboard accessible
# Open http://<pi-ip>:3000 on your laptop browser — should show login page

# Note the Pi's IP address
hostname -I    # e.g., 192.168.1.100
```

---

## The Demo (5 minutes)

### Setup (30 seconds)
1. Plug the Pi into their router via ethernet
2. Power it on
3. Wait ~90 seconds for Docker to start (the Pi should already have images cached from your home setup)
4. If this is a fresh deployment, the setup wizard appears at `http://<pi-ip>:3000`

### Step 1: Point their router's DNS at the Pi (2 minutes)
Open their router admin panel and change the DNS server to the Pi's IP address. See `docs/pi-deployment.md` for router-specific instructions.

**Talk track while configuring:**
> "Right now, every device on your network sends DNS requests — that's the system that converts website names to IP addresses — to your ISP. Your ISP doesn't filter anything. Malware sites, phishing pages, ad trackers — all resolve normally. I'm changing one setting so all those requests go through this device first."

### Step 2: Show the Dashboard (1 minute)
Open `http://<pi-ip>:3000` on their laptop browser. Log in with the demo password.

**What they see:**
- KPI cards: Total queries, blocked queries, allowed queries, block rate
- DNS Engine status: Running
- Numbers updating in real-time as their devices make DNS queries

**Talk track:**
> "This is your network's DNS activity in real-time. See those numbers going up? That's every device on your network — computers, phones, printers — all their DNS queries are now flowing through HydraDNS."

### Step 3: Show Blocking in Action (1 minute)
On their laptop browser, try to visit a known blocked domain:
- `http://ads.google.com` — will fail to load
- Open the Query Logs page in the dashboard — show the "block" entry appear immediately

**Talk track:**
> "See this? ads.google.com just got blocked. Every ad tracker, known malware domain, and phishing site is blocked before it can even connect. We're blocking from a list of over 90,000 known bad domains, updated automatically every 6 hours."

### Step 4: Show AI Threat Detection (30 seconds)
In the Query Logs page, point out any entries with the red "THREAT" badge.

If no suspicious domains have appeared naturally, trigger one:
```bash
# From your laptop terminal
dig @<pi-ip> a1b2c3d4e5f6a7b8c9d0.evil.com
```

**Talk track:**
> "This is what makes HydraDNS different from a simple blocklist. See this domain flagged in red? Our AI detection caught it as a suspicious domain — it has patterns that match malware command-and-control servers. Even if a threat isn't on any blocklist yet, we catch it."

Show the threat score (90%) and detection method (dga_hex) in the dashboard.

### Step 5: Show Policy Control (30 seconds)
Go to the Policies page. Create a new block policy:
- Name: "Block Social Media"
- Action: BLOCK
- Domains: `tiktok.com, instagram.com, facebook.com`

Then try to visit `tiktok.com` on their browser. It fails. Show the block entry in Query Logs.

**Talk track:**
> "You can block any category of sites in seconds. During school hours, block social media. During exams, block everything except educational sites. Changes take effect within 5 seconds — no restart needed."

### Step 6: Close (30 seconds)
> "This device costs ₹8,000 to install. I manage it remotely for ₹1,500/month. Every device on your network is protected — no software to install on individual machines. Let me leave this running for a week so you can see what's happening on your network. I'll check in next [day]."

---

## Demo Failure Modes & Recovery

| Problem | Quick Fix |
|---------|-----------|
| Pi won't boot | Have a backup Pi or run from your laptop via Docker |
| Dashboard shows "API unreachable" | Wait 60 seconds for Docker containers to start. Check `docker compose logs core` |
| DNS queries not being blocked | Check router DNS settings point to Pi IP. Try `dig @<pi-ip> ads.google.com` to confirm |
| Query logs empty | DNS might still be going through old resolver. Flush DNS on laptop: `ipconfig /flushdns` (Windows) or restart browser |
| Blocklist shows 0 domains | Blocklist is still loading. Wait 60 seconds and refresh |
| "Setup required" error | Run setup via API (see Pre-Configure section above) |
| Port 53 conflict on Pi | Run `sudo systemctl disable --now systemd-resolved` on the Pi |

## Laptop Backup Plan
If the Pi fails, you can run the demo from your laptop:
```bash
git clone --recursive https://github.com/hydradns/hydradns.git
cd hydradns
docker compose up -d core
# Dashboard won't be available via Docker, but you can show the API:
curl http://localhost:8080/api/v1/dashboard/summary
# And DNS:
dig @localhost -p 53 ads.google.com
```

---

## What's Verified Working (E2E test results)

| Feature | Status | Details |
|---------|--------|---------|
| Setup wizard | PASS | 2-step: password + blocklists |
| Authentication | PASS | Bearer token on all endpoints, 401/403 correct |
| Blocklist blocking | PASS | 92,283 domains from StevenBlack, ads.google.com REFUSED |
| Policy blocking | PASS | API-created policies enforced within 5s, tiktok.com REFUSED |
| AI threat detection | PASS | DGA hex (0.9), subdomain depth (0.5), entropy (0.4) flagged |
| Query logging | PASS | All queries logged with threat fields, async writes |
| Dashboard stats | PASS | Total/blocked/allowed counts update in real-time |
| Blocklist auto-refresh | PASS | Refreshes every 6h (configurable), ETag caching |
| DNS forwarding | PASS | Clean domains resolve normally (github.com → IP) |

---

## Demo Domains to Use

### Blocked by blocklist (instant, no setup needed)
- `ads.google.com` — ad tracker
- `tracking.example.com` — if on StevenBlack list
- `malware.wicar.org` — test malware domain (may not be on StevenBlack)

### Blocked by policy (create during demo)
- `tiktok.com` — social media (resonates with schools)
- `instagram.com` — social media
- `facebook.com` — social media

### Flagged by AI detection (use dig from terminal)
- `a1b2c3d4e5f6a7b8c9d0.evil.com` — DGA hex pattern (score: 0.9)
- `a.b.c.d.e.f.g.evil.com` — deep subdomain (score: 0.5)
- `xkq7mz9plw2vb8nt.malware.org` — high entropy (score: 0.4)

### Normal domains (should resolve fine)
- `google.com` — everyone's first test
- `github.com` — for your credibility
- `wikipedia.org` — educational

---

## Post-Demo Follow-Up

1. **Leave it running** — let them experience it for a week
2. **Check in after 3 days** — "How's the network? Noticed anything different?"
3. **Show them the dashboard after a week** — cumulative stats are impressive (thousands of blocked queries)
4. **Ask:** "Would you like to continue using this? The monthly management fee is ₹1,500."
