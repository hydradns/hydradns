// SPDX-License-Identifier: GPL-3.0-or-later
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hydradns/hydradns/apps/core/internal/storage/models"
	"github.com/hydradns/hydradns/apps/core/internal/storage/repositories"
)

// auditDTO is the wire shape for an audit event. BeforeJSON / AfterJSON
// are parsed into dynamic objects so the dashboard can render them
// without dealing with double-encoded strings.
type auditDTO struct {
	ID         uint        `json:"id"`
	ActorID    *uint       `json:"actor_id"`
	Action     string      `json:"action"`
	Target     string      `json:"target"`
	Before     interface{} `json:"before"`
	After      interface{} `json:"after"`
	ClientIP   string      `json:"client_ip"`
	UserAgent  string      `json:"user_agent"`
	CreatedAt  time.Time   `json:"created_at"`
}

// toAuditDTO is a method (not a free function) so it can consult
// h.DemoMode and redact ClientIP. In practice GET /audit is gated to
// operator+ (see routes/router.go) and demo mode only ever seeds a
// read_only user, so this path is unreachable in a stock demo deployment
// today. The redaction is applied anyway, defensively, so it stays
// correct if that role gate ever changes. See maskClientIP in common.go.
func (h *APIHandler) toAuditDTO(evt models.AuditEvent) auditDTO {
	clientIP := evt.ClientIP
	if h.DemoMode {
		clientIP = maskClientIP(clientIP)
	}
	d := auditDTO{
		ID:        evt.ID,
		ActorID:   evt.ActorID,
		Action:    evt.Action,
		Target:    evt.Target,
		ClientIP:  clientIP,
		UserAgent: evt.UserAgent,
		CreatedAt: evt.CreatedAt,
	}
	if evt.BeforeJSON != nil {
		var v interface{}
		if err := json.Unmarshal([]byte(*evt.BeforeJSON), &v); err == nil {
			d.Before = v
		}
	}
	if evt.AfterJSON != nil {
		var v interface{}
		if err := json.Unmarshal([]byte(*evt.AfterJSON), &v); err == nil {
			d.After = v
		}
	}
	return d
}

// ListAuditEvents handles GET /audit.
// Admin + operator only (routed).
//
// Query params (all optional):
//   actor_id=<uint>   filter by actor
//   action=<string>   exact match, e.g. "policy.create"
//   target=<string>   exact match, e.g. "policy:block-social"
//   from=<rfc3339>    lower bound on created_at
//   to=<rfc3339>      upper bound on created_at
//   limit=<int>       page size (default 100, max 500)
//   offset=<int>      page offset
//
// Response carries a total count so the UI can render pagination
// without a second round-trip.
func (h *APIHandler) ListAuditEvents(c *gin.Context) {
	filter, err := parseAuditFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": err.Error()})
		return
	}

	events, err := h.Store.Audit.Query(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "failed to query audit events"})
		return
	}
	// Count uses the same filter minus limit/offset; reuse the struct
	// with those fields cleared.
	totalFilter := filter
	totalFilter.Limit = 0
	totalFilter.Offset = 0
	total, err := h.Store.Audit.Count(totalFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "failed to count audit events"})
		return
	}

	out := make([]auditDTO, 0, len(events))
	for _, e := range events {
		out = append(out, h.toAuditDTO(e))
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"total":  total,
			"limit":  filter.Limit,
			"offset": filter.Offset,
			"events": out,
		},
	})
}

func parseAuditFilter(c *gin.Context) (repositories.AuditFilter, error) {
	f := repositories.AuditFilter{
		Action: c.Query("action"),
		Target: c.Query("target"),
	}

	if s := c.Query("actor_id"); s != "" {
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return f, err
		}
		v := uint(n)
		f.ActorID = &v
	}
	if s := c.Query("from"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return f, err
		}
		f.From = &t
	}
	if s := c.Query("to"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return f, err
		}
		f.To = &t
	}

	limit := 100
	if s := c.Query("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			return f, err
		}
		if n > 500 {
			n = 500
		}
		limit = n
	}
	f.Limit = limit

	if s := c.Query("offset"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			return f, err
		}
		f.Offset = n
	}

	return f, nil
}
