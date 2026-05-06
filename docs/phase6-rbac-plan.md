# Phase 6: RBAC + Audit Plan

**Date:** 2026-04-23
**Status:** Design review
**Branch:** `feat/rbac-and-audit` (apps/core submodule)

## Goal

Replace the `AdminCredential` singleton with a real user model so HydraDNS can be sold to anyone with more than one admin. Tack on audit logging for every write operation, because every compliance review asks for it in the first week.

Scope is intentionally narrow: **RBAC and audit only**. SSO (OIDC + SAML), MFA, and user-facing dashboard changes are follow-up PRs in the same phase. Each is independently shippable.

## Non-goals for this PR

Listed to short-circuit scope creep:

- SSO (OIDC / SAML) - separate PR
- MFA (TOTP) - separate PR
- Dashboard UI for user management - separate PR (backend only here)
- CLI user-management commands - separate PR
- MCP token scoping to role - separate PR (tools already work; role check lands when RBAC ships)
- Password reset flow - deferred, admin role can rotate
- Migration tooling for existing production Pis - separate checklist once the core PR lands

## Data model

Four changes. Every existing model stays put.

### `User` (new)

Replaces the `AdminCredential` singleton.

```go
type User struct {
    ID           uint      `gorm:"primaryKey"`
    Email        string    `gorm:"uniqueIndex;not null"`
    PasswordHash string    `gorm:"not null"`
    Role         string    `gorm:"not null;check:role IN ('admin','operator','read_only')"`
    Disabled     bool      `gorm:"not null;default:false"`
    CreatedAt    time.Time
    UpdatedAt    time.Time
    LastLoginAt  *time.Time
}
```

Rationale:

- Email as the login identifier because every commercial DNS product uses email and SSO will plug in with email as the claim.
- Three roles, no more. Adding roles later is cheap; removing them requires data migration. `operator` is the default for normal admins; `admin` is the only role that can create/delete users; `read_only` sees everything but writes nothing.
- `Disabled` flag instead of hard-delete so audit history stays referentially intact.

### `Token` (new)

Per-user API tokens, multiple per user, revocable.

```go
type Token struct {
    ID          uint       `gorm:"primaryKey"`
    UserID      uint       `gorm:"not null;index"`
    Hash        string     `gorm:"uniqueIndex;not null"` // SHA-256 of the token, not the plaintext
    Label       string     `gorm:"not null"`              // e.g. "laptop-cli", "mcp-gemini"
    CreatedAt   time.Time
    LastUsedAt  *time.Time
    RevokedAt   *time.Time
    ExpiresAt   *time.Time
}
```

Rationale:

- Hash, not plaintext. The current system stores the UUID itself. Losing the DB today means every active token is compromised.
- `Label` because a user with a phone, a laptop, and an MCP server wants to know which token to revoke when a device is lost.
- `ExpiresAt` is nullable to preserve backwards-compat for long-lived tokens, but new tokens get a 90-day default. Enforced at validation.

### `AuditEvent` (new)

```go
type AuditEvent struct {
    ID         uint      `gorm:"primaryKey"`
    ActorID    *uint     `gorm:"index"`         // nullable for system-initiated events
    Action     string    `gorm:"not null;index"` // e.g. "policy.create", "blocklist.delete"
    Target     string    `gorm:"not null"`       // e.g. "policy:block-social"
    BeforeJSON *string   // nullable; absent on creates
    AfterJSON  *string   // nullable; absent on deletes
    ClientIP   string    `gorm:"not null"`
    UserAgent  string    `gorm:"not null"`
    CreatedAt  time.Time `gorm:"index"`
}
```

Rationale:

- JSON columns for before/after so we don't need to evolve the schema every time a domain model adds a field. SQLite TEXT with JSON validation at write time.
- `Action` uses a namespaced string (`policy.create`, not an enum). Cheap to extend, easy to filter.
- Indexed on `ActorID`, `Action`, and `CreatedAt` - the three columns compliance queries filter on.

### `AdminCredential` (deprecated, not dropped)

Keep the table for one release cycle so rollback is safe. New code reads only from `User`. Migration copies the singleton into `users` on first boot of the new binary, then stops reading `AdminCredential` forever. Table drop lands in a follow-up migration once we've seen two stable releases.

## Migration path

SQLite, single-writer, every boot runs GORM `AutoMigrate`. The critical invariant: **an existing Pi running this upgrade must not lose dashboard access**.

1. `AutoMigrate` creates `users`, `tokens`, `audit_events`.
2. On startup, if `users` is empty and `admin_credentials` has exactly one row, copy it:
    - `email = "admin@hydradns.local"` (placeholder; user can rename on first login)
    - `password_hash = admin_credentials.password_hash`
    - `role = "admin"`
    - Create a matching `Token` row with `hash = sha256(admin_credentials.api_key)`, `label = "migrated-from-singleton"`.
3. Log a single `system.migrate.admin_singleton` audit event with `actor_id = NULL`.
4. On subsequent boots, the migration check sees a non-empty `users` table and skips.

Existing bearer tokens continue to work transparently because the token-hash lookup finds the migrated row.

## Middleware changes

Today: `Auth()` reads the token, looks up `AdminCredential.APIKey`, and calls `c.Next()` with no user context attached.

