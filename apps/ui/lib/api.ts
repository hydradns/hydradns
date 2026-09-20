import type {
  ApiResponse,
  DashboardSummary,
  DnsEngineStatus,
  DnsMetrics,
  Resolver,
  BlocklistListData,
  Blocklist,
  CreateBlocklistRequest,
  UpdateBlocklistRequest,
  PolicyListData,
  Policy,
  CreatePolicyRequest,
  UpdatePolicyRequest,
  QueryLogEntry,
  BypassAttemptsData,
  UserListData,
  User,
  CreateUserRequest,
  UpdateUserRequest,
  Token,
  CreateTokenRequest,
  TokenSecret,
  AuditListData,
  AuditQuery,
  QueryLogFilters,
  QueryLogPage,
} from "./types"
import { getApiBaseUrl } from "./api-base"
import { toast } from "sonner"

function apiUrl(path: string): string {
  return `${getApiBaseUrl()}/api/v1${path}`
}

// DEMO_MODE_ERROR is the exact error text the control plane's DemoGuard
// middleware returns on every rejected mutating request. See
// apps/core/cmd/controlplane/middlewares/demo.go:57 (the Go string this
// must match char-for-char; there's no shared constant across the two
// languages, so a wording change on either side needs the other updated by
// hand). Matched here so the toast below only fires for that specific
// rejection, not for every 403 (e.g. a read_only user's role-based
// "forbidden" still surfaces through the normal thrown-Error / per-page
// inline-error path unchanged). Exported so tests can assert against it
// directly instead of duplicating the literal.
export const DEMO_MODE_ERROR = "demo mode: changes are disabled"

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const controller = new AbortController()
  const timeout = setTimeout(() => controller.abort(), 10000)

  const headers: Record<string, string> = { "Content-Type": "application/json" }
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("hydra_token")
    if (token) {
      headers["Authorization"] = `Bearer ${token}`
    }
  }

  try {
    const { headers: extraHeaders, ...restOptions } = options || {}
    const res = await fetch(apiUrl(path), {
      ...restOptions,
      headers: { ...headers, ...(extraHeaders as Record<string, string>) },
      signal: controller.signal,
    })

    if (res.status === 401 && typeof window !== "undefined") {
      localStorage.removeItem("hydra_token")
      document.cookie = "hydra_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT"
      window.location.href = "/login"
      throw new Error("Session expired")
    }

    const json: ApiResponse<T> = await res.json()
    if (json.status === "error") {
      // Single choke point for the demo-mode rejection: every write in the
      // app funnels through this function, so this is the one place that
      // needs to know about it (see lib/api.ts callers; none of them
      // special-case demo mode themselves). The error still throws below
      // so any page-level inline error handling keeps working unchanged;
      // the toast just makes the "why" immediately visible.
      if (res.status === 403 && json.error === DEMO_MODE_ERROR) {
        toast.error("Read-only demo. Changes are disabled. Install your own HydraDNS to try this for real.")
      }
      throw new Error(json.error || "Unknown API error")
    }
    return json.data
  } catch (e) {
    if (e instanceof DOMException && e.name === "AbortError") {
      throw new Error("Request timed out. Is the API running?")
    }
    throw e
  } finally {
    clearTimeout(timeout)
  }
}

// Dashboard
export const getDashboardSummary = () =>
  request<DashboardSummary>("/dashboard/summary")

// DNS Engine
export const getDnsEngineStatus = () =>
  request<DnsEngineStatus>("/dns/engine")

export const toggleDnsEngine = (enabled: boolean) =>
  request<{ enabled: boolean }>("/dns/engine", {
    method: "POST",
    body: JSON.stringify({ enabled }),
  })

export const getDnsMetrics = () =>
  request<DnsMetrics>("/dns/metrics")

// Resolvers are read-only from the dashboard: the control plane has no
// POST/PUT/DELETE route for /dns/resolvers (they're configured via
// configs/config.yaml). See apps/ui/app/dashboard/resolvers/page.tsx.
export const getResolvers = () =>
  request<Resolver[]>("/dns/resolvers")

// Blocklists
export const getBlocklists = () =>
  request<BlocklistListData>("/blocklists")

export const getBlocklist = (id: string) =>
  request<Blocklist>(`/blocklists/${id}`)

export const createBlocklist = (data: CreateBlocklistRequest) =>
  request<Blocklist>("/blocklists", {
    method: "POST",
    body: JSON.stringify(data),
  })

