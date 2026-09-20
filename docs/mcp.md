# MCP Server

`hydra mcp` starts a Model Context Protocol server that exposes HydraDNS management tools to
AI assistants and agents: query DNS engine status, read logs and metrics, and create or remove
blocking policies. It talks to the same control-plane API (`:8080`) as the CLI and the
dashboard, and does not read or write DNS query data directly.

Two transports:

- **stdio (default).** JSON-RPC 2.0 over stdin/stdout. This is what MCP clients (Claude
  Desktop, Claude Code, Cursor, VS Code, Gemini CLI) expect, and what the container image
  described below runs.
- **HTTP (`--http`).** Same tool set over HTTP, for driving a fleet of HydraDNS instances
  remotely. Every request needs a bearer token (`--http-token` or `HYDRA_MCP_TOKEN`). Carries
  management traffic only, never DNS query data.

The `initialize` response's `serverInfo.version` reports the CLI's own version (the same
string `hydra version` prints), not a hardcoded value. Useful for confirming which build an
agent is actually talking to.

## Configuration

The MCP server reuses the CLI's own connection settings; there is nothing MCP-specific to
configure beyond the role.

| Variable | Purpose | Precedence |
|---|---|---|
| `HYDRA_API_URL` | Base URL of the HydraDNS control-plane API | `--api` flag, then `HYDRA_API_URL`, then default `http://localhost:8080` |
| `HYDRA_TOKEN` | Bearer token for the control-plane API | `--token` flag, then `HYDRA_TOKEN`, then `~/.hydra/token` (written by `hydra login` / the setup wizard), then empty |
| `MCP_ROLE` | Permission scope for the current MCP session (`admin`, `operator`, `reporter`) | Read once at server start; unset or empty means `admin` |
| `HYDRA_MCP_TOKEN` | Bearer token required of *incoming* HTTP clients, `--http` transport only | `--http-token` flag, then `HYDRA_MCP_TOKEN`; unrelated to `HYDRA_TOKEN` |

`HYDRA_TOKEN` and `HYDRA_MCP_TOKEN` are easy to confuse: `HYDRA_TOKEN` is the credential the
MCP server uses to call *out* to the HydraDNS API (needed in every mode). `HYDRA_MCP_TOKEN` is
a credential HTTP *clients* present to the MCP server itself, and only exists for `--http`.

Source: `apps/cli/cmd/root.go`, `apps/cli/cmd/mcp.go`, `apps/cli/mcp/server.go`,
`apps/cli/mcp/roles.go`.

## Roles

Set `MCP_ROLE` before starting the server to restrict what an agent can do:

- **`admin`** (default when `MCP_ROLE` is unset): every tool.
- **`operator`**: every mutating tool except `toggle_engine`.
- **`reporter`**: read-only tools only (see below). Any unrecognized `MCP_ROLE` value also
  safe-defaults to `reporter`, not to `admin`.

Whether a tool is read-only is an explicit flag on each tool's registration (`apps/cli/mcp/server.go`, `toolRegistry()`), not a guess from its name. `explain_anomaly` and
`compare_to_last_month` only read and analyze data, so they are read-only and a `reporter`
token can call them. An unregistered tool name is treated as mutating.

A denied call returns a structured JSON-RPC error (code `-32003`) rather than executing:

```json
{"error": {"code": -32003, "message": "permission denied: role \"reporter\" is not permitted to call tool \"block_domain\"", "data": {"role": "reporter", "tool": "block_domain", "reason": "role_not_permitted"}}}
```

## Tools

14 tools, from `apps/cli/mcp/server.go`. "Confirm?" reflects the `destructiveHint` /
`confirmationRequired` annotations every MCP client receives via `tools/list`; a compliant
client should ask before running those.

| Tool | Description | Read-only / Confirm? |
|---|---|---|
| `get_status` | Get DNS engine status and query statistics | read-only |
| `list_policies` | List all DNS policies | read-only |
| `list_blocklists` | List blocklist sources and domain counts | read-only |
| `get_query_logs` | Get recent DNS query logs | read-only |
| `get_metrics` | Get DNS query performance metrics including latency percentiles | read-only |
| `get_weekly_summary` | Natural-language rollup of DNS security activity built from live stats, metrics, and query logs | read-only |
| `explain_anomaly` | Inspect current DNS activity and describe anything unusual (error rate, latency, block-rate spikes, a client dominating traffic); optional baseline to compare against | read-only |
| `compare_to_last_month` | Compare current traffic and block rate against a supplied baseline window and describe the change in plain language | read-only |
| `toggle_engine` | Enable or disable the DNS engine | confirm, blocked for `operator` too, `admin` only |
| `block_domain` | Block a domain by creating a block policy | confirm |
| `create_policy` | Create a policy (BLOCK/ALLOW/REDIRECT) for multiple domains in one rule; prefer this over `block_domain` for more than one domain | confirm |
| `unblock_domain` | Remove a block policy by its ID | confirm |
| `bulk_unblock` | Remove multiple policies at once by ID; reports which succeeded/failed | confirm |
| `delete_policy` | Delete a single policy (BLOCK/ALLOW/REDIRECT) by its ID | confirm |