New: `Auth()` reads the token, looks up `Token` by hash, loads the owning `User`, checks `Disabled` and `ExpiresAt`, updates `LastUsedAt`, attaches the user to the request context via `c.Set("user", user)`, and continues.

A second middleware `RequireRole(roles ...string)` wraps handlers that need a specific role. Handlers that do not call `RequireRole` are accessible to any authenticated user (read-level). Write handlers declare their required role:

```go
r.POST("/api/v1/policies",   RequireRole("admin", "operator"), h.CreatePolicy)
r.DELETE("/api/v1/policies/:id", RequireRole("admin", "operator"), h.DeletePolicy)
r.POST("/api/v1/users",      RequireRole("admin"),              h.CreateUser)
```

`read_only` users get no `RequireRole`-guarded handler. They can GET everything.

## Audit envelope

Every mutating handler calls a single helper:

```go
audit.Record(c, "policy.create", "policy:"+p.ID, nil, &p)
```

The helper pulls the user from context, captures IP and UA from the request, and writes one row. Failures to write audit events log an error but do not fail the request - auditing is observation, not a gate.

The fifteen mutating endpoints (policies, blocklists, engine toggle, users, tokens, resolvers-when-wired) each gain exactly one `audit.Record` call. No other changes to handler logic.

## Failure modes worth naming

- **Token hash collision:** SHA-256 is wide enough that a birthday collision with 10^9 tokens per customer has probability ~10^-60. Ignored.
- **Audit table grows unbounded:** same problem as `DNSQuery`. Tracked in the Deferred Items list; out of scope here. Rotation will land with Tier 3's retention work.
- **Migration runs twice:** the `users` empty-check is the idempotency guard. Tested explicitly.
- **Existing UUID token longer than 90 days:** migrated tokens get `ExpiresAt = NULL`. Expiry only applies to newly-minted tokens. Documented.
- **Concurrent user creation with duplicate email:** `UNIQUE` constraint on `email` handles it. Handler returns 409.

## API surface additions

Minimum viable. Every new endpoint is backend-only in this PR; UI wiring follows.

```
POST   /api/v1/users                 admin only
GET    /api/v1/users                 admin, operator, read_only
GET    /api/v1/users/me              any authenticated
PATCH  /api/v1/users/:id             admin (any user), self (own password/email)
DELETE /api/v1/users/:id             admin
POST   /api/v1/users/:id/disable     admin

POST   /api/v1/tokens                any authenticated (for their own user)
GET    /api/v1/tokens                any authenticated (returns only their own)
DELETE /api/v1/tokens/:id            any authenticated (only their own, or admin for any)

GET    /api/v1/audit                 admin, operator
```

`/api/v1/auth/setup` keeps its behaviour but creates an `admin`-role `User` directly. `/api/v1/auth/login` returns a newly-minted `Token` each time, not the reused `APIKey`.

## Testing plan

Per `review-agents.md` and `code-review-mistakes.md`:

- `internal/storage/repositories/auth_test.go` - user CRUD, token lifecycle, role constraint check.
- `internal/storage/repositories/audit_test.go` - record + query audit events, nullable before/after.
- `cmd/controlplane/middlewares/auth_test.go` - middleware happy path, expired token, revoked token, disabled user, missing role.
- `cmd/controlplane/handlers/users_test.go` - create, list, patch self vs patch other, role-boundary rejection.
- One integration test that brings up the Gin router with an in-memory SQLite and exercises the migration from a pre-populated `admin_credentials` row.

Every test asserts both the happy path and the negative assertion (e.g., a `read_only` user's response does NOT contain the policy-create button in the response JSON for a roles-permission endpoint). No wishy-washy `if err == nil` checks.

## Rollout order (suggested)

One commit per bullet, reviewable in 10 minutes each:

1. Add new models + GORM auto-migration + unit tests
2. Add `Token` repository + tests (replaces `AuthRepository.ValidateAPIKey` with `TokenRepository.ValidateHash`)
3. Add `AuditEvent` repository + `audit.Record` helper + tests
4. Update middleware to resolve user from token, add `RequireRole`
5. Bolt `audit.Record` calls onto every mutating handler (policies, blocklists, engine, setup, login)
6. Add user + token management handlers
7. Add audit query endpoint
8. Migration path from `AdminCredential` singleton (with test that verifies existing token still works)

Steps 1-4 land before step 5 depends on them. Steps 5-7 can land in parallel once 1-4 are in.

## What the reviewer should flag

If any of these are in the final PR, they are scope creep:

- SSO, OIDC, SAML code
- MFA / TOTP code
- Dashboard UI changes
- CLI changes beyond what's needed to test the backend
- Refactors of unrelated files
- New config knobs that don't have a concrete user story ("admin can override token expiry" - cut it)

## Open questions

Naming these up front so we agree before code:

1. **Email required or optional?** Commercial products require it. Homelab users on a single-box deploy might resist. **Proposed:** required, with `admin@hydradns.local` accepted so no external mail is needed.
2. **Token expiry default.** **Proposed:** 90 days, configurable per token at creation.
3. **First user role on `/auth/setup`.** **Proposed:** `admin`. No way to bootstrap without an admin.
4. **Do we surface `Token.Hash` anywhere?** **Proposed:** no. Plaintext is returned exactly once at creation, then only the hash is kept.
5. **Audit retention.** **Proposed:** unbounded for now, aligned with `DNSQuery`. Both rotate together when Tier 3 retention lands.
