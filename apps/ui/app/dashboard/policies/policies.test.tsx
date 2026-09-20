import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"

import PoliciesPage from "./page"
import { SidebarProvider } from "@/components/ui/sidebar"
import * as api from "@/lib/api"
import type { Policy, PolicyListData } from "@/lib/types"

// H1 fix: the Edit/Create Policy drawer used to send `schedule` and
// `client_scope`, fields the backend's CreatePolicyRequest/
// UpdatePolicyRequest (apps/core/cmd/controlplane/handlers/policies.go)
// doesn't have at all — Gin's ShouldBindJSON silently drops them, so a user
// filling them in saw no error but nothing was ever saved. Both inputs and
// both fields have been removed entirely; these tests assert the drawer no
// longer offers them and that the request payload never includes them.
vi.mock("@/lib/api", () => ({
  getPolicies: vi.fn(),
  createPolicy: vi.fn(),
  updatePolicy: vi.fn(),
  deletePolicy: vi.fn(),
}))

const EXISTING: Policy = {
  id: "block-ads",
  name: "Block Ads",
  description: "",
  category: "",
  action: "BLOCK",
  domains: ["ads.example.net"],
  priority: 100,
  enabled: true,
}

const LIST: PolicyListData = {
  total_policies: 1,
  active_policies: 1,
  inactive_policies: 0,
  list: [EXISTING],
}

function renderPage() {
  return render(
    <SidebarProvider>
      <PoliciesPage />
    </SidebarProvider>,
  )
}

describe("PoliciesPage", () => {
  beforeEach(() => {
    vi.mocked(api.getPolicies).mockResolvedValue(LIST)
    vi.mocked(api.createPolicy).mockResolvedValue(EXISTING)
    vi.mocked(api.updatePolicy).mockResolvedValue(EXISTING)
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it("does not render Schedule or Client Scope inputs in the Add Policy drawer", async () => {
    renderPage()

    fireEvent.click(screen.getByRole("button", { name: /add policy/i }))

    expect(await screen.findByText(/create a new dns policy rule/i)).toBeInTheDocument()
    expect(screen.queryByText(/schedule/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/client scope/i)).not.toBeInTheDocument()
  })

  it("createPolicy is called with a payload that never includes schedule or client_scope", async () => {
    renderPage()

    fireEvent.click(screen.getByRole("button", { name: /add policy/i }))
    await screen.findByText(/create a new dns policy rule/i)

    fireEvent.change(screen.getByPlaceholderText(/block social media/i), {
      target: { value: "Block Gambling" },
    })
    fireEvent.change(screen.getByPlaceholderText(/facebook.com/i), {
      target: { value: "bet365.com" },
    })
    fireEvent.click(screen.getByRole("button", { name: /save policy/i }))

    await waitFor(() => expect(api.createPolicy).toHaveBeenCalledTimes(1))
    const payload = vi.mocked(api.createPolicy).mock.calls[0][0]
    expect(payload).not.toHaveProperty("schedule")
    expect(payload).not.toHaveProperty("client_scope")
    expect(payload).toMatchObject({ name: "Block Gambling", domains: ["bet365.com"] })
  })

  it("updatePolicy is called with a payload that never includes schedule or client_scope", async () => {
    renderPage()

    await screen.findByText("Block Ads")
    fireEvent.click(screen.getByTitle("Edit policy"))
    await screen.findByText(/update this dns policy rule/i)

    fireEvent.click(screen.getByRole("button", { name: /update policy/i }))

    await waitFor(() => expect(api.updatePolicy).toHaveBeenCalledTimes(1))
    const [, payload] = vi.mocked(api.updatePolicy).mock.calls[0]
    expect(payload).not.toHaveProperty("schedule")
    expect(payload).not.toHaveProperty("client_scope")
  })
})
