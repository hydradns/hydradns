# Raspberry Pi Deployment Guide

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

## Router DNS Configuration

Point your router's DNS server to your Pi's IP address. This makes every device on the network use HydraDNS automatically.

### TP-Link

1. Open `http://192.168.0.1` or `http://tplinkwifi.net`
2. Go to **Advanced** > **Network** > **DHCP Server**
3. Set **Primary DNS** to your Pi's IP address
4. Set **Secondary DNS** to `8.8.8.8` (fallback if Pi is down)
5. Save and reboot router

### D-Link

1. Open `http://192.168.0.1` or `http://dlinkrouter.local`
2. Go to **Setup** > **Internet Setup**
3. Under DNS, select **Manual**
4. Set **Primary DNS** to your Pi's IP
5. Set **Secondary DNS** to `8.8.8.8`
6. Save

### Netgear

1. Open `http://192.168.1.1` or `http://routerlogin.net`
2. Go to **Internet** settings
3. Under **Domain Name Server**, select **Use These DNS Servers**
4. Set **Primary DNS** to your Pi's IP
5. Set **Secondary DNS** to `8.8.8.8`
6. Apply

### JioFiber

1. Open `http://192.168.29.1`
2. Go to **Network** > **LAN** > **DHCP Server**
3. Set **DNS Server** to your Pi's IP
4. Save and reboot

### Airtel Xstream

1. Open `http://192.168.1.1`
2. Go to **LAN** > **DHCP Settings**
3. Set **Primary DNS** to your Pi's IP
4. Set **Secondary DNS** to `8.8.8.8`
5. Save

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
