# Changelog

All notable changes to HydraDNS are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project aims
to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html) from v0.1.0.

## [Unreleased]

## [0.1.0] - 2026-09-20

### Added
- Added a public demo mode (`HYDRA_DEMO_MODE=true`): a read-only demo with a fixed
  password, seeded synthetic policies/blocklists/query-logs, and masked client IPs, meant
  for a public clickable demo rather than a normal self-hosted install. It refuses to
  start against a database that already has real users, and ships with its own
  `demo/docker-compose.demo.yml` stack and `demo/README.md`.
- `hydra setup` is a first-boot CLI command that creates the admin account through the
  setup endpoint and stores the token, the same way `hydra login` does. It supports a
  no-echo password prompt or `--password-stdin`; there is deliberately no `--password`
  flag.
- Added API routes backing dashboard features that previously had no server side:
  `GET /analytics/logs` (server-side paging and filtering for the Logs page, page size
  capped at 200), `GET /analytics/bypass` (DoH/DoT bootstrap-detection attempts),
  `PUT /policies/:id`, and `PATCH /blocklists/:id`. The latter two carry the same
  validation, role checks, and audit events as create/delete.
- Added a legacy-database upgrade path: on startup, if `hydradns.db` doesn't exist but a
  pre-rename `phantomdns.db` does, the control plane opens that file instead of starting
  empty. This is a one-time upgrade fallback, not the default for new installs.
- The DoH-bypass-attempts panel is now hidden on the dashboard by default; set
  `NEXT_PUBLIC_SHOW_BYPASS_PANEL=true` at build time to show it (e.g. for
  technical/internal deployments).
- Added `apps/cli/Dockerfile` and published `ghcr.io/hydradns/hydra-cli`, a small
  non-root image whose default command runs `hydra mcp` (stdio JSON-RPC), so the CLI's
  MCP server can run via `docker run -i --rm -e HYDRA_API_URL -e HYDRA_TOKEN
  ghcr.io/hydradns/hydra-cli` without a local Go toolchain. See `docs/mcp.md` for the
  full tool list, roles, and per-platform container networking.
- Added `docs/mcp.md`, public documentation for the `hydra mcp` server: the 14 tools,
  `MCP_ROLE` permission scopes, security notes, and verified client configuration for
  Claude Desktop, Claude Code, Cursor, VS Code, and the Gemini CLI, for both the local
  binary and the container image.
- Added blocklist parsers for the `domains` (one host per line) and `adblock` (EasyList
  `||domain^`) formats, alongside the existing `hosts` format.
- Added `docs/releasing.md`, a release runbook for cutting tags: pre-flight checks, what
  `release.yml` publishes, GHCR visibility/verification, and rollback.
- Added an automatic same-host CORS rule on the control plane API
  (`apps/core/cmd/controlplane/middlewares/cors.go`): a request is allowed when its
  `Origin` hostname matches the `Host` header it was reached on and that hostname is an
  IP literal or `localhost`, e.g. so a dashboard opened at `http://192.168.1.53:3000` can
  call the API at `http://192.168.1.53:8080` with no `CORS_ORIGINS` edit. Named hosts are
  excluded on purpose, since a DNS-rebinding attack could make `Origin` and `Host` agree
  on an attacker-controlled name and so still need an explicit `CORS_ORIGINS` entry;
  disable the whole rule with `CORS_ALLOW_SAME_HOST=false`.

### Changed
- Consolidated the five service repositories into a single monorepo under `apps/`. A
  plain `git clone` now checks out the whole stack.
- Renamed the project internals from PhantomDNS to HydraDNS (Go module paths, protobuf
  package, environment variables, and the on-disk database).
- `docker-compose.yml` now declares `image: ghcr.io/hydradns/<service>:${HYDRA_VERSION:-latest}`
  alongside the existing `build:` section for `core` and `ui`, so `docker compose up -d`
  pulls a published release instead of compiling from source (before the first release,
  or offline, Compose still falls back to building locally). Pin a version via
  `HYDRA_VERSION` in `.env`.
