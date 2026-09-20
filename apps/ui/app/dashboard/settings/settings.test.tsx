import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, render, screen } from "@testing-library/react"

import SettingsPage from "./page"
import { SidebarProvider } from "@/components/ui/sidebar"

// There is no GET/PATCH /settings route on the control plane (see
// lib/api.ts), so the page no longer calls fetch at all — it renders a
// disabled preview with a "not available yet" note instead of pretending
// changes save. This test asserts both halves of that fix: no fetch call,
// and controls that are visibly disabled.
function renderPage() {
  return render(
    <SidebarProvider>
      <SettingsPage />
    </SidebarProvider>,
  )
}

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe("SettingsPage", () => {
  it("renders a 'not available yet' note and never calls fetch", () => {
    const fetchMock = vi.fn()
    vi.stubGlobal("fetch", fetchMock)

    renderPage()

    expect(screen.getByText(/not available yet/i)).toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it("disables every settings control", () => {
    renderPage()

    const retention = screen.getByLabelText(/query log retention/i)
    expect(retention).toBeDisabled()

    const engineToggle = screen.getByRole("switch", { name: /dns engine/i })
    expect(engineToggle).toBeDisabled()

    // No Save button, since there's nothing to save to.
    expect(screen.queryByRole("button", { name: /save/i })).not.toBeInTheDocument()
  })
})
