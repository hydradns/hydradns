# HydraDNS public demo

This directory hosts a read-only, clickable demo of the HydraDNS dashboard —
the same images the real product ships, running with `HYDRA_DEMO_MODE=true`
so a visitor can explore the whole UI without installing anything and
without being able to change (or see anything sensitive about) a real
deployment.

## What demo mode actually is

Setting `HYDRA_DEMO_MODE=true` on the `core` container (the only thing
`docker-compose.demo.yml` does beyond normal config) does four things,
entirely server-side:

1. **Every mutating request is rejected.** A guard middleware
   (`middlewares.DemoGuard`, `apps/core/cmd/controlplane/middlewares/demo.go`)
   runs before authentication and rejects any request that isn't GET, HEAD,
   or OPTIONS with `403 {"status":"error","error":"demo mode: changes are
   disabled"}` — except `POST /api/v1/auth/login`, which is how you sign in
   as the demo account. `POST /api/v1/auth/setup` is **not** allowlisted:
   demo mode never needs it (see next point), and allowing it would let any
   visitor mint their own admin account. Because this runs ahead of
   authentication, no role — including admin, if a demo deployment somehow
   had one — can bypass it. This is the real security boundary; everything
   the dashboard UI does (the banner, buttons that still look clickable) is
   cosmetic on top of it.
2. **A fixed-password, read-only demo account is seeded at startup.**
   Username/email `demo@hydradns.local`, password `hydradns-demo`, role
   `read_only`. It's created idempotently (safe across restarts) and is
   never an admin account. The password is deliberately public — see
   "Login and the rate limit" below for why that's fine.
3. **The database is filled with synthetic data**, not real DNS traffic: a
   handful of policies, a few blocklist sources with plausible domain
   counts, and ~3,000 synthetic query-log rows spread over the last 7 days
   (allowed/blocked/redirected/flagged, including some
   `detection_method=doh_bootstrap` bypass attempts) so every chart on the
   Overview, Logs, Policies, Blocklists, and Bypass panels has something to
   show. This regenerates every 30 minutes so a long-running demo container
   doesn't drift into showing "last activity: 3 weeks ago." See
   `apps/core/cmd/controlplane/demoseed/` — it's a self-contained package,
   deliberately easy to delete if this feature is ever removed.
4. **Client IPs are redacted in every response that carries one**
   (`192.168.1.42` → `192.168.1.x`) on the query-log, bypass-attempts, and
   audit-log endpoints — defense in depth on top of the fact that, as long
   as you don't publish DNS ports (see below), every row in those tables is
   synthetic to begin with.

## What still works vs. what's blocked

Visitors can browse every page and see real (synthetic) data: Overview,
Logs (with filters/pagination), Policies, Blocklists, DNS engine status,
metrics, and the bypass-attempts panel. They can also log in/out as the
demo account.

Anything that mutates state — creating/editing/deleting a policy or
blocklist, toggling the DNS engine, managing users or tokens, the setup
wizard — returns the 403 above. The dashboard doesn't hide these controls;
per the design goal, a visitor should be able to see what the product can
do, they just can't actually do it. Clicking one shows a toast: "Read-only
demo — changes are disabled. Install your own HydraDNS to try this for
real."

