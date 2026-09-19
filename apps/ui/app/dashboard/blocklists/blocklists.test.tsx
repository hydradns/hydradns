import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"

import BlocklistsPage from "./page"
import { SidebarProvider } from "@/components/ui/sidebar"
import * as api from "@/lib/api"
import type { BlocklistListData, Category } from "@/lib/types"

vi.mock("@/lib/api", () => ({
  getBlocklists: vi.fn(),
  createBlocklist: vi.fn(),
  deleteBlocklist: vi.fn(),
  toggleBlocklist: vi.fn(),
  getCategories: vi.fn(),
  toggleCategory: vi.fn(),
}))

const BLOCKLISTS: BlocklistListData = {
  total_blocklists: 0,
  total_domains: 0,
  active_lists: [],
}

const CATEGORIES: Category[] = [
  { id: "malware", name: "Malware", description: "Known bad", enabled: false, domains_count: 1200 },
]

describe("BlocklistsPage", () => {
  beforeEach(() => {
    vi.mocked(api.getBlocklists).mockResolvedValue(BLOCKLISTS)
    vi.mocked(api.getCategories).mockResolvedValue(CATEGORIES)
    vi.mocked(api.toggleCategory).mockResolvedValue(CATEGORIES[0])
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it("toggles a curated category and calls the toggleCategory API", async () => {
    render(
      <SidebarProvider>
        <BlocklistsPage />
      </SidebarProvider>,
    )

    const toggle = await screen.findByRole("switch", { name: /toggle malware/i })
    fireEvent.click(toggle)

    await waitFor(() => {
      expect(api.toggleCategory).toHaveBeenCalledWith("malware", true)
    })
  })
})
