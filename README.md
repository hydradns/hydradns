# HydraDNS

**DNS-layer security gateway for your entire network.**

Block ads, malware, and trackers at the DNS level — before they ever reach your devices. Self-hosted, private, and fast.

[![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go)](https://go.dev)
[![Next.js](https://img.shields.io/badge/Next.js-16-000?logo=next.js)](https://nextjs.org)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker)](https://docs.docker.com/compose/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

---

## Quick Start

```bash
# Clone with submodules
git clone --recursive https://github.com/hydradns/hydradns.git
cd hydradns

# Start everything
docker compose up -d

# Verify DNS is working
dig @localhost example.com

# Check the dashboard
open http://localhost:3000
```

That's it. DNS filtering is active. Point your router's DNS to this machine's IP and your whole network is protected.

---

## Architecture

```
                    +-----------+
                    |  Browser  |
                    +-----+-----+
                          |
                    +-----v-----+
                    |  Dashboard |  :3000  (Next.js)
                    +-----+-----+
                          |
                    +-----v-----+
         +--------->  Control   |  :8080  (Go + Gin REST API)
         |          |   Plane   |
         |          +-----+-----+
         |                |  gRPC :50051
         |          +-----v-----+
  CLI/MCP|          |   Data    |  :53    (DNS UDP/TCP)
  hydra  +--------->   Plane    |
                    +-----+-----+
                          |
               +----------+----------+
               |          |          |
          +----v---+ +----v---+ +----v---+
          |Blocklist| | Policy | |Upstream|
          | Engine  | | Engine | |Resolvers|
          +--------+ +--------+ +--------+
```

| Service | Directory | Tech | Port |
|:--------|:----------|:-----|:-----|
| Core (Control + Data Plane) | `apps/core` | Go 1.24, Gin, gRPC, GORM/SQLite | 8080, 53 |
| Dashboard | `apps/ui` | Next.js 16, React 19, TypeScript, Tailwind | 3000 |
| Landing Page | `apps/landing` | Vite, React 18, TypeScript | 3001 |
| Scanner | `apps/scanner` | Go, network detection | — |
| CLI + MCP | `apps/cli` | Go, Cobra, JSON-RPC 2.0 | — |

### DNS Query Pipeline

Every DNS query goes through a 3-step pipeline with early exit:

1. **Blocklist check** — if domain is on any blocklist, respond `REFUSED` immediately
2. **Policy evaluation** — Bloom filter for O(1) negative lookup, then exact match. Highest priority wins
3. **Upstream forward** — pool-per-resolver with failover (5s timeout, 2 retries)

---

## Dashboard

The web dashboard at `localhost:3000` lets you:

- View real-time query statistics (total, blocked, allowed, block rate)
- Toggle the DNS engine on/off
- Manage blocklist sources (add/remove/view domain counts)
- Create and delete DNS policies (block, allow, redirect)
- Search and filter query logs

<!-- TODO: Add screenshot -->

---

## CLI

The `hydra` CLI wraps the control plane API for terminal-based management.

```bash
# Build the CLI
cd apps/cli && go build -o hydra .

# Check status
hydra status

# Block a domain
hydra block ads.example.com

# View query logs
hydra logs

# Manage blocklists
hydra blocklists
hydra blocklists add --id steven-black --name "StevenBlack" \
  --url "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts"

# Manage policies
hydra policies
hydra policies delete my-policy-id

# Engine control
hydra engine enable
hydra engine disable

# View metrics
hydra metrics
```

Set `HYDRA_API_URL` to point at a remote instance (default: `http://localhost:8080`).

---

## MCP Server (AI Integration)

HydraDNS includes a built-in [Model Context Protocol](https://modelcontextprotocol.io/) server, letting AI assistants manage your DNS firewall conversationally.

```bash
# Start MCP server (JSON-RPC 2.0 over stdio)
hydra mcp
```

### Claude Code Setup

Add to your Claude Code MCP config:

```json
{
  "mcpServers": {
    "hydradns": {
      "command": "/path/to/hydra",
      "args": ["mcp"],
      "env": {
        "HYDRA_API_URL": "http://localhost:8080"
      }
    }
  }
}
```

### Available MCP Tools

| Tool | Description |
|:-----|:------------|
| `get_status` | Engine status and query statistics |
| `toggle_engine` | Enable or disable DNS engine |
| `block_domain` | Block a domain (creates a policy) |
| `unblock_domain` | Remove a block policy |
| `list_policies` | List all DNS policies |
| `list_blocklists` | List blocklist sources |
| `get_query_logs` | Recent DNS query logs |
| `get_metrics` | Latency percentiles and performance grade |

**Example conversation:** "Block all social media domains" — Claude calls `block_domain` for each domain.

---

## Development

### Prerequisites

- Go 1.24+
- Node.js 20+
- Docker & Docker Compose

### Working on a Service

Each service is a separate Git submodule. Work inside the service directory:

```bash
cd apps/core
make build        # Compile controlplane & dataplane
make test         # Run tests with coverage
make fmt          # Format code
make vet          # Vet code
make lint         # golangci-lint

cd apps/ui
npm run dev       # Dev server on :3000
npm run build     # Production build

cd apps/cli
go build -o hydra .  # Build CLI binary
```

### Full Stack Commands (from root)

```bash
make setup        # Initialize submodules
make start        # docker compose up -d
make stop         # docker compose down
make update       # Pull latest submodule changes
make logs         # Tail all logs
make build-core   # Rebuild core service
make restart-core # Rebuild + restart core
```

---

## Project Structure

```
hydradns/
├── apps/
│   ├── core/           # Go DNS engine + API
│   │   ├── cmd/        #   controlplane + dataplane binaries
│   │   ├── internal/   #   blocklist, dnsengine, policy, storage
│   │   ├── configs/    #   config.yaml + policies.json
│   │   └── proto/      #   gRPC protobuf definitions
│   ├── ui/             # Next.js dashboard
│   ├── landing/        # Vite marketing site
│   ├── scanner/        # Network detection worker
│   └── cli/            # CLI + MCP server
│       ├── cmd/        #   Cobra commands
│       ├── api/        #   HTTP client for control plane
│       └── mcp/        #   MCP JSON-RPC server
├── docker-compose.yml  # Full stack orchestration
├── Makefile            # Convenience commands
└── scripts/            # Setup scripts
```

---

## Deployment

### Docker Compose (recommended)

```bash
docker compose up -d
```

Core runs as a combined container (controlplane + dataplane) with:
- SQLite database persisted in a Docker volume
- Health check on `/health` endpoint
- Automatic blocklist fetching on startup

### Raspberry Pi / VPS

```bash
curl -fsSL https://raw.githubusercontent.com/hydradns/hydradns/main/scripts/install.sh | bash
```

Then point your router's DNS server to `<pi-ip>`.

---

## Configuration

| Env Variable | Default | Description |
|:-------------|:--------|:------------|
| `PHANTOM_CONFIG` | `configs/config.yaml` | Path to config file |
| `PHANTOM_DB` | `phantomdns.db` | SQLite database path |
| `PHANTOM_POLICIES` | `configs/policies.json` | Policy file path |
| `CORS_ORIGINS` | `http://localhost:3000` | Allowed CORS origins |
| `HYDRA_API_URL` | `http://localhost:8080` | CLI/MCP API target |

---

## License

[MIT](LICENSE)
