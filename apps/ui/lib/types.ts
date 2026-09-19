// API response envelope
export interface ApiResponse<T> {
  status: "success" | "error"
  data: T
  error: string | null
}

// Dashboard
export interface DashboardSummary {
  total_queries: number
  blocked_queries: number
  allowed_queries: number
  redirected_queries: number
  block_rate_percent: number
}

// DNS Engine
export interface DnsEngineStatus {
  enabled: boolean
  accepting_queries: boolean
  last_error: string
}

export interface DnsMetrics {
  window_seconds: number
  queries: {
    total: number
    errors: number
    error_rate: number
  }
  latency_ms: {
    p50: number
    p95: number
    p99: number
  }
  grade: "excellent" | "good" | "degraded" | "bad" | "unknown"
}

export interface Resolver {
  id: string
  name: string
  address: string
  protocol: string
}

export interface CreateResolverRequest {
  id: string
  name: string
  address: string
  protocol: string
}

export type UpdateResolverRequest = Partial<Omit<CreateResolverRequest, "id">>

// Blocklists
export interface Blocklist {
  id: string
  name: string
  url: string
  format: string
  category: string
  domains_count: number
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface BlocklistListData {
  total_blocklists: number
  total_domains: number
  active_lists: Blocklist[]
}

export interface CreateBlocklistRequest {
  id: string
  name: string
  url: string
  format: string
  category?: string
}

export type UpdateBlocklistRequest = Partial<Omit<CreateBlocklistRequest, "id">> & {
  enabled?: boolean
}

// Curated categories (bundled threat/content groups the operator can toggle)
export interface Category {
  id: string
  name: string
  description: string
  enabled: boolean
  domains_count: number
}

// Policies
export interface Policy {
  id: string
  name: string
  description: string
  category: string
  action: string
  redirect_ip?: string
  domains: string[]
  priority: number
  enabled: boolean
  // Optional scheduling + client scoping (backend CRUD adds these)
  schedule?: string
  client_scope?: string
}

export interface PolicyListData {
  total_policies: number
  active_policies: number
  inactive_policies: number
  list: Policy[]
}

export interface CreatePolicyRequest {
  id: string
  name: string
  description?: string
  category?: string
  action: string
  redirect_ip?: string
  domains: string[]
  priority?: number
  schedule?: string
  client_scope?: string
}

export type UpdatePolicyRequest = Partial<Omit<CreatePolicyRequest, "id">> & {
  enabled?: boolean
}

// Query Logs
export interface QueryLogEntry {
  id: number
  domain: string
  client_ip: string
  action: string
  timestamp: string
  is_suspicious: boolean
  threat_score: number
  detection_method?: string
  threat_reason?: string
}

// Filters accepted by GET /analytics/logs (server-side pagination + filtering).
// Empty/undefined fields are omitted from the query string.
export interface QueryLogFilters {
  client?: string
  action?: string
  domain?: string
  suspicious?: boolean
  start?: string // ISO-8601 lower bound (inclusive)
  end?: string // ISO-8601 upper bound (inclusive)
  page?: number
  page_size?: number
}

// Paginated envelope returned by GET /analytics/logs.
export interface QueryLogPage {
  items: QueryLogEntry[]
  total: number
  page: number
  page_size: number
}

// Analytics
export interface AnalyticsSummary {
  total_queries: number
  blocked_queries: number
  allowed_queries: number
  block_rate_percent: number
}

// Encrypted-DNS bypass attempts (DoH / DoT / DoQ)
// A "bypass attempt" is a client trying to reach an encrypted-DNS resolver
// directly so its queries never pass through HydraDNS filtering.
export type BypassProtocol = "doh" | "dot" | "doq"

export interface BypassAttempt {
  client_ip: string
  client_name?: string
  protocol: BypassProtocol
  // Resolver the client tried to reach, e.g. "cloudflare-dns.com".
  target: string
  attempts: number
  // ISO timestamp of the most recent attempt.
  last_attempt: string
  // Whether HydraDNS successfully blocked the bypass.
  blocked: boolean
}

export interface BypassAttemptsData {
  total_attempts: number
  unique_clients: number
  attempts: BypassAttempt[]
}

// RBAC — Users
export type UserRole = "admin" | "operator" | "read_only"

export interface User {
  id: string
  username: string
  role: UserRole
  enabled: boolean
  created_at: string
  updated_at: string
  last_login_at?: string
}

export interface UserListData {
  total_users: number
  users: User[]
}

export interface CreateUserRequest {
  username: string
  password: string
  role: UserRole
}

export interface UpdateUserRequest {
  role?: UserRole
  password?: string
  enabled?: boolean
}

// RBAC — API tokens (scoped to a user)
export interface Token {
  id: string
  name: string
  prefix: string
  role: UserRole
  revoked: boolean
  created_at: string
  last_used_at?: string
  expires_at?: string
}

export interface CreateTokenRequest {
  name: string
  expires_in_days?: number
}

// The plaintext secret is only ever returned once, on create/rotate.
export interface TokenSecret {
  token: Token
  secret: string
}

// RBAC — Audit log
export interface AuditEvent {
  id: string
  actor: string
  action: string
  target: string
  target_type?: string
  before: Record<string, unknown> | null
  after: Record<string, unknown> | null
  ip: string
  timestamp: string
}

export interface AuditListData {
  total: number
  page: number
  page_size: number
  events: AuditEvent[]
}

export interface AuditQuery {
  page?: number
  page_size?: number
  actor?: string
  action?: string
}

// Settings (control-plane engine configuration)
export interface Settings {
  engine_enabled: boolean
  block_page_enabled: boolean
  cache_enabled: boolean
  cache_ttl_seconds: number
  upstream_timeout_ms: number
  log_retention_days: number
}

export type UpdateSettingsRequest = Partial<Settings>
