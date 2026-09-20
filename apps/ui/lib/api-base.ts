// Single source of truth for the control-plane API base URL.
//
// Next.js inlines NEXT_PUBLIC_* at build time, so whatever value is baked
// into a published Docker image is fixed for the life of that image. The
// release workflow (.github/workflows/release.yml) and docker-compose.yml
// both bake/pass NEXT_PUBLIC_API_URL=http://localhost:8080, which only
// works when the browser and the API happen to be on the same machine —
// not the normal appliance case (Pi on the LAN, browser on a laptop).
//
// Resolution order, in the browser:
//   1. An explicit NEXT_PUBLIC_API_URL that is NOT the legacy default
//      always wins. Operators behind a reverse proxy, or running the API
//      on a different host/port, rely on this.
//   2. Otherwise, derive from the page's own location: same protocol +
//      hostname as window.location, port 8080. So a dashboard opened at
//      http://192.168.1.53:3000 calls http://192.168.1.53:8080, and
//      http://localhost:3000 calls http://localhost:8080.
//   A baked value that is *exactly* the legacy default
//   ("http://localhost:8080") is treated as case 2 (i.e. as if unset)
//   whenever the page itself is not being viewed on localhost/127.0.0.1/
//   [::1] — that value can only be the unmodified CI/compose default in
//   that case, never a deliberate operator choice, so pinning to it would
//   silently re-break the LAN case this helper exists to fix.
//
// On the server (no `window` — SSR, middleware, route handlers) there is no
// page location to derive from. Nothing server-side in this app actually
// calls the control-plane API today — every page that imports lib/api.ts or
// lib/auth.ts is a "use client" component, and middleware.ts only reads a
// cookie, never fetch — so this is a boring, safe fallback rather than a
// real code path. (A previous version of this file added a dedicated
// HYDRA_API_INTERNAL_URL env var for this branch; it was removed as
// undocumented dead code with no caller — see docs/internal/launch-kit/
// review/ui-and-docs-review.md LOW #7. Re-add a real knob here if a server
// component ever needs to call the API.)
const LEGACY_DEFAULT = "http://localhost:8080"

function isLoopbackHostname(hostname: string): boolean {
  // Per the WHATWG URL spec, `location.hostname` for an IPv6 literal already
  // includes the brackets (e.g. "[::1]"), same as `location.host` minus the
  // port — so no bracket-stripping/adding is needed here or below.
  return hostname === "localhost" || hostname === "127.0.0.1" || hostname === "[::1]"
}

// Same protocol + hostname as the page, fixed port 8080.
function deriveFromLocation(location: Pick<Location, "protocol" | "hostname">): string {
  const { protocol, hostname } = location
  return `${protocol}//${hostname}:8080`
}

export function getApiBaseUrl(): string {
  const configured = process.env.NEXT_PUBLIC_API_URL

  if (typeof window === "undefined") {
    return LEGACY_DEFAULT
  }

  if (configured && configured !== LEGACY_DEFAULT) {
    return configured
  }

  if (configured === LEGACY_DEFAULT && isLoopbackHostname(window.location.hostname)) {
    return configured
  }

  return deriveFromLocation(window.location)
}
