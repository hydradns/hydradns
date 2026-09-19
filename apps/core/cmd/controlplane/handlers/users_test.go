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

// testHarness wires an in-memory DB + every repo + an APIHandler with a
// live audit recorder, and returns a Gin router that exposes the user
// and token endpoints under realistic middleware.
type testHarness struct {
	db     *gorm.DB
	store  *repositories.Store
	router *gin.Engine
}

func newHarness(t *testing.T) *testHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Token{}, &models.AuditEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := repositories.NewStore(db)

	h := &APIHandler{
		Store: *store,
		Audit: audit.New(store.Audit),
	}
	r := gin.New()
	r.Use(middlewares.Auth(store.Users, store.Tokens))

	users := r.Group("/api/v1/users")
	users.GET("/me", h.GetMe)
	users.GET("", h.ListUsers)
	users.POST("",
		middlewares.RequireRole(models.RoleAdmin),
		h.CreateUser)
	users.PATCH("/:id", h.PatchUser)
	users.POST("/:id/disable",
		middlewares.RequireRole(models.RoleAdmin),
		h.SetUserDisabled)
	users.DELETE("/:id",
		middlewares.RequireRole(models.RoleAdmin),
		h.DeleteUser)

	tokens := r.Group("/api/v1/tokens")
	tokens.GET("", h.ListTokens)
	tokens.POST("", h.CreateToken)
	tokens.DELETE("/:id", h.RevokeToken)

	r.GET("/api/v1/audit",
		middlewares.RequireRole(models.RoleOperator),
		h.ListAuditEvents)

	return &testHarness{db: db, store: store, router: r}
}

