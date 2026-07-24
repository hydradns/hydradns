import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"

import UsersPage from "./page"
import { SidebarProvider } from "@/components/ui/sidebar"
import { getUsers } from "@/lib/api"
import type { UserListData } from "@/lib/types"

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
vi.mock("@/lib/api", () => ({
  getUsers: vi.fn(),
  createUser: vi.fn(),
  updateUser: vi.fn(),
  deleteUser: vi.fn(),
  getUserTokens: vi.fn(),
  createUserToken: vi.fn(),
  rotateUserToken: vi.fn(),
  revokeUserToken: vi.fn(),
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
})
