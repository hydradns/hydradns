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

// analyticsHarness is a self-contained test rig for the analytics
// endpoints (GET /analytics/logs, GET /analytics/bypass), independent of
// the users_test.go harness so this file's changes stay self-contained.
type analyticsHarness struct {
	db     *gorm.DB
	store  *repositories.Store
	router *gin.Engine
	// h is the same *APIHandler the routes below are bound to; tests may
	// mutate its fields (DemoMode, AnonymizeSecret) after construction to
	// exercise those code paths without a second harness.
	h *APIHandler
}

func newAnalyticsHarness(t *testing.T) *analyticsHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Token{}, &models.AuditEvent{}, &models.DNSQuery{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := repositories.NewStore(db)

	h := &APIHandler{Store: *store, Audit: audit.New(store.Audit)}
	r := gin.New()
	r.Use(middlewares.Auth(store.Users, store.Tokens))

	analytics := r.Group("/api/v1/analytics")
	analytics.GET("/logs", h.GetQueryLogsPage)
	analytics.GET("/bypass", h.GetBypassAttempts)
	analytics.GET("/audits", h.GetAuditLogs)
	analytics.GET("/summary", h.GetAnalyticsSummary)

	return &analyticsHarness{db: db, store: store, router: r, h: h}
}

func (th *analyticsHarness) seedUser(t *testing.T, email, role string) string {
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

func (th *analyticsHarness) do(method, path, bearer string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBuffer(nil))
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	th.router.ServeHTTP(rec, req)
	return rec
}

type queryLogPageResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
	Data   struct {
		Items    []QueryLogEntry `json:"items"`
		Total    int64           `json:"total"`
		Page     int             `json:"page"`
		PageSize int             `json:"page_size"`
	} `json:"data"`
}

// --- GET /analytics/logs ---

