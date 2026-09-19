// SPDX-License-Identifier: GPL-3.0-or-later
package middlewares

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hydradns/hydra-core/cmd/controlplane/audit"
	"github.com/hydradns/hydra-core/internal/storage/models"
	"github.com/hydradns/hydra-core/internal/storage/repositories"
)

// exemptPaths never hit auth: health, setup wizard, and login. Setup is
// exempt until the first user exists; once users > 0, Setup itself
// returns 409.
var exemptPaths = map[string]bool{
	"/health":             true,
	"/":                   true,
	"/api/v1/auth/status": true,
	"/api/v1/auth/login":  true,
	"/api/v1/auth/setup":  true,
}

// Auth resolves a bearer token to a User and attaches the User to the
// request context. Requests without a token, with an unknown token, or
// with a revoked/expired token respond 401 with the same body, so
// callers cannot distinguish cause by side channel.
//
// The pre-RBAC middleware accepted any matching AdminCredential.APIKey.
// Post-RBAC tokens are looked up by SHA-256 hash, so the plaintext never
// touches the DB. Legacy tokens migrated into the new table keep working
// because the migration preserved their plaintext-to-hash mapping.
func Auth(users repositories.UserRepository, tokens repositories.TokenRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		if exemptPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// If the system has never been set up, block everything except
		// the exempt paths above so callers are nudged toward /setup.
		n, err := users.Count()
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{"status": "error", "error": "internal error"})
			return
		}
		if n == 0 {
			c.AbortWithStatusJSON(403, gin.H{
				"status": "error",
				"error":  "setup required — complete setup at /api/v1/auth/setup",
			})
			return
		}

		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(401, gin.H{
				"status": "error",
				"error":  "unauthorized — provide Authorization: Bearer <token>",
			})
			return
		}

		plaintext := strings.TrimPrefix(header, "Bearer ")
		tok, user, err := tokens.ResolveActive(plaintext)
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{"status": "error", "error": "internal error"})
			return
		}
		if tok == nil || user == nil {
			c.AbortWithStatusJSON(401, gin.H{"status": "error", "error": "invalid token"})
			return
		}

		// Attach the User to the context under the key the audit helper
		// reads so every downstream handler can pull it back out uniformly.
		c.Set(audit.ContextUserKey, user)
		c.Next()
	}
}

// RequireRole guards mutating endpoints. Unlisted handlers are readable
// by any authenticated user (read_only included); listing a role on a
// handler is how "write" access is opted into.
//
// Admin bypasses every role check because "admin" is strictly more
// privileged than any operator or read_only access. Operators and
// read_only users only match handlers that explicitly list their role.
func RequireRole(allowed ...string) gin.HandlerFunc {
	allowSet := make(map[string]bool, len(allowed))
	for _, r := range allowed {
		allowSet[r] = true
	}
	return func(c *gin.Context) {
		u, ok := UserFromContext(c)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"status": "error", "error": "unauthenticated"})
			return
		}
		if u.Role == models.RoleAdmin {
			c.Next()
			return
		}
		if !allowSet[u.Role] {
			c.AbortWithStatusJSON(403, gin.H{"status": "error", "error": "forbidden"})
			return
		}
		c.Next()
	}
}

// UserFromContext pulls the authenticated User off the Gin context.
// Handlers should prefer this over c.Get directly so the context key
// stays encapsulated.
func UserFromContext(c *gin.Context) (*models.User, bool) {
	v, exists := c.Get(audit.ContextUserKey)
	if !exists {
		return nil, false
	}
	u, ok := v.(*models.User)
	return u, ok
}
