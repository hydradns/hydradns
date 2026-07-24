import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react"

import LogsPage from "./page"
import { SidebarProvider } from "@/components/ui/sidebar"
import type { QueryLogEntry } from "@/lib/types"

// jsdom does not implement matchMedia, which SidebarProvider's mobile hook calls.
Object.defineProperty(window, "matchMedia", {
  writable: true,
  configurable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => false,
  }),
})

// The page uses <SidebarTrigger />, which reads context supplied by the
// dashboard layout in the running app; provide it here.
function renderPage() {
  return render(
    <SidebarProvider>
      <LogsPage />
    </SidebarProvider>,
  )
}

const sampleLogs: QueryLogEntry[] = [
  {
    id: 1,
    domain: "evil.com",
    client_ip: "10.0.0.5",
    action: "block",
    timestamp: "2026-07-19T10:00:00Z",
    is_suspicious: true,
    threat_score: 0.92,
    threat_reason: "Known malware C2",
  },
  {
    id: 2,
    domain: "good.com",
    client_ip: "10.0.0.6",
    action: "allow",
    timestamp: "2026-07-19T10:01:00Z",
    is_suspicious: false,
    threat_score: 0.05,
  },
  {
    // suspicious but not blocked -> "flagged" (amber highlight)
    id: 3,
    domain: "sketchy.net",
    client_ip: "10.0.0.7",
    action: "allow",
    timestamp: "2026-07-19T10:02:00Z",
    is_suspicious: true,
    threat_score: 0.55,
    threat_reason: "New domain, low reputation",
  },
]

function rowOf(text: string): HTMLElement {
  return screen.getByText(text).closest("tr") as HTMLElement
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

type FetchMock = ReturnType<typeof vi.fn>

// Routes analytics/logs -> a paginated page, policies -> a created policy.
function installFetch(total = sampleLogs.length): FetchMock {
  const fetchMock = vi.fn((input: RequestInfo | URL) => {
    const url = String(input)
    if (url.includes("/analytics/logs")) {
      return Promise.resolve(
        jsonOk({ items: sampleLogs, total, page: 1, page_size: 50 }),
      )
    }
    if (url.includes("/policies")) {
      return Promise.resolve(jsonOk({ id: "quick", name: "quick" }))
    }
    return Promise.resolve(jsonOk({}))
  })
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

function urls(fetchMock: FetchMock): string[] {
  return fetchMock.mock.calls.map((c) => String(c[0]))
}

beforeEach(() => {
  localStorage.clear()
})

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  vi.clearAllMocks()
})

describe("LogsPage", () => {
  it("renders rows returned by GET /analytics/logs, including the block reason", async () => {
    installFetch()
    renderPage()

    expect(await screen.findByText("evil.com")).toBeInTheDocument()
    expect(screen.getByText("good.com")).toBeInTheDocument()
    // Block reason column
    expect(screen.getByText("Known malware C2")).toBeInTheDocument()
  })

  it("highlights suspicious (flagged) rows amber and blocked rows red", async () => {
    installFetch()
    renderPage()

    await screen.findByText("sketchy.net")
    // suspicious-but-not-blocked -> amber
    expect(rowOf("sketchy.net").className).toContain("border-l-hydra-amber")
    // blocked -> red
    expect(rowOf("evil.com").className).toContain("border-l-hydra-red")
  })

  it("refetches server-side with the domain filter and resets to page 1", async () => {
    const fetchMock = installFetch()
    renderPage()
    await screen.findByText("evil.com")

    fireEvent.change(screen.getByLabelText("Filter by domain"), {
      target: { value: "evil" },
    })

    await waitFor(() =>
      expect(urls(fetchMock).some((u) => u.includes("domain=evil"))).toBe(true),
    )
    // page is reset to 1 on filter change
    expect(urls(fetchMock).some((u) => u.includes("domain=evil") && u.includes("page=1"))).toBe(true)
  })

  it("passes the suspicious-only flag to the endpoint", async () => {
    const fetchMock = installFetch()
    renderPage()
    await screen.findByText("evil.com")

    fireEvent.click(screen.getByLabelText("Suspicious only"))

    await waitFor(() =>
      expect(urls(fetchMock).some((u) => u.includes("suspicious=true"))).toBe(true),
    )
  })

  it("advances the page with server-side pagination", async () => {
    // total > page_size so Next is enabled
    const fetchMock = installFetch(120)
    renderPage()
    await screen.findByText("evil.com")

    fireEvent.click(screen.getByRole("button", { name: /next/i }))

    await waitFor(() =>
      expect(urls(fetchMock).some((u) => u.includes("page=2"))).toBe(true),
    )
  })

  it("one-click Allow POSTs an ALLOW policy for the row domain", async () => {
    const fetchMock = installFetch()
    renderPage()
    await screen.findByText("evil.com")

    fireEvent.click(screen.getByRole("button", { name: /allow evil\.com/i }))

    await waitFor(() => {
      const call = fetchMock.mock.calls.find((c) => String(c[0]).includes("/policies"))
      expect(call).toBeTruthy()
      const opts = call![1] as RequestInit
      expect(opts.method).toBe("POST")
      const body = JSON.parse(String(opts.body))
      expect(body.action).toBe("ALLOW")
      expect(body.domains).toContain("evil.com")
    })

    await waitFor(() =>
      expect(within(rowOf("evil.com")).getByText("Allowed")).toBeInTheDocument(),
    )
  })

  it("one-click Block POSTs a BLOCK policy for the row domain", async () => {
    const fetchMock = installFetch()
    renderPage()
    await screen.findByText("good.com")

    fireEvent.click(screen.getByRole("button", { name: /block good\.com/i }))

    await waitFor(() => {
      const call = fetchMock.mock.calls.find((c) => String(c[0]).includes("/policies"))
      expect(call).toBeTruthy()
      const body = JSON.parse(String((call![1] as RequestInit).body))
      expect(body.action).toBe("BLOCK")
      expect(body.domains).toContain("good.com")
    })
  })
})
