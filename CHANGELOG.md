# Changelog

All notable changes to HydraDNS are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project aims
to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html) from v0.1.0.

## [Unreleased]

### Added
- **Public demo mode** (`HYDRA_DEMO_MODE=true`): a `DemoGuard` middleware rejects every
  mutating request except login before auth even runs, a fixed-password read-only demo
  user and synthetic policies/blocklists/query-logs are seeded on startup (refreshed every
  30 minutes so a long-running demo doesn't go stale), and client IPs are masked in every
  response that carries one. Refuses to start against a database that already has real
  users. Ships with a standalone `demo/docker-compose.demo.yml` stack and `demo/README.md`
  covering the reverse-proxy/TLS layout and abuse considerations. Not for a normal
  self-hosted install — only for hosting a public, clickable demo.
- `hydra setup`: a first-boot CLI command that creates the admin account through the setup
  endpoint and stores the token, the same way `hydra login` does (no-echo password prompt
  or `--password-stdin`; there is deliberately no `--password` flag).
- New API routes backing dashboard features that previously had no server-side
  implementation: `GET /analytics/logs` (server-side paging and filtering for the Logs
  page, page size capped at 200), `GET /analytics/bypass` (DoH/DoT bootstrap-detection
  attempts), `PUT /policies/:id`, and `PATCH /blocklists/:id` — the latter two carry the
  same validation, role checks, and audit events as create/delete.
- A legacy-database upgrade path: on startup, if the new default database file
  (`hydradns.db`) doesn't exist but a pre-rename `phantomdns.db` does, the control plane
  opens that file instead of starting with an empty database. One-time upgrade fallback,
  not the default path for new installs.
- The DoH-bypass-attempts panel is now hidden on the dashboard by default; set
  `NEXT_PUBLIC_SHOW_BYPASS_PANEL=true` at build time to show it (e.g. for technical/internal
  deployments).
- `apps/cli/Dockerfile` and a `hydra-cli` entry in `release.yml`'s multi-arch GHCR image
  matrix, alongside the existing `core` and `ui` images. Publishes
  `ghcr.io/hydradns/hydra-cli`, a small non-root image whose default command runs
  `hydra mcp` (stdio JSON-RPC), so the CLI's MCP server can be listed in the official MCP
  registry (`registryType: "oci"`) and run as `docker run -i --rm -e HYDRA_API_URL -e
  HYDRA_TOKEN ghcr.io/hydradns/hydra-cli` without a local Go toolchain. `ci.yml`'s Docker job
  gained a matching build-only check on PRs. See `docs/mcp.md` for the full tool list, roles,
  and exact container networking invocations per platform (Linux, Docker Desktop, this
  project's own `docker-compose.yml`).
- `docs/mcp.md`: public documentation for the `hydra mcp` server — the 14 tools, `MCP_ROLE`
  permission scopes, security notes, and verified client configuration for Claude Desktop,
  Claude Code, Cursor, VS Code, and the Gemini CLI, for both the local binary and the new
  container image.
- Blocklist parsers for the `domains` (one host per line) and `adblock`
  (EasyList `||domain^`) formats, alongside the existing `hosts` format.
- `docs/releasing.md`: a release runbook for cutting tags (pre-flight checks,
  what `release.yml` publishes, GHCR visibility/verification, rollback).
- Automatic same-host CORS rule on the control plane API: a request is
  allowed when its `Origin` hostname matches the `Host` header it was
  reached on (ports ignored) and that hostname is an IP literal or
  `localhost`. This is what lets a dashboard opened at
  `http://192.168.1.53:3000` call the API at `http://192.168.1.53:8080`
  with no `CORS_ORIGINS` edit. Restricted to IP literals and `localhost` on
  purpose: a DNS-rebinding attack would make the `Origin` and `Host`
  hostnames agree on an attacker-controlled name, so named hosts are
  excluded and still need an explicit `CORS_ORIGINS` entry. Disable with
  `CORS_ALLOW_SAME_HOST=false`. See
  `apps/core/cmd/controlplane/middlewares/cors.go`.

### Changed
- Consolidated the five service repositories into a single monorepo under
  `apps/`. A plain `git clone` now checks out the whole stack.
- Renamed the project internals from PhantomDNS to HydraDNS (Go module paths,
  protobuf package, environment variables, and the on-disk database).
- `docker-compose.yml` now declares `image: ghcr.io/hydradns/<service>:${HYDRA_VERSION:-latest}`
  alongside the existing `build:` section for `core` and `ui`. Once a
  release is published, `docker compose up -d` pulls it instead of
  compiling from source; before the first release (or offline), Compose
  falls back to building locally, unchanged from before. Pin a version via
  `HYDRA_VERSION` in `.env`.
- The dashboard now derives the control-plane API's address from the page's
  own URL at runtime (same protocol and hostname, port 8080) instead of
  relying only on the value baked in at build time, so the published
  `ghcr.io/hydradns/ui` image works over a LAN IP without a rebuild.
  `NEXT_PUBLIC_API_URL` is now needed only to override this — a reverse
  proxy, or an API on a different host/port than the dashboard.

### Fixed
- `docker-compose.yml` shipped `CORS_ORIGINS=*` for the `core` service,
  which — combined with the control plane's `AllowCredentials: true` —
  disabled CORS protection on the API entirely. Default is now
  `CORS_ORIGINS=${CORS_ORIGINS:-http://localhost:3000}` plus the automatic
  same-host rule above, matching `.env.example` and the documented default.
  An explicit `CORS_ORIGINS` entry is only needed for a named host, not for
  ordinary LAN-by-IP access.
- `demo/install.tape` typed a comment claiming a blocked `dig` query returns
  `REFUSED`; the shipped default (`BLOCK_RESPONSE=zero`) actually returns
  `A 0.0.0.0`. The tape also assumed a prebuilt `./apps/cli/hydra` binary and
  an already-authenticated CLI, neither of which exist on a fresh clone
  before the first tagged release. Rewritten to build the CLI and mint a
  setup token first, matching real first-boot behavior.
- `docs/pi-deployment.md` repeated the same `REFUSED` claim in its DNS
  verification and secondary-DNS-fallback sections. Its "Dashboard not
  accessible from LAN" section originally documented `CORS_ORIGINS` and
  `NEXT_PUBLIC_API_URL` as manual steps required for LAN-by-IP access; both
  are now automatic (see the same-host CORS rule and the dashboard's
  runtime API-URL derivation above), so that section now covers only the
  two cases that still need a manual step: a named host, and a dashboard
  served over HTTPS.
- The in-memory blocklist used to be built from every stored entry with no check on the
  source's enabled flag, so disabling a blocklist did not stop it from blocking. It's now
  built from enabled sources only, and the dataplane polls a cheap signature of the
  blocklist tables (`BLOCKLIST_POLL_INTERVAL`, default 5s; `0` disables) and rebuilds the
  in-memory set when it changes, so add/toggle/delete/finished-download no longer wait for
  the 6-hour `BLOCKLIST_UPDATE_INTERVAL` refresh cycle — the same ~5s propagation policy
  edits already had.
- Query-log rows were stored with the client's ephemeral port attached (`ip:port`) instead
  of the bare IP; the client-IP filter and anonymization both now operate on the address
  alone, and legacy rows with a port still match the filter precisely.
- CLI release binaries built without the version stamped in, so every release reported the
  hardcoded fallback `1.0.0` regardless of the actual tag — `hydra update` could never see a
  release as newer than what was already installed. `release.yml` now passes the tag version
  via `-X` at build time.
- The default `apps/core/configs/policies.json` seed policy blocked `apple.com`, breaking
  iCloud, the App Store, and iMessage for a first-time user on Apple devices. Removed; the
  seed now ships only `block-ads` and `block-malware` example policies, and a test asserts
  the shipped policy file never blocks a critical platform domain again.

### Security
- Per-IP rate limiting on `POST /auth/login` and `POST /auth/setup` (fixed window, 10
  attempts / 5 minutes by default, shared across both endpoints), plus `TRUSTED_PROXIES`
  (comma-separated CIDRs/IPs) so the control plane only honors `X-Forwarded-For` from a
  reverse proxy you explicitly trust — otherwise `c.ClientIP()` always resolves to the real
  socket address. This also makes the audit log's recorded client IP trustworthy; previously
  Gin trusted every proxy by default, letting any caller spoof its own client IP.
- Opt-in client-IP pseudonymisation: `HYDRA_ANONYMIZE_CLIENT_IPS` (default `false`) hashes
  a client's IP with HMAC-SHA256 before it's written to the query log, using a per-install
  secret generated on first boot and persisted next to the database
  (`HYDRA_ANON_SECRET` overrides it). This is pseudonymisation, not anonymisation: anyone
  who holds both the database and the secret file can brute-force the small IPv4 address
  space back to the original addresses. Previously this config key was parsed but never
  wired to anything, so enabling it had no effect at all.

[Unreleased]: https://github.com/hydradns/hydradns/commits/main