The demo **does not accept real DNS traffic.** No DNS port (53/1053) is
published anywhere in `docker-compose.demo.yml`. The `core` container still
runs the normal combined controlplane+dataplane image — this is the
simplest, most honest option: no dataplane code changes, and
`GET /dns/engine` / `GET /dns/metrics` answer from a genuinely live
dataplane (you'll just see near-zero live query metrics, since nothing is
really being resolved — that's expected and correct).

## Running it

```bash
cd demo
cp .env.example .env   # then edit DEMO_PUBLIC_ORIGIN — see "Required .env values" below
docker compose -f docker-compose.demo.yml up -d
```

Then put a reverse proxy in front of it — see "TLS and the reverse proxy
layout" below; running it with plain HTTP on a public host is not
recommended (login submits a password over the wire, and browsers
increasingly restrict things like clipboard/cookies on non-HTTPS origins).

### Required `.env` values

`demo/.env.example` ships with placeholder values only — the correct values
are specific to your domain, so copy it and edit before starting the stack:

```bash
# The public origin visitors will use, scheme included. Must match exactly
# what's in the browser's address bar (scheme + host), not the internal
# docker hostname.
DEMO_PUBLIC_ORIGIN=https://demo.example.com

# Only set this if you're running a reverse proxy (you should — see
# below). Value is that proxy's container name or address on the
# hydra-demo-net network, e.g. "caddy" if you name the service that.
DEMO_TRUSTED_PROXY=caddy

# Optional: pin a released version instead of tracking :latest.
HYDRA_VERSION=latest
```

If `DEMO_PUBLIC_ORIGIN` is unset, `docker compose up` fails fast with a
clear error (`docker-compose.demo.yml` uses `${DEMO_PUBLIC_ORIGIN:?...}`)
rather than starting with a CORS configuration that silently rejects the
dashboard.

## TLS and the reverse proxy layout

The dashboard derives the control-plane API URL from its own page URL at
runtime: same protocol and hostname, port 8080
(`apps/ui/lib/api-base.ts`). So behind a proxy on `https://demo.example.com`,
the API must *also* answer HTTPS at `https://demo.example.com:8080` — not
just at 443. The simplest correct layout is one reverse proxy, one
certificate, two site blocks on the same hostname (443 for the dashboard,
8080 for the API), both proxying to the two containers over the internal
compose network. `docker-compose.demo.yml` intentionally does not include
a proxy service or bake in a Caddyfile (TLS setup is host-specific — DNS,
certs, and firewall rules are yours to own); run Caddy yourself, e.g. as a
sibling container attached to `hydra-demo-net`, or directly on the host.

Exact `Caddyfile`:

```caddyfile
demo.example.com {
	reverse_proxy ui:3000
}

demo.example.com:8080 {
	reverse_proxy core:8080
}
```

Caddy auto-provisions and renews a Let's Encrypt certificate for
`demo.example.com` and reuses the same certificate for the `:8080` site
block (same hostname, so no extra ACME challenge is needed). If running
Caddy as a container, attach it to `hydra-demo-net` (so `ui:3000` and
`core:8080` resolve) and publish `80:80` (ACME HTTP challenge), `443:443`,
and `8080:8080` on the host — port **8080 must be reachable over HTTPS
from the public internet**, not just from inside the compose network,
since the dashboard's JS calls it directly from the visitor's browser.

Remember to also set, in `demo/.env`:

```bash
DEMO_PUBLIC_ORIGIN=https://demo.example.com
DEMO_TRUSTED_PROXY=caddy   # or whatever you name the proxy service/container
```

`CORS_ORIGINS` is derived from `DEMO_PUBLIC_ORIGIN` automatically by
`docker-compose.demo.yml` — the control plane's automatic same-host CORS
allowance only covers IP literals and `localhost` (by design, to prevent
DNS rebinding — see `apps/core/cmd/controlplane/middlewares/cors.go`), so a
named public hostname like `demo.example.com` needs this explicit entry;
without it the dashboard loads but every API call fails as a CORS error in
the browser console.

**Path-based alternative:** if you'd rather not expose a second public
port, you can instead proxy `https://demo.example.com/api/*` to `core:8080`
under the same 443 site block, but then the dashboard image must be
rebuilt with `NEXT_PUBLIC_API_URL=https://demo.example.com/api` baked in at
build time (Next.js inlines this value; a runtime env var on an
already-built image has no effect) — see the root `docker-compose.yml`'s
`ui.build.args` for where that goes. The two-port layout above needs no
image rebuild, which is why it's the default recommendation here.

## Login and the rate limit

The demo password (`hydradns-demo`) is intentionally public — the API
allows only `POST /auth/login` through in demo mode regardless of
credentials used, so "guessing" it buys an attacker nothing they don't
already have. Given that, this deployment relaxes (does not remove) the
login endpoint's per-IP rate limit specifically when `HYDRA_DEMO_MODE=true`:
100 attempts / 5 minutes instead of the normal 10 / 5 minutes (see
`demoLoginRateLimitAttempts` in
`apps/core/cmd/controlplane/routes/router.go`). Only a failed (4xx) login
attempt counts against that budget; a successful one does not, so
visitors logging in and out repeatedly with the correct demo password
never trip it. This was a deliberate
trade-off: a public demo is often reached by many visitors behind one
shared NAT or corporate proxy, and the normal 10-attempt budget — meant to
slow down a real password-guessing attack — would routinely lock out an
entire office over a handful of people clicking "Enter Demo." The relaxed
limit keeps *a* bound in place (still protects against unbounded scripted
abuse / log spam) without that collateral lockout. This is the least-risky
option available: tightening `DEMO_TRUSTED_PROXY` correctly (see above) so
each visitor's real IP is seen individually, rather than raising the limit
further or removing it, is the better lever if lockouts still happen in
practice.

## Resetting the demo

Everything the demo has "learned" — the seeded data, the demo account, any
attempted-then-rejected requests reflected in query counts — lives in the
`demo-core-data` volume. To reset to a clean state:

```bash
cd demo
docker compose -f docker-compose.demo.yml down
docker volume rm demo_demo-core-data   # prefix is the compose project name; `docker volume ls` to confirm
docker compose -f docker-compose.demo.yml up -d
```

You don't normally need to do this: the query-log/statistics portion of
the seed already re-anchors itself to "now" every 30 minutes on its own
(see `demoseed.StartRefreshLoop`), so a long-running demo doesn't go stale
without a reset. A full reset is for when you want to pick up a newer seed
data set after upgrading, or to clear out anything unexpected.

## Abuse considerations

- **Write access:** blocked entirely by `DemoGuard`, ahead of
  authentication — see above. This is the main thing this section would
  otherwise be about.
- **DNS abuse (open resolver, DNS amplification, etc.):** not applicable —
  no DNS port is published, ever, in this compose file.
- **Login brute force / credential stuffing:** the account is read-only
  and its password is meant to be public, so there is nothing to
  "compromise" by logging in — see "Login and the rate limit" above for why
  the throttle is relaxed rather than removed.
- **Resource exhaustion:** `deploy.resources.limits` caps both containers'
  CPU/memory. The query-log page is already server-side paginated and
  capped (`apps/core` hard-caps page size at 200 rows), so a scripted
  scraper hitting `/analytics/logs` in a loop is bounded per request; it
  is not currently additionally rate-limited beyond that, so if this
  becomes a real problem, add a general per-IP rate limit at the reverse
  proxy (Caddy's `rate_limit` via a plugin, or a sidecar like fail2ban on
  access logs) rather than in the application — this demo intentionally
  does not invent a second, redundant rate-limiting layer in Go for
  read-only GETs.
- **Information disclosure:** reviewed every GET route that returns
  anything resembling a secret or PII: token values are never returned by
  any GET route in any mode (verified against
  `apps/core/cmd/controlplane/handlers/tokens.go`'s DTO, which excludes
  `Hash`), the demo user's own email is a synthetic placeholder
  (`demo@hydradns.local`), and every client-IP-bearing response (query
  logs, bypass attempts, the RBAC audit log) is redacted in demo mode — see
  "What demo mode actually is" above.
