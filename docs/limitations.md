# Known Limitations

Honest list of what HydraDNS doesn't do yet, checked against the code, not the pitch.
If something here changes, update this file in the same commit.

## Security

- **No TLS on the dashboard.** The Next.js dashboard is served over plain HTTP; nothing in
  this repo terminates HTTPS for it. *Impact:* login password and bearer token cross the
  network in clear text unless you add TLS yourself. *Workaround:* put a reverse proxy
  (Caddy, nginx, Traefik) in front of the dashboard and terminate TLS there, or only expose
  it on a network you trust.
- **No TLS on gRPC.** The control plane's connection to the dataplane uses
  `grpc.WithInsecure()`. *Impact:* only matters if you split control plane and dataplane
  across hosts — the shipped Docker Compose stack runs them in one container, so this
  traffic never leaves localhost. *Workaround:* don't split the two across a network you
  don't trust until this is fixed.
- **No encrypted DNS in or out.** Upstream resolvers are plain UDP/TCP (e.g. `8.8.8.8:53`);
  there's no DoH or DoT listener for clients, and no DoH/DoT upstream support. The one
  DoH-related feature that exists is a hardcoded list of DoH bootstrap hostnames that get
  NXDOMAIN'd so browsers fall back to system DNS — that's a bypass mitigation, not encrypted
  DNS. *Impact:* your DNS to and from HydraDNS itself is plaintext, same as most home/office
  DNS setups. *Workaround:* none from HydraDNS today; pair it with a network-level VPN if you
  need transport encryption.
- **No DNSSEC validation.** Responses aren't checked against DNSSEC signatures. *Impact:*
  HydraDNS doesn't protect against a spoofed/cache-poisoned upstream answer for a
  DNSSEC-signed zone. *Workaround:* point upstream resolvers at a validating resolver (e.g.
  Cloudflare 1.1.1.1 or Quad9) — you inherit their validation, though HydraDNS itself
  doesn't re-check it.
- **Query logs store client IPs as-is by default.** Per-device DNS activity is visible in the
  query log unless you turn anonymization on. *Impact:* anyone with dashboard or DB access
  can see which device on your network looked up which domain — by design, since per-device
  visibility is the point of a home/office DNS firewall's query log. *Workaround:* set
  `HYDRA_ANONYMIZE_CLIENT_IPS=true` (or `dataplane.anonymization.enabled: true` in
  `config.yaml`) to hash client IPs with a per-install secret before they're written to the
  log; until then, restrict who has dashboard/API access.
- **No MFA/TOTP or SSO.** Login is email + password only; roles (`admin`/`operator`/
  `read_only`) exist, but there's no second factor and no OIDC/SAML integration.
  *Workaround:* use a strong, unique password per account and rotate tokens periodically
  from the dashboard's token page.

## Policy engine

- **Regex and wildcard policies are accepted but not enforced.** You can save a policy with
  a `regexes` field or a wildcard domain and it will validate and store, but the query-time
  evaluator only does exact-domain and parent-domain matching — the regex/wildcard path is
  a no-op. *Impact:* a policy that looks like it blocks `*.badsite.com` via regex won't
  actually block anything beyond exact domain matches. *Workaround:* list every domain you
  want blocked explicitly, or use a blocklist source instead of a regex policy.
- **No inline edit for policies or blocklists.** The API supports create and delete only.
  *Impact:* changing a policy's domains or a blocklist's URL means deleting and recreating
  it. *Workaround:* delete + recreate; note this loses the item's ID and creation date.

## Dashboard and operations

- **Query log is capped at 100 rows, no pagination.** The recent-queries view always shows
  the newest 100 entries. *Impact:* you can't page back further in the UI. *Workaround:*
  query the SQLite DB directly, or use the CLI/MCP `get_query_logs` tool with your own
  filtering.
- **Settings page has no backend.** The dashboard has a Settings screen, but the control
  plane has no `/settings` endpoint — the page can't actually persist anything yet.
  *Impact:* toggles on that page don't do anything durable. *Workaround:* use `hydra engine`,
  the Policies/Blocklists pages, or the config file/env vars documented in `CLAUDE.md` for
  the settings that do exist.
- **`/dns/resolvers` is read-only.** It reflects the resolvers in `config.yaml`; there's no
  API or UI to add/remove upstream resolvers. *Workaround:* edit `configs/config.yaml` and
  restart the dataplane.
- **No container self-update.** The `hydra` CLI binary can self-update (`hydra update`), but
  the `core`/`ui` Docker containers don't auto-pull new versions. *Workaround:* re-run
  `docker compose pull && docker compose up -d` (or your install script) manually.
- **No fleet/remote monitoring.** There's no heartbeat or alerting pipeline for
  multi-device deployments; each install is monitored on its own. *Workaround:* wire your
  own uptime check against `/health`.

## Performance numbers

- **All throughput and latency testing so far has been on a development laptop under WSL2,
  not a Raspberry Pi.** The engine has been load-tested up to roughly 9,500 queries/second
  on that laptop (22 cores) with the DNS engine holding p50 ≈ 5ms / p99 ≈ 20ms, and soak-tested
  for 3 minutes at ~1,380 QPS (248k queries, zero engine errors) — not a 24-hour run, and not
  on the ARM hardware most people will actually deploy on. *Impact:* none of that tells you
  what to expect on a Pi 4 or an Orange Pi Zero — CPU, thermals, and SD-card I/O are all
  different there. *Workaround:* treat these numbers as "the engine itself is not the
  bottleneck on a laptop," not as a hardware sizing guide, until a real Pi benchmark pass
  exists.

---

Found something here that's stale, or something missing that should be on this list?
Open an issue — a wrong "known limitation" is arguably worse than an undocumented one.
