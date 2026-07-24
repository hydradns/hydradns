import type {
  ApiResponse,
  DashboardSummary,
  DnsEngineStatus,
  DnsMetrics,
  Resolver,
  CreateResolverRequest,
  UpdateResolverRequest,
  BlocklistListData,
  Blocklist,
  CreateBlocklistRequest,
  UpdateBlocklistRequest,
  Category,
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
  Settings,
  UpdateSettingsRequest,
  QueryLogFilters,
  QueryLogPage,
} from "./types"

const BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"
const API = `${BASE_URL}/api/v1`

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
    const res = await fetch(`${API}${path}`, {
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
      throw new Error(json.error || "Unknown API error")
    }
    return json.data
  } catch (e) {
    if (e instanceof DOMException && e.name === "AbortError") {
      throw new Error("Request timed out — is the API running?")
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

export const getResolvers = () =>
  request<Resolver[]>("/dns/resolvers")

export const createResolver = (data: CreateResolverRequest) =>
  request<Resolver>("/dns/resolvers", {
    method: "POST",
    body: JSON.stringify(data),
  })

export const updateResolver = (id: string, data: UpdateResolverRequest) =>
  request<Resolver>(`/dns/resolvers/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  })

export const deleteResolver = (id: string) =>
  request<Record<string, unknown>>(`/dns/resolvers/${id}`, { method: "DELETE" })

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

// Curated categories
export const getCategories = () =>
  request<Category[]>("/blocklists/categories")

export const toggleCategory = (id: string, enabled: boolean) =>
  request<Category>(`/blocklists/categories/${id}`, {
    method: "PATCH",
    body: JSON.stringify({ enabled }),
  })

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

// Recent query logs — unpaginated feed backing the dashboard widgets.
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

// Per-user API tokens
export const getUserTokens = (userId: string) =>
  request<Token[]>(`/users/${userId}/tokens`)

export const createUserToken = (userId: string, data: CreateTokenRequest) =>
  request<TokenSecret>(`/users/${userId}/tokens`, {
    method: "POST",
    body: JSON.stringify(data),
  })

export const rotateUserToken = (userId: string, tokenId: string) =>
  request<TokenSecret>(`/users/${userId}/tokens/${tokenId}/rotate`, {
    method: "POST",
  })

export const revokeUserToken = (userId: string, tokenId: string) =>
  request<Record<string, unknown>>(`/users/${userId}/tokens/${tokenId}`, {
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

// Settings
export const getSettings = () =>
  request<Settings>("/settings")

export const updateSettings = (data: UpdateSettingsRequest) =>
  request<Settings>("/settings", {
    method: "PATCH",
    body: JSON.stringify(data),
  })

// Query Logs — server-side pagination + filtering via GET /analytics/logs.
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
