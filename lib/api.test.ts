import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import {
  getBypassAttempts,
  getDashboardSummary,
  getUsers,
  createUser,
  updateUser,
  deleteUser,
  getUserTokens,
  createUserToken,
  rotateUserToken,
  revokeUserToken,
  getAuditEvents,
  allowDomain,
  blockDomain,
  getQueryLogs,
} from "@/lib/api"
import type { ApiResponse, BypassAttemptsData, DashboardSummary } from "@/lib/types"

const BASE = "http://localhost:8080/api/v1"

/** Stub global.fetch with a single JSON response envelope. */
function stubFetch<T>(body: ApiResponse<T>, status = 200) {
  const fetchMock = vi.fn().mockResolvedValue({
    status,
    json: async () => body,
  })
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

// Builds a fake fetch Response whose json() yields the success envelope.
function okResponse<T>(data: T) {
  return {
    status: 200,
    json: async () => ({ status: "success", data, error: null }),
  } as unknown as Response
}

function errorResponse(message: string) {
  return {
    status: 200,
    json: async () => ({ status: "error", data: null, error: message }),
  } as unknown as Response
}

function fetchMock() {
  return vi.fn<typeof fetch>()
}

// Builds a fake fetch Response for the query-log / quick-action tests below,
// which only assert on the request the client makes, not envelope shape.
function jsonOk(data: unknown) {
  return {
    ok: true,
    status: 200,
    json: async () => ({ status: "success", data, error: null }),
  } as unknown as Response
}

function stubFetchData(data: unknown) {
  const fetchMock = vi.fn(() => Promise.resolve(jsonOk(data)))
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

function lastUrl(fetchMock: ReturnType<typeof vi.fn>) {
  return String(fetchMock.mock.calls.at(-1)?.[0])
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe("getBypassAttempts", () => {
  it("unwraps the data envelope and hits /analytics/bypass", async () => {
    const data: BypassAttemptsData = {
      total_attempts: 12,
      unique_clients: 2,
      attempts: [
        {
          client_ip: "10.0.0.5",
          client_name: "kids-tablet",
          protocol: "doh",
          target: "cloudflare-dns.com",
          attempts: 9,
          last_attempt: "2026-07-19T10:00:00Z",
          blocked: true,
        },
      ],
    }
    const fetchMock = stubFetch<BypassAttemptsData>({ status: "success", data, error: null })

    const result = await getBypassAttempts()

    expect(result).toEqual(data)
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const calledUrl = fetchMock.mock.calls[0][0] as string
    expect(calledUrl).toContain("/api/v1/analytics/bypass")
  })

  it("throws the API error message when the envelope reports an error", async () => {
    stubFetch<BypassAttemptsData>(
      { status: "error", data: null as unknown as BypassAttemptsData, error: "bypass telemetry offline" },
    )

    await expect(getBypassAttempts()).rejects.toThrow("bypass telemetry offline")
  })
})

describe("getDashboardSummary", () => {
  it("returns the parsed summary payload", async () => {
    const data: DashboardSummary = {
      total_queries: 100,
      blocked_queries: 10,
      allowed_queries: 90,
      redirected_queries: 0,
      block_rate_percent: 10,
    }
    const fetchMock = stubFetch<DashboardSummary>({ status: "success", data, error: null })

    const result = await getDashboardSummary()

    expect(result).toEqual(data)
    expect(fetchMock.mock.calls[0][0]).toContain("/api/v1/dashboard/summary")
  })
})

describe("api RBAC methods", () => {
  beforeEach(() => {
    localStorage.clear()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it("getUsers issues a GET to /users and unwraps data", async () => {
    const payload = { total_users: 1, users: [{ id: "u1", username: "root", role: "admin" }] }
    const spy = fetchMock().mockResolvedValue(okResponse(payload))
    vi.stubGlobal("fetch", spy)

    const result = await getUsers()

    expect(spy).toHaveBeenCalledTimes(1)
    expect(spy.mock.calls[0][0]).toBe(`${BASE}/users`)
    // No explicit method means a GET.
    expect(spy.mock.calls[0][1]?.method).toBeUndefined()
    expect(result).toEqual(payload)
  })

  it("createUser POSTs the JSON body with a JSON content-type", async () => {
    const created = { id: "u2", username: "jane", role: "operator" }
    const spy = fetchMock().mockResolvedValue(okResponse(created))
    vi.stubGlobal("fetch", spy)

    await createUser({ username: "jane", password: "s3cret", role: "operator" })

    const [url, init] = spy.mock.calls[0]
    expect(url).toBe(`${BASE}/users`)
    expect(init?.method).toBe("POST")
    expect(JSON.parse(init?.body as string)).toEqual({
      username: "jane",
      password: "s3cret",
      role: "operator",
    })
    expect((init?.headers as Record<string, string>)["Content-Type"]).toBe("application/json")
  })

  it("updateUser PATCHes /users/:id with the partial body", async () => {
    const spy = fetchMock().mockResolvedValue(okResponse({ id: "u2", username: "jane", role: "admin" }))
    vi.stubGlobal("fetch", spy)

    await updateUser("u2", { role: "admin", enabled: false })

    const [url, init] = spy.mock.calls[0]
    expect(url).toBe(`${BASE}/users/u2`)
    expect(init?.method).toBe("PATCH")
    expect(JSON.parse(init?.body as string)).toEqual({ role: "admin", enabled: false })
  })

  it("deleteUser sends a DELETE to /users/:id", async () => {
    const spy = fetchMock().mockResolvedValue(okResponse({}))
    vi.stubGlobal("fetch", spy)

    await deleteUser("u9")

    expect(spy.mock.calls[0][0]).toBe(`${BASE}/users/u9`)
    expect(spy.mock.calls[0][1]?.method).toBe("DELETE")
  })

  it("getUserTokens GETs the per-user token collection", async () => {
    const tokens = [{ id: "t1", name: "ci", prefix: "hyd_", role: "operator", revoked: false }]
    const spy = fetchMock().mockResolvedValue(okResponse(tokens))
    vi.stubGlobal("fetch", spy)

    const result = await getUserTokens("u2")

    expect(spy.mock.calls[0][0]).toBe(`${BASE}/users/u2/tokens`)
    expect(result).toEqual(tokens)
  })

  it("createUserToken POSTs the token request and returns the one-time secret", async () => {
    const secret = { token: { id: "t2", name: "ci", prefix: "hyd_", role: "operator", revoked: false }, secret: "hyd_live_abc" }
    const spy = fetchMock().mockResolvedValue(okResponse(secret))
    vi.stubGlobal("fetch", spy)

    const result = await createUserToken("u2", { name: "ci", expires_in_days: 90 })

    const [url, init] = spy.mock.calls[0]
    expect(url).toBe(`${BASE}/users/u2/tokens`)
    expect(init?.method).toBe("POST")
    expect(JSON.parse(init?.body as string)).toEqual({ name: "ci", expires_in_days: 90 })
    expect(result.secret).toBe("hyd_live_abc")
  })

  it("rotateUserToken POSTs to the rotate sub-resource", async () => {
    const spy = fetchMock().mockResolvedValue(
      okResponse({ token: { id: "t2", name: "ci", prefix: "hyd_", role: "operator", revoked: false }, secret: "hyd_live_new" }),
    )
    vi.stubGlobal("fetch", spy)

    const result = await rotateUserToken("u2", "t2")

    expect(spy.mock.calls[0][0]).toBe(`${BASE}/users/u2/tokens/t2/rotate`)
    expect(spy.mock.calls[0][1]?.method).toBe("POST")
    expect(result.secret).toBe("hyd_live_new")
  })

  it("revokeUserToken DELETEs the specific token", async () => {
    const spy = fetchMock().mockResolvedValue(okResponse({}))
    vi.stubGlobal("fetch", spy)

    await revokeUserToken("u2", "t2")

    expect(spy.mock.calls[0][0]).toBe(`${BASE}/users/u2/tokens/t2`)
    expect(spy.mock.calls[0][1]?.method).toBe("DELETE")
  })

  it("getAuditEvents encodes pagination and filters into the query string", async () => {
    const spy = fetchMock().mockResolvedValue(
      okResponse({ total: 0, page: 2, page_size: 25, events: [] }),
    )
    vi.stubGlobal("fetch", spy)

    await getAuditEvents({ page: 2, page_size: 25, actor: "root", action: "user.create" })

    const url = spy.mock.calls[0][0] as string
    expect(url.startsWith(`${BASE}/audit?`)).toBe(true)
    const qs = new URLSearchParams(url.split("?")[1])
    expect(qs.get("page")).toBe("2")
    expect(qs.get("page_size")).toBe("25")
    expect(qs.get("actor")).toBe("root")
    expect(qs.get("action")).toBe("user.create")
  })

  it("getAuditEvents omits the query string when no params are given", async () => {
    const spy = fetchMock().mockResolvedValue(
      okResponse({ total: 0, page: 1, page_size: 25, events: [] }),
    )
    vi.stubGlobal("fetch", spy)

    await getAuditEvents()

    expect(spy.mock.calls[0][0]).toBe(`${BASE}/audit`)
  })

  it("attaches a bearer token from localStorage when present", async () => {
    localStorage.setItem("hydra_token", "session-jwt")
    const spy = fetchMock().mockResolvedValue(okResponse({ total_users: 0, users: [] }))
    vi.stubGlobal("fetch", spy)

    await getUsers()

    const headers = spy.mock.calls[0][1]?.headers as Record<string, string>
    expect(headers["Authorization"]).toBe("Bearer session-jwt")
  })

  it("throws with the server message when the envelope is an error", async () => {
    const spy = fetchMock().mockResolvedValue(errorResponse("forbidden: admin required"))
    vi.stubGlobal("fetch", spy)

    await expect(createUser({ username: "x", password: "y", role: "admin" })).rejects.toThrow(
      "forbidden: admin required",
    )
  })
})

describe("getQueryLogs", () => {
  it("hits /analytics/logs with default pagination when no filters are given", async () => {
    const fetchMock = stubFetchData({ items: [], total: 0, page: 1, page_size: 50 })
    await getQueryLogs()
    const url = lastUrl(fetchMock)
    expect(url).toContain("/api/v1/analytics/logs")
    expect(url).toContain("page=1")
    expect(url).toContain("page_size=50")
  })

  it("serializes provided filters and omits empty / action=all", async () => {
    const fetchMock = stubFetchData({ items: [], total: 0, page: 2, page_size: 50 })
    await getQueryLogs({
      domain: "evil",
      client: "10.0.0.5",
      action: "block",
      suspicious: true,
      page: 2,
    })
    const url = lastUrl(fetchMock)
    expect(url).toContain("domain=evil")
    expect(url).toContain("client=10.0.0.5")
    expect(url).toContain("action=block")
    expect(url).toContain("suspicious=true")
    expect(url).toContain("page=2")
  })

  it("does not emit action=all or a false suspicious flag", async () => {
    const fetchMock = stubFetchData({ items: [], total: 0, page: 1, page_size: 50 })
    await getQueryLogs({ action: "all", suspicious: false })
    const url = lastUrl(fetchMock)
    expect(url).not.toContain("action=all")
    expect(url).not.toContain("suspicious")
  })
})

describe("allowDomain / blockDomain", () => {
  it("allowDomain POSTs an ALLOW policy scoped to the domain", async () => {
    const fetchMock = stubFetchData({ id: "quick-allow-evil-com" })
    await allowDomain("evil.com")
    const [url, opts] = fetchMock.mock.calls.at(-1) as unknown as [string, RequestInit]
    expect(String(url)).toContain("/api/v1/policies")
    expect(opts.method).toBe("POST")
    const body = JSON.parse(String(opts.body))
    expect(body.action).toBe("ALLOW")
    expect(body.domains).toEqual(["evil.com"])
    expect(body.id).toBe("quick-allow-evil-com")
  })

  it("blockDomain POSTs a BLOCK policy scoped to the domain", async () => {
    const fetchMock = stubFetchData({ id: "quick-block-evil-com" })
    await blockDomain("evil.com")
    const [, opts] = fetchMock.mock.calls.at(-1) as unknown as [string, RequestInit]
    const body = JSON.parse(String(opts.body))
    expect(body.action).toBe("BLOCK")
    expect(body.domains).toEqual(["evil.com"])
  })
})
