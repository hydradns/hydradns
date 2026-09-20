package handlers

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hydradns/hydra-core/internal/storage/models"
	"golang.org/x/crypto/bcrypt"
)

// defaultAdminEmail is used when Setup is called without an email
// (existing dashboard before RBAC rollout) and as the placeholder email
// for admins migrated from the pre-RBAC AdminCredential singleton.
const defaultAdminEmail = "admin@hydradns.local"

// GetAuthStatus returns whether initial setup has been completed.
// "Setup completed" now means "at least one User exists"; the legacy
// AdminCredential is consulted only via the migration on first boot.
func (h *APIHandler) GetAuthStatus(c *gin.Context) {
	n, err := h.Store.Users.Count()
	if err != nil {
		log.Printf("auth status check failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		// demo_mode lets the UI (a single published image, no build-time
		// flag) discover a public demo deployment from this one
		// unauthenticated call: show the banner, prefill the demo
		// credentials, and interpret a 403 with the demo-mode error text
		// as "expected", without needing NEXT_PUBLIC_* wiring at build
		// time. See middlewares.DemoGuard for the actual enforcement.
		"data": gin.H{"setup_complete": n > 0, "demo_mode": h.DemoMode},
	})
}

type setupRequest struct {
	Email      string                  `json:"email,omitempty"`
	Password   string                  `json:"password" binding:"required,min=8"`
	Blocklists []setupBlocklistRequest `json:"blocklists,omitempty"`
}

type setupBlocklistRequest struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	URL    string `json:"url"`
	Format string `json:"format"`
}

// Setup creates the first admin User + a long-lived Token, and optionally
// configures blocklists. Only works if no users exist yet (409 otherwise).
func (h *APIHandler) Setup(c *gin.Context) {
	n, err := h.Store.Users.Count()
	if err != nil {
		log.Printf("setup check failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "internal error"})
		return
	}
	if n > 0 {
		c.JSON(http.StatusConflict, gin.H{"status": "error", "error": "setup already completed"})
		return
	}

	var req setupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": "password is required (minimum 8 characters)"})
		return
	}

	email := req.Email
	if email == "" {
		email = defaultAdminEmail
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "internal error"})
		return
	}

	user, err := h.Store.Users.Create(email, string(hash), models.RoleAdmin)
	if err != nil {
		log.Printf("admin user creation failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "failed to create admin"})
		return
	}

	// Setup tokens never expire; the first admin needs a stable key to
	// bootstrap the dashboard. Operators can rotate from the UI later.
	plaintext, _, err := h.Store.Tokens.CreateUnexpiring(user.ID, "setup")
	if err != nil {
		log.Printf("setup token mint failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "failed to mint token"})
		return
	}

	h.Audit.Record(c, "user.setup", "user:"+user.Email, nil, map[string]string{
		"email": user.Email,
		"role":  user.Role,
	})

	// Create blocklist sources if provided
	var warnings []string
	for _, bl := range req.Blocklists {
		// Validate URL scheme to prevent SSRF
		if !strings.HasPrefix(bl.URL, "http://") && !strings.HasPrefix(bl.URL, "https://") {
			warnings = append(warnings, "skipped "+bl.Name+": URL must use http:// or https://")
			continue
		}
		if bl.Format == "" {
			bl.Format = "hosts"
		}
		src := &models.BlocklistSource{
			ID:      bl.ID,
			Name:    bl.Name,
			URL:     bl.URL,
			Format:  bl.Format,
			Enabled: true,
		}
		if err := h.Store.Blocklist.CreateSource(src); err != nil {
			log.Printf("blocklist source creation failed for %s: %v", bl.ID, err)
			warnings = append(warnings, "failed to add "+bl.Name)
			continue
		}

		// Same async-fetch path as the main CreateBlocklist handler so
		// the setup wizard ends with populated blocklists, not empty ones.
		if h.BlocklistEngine != nil {
			srcCopy := *src
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()
				if err := h.BlocklistEngine.UpdateSource(ctx, srcCopy, ""); err != nil {
					log.Printf("initial blocklist fetch failed for %s: %v", srcCopy.ID, err)
				}
			}()
		}
	}

	resp := gin.H{"token": plaintext}
	if len(warnings) > 0 {
		resp["warnings"] = warnings
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   resp,
	})
}

type loginRequest struct {
	Email    string `json:"email,omitempty"`
	Password string `json:"password" binding:"required"`
}

// Login validates credentials and returns a freshly-minted bearer token.
//
// Backwards-compat: if the request omits "email" and exactly one user
// exists, that user is the login target. The dashboard will be updated
// to always send email in a follow-up submodule bump; until then this
// keeps the first-boot flow identical to the pre-RBAC behaviour.
func (h *APIHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": "password is required"})
		return
	}

	user, err := h.resolveLoginTarget(req.Email)
	if err != nil || user == nil || user.Disabled {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "error": "invalid credentials"})
		return
	}

	// Mint a login-session token. 90-day default expiry keeps stale
	// browser sessions from lingering after a device is retired.
	plaintext, _, err := h.Store.Tokens.CreateForUser(user.ID, "login-session", 0)
	if err != nil {
		log.Printf("login token mint failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "failed to mint token"})
		return
	}

	_ = h.Store.Users.TouchLogin(user.ID)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   gin.H{"token": plaintext},
	})
}

func (h *APIHandler) resolveLoginTarget(email string) (*models.User, error) {
	if email != "" {
		return h.Store.Users.GetByEmail(email)
	}
	// No email provided: only acceptable if there is exactly one user,
	// the single-admin backward-compat case.
	all, err := h.Store.Users.List()
	if err != nil {
		return nil, err
	}
	if len(all) != 1 {
		return nil, nil
	}
	u := all[0]
	return &u, nil
}
