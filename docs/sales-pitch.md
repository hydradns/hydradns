# HydraDNS — Sales Hook & Call Pitch

**Last updated:** 2026-05-06
**Audience:** Schools, small hospitals, and small businesses (10-500 staff) in Delhi NCR. Non-technical decision makers who own the network and the budget.

This doc is for the seller, not the buyer. Re-read the "Why you should be confident" section before every call. Everything below is calibrated to what HydraDNS actually does today on a real Raspberry Pi, not what it might do in a future release.

---

## Why you should be confident

You are not selling Umbrella. You are not selling Cloudflare Gateway. You are not even selling against them. **Your real competitor is "doing nothing"**, which is what 95% of small Delhi NCR networks actually have right now.

Five things only HydraDNS has at this price point:

1. **All queries stay on-premises.** NextDNS sees every domain your client visits. Cloudflare sees them. OpenDNS sees them. HydraDNS does not — the box is theirs, the data is theirs. That is a genuine sovereignty pitch and you should lead with it for hospitals.
2. **AI-native control.** You can manage it conversationally through Claude or Gemini via MCP. No incumbent has this. It is a wedge, not a feature.
3. **Open source, auditable.** Nobody on staff is going to read the source, but the fact that it is auditable is the answer to "what if you disappear tomorrow?". The Pi is theirs, the software runs forever.
4. **Flat-fee pricing.** A 50-person school and a 500-person school pay the same. No-one in this market matches that.
5. **You install it personally.** They are not buying software. They are buying a person who shows up, plugs in a box, configures it for their specific network, and hands them a working dashboard. That is the deliverable.

The product runs. It blocks 82,000+ malicious domains the moment you plug it in. It catches phishing and malware in real time. The dashboard is real, the logs are real, the demo will work.

When you feel underconfident on a call, it is almost always because you are accidentally comparing to Umbrella in your head. Stop. The buyer does not have Umbrella. The buyer has a TP-Link router and antivirus on three of seven laptops. You are 100x better than what they have.

---

## Opening Hook (15 seconds)

> "Right now, every device on your network can quietly reach phishing sites, malware drops, and adult content, and you have zero visibility into any of it. I sell a small box that plugs into your router, blocks all of that, and gives you a live dashboard of what was blocked and on which device. Costs less than your monthly cleaning bill. Worth 10 minutes?"

### Why this works
- **Concrete fear, not jargon.** Not "DNS layer threat surface". Words a principal or a hospital admin uses: phishing, malware, adult content.
- **Quantifies the gap.** "Zero visibility" is true and uncomfortable. They have no idea what is on their network right now.
- **Visible deliverable.** The dashboard is a thing they can picture, not a service.
- **Disqualifies politely.** "Worth 10 minutes?" is a fair ask, lets them say no without losing face.

### Variants by persona

**School principal / admin:**
> "Your students are reaching social media, gaming sites, and worse during class hours, and you have no way to see it or stop it. I install a small device that gives you a one-page dashboard of every site every device tried to visit, and one-click category blocking. 23,000 rupees the first year. I do the install. Worth 10 minutes?"

**Hospital admin:**
> "If a single ransomware DNS lookup succeeds on your network, you are paying lakhs to recover and explaining it to NABH. Enterprise blocking costs 3 lakh a year and ships your patient query data to a foreign cloud. I sell the same protection on a box that lives in your server room. 23,000 rupees the first year, 15,000 a year after, your data never leaves your network. Worth 10 minutes?"

**Small business owner:**
> "Your staff is on social media half the day, and your network is one phishing email away from a full breach. Pi-hole on a Raspberry Pi blocks all of it for free, but nobody on your team is going to set it up or maintain it. I sell a managed version. 23,000 rupees, plug, dashboard, done. Worth 10 minutes?"

**Technically literate IT manager (rare but happens):**
> "You probably know about Pi-hole and AdGuard Home. HydraDNS is the same idea with a real management API, AI-native control through MCP, and a managed-service contract so you don't have to maintain it yourself. Open source, queries never leave your LAN. 23,000 rupees the first year. Want to see the dashboard?"

---

## The 90-Second Call Pitch

