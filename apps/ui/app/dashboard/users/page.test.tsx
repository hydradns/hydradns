import { beforeEach, describe, expect, it, vi } from "vitest"
import { fireEvent, render, screen, waitFor } from "@testing-library/react"

import UsersPage from "./page"
import { SidebarProvider } from "@/components/ui/sidebar"
import { getUsers, getMyTokens, createMyToken, revokeMyToken } from "@/lib/api"
import type { UserListData, Token } from "@/lib/types"

// Pages render a <SidebarTrigger>, which needs the provider context the
// dashboard layout normally supplies.
function renderPage() {
  return render(
    <SidebarProvider>
      <UsersPage />
    </SidebarProvider>,
  )
}

// The page only touches lib/api; stub the whole module so no real fetch runs.
// H3 fix: tokens are "my tokens" now (flat /tokens, no nested /users/:id/
// route, no rotate endpoint) — see lib/api.ts and app/dashboard/users/page.tsx.
vi.mock("@/lib/api", () => ({
  getUsers: vi.fn(),
  createUser: vi.fn(),
  updateUser: vi.fn(),
  deleteUser: vi.fn(),
  getMyTokens: vi.fn(),
  createMyToken: vi.fn(),
  revokeMyToken: vi.fn(),
}))

const sample: UserListData = {
  total_users: 2,
  users: [
    {
      id: "u1",
      username: "root",
      role: "admin",
      enabled: true,
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
      last_login_at: "2026-07-19T00:00:00Z",
    },
    {
      id: "u2",
      username: "jane",
      role: "operator",
      enabled: false,
      created_at: "2026-02-01T00:00:00Z",
      updated_at: "2026-02-01T00:00:00Z",
    },
  ],
}

describe("UsersPage", () => {
  beforeEach(() => {
    vi.mocked(getUsers).mockReset()
  })

  it("renders each account with its role and status", async () => {
    vi.mocked(getUsers).mockResolvedValue(sample)

    renderPage()

    // Rows appear after the initial fetch resolves.
    expect(await screen.findByText("root")).toBeInTheDocument()
    expect(screen.getByText("jane")).toBeInTheDocument()

    // Role badges render their human labels.
    expect(screen.getByText("Admin")).toBeInTheDocument()
    expect(screen.getByText("Operator")).toBeInTheDocument()

    // Enabled vs disabled status.
    expect(screen.getByText("ACTIVE")).toBeInTheDocument()
    expect(screen.getByText("DISABLED")).toBeInTheDocument()

    // Header action is always available.
    expect(screen.getByRole("button", { name: /add user/i })).toBeInTheDocument()
  })

  it("shows the empty state when there are no users", async () => {
    vi.mocked(getUsers).mockResolvedValue({ total_users: 0, users: [] })

    renderPage()

    expect(await screen.findByText("No users yet")).toBeInTheDocument()
  })

  it("surfaces API errors instead of the table", async () => {
    vi.mocked(getUsers).mockRejectedValue(new Error("forbidden: admin required"))

    renderPage()

    await waitFor(() =>
      expect(screen.getByText("forbidden: admin required")).toBeInTheDocument(),
    )
  })

  it("does not render a per-user Tokens action (tokens are scoped to 'my tokens')", async () => {
    vi.mocked(getUsers).mockResolvedValue(sample)

    renderPage()

    await screen.findByText("root")
    expect(screen.queryByTitle("Manage tokens")).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: /my tokens/i })).toBeInTheDocument()
  })

  it("opens the My Tokens drawer, lists own tokens, creates one, and reveals the secret once", async () => {
    vi.mocked(getUsers).mockResolvedValue(sample)
    const existingToken: Token = {
      id: 1,
      user_id: 1,
      label: "laptop-cli",
      created_at: "2026-01-01T00:00:00Z",
      last_used_at: null,
      revoked_at: null,
      expires_at: null,
    }
    vi.mocked(getMyTokens).mockResolvedValue([existingToken])
    vi.mocked(createMyToken).mockResolvedValue({
      token: "hydra_live_abc123",
      meta: { ...existingToken, id: 2, label: "ci-pipeline" },
    })

    renderPage()
    await screen.findByText("root")

    fireEvent.click(screen.getByRole("button", { name: /my tokens/i }))

    expect(await screen.findByText("laptop-cli")).toBeInTheDocument()
    // Scoping is explained in the drawer copy, per H3.
    expect(screen.getByText(/you can only manage your own tokens/i)).toBeInTheDocument()

    fireEvent.change(screen.getByLabelText(/label/i), { target: { value: "ci-pipeline" } })
    fireEvent.click(screen.getByRole("button", { name: /create token/i }))

    await waitFor(() => {
      expect(createMyToken).toHaveBeenCalledWith({ label: "ci-pipeline", expiry_days: 90 })
    })
    expect(await screen.findByText("hydra_live_abc123")).toBeInTheDocument()
  })

  it("revokes a token via the flat /tokens route (no rotate control exists)", async () => {
    vi.mocked(getUsers).mockResolvedValue(sample)
    const existingToken: Token = {
      id: 5,
      user_id: 1,
      label: "old-key",
      created_at: "2026-01-01T00:00:00Z",
      last_used_at: null,
      revoked_at: null,
      expires_at: null,
    }
    vi.mocked(getMyTokens).mockResolvedValue([existingToken])
    vi.mocked(revokeMyToken).mockResolvedValue({})

    renderPage()
    await screen.findByText("root")
    fireEvent.click(screen.getByRole("button", { name: /my tokens/i }))
    await screen.findByText("old-key")

    expect(screen.queryByTitle("Rotate token")).not.toBeInTheDocument()
    fireEvent.click(screen.getByTitle("Revoke token"))

    await waitFor(() => expect(revokeMyToken).toHaveBeenCalledWith(5))
  })
})
