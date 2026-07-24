import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { cleanup, render, screen } from "@testing-library/react"

import ResolversPage from "./page"
import { SidebarProvider } from "@/components/ui/sidebar"
import * as api from "@/lib/api"
import type { Resolver } from "@/lib/types"

vi.mock("@/lib/api", () => ({
  getResolvers: vi.fn(),
  createResolver: vi.fn(),
  updateResolver: vi.fn(),
  deleteResolver: vi.fn(),
}))

const RESOLVERS: Resolver[] = [
  { id: "cf", name: "Cloudflare", address: "1.1.1.1:53", protocol: "udp" },
  { id: "google", name: "Google", address: "8.8.8.8:53", protocol: "tcp" },
]

describe("ResolversPage", () => {
  beforeEach(() => {
    vi.mocked(api.getResolvers).mockResolvedValue(RESOLVERS)
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it("renders the resolver table with rows from the API", async () => {
    render(
      <SidebarProvider>
        <ResolversPage />
      </SidebarProvider>,
    )

    expect(await screen.findByText("Cloudflare")).toBeInTheDocument()
    expect(screen.getByText("1.1.1.1:53")).toBeInTheDocument()
    expect(screen.getByText("Google")).toBeInTheDocument()
    expect(screen.getByText("8.8.8.8:53")).toBeInTheDocument()

    // Column headers present -> a real table rendered.
    expect(screen.getByRole("columnheader", { name: /address/i })).toBeInTheDocument()
  })
})
