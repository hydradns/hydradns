# Deployment Guide

Written for the Raspberry Pi, but applies to any always-on machine (old laptop, mini PC, VPS). The [static IP](#give-the-device-a-static-ip) and [router DNS](#router-dns-configuration) sections cover Linux, macOS, and Windows hosts.

## Hardware Requirements

- Raspberry Pi 4 (2GB+ RAM recommended)
- MicroSD card (16GB+, high-endurance recommended) or USB SSD
- Ethernet connection to your router
- Power supply (5V 3A USB-C)

## Install Docker on Raspberry Pi OS

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com | sh

# Add your user to docker group (avoids sudo)
sudo usermod -aG docker $USER

# Log out and back in, then verify
docker --version
docker compose version
```

## Install HydraDNS

```bash
curl -fsSL https://raw.githubusercontent.com/hydradns/hydradns/main/scripts/install.sh | bash
```

This will:
1. Clone the repository
2. Disable `systemd-resolved` if it's blocking port 53
3. Create `.env` from `.env.example` if you don't already have one, then
   start all services, pulling the published `linux/arm64` images from GHCR
   if a release exists, or building from source locally as a fallback
   otherwise (see `docs/releasing.md`)
4. Print your Pi's IP address and dashboard URL

Note: this means the install-script path's `CORS_ORIGINS` default differs from a bare
`git clone && docker compose up -d` with no `.env` at all. `.env.example` sets
`http://localhost:3000,http://127.0.0.1:3000` (two origins), while `docker-compose.yml`'s own
fallback (used only when no `.env` exists) is `http://localhost:3000`. Both are safe defaults;
this is called out here so the difference doesn't look like a bug if you diff the two setups.

## First-Time Setup

1. Open `http://<pi-ip>:3000` in a browser on any device on your network
2. Complete the setup wizard:
   - Set an admin password
   - Choose upstream DNS providers
   - Select blocklist sources
3. You'll be redirected to the dashboard

> **Dashboard blank or stuck on "loading"?** See [Dashboard not accessible from LAN](#dashboard-not-accessible-from-lan) below. Opening the dashboard by IP (e.g. `http://192.168.1.53:3000`) works with no configuration; a named host (`pi.local`, a reverse-proxy domain) needs one setting added.

## Give the Device a Static IP

Your router will forward every DNS query on the network to this device's IP address. If the device gets its address from DHCP, the router can hand it a different IP after a reboot or lease renewal, and DNS for the entire network silently breaks. Pin the IP **before** configuring the router.

Whichever method you use, note these values first (from the device's current connection): its IP address, the subnet mask (usually `255.255.255.0`, i.e. `/24`), and the gateway (your router's IP, e.g. `192.168.1.1`).

> **Tip:** Set the device's *own* DNS server to a public resolver (e.g. `1.1.1.1`), not to itself. HydraDNS needs working DNS to download blocklists even while its container is restarting.

### Option 1: DHCP Reservation on the Router (recommended)

The router always hands the same IP to the device's MAC address. Works identically regardless of OS, and survives OS reinstalls.

1. Find the device's MAC address:
   - Linux / Raspberry Pi: `ip link show eth0`
   - macOS: **System Settings** > **Network** > your connection > **Details** > **Hardware**
   - Windows: `ipconfig /all` (look for *Physical Address*)
2. Open your router admin page and go to the DHCP/LAN settings. The feature is called **Address Reservation** (TP-Link), **Static DHCP** (D-Link), **Address Reservation** under LAN Setup (Netgear), or **DHCP Binding** (JioFiber/Airtel).
3. Bind the MAC address to a fixed IP inside your subnet (e.g. `192.168.1.53`).
4. Reboot the device (or renew its lease) and confirm it comes up on the reserved IP.

### Option 2: Static IP on the Device

Pick an address **outside the router's DHCP pool** (check the pool range in the router's DHCP settings) so the router never assigns it to another device.

#### Raspberry Pi OS / Linux

Raspberry Pi OS Bookworm and most modern distros use NetworkManager:

```bash
# List connection names first: nmcli con show
sudo nmcli con mod "Wired connection 1" \
  ipv4.method manual \
  ipv4.addresses 192.168.1.53/24 \
  ipv4.gateway 192.168.1.1 \
  ipv4.dns 1.1.1.1
sudo nmcli con up "Wired connection 1"
```

Older Raspberry Pi OS (Bullseye and earlier) uses `dhcpcd`: append to `/etc/dhcpcd.conf` and reboot:

```
interface eth0
static ip_address=192.168.1.53/24
static routers=192.168.1.1
static domain_name_servers=1.1.1.1
```

#### macOS

1. **System Settings** > **Network** > select your connection (Ethernet or Wi-Fi) > **Details**
2. **TCP/IP** tab > set **Configure IPv4** to **Manually**
3. Enter the IP address (e.g. `192.168.1.53`), subnet mask `255.255.255.0`, and router (gateway) IP
4. **DNS** tab > add `1.1.1.1` as the DNS server
5. **OK**, then verify with `ifconfig` in Terminal

Or from the terminal (list service names with `networksetup -listallnetworkservices`):

```bash
sudo networksetup -setmanual "Ethernet" 192.168.1.53 255.255.255.0 192.168.1.1
sudo networksetup -setdnsservers "Ethernet" 1.1.1.1
```

#### Windows

1. **Settings** > **Network & Internet** > **Ethernet** (or **Wi-Fi** > your network)
2. Under **IP assignment**, click **Edit** > choose **Manual** > turn on **IPv4**
3. Enter the IP address (e.g. `192.168.1.53`), subnet prefix length `24`, gateway (router IP), and preferred DNS `1.1.1.1`
4. **Save**, then verify with `ipconfig` in a terminal

Or via PowerShell (run as Administrator; adapter names via `Get-NetAdapter`):

```powershell
New-NetIPAddress -InterfaceAlias "Ethernet" -IPAddress 192.168.1.53 -PrefixLength 24 -DefaultGateway 192.168.1.1
Set-DnsClientServerAddress -InterfaceAlias "Ethernet" -ServerAddresses 1.1.1.1
```

### Verify

From another device on the network:

```bash
ping 192.168.1.53
dig @192.168.1.53 example.com
```

Both should succeed, and the dashboard should load at `http://192.168.1.53:3000`. Only then move on to pointing the router at it.

## Router DNS Configuration

Point your router's DNS server to your device's static IP address. This makes every device on the network use HydraDNS automatically.

### TP-Link

1. Open `http://192.168.0.1` or `http://tplinkwifi.net`
2. Go to **Advanced** > **Network** > **DHCP Server**
3. Set **Primary DNS** to your device's static IP
4. Leave **Secondary DNS** empty (or set it to the same IP; see [Critical: DNS Configuration](#critical-dns-configuration))
5. Save and reboot router

### D-Link

1. Open `http://192.168.0.1` or `http://dlinkrouter.local`
2. Go to **Setup** > **Internet Setup**
3. Under DNS, select **Manual**
4. Set **Primary DNS** to your device's static IP
5. Leave **Secondary DNS** empty (or set it to the same IP)
6. Save

### Netgear

1. Open `http://192.168.1.1` or `http://routerlogin.net`
2. Go to **Internet** settings
3. Under **Domain Name Server**, select **Use These DNS Servers**
4. Set **Primary DNS** to your device's static IP
5. Leave **Secondary DNS** empty (or set it to the same IP)
6. Apply

### JioFiber

1. Open `http://192.168.29.1`
2. Go to **Network** > **LAN** > **DHCP Server**
3. Set **DNS Server** to your device's static IP
4. Save and reboot

### Airtel Xstream

1. Open `http://192.168.1.1`
2. Go to **LAN** > **DHCP Settings**
3. Set **Primary DNS** to your device's static IP
4. Leave **Secondary DNS** empty (or set it to the same IP)
5. Save

## Critical: DNS Configuration

**Do NOT set a secondary/fallback DNS** (like 8.8.8.8) on the router. By default HydraDNS answers a blocked query with `A 0.0.0.0` (`BLOCK_RESPONSE=zero`, the default; see the `BLOCK_RESPONSE` env var), which most clients treat as "connection refused" and stop there. But some clients and routers just move on to the secondary DNS server on *any* non-standard answer, which resolves the domain normally and bypasses the filter entirely. That risk is worse if you switch `BLOCK_RESPONSE` to `refused`, which some OSes and routers explicitly treat as a signal to fail over.

- **Primary DNS:** Your HydraDNS server IP
- **Secondary DNS:** Leave empty (or set to the same HydraDNS IP)

Also create a policy to block DNS-over-HTTPS providers, which browsers use to bypass system DNS:

```bash
TOKEN="your-token-here"
curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  http://<server-ip>:8080/api/v1/policies \
  -d '{"id":"block-doh","name":"Block DNS-over-HTTPS","action":"BLOCK","domains":["dns.google","dns.google.com","cloudflare-dns.com","mozilla.cloudflare-dns.com","doh.opendns.com","dns.quad9.net","doh.cleanbrowsing.org","dns.adguard.com","dns.nextdns.io","doh.dns.sb"],"priority":200}'
```

## Verify It's Working

From any device on the network:

```bash
# Check DNS resolves through HydraDNS
dig @<pi-ip> example.com

# Check a known blocked domain (blocked by the default "block-ads" policy)
dig @<pi-ip> doubleclick.net
# Should return 0.0.0.0 (BLOCK_RESPONSE=zero, the default) if blocking is active
```

Or open the dashboard at `http://<pi-ip>:3000` and watch the query log update in real time.

## Management

```bash
cd ~/hydradns

# View logs
docker compose logs -f

# Stop
docker compose stop

# Start
docker compose up -d

# Update
git pull
docker compose pull   # fetch the latest published core/ui images, if any
docker compose up -d  # recreates containers; builds from source only if no image was pulled
```

## CLI (Optional)

Build the CLI directly on the Pi:

```bash
cd ~/hydradns/apps/cli
go build -o hydra .
sudo mv hydra /usr/local/bin/

hydra login
hydra status
hydra block malicious-site.com
```

## Troubleshooting

### Port 53 already in use

If `systemd-resolved` wasn't disabled by the installer:

```bash
sudo systemctl disable --now systemd-resolved
sudo rm /etc/resolv.conf
echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
docker compose restart core
```

### Dashboard not accessible from LAN

First check the Pi's firewall:

```bash
sudo ufw allow 53/udp
sudo ufw allow 53/tcp
sudo ufw allow 3000/tcp
sudo ufw allow 8080/tcp
```

Opening the dashboard by the Pi's LAN IP, e.g. `http://192.168.1.53:3000`,
works with no configuration:

- The dashboard's JS derives the control plane's address from the page's own
  URL at runtime (same protocol and hostname, port 8080), so it calls
  `http://192.168.1.53:8080` automatically. `NEXT_PUBLIC_API_URL` is only for
  overriding this, e.g. a reverse proxy or an API on a different host/port.
- The control plane's CORS allowlist (`CORS_ORIGINS`, default
  `http://localhost:3000`) is checked first, but the API also auto-allows a
  request whose `Origin` hostname matches the `Host` header it was reached
  on, as long as that hostname is an IP literal or `localhost`. So
  `Origin: http://192.168.1.53:3000` against `Host: 192.168.1.53:8080` is
  allowed without touching `CORS_ORIGINS`.

If the page loads but never gets past "loading" (or the browser console shows
CORS errors), it's one of these two remaining cases:

1. **You're using a named host, not an IP.** `pi.local`, `hydra.lan`, or a
   reverse-proxy domain don't get the automatic CORS pass; that rule is
   restricted to IP literals and `localhost` specifically to avoid DNS
   rebinding (an attacker page rebinding a name to your Pi's IP would
   otherwise pass the same check). Add the exact origin to `.env` and
   restart the `core` service:

   ```bash
   echo "CORS_ORIGINS=http://localhost:3000,http://pi.local:3000" >> .env
   docker compose up -d core
   ```

2. **The dashboard is served over HTTPS in front of a proxy.** The dashboard
   derives an `https://` API URL to match its own page, but the control
   plane itself has no TLS. Point the dashboard at the proxy's HTTPS
   endpoint for the API too, via `NEXT_PUBLIC_API_URL`, or terminate TLS for
   both dashboard and API behind the same proxy. If your proxy only
   terminates TLS on port 443 (common for a single-port setup), see the
   "runtime API-URL derivation assumes a two-port reverse proxy" entry in
   [docs/limitations.md](limitations.md). Port 8080 is fixed unless you
   rebuild the dashboard image with `NEXT_PUBLIC_API_URL` set at build time.

### Slow first startup

If a tagged release exists, `docker compose up -d` pulls prebuilt `linux/arm64`
images and should be quick (network-bound). If no release exists yet, or the
images aren't public, Compose falls back to building from source, and the
first `docker compose build` on a Pi can take 10-15 minutes. Subsequent
starts use cached images/layers either way and take under 30 seconds.

### SD card wear

For long-term deployments, consider booting from a USB SSD instead of an SD card. SQLite WAL mode reduces write amplification, but query logs still generate writes. Set `BLOCKLIST_UPDATE_INTERVAL=24h` to reduce fetch frequency.
