import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { cleanup, render, screen, waitFor } from "@testing-library/react"

import DashboardPage from "./page"
import { SidebarProvider } from "@/components/ui/sidebar"

// The page uses <SidebarTrigger />, which reads context supplied by the
// dashboard layout in the running app; provide it here (same pattern as
// app/dashboard/logs/page.test.tsx).
function renderPage() {
  return render(
    <SidebarProvider>
      <DashboardPage />
    </SidebarProvider>,
  )
}

// Minimal Response-like object shaped for lib/api's `request` helper, which
// reads `res.status` and `res.json()` and expects the { status, data, error }
// envelope.
function jsonOk(data: unknown) {
  return {
    ok: true,
    status: 200,
    json: async () => ({ status: "success", data, error: null }),
  } as unknown as Response
}

function installFetch() {
  const fetchMock = vi.fn((input: RequestInfo | URL) => {
    const url = String(input)
    if (url.includes("/dashboard/summary")) {
      return Promise.resolve(
        jsonOk({
          total_queries: 100,
          blocked_queries: 10,
          allowed_queries: 90,
          redirected_queries: 0,
          block_rate_percent: 10,
        }),
      )
    }
    if (url.includes("/analytics/bypass")) {
      return Promise.resolve(
        jsonOk({
          attempts: [
            {
              client_ip: "10.0.0.9",
              protocol: "doh",
              target: "1.1.1.1",
              attempts: 3,
              last_attempt: "2026-07-19T10:00:00Z",
              blocked: true,
            },
          ],
          total_attempts: 3,
          unique_clients: 1,
        }),
      )
    }
    if (url.includes("/analytics/audits")) {
      return Promise.resolve(jsonOk([]))
    }
    return Promise.resolve(jsonOk({}))
  })
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

beforeEach(() => {
  localStorage.clear()
})

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  vi.unstubAllEnvs()
  vi.clearAllMocks()
})

// The bypass-attempts panel was cancelled as a shipped dashboard feature:
// surfacing a filtering limitation to non-technical buyers is an
// anti-feature (mitigations should be invisible). It stays in the codebase
// behind NEXT_PUBLIC_SHOW_BYPASS_PANEL for technical/internal deployments,
// but must be hidden unless that flag is explicitly turned on.
describe("DashboardPage bypass panel visibility", () => {
  it("hides the bypass panel by default (flag unset)", async () => {
    vi.stubEnv("NEXT_PUBLIC_SHOW_BYPASS_PANEL", undefined as unknown as string)
    installFetch()
    renderPage()

    // Wait for the page's other data-dependent content to settle so we're
    // not just catching the panel before its own fetch resolves.
    await screen.findByText("Query Activity")
    await waitFor(() => {
      expect(
        screen.queryByLabelText("Encrypted-DNS bypass attempts"),
      ).not.toBeInTheDocument()
    })
    expect(screen.queryByText("Encrypted-DNS Bypass Attempts")).not.toBeInTheDocument()
  })

  it("shows the bypass panel when NEXT_PUBLIC_SHOW_BYPASS_PANEL=true", async () => {
    vi.stubEnv("NEXT_PUBLIC_SHOW_BYPASS_PANEL", "true")
    installFetch()
    renderPage()

    expect(
      await screen.findByLabelText("Encrypted-DNS bypass attempts"),
    ).toBeInTheDocument()
    expect(screen.getByText("Encrypted-DNS Bypass Attempts")).toBeInTheDocument()
  })
})
