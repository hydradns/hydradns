// SPDX-License-Identifier: GPL-3.0-or-later
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/hydradns/hydra-core/cmd/controlplane/audit"
	"github.com/hydradns/hydra-core/cmd/controlplane/middlewares"
	"github.com/hydradns/hydra-core/internal/storage/models"
	"github.com/hydradns/hydra-core/internal/storage/repositories"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// policiesHarness is a self-contained test rig for the policy CRUD
// endpoints, independent of the users_test.go harness so this file's
// changes stay self-contained.
type policiesHarness struct {
	db     *gorm.DB
	store  *repositories.Store
	router *gin.Engine
}

func newPoliciesHarness(t *testing.T) *policiesHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Token{}, &models.AuditEvent{}, &models.Policy{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := repositories.NewStore(db)

	h := &APIHandler{Store: *store, Audit: audit.New(store.Audit)}
	r := gin.New()
	r.Use(middlewares.Auth(store.Users, store.Tokens))

	writeRoles := []string{models.RoleOperator}
	policies := r.Group("/api/v1/policies")
	policies.GET("", h.ListPolicies)
	policies.POST("", middlewares.RequireRole(writeRoles...), h.CreatePolicy)
	policies.GET("/:id", h.GetPolicy)
	policies.PUT("/:id", middlewares.RequireRole(writeRoles...), h.UpdatePolicy)
	policies.DELETE("/:id", middlewares.RequireRole(writeRoles...), h.DeletePolicy)

	return &policiesHarness{db: db, store: store, router: r}
}

func (th *policiesHarness) seedUser(t *testing.T, email, role string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("password1234"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	u, err := th.store.Users.Create(email, string(hash), role)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	plaintext, _, err := th.store.Tokens.CreateForUser(u.ID, "test", time.Hour)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	return plaintext
}

func (th *policiesHarness) do(method, path, bearer string, body interface{}) *httptest.ResponseRecorder {
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	th.router.ServeHTTP(rec, req)
	return rec
}

type policyResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
	Data   Policy `json:"data"`
}

