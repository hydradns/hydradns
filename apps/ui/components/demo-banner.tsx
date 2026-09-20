"use client"

import { useEffect, useState } from "react"
import { Sparkles } from "lucide-react"
import { getAuthStatus } from "@/lib/auth"

// REPO_URL is the "install your own" link the banner points to. Kept as a
// constant here (not read from an env var) since it never changes per
// deployment — it always points at the upstream project, regardless of
// where a given demo instance is hosted.
const REPO_URL = "https://github.com/hydradns/hydradns"

// DemoBanner renders a slim, dismiss-free strip when the control plane
// reports demo_mode: true on GET /api/v1/auth/status (unauthenticated —
// see lib/auth.ts getAuthStatus). Mounted once in the root layout so it
// appears above every page (login, setup, dashboard) without needing to be
// added per-route. Renders nothing while the check is in flight or when
// demo mode is off, so a normal self-hosted install never sees it.
export function DemoBanner() {
  const [demoMode, setDemoMode] = useState(false)

  useEffect(() => {
    let cancelled = false
    getAuthStatus().then((status) => {
      if (!cancelled) setDemoMode(status.demoMode)
    })
    return () => {
      cancelled = true
    }
  }, [])

  if (!demoMode) return null

  return (
    <div
      role="status"
      className="flex items-center justify-center gap-2 bg-[#00D4AA]/10 border-b border-[#00D4AA]/30 px-4 py-2 text-center text-xs sm:text-sm text-[#00D4AA]"
    >
      <Sparkles className="w-3.5 h-3.5 shrink-0" aria-hidden="true" />
      <span>
        Read-only demo. Changes are disabled.{" "}
        <a
          href={REPO_URL}
          target="_blank"
          rel="noreferrer"
          className="font-medium underline underline-offset-2 hover:text-[#00BF9A]"
        >
          Install your own
        </a>
      </span>
    </div>
  )
}