## Security notes

- **The token is API access, full stop.** Whatever role you don't enforce with `MCP_ROLE`, the
  underlying `HYDRA_TOKEN` can still do directly against the API (the role gate lives in the
  MCP layer, not the API). Give an agent a `reporter`- or `operator`-scoped mindset by handing
  it a token you're comfortable with regardless of `MCP_ROLE`, or mint separate tokens per
  trust level if/when the API supports scoped tokens.
- **No TLS on the control-plane API today** (see `docs/limitations.md`). `HYDRA_API_URL` and
  the bearer token cross the network in clear text. Keep the API on a trusted LAN, or put a
  TLS-terminating reverse proxy in front of it and point `HYDRA_API_URL` at that instead.
- Prefer `MCP_ROLE=reporter` or `MCP_ROLE=operator` over the `admin` default for
  agent-facing setups; `admin` is the default only to avoid changing behavior for existing
  installs that predate roles.
- stdio mode: nothing but JSON-RPC lines belongs on stdout. `hydra mcp` writes only protocol
  responses to stdout and startup/error messages to stderr (verified below); don't add
  `fmt.Println`/`log` output to `apps/cli/mcp/*.go` without checking this stays true.

## Running it

### As a local binary

Build or download `hydra`, then either let a client launch it directly (see Client
configuration below) or run it by hand:

```bash
HYDRA_API_URL=http://localhost:8080 HYDRA_TOKEN=<your-token> hydra mcp
```

`hydra mcp config --client claude|gemini|cursor` prints ready-to-paste JSON for this mode,
using whichever `--api`/`--token`/env values are active when you run it.

### As the container image

`ghcr.io/hydradns/hydra-cli` packages the same `hydra` binary with `mcp` as its default
command:

```bash
docker run -i --rm \
  -e HYDRA_API_URL \
  -e HYDRA_TOKEN \
  ghcr.io/hydradns/hydra-cli
```

`-i` (interactive, keep stdin open) is required: this is a stdio server, not a background
service. Do not add `-d`; a detached container has no stdin/stdout for the client to talk to.

The container cannot reach `http://localhost:8080` on its own. Inside a container,
`localhost` means the container itself, not your host machine. Point it at the real
control-plane address instead, verified against Docker's docs:

1. **LAN IP: works everywhere, no special flags.** The `core` service already publishes
   `8080:8080` to all interfaces on the host (`docker-compose.yml`), so from any other machine
   (or from inside a container without special networking) `http://<host-LAN-IP>:8080` works
   unconditionally. Simplest option if you already know the host's LAN IP.

2. **Linux, plain Docker Engine, controlplane on the same machine:**
   ```bash
   docker run -i --rm --network host \
     -e HYDRA_API_URL=http://localhost:8080 \
     -e HYDRA_TOKEN \
     ghcr.io/hydradns/hydra-cli
   ```
   `--network host` puts the container directly on the host's network namespace, so
   `localhost` inside the container really is the host. Linux-only: `--network host` is not
   available the same way on Docker Desktop for Mac/Windows
   (<https://docs.docker.com/reference/cli/docker/container/run/#network>, "host" driver).

3. **Docker Desktop (Mac/Windows), or Linux without `--network host`:**
   ```bash
   docker run -i --rm \
     -e HYDRA_API_URL=http://host.docker.internal:8080 \
     -e HYDRA_TOKEN \
     ghcr.io/hydradns/hydra-cli
   ```
   `host.docker.internal` is resolved automatically by Docker Desktop. Plain Docker Engine on
   Linux needs one extra flag to get the same resolution:
   ```bash
   docker run -i --rm --add-host=host.docker.internal:host-gateway \
     -e HYDRA_API_URL=http://host.docker.internal:8080 \
     -e HYDRA_TOKEN \
     ghcr.io/hydradns/hydra-cli
   ```
   Source: <https://docs.docker.com/reference/cli/docker/container/run/> ("The `--add-host`
   flag supports a special `host-gateway` value... It's conventional to use
   `host.docker.internal` as the hostname referring to `host-gateway`.").

4. **Running via this project's `docker-compose.yml`:** the `core` service is already reachable
   from any container on the same Compose network by its service name. Find the actual network
   name (Compose prefixes the `hydra-net` name declared in `docker-compose.yml` with the
   project name, e.g. `hydradns_hydra-net` for a plain `git clone .../hydradns.git && cd
   hydradns` checkout; confirm with `docker network ls`, since the project name follows
   whatever directory the compose file lives in, or `COMPOSE_PROJECT_NAME` if set; see
   <https://docs.docker.com/compose/how-tos/project-name/>), then:
   ```bash
   docker run -i --rm --network <project>_hydra-net \
     -e HYDRA_API_URL=http://core:8080 \
     -e HYDRA_TOKEN \
     ghcr.io/hydradns/hydra-cli
   ```
   `core` here is the Compose service name, resolved by Compose's embedded DNS on that
   network; this works regardless of what the network itself ends up named.

## Client configuration

Verified against each project's current docs (fetched 2026-09-20). Every example below assumes
`HYDRA_API_URL` and `HYDRA_TOKEN` are set in your shell before generating/pasting the config,
or edits them in by hand.

### Claude Desktop

Edit `claude_desktop_config.json`:
- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`

Local binary:
```json
{
  "mcpServers": {
    "hydradns": {
      "command": "/path/to/hydra",
      "args": ["mcp"],
      "env": {
        "HYDRA_API_URL": "http://localhost:8080",
        "HYDRA_TOKEN": "..."
      }
    }
  }
}
```

Container (adjust the API URL per the networking section above):
```json
{
  "mcpServers": {
    "hydradns": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "-e", "HYDRA_API_URL", "-e", "HYDRA_TOKEN", "ghcr.io/hydradns/hydra-cli"],
      "env": {
        "HYDRA_API_URL": "http://host.docker.internal:8080",
        "HYDRA_TOKEN": "..."
      }
    }
  }
}
```

Reference: <https://modelcontextprotocol.io/quickstart/user>.

### Claude Code

CLI (recommended; `--` separates Claude Code's own flags from the server command):
```bash
claude mcp add --transport stdio hydradns --scope project \
  -e HYDRA_API_URL=http://localhost:8080 -e HYDRA_TOKEN=... \
  -- /path/to/hydra mcp
