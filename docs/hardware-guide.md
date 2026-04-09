# HydraDNS — Hardware Guide

**Last updated:** 2026-04-09

---

## Minimum Requirements

DNS filtering is extremely lightweight. Here's what HydraDNS actually needs:

| Resource | Minimum | Recommended | Why |
|:---------|:--------|:------------|:----|
| CPU | 4x ARM Cortex-A53 @ 1.0 GHz | 4x A53 @ 1.5 GHz+ | A single A53 core handles 5,000+ DNS qps. SMB peak is ~100 qps. Massively overkill. |
| RAM | 1 GB | 2 GB | Go runtime (~30MB) + 100K blocklist (~20MB) + SQLite (~15MB) + Docker (~100MB) + UI (~120MB) = ~320MB. 1GB is tight, 2GB comfortable. |
| Storage | 8 GB | 16-32 GB | 180-day logs for 50 devices = ~3.6 GB. For 200 devices = ~18 GB. |
| Ethernet | 100 Mbps | Gigabit | DNS traffic is tiny (~10 Mbps at 10K qps). GbE is for perception, not need. |
| WiFi | Optional | Nice to have | For setup/management. Not needed for DNS filtering. |
| Power | 3W | 5W | Lower = cheaper PSU, less heat, more reliable. |

---

## Hardware Options (Cheapest First)

### Tier 1: Budget (~3,500 - 5,500 INR)

| Board | CPU | RAM | Storage | Ethernet | WiFi | Price (INR) | Best For |
|:------|:----|:----|:--------|:---------|:-----|:------------|:---------|
| **Orange Pi Zero 3 (1GB)** | 4x A53 @ 1.5 GHz | 1 GB | microSD | 1x GbE | Yes | ~3,999 | Absolute cheapest Docker-capable device |
| **NanoPi R2S** | 4x A53 @ 1.5 GHz | 1 GB | microSD | **2x GbE** | No | ~4,050 | Dual ethernet — natural inline device |
| **Orange Pi Zero 3 (2GB)** | 4x A53 @ 1.5 GHz | 2 GB | microSD | 1x GbE | Yes | ~4,500-5,000 | **Best value. Recommended for bulk.** |
| **Raspberry Pi 4 (2GB)** | 4x A72 @ 1.5 GHz | 2 GB | microSD | 1x GbE | Yes | ~4,500-5,200 | Best community support, easy dev |

### Tier 2: Mid-Range (~5,500 - 10,000 INR)

| Board | CPU | RAM | Storage | Ethernet | WiFi | Price (INR) | Best For |
|:------|:----|:----|:--------|:---------|:-----|:------------|:---------|
| **Orange Pi Zero 3 (4GB)** | 4x A53 @ 1.5 GHz | 4 GB | microSD | 1x GbE | Yes | ~5,500-6,500 | Extra headroom for larger networks |
| **Raspberry Pi 4 (4GB)** | 4x A72 @ 1.5 GHz | 4 GB | microSD | 1x GbE | Yes | ~5,800-6,500 | Prototyping and demos |
| **Raspberry Pi 5 (4GB)** | 4x A76 @ 2.4 GHz | 4 GB | microSD | 1x GbE | Yes | ~6,500-7,500 | Fastest Pi, overkill for DNS |
| **NanoPi R4S (4GB)** | 2x A72 + 4x A53 | 4 GB | microSD | **2x GbE** | No | ~7,000-8,000 | Dual ethernet, higher throughput |
| **NanoPi R5S (4GB)** | 4x A55 @ 2.0 GHz | 4 GB | **32GB eMMC** | **3x GbE (1x 2.5G)** | No | ~8,000-10,000 | Built-in storage, triple ethernet |

### Tier 3: Enterprise (~12,000+ INR)

| Device | CPU | RAM | Storage | Ethernet | Price (INR) | Best For |
|:-------|:----|:----|:--------|:---------|:------------|:---------|
| **Intel N100 Mini PC** | 4C/4T x86 @ 3.4 GHz | 8-16 GB | 128-512 GB SSD | 1-2x GbE | ~12,000-16,500 | Hospitals, large networks, high reliability |
| **Banana Pi BPI-R3** | 4x A53 @ 2.0 GHz | 2 GB | 8GB eMMC | **5x GbE + 2x SFP** | WiFi 6 | ~12,000-15,000 | Router replacement, max ethernet |

---

## Recommendation by Deployment Phase

| Phase | Hardware | Unit Cost | Total with Accessories | Why |
|:------|:---------|:----------|:----------------------|:----|
| **Demo / Pilot** | Raspberry Pi 4 (2GB) | 5,000 | ~6,500 (+ case, SD, PSU) | Best docs, community, easy to demo |
| **First 10-50 clients** | Orange Pi Zero 3 (2GB) | 4,500 | ~5,400 (+ case, SD, PSU) | 30% cheaper than Pi, same capability |
| **Bulk 100-500 units** | Orange Pi Zero 3 (2GB) bulk | 4,000 | ~4,800 (negotiate direct) | Volume pricing from distributor |
| **1,000+ units** | Custom ODM (H618 SoM) | 1,500-2,500 | ~2,000-3,000 | Chinese SoM + Indian assembly |
| **Hospital / Enterprise** | Intel N100 Mini PC | 12,000 | ~14,000 | SSD reliability, x86 ecosystem |

