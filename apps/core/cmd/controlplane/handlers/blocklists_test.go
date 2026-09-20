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
	"github.com/hydradns/hydradns/apps/core/cmd/controlplane/audit"
	"github.com/hydradns/hydradns/apps/core/cmd/controlplane/middlewares"
	"github.com/hydradns/hydradns/apps/core/internal/storage/models"
	"github.com/hydradns/hydradns/apps/core/internal/storage/repositories"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// blocklistsHarness is a self-contained test rig for the blocklist source
// CRUD endpoints, independent of the users_test.go harness.
type blocklistsHarness struct {
	db     *gorm.DB
	store  *repositories.Store
	router *gin.Engine
}

func newBlocklistsHarness(t *testing.T) *blocklistsHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Token{}, &models.AuditEvent{},
		&models.BlocklistSource{}, &models.BlocklistSnapshot{}, &models.BlocklistEntry{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := repositories.NewStore(db)

	h := &APIHandler{Store: *store, Audit: audit.New(store.Audit)}
	r := gin.New()
	r.Use(middlewares.Auth(store.Users, store.Tokens))

	writeRoles := []string{models.RoleOperator}
	blocklists := r.Group("/api/v1/blocklists")
	blocklists.GET("", h.ListBlocklists)
	blocklists.POST("", middlewares.RequireRole(writeRoles...), h.CreateBlocklist)
	blocklists.GET("/:id", h.GetBlocklist)
	blocklists.PATCH("/:id", middlewares.RequireRole(writeRoles...), h.UpdateBlocklist)
	blocklists.DELETE("/:id", middlewares.RequireRole(writeRoles...), h.DeleteBlocklist)

	return &blocklistsHarness{db: db, store: store, router: r}
}

func (th *blocklistsHarness) seedUser(t *testing.T, email, role string) string {
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

func (th *blocklistsHarness) do(method, path, bearer string, body interface{}) *httptest.ResponseRecorder {
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

type blocklistResponse struct {
	Status string    `json:"status"`
	Error  string    `json:"error"`
	Data   Blocklist `json:"data"`
}

func seedBlocklistSource(t *testing.T, th *blocklistsHarness, id string) {
	t.Helper()
	if err := th.store.Blocklist.CreateSource(&models.BlocklistSource{
		ID: id, Name: "Test List", URL: "http://example.com/hosts", Format: "hosts",
		Category: "ads", Enabled: true, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed source: %v", err)
	}
}

func TestUpdateBlocklist_ToggleEnabled(t *testing.T) {
	th := newBlocklistsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)
	seedBlocklistSource(t, th, "src1")

	rec := th.do("PATCH", "/api/v1/blocklists/src1", tok, gin.H{"enabled": false})
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp blocklistResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.Enabled {
		t.Errorf("expected enabled=false after toggle")
	}

	src, _ := th.store.Blocklist.GetSource("src1")
	if src.Enabled {
		t.Error("expected persisted source to be disabled")
	}
}

func TestUpdateBlocklist_UnknownID404(t *testing.T) {
	th := newBlocklistsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	rec := th.do("PATCH", "/api/v1/blocklists/nope", tok, gin.H{"enabled": false})
	if rec.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", rec.Code)
	}
}

func TestUpdateBlocklist_ReadOnlyForbidden(t *testing.T) {
	th := newBlocklistsHarness(t)
	roTok := th.seedUser(t, "ro@x.com", models.RoleReadOnly)
	seedBlocklistSource(t, th, "src1")

	rec := th.do("PATCH", "/api/v1/blocklists/src1", roTok, gin.H{"enabled": false})
	if rec.Code != http.StatusForbidden {
		t.Errorf("got %d, want 403", rec.Code)
	}
}

func TestUpdateBlocklist_NameAndCategoryEdit(t *testing.T) {
	th := newBlocklistsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)
	seedBlocklistSource(t, th, "src1")

	newName := "Renamed List"
	newCategory := "malware"
	rec := th.do("PATCH", "/api/v1/blocklists/src1", tok, gin.H{
		"name": newName, "category": newCategory,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp blocklistResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.Name != newName || resp.Data.Category != newCategory {
		t.Errorf("unexpected result: %+v", resp.Data)
	}
	// URL/format untouched.
	if resp.Data.URL != "http://example.com/hosts" || resp.Data.Format != "hosts" {
		t.Errorf("expected url/format unchanged, got %+v", resp.Data)
	}
}

// --- CreateBlocklist must validate URL scheme, same as UpdateBlocklist ---

func TestCreateBlocklist_URLMustBeHTTP(t *testing.T) {
	th := newBlocklistsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	rec := th.do("POST", "/api/v1/blocklists", tok, gin.H{
		"id": "evil", "name": "Evil", "url": "http://169.254.169.254/latest/meta-data/", "format": "hosts",
	})
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("expected an internal-looking http:// URL to still be accepted (no IP allowlisting), got %d body=%s", rec.Code, rec.Body.String())
	}

	rec = th.do("POST", "/api/v1/blocklists", tok, gin.H{
		"id": "ftp-src", "name": "FTP", "url": "ftp://example.com/list", "format": "hosts",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400 for a non-http(s) scheme", rec.Code)
	}

	if _, err := th.store.Blocklist.GetSource("ftp-src"); err == nil {
		t.Error("expected no source to be created for the rejected scheme")
	}
}

func TestCreateBlocklist_FileSchemeRejected(t *testing.T) {
	th := newBlocklistsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	rec := th.do("POST", "/api/v1/blocklists", tok, gin.H{
		"id": "file-src", "name": "File", "url": "file:///etc/passwd", "format": "hosts",
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400 for file:// scheme", rec.Code)
	}
}

func TestUpdateBlocklist_URLMustBeHTTP(t *testing.T) {
	th := newBlocklistsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)
	seedBlocklistSource(t, th, "src1")

	rec := th.do("PATCH", "/api/v1/blocklists/src1", tok, gin.H{"url": "ftp://evil.example/list"})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", rec.Code)
	}
}

func TestUpdateBlocklist_RecordsAuditEvent(t *testing.T) {
	th := newBlocklistsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)
	seedBlocklistSource(t, th, "src1")

	rec := th.do("PATCH", "/api/v1/blocklists/src1", tok, gin.H{"enabled": false})
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}

	events, err := th.store.Audit.Query(repositories.AuditFilter{Action: "blocklist.update"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 blocklist.update audit event, got %d", len(events))
	}
	if events[0].Target != "blocklist:src1" {
		t.Errorf("expected target blocklist:src1, got %q", events[0].Target)
	}
}

func TestUpdateBlocklist_IDInBodyIgnored(t *testing.T) {
	th := newBlocklistsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)
	seedBlocklistSource(t, th, "src1")

	rec := th.do("PATCH", "/api/v1/blocklists/src1", tok, gin.H{"id": "hijacked", "name": "New Name"})
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp blocklistResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.ID != "src1" {
		t.Errorf("expected path id to win, got %q", resp.Data.ID)
	}
	if _, err := th.store.Blocklist.GetSource("hijacked"); err == nil {
		t.Error("expected no source created under the body's id")
	}
}