export const updateBlocklist = (id: string, data: UpdateBlocklistRequest) =>
  request<Blocklist>(`/blocklists/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  })

export const toggleBlocklist = (id: string, enabled: boolean) =>
  request<Blocklist>(`/blocklists/${id}`, {
    method: "PATCH",
    body: JSON.stringify({ enabled }),
  })

export const deleteBlocklist = (id: string) =>
  request<Record<string, unknown>>(`/blocklists/${id}`, { method: "DELETE" })

// Policies
export const getPolicies = () =>
  request<PolicyListData>("/policies")

export const getPolicy = (id: string) =>
  request<Policy>(`/policies/${id}`)

export const createPolicy = (data: CreatePolicyRequest) =>
  request<Policy>("/policies", {
    method: "POST",
    body: JSON.stringify(data),
  })

export const updatePolicy = (id: string, data: UpdatePolicyRequest) =>
  request<Policy>(`/policies/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  })

export const deletePolicy = (id: string) =>
  request<Record<string, unknown>>(`/policies/${id}`, { method: "DELETE" })

// Recent query logs: unpaginated feed backing the dashboard widgets.
export const getRecentQueryLogs = () =>
  request<QueryLogEntry[]>("/analytics/audits")

// Encrypted-DNS bypass attempts (clients trying to evade filtering via DoH/DoT/DoQ)
export const getBypassAttempts = () =>
  request<BypassAttemptsData>("/analytics/bypass")

// Users (RBAC)
export const getUsers = () =>
  request<UserListData>("/users")

export const createUser = (data: CreateUserRequest) =>
  request<User>("/users", {
    method: "POST",
    body: JSON.stringify(data),
  })

export const updateUser = (id: string, data: UpdateUserRequest) =>
  request<User>(`/users/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  })

export const deleteUser = (id: string) =>
  request<Record<string, unknown>>(`/users/${id}`, { method: "DELETE" })

// API tokens. The control plane scopes these to the caller: GET/POST
// /tokens and DELETE /tokens/:id all operate on "your own tokens" (see
// apps/core/cmd/controlplane/routes/router.go's tokens group and
// handlers/tokens.go). There is no nested /users/:id/tokens route and no
// rotate endpoint; the UI is scoped to "my tokens" to match.
export const getMyTokens = () =>
  request<Token[]>("/tokens")

export const createMyToken = (data: CreateTokenRequest) =>
  request<TokenSecret>("/tokens", {
    method: "POST",
    body: JSON.stringify(data),
  })

export const revokeMyToken = (tokenId: number) =>
  request<Record<string, unknown>>(`/tokens/${tokenId}`, {
    method: "DELETE",
  })

// Audit log
export const getAuditEvents = (params: AuditQuery = {}) => {
  const query = new URLSearchParams()
  if (params.page) query.set("page", String(params.page))
  if (params.page_size) query.set("page_size", String(params.page_size))
  if (params.actor) query.set("actor", params.actor)
  if (params.action) query.set("action", params.action)
  const qs = query.toString()
  return request<AuditListData>(`/audit${qs ? `?${qs}` : ""}`)
}

// Note: there is no GET/PATCH /settings route on the control plane today
// (apps/core/cmd/controlplane/routes/router.go has no /settings group), so
// there are no settings API functions here. See
// apps/ui/app/dashboard/settings/page.tsx, which renders the controls
// disabled with a note instead of calling a route that doesn't exist.

// Query Logs: server-side pagination + filtering via GET /analytics/logs.
// Undefined/empty filter fields are omitted so the backend applies its defaults.
export const getQueryLogs = (filters: QueryLogFilters = {}) => {
  const params = new URLSearchParams()
  if (filters.client) params.set("client", filters.client)
  if (filters.action && filters.action !== "all") params.set("action", filters.action)
  if (filters.domain) params.set("domain", filters.domain)
  if (filters.suspicious) params.set("suspicious", "true")
  if (filters.start) params.set("start", filters.start)
  if (filters.end) params.set("end", filters.end)
  params.set("page", String(filters.page ?? 1))
  params.set("page_size", String(filters.page_size ?? 50))
  return request<QueryLogPage>(`/analytics/logs?${params.toString()}`)
}

// One-click policy actions from a log row. Both create a high-priority policy
// rule scoped to the single domain, hitting the same POST /policies endpoint the
// Policies page uses.
function quickPolicySlug(domain: string) {
  return domain
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
}

function quickPolicy(domain: string, action: "ALLOW" | "BLOCK") {
  const body: CreatePolicyRequest = {
    id: `quick-${action.toLowerCase()}-${quickPolicySlug(domain)}`,
    name: `Quick ${action.toLowerCase()} ${domain}`,
    action,
    domains: [domain],
    priority: 200,
  }
  return request<Policy>("/policies", {
    method: "POST",
    body: JSON.stringify(body),
  })
}

export const allowDomain = (domain: string) => quickPolicy(domain, "ALLOW")

export const blockDomain = (domain: string) => quickPolicy(domain, "BLOCK")