func TestGetQueryLogsPage_ReadableByReadOnly(t *testing.T) {
	th := newAnalyticsHarness(t)
	roTok := th.seedUser(t, "ro@x.com", models.RoleReadOnly)

	th.db.Create(&models.DNSQuery{Domain: "a.com", ClientIP: "1.1.1.1", Action: "allow", Timestamp: time.Now()})

	rec := th.do("GET", "/api/v1/analytics/logs", roTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("read_only: got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetQueryLogsPage_DefaultsAndEnvelope(t *testing.T) {
	th := newAnalyticsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	now := time.Now()
	for i := 0; i < 3; i++ {
		th.db.Create(&models.DNSQuery{
			Domain: "a.com", ClientIP: "1.1.1.1", Action: "allow",
			Timestamp: now.Add(time.Duration(i) * time.Second),
		})
	}

	rec := th.do("GET", "/api/v1/analytics/logs", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp queryLogPageResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "success" {
		t.Errorf("status: got %q", resp.Status)
	}
	if resp.Data.Total != 3 || len(resp.Data.Items) != 3 {
		t.Errorf("expected 3 items/total, got items=%d total=%d", len(resp.Data.Items), resp.Data.Total)
	}
	if resp.Data.Page != 1 {
		t.Errorf("expected default page=1, got %d", resp.Data.Page)
	}
	if resp.Data.PageSize != 50 {
		t.Errorf("expected default page_size=50, got %d", resp.Data.PageSize)
	}
	// Newest first.
	if resp.Data.Items[0].Domain != "a.com" || !resp.Data.Items[0].Timestamp.After(resp.Data.Items[2].Timestamp) {
		t.Errorf("expected newest-first ordering, got %+v", resp.Data.Items)
	}
}

func TestGetQueryLogsPage_PageSizeClampedToHardMax(t *testing.T) {
	th := newAnalyticsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	rec := th.do("GET", "/api/v1/analytics/logs?page_size=100000", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp queryLogPageResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.PageSize > 200 {
		t.Errorf("expected page_size clamped to a hard max, got %d", resp.Data.PageSize)
	}
}

func TestGetQueryLogsPage_InvalidPageIsBadRequest(t *testing.T) {
	th := newAnalyticsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	rec := th.do("GET", "/api/v1/analytics/logs?page=not-a-number", tok)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", rec.Code)
	}
}

func TestGetQueryLogsPage_InvalidTimeIsBadRequest(t *testing.T) {
	th := newAnalyticsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	rec := th.do("GET", "/api/v1/analytics/logs?start=not-a-date", tok)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", rec.Code)
	}
}

func TestGetQueryLogsPage_FilterByDomainAndAction(t *testing.T) {
	th := newAnalyticsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	th.db.Create(&models.DNSQuery{Domain: "ads.example.com", ClientIP: "1.1.1.1", Action: "block", Timestamp: time.Now()})
	th.db.Create(&models.DNSQuery{Domain: "example.com", ClientIP: "1.1.1.1", Action: "allow", Timestamp: time.Now()})

	rec := th.do("GET", "/api/v1/analytics/logs?domain=ads&action=block", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp queryLogPageResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.Total != 1 || len(resp.Data.Items) != 1 || resp.Data.Items[0].Domain != "ads.example.com" {
		t.Errorf("expected only the blocked ads domain, got %+v", resp.Data)
	}
}

func TestGetQueryLogsPage_ActionAllMeansNoFilter(t *testing.T) {
	th := newAnalyticsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	th.db.Create(&models.DNSQuery{Domain: "a.com", ClientIP: "1.1.1.1", Action: "block", Timestamp: time.Now()})
	th.db.Create(&models.DNSQuery{Domain: "b.com", ClientIP: "1.1.1.1", Action: "allow", Timestamp: time.Now()})

	rec := th.do("GET", "/api/v1/analytics/logs?action=all", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}
	var resp queryLogPageResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.Total != 2 {
		t.Errorf("expected action=all to mean no filter, got total=%d", resp.Data.Total)
	}
}

func TestGetQueryLogsPage_ClientFilterPrecise(t *testing.T) {
	th := newAnalyticsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	th.db.Create(&models.DNSQuery{Domain: "a.com", ClientIP: "192.168.1.5", Action: "allow", Timestamp: time.Now()})
	th.db.Create(&models.DNSQuery{Domain: "b.com", ClientIP: "192.168.1.50", Action: "allow", Timestamp: time.Now()})

	rec := th.do("GET", "/api/v1/analytics/logs?client=192.168.1.5", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}
	var resp queryLogPageResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.Total != 1 || resp.Data.Items[0].ClientIP != "192.168.1.5" {
		t.Errorf("expected exact client match only, got %+v", resp.Data)
	}
}

// --- M6: reachable offset is capped, count is capped ---

func TestGetQueryLogsPage_PageBeyondReachableOffsetIsBadRequest(t *testing.T) {
	th := newAnalyticsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	// page * page_size = 100001 * 200 far exceeds the 100,000 cap.
	rec := th.do("GET", "/api/v1/analytics/logs?page=100001&page_size=200", tok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400 for an unreachable offset", rec.Code)
	}
}

func TestGetQueryLogsPage_PageAtReachableOffsetBoundaryIsOK(t *testing.T) {
	th := newAnalyticsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	// 500 * 200 = 100,000, exactly at the cap: must still be served.
	rec := th.do("GET", "/api/v1/analytics/logs?page=500&page_size=200", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d body=%s, want 200 at the exact offset boundary", rec.Code, rec.Body.String())
	}
}

// --- M8: demo mode rejects the client filter (it would otherwise recover
// a masked IP by 256-request oracle) ---

func TestGetQueryLogsPage_DemoModeRejectsClientFilter(t *testing.T) {
	th := newAnalyticsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)
	th.h.DemoMode = true

	th.db.Create(&models.DNSQuery{Domain: "a.com", ClientIP: "192.168.1.5", Action: "allow", Timestamp: time.Now()})

	rec := th.do("GET", "/api/v1/analytics/logs?client=192.168.1.5", tok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400 (client filter unavailable in demo mode)", rec.Code)
	}
}

func TestGetQueryLogsPage_DemoModeStillServesUnfilteredLogs(t *testing.T) {
	th := newAnalyticsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)
	th.h.DemoMode = true

	th.db.Create(&models.DNSQuery{Domain: "a.com", ClientIP: "192.168.1.5", Action: "allow", Timestamp: time.Now()})

	rec := th.do("GET", "/api/v1/analytics/logs", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 for an unfiltered request in demo mode", rec.Code)
	}
	var resp queryLogPageResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.Total != 1 || resp.Data.Items[0].ClientIP != "192.168.1.x" {
		t.Errorf("expected 1 masked row, got %+v", resp.Data)
	}
}

// --- M3: anonymization hashes the client filter to match hashed storage ---

func TestGetQueryLogsPage_AnonymizedClientFilterMatchesHashedStorage(t *testing.T) {
	th := newAnalyticsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	const secret = "test-anon-secret"
	th.h.AnonymizeSecret = secret

	hashed, ok := hashClientIPForFilter(secret, "192.168.1.5")
	if !ok {
		t.Fatal("expected 192.168.1.5 to hash successfully")
	}
	th.db.Create(&models.DNSQuery{Domain: "a.com", ClientIP: hashed, Action: "allow", Timestamp: time.Now()})
	// A different client's hash must not collide.
	otherHashed, _ := hashClientIPForFilter(secret, "192.168.1.6")
	th.db.Create(&models.DNSQuery{Domain: "b.com", ClientIP: otherHashed, Action: "allow", Timestamp: time.Now()})

	rec := th.do("GET", "/api/v1/analytics/logs?client=192.168.1.5", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp queryLogPageResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.Total != 1 || len(resp.Data.Items) != 1 || resp.Data.Items[0].Domain != "a.com" {
		t.Errorf("expected exactly the row matching the hashed filter, got %+v", resp.Data)
	}
}

func TestGetQueryLogsPage_AnonymizedClientFilterInvalidValueIsBadRequest(t *testing.T) {
	th := newAnalyticsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)
	th.h.AnonymizeSecret = "test-anon-secret"

	rec := th.do("GET", "/api/v1/analytics/logs?client=not-an-ip", tok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400 for an unparseable client filter under anonymization", rec.Code)
	}
}

func TestGetQueryLogsPage_Pagination(t *testing.T) {
	th := newAnalyticsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	for i := 0; i < 5; i++ {
		th.db.Create(&models.DNSQuery{Domain: "a.com", ClientIP: "1.1.1.1", Action: "allow", Timestamp: time.Now()})
	}

	rec := th.do("GET", "/api/v1/analytics/logs?page=2&page_size=2", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}
	var resp queryLogPageResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.Total != 5 || len(resp.Data.Items) != 2 || resp.Data.Page != 2 {
		t.Errorf("expected page 2 of 2 items (5 total), got %+v", resp.Data)
	}
}

// --- GET /analytics/bypass ---

type bypassResponse struct {
	Status string `json:"status"`
	Data   struct {
		TotalAttempts int64           `json:"total_attempts"`
		UniqueClients int64           `json:"unique_clients"`
		Attempts      []BypassAttempt `json:"attempts"`
	} `json:"data"`
}

func TestGetBypassAttempts_EmptyIsSuccessNotError(t *testing.T) {
	th := newAnalyticsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	rec := th.do("GET", "/api/v1/analytics/bypass", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp bypassResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.TotalAttempts != 0 || len(resp.Data.Attempts) != 0 {
		t.Errorf("expected empty result, got %+v", resp.Data)
	}
}

func TestGetBypassAttempts_AggregatesByClientAndTarget(t *testing.T) {
	th := newAnalyticsHarness(t)
	tok := th.seedUser(t, "op@x.com", models.RoleOperator)

	now := time.Now()
	// Two attempts from the same client to the same target.
	th.db.Create(&models.DNSQuery{
		Domain: "dns.google", ClientIP: "192.168.1.5", Action: "block",
		DetectionMethod: models.DetectionMethodDoHBootstrap, Timestamp: now,
	})
	th.db.Create(&models.DNSQuery{
		Domain: "dns.google", ClientIP: "192.168.1.5", Action: "block",
		DetectionMethod: models.DetectionMethodDoHBootstrap, Timestamp: now.Add(time.Minute),
	})
	// A different client to a different target.
	th.db.Create(&models.DNSQuery{
		Domain: "cloudflare-dns.com", ClientIP: "192.168.1.9", Action: "block",
		DetectionMethod: models.DetectionMethodDoHBootstrap, Timestamp: now,
	})
	// An ordinary blocklist block — must NOT be counted as a bypass attempt.
	th.db.Create(&models.DNSQuery{
		Domain: "ads.example.com", ClientIP: "192.168.1.5", Action: "block", Timestamp: now,
	})

	rec := th.do("GET", "/api/v1/analytics/bypass", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp bypassResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.TotalAttempts != 3 {
		t.Errorf("expected 3 total attempts (bootstrap rows only), got %d", resp.Data.TotalAttempts)
	}
	if resp.Data.UniqueClients != 2 {
		t.Errorf("expected 2 unique clients, got %d", resp.Data.UniqueClients)
	}
	if len(resp.Data.Attempts) != 2 {
		t.Fatalf("expected 2 (client,target) groups, got %d: %+v", len(resp.Data.Attempts), resp.Data.Attempts)
	}
	// The 2-attempt group should sort first (ORDER BY attempts DESC).
	top := resp.Data.Attempts[0]
	if top.ClientIP != "192.168.1.5" || top.Target != "dns.google" || top.Attempts != 2 {
		t.Errorf("expected top group to be 192.168.1.5/dns.google x2, got %+v", top)
	}
	if !top.Blocked {
		t.Errorf("expected blocked=true for a group made entirely of block rows")
	}
}
