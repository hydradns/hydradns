const BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"
const API = `${BASE_URL}/api/v1`

export function getToken(): string | null {
  if (typeof window === "undefined") return null
  return localStorage.getItem("hydra_token")
}

export function setToken(token: string) {
  localStorage.setItem("hydra_token", token)
  document.cookie = `hydra_token=${token}; path=/; SameSite=Strict`
}

export function clearToken() {
  localStorage.removeItem("hydra_token")
  document.cookie = "hydra_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT"
}

// Returns: "complete" | "needs_setup" | "unreachable"
export async function checkSetupStatus(): Promise<"complete" | "needs_setup" | "unreachable"> {
  try {
    const res = await fetch(`${API}/auth/status`)
    const json = await res.json()
    return json.data?.setup_complete ? "complete" : "needs_setup"
  } catch {
    return "unreachable"
  }
}

export async function login(password: string): Promise<string> {
  const res = await fetch(`${API}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password }),
  })
  const json = await res.json()
  if (json.status === "error") {
    throw new Error(json.error || "Login failed")
  }
  return json.data.token
}

export async function setup(data: {
  password: string
  blocklists?: { id: string; name: string; url: string; format: string }[]
}): Promise<string> {
  const res = await fetch(`${API}/auth/setup`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  })
  const json = await res.json()
  if (json.status === "error") {
    throw new Error(json.error || "Setup failed")
  }
  return json.data.token
}
