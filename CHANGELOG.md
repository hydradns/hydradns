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

### Fixed
- `docker-compose.yml` shipped `CORS_ORIGINS=*` for the `core` service,
  which — combined with the control plane's `AllowCredentials: true` —
  disabled CORS protection on the API entirely. Default is now
  `CORS_ORIGINS=${CORS_ORIGINS:-http://localhost:3000}`, matching
  `.env.example` and the documented default; overriding it for LAN/Pi
  dashboard access is documented in `docs/pi-deployment.md`.
- `demo/install.tape` typed a comment claiming a blocked `dig` query returns
  `REFUSED`; the shipped default (`BLOCK_RESPONSE=zero`) actually returns
  `A 0.0.0.0`. The tape also assumed a prebuilt `./apps/cli/hydra` binary and
  an already-authenticated CLI, neither of which exist on a fresh clone
  before the first tagged release. Rewritten to build the CLI and mint a
  setup token first, matching real first-boot behavior.
- `docs/pi-deployment.md` repeated the same `REFUSED` claim in its DNS
  verification and secondary-DNS-fallback sections, and didn't cover the
  `CORS_ORIGINS` / `NEXT_PUBLIC_API_URL` same-machine defaults that break
  dashboard access from a LAN IP (the common Pi deployment case).

[Unreleased]: https://github.com/hydradns/hydradns/commits/main