```

Or a project-scoped `.mcp.json`:
```json
{
  "mcpServers": {
    "hydradns": {
      "type": "stdio",
      "command": "/path/to/hydra",
      "args": ["mcp"],
      "env": {
        "HYDRA_API_URL": "http://localhost:8080",
        "HYDRA_TOKEN": "..."
      }
    }
  }
}
```

Container, via the CLI:
```bash
claude mcp add --transport stdio hydradns --scope project \
  -e HYDRA_API_URL=http://host.docker.internal:8080 -e HYDRA_TOKEN=... \
  -- docker run -i --rm -e HYDRA_API_URL -e HYDRA_TOKEN ghcr.io/hydradns/hydra-cli
```

Reference: <https://code.claude.com/docs/en/mcp>.

### Cursor

`.cursor/mcp.json` (project) or `~/.cursor/mcp.json` (global):
```json
{
  "mcpServers": {
    "hydradns": {
      "command": "/path/to/hydra",
      "args": ["mcp"],
      "env": {
        "HYDRA_API_URL": "http://localhost:8080",
        "HYDRA_TOKEN": "..."
      }
    }
  }
}
```

Container: replace `command`/`args` the same way as the Claude Desktop container example
above.

Reference: <https://cursor.com/docs/context/mcp>.

### VS Code

`.vscode/mcp.json` (workspace) or the user MCP config. Note the top-level key is `servers`,
not `mcpServers`:
```json
{
  "servers": {
    "hydradns": {
      "type": "stdio",
      "command": "/path/to/hydra",
      "args": ["mcp"],
      "env": {
        "HYDRA_API_URL": "http://localhost:8080",
        "HYDRA_TOKEN": "..."
      }
    }
  }
}
```

Container:
```json
{
  "servers": {
    "hydradns": {
      "type": "stdio",
      "command": "docker",
      "args": ["run", "-i", "--rm", "-e", "HYDRA_API_URL", "-e", "HYDRA_TOKEN", "ghcr.io/hydradns/hydra-cli"],
      "env": {
        "HYDRA_API_URL": "http://host.docker.internal:8080",
        "HYDRA_TOKEN": "..."
      }
    }
  }
}
```

Reference: <https://code.visualstudio.com/docs/copilot/customization/mcp-servers>.

### Gemini CLI

`~/.gemini/settings.json` (global) or `.gemini/settings.json` (project):
```json
{
  "mcpServers": {
    "hydradns": {
      "command": "/path/to/hydra",
      "args": ["mcp"],
      "env": {
        "HYDRA_API_URL": "http://localhost:8080",
        "HYDRA_TOKEN": "..."
      }
    }
  }
}
```

Container: replace `command`/`args` the same way as the Claude Desktop container example
above.

Reference: <https://github.com/google-gemini/gemini-cli/blob/main/docs/tools/mcp-server.md>.

### Not covered here

No other client configs are included. If a client isn't listed above, check its own MCP docs
for the exact JSON shape before assuming it matches one of the examples above. The `servers`
vs `mcpServers` top-level key difference between VS Code and everything else is a real
gotcha.
