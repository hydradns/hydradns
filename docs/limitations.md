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
- **Client-IP "anonymization" is pseudonymisation, not anonymisation.** Turning on
  `HYDRA_ANONYMIZE_CLIENT_IPS` hashes each client IP with HMAC-SHA256 and a per-install
  secret before it's written to the query log — it does not remove the ability to identify a
  device. *Impact:* IPv4 addresses are a small enough space (2^32) that anyone who obtains
  both the database and the secret (`HYDRA_ANON_SECRET`, or the generated
  `<data dir>/anon_secret` file) can brute-force every hash back to the original address in
  practice. It stops casual inspection of the query log, not a determined attacker with both
  artifacts. *Workaround:* protect the secret file/env var with the same care as the database
  itself; don't treat this setting as true anonymisation for compliance purposes.
- **First-boot setup is "first request wins" on the LAN.** `POST /api/v1/auth/setup` is open
  (unauthenticated) until the first admin user is created, and it's reachable from anywhere
  that can reach the control plane's port. *Impact:* if you expose the box to an untrusted
  network before completing setup, whoever gets there first becomes the admin, not you.
  *Workaround:* finish the setup wizard before exposing the box to an untrusted network (LAN
  or otherwise) — don't publish the control plane's port to the internet pre-setup.
- **The login/setup rate limit is per-IP and in-memory.** `POST /auth/login` and
  `POST /auth/setup` are throttled per client IP (10 attempts / 5 minutes by default, 100 / 5
  minutes when `HYDRA_DEMO_MODE=true`, shared across both endpoints). Only failed attempts
  count against the budget — a 4xx response (bad credentials, bad request body, setup already
  complete) consumes budget; a successful login and a 5xx (the server's own fault) do not.
  *Impact:* the counter lives in process memory, so it resets on every restart, and it only
  ever sees one bucket per source IP — correctness in front of a reverse proxy depends on
  `TRUSTED_PROXIES` being configured for that proxy, otherwise every client behind it shares
  one bucket (or, if the proxy's own address isn't trusted, the limiter can be bypassed
  entirely by spoofing `X-Forwarded-For`, per `TRUSTED_PROXIES`' own default of trusting
  nothing). *Workaround:* set `TRUSTED_PROXIES` correctly if you're behind a reverse proxy;
  don't rely on this limiter surviving a restart or coordinating across multiple instances.
- **The blocklist fetcher will download from hosts on your LAN.** Blocklist URLs must be
  `http` or `https`. The fetcher refuses loopback, link-local (including the `169.254.169.254`
  cloud metadata address), unspecified and multicast addresses, checks the address at connect
  time on every redirect hop, follows at most 5 redirects and caps a download at 128 MiB.
  Private LAN addresses are allowed on purpose, because people host lists on their own network.
  *Impact:* an operator account, or a list host that redirects, can make the box request a URL
  on your LAN. *Workaround:* only give operator accounts to people you trust and only add list
  URLs from hosts you trust.
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
- **A new or edited blocklist starts working after its download finishes.** Adding, enabling,
  disabling or deleting a blocklist reaches the DNS engine within about 5 seconds
  (`BLOCKLIST_POLL_INTERVAL`), the same as policy edits. A new source, or a source whose URL
  you changed, takes effect when the download of the new list completes; the previous list's
  entries are replaced at that point. *Impact:* seconds to a few minutes of delay for large
  lists on slow links.
- **A blocklist source keeps the last 10 snapshot metadata rows.** Older snapshot metadata
  (fetch time, size, checksum) is pruned; this is bookkeeping only and does not affect which
  domains are currently blocked. *Impact:* you can't see fetch history older than the last 10
  fetches for a given source. *Workaround:* none needed for normal use.
- **If a blocklist rebuild fails, the in-memory list keeps its previous contents** rather than
  going empty, and the dataplane retries on the next `BLOCKLIST_POLL_INTERVAL` tick.
  *Impact:* a transient DB error does not open up traffic that should be blocked; you keep
  enforcing the last-known-good list until the retry succeeds. *Workaround:* none needed;
  check logs if rebuilds keep failing.

## Dashboard and operations

- **Dashboard recent-activity widget shows the newest 100 queries only.** The dedicated Logs
  page pages and filters on the server; its domain filter is a prefix match, not a substring
  search. *Workaround:* use the Logs page for anything older than the newest 100 rows.
- **`GET /analytics/logs` rejects requests that reach too far into the table.** `page *
  page_size` above 100,000 returns HTTP 400, and the total-row count is capped at 100,000 too
  (the response includes `total_capped: true` when the real count is higher than that). The
  `client=` filter works with `HYDRA_ANONYMIZE_CLIENT_IPS` on (it hashes the filter value the
  same way stored IPs are hashed) but is rejected outright in demo mode, since an exact-match
  filter against unmasked storage would let a demo visitor use it to recover the IP octet
  masked in responses. *Impact:* you cannot page arbitrarily deep into a very large query log,
  and demo-mode visitors cannot filter by client. *Workaround:* narrow the search with
  `domain`/`action`/`start`/`end` instead of paging deep.
- **Settings page has no backend.** The dashboard has a Settings screen, but the control
  plane has no `/settings` endpoint — the page can't actually persist anything yet.
  *Impact:* toggles on that page don't do anything durable. *Workaround:* use `hydra engine`,
  the Policies/Blocklists pages, or the config file/env vars documented in `CLAUDE.md` for
  the settings that do exist.
- **`/dns/resolvers` is read-only.** It reflects the resolvers in `config.yaml`; there's no
  API or UI to add/remove upstream resolvers. *Workaround:* edit `configs/config.yaml` and
  restart the dataplane.
- **The dashboard's runtime API-URL derivation assumes a two-port reverse proxy.** With
  `NEXT_PUBLIC_API_URL` left unset, the dashboard calls the control plane at the page's own
  hostname on port 8080 (`apps/ui/lib/api-base.ts`). *Impact:* behind a reverse proxy that
  only terminates TLS on 443 (a common single-port setup), the API is unreachable at 8080
  over HTTPS and every dashboard API call fails, unless port 8080 is also exposed over HTTPS
  on the same hostname. *Workaround:* either expose a second HTTPS site block on port 8080
  proxying to the control plane (see `demo/README.md`'s Caddyfile for a worked example), or
  rebuild the `ui` image with `NEXT_PUBLIC_API_URL` set to a path-based API URL under the
  same 443 origin — a runtime env var alone has no effect on an already-built image, since
  Next.js inlines this value at build time.
- **Demo mode (`HYDRA_DEMO_MODE`) is for public demos only.** It refuses to start against a
  database that already has users other than the seeded demo account, and it also refuses to
  start against a database that has query-log rows but no users at all (a pre-RBAC volume, or
  one mid-migration), specifically so it can't be turned on accidentally against a real
  deployment. *Impact:* none for a normal install — this is a guardrail, not a general-purpose
  feature. *Workaround:* n/a; use it only with a fresh volume, as documented in
  `demo/README.md`.
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
