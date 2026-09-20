// SPDX-License-Identifier: GPL-3.0-or-later
package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hydradns/hydradns/apps/core/cmd/controlplane/middlewares"
	"github.com/hydradns/hydradns/apps/core/internal/storage/models"
	"golang.org/x/crypto/bcrypt"
)

// userDTO is the wire shape for User. PasswordHash is never serialised.
type userDTO struct {
	ID          uint       `json:"id"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	Disabled    bool       `json:"disabled"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastLoginAt *time.Time `json:"last_login_at"`
}

func toUserDTO(u models.User) userDTO {
	return userDTO{
		ID:          u.ID,
		Email:       u.Email,
		Role:        u.Role,
		Disabled:    u.Disabled,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
		LastLoginAt: u.LastLoginAt,
	}
}

type createUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role" binding:"required"`
}

// CreateUser handles POST /users. Admin-only (guarded by the route).
func (h *APIHandler) CreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": err.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "internal error"})
		return
	}

	u, err := h.Store.Users.Create(req.Email, string(hash), req.Role)
	if err != nil {
		// UNIQUE violation on email or a role validation failure. Treat
		// both as 400 so the UI can surface a friendly message.
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": err.Error()})
		return
	}

	out := toUserDTO(*u)
	h.Audit.Record(c, "user.create", "user:"+u.Email, nil, out)

	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": out})
}

// ListUsers handles GET /users. Readable by any authenticated user.
func (h *APIHandler) ListUsers(c *gin.Context) {
	users, err := h.Store.Users.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "failed to list users"})
		return
	}
	out := make([]userDTO, 0, len(users))
	for _, u := range users {
		out = append(out, toUserDTO(u))
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": out})
}

// GetMe handles GET /users/me. Returns the authenticated caller, so
// the UI never has to guess which row in /users corresponds to them.
func (h *APIHandler) GetMe(c *gin.Context) {
	u, ok := middlewares.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "error": "unauthenticated"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": toUserDTO(*u)})
}

type patchUserRequest struct {
	Email    *string `json:"email,omitempty" binding:"omitempty,email"`
	Password *string `json:"password,omitempty" binding:"omitempty,min=8"`
	Role     *string `json:"role,omitempty"`
}

// PatchUser handles PATCH /users/:id.
// - Admin can edit any user (all fields).
// - Non-admins can edit only their own email + password (role field is
//   silently ignored for them; explicit 403 if they target another user).
func (h *APIHandler) PatchUser(c *gin.Context) {
	caller, ok := middlewares.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "error": "unauthenticated"})
		return
	}

	targetID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": "invalid user id"})
		return
	}

	isSelf := caller.ID == targetID
	isAdmin := caller.Role == models.RoleAdmin
	if !isSelf && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "error": "forbidden"})
		return
	}

	var req patchUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": err.Error()})
		return
	}

	before, err := h.Store.Users.Get(targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "error": "user not found"})
		return
	}
	beforeDTO := toUserDTO(*before)

	if req.Email != nil {
		if err := h.Store.Users.UpdateEmail(targetID, *req.Email); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": err.Error()})
			return
		}
	}
	if req.Password != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "internal error"})
			return
		}
		if err := h.Store.Users.UpdatePassword(targetID, string(hash)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "failed to update password"})
			return
		}
	}
	if req.Role != nil {
		if !isAdmin {
			// Non-admins cannot change roles even on themselves. This
			// prevents a self-promotion loophole.
			c.JSON(http.StatusForbidden, gin.H{"status": "error", "error": "only admins can change roles"})
			return
		}
		if err := h.Store.Users.UpdateRole(targetID, *req.Role); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": err.Error()})
			return
		}
	}

	after, err := h.Store.Users.Get(targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "internal error"})
		return
	}
	afterDTO := toUserDTO(*after)

	h.Audit.Record(c, "user.update", "user:"+after.Email, beforeDTO, afterDTO)

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": afterDTO})
}

// SetUserDisabled handles POST /users/:id/disable. Admin-only.
// Body: {"disabled": true|false}. Disabling revokes nothing by itself,
// but any token resolution for a disabled user fails in ResolveActive.
func (h *APIHandler) SetUserDisabled(c *gin.Context) {
	targetID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": "invalid user id"})
		return
	}
	var body struct {
		Disabled bool `json:"disabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": err.Error()})
		return
	}

	before, err := h.Store.Users.Get(targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "error": "user not found"})
		return
	}

	if err := h.Store.Users.SetDisabled(targetID, body.Disabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "failed to update user"})
		return
	}

	after, _ := h.Store.Users.Get(targetID)
	h.Audit.Record(c, "user.disable", "user:"+before.Email, toUserDTO(*before), toUserDTO(*after))

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": toUserDTO(*after)})
}

// DeleteUser handles DELETE /users/:id. Admin-only.
// The last admin cannot be deleted to prevent a lock-out.
func (h *APIHandler) DeleteUser(c *gin.Context) {
	targetID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": "invalid user id"})
		return
	}

	target, err := h.Store.Users.Get(targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "error": "user not found"})
		return
	}
	if target.Role == models.RoleAdmin {
		// Guard against deleting the last admin.
		all, err := h.Store.Users.List()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "internal error"})
			return
		}
		adminCount := 0
		for _, u := range all {
			if u.Role == models.RoleAdmin {
				adminCount++
			}
		}
		if adminCount <= 1 {
			c.JSON(http.StatusConflict, gin.H{"status": "error", "error": "cannot delete the last admin"})
			return
		}
	}

	if err := h.Store.Users.Delete(targetID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "failed to delete user"})
		return
	}

	h.Audit.Record(c, "user.delete", "user:"+target.Email, toUserDTO(*target), nil)

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": map[string]interface{}{}})
}

func parseUintParam(s string) (uint, error) {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(n), nil
}