The previous version of this doc had a 5-minute pitch. Cut it. 5 minutes is a presentation, not a sales call. Here is the 90-second version. If they want more, they will ask.

### Beat 1 — The gap (20s)

> "Most schools and small businesses have a Wi-Fi router and that is it. The router does not block bad sites, does not log activity, and gives you no control. The only alternatives today are enterprise firewalls from Cisco that cost 3 lakh a year, or self-hosted tools like Pi-hole that nobody on staff is going to set up. I sit in the gap."

### Beat 2 — What it is (20s)

> "A small device, a Raspberry Pi, plugged into your router. Every DNS lookup on your network goes through it. If a device tries to reach a known malware site or a phishing domain, the lookup fails and the page never loads. You get a web dashboard of total queries, blocked queries, and which device hit what. Categories are one-click: malware, phishing, gambling, adult, social media, ads."

### Beat 3 — Make it real (20s)

> "*(Open dashboard or screenshot.)* This is a real network running for one week. 47,000 lookups, 12,300 blocked, 26 percent block rate. Top blocked sites are doubleclick.net, googletagmanager.com, and three malware domains caught in real time. Your network would look similar within 24 hours."

### Beat 4 — Price (15s)

> "23,000 rupees the first year, 15,000 a year after that. That includes the device, the install, the dashboard, blocklist updates, and remote management. No per-user pricing. A 500-person network pays the same as a 50-person one."

### Beat 5 — Ask (15s)

> "Free one-week trial. I install it, you see your own data, and at the end of the week if you don't want to keep it I take the device back. What does your network look like — how many devices, what do you currently use?"

Stop talking. Wait for the answer.

---

## The Demo — show, don't tell

You do not need to explain DNS. The dashboard explains it.

1. **Plug the Pi into their router** in front of them. Two minutes.
2. **Open the dashboard on their laptop.** Total queries already ticking up. KPI cards filling in.
3. **Open instagram.com on the same laptop.** Page fails. Switch to the Logs tab. Show the `instagram.com` row appearing with action `block`. They see the cause and effect within a minute of the box being plugged in.
4. **Open a known malware test domain** (e.g. one Steven Black includes). Same thing. Logs show `block` with the source list.
5. **Add `linkedin.com` to a block list via the dashboard.** Refresh in their browser. It fails. Remove the rule. It works again. Sub-second feedback loop. This is the moment that closes the deal.
6. **Leave it running.** "I'll come back in a week."

What they will remember from this demo: blocking is instant, the dashboard is real, and they can see their own staff's traffic. That is enough.

---

## Objection Handling

Objections in this market are mostly anxiety, not analysis. Treat them as a request for reassurance.

