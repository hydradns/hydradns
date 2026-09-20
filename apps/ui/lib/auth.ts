import { getApiBaseUrl } from "./api-base"

function apiUrl(path: string): string {
  return `${getApiBaseUrl()}/api/v1${path}`
}

// DEMO_PASSWORD is the fixed, publicly documented password for the
// read-only demo account (see demo/README.md). It must match
// apps/core/cmd/controlplane/demoseed.DemoUserPassword exactly — that Go
// constant is the source of truth; this is duplicated here only because
// the UI and control plane are separate submodules/deploys with no shared
// build-time config. It is not a secret: the demo's actual security
// boundary is the server-side DemoGuard middleware, which rejects every
// mutating request regardless of credentials.
export const DEMO_PASSWORD = "hydradns-demo"

export interface AuthStatus {
  status: "complete" | "needs_setup" | "unreachable"
  demoMode: boolean
}

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

// getAuthStatus is the single call site for GET /api/v1/auth/status
// (unauthenticated). It reports both whether setup is complete and
// whether this deployment is a public demo (HYDRA_DEMO_MODE=true on the
// control plane) — the latter lets the UI show the demo banner and
// prefill the login form from one published image, with no build-time
// flag needed.
export async function getAuthStatus(): Promise<AuthStatus> {
  try {
    const res = await fetch(apiUrl("/auth/status"))
    const json = await res.json()
    const demoMode = Boolean(json.data?.demo_mode)
    return { status: json.data?.setup_complete ? "complete" : "needs_setup", demoMode }
  } catch {
    return { status: "unreachable", demoMode: false }
  }
}

// Returns: "complete" | "needs_setup" | "unreachable"
export async function checkSetupStatus(): Promise<"complete" | "needs_setup" | "unreachable"> {
  return (await getAuthStatus()).status
}

export async function login(password: string): Promise<string> {
  const res = await fetch(apiUrl("/auth/login"), {
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
  const res = await fetch(apiUrl("/auth/setup"), {
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
