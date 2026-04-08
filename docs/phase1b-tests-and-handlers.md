# Phase 1b: Tests & Real Handler Wiring

**Date:** 2026-04-05
**Status:** Complete

## Context

After fixing the core bugs, Phase 1 still had two gaps: zero test coverage and mock data in all control plane handlers. This phase closed both.

## Tests Written (52 total, 5 packages)

| Package | Tests | Coverage |
|---|---|---|
| `internal/policy` | 14 | Evaluate, normalization, priority, tie-break, bloom filter, redirect, edge cases |
| `internal/dnsengine` | 16 | ProcessDNSQuery (drain, blocklist, policy, ordering, normalization, errors), respondBlocked, respondRedirect, status |
| `internal/storage/repositories` | 20 | BlocklistRepo (IsBlocked, normalization, GetAll, SaveSnapshot, source CRUD, cascade delete), QueryLogRepo, StatisticsRepo (counters), SystemStateRepo, PolicyRepo (CRUD, ordering) |
| `internal/metrics` | 5 | Bucket placement, aggregation, percentile estimation |
| `internal/blocklist/parser` | 6 | Hosts parser (parse, comments, empty), registry |

### Testing approach
- **Policy & DNS engine**: Pure unit tests with mock interfaces (`mockBlocklist`, `mockResponseWriter`)
- **Storage repos**: Integration tests against in-memory SQLite (`:memory:`) — tests the real GORM queries
- **Metrics & parser**: Pure unit tests, no mocks needed

## Handlers Wired to Real Storage

### New storage code
- `PolicyRepository` — new interface + implementation with List, GetByID, Create, Update, Delete
- `Policy` model upgraded — string ID, name, description, domains (JSON text), priority, enabled
- `BlocklistRepository` extended — added ListSources, GetSource, CreateSource, DeleteSource (cascade), CountEntriesBySource

### Mock data removed

| Handler | Before | After |
|---|---|---|
| Blocklists | `mockBlocklists` slice | `BlocklistRepository` source CRUD |
| Policies | `mockPolicies` slice | `PolicyRepository` CRUD |
| Dashboard | Hardcoded numbers | `StatisticsRepository.ListRecent` |
| Analytics | Hardcoded numbers | `StatisticsRepository.ListRecent` |
| Audit logs | Fake entries | `QueryLogRepository.ListRecent` |
| Resolvers | `mockResolvers` + add/delete | Read-only from `config.yaml` |

### Removed endpoints
- `POST /dns/resolvers` and `DELETE /dns/resolvers/:id` — resolvers are config-managed, not runtime-mutable

## Verification

- `go build ./...` — clean
- `go vet ./...` — clean
- `go test ./...` — 52 tests, all passing

## Lessons Learned

1. **In-memory SQLite is perfect for repo tests** — No mocking GORM, no test containers. Just `sqlite.Open(":memory:")` with AutoMigrate. Fast (0.1s) and tests real SQL.
2. **Mock interfaces at boundaries, not everywhere** — The `BlocklistChecker` interface on the engine was the right seam. No need to mock the policy engine — just load real policies into it.
3. **Domains as JSON text in SQLite** — Storing `[]string` as a JSON text column is pragmatic for SQLite. Not ideal for querying individual domains, but fine for CRUD.
4. **Resolvers don't belong in the DB** — They're infrastructure config, not user data. Reading from `config.yaml` is simpler and more correct than CRUD endpoints.