| Objection | Response |
|---|---|
| **"We already have a firewall."** | "A firewall blocks ports and IPs at the network edge. It does not block by category, does not log per device, and most school firewalls let everything through on ports 80 and 443 by default. HydraDNS is a different layer. They work together, not against each other." |
| **"We have antivirus."** | "Antivirus catches a malicious file once it has landed on a device. HydraDNS prevents the device from ever reaching the malicious site. It is a layer up, and it covers the devices that cannot run antivirus — phones, smart TVs, IP cameras, guest Wi-Fi." |
| **"Why not just use OpenDNS or Quad9?"** | "Those are upstream resolvers, they block based on a generic global list. They do not give you a dashboard, do not log per device, do not let you whitelist or set custom rules, and you have nobody to call when something breaks. Also, every query you make is sent to their cloud. HydraDNS does the same blocking on a box you own, with a dashboard, and your queries never leave your network." |
| **"15,000 rupees a year is a lot for software."** | "It is roughly 1,250 rupees a month. Compare it to one phishing breach. The Indian average breach cost in 2024 was 17.9 crore. You are buying insurance, plus you are buying time — your staff stops being the ones managing this." |
| **"Can we just buy the device and not pay the renewal?"** | "Yes. The software is open source, the Pi is yours, you can run it forever. The annual fee buys blocklist updates, software updates, monitoring, and the support phone number. Without that, the box becomes stale and you are on your own. Most clients keep the renewal because finding out something broke a week after the fact is exactly the problem they were trying to solve. But the choice is yours, no lock-in." |
| **"How do I know you'll still be around next year?"** | "Two reasons. One, the software is open source and the Pi is yours — even if I disappear, the device keeps blocking, and any technical person can pick up the source. Two, my contact details and source repos are in your contract. You are not buying a SaaS subscription that vanishes when the company folds. You are buying hardware you own and software you can audit." |
| **"Send me an email."** | "I will. But before I do — would you rather I ship a one-page summary or come install it for free for a week? The week is more useful because the data is your network, not mine. Either way, summary is in your inbox tonight." |
| **"We need to discuss internally."** | "Of course. While you discuss — can I leave the device with you for the week? You can decide after seeing your own dashboard. If the answer is no after a week, I take it back. Costs nothing." |
| **"What if a student just types the IP address directly and bypasses DNS?"** | "True, that bypasses any DNS-based filter — Cisco's, Cloudflare's, ours, all of them. DNS-layer blocking handles roughly 95% of normal traffic, including almost all malware (which uses DNS because it has to be reliable) and almost all phishing (the click goes through DNS). The 5% who know how to type an IP address are not the problem we are solving. We are not the only line of defence — we are the most cost-effective first line. Endpoint AV handles the rest." |
| **"What about DNS-over-HTTPS in the browser?"** | "Real concern. Modern browsers can encrypt DNS to Cloudflare or Google directly and bypass us. Two answers: one, we ship a curated blocklist of known DoH endpoints so when a browser starts up and asks for `dns.google`, that lookup itself fails and the browser falls back to system DNS, which is us. Two, on managed devices a Group Policy entry disables browser DoH outright. We document both during installation." |
| **"What if someone uses a VPN?"** | "A VPN tunnels all traffic, including DNS, around any local filter. We do not stop VPNs — that is a network firewall and endpoint policy job. What we do is flag suspicious VPN endpoints when they appear in DNS, so you know which devices started using one. Combined with a small allowed-VPN list at the firewall level, you keep visibility." |

---

## Defense in Depth — the honest version

DNS filtering is **a layer**, not the whole stack. Sophisticated bypasses exist. You should know exactly what HydraDNS catches and what it doesn't, because the technical buyer will ask, and pretending otherwise is what kills the deal.

### What HydraDNS catches (~95% of relevant traffic)

- Anyone using normal browsing — kids on Chromebooks, staff on iPhones, IoT devices.
- Almost all malware command-and-control. Malware authors use DNS because their infrastructure changes daily; hard-coded IPs go stale fast.
- Almost all phishing — the link in the email points to a domain, and the click triggers a DNS lookup.
- Ad networks, trackers, telemetry, analytics. These all use long DNS chains that we kill at the first hop.
- Accidental visits to malicious domains via search-result poisoning or compromised sites.

### What HydraDNS does NOT catch on its own

- A user who types an IP address directly into the browser. No DNS query, no block.
- Browser-level DNS-over-HTTPS (DoH) when the browser bootstraps from a hard-coded IP.
- VPN traffic — everything tunnels.
- Hard-coded IP malware (rare but exists).
- Hosts-file overrides on a managed device.

### What we ship to mitigate the bypasses we can mitigate

- **DoH bootstrap blocklist.** Curated list of hostnames every browser uses to find its DoH server (`dns.google`, `cloudflare-dns.com`, `mozilla.cloudflare-dns.com`, `dns.quad9.net`, etc.). When the browser tries to resolve them at startup, HydraDNS returns NXDOMAIN. Browser falls back to system DNS, which is us. Defeats default-on browser DoH for >80% of cases.
- **Router firewall snippets.** During install, on routers that support it (pfSense, MikroTik, OpenWrt, prosumer ASUS), we set:
  - Block outbound port 53 from any device except the HydraDNS Pi. Forces all DNS through us.
  - Block outbound port 853 (DoT). Same reason.
  - Optional: redirect known DoH provider IPs (`1.1.1.1`, `8.8.8.8`, `9.9.9.9`) at the router so the browser cannot reach them on `:443` either.
