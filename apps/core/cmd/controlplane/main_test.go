// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
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
