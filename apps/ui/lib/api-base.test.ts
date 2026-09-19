import { afterEach, describe, expect, it, vi } from "vitest"

import { getApiBaseUrl } from "./api-base"

const ORIGINAL_LOCATION = window.location

function stubLocation(href: string) {
  const { protocol, hostname } = new URL(href)
  Object.defineProperty(window, "location", {
    value: { ...ORIGINAL_LOCATION, protocol, hostname, href },
    writable: true,
    configurable: true,
  })
}

function restoreLocation() {
  Object.defineProperty(window, "location", {
    value: ORIGINAL_LOCATION,
    writable: true,
    configurable: true,
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
  restoreLocation()
  vi.unstubAllEnvs()
})

describe("getApiBaseUrl (browser)", () => {
  it("derives from location when NEXT_PUBLIC_API_URL is unset and the page is a LAN IP", () => {
    vi.stubEnv("NEXT_PUBLIC_API_URL", "")
    stubLocation("http://192.168.1.53:3000/dashboard")

    expect(getApiBaseUrl()).toBe("http://192.168.1.53:8080")
  })

  it("derives from location when NEXT_PUBLIC_API_URL is unset and the page is localhost", () => {
    vi.stubEnv("NEXT_PUBLIC_API_URL", "")
    stubLocation("http://localhost:3000/dashboard")

    expect(getApiBaseUrl()).toBe("http://localhost:8080")
  })

  it("treats a baked legacy-default env value as unset when the page is on a LAN IP", () => {
    vi.stubEnv("NEXT_PUBLIC_API_URL", "http://localhost:8080")
    stubLocation("http://192.168.1.53:3000/dashboard")

    expect(getApiBaseUrl()).toBe("http://192.168.1.53:8080")
  })

  it("keeps the baked legacy-default env value when the page itself is on localhost", () => {
    vi.stubEnv("NEXT_PUBLIC_API_URL", "http://localhost:8080")
    stubLocation("http://localhost:3000/dashboard")

    expect(getApiBaseUrl()).toBe("http://localhost:8080")
  })

  it("always prefers an explicit, non-default baked env value over location", () => {
    vi.stubEnv("NEXT_PUBLIC_API_URL", "https://api.example.com")
    stubLocation("http://192.168.1.53:3000/dashboard")

    expect(getApiBaseUrl()).toBe("https://api.example.com")
  })

  it("wraps IPv6 literal hostnames in brackets when deriving from location", () => {
    vi.stubEnv("NEXT_PUBLIC_API_URL", "")
    stubLocation("http://[fe80::1]:3000/dashboard")

    expect(getApiBaseUrl()).toBe("http://[fe80::1]:8080")
  })

  it("treats the IPv6 loopback literal as localhost for the baked-default check", () => {
    vi.stubEnv("NEXT_PUBLIC_API_URL", "http://localhost:8080")
    stubLocation("http://[::1]:3000/dashboard")

    expect(getApiBaseUrl()).toBe("http://localhost:8080")
  })

  it("derives an https API URL for a page served over https (same-protocol; documented limitation: the API has no TLS today, so this only works behind a TLS-terminating reverse proxy that also proxies the API, otherwise it's mixed content)", () => {
    vi.stubEnv("NEXT_PUBLIC_API_URL", "")
    stubLocation("https://dashboard.example.com:3000/dashboard")

    expect(getApiBaseUrl()).toBe("https://dashboard.example.com:8080")
  })
})

describe("getApiBaseUrl (server, no window)", () => {
  it("falls back to HYDRA_API_INTERNAL_URL when set", () => {
    vi.stubGlobal("window", undefined)
    vi.stubEnv("HYDRA_API_INTERNAL_URL", "http://core-internal:9090")

    expect(getApiBaseUrl()).toBe("http://core-internal:9090")
  })

  it("falls back to the legacy default when HYDRA_API_INTERNAL_URL is unset", () => {
    vi.stubGlobal("window", undefined)
    vi.stubEnv("HYDRA_API_INTERNAL_URL", "")

    expect(getApiBaseUrl()).toBe("http://localhost:8080")
  })
})
