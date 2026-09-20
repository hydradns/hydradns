// SPDX-License-Identifier: GPL-3.0-or-later
package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hydradns/hydradns/apps/core/cmd/controlplane/middlewares"
	"github.com/hydradns/hydradns/apps/core/internal/storage/models"
)

// tokenDTO is the wire shape for Token. The stored hash is never
// serialised; the plaintext token only appears in the create response.
type tokenDTO struct {
	ID         uint       `json:"id"`
	UserID     uint       `json:"user_id"`
	Label      string     `json:"label"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
}

func toTokenDTO(t models.Token) tokenDTO {
	return tokenDTO{
		ID:         t.ID,
		UserID:     t.UserID,
		Label:      t.Label,
		CreatedAt:  t.CreatedAt,
		LastUsedAt: t.LastUsedAt,
		RevokedAt:  t.RevokedAt,
		ExpiresAt:  t.ExpiresAt,
	}
}

type createTokenRequest struct {
	Label      string `json:"label" binding:"required"`
	ExpiryDays int    `json:"expiry_days,omitempty"`
}

// CreateToken handles POST /tokens. Each authenticated user mints tokens
// for their own account. The plaintext is returned exactly once; the
// server only stores the SHA-256 hash.
func (h *APIHandler) CreateToken(c *gin.Context) {
	caller, ok := middlewares.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "error": "unauthenticated"})
		return
	}
	var req createTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": err.Error()})
		return
	}
	var expiry time.Duration
	if req.ExpiryDays > 0 {
		expiry = time.Duration(req.ExpiryDays) * 24 * time.Hour
	}

	plaintext, tok, err := h.Store.Tokens.CreateForUser(caller.ID, req.Label, expiry)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": err.Error()})
		return
	}

	h.Audit.Record(c, "token.create", "token:"+req.Label,
		nil,
		map[string]interface{}{
			"id":         tok.ID,
			"user_id":    tok.UserID,
			"label":      tok.Label,
			"expires_at": tok.ExpiresAt,
		})

	// Plaintext is returned once; the client must store it now.
	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"data": gin.H{
			"token": plaintext,
			"meta":  toTokenDTO(*tok),
		},
	})
}

// ListTokens handles GET /tokens. Returns the caller's own tokens only.
// Admins see all tokens across all users via an ?all=true query flag so
// compliance reviews can spot shared or stale keys.
func (h *APIHandler) ListTokens(c *gin.Context) {
	caller, ok := middlewares.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "error": "unauthenticated"})
		return
	}

	if c.Query("all") == "true" {
		if caller.Role != models.RoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{"status": "error", "error": "forbidden"})
			return
		}
		all, err := h.listAllTokens()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "failed to list tokens"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "data": all})
		return
	}

	tokens, err := h.Store.Tokens.ListForUser(caller.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "failed to list tokens"})
		return
	}
	out := make([]tokenDTO, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, toTokenDTO(t))
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": out})
}

// listAllTokens fans out ListForUser across every user. Callers must
// have already verified the admin role.
func (h *APIHandler) listAllTokens() ([]tokenDTO, error) {
	users, err := h.Store.Users.List()
	if err != nil {
		return nil, err
	}
	var out []tokenDTO
	for _, u := range users {
		tokens, err := h.Store.Tokens.ListForUser(u.ID)
		if err != nil {
			return nil, err
		}
		for _, t := range tokens {
			out = append(out, toTokenDTO(t))
		}
	}
	return out, nil
}

// RevokeToken handles DELETE /tokens/:id. Users can revoke their own
// tokens; admins can revoke any token.
func (h *APIHandler) RevokeToken(c *gin.Context) {
	caller, ok := middlewares.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "error": "unauthenticated"})
		return
	}
	tokenID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": "invalid token id"})
		return
	}

	// Confirm ownership (or admin) before we mutate. The token repo's
	// ListForUser makes this easy without adding a Get-by-id method.
	owned := caller.Role == models.RoleAdmin
	var match *tokenDTO
	if !owned {
		callerTokens, err := h.Store.Tokens.ListForUser(caller.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "internal error"})
			return
		}
		for i := range callerTokens {
			if callerTokens[i].ID == tokenID {
				owned = true
				d := toTokenDTO(callerTokens[i])
				match = &d
				break
			}
		}
	}
	if !owned {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "error": "forbidden"})
		return
	}

	if err := h.Store.Tokens.Revoke(tokenID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "failed to revoke token"})
		return
	}

	target := "token:" + parseUintParamDisplay(c.Param("id"))
	var before interface{}
	if match != nil {
		before = match
	}
	h.Audit.Record(c, "token.revoke", target, before, nil)

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": map[string]interface{}{}})
}

// parseUintParamDisplay is a lint-friendly wrapper that always returns a
// string, for use in audit targets where we want the original parameter
// text even if it was invalid.
func parseUintParamDisplay(s string) string { return s }
