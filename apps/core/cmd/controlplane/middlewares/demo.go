// SPDX-License-Identifier: Apache-2.0
package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// demoGuardAllowed is the narrow allowlist of mutating requests a public,
// read-only demo still needs to serve. Keyed by method, valued by the exact
// path (not a prefix) so nothing broader than intended slips through.
//
// POST /api/v1/auth/login is allowed so a visitor can actually sign in as
// the seeded demo user. There is currently no dedicated logout endpoint in
// this codebase (see cmd/controlplane/routes/router.go); if one is added
// later, it belongs in this map too.
//
// POST /api/v1/auth/setup is deliberately NOT allowlisted. Demo mode always
// seeds its own read_only user at startup (see cmd/controlplane/demoseed),
// so setup is never needed, and allowing it would let any visitor mint a
// brand-new admin account with a password of their choosing.
var demoGuardAllowed = map[string]string{
	http.MethodPost: "/api/v1/auth/login",
}

// DemoGuard rejects every mutating request (any method other than GET,
// HEAD, or OPTIONS) with 403, except the narrow allowlist above. It is the
// actual security boundary for a public demo deployment (HYDRA_DEMO_MODE=
// true): everything the UI does to look "read-only" (the banner, the
// disabled-looking buttons) is cosmetic on top of this.
//
// This must be installed BEFORE Auth() in the middleware chain (see
// main.go) so it runs, and can reject, before any role (including admin,
// if a demo deployment somehow had one) is even resolved. A role check
// alone would not be a sufficient boundary here: RequireRole is opt-in per
// route and admin bypasses it entirely, so a guard that ran after Auth (or
// that was expressed as a role restriction) could be defeated by any
// endpoint that forgot to opt in, or by an admin token. Running first and
// rejecting by HTTP method means there is nothing to forget.
func DemoGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		if allowedPath, ok := demoGuardAllowed[c.Request.Method]; ok && c.Request.URL.Path == allowedPath {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"status": "error",
			"data":   nil,
			"error":  "demo mode: changes are disabled",
		})
	}
}
