// DEMO_PASSWORD is the fixed, publicly documented password for the
// read-only demo account (see demo/README.md). It must match
// apps/core/cmd/controlplane/demoseed.DemoUserPassword exactly. That Go
// constant is the source of truth; this is duplicated here only because
// the UI and control plane are separate submodules/deploys with no shared
// build-time config. It is not a secret: the demo's actual security
// boundary is the server-side DemoGuard middleware, which rejects every
// mutating request regardless of credentials.
//
// This lives in its own module (rather than lib/auth.ts) so the login page
// can `import()` it dynamically, only after /auth/status confirms
// demo_mode: true. Every non-demo, self-hosted install never loads this
// chunk, so the string never ships in the login page's main bundle. See
// app/login/page.tsx and lib/auth.ts:getAuthStatus.
export const DEMO_PASSWORD = "hydradns-demo"
