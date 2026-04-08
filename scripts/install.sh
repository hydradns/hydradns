#!/usr/bin/env bash
#
# HydraDNS Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/hydradns/hydradns/main/scripts/install.sh | bash
#
set -euo pipefail

REPO="https://github.com/hydradns/hydradns.git"
INSTALL_DIR="${HYDRA_DIR:-$HOME/hydradns}"
BRANCH="${HYDRA_BRANCH:-main}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[+]${NC} $1"; }
warn()  { echo -e "${YELLOW}[!]${NC} $1"; }
error() { echo -e "${RED}[x]${NC} $1"; exit 1; }

echo ""
echo "  _   _           _            ____  _   _ ____  "
echo " | | | |_   _  __| |_ __ __ _ |  _ \\| \\ | / ___| "
echo " | |_| | | | |/ _\` | '__/ _\` || | | |  \\| \\___ \\ "
echo " |  _  | |_| | (_| | | | (_| || |_| | |\\  |___) |"
echo " |_| |_|\\__, |\\__,_|_|  \\__,_||____/|_| \\_|____/ "
echo "        |___/                                     "
echo ""
echo "  DNS Security Gateway — Self-Hosted & Private"
echo ""

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    aarch64|arm64) info "Detected ARM64 (Raspberry Pi)" ;;
    x86_64)        info "Detected AMD64" ;;
    *)             warn "Untested architecture: $ARCH" ;;
esac

# Check dependencies
command -v git >/dev/null 2>&1 || error "git is required but not installed"
command -v docker >/dev/null 2>&1 || error "docker is required but not installed"

# Check docker compose (v2 plugin or standalone)
if docker compose version >/dev/null 2>&1; then
    COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE="docker-compose"
else
    error "docker compose is required but not installed"
fi

# Clone or update
if [ -d "$INSTALL_DIR" ]; then
    info "Updating existing installation at $INSTALL_DIR"
    cd "$INSTALL_DIR"
    git pull --recurse-submodules
else
    info "Cloning HydraDNS to $INSTALL_DIR"
    git clone --recursive -b "$BRANCH" "$REPO" "$INSTALL_DIR"
    cd "$INSTALL_DIR"
fi

# Create .env if it doesn't exist
if [ ! -f .env ]; then
    info "Creating .env from example"
    if [ -f .env.example ]; then
        cp .env.example .env
    fi
fi

# Check if systemd-resolved is blocking port 53
if systemctl is-active --quiet systemd-resolved 2>/dev/null; then
    warn "systemd-resolved is running and may block port 53"
    warn "Disabling it... (DNS will be handled by HydraDNS)"
    sudo systemctl disable --now systemd-resolved 2>/dev/null || true
    # Point resolv.conf to a real resolver temporarily
    if [ -L /etc/resolv.conf ]; then
        sudo rm /etc/resolv.conf
        echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf >/dev/null
    fi
fi

# Build and start
info "Building containers (this may take a few minutes on first run)..."
$COMPOSE build

info "Starting HydraDNS..."
$COMPOSE up -d

# Wait for health
info "Waiting for core to be healthy..."
TRIES=0
MAX_TRIES=30
while [ $TRIES -lt $MAX_TRIES ]; do
    if $COMPOSE ps core 2>/dev/null | grep -q "healthy"; then
        break
    fi
    TRIES=$((TRIES + 1))
    sleep 2
done

if [ $TRIES -eq $MAX_TRIES ]; then
    warn "Core service hasn't become healthy yet. Check: $COMPOSE logs core"
else
    info "Core is healthy!"
fi

# Get the machine IP for router config
LOCAL_IP=$(hostname -I 2>/dev/null | awk '{print $1}' || echo "your-server-ip")

echo ""
info "HydraDNS is running!"
echo ""
echo "  Dashboard:  http://${LOCAL_IP}:3000"
echo "  API:        http://${LOCAL_IP}:8080"
echo "  DNS:        ${LOCAL_IP}:53"
echo ""
echo "  To use as your network DNS, point your router's"
echo "  DNS server setting to: ${LOCAL_IP}"
echo ""
echo "  Test it:  dig @${LOCAL_IP} example.com"
echo ""
echo "  Commands:"
echo "    cd $INSTALL_DIR"
echo "    $COMPOSE logs -f        # View logs"
echo "    $COMPOSE stop           # Stop"
echo "    $COMPOSE up -d          # Start"
echo ""
