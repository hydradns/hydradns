# Phase 4: CLI + MCP Plan

**Date:** 2026-04-07
**Status:** Complete

## Architecture

Single Go binary (`hydra`) with two modes:
1. **CLI mode** — interactive commands wrapping the REST API
2. **MCP mode** — `hydra mcp` starts an MCP server (JSON-RPC 2.0 over stdio) exposing CLI operations as tools

## CLI Commands

| Command | Maps to | Description |
|---|---|---|
| `hydra status` | GET /dashboard/summary + GET /dns/engine | Overview: query counts, block rate, engine status |
| `hydra engine` | GET /dns/engine | Engine status |
| `hydra engine enable` | POST /dns/engine {enabled:true} | Enable DNS engine |
| `hydra engine disable` | POST /dns/engine {enabled:false} | Disable DNS engine |
| `hydra metrics` | GET /dns/metrics | Query metrics and latency |
| `hydra block <domain>` | POST /policies | Quick-block a domain (auto-generates policy) |
| `hydra unblock <id>` | DELETE /policies/:id | Remove a block policy |
| `hydra policies` | GET /policies | List all policies |
| `hydra policies delete <id>` | DELETE /policies/:id | Delete a policy |
| `hydra blocklists` | GET /blocklists | List blocklist sources |
| `hydra blocklists add` | POST /blocklists | Add a blocklist source (flags for url, format, etc.) |
| `hydra blocklists delete <id>` | DELETE /blocklists/:id | Delete a source |
| `hydra logs` | GET /analytics/audits | Recent query logs |
| `hydra mcp` | — | Start MCP server on stdio |

## MCP Tools (exposed via `hydra mcp`)

| Tool | Description | Parameters |
|---|---|---|
| `get_status` | Get DNS engine status and query stats | none |
| `toggle_engine` | Enable or disable DNS engine | enabled: bool |
| `block_domain` | Block a domain | domain: string |
| `unblock_domain` | Remove a block policy | policy_id: string |
| `list_policies` | List all DNS policies | none |
| `list_blocklists` | List blocklist sources | none |
| `get_query_logs` | Get recent DNS query logs | none |
| `get_metrics` | Get DNS query performance metrics | none |

## Tech Stack
- Go + cobra for CLI
- Custom MCP JSON-RPC handler for MCP server (no external SDK needed)
- Shared API client package

## File Structure
```
apps/cli/
├── go.mod
├── main.go
├── cmd/          — cobra commands
│   ├── root.go
│   ├── status.go
│   ├── engine.go
│   ├── block.go
│   ├── policies.go
│   ├── blocklists.go
│   ├── logs.go
│   └── mcp.go
├── api/          — HTTP client for control plane
│   └── client.go
└── mcp/          — MCP server implementation
    └── server.go
```
