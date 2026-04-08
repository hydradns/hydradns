import type {
  ApiResponse,
  DashboardSummary,
  DnsEngineStatus,
  DnsMetrics,
  Resolver,
  BlocklistListData,
  Blocklist,
  CreateBlocklistRequest,
  PolicyListData,
  Policy,
  CreatePolicyRequest,
  QueryLogEntry,
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

export const deletePolicy = (id: string) =>
  request<Record<string, unknown>>(`/policies/${id}`, { method: "DELETE" })

// Query Logs
export const getQueryLogs = () =>
  request<QueryLogEntry[]>("/analytics/audits")