- The dashboard now derives the control-plane API's address from the page's own URL at
  runtime (same protocol and hostname, port 8080) instead of relying only on the value
  baked in at build time, so the published `ghcr.io/hydradns/ui` image works over a LAN IP
  without a rebuild. `NEXT_PUBLIC_API_URL` is now needed only to override this, e.g. for a
  reverse proxy or an API on a different host/port than the dashboard.

### Fixed
- `hydra update` could never find an update once the CLI moved into this monorepo, since
  its release feed still pointed at the archived `hydradns/hydra-cli` repo; it now points
  at `hydradns/hydradns`, where `release.yml`'s `release-cli` job publishes the CLI
  binaries. `hydra update` also refuses to install a binary without a matching SHA-256
  checksum, so a new `release-cli-checksums` job now generates and uploads
  `checksums.txt` for each release.
- `docker-compose.yml` shipped `CORS_ORIGINS=*` for the `core` service, which combined
  with the control plane's `AllowCredentials: true` disabled CORS protection on the API
  entirely. The default is now `CORS_ORIGINS=${CORS_ORIGINS:-http://localhost:3000}` plus
  the automatic same-host rule above, matching `.env.example`; an explicit `CORS_ORIGINS`
  entry is only needed for a named host, not ordinary LAN-by-IP access.
- `demo/install.tape` incorrectly commented that a blocked `dig` query returns `REFUSED`;
  the shipped default (`BLOCK_RESPONSE=zero`) actually returns `A 0.0.0.0`. It also
  assumed a prebuilt `./apps/cli/hydra` binary and an already-authenticated CLI, neither
  of which exist on a fresh clone before the first tagged release, so it's rewritten to
  build the CLI and mint a setup token first.
- `docs/pi-deployment.md` repeated the same `REFUSED` claim in its DNS verification and
  secondary-DNS-fallback sections, and its "Dashboard not accessible from LAN" section
  documented `CORS_ORIGINS` and `NEXT_PUBLIC_API_URL` as manual steps for LAN-by-IP
  access. Both are now automatic (see the same-host CORS rule and the dashboard's runtime
  API-URL derivation above), so that section now covers only the two cases that still
  need a manual step: a named host, and a dashboard served over HTTPS.
- The in-memory blocklist used to be built from every stored entry with no check on the
  source's enabled flag, so disabling a blocklist did not stop it from blocking. It's now
  built from enabled sources only, and the dataplane polls a cheap signature of the
  blocklist tables (`BLOCKLIST_POLL_INTERVAL`, default 5s; `0` disables) and rebuilds the
  in-memory set when it changes, so add/toggle/delete/finished-download propagate in
  about 5s instead of waiting for the 6-hour `BLOCKLIST_UPDATE_INTERVAL` cycle, the same
  speed policy edits already had.
- Query-log rows were stored with the client's ephemeral port attached (`ip:port`)
  instead of the bare IP; the client-IP filter and anonymization both now operate on the
  address alone, and legacy rows with a port still match the filter precisely.
- CLI release binaries built without the version stamped in, so every release reported
  the hardcoded fallback `1.0.0` regardless of the actual tag, and `hydra update` could
  never see a release as newer than what was already installed. `release.yml` now passes
  the tag version via `-X` at build time.
- The default `apps/core/configs/policies.json` seed policy blocked `apple.com`, breaking
  iCloud, the App Store, and iMessage for a first-time user on Apple devices. It's
  removed: the seed now ships only `block-ads` and `block-malware` example policies, and
  a test asserts the shipped policy file never blocks a critical platform domain again.
- The blocklist fetch pipeline is now single-flighted with a bounded retry, and a
  source's entries are fully replaced (not appended) on every new snapshot, so a domain
  removed upstream actually stops being blocked and a URL edit takes effect once the next
  download completes. Only the last 10 snapshot metadata rows are kept per source (this
  caps metadata only, not which domains are blocked), and if an in-memory rebuild fails
  it keeps its previous, still-enforced contents and retries on the next
  `BLOCKLIST_POLL_INTERVAL` tick.
