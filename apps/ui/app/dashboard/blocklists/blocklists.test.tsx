import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"

import BlocklistsPage from "./page"
import { SidebarProvider } from "@/components/ui/sidebar"
import * as api from "@/lib/api"
import type { Blocklist, BlocklistListData } from "@/lib/types"

// The curated-categories feature (GET /blocklists/categories,
// PATCH /blocklists/categories/:id) had no backend route and 404'd on every
// page load. It's been removed from the UI entirely: no getCategories /
// toggleCategory exports exist anymore, so they aren't mocked here.
vi.mock("@/lib/api", () => ({
  getBlocklists: vi.fn(),
  createBlocklist: vi.fn(),
  deleteBlocklist: vi.fn(),
  toggleBlocklist: vi.fn(),
}))

const SOURCE: Blocklist = {
  id: "steven-black",
  name: "StevenBlack Hosts",
  url: "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
  format: "hosts",
  category: "",
  domains_count: 150000,
  enabled: true,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
}

const BLOCKLISTS: BlocklistListData = {
  total_blocklists: 1,
  total_domains: 150000,
  active_lists: [SOURCE],
}

describe("BlocklistsPage", () => {
  beforeEach(() => {
    vi.mocked(api.getBlocklists).mockResolvedValue(BLOCKLISTS)
    vi.mocked(api.toggleBlocklist).mockResolvedValue({ ...SOURCE, enabled: false })
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it("renders blocklist sources and does not render a curated-categories section", async () => {
    render(
      <SidebarProvider>
        <BlocklistsPage />
      </SidebarProvider>,
    )

    expect(await screen.findByText("StevenBlack Hosts")).toBeInTheDocument()
    expect(screen.queryByText(/curated categories/i)).not.toBeInTheDocument()
  })

  it("toggles a blocklist source and calls the toggleBlocklist API", async () => {
    render(
      <SidebarProvider>
        <BlocklistsPage />
      </SidebarProvider>,
    )

    const toggle = await screen.findByRole("switch", { name: /toggle stevenblack hosts/i })
    fireEvent.click(toggle)

    await waitFor(() => {
      expect(api.toggleBlocklist).toHaveBeenCalledWith("steven-black", false)
    })
  })
})
