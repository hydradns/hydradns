// SPDX-License-Identifier: GPL-3.0-or-later
package audit

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lopster568/phantomDNS/internal/storage/models"
	"github.com/lopster568/phantomDNS/internal/storage/repositories"
)

// stubRepo captures recorded events without hitting a real DB.
type stubRepo struct {
	events []*models.AuditEvent
	err    error
}

func (s *stubRepo) Record(evt *models.AuditEvent) error {
	if s.err != nil {
		return s.err
	}
	s.events = append(s.events, evt)
	return nil
}
func (s *stubRepo) Query(_ repositories.AuditFilter) ([]models.AuditEvent, error) {
	return nil, nil
}
func (s *stubRepo) Count(_ repositories.AuditFilter) (int64, error) { return 0, nil }

func ginContextWith(user *models.User, ip, ua string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", nil)
	c.Request.RemoteAddr = ip + ":1234"
	c.Request.Header.Set("User-Agent", ua)
	if user != nil {
		c.Set(ContextUserKey, user)
	}
	return c
}

func TestRecord_HappyPath_CapturesActorAndMarshalsPayloads(t *testing.T) {
	repo := &stubRepo{}
	rec := New(repo)

	user := &models.User{Email: "u@x.com", Role: models.RoleAdmin}
	user.ID = 42
	c := ginContextWith(user, "10.0.0.5", "hydra-cli/1.0")

	before := map[string]int{"priority": 50}
	after := map[string]int{"priority": 100}
	rec.Record(c, "policy.update", "policy:abc", before, after)

	if len(repo.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(repo.events))
	}
	evt := repo.events[0]
	if evt.Action != "policy.update" || evt.Target != "policy:abc" {
		t.Errorf("action/target: %q %q", evt.Action, evt.Target)
	}
	if evt.ActorID == nil || *evt.ActorID != 42 {
		t.Errorf("actor: %+v", evt.ActorID)
	}
	if evt.ClientIP == "" {
		t.Error("ClientIP should be populated")
	}
	if evt.UserAgent != "hydra-cli/1.0" {
		t.Errorf("UA: %q", evt.UserAgent)
	}
	if evt.BeforeJSON == nil || *evt.BeforeJSON != `{"priority":50}` {
		t.Errorf("before: %v", evt.BeforeJSON)
	}
	if evt.AfterJSON == nil || *evt.AfterJSON != `{"priority":100}` {
		t.Errorf("after: %v", evt.AfterJSON)
	}
}

func TestRecord_NoUser_LeavesActorNil(t *testing.T) {
	repo := &stubRepo{}
	rec := New(repo)

	// setup / login flows have no authenticated user yet.
	c := ginContextWith(nil, "10.0.0.6", "mozilla")
	rec.Record(c, "user.setup", "user:admin@hydradns.local", nil, map[string]string{"role": "admin"})

	if len(repo.events) != 1 {
		t.Fatalf("expected 1 event")
	}
	if repo.events[0].ActorID != nil {
		t.Errorf("actor should be nil, got %v", repo.events[0].ActorID)
	}
	if repo.events[0].BeforeJSON != nil {
		t.Error("before should remain nil on create events")
	}
	if repo.events[0].AfterJSON == nil {
		t.Error("after should have been marshalled")
	}
}

func TestRecord_NilRecorder_Safe(t *testing.T) {
	// Guards against a half-wired handler: calling through a nil receiver
	// must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Record on nil *Recorder panicked: %v", r)
		}
	}()
	var rec *Recorder
	c := ginContextWith(nil, "10.0.0.7", "ua")
	rec.Record(c, "x", "y", nil, nil)
}

func TestRecord_RepoFailure_DoesNotPanic(t *testing.T) {
	// A failing audit write must log and return; the caller continues.
	repo := &stubRepo{err: errBoom}
	rec := New(repo)
	c := ginContextWith(nil, "10.0.0.8", "ua")
	rec.Record(c, "x", "y", nil, nil)
}

// errBoom is used by the failure test above.
var errBoom = &stubErr{msg: "simulated db failure"}

type stubErr struct{ msg string }

func (e *stubErr) Error() string { return e.msg }
