import { beforeEach, describe, expect, it, vi } from "vitest"
import { fireEvent, render, screen, waitFor } from "@testing-library/react"

import AuditPage from "./page"
import { SidebarProvider } from "@/components/ui/sidebar"
import { getAuditEvents } from "@/lib/api"
import type { AuditListData } from "@/lib/types"

// The page renders a <SidebarTrigger>, which needs the provider context.
function renderPage() {
  return render(
    <SidebarProvider>
      <AuditPage />
    </SidebarProvider>,
  )
}

vi.mock("@/lib/api", () => ({
  getAuditEvents: vi.fn(),
}))

const page1: AuditListData = {
  total: 30,
  page: 1,
  page_size: 25,
  events: [
    {
      id: "a1",
      actor: "root",
      action: "user.create",
      target: "jane",
      target_type: "user",
      before: null,
      after: { role: "operator" },
      ip: "10.0.0.5",
      timestamp: "2026-07-19T10:00:00Z",
    },
    {
      id: "a2",
      actor: "root",
      action: "token.revoke",
      target: "ci-token",
      target_type: "token",
      before: { revoked: false },
      after: { revoked: true },
      ip: "10.0.0.5",
      timestamp: "2026-07-19T11:00:00Z",
    },
  ],
}

describe("AuditPage", () => {
  beforeEach(() => {
    vi.mocked(getAuditEvents).mockReset()
    vi.mocked(getAuditEvents).mockResolvedValue(page1)
  })

  it("requests the first page on mount and renders the events", async () => {
    renderPage()

    expect(await screen.findByText("user.create")).toBeInTheDocument()
    expect(screen.getByText("token.revoke")).toBeInTheDocument()

    // Initial load asks for page 1 with the fixed page size.
    expect(getAuditEvents).toHaveBeenCalledWith(
      expect.objectContaining({ page: 1, page_size: 25 }),
    )
  })

  it("expands a row to reveal the before/after snapshot", async () => {
    renderPage()

    const row = await screen.findByText("token.revoke")
    // Details are collapsed until the row is clicked.
    expect(screen.queryByText("Before")).not.toBeInTheDocument()

    fireEvent.click(row)

    expect(await screen.findByText("Before")).toBeInTheDocument()
    expect(screen.getByText("After")).toBeInTheDocument()
  })

  it("advances the page and refetches with the new page number", async () => {
    renderPage()

    await screen.findByText("user.create")

    fireEvent.click(screen.getByRole("button", { name: /next/i }))

    await waitFor(() =>
      expect(getAuditEvents).toHaveBeenLastCalledWith(
        expect.objectContaining({ page: 2, page_size: 25 }),
      ),
    )
  })
})