- The per-IP rate limiter's eviction path could scan up to its full entry cap under one
  global lock when the tracked-IP map was full and no sampled entry had expired, letting
  an attacker with a large address pool (e.g. an IPv6 /64) hold that lock for an
  O(maxEntries) scan on every request from a fresh IP. It now inspects a bounded sample
  (64 entries) and resets the whole map rather than continuing the scan.
- `HYDRA_DEMO_MODE=true` now also refuses to start against a database that has query-log
  rows but no users. Previously the "zero users" check alone could let it seed the demo
  user over a pre-RBAC volume (or one mid-migration), and the periodic demo-data refresh
  loop would then delete that real history on its first tick.
- `GET /analytics/logs` now caps `page * page_size` at 100,000 (400 if exceeded) and its
  row count at the same figure, returning `total_capped: true` in the response when the
  count was capped. Previously an arbitrarily large `page` forced a full linear OFFSET
  scan, and the count for an unindexed filter (e.g. `suspicious`) was a full table scan
  on every page load.
- Policy `action` is now validated (`BLOCK`/`ALLOW`/`REDIRECT`, case-insensitive) on both
  create and update, and `REDIRECT` requires a valid `redirect_ip` on both. Previously
  only create checked this, so an update could silently turn a policy into a no-op with
  an unrecognized action string.
- `blocklist_entries.source_id` is now indexed. `DeleteSource` and the dashboard's
  per-source domain count were doing an unindexed scan of the whole table.
- SQLite's `busy_timeout` is now set to 30 seconds. The combined `core` container starts
  the controlplane and dataplane processes together, and both run database migrations
  against the same single-writer SQLite file on startup; without a busy timeout, one
  process's schema migration could make the other's migration attempt fail outright with
  `SQLITE_BUSY` instead of waiting for it to finish. See the upgrade note below.
- The dataplane's blocklist signature poll loop now has its stop function wired into
  shutdown instead of discarded, so `SIGINT`/`SIGTERM` stops that background goroutine
  cleanly.
- MCP tool read-only classification is now an explicit per-tool flag instead of guessed
  from a `get_`/`list_` naming prefix. `explain_anomaly` and `compare_to_last_month` only
  read data but don't match that prefix, so the old heuristic classified them as
  mutating, blocking the `reporter` MCP role from calling them and marking them
  confirmation-required for no reason.
- `docker-compose.yml` only passed
  `HYDRA_CONFIG`/`HYDRA_DB`/`HYDRA_POLICIES`/`CORS_ORIGINS`/`CORS_ALLOW_SAME_HOST` into
  the `core` container, so every other documented runtime setting in `.env.example`
  (`BLOCK_RESPONSE`, `BLOCKLIST_UPDATE_INTERVAL`, `BLOCKLIST_POLL_INTERVAL`,
  `QUERY_LOG_RETENTION_DAYS`, `QUERY_LOG_MAX_ROWS`, `QUERY_LOG_CLEANUP_INTERVAL`,
  `HYDRA_ANONYMIZE_CLIENT_IPS`, `HYDRA_ANON_SECRET`, `TRUSTED_PROXIES`) had no effect no
  matter what you put in `.env`. All nine are now forwarded, each with a default
  matching the code's own default. `DNS_LISTEN_ADDR` and `HYDRA_DEMO_MODE` are
  deliberately still not forwarded: the DNS port mapping is fixed so overriding the
  internal bind address would only break it, and demo mode has its own compose file
  (`demo/docker-compose.demo.yml`) so a stray value here can't turn a real install into a
  demo.

### Security
- Blocklist downloads refuse non-http(s) URLs and loopback, link-local (cloud metadata),
  unspecified and multicast targets, checked at connect time on every redirect hop.
  Redirects are capped at 5 and a download at 128 MiB. Private LAN hosts remain allowed.
- Added per-IP rate limiting on `POST /auth/login` and `POST /auth/setup` (fixed window,
  10 attempts / 5 minutes by default, shared across both endpoints), plus
  `TRUSTED_PROXIES` (comma-separated CIDRs/IPs) so the control plane only honors
  `X-Forwarded-For` from a reverse proxy you explicitly trust; otherwise `c.ClientIP()`
  always resolves to the real socket address. This also makes the audit log's recorded
  client IP trustworthy: previously Gin trusted every proxy by default, letting any
  caller spoof its own client IP.
