# Critique #002: Tier 1 Demo-Ready Review

**Date:** 2026-04-08
**Scope:** All Tier 1 changes (auth, setup wizard, blocklist auto-update, port 53, Pi deployment)
**Findings:** 16 issues (3 critical, 5 high, 5 medium, 3 low)

## Fixed Issues

### Critical

| # | Issue | Fix |
|---|---|---|
| 2 | Auth bypass before setup — all routes unprotected when no admin exists | Changed middleware: when `!setup`, return 403 on non-exempt routes instead of allowing all |
| 8 | Error messages leak internal DB details (GORM errors) | Return generic messages to clients, log details server-side |

### High

| # | Issue | Fix |
|---|---|---|
| 7 | SSRF via blocklist URLs — no scheme validation | Added `http://`/`https://` scheme check before creating blocklist source |
| 11 | Options spread in api.ts overwrites auth headers | Destructure options, merge headers explicitly |
| 12 | Setup ignores CreateSource errors — silent data loss | Check error, accumulate warnings array returned in response |
| 13 | Upstream resolvers field accepted but silently discarded | Removed from backend API and setup wizard (DNS config is static) |

### Medium

| # | Issue | Fix |
|---|---|---|
| 14 | Login page redirect loop when API unreachable | Changed `checkSetupStatus()` to return tri-state: complete/needs_setup/unreachable |

## Deferred Issues (acceptable for demo)

| # | Severity | Issue | Why deferred |
|---|---|---|---|
| 1 | CRITICAL | TOCTOU race on setup endpoint | SQLite unique index prevents duplicates; error message now generic |
| 3 | CRITICAL | API key stored as plaintext in DB | Acceptable for single-user LAN device; noted as tech debt |
| 4 | HIGH | Non-expiring, non-rotatable token | Single admin on LAN device; no real session management needed for demo |
| 5 | HIGH | Frontend middleware only checks cookie existence, not validity | API returns 401 on invalid tokens; user sees login redirect |
| 6 | HIGH | Cookie lacks Secure/HttpOnly flags | HTTP-only LAN device; no TLS yet |
| 9 | MEDIUM | SQLite dual-process contention | WAL mode + single writer handles it; add PRAGMA busy_timeout later |
| 10 | MEDIUM | Blocklist refresh overlap possible | 6h default interval makes overlap negligible |
| 15 | LOW | CLI --token flag visible in process list | Env var and file alternatives exist and are documented |
| 16 | LOW | Pure-Go SQLite performance on Pi | Acceptable for demo scale; monitor under real load |

## Setup Wizard Change

Reduced from 3 steps to 2 steps:
1. Set admin password
2. Choose blocklist sources

Removed the upstream DNS step because the backend has no way to persist resolver changes at runtime (config.yaml is read-only after startup). Upstream DNS is now a deployment-time config only.
