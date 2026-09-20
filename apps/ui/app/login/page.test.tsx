import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"

const replaceMock = vi.fn()
const pushMock = vi.fn()
vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: replaceMock, push: pushMock }),
}))

import LoginPage from "@/app/login/page"
import { DEMO_PASSWORD } from "@/lib/auth"

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
  replaceMock.mockClear()
  pushMock.mockClear()
  localStorage.clear()
})

describe("LoginPage demo mode", () => {
  it("prefills the demo password and shows the Enter Demo button when demo_mode is true", async () => {
    stubAuthStatus({ setup_complete: true, demo_mode: true })

    render(<LoginPage />)

    const passwordInput = await screen.findByLabelText<HTMLInputElement>(/password/i)
    await waitFor(() => expect(passwordInput.value).toBe(DEMO_PASSWORD))

    expect(screen.getByRole("button", { name: /enter demo/i })).toBeInTheDocument()
    // Never route a demo visitor to the setup wizard, even though
    // setup_complete + demo_mode together already prevent that server-side.
    expect(replaceMock).not.toHaveBeenCalledWith("/setup")
  })

  it("leaves the password field empty and shows Sign In when demo_mode is false", async () => {
    stubAuthStatus({ setup_complete: true, demo_mode: false })

    render(<LoginPage />)

    const passwordInput = await screen.findByLabelText<HTMLInputElement>(/password/i)
    await waitFor(() => expect(screen.queryByText(/sign in/i)).toBeInTheDocument())
    expect(passwordInput.value).toBe("")
    expect(screen.queryByRole("button", { name: /enter demo/i })).not.toBeInTheDocument()
  })

  it("redirects to /setup when setup is incomplete and it is not a demo", async () => {
    stubAuthStatus({ setup_complete: false, demo_mode: false })

    render(<LoginPage />)

    await waitFor(() => expect(replaceMock).toHaveBeenCalledWith("/setup"))
  })
})
