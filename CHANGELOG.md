# Changelog

All notable changes to HydraDNS are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project aims
to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html) from v0.1.0.

## [Unreleased]

### Added
- Blocklist parsers for the `domains` (one host per line) and `adblock`
  (EasyList `||domain^`) formats, alongside the existing `hosts` format.

### Changed
- Consolidated the five service repositories into a single monorepo under
  `apps/`. A plain `git clone` now checks out the whole stack.
- Renamed the project internals from PhantomDNS to HydraDNS (Go module paths,
  protobuf package, environment variables, and the on-disk database).

[Unreleased]: https://github.com/hydradns/hydradns/commits/main
