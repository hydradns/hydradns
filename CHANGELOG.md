# Changelog

All notable changes to HydraDNS are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project aims
to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html) from v0.1.0.

## [Unreleased]

### Added
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

[Unreleased]: https://github.com/hydradns/hydradns/commits/main