---

## Cost Comparison: Hardware per Client

| Hardware | Unit Cost (INR) | Annual Price (INR) | Hardware as % of Year 1 |
|:---------|:----------------|:-------------------|:-----------------------|
| Raspberry Pi 4 (2GB) | 6,500 | 15,000 | 30% |
| Orange Pi Zero 3 (2GB) | 5,400 | 15,000 | 26% |
| Bulk Orange Pi (100+) | 4,800 | 15,000 | 24% |
| Custom ODM (1000+) | 3,000 | 15,000 | 17% |

At scale, hardware drops to 17% of year 1 revenue. By year 2 (renewal only), it's pure margin.

---

## What Competitors Use

| Product | Hardware | Price | Insight |
|:--------|:---------|:------|:--------|
| **Pi-hole** | Any Pi (even Pi Zero) | $15-75 | Runs on 512 MB RAM, no Docker needed (native C daemon) |
| **Firewalla Purple** | Amlogic S922X, 4GB | $319 | ARM-based, closest to our approach |
| **Firewalla Gold Plus** | Intel J4125, 4GB | $468 | x86 for higher throughput |
| **Ubiquiti USG** | Dual-core MIPS, 512MB | $109 | Specialized networking SoC, not general-purpose |
| **Sophos XGS 87** | Custom x86 dual-processor | ~55,000-90,000 INR | Massive overkill for DNS filtering |
| **WiJungle** | Custom x86 | Not public | Indian-made, targets same market |

**Key insight:** Firewalla at $319 (26,000 INR) is the closest product to HydraDNS. We deliver similar functionality at 5,400 INR hardware + 15,000 INR/year. That's a compelling price gap.

---

## Custom Hardware Path (1,000+ Units)

### Bill of Materials (estimated at 1,000 MOQ)

| Component | Cost/Unit |
|:----------|:----------|
| SoC (Allwinner H618) | $3-5 (250-415 INR) |
| 2 GB LPDDR4 | $3-5 |
| 8 GB eMMC | $2-3 |
| GbE PHY + RJ45 | $1-2 |
| PCB + passives + assembly | $3-5 |
| Enclosure (plastic mold) | $2-4 |
| USB-C power supply | $1-2 |
| **Total BOM** | **$15-26 (1,250-2,150 INR)** |

### One-Time NRE Costs

| Item | Cost |
|:-----|:-----|
| PCB design | $5,000-15,000 |
| Injection mold tooling | $5,000-10,000 |
| BIS certification | $3,000-8,000 |
| Firmware BSP | $5,000-10,000 |
| **Total NRE** | **$18,000-43,000 (15-36L INR)** |

Amortized over 1,000 units = 1,500-3,600 INR/unit. Over 5,000 units = 300-720 INR/unit.

### ODM Sources

**SoM + Design:**
- Boardcon (Shenzhen) — Rockchip/Allwinner SBC ODM
- MYIR Tech — Allwinner SoM + custom carrier
- Geniatech — RK3566 OSM modules

**Indian Assembly + PCB:**
- LionCircuits (Bengaluru) — PCB + turnkey PCBA
- Elpro Technologies (Bengaluru) — embedded computer manufacturer

### Recommended SoC

| SoC | Best For | Bulk Price |
|:----|:---------|:-----------|
| **Allwinner H618** | Cheapest option, proven in Orange Pi Zero 3 | ~$3 |
| **Rockchip RK3328** | Dual GbE capable, router form factor | ~$4 |
| **MediaTek MT7986** | Built-in WiFi 6 + hardware NAT | ~$8 |

---

## Optimization: Reduce RAM Usage

The biggest RAM consumer is the Next.js dashboard running in a Node.js container (~120 MB). To run on 1 GB devices:

| Optimization | RAM Saved | Effort |
|:-------------|:----------|:-------|
| Export Next.js as static files, serve via nginx | ~100 MB | Medium — build static export, replace Node container with nginx |
| Run Go services without Docker (native binary) | ~80 MB | Medium — use systemd instead of Docker |
| Both | ~180 MB | Brings total to ~150-200 MB. Runs on 512 MB devices. |

For bulk deployment, building a lightweight Armbian image with native binaries (no Docker) is the path to running on the cheapest hardware.

---

## Decision: What to Buy Right Now

**For your current demo setup:** You already have it running on a laptop. No additional hardware needed.

**For first 5 paying clients:** Buy 5x Orange Pi Zero 3 (2GB) at ~4,500 INR each + accessories = ~27,000 INR total investment. Sell at 10,000 hardware + 15,000/year = 25,000 per client year 1. **ROI on first client.**

**Don't invest in custom hardware until you have 50+ clients.** Off-the-shelf SBCs are fine until then.
