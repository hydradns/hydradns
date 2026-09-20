// SPDX-License-Identifier: GPL-3.0-or-later
//
// Package audit is a thin helper that every mutating handler calls to
// record who did what, against which target, from where. The helper owns
// the extraction of the actor + client IP + user agent from the Gin
// context so handlers never reach into middleware internals.
//
// Failures are observational: a failed audit write is logged but does
// not fail the originating request. See phase6-rbac-plan.md for the
// full rationale.
package audit

import (
	"encoding/json"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/hydradns/hydradns/apps/core/internal/storage/models"
	"github.com/hydradns/hydradns/apps/core/internal/storage/repositories"
)

// ContextUserKey is the Gin context key the auth middleware uses to
// attach the authenticated User. Duplicated as a string constant here so
// the audit helper does not import the middleware package (which would
// otherwise create a cycle: middleware -> audit -> middleware).
const ContextUserKey = "auth.user"

// Recorder wraps a repository so handlers can call Record without
// threading the Store through every call site.
type Recorder struct {
	repo repositories.AuditRepository
}

func New(repo repositories.AuditRepository) *Recorder {
	return &Recorder{repo: repo}
}

// Record writes one audit event. before and after may be nil (create has
// no before, delete has no after). Any marshalling failure is downgraded
// to a log line; a caller with a broken payload should still succeed on
// the mutation itself.
func (r *Recorder) Record(c *gin.Context, action, target string, before, after interface{}) {
	if r == nil || r.repo == nil {
		return
	}

	evt := &models.AuditEvent{
		Action:    action,
		Target:    target,
		ClientIP:  c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}
	if u, ok := actor(c); ok {
		id := u.ID
		evt.ActorID = &id
	}
	if s, ok := marshalJSON(before); ok {
		evt.BeforeJSON = &s
	}
	if s, ok := marshalJSON(after); ok {
		evt.AfterJSON = &s
	}

	if err := r.repo.Record(evt); err != nil {
		log.Printf("audit: failed to record %s on %s: %v", action, target, err)
	}
}

// actor pulls the authenticated user off the Gin context. Returns (nil,
// false) on unauthenticated requests (setup, login, system events).
func actor(c *gin.Context) (*models.User, bool) {
	v, exists := c.Get(ContextUserKey)
	if !exists {
		return nil, false
	}
	u, ok := v.(*models.User)
	return u, ok
}

func marshalJSON(v interface{}) (string, bool) {
	if v == nil {
		return "", false
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", false
	}
	return string(b), true
}