// seedUser provisions a user of the given role with a known password and
// an active token, and returns the plaintext token string the caller
// can plug into Authorization: Bearer <tok>.
func (th *testHarness) seedUser(t *testing.T, email, role string) string {
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

func (th *testHarness) do(method, path, bearer string, body interface{}) *httptest.ResponseRecorder {
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

func TestCreateUser_AdminOnly(t *testing.T) {
	th := newHarness(t)
	adminTok := th.seedUser(t, "admin@x.com", models.RoleAdmin)
	opTok := th.seedUser(t, "op@x.com", models.RoleOperator)

	payload := gin.H{
		"email":    "new@x.com",
		"password": "supersecret",
		"role":     models.RoleOperator,
	}

	// Operator is rejected
	rec := th.do("POST", "/api/v1/users", opTok, payload)
	if rec.Code != http.StatusForbidden {
		t.Errorf("operator create: got %d, want 403", rec.Code)
	}

	// Admin succeeds
	rec = th.do("POST", "/api/v1/users", adminTok, payload)
	if rec.Code != http.StatusCreated {
		t.Errorf("admin create: got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	th := newHarness(t)
	adminTok := th.seedUser(t, "admin@x.com", models.RoleAdmin)

	payload := gin.H{"email": "admin@x.com", "password": "supersecret", "role": models.RoleOperator}
	rec := th.do("POST", "/api/v1/users", adminTok, payload)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("duplicate email: got %d, want 400", rec.Code)
	}
}

func TestGetMe_ReturnsSelf(t *testing.T) {
	th := newHarness(t)
	tok := th.seedUser(t, "alice@x.com", models.RoleOperator)

	rec := th.do("GET", "/api/v1/users/me", tok, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get me: got %d", rec.Code)
	}
	var resp struct {
		Status string  `json:"status"`
		Data   userDTO `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.Email != "alice@x.com" || resp.Data.Role != models.RoleOperator {
		t.Errorf("unexpected body: %+v", resp.Data)
	}
}

func TestPatchUser_SelfCannotChangeRole(t *testing.T) {
	th := newHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	// Look up their own ID so we can PATCH self.
	var me models.User
	th.db.Where("email = ?", "op@x.com").First(&me)

	rec := th.do("PATCH", "/api/v1/users/"+uintToStr(me.ID), tok, gin.H{
		"role": models.RoleAdmin,
	})
	if rec.Code != http.StatusForbidden {
		t.Errorf("self role change: got %d, want 403", rec.Code)
	}

	// But self CAN change own email
	rec = th.do("PATCH", "/api/v1/users/"+uintToStr(me.ID), tok, gin.H{
		"email": "renamed@x.com",
	})
	if rec.Code != http.StatusOK {
		t.Errorf("self email change: got %d", rec.Code)
	}
}

func TestPatchUser_NonAdminTargetingOther_403(t *testing.T) {
	th := newHarness(t)
	opTok := th.seedUser(t, "op@x.com", models.RoleOperator)
	th.seedUser(t, "other@x.com", models.RoleReadOnly)

	var other models.User
	th.db.Where("email = ?", "other@x.com").First(&other)

	rec := th.do("PATCH", "/api/v1/users/"+uintToStr(other.ID), opTok, gin.H{
		"email": "hijack@x.com",
	})
	if rec.Code != http.StatusForbidden {
		t.Errorf("non-admin patching other: got %d, want 403", rec.Code)
	}
}

func TestDeleteUser_LastAdminProtected(t *testing.T) {
	th := newHarness(t)
	adminTok := th.seedUser(t, "admin@x.com", models.RoleAdmin)
	var admin models.User
	th.db.Where("email = ?", "admin@x.com").First(&admin)

	rec := th.do("DELETE", "/api/v1/users/"+uintToStr(admin.ID), adminTok, nil)
	if rec.Code != http.StatusConflict {
		t.Errorf("last-admin delete: got %d, want 409", rec.Code)
	}
}

func TestCreateToken_ReturnsPlaintextOnce(t *testing.T) {
	th := newHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	rec := th.do("POST", "/api/v1/tokens", tok, gin.H{"label": "laptop"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create token: got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data struct {
			Token string   `json:"token"`
			Meta  tokenDTO `json:"meta"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.Token == "" {
		t.Error("plaintext token should be returned on create")
	}
	if resp.Data.Meta.Label != "laptop" {
		t.Errorf("label: %q", resp.Data.Meta.Label)
	}
}

func TestListTokens_ScopedToCallerByDefault(t *testing.T) {
	th := newHarness(t)
	aliceTok := th.seedUser(t, "alice@x.com", models.RoleOperator)
	th.seedUser(t, "bob@x.com", models.RoleOperator)

	rec := th.do("GET", "/api/v1/tokens", aliceTok, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list tokens: %d", rec.Code)
	}
	var resp struct {
		Data []tokenDTO `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	// Alice's seed has exactly one token. Bob's token must not leak in.
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 token scoped to caller, got %d", len(resp.Data))
	}
}

func TestListTokens_AllRequiresAdmin(t *testing.T) {
	th := newHarness(t)
	opTok := th.seedUser(t, "op@x.com", models.RoleOperator)

	rec := th.do("GET", "/api/v1/tokens?all=true", opTok, nil)
	if rec.Code != http.StatusForbidden {
		t.Errorf("operator ?all=true: got %d, want 403", rec.Code)
	}
}

func TestRevokeToken_ScopedToOwner(t *testing.T) {
	th := newHarness(t)
	aliceTok := th.seedUser(t, "alice@x.com", models.RoleOperator)
	bobTok := th.seedUser(t, "bob@x.com", models.RoleOperator)

	// Grab Bob's token id
	var bob models.User
	th.db.Where("email = ?", "bob@x.com").First(&bob)
	var bobToken models.Token
	th.db.Where("user_id = ?", bob.ID).First(&bobToken)

	// Alice cannot revoke Bob's token
	rec := th.do("DELETE", "/api/v1/tokens/"+uintToStr(bobToken.ID), aliceTok, nil)
	if rec.Code != http.StatusForbidden {
		t.Errorf("cross-user revoke: got %d, want 403", rec.Code)
	}

	// Bob can revoke his own
	rec = th.do("DELETE", "/api/v1/tokens/"+uintToStr(bobToken.ID), bobTok, nil)
	if rec.Code != http.StatusOK {
		t.Errorf("self-revoke: got %d", rec.Code)
	}
}

func uintToStr(n uint) string {
	return strconvFormatUint(uint64(n))
}

// strconvFormatUint is split out so we can avoid importing strconv twice.
func strconvFormatUint(n uint64) string {
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = digits[n%10]
		n /= 10
	}
	return string(b[i:])
}
