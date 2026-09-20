// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/hydradns/hydra-core/internal/storage/models"
	"github.com/hydradns/hydra-core/internal/storage/repositories"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func openMiddlewareTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Token{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// TestBuildMiddlewareChain_DemoGuardOnlyWhenEnabled proves demo mode's
// guard is not merely inert when disabled — it is not installed on the
// engine at all, so disabled behaviour is byte-for-byte what it was before
// demo mode existed. Structural proof (Handlers count) plus a behavioral
// check (an actual mutating request) so the test fails if either regresses.
func TestBuildMiddlewareChain_DemoGuardOnlyWhenEnabled(t *testing.T) {
	db := openMiddlewareTestDB(t)
	users := repositories.NewUserRepo(db)
	tokens := repositories.NewTokenRepo(db)
	u, err := users.Create("admin@x.com", "hash", models.RoleAdmin)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	token, _, err := tokens.CreateForUser(u.ID, "test", 0)
	if err != nil {
		t.Fatalf("seed token: %v", err)
	}

	rOff := gin.New()
	buildMiddlewareChain(rOff, false, users, tokens)
	offHandlerCount := len(rOff.Handlers)

	rOn := gin.New()
	buildMiddlewareChain(rOn, true, users, tokens)
	onHandlerCount := len(rOn.Handlers)

	if onHandlerCount != offHandlerCount+1 {
		t.Errorf("expected demo mode to add exactly one middleware to the chain, got %d (off) vs %d (on)", offHandlerCount, onHandlerCount)
	}

	rOff.POST("/api/v1/policies", func(c *gin.Context) { c.Status(http.StatusCreated) })
	rOn.POST("/api/v1/policies", func(c *gin.Context) { c.Status(http.StatusCreated) })

	// Demo mode off: a normal admin-authenticated mutation succeeds exactly
	// as it did before this feature existed.
	rec := doAuthedPost(rOff, "/api/v1/policies", token)
	if rec.Code != http.StatusCreated {
		t.Errorf("demo mode off: admin POST got %d, want 201 (unchanged behaviour)", rec.Code)
	}

	// Demo mode on: the same admin-authenticated mutation is rejected by
	// the guard before Auth even runs.
	rec = doAuthedPost(rOn, "/api/v1/policies", token)
	if rec.Code != http.StatusForbidden {
		t.Errorf("demo mode on: admin POST got %d, want 403 (blocked ahead of auth)", rec.Code)
	}
}

func doAuthedPost(r http.Handler, path, bearer string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, nil)
	req.Header.Set("Authorization", "Bearer "+bearer)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func doGet(r *gin.Engine, remoteAddr, xff string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = remoteAddr
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// TestConfigureTrustedProxies_DefaultTrustsNone documents the security
// property the login rate limiter and the audit log both depend on:
// with TRUSTED_PROXIES unset, gin must not honor X-Forwarded-For, so
// c.ClientIP() always resolves to the real socket address.
func TestConfigureTrustedProxies_DefaultTrustsNone(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "")
	r := gin.New()
	if err := configureTrustedProxies(r); err != nil {
		t.Fatalf("configureTrustedProxies: %v", err)
	}

	r.GET("/ip", func(c *gin.Context) {
		c.String(200, c.ClientIP())
	})

	rec := doGet(r, "203.0.113.5:1111", "1.2.3.4")
	if rec.Body.String() != "203.0.113.5" {
		t.Errorf("expected ClientIP to ignore X-Forwarded-For and resolve to the real address, got %q", rec.Body.String())
	}
}

func TestConfigureTrustedProxies_ParsesCSV(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "10.0.0.1, 10.0.0.2")
	r := gin.New()
	if err := configureTrustedProxies(r); err != nil {
		t.Fatalf("configureTrustedProxies: %v", err)
	}

	r.GET("/ip", func(c *gin.Context) {
		c.String(200, c.ClientIP())
	})

	// Request arrives directly from a trusted proxy address: X-Forwarded-For
	// should now be honored.
	rec := doGet(r, "10.0.0.1:1111", "203.0.113.9")
	if rec.Body.String() != "203.0.113.9" {
		t.Errorf("expected a configured trusted proxy to make X-Forwarded-For authoritative, got %q", rec.Body.String())
	}
}

func TestConfigureTrustedProxies_InvalidValueErrors(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "not-an-ip-or-cidr")
	r := gin.New()
	if err := configureTrustedProxies(r); err == nil {
		t.Error("expected an error for an unparseable TRUSTED_PROXIES entry")
	}
}
