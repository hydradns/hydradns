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
3. Build Docker images for arm64
4. Start all services
5. Print your Pi's IP address and dashboard URL

## First-Time Setup

1. Open `http://<pi-ip>:3000` in a browser on any device on your network
2. Complete the setup wizard:
   - Set an admin password
   - Choose upstream DNS providers
   - Select blocklist sources
3. You'll be redirected to the dashboard

## Give the Device a Static IP

Your router will forward every DNS query on the network to this device's IP address. If the device gets its address from DHCP, the router can hand it a different IP after a reboot or lease renewal — and DNS for the entire network silently breaks. Pin the IP **before** configuring the router.

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

Older Raspberry Pi OS (Bullseye and earlier) uses `dhcpcd` — append to `/etc/dhcpcd.conf` and reboot:

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
4. Leave **Secondary DNS** empty (or set it to the same IP — see [Critical: DNS Configuration](#critical-dns-configuration))
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

**Do NOT set a secondary/fallback DNS** (like 8.8.8.8) on the router. When HydraDNS blocks a domain by returning REFUSED, the operating system will try the secondary DNS server, which resolves the domain normally — bypassing the filter entirely.

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

# Check a known blocked domain
dig @<pi-ip> ads.google.com
# Should return REFUSED if blocklists are active
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
git pull --recurse-submodules
docker compose build
docker compose up -d
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

Check the Pi's firewall:

```bash
sudo ufw allow 53/udp
sudo ufw allow 53/tcp
sudo ufw allow 3000/tcp
sudo ufw allow 8080/tcp
```

### Slow first startup

The first `docker compose build` on a Pi can take 10-15 minutes. Subsequent starts use cached images and take under 30 seconds.

### SD card wear

For long-term deployments, consider booting from a USB SSD instead of an SD card. SQLite WAL mode reduces write amplification, but query logs still generate writes. Set `BLOCKLIST_UPDATE_INTERVAL=24h` to reduce fetch frequency.