func TestUpdatePolicy_HappyPath(t *testing.T) {
	th := newPoliciesHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	th.do("POST", "/api/v1/policies", tok, gin.H{
		"id": "block-ads", "name": "Block Ads", "action": "BLOCK",
		"domains": []string{"ads.example.com"}, "priority": 100,
	})

	rec := th.do("PUT", "/api/v1/policies/block-ads", tok, gin.H{
		"name": "Block Ads v2", "action": "BLOCK",
		"domains": []string{"ads.example.com", "tracker.com"}, "priority": 150,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp policyResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.Name != "Block Ads v2" || resp.Data.Priority != 150 || len(resp.Data.Domains) != 2 {
		t.Errorf("unexpected updated policy: %+v", resp.Data)
	}
	if resp.Data.ID != "block-ads" {
		t.Errorf("expected id unchanged, got %q", resp.Data.ID)
	}
}

func TestUpdatePolicy_PathIDWinsOverBodyID(t *testing.T) {
	th := newPoliciesHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	th.do("POST", "/api/v1/policies", tok, gin.H{
		"id": "real-id", "name": "Real", "action": "BLOCK", "domains": []string{"a.com"},
	})

	// Body carries a different id; the struct bound for PUT has no id
	// field at all, so this is also a compile-time guarantee, not just a
	// runtime one — but assert the observable behavior too.
	rec := th.do("PUT", "/api/v1/policies/real-id", tok, gin.H{
		"id": "attacker-id", "name": "Renamed", "action": "BLOCK", "domains": []string{"a.com"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp policyResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.ID != "real-id" {
		t.Errorf("expected path id to win, got %q", resp.Data.ID)
	}

	// The attacker-id must not have been created as a new row.
	if _, err := th.store.Policies.GetByID("attacker-id"); err == nil {
		t.Error("expected no policy created under the body's id")
	}
}

func TestUpdatePolicy_UnknownID404(t *testing.T) {
	th := newPoliciesHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	rec := th.do("PUT", "/api/v1/policies/nope", tok, gin.H{
		"name": "X", "action": "BLOCK", "domains": []string{"a.com"},
	})
	if rec.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", rec.Code)
	}
}

func TestUpdatePolicy_ReadOnlyForbidden(t *testing.T) {
	th := newPoliciesHarness(t)
	opTok := th.seedUser(t, "op@x.com", models.RoleOperator)
	roTok := th.seedUser(t, "ro@x.com", models.RoleReadOnly)

	th.do("POST", "/api/v1/policies", opTok, gin.H{
		"id": "p1", "name": "P1", "action": "BLOCK", "domains": []string{"a.com"},
	})

	rec := th.do("PUT", "/api/v1/policies/p1", roTok, gin.H{
		"name": "Hijacked", "action": "BLOCK", "domains": []string{"a.com"},
	})
	if rec.Code != http.StatusForbidden {
		t.Errorf("got %d, want 403", rec.Code)
	}
}

func TestUpdatePolicy_AdminAllowed(t *testing.T) {
	th := newPoliciesHarness(t)
	opTok := th.seedUser(t, "op@x.com", models.RoleOperator)
	adminTok := th.seedUser(t, "admin@x.com", models.RoleAdmin)

	th.do("POST", "/api/v1/policies", opTok, gin.H{
		"id": "p1", "name": "P1", "action": "BLOCK", "domains": []string{"a.com"},
	})

	rec := th.do("PUT", "/api/v1/policies/p1", adminTok, gin.H{
		"name": "P1 by admin", "action": "BLOCK", "domains": []string{"a.com"},
	})
	if rec.Code != http.StatusOK {
		t.Errorf("admin: got %d, want 200", rec.Code)
	}
}

func TestUpdatePolicy_MissingRequiredFieldIsBadRequest(t *testing.T) {
	th := newPoliciesHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	th.do("POST", "/api/v1/policies", tok, gin.H{
		"id": "p1", "name": "P1", "action": "BLOCK", "domains": []string{"a.com"},
	})

	// Missing "action" — same required-field validation as create.
	rec := th.do("PUT", "/api/v1/policies/p1", tok, gin.H{
		"name": "P1", "domains": []string{"a.com"},
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", rec.Code)
	}
}

func TestUpdatePolicy_PreservesEnabledWhenOmitted(t *testing.T) {
	th := newPoliciesHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	th.do("POST", "/api/v1/policies", tok, gin.H{
		"id": "p1", "name": "P1", "action": "BLOCK", "domains": []string{"a.com"},
	})
	// Sanity: create always sets Enabled=true.
	got, _ := th.store.Policies.GetByID("p1")
	if !got.Enabled {
		t.Fatalf("expected created policy to be enabled")
	}

	rec := th.do("PUT", "/api/v1/policies/p1", tok, gin.H{
		"name": "P1 renamed", "action": "BLOCK", "domains": []string{"a.com"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp policyResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if !resp.Data.Enabled {
		t.Errorf("expected enabled to be preserved when omitted from the body")
	}
}

func TestUpdatePolicy_CanExplicitlyDisable(t *testing.T) {
	th := newPoliciesHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	th.do("POST", "/api/v1/policies", tok, gin.H{
		"id": "p1", "name": "P1", "action": "BLOCK", "domains": []string{"a.com"},
	})

	rec := th.do("PUT", "/api/v1/policies/p1", tok, gin.H{
		"name": "P1", "action": "BLOCK", "domains": []string{"a.com"}, "enabled": false,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp policyResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.Enabled {
		t.Errorf("expected enabled=false to be honored")
	}
}

func TestUpdatePolicy_RecordsAuditEvent(t *testing.T) {
	th := newPoliciesHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	th.do("POST", "/api/v1/policies", tok, gin.H{
		"id": "p1", "name": "P1", "action": "BLOCK", "domains": []string{"a.com"}, "priority": 10,
	})
	rec := th.do("PUT", "/api/v1/policies/p1", tok, gin.H{
		"name": "P1 v2", "action": "BLOCK", "domains": []string{"a.com"}, "priority": 20,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}

	events, err := th.store.Audit.Query(repositories.AuditFilter{Action: "policy.update"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 policy.update audit event, got %d", len(events))
	}
	if events[0].Target != "policy:p1" {
		t.Errorf("expected target policy:p1, got %q", events[0].Target)
	}
	if events[0].BeforeJSON == nil || events[0].AfterJSON == nil {
		t.Error("expected both before and after snapshots on an update")
	}
}

func TestUpdatePolicy_PreservesCreatedAt(t *testing.T) {
	th := newPoliciesHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	th.do("POST", "/api/v1/policies", tok, gin.H{
		"id": "p1", "name": "P1", "action": "BLOCK", "domains": []string{"a.com"},
	})
	before, _ := th.store.Policies.GetByID("p1")
	createdAt := before.CreatedAt

	time.Sleep(5 * time.Millisecond)
	rec := th.do("PUT", "/api/v1/policies/p1", tok, gin.H{
		"name": "P1 v2", "action": "BLOCK", "domains": []string{"a.com"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}

	after, _ := th.store.Policies.GetByID("p1")
	if !after.CreatedAt.Equal(createdAt) {
		t.Errorf("expected CreatedAt preserved across update: before=%v after=%v", createdAt, after.CreatedAt)
	}
}