- Opt-in client-IP pseudonymisation: `HYDRA_ANONYMIZE_CLIENT_IPS` (default `false`)
  hashes a client's IP with HMAC-SHA256 before it's written to the query log, using a
  per-install secret generated on first boot and persisted next to the database
  (`HYDRA_ANON_SECRET` overrides it). This is pseudonymisation, not anonymisation: anyone
  holding both the database and the secret file can brute-force the small IPv4 address
  space back to the original addresses. Previously this config key was parsed but never
  wired to anything, so enabling it had no effect.
- `POST /auth/login` and `POST /auth/setup` now only count a failed (4xx) attempt against
  the per-IP rate limit; a successful login and a 5xx (the server's own fault) no longer
  consume budget. Previously every call counted identically, so a legitimate admin
  re-authenticating from several devices behind one shared/NAT IP could lock themselves
  out purely by logging in successfully. The limit is relaxed to 100 attempts / 5 minutes
  when `HYDRA_DEMO_MODE=true`, since the demo password is fixed and publicly documented.
- Boolean environment variables (`HYDRA_DEMO_MODE`, `CORS_ALLOW_SAME_HOST`) now go
  through one shared parser: `true`/`1`/`yes`/`on` and `false`/`0`/`no`/`off`,
  case-insensitive and whitespace-trimmed. An unrecognized value now fails startup with
  an error naming the variable and value, instead of silently keeping the default;
  previously a near-miss like `"1"`, `"yes"`, or a trailing space from a Docker env_file
  line left `HYDRA_DEMO_MODE` silently off, and any value other than the literal
  `"false"` left `CORS_ALLOW_SAME_HOST` silently on.
- `CORS_ORIGINS` entries are now trimmed and validated (must be `*` or a bare
  `http(s)://host` with no path) before reaching the CORS library, which otherwise
  panicked at startup on the first malformed entry with no indication of which one or
  why. An invalid entry now fails startup with a clear message naming `CORS_ORIGINS` and
  the offending entry.
- `CreateBlocklist` (and the setup wizard's optional blocklist bootstrap) now validate
  the source URL scheme the same way `UpdateBlocklist` already did. Previously an
  operator-role user could add a blocklist source pointing at an internal address (e.g. a
  cloud metadata endpoint or a LAN admin page) through create or setup, just not through
  update.
- `GET /analytics/logs`'s `client=` filter is now rejected outright (400) in demo mode:
  an exact-match filter against the unmasked stored rows would otherwise let a public
  demo visitor use the filter as an oracle to recover the octet masked in the response.
- The anonymization secret generator now fails startup, rather than silently falling back
  to a shared hardcoded key, if `crypto/rand` cannot produce entropy. That fallback would
  have made every affected install share the same HMAC key, defeating anonymization for
  all of them at once.
- The CLI's token file (`~/.hydra/token`) is now written atomically (temp file, fsync,
  rename) with its directory forced to `0700` and the file to `0600` on every write, and
  a symlinked directory or token path is refused rather than followed.
- The CLI now prints a one-line stderr warning when its configured API URL is plain
  `http://` to a non-local address, since the login password and bearer token would
  otherwise cross the network in cleartext with no indication.

### Upgrade notes
- This release adds two database indexes (`dns_queries.action`,
  `blocklist_entries.source_id`) and a `busy_timeout` setting; both the controlplane and
  dataplane processes run migrations at startup. On an existing install with a large
  `dns_queries` table (SD-card storage, hundreds of thousands to ~1,000,000 rows), the
  first start after upgrading builds these indexes and may take noticeably longer than a
  normal restart. This is expected and one-time; see `docs/releasing.md` for details.

[Unreleased]: https://github.com/hydradns/hydradns/compare/v0.1.0...main
[0.1.0]: https://github.com/hydradns/hydradns/releases/tag/v0.1.0
