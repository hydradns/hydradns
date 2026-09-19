import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"

import SettingsPage from "./page"
import { SidebarProvider } from "@/components/ui/sidebar"
import type { Settings } from "@/lib/types"

const SAMPLE: Settings = {
  engine_enabled: true,
  block_page_enabled: false,
  cache_enabled: true,
  cache_ttl_seconds: 300,
  upstream_timeout_ms: 2000,
  log_retention_days: 30,
}

function mockResponse(data: unknown) {
  return {
    status: 200,
    json: async () => ({ status: "success", data, error: null }),
  } as unknown as Response
}

describe("SettingsPage", () => {
  beforeEach(() => {
    // Mock fetch: GET /settings returns SAMPLE, PATCH echoes the body back.
    global.fetch = vi.fn().mockResolvedValue(mockResponse(SAMPLE))
  })

  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
  })

  it("submits the form and calls the update settings API (PATCH /settings)", async () => {
    render(
      <SidebarProvider>
        <SettingsPage />
      </SidebarProvider>,
    )

    const saveButton = await screen.findByRole("button", { name: /save changes/i })
    // Enabled only once initial GET settles.
    await waitFor(() => expect(saveButton).not.toBeDisabled())

    // Edit a field, then submit.
    const retention = screen.getByLabelText(/query log retention/i)
    fireEvent.change(retention, { target: { value: "14" } })
    fireEvent.click(saveButton)

    // Confirm a PATCH to /settings was issued via fetch.
    await waitFor(() => {
      const patchCall = (global.fetch as ReturnType<typeof vi.fn>).mock.calls.find(
        ([, opts]) => (opts as RequestInit | undefined)?.method === "PATCH",
      )
      expect(patchCall).toBeTruthy()
      expect(String(patchCall![0])).toContain("/settings")
      const body = JSON.parse((patchCall![1] as RequestInit).body as string)
      expect(body.log_retention_days).toBe(14)
    })

    // Success confirmation renders.
    expect(await screen.findByText(/settings saved/i)).toBeInTheDocument()
  })
})
