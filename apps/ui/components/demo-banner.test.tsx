import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"

import { DemoBanner } from "@/components/demo-banner"

function stubAuthStatus(data: { setup_complete: boolean; demo_mode: boolean }) {
  const fetchMock = vi.fn().mockResolvedValue({
    status: 200,
    json: async () => ({ status: "success", data, error: null }),
  })
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe("DemoBanner", () => {
  it("renders nothing when demo_mode is false", async () => {
    stubAuthStatus({ setup_complete: true, demo_mode: false })

    render(<DemoBanner />)

    // Give the effect's fetch a tick to resolve, then assert nothing rendered.
    await waitFor(() => expect(fetch).toHaveBeenCalled())
    expect(screen.queryByText(/Read-only demo/i)).not.toBeInTheDocument()
  })

  it("renders the banner text and install link when demo_mode is true", async () => {
    stubAuthStatus({ setup_complete: true, demo_mode: true })

    render(<DemoBanner />)

    expect(await screen.findByText(/Read-only demo\. Changes are disabled\./i)).toBeInTheDocument()
    const link = screen.getByRole("link", { name: /install your own/i })
    expect(link).toHaveAttribute("href", "https://github.com/hydradns/hydradns")
  })

  it("renders nothing while the auth-status check has not resolved yet", () => {
    vi.stubGlobal("fetch", vi.fn(() => new Promise(() => {}))) // never resolves

    render(<DemoBanner />)

    expect(screen.queryByText(/Read-only demo/i)).not.toBeInTheDocument()
  })
})