- **Bypass detection on the dashboard.** Devices that suddenly stop querying HydraDNS get flagged. That's the signal of a roaming user, a VPN, or a manual DNS change.
- **Group Policy templates** for Windows-managed networks (schools, hospitals): disable browser DoH, lock the DNS server setting, prevent users from editing the hosts file.

### What you say to the buyer

> "We are honest about what DNS filtering can and can't do. It catches the 95% of traffic that goes through normal channels — almost all malware, almost all phishing, all the ad and tracker noise. It does not catch a determined user who knows how to type an IP address. That 5% is what your endpoint AV and your network firewall are for. Defense in depth means three layers — DNS, endpoint, network — and we are the first one because it's the cheapest and gives you the most visibility per rupee."

The technical buyer who asks the IP-bypass question is testing whether you understand the limits of your own product. Answering honestly is what closes the deal.

---

## What NOT to say

- **"AI-powered."** Save it for the threat-detection moment. Used too early it sounds like marketing.
- **"Enterprise-grade."** It is not, and the buyer can tell. Stick to "designed for small networks like yours".
- **"Replaces your firewall."** It does not. Different layer. Saying this loses you the technical buyer.
- **"Zero false positives."** Blocklists have noise. The dashboard's whitelist function is the answer.
- **"Compatible with all networks."** Most home-grade routers cannot be told to use a custom DNS cleanly. Establish what router they have before promising.

---

## Confidence Anchors — things to remember when the call goes sideways

When the buyer pushes back hard, your instinct will be to discount or apologise. Don't. Read these:

- The dashboard worked in the demo. The blocking is real. You watched it happen.
- 82,000+ malicious domains are blocked the moment you plug it in. That is a real number.
- The price is fair. 23,000 rupees for hardware + install + a year of management is below cost in most markets.
- They have nothing right now. Whatever you ship, it is more than what they have.
- Most "no"s in this market mean "I need to think about it". Leave the device for a week and most of those become "yes".
- You are not selling enterprise software. You are selling **a person** who shows up and makes their network safer. That is the actual product.

---

## Discovery Questions (use during the trial setup)

Skip none of these.

1. How many devices are on the network roughly? (Sets expectations for query volume.)
2. Who manages the router today? Do you have admin access? (No admin access = trial cannot proceed.)
3. What is your Internet provider and router model? (Determines if we can override DNS cleanly.)
4. What categories do you definitely want blocked? (Sets day-one default policy.)
5. Are there sites that absolutely cannot be blocked? (Common: bank portals, school CMS, payroll, EHR systems.)
6. Do you have any compliance requirements? (Hospital → ask about HIPAA-equivalent / NABH expectations. School → parental control reporting.)

---

## The After-Trial Close

End-of-week meeting. On their site. With their real numbers from the dashboard.

> "Here is what happened on your network this week. *(Show the dashboard, full screen.)* 47,000 lookups. 12,300 blocked. The top three blocked domains were *(read the names)*. There were two suspicious domains we caught at 3 AM — most likely an IoT device phoning home. If you want to keep this running, the annual is 23,000 today and 15,000 every year after. I'll have the box configured exactly like this within an hour. If you want me to take it back, I can do that today too."

The numbers do the work. The dashboard is the closer. **Do not pitch in this meeting.** Just read what happened and ask the question.

---

## When the trial fails

It will, sometimes. Most common reasons:

- **Router cannot have its DNS overridden** (cheap ISP routers). Workaround: install the Pi as DHCP server, or use the router's "DNS Relay" feature. If neither is possible, walk away politely.
- **Buyer never opens the dashboard during the week.** Pre-empt this. Send them a 2-line WhatsApp every other day: "saw 47 phishing attempts blocked yesterday — you can see them at 192.168.x.x:3000". Engagement during the trial is what makes the close trivial at the end.
- **Buyer has a real IT person who insists on Umbrella or similar.** That is fine. Walk away. Your buyer is not the IT person, it is the principal or admin who signs the cheque. Leave a one-pager with the principal anyway.

---

## Final note to seller

The pitch is honest. The price is fair. The product works. Your job is to put the device in front of them long enough for the dashboard to do the closing.
